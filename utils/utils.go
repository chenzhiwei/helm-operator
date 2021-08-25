package utils

func ManifestsSecretName(n, ns string) string {
	return ns + "-" + n + "-" + "manifests"
}
