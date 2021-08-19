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

package controllers

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appv1 "github.com/chenzhiwei/helm-operator/api/v1"
	"github.com/chenzhiwei/helm-operator/utils/constant"
	"github.com/chenzhiwei/helm-operator/utils/helm"
	"github.com/chenzhiwei/helm-operator/utils/pointer"
	"github.com/chenzhiwei/helm-operator/utils/yaml"
)

// HelmChartReconciler reconciles a HelmChart object
type HelmChartReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.siji.io,resources=helmcharts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.siji.io,resources=helmcharts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.siji.io,resources=helmcharts/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelmChart object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *HelmChartReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	cr := &appv1.HelmChart{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to get HelmChart")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The CR is being deleted
	if cr.DeletionTimestamp != nil {
		// delete resources in other namespaces or cluster scoped resources
		if err := r.cleanResources(ctx, cr); err != nil {
			log.Error(err, "Failed to delete HelmDog "+req.Name)
			return ctrl.Result{}, err
		}

		// delete finalizer
		if controllerutil.ContainsFinalizer(cr, constant.FinalizerName) {
			controllerutil.RemoveFinalizer(cr, constant.FinalizerName)
			if err := r.Update(ctx, cr); err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// add finalizer
	if !controllerutil.ContainsFinalizer(cr, constant.FinalizerName) {
		controllerutil.AddFinalizer(cr, constant.FinalizerName)
		if err := r.Update(ctx, cr); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	manifests, err := helm.GetManifests(cr.Name, cr.Namespace, cr.Spec.Chart.Path, cr.Spec.Values.Raw)
	if err != nil {
		log.Error(err, "Failed to generate Helm manifests")
		return ctrl.Result{}, err
	}

	var resources []appv1.Resource

	// var objects []*unstructured.Unstructured
	for _, m := range manifests {
		obj, _ := yaml.YamlToObject(m)

		mapper, err := r.Client.RESTMapper().RESTMapping(obj.GroupVersionKind().GroupKind(), obj.GroupVersionKind().Version)
		if err != nil {
			log.Error(err, "Failed to get RESTMapper")
			return ctrl.Result{}, err
		}

		// set namespace to obj because Helm does not add it
		ns := obj.GetNamespace()
		if ns == "" && mapper.Scope.Name() == "namespace" {
			obj.SetNamespace(cr.Namespace)
		}

		if obj.GetNamespace() == cr.Namespace {
			if err := controllerutil.SetControllerReference(cr, obj, r.Scheme); err != nil {
				log.Error(err, "Failed to set owner reference")
				return ctrl.Result{}, err
			}
		} else {
			// store the cluster scoped resource for cleanResources
			resource := appv1.Resource{
				Group:     obj.GroupVersionKind().Group,
				Version:   obj.GroupVersionKind().Version,
				Kind:      obj.GroupVersionKind().Kind,
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			}
			resources = append(resources, resource)
		}

		log.Info("Creating Helm manifest", "Object", obj)
		// TODO: better server side apply
		patchOptions := &client.PatchOptions{
			FieldManager: "helmchart-controller",
			Force:        pointer.Bool(true),
		}
		if err := r.Patch(ctx, obj, client.Apply, patchOptions); err != nil {
			log.Error(err, "Failed running server side apply")
			return ctrl.Result{}, err
		}
	}

	if len(resources) > 0 {
		helmDog := &unstructured.Unstructured{}
		helmDog.Object = map[string]interface{}{
			"spec": map[string]interface{}{
				"resources": resources,
			},
		}

		helmDog.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   appv1.GroupVersion.Group,
			Version: appv1.GroupVersion.Version,
			Kind:    "HelmDog",
		})
		helmDog.SetName(req.Name)
		helmDog.SetNamespace(req.Namespace)

		controllerutil.AddFinalizer(helmDog, constant.FinalizerName)

		log.Info("Creating HelmDog", "Name", req.Name)
		// TODO: better server side apply
		patchOptions := &client.PatchOptions{
			FieldManager: "helmchart-controller",
			Force:        pointer.Bool(true),
		}
		if err := r.Patch(ctx, helmDog, client.Apply, patchOptions); err != nil {
			log.Error(err, "Failed running server side apply on helmdog")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HelmChartReconciler) cleanResources(ctx context.Context, cr *appv1.HelmChart) error {
	helmDog := &appv1.HelmDog{}
	helmDog.SetName(cr.Name)
	helmDog.SetNamespace(cr.Namespace)

	err := r.Delete(ctx, helmDog)
	return client.IgnoreNotFound(err)
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmChartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Only watch these widely used resources
	// The reconcile period is 5 hours which is for other resources
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.HelmChart{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&netv1.Ingress{}).
		Complete(r)
}
