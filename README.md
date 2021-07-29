# Helm Operator

Helm Operator is designed to install and manage Helm charts with Kubernetes CRD resource.

Helm Operator does not create the Helm releases, it only uses Helm as the template engine to generate the Kubernetes resources.

Currently it can only install the Helm resources, and more features are on the way!


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

## TODO

* Enable validating webhook

   This is used to ensure the user who create the `HelmChart` has the permission to create the resources inside the Helm chart.

* Enable the update operation

    This is used to ensure when user change the `HelmChart` CR, the resources can be updated to according to the change.

* Enable the clean up

    When a Helm chart is updated, the resources inside it may change, so we need to ensure removed the resources can be cleaned up.

    When a HelmChart CR is deleted, we need to ensure the cluster-scoped resources and the resources in another namespace can be cleaned up.

* Enable the Helm hook and chart dependencies support

    Helm hooks usually cause a lot of problems, so it is better to not to use hooks.

    Helm chart dependencies have a lot of limitations, so it is better to not use dependencies.
