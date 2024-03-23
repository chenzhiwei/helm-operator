/*
Copyright 2024.

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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mitchellh/hashstructure/v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1 "github.com/chenzhiwei/helm-operator/api/v1"
)

// HelmReleaseReconciler reconciles a HelmRelease object
type HelmReleaseReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

//+kubebuilder:rbac:groups=app.siji.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.siji.io,resources=helmreleases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.siji.io,resources=helmreleases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelmRelease object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.0/pkg/reconcile
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cr := &appv1.HelmRelease{}

	// get the HelmRelease cr
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	finalizerName := "release/finalizer"

	// the HelmRelease cr is being deleted
	if !cr.DeletionTimestamp.IsZero() {
		log.Info("delete the helm release")
		if err := r.uninstallHelmRelease(cr); err != nil {
			log.Error(err, "failed to uninstall the helm release")
			return ctrl.Result{}, err
		}

		if controllerutil.ContainsFinalizer(cr, finalizerName) {
			controllerutil.RemoveFinalizer(cr, finalizerName)
			if err := r.Update(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{}, nil
	}

	// record the hashcode of release Spec
	hashcode, err := hashstructure.Hash(cr.Spec, hashstructure.FormatV2, nil)
	if err != nil {
		log.Error(err, "failed to get hashcode of HelmRelease")
		return ctrl.Result{}, err
	}

	// no changes to the HelmRelease
	if cr.Status.HashedSpec == fmt.Sprintf("%d", hashcode) {
		log.Info("no changes to HelmRelease, do nothing")
		return ctrl.Result{}, nil
	}

	// install or upgrade the HelmRelease
	// TODO: install or upgrade with more acurate checks
	release := &release.Release{}
	if cr.Status.HashedSpec == "" {
		log.Info("install the HelmRelease")
		release, err = r.installHelmRelease(ctx, cr)
		if err != nil {
			log.Error(err, "failed to install the HelmRelease")
			return ctrl.Result{}, err
		}
	} else {
		log.Info("upgrade the HelmRelease")
		release, err = r.upgradeHelmRelease(ctx, cr)
		if err != nil {
			log.Error(err, "failed to upgrade the HelmRelease")
			return ctrl.Result{}, err
		}
	}

	// update hashcode to the status
	cr.Status.HashedSpec = fmt.Sprintf("%d", hashcode)
	cr.Status.Revision = strconv.Itoa(release.Version)
	cr.Status.Chart = fmt.Sprintf("%s-%s", release.Chart.Name(), release.Chart.Metadata.Version)
	if release.Info.LastDeployed.IsZero() {
		cr.Status.Updated = cr.CreationTimestamp.String()
	} else {
		cr.Status.Updated = release.Info.LastDeployed.String()
	}

	if err := r.Status().Update(ctx, cr); err != nil {
		log.Error(err, "failed to update HelmRelease status", "status", cr.Status)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *HelmReleaseReconciler) installHelmRelease(ctx context.Context, cr *appv1.HelmRelease) (*release.Release, error) {
	install := action.NewInstall(r.helmActionConfig(cr))
	install.ReleaseName = cr.Name
	install.Namespace = cr.Namespace
	install.ChartPathOptions.InsecureSkipTLSverify = cr.Spec.Chart.SkipTLSVerify

	lastIndex := strings.LastIndex(cr.Spec.Chart.Address, ":")
	install.ChartPathOptions.Version = cr.Spec.Chart.Address[lastIndex+1:]

	path, err := install.ChartPathOptions.LocateChart(cr.Spec.Chart.Address[:lastIndex], cli.New())
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	values := make(map[string]interface{})
	json.Unmarshal(cr.Spec.Values.Raw, &values)

	result, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *HelmReleaseReconciler) upgradeHelmRelease(ctx context.Context, cr *appv1.HelmRelease) (*release.Release, error) {
	upgrade := action.NewUpgrade(r.helmActionConfig(cr))
	upgrade.MaxHistory = 6
	upgrade.Namespace = cr.Namespace
	upgrade.ChartPathOptions.InsecureSkipTLSverify = cr.Spec.Chart.SkipTLSVerify

	lastIndex := strings.LastIndex(cr.Spec.Chart.Address, ":")
	upgrade.ChartPathOptions.Version = cr.Spec.Chart.Address[lastIndex+1:]

	path, err := upgrade.ChartPathOptions.LocateChart(cr.Spec.Chart.Address[:lastIndex], cli.New())
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(path)
	if err != nil {
		return nil, err
	}

	values := make(map[string]interface{})
	json.Unmarshal(cr.Spec.Values.Raw, &values)

	result, err := upgrade.RunWithContext(ctx, cr.Name, chart, values)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (r *HelmReleaseReconciler) uninstallHelmRelease(cr *appv1.HelmRelease) error {
	uninstall := action.NewUninstall(r.helmActionConfig(cr))
	if _, err := uninstall.Run(cr.Name); err != nil {
		return err
	}

	return nil
}

func (r *HelmReleaseReconciler) helmActionConfig(cr *appv1.HelmRelease) *action.Configuration {
	registryClient, _ := registry.NewRegistryClientWithTLS(io.Discard, "", "", "", cr.Spec.Chart.SkipTLSVerify, "", false)
	kubeConfig := genericclioptions.NewConfigFlags(false)
	kubeConfig.WrapConfigFn = func(*rest.Config) *rest.Config {
		return r.RestConfig
	}
	lazyClient := &lazyClient{
		namespace: cr.Namespace,
		clientFn: func() (*kubernetes.Clientset, error) {
			return kubernetes.NewForConfig(r.RestConfig)
		},
	}

	d := driver.NewSecrets(newSecretClient(lazyClient))
	store := storage.Init(d)
	return &action.Configuration{
		RegistryClient:   registryClient,
		RESTClientGetter: kubeConfig,
		KubeClient:       kube.New(kubeConfig),
		Releases:         store,
		Log:              func(format string, v ...interface{}) {},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1.HelmRelease{}).
		Complete(r)
}
