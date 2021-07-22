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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/chenzhiwei/helm-operator/api/v1"
	"github.com/chenzhiwei/helm-operator/utils/constant"
	"github.com/chenzhiwei/helm-operator/utils/helm"
)

// HelmChartReconciler reconciles a HelmChart object
type HelmChartReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=app.k8s.io,resources=helmcharts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.k8s.io,resources=helmcharts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.k8s.io,resources=helmcharts/finalizers,verbs=update

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
	logger := log.FromContext(ctx)

	cr := &appv1.HelmChart{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		logger.Error(err, "Failed to get HelmChart")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// handle finalizer logic
	if cr.DeletionTimestamp == nil {
		// add finalizer
		if !controllerutil.ContainsFinalizer(cr, constant.FinalizerName) {
			controllerutil.AddFinalizer(cr, constant.FinalizerName)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The CR is being deleted
		// delete external resource
		if err := r.cleanResources(cr); err != nil {
			return ctrl.Result{}, err
		}
		// delete finalizer
		if controllerutil.ContainsFinalizer(cr, constant.FinalizerName) {
			controllerutil.RemoveFinalizer(cr, constant.FinalizerName)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	manifests, err := helm.GetManifests(cr.Name, cr.Namespace, cr.Spec.Chart, cr.Spec.Values.Raw)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, m := range manifests {
		logger.Info("Deploying file name", m.Name)
		logger.Info("Deploying file content", m.Content)
	}

	return ctrl.Result{}, nil
}

func (r *HelmChartReconciler) cleanResources(cr *appv1.HelmChart) error {
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmChartReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.HelmChart{}).
		Complete(r)
}