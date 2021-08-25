# Helm Operator

Helm Operator is designed to install and manage Helm charts with Kubernetes CRD resource.

Helm Operator does not create the Helm releases, it only uses Helm as the template engine to generate the Kubernetes resources.

Helm Operator enables Server Side Apply and enforces the fields ownership.

More features are on the way!


## How to install

```
kubectl apply -f https://github.com/chenzhiwei/helm-operator/raw/master/config/allinone.yaml
```

By default, it will create following resources:

* helm-operator Namespace, which the operator deployment runs in
* cluster-admin-helm-operator ClusterRoleBinding, which gives cluster-admin permission to the operator
* helm-operator Deployment, the operator deployment
* helmcharts.app.siji.io CRD, defines the chart resource
* helmdogs.app.siji.io CRD, used by HelmChart to clean up cluster scoped and non-cr namespace resources

Run following commands to uninstall:

```
kubectl delete helmchart --all --all-namespaces
kubectl delete helmdog --all --all-namespaces
kubectl delete namespace helm-operator
kubectl delete crd helmcharts.app.siji.io helmdogs.app.siji.io
```

## How to install with webhook enabled

```
kubectl apply -f https://github.com/chenzhiwei/helm-operator/raw/master/config/allinone-webhook.yaml
```

By default, it will create the normal resources and plus following:

* helm-operator-webhook-service, which is used for validating webhook
* helm-operator-validating-webhook, a `ValidatingWebhookConfiguration` to validate the permissions

Run following commands to uninstall:

```
kubectl delete validatingwebhookconfiguration helm-operator-validating-webhook
kubectl delete helmchart --all --all-namespaces
kubectl delete helmdog --all --all-namespaces
kubectl delete namespace helm-operator
kubectl delete crd helmcharts.app.siji.io helmdogs.app.siji.io
```

## How to use

Create a `HelmChart` CR, and this operator will install the resources inside the Helm chart.

```
apiVersion: app.siji.io/v1
kind: HelmChart
metadata:
  name: helmchart-sample
spec:
  chart:
    path: https://gitlab.com/chenzhiwei/charts/-/raw/master/release/nginx-0.1.0.tgz
  values:
    replicaCount: 2
    image:
      repository: docker.io/siji/nginx
      tag: latest
```


## Design Idea

Helm is a very popular package tool for Kubernetes, but it also has some limitations, especially handling CRDs.

This Helm Operator leverages the Kubernetes CustomResourceDefinition to manage the full lifecycle of a Helm chart.

Users can create a `HelmChart` CR with Helm chart path and values, the operator will use Helm library to generate the final manifests and then call the Kubernetes API to CRUD on these manifests.

For Helm chart manifests have same namespace with the `HelmChart` CR, the operator will add an ownerreference to these manifests; for those manifests who are cluster scoped or in different namespaces, the operator will create another `HelmDog` CR to store them for later update or delete.

When a Helm chart is updated, there may have newly added and removed manifests, the operator will find the diff and perform creating or removing actions on them.


## Features

1. Share same resource in multiple charts

    This can be achieved by setting an annotation `app.siji.io/keep=anything`.

    A use case is a ConfigMap contains some metadata, and multiple charts share this single ConfigMap.

2. Force clean up the CRDs in a chart when uninstalling

    This can be achieved by setting an annotation `app.siji.io/force-crd-delete=anything`.

3. Runtime control on installed Helm charts

    When users update the Helm chart objects, the operator will rollback them. Users should update the `HelmChart` CR to update the objects.

4. Fine-grained permission control

   This is used to ensure the user who create the `HelmChart` has the permission to create the resources inside the Helm chart.

   Users can enable the ValidatingWebhookConfiguration and each Create or Update operation will be validated to ensure the user has right permission.

5. Helm chart in standard OCI/Docker image(WIP)

    In Kubernetes, all workloads are image based, setting up a Helm registry or HTTP server is a little annoying.

    We can put the Helm chart directory or `.tgz` package into a standard OCI/Docker image, the only rule is we have an agreement to put it into last layer.

    This operator can call image registry API to fetch the last layer and get the helm chart package.

    An example is: `docker://docker.io/siji/helm-chart:latest#file=nginx-0.2.0.tgz`, the Helm chart package `nginx-0.2.0.tgz` is in the last layer of this image.


## Limitations

Do not support hooks and dependencies.
