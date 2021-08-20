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


## Limitations

Do not support hooks and dependencies.


## TODO

* Enable validating webhook

   This is used to ensure the user who create the `HelmChart` has the permission to create the resources inside the Helm chart.
