# Dev

```
kubebuilder init --domain siji.io --repo github.com/chenzhiwei/helm-operator

kubebuilder create api --group app --version v1 --kind HelmRelease --controller --resource

make manifests

make generate

make install

make run
```
