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
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appv1 "github.com/chenzhiwei/helm-operator/api/v1"
	"github.com/chenzhiwei/helm-operator/utils"
)

// HelmDogReconciler reconciles a HelmDog object
type HelmDogReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.siji.io,resources=helmdogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.siji.io,resources=helmdogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.siji.io,resources=helmdogs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelmDog object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *HelmDogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	cr := &appv1.HelmDog{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "Failed to get HelmDog")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// The CR is being deleted
	if cr.DeletionTimestamp != nil {
		if err := r.deleteResources(ctx, cr.Spec.Resources); err != nil {
			log.Error(err, "Failed to delete resources")
			return ctrl.Result{}, err
		}

		cr.SetFinalizers(nil)
		if err := r.Update(ctx, cr); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "Failed to remove finalizers")
			}
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}

		return ctrl.Result{}, nil
	}

	// No resources in CR, ignore it
	if len(cr.Spec.Resources) == 0 {
		return ctrl.Result{}, nil
	}

	// Remove the unused resources
	resources := utils.GetDeletedResources(cr.Spec.Resources, cr.Status.Resources)
	if len(resources) == 0 {
		return ctrl.Result{}, nil
	} else {
		if err := r.deleteResources(ctx, resources); err != nil {
			log.Error(err, "Failed to delete resources")
			return ctrl.Result{}, err
		}

		cr.Status.Resources = cr.Spec.Resources
		if err := r.Status().Update(ctx, cr); err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *HelmDogReconciler) deleteResources(ctx context.Context, resources []appv1.Resource) error {
	log := ctrl.LoggerFrom(ctx)

	var errMsg []string
	for i := len(resources) - 1; i >= 0; i-- {
		res := resources[i]
		log.Info("Delete Resource", "group", res.Group, "version", res.Version, "kind", res.Kind, "name", res.Name, "namespace", res.Namespace)
		if err := r.deleteResource(ctx, res); err != nil {
			errMsg = append(errMsg, fmt.Sprintf("Failed to delete: %s.%s/%s, name: %s, namespace: %s, msg: %s", res.Kind, res.Group, res.Version, res.Name, res.Namespace, err.Error()))
		}
	}

	if len(errMsg) > 0 {
		return fmt.Errorf(strings.Join(errMsg, ","))
	}

	return nil
}

func (r *HelmDogReconciler) deleteResource(ctx context.Context, res appv1.Resource) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   res.Group,
		Version: res.Version,
		Kind:    res.Kind,
	})

	if err := r.Get(ctx, types.NamespacedName{Name: res.Name, Namespace: res.Namespace}, obj); err != nil {
		return client.IgnoreNotFound(err)
	}

	// Do not delete the resource if it has annotation app.siji.io/keep
	// This is mainly for multiple charts share same resource
	if _, ok := obj.GetAnnotations()["app.siji.io/keep"]; ok {
		return nil
	}

	// Do not delete the CRD
	if res.Kind == "CustomResourceDefinition" {
		return nil
	}

	// Delete the resource
	if err := r.Delete(ctx, obj); err != nil {
		return client.IgnoreNotFound(err)
	}

	// Trying to get the resource
	if err := r.Get(ctx, types.NamespacedName{Name: res.Name, Namespace: res.Namespace}, obj); err != nil {
		return client.IgnoreNotFound(err)
	}

	// Force delete the resource by removing its finalizers
	if len(obj.GetFinalizers()) > 0 {
		obj.SetFinalizers(nil)
		if err := r.Update(ctx, obj); err != nil {
			return client.IgnoreNotFound(err)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmDogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.HelmDog{}).
		Complete(r)
}
