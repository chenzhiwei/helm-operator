/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"bytes"
	"context"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/chenzhiwei/helm-operator/utils"
	"github.com/chenzhiwei/helm-operator/utils/constant"
	"github.com/chenzhiwei/helm-operator/utils/helm"
	"github.com/chenzhiwei/helm-operator/utils/yaml"
)

func (r *HelmChart) SetupWebhookWithManager(mgr ctrl.Manager) error {
	mgr.GetWebhookServer().Register("/validate-app-siji-io-v1-helmchart", &webhook.Admission{Handler: &validatingHandler{Client: mgr.GetClient()}})

	// leave this does not affect anything
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-app-siji-io-v1-helmchart,mutating=false,failurePolicy=fail,sideEffects=None,groups=app.siji.io,resources=helmcharts,verbs=create;update,versions=v1,name=vhelmchart.kb.io,admissionReviewVersions={v1,v1beta1}

type validatingHandler struct {
	Client  client.Client
	Decoder *admission.Decoder
}

func (h *validatingHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	var log = ctrl.Log.WithName("webhook.helmchart")

	helmChart := &HelmChart{}
	err := h.Decoder.Decode(req, helmChart)
	if err != nil {
		log.Error(err, "failed to decode admission request to helmchart")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// We need to check the user who create the CR whether has the permission to deploy the resources inside Helm chart
	// In this function, we can:
	// 1. Get the UserInfo from admission.Request.AdmissionRequest.UserInfo
	// 2. Get the Helm chart resources
	// 3. Use SubjectAccessReview to check if the user can CRUD Helm chart resources or not
	// 4. Return admission.Allowed if has permission, otherwise admission.Denied
	//
	// In order to avoid the helm chart is changed later, so storing the Helm manifests to a secret for later use
	// and do not re-fetch the Helm chart from remote in helmchart controller reconcile function

	userInfo := req.UserInfo

	if req.Operation == admissionv1.Create || req.Operation == admissionv1.Update {
		manifests, err := helm.GetManifests(helmChart.Name, helmChart.Namespace, helmChart.Spec.Chart.Path, helmChart.Spec.Values.Raw)
		if err != nil {
			log.Error(err, "failed to get Helm manifests")
			return admission.Errored(http.StatusBadRequest, err)
		}

		for _, m := range manifests {
			obj, _ := yaml.YamlToObject(m)
			obj.SetNamespace(helmChart.Namespace)
			status, err := h.checkPermission(ctx, userInfo, obj)
			if err != nil {
				log.Error(err, "failed to check permission")
				return admission.Errored(http.StatusBadRequest, err)
			}
			if status.Allowed == false {
				log.Info("not allowed to create", "resource", obj.GetKind(), "reason", status.Reason)
				reason := status.Reason
				if reason == "" {
					reason = "not allowed to create " + obj.GetKind()
				}
				return admission.Denied(reason)
			}
		}

		sep := []byte("\n---\n")
		manifestsBytes := bytes.Join(manifests, sep)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.ManifestsSecretName(helmChart.Name, helmChart.Namespace),
				Namespace: constant.HelmOperatorNamespace,
			},

			Data: map[string][]byte{
				"manifests": manifestsBytes,
			},
		}

		log.V(2).Info("storing Helm manifests to a secret")
		if err := h.Client.Create(ctx, secret); err != nil {
			if !errors.IsAlreadyExists(err) {
				log.Error(err, "failed to create the manifests secret")
				return admission.Errored(http.StatusBadRequest, err)
			}
			log.V(2).Info("update the manifests secret with new manifestsBytes")
			if err := h.Client.Update(ctx, secret); err != nil {
				log.Error(err, "failed to update the manifests secret")
				return admission.Errored(http.StatusBadRequest, err)
			}
		}
	}

	return admission.Allowed("")
}

func (h *validatingHandler) checkPermission(ctx context.Context, userInfo authenticationv1.UserInfo, obj *unstructured.Unstructured) (authorizationv1.SubjectAccessReviewStatus, error) {
	mapper, err := h.Client.RESTMapper().RESTMapping(obj.GroupVersionKind().GroupKind(), obj.GroupVersionKind().Version)
	if err != nil {
		return authorizationv1.SubjectAccessReviewStatus{}, err
	}
	sar := &authorizationv1.SubjectAccessReview{
		Spec: authorizationv1.SubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: obj.GetNamespace(),
				Verb:      "*",
				Group:     obj.GroupVersionKind().Group,
				Version:   obj.GroupVersionKind().Version,
				Resource:  mapper.Resource.Resource,
			},
			UID:    userInfo.UID,
			User:   userInfo.Username,
			Groups: userInfo.Groups,
			Extra:  convertToSARExtra(userInfo.Extra),
		},
	}

	if err := h.Client.Create(ctx, sar); err != nil {
		return authorizationv1.SubjectAccessReviewStatus{}, err
	}

	return sar.Status, nil
}

func convertToSARExtra(extra map[string]authenticationv1.ExtraValue) map[string]authorizationv1.ExtraValue {
	if extra == nil {
		return nil
	}
	ret := map[string]authorizationv1.ExtraValue{}
	for k, v := range extra {
		ret[k] = authorizationv1.ExtraValue(v)
	}

	return ret
}

func (h *validatingHandler) InjectDecoder(d *admission.Decoder) error {
	h.Decoder = d
	return nil
}
