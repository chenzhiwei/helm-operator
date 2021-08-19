package utils

import (
	"reflect"

	appv1 "github.com/chenzhiwei/helm-operator/api/v1"
)

func GetDeletedResources(resources, currentResources []appv1.Resource) []appv1.Resource {
	var result []appv1.Resource
	for _, resource := range currentResources {
		if !containResource(resources, resource) {
			result = append(result, resource)
		}
	}

	return result
}

func containResource(resources []appv1.Resource, resource appv1.Resource) bool {
	for _, res := range resources {
		if reflect.DeepEqual(res, resource) {
			return true
		}
	}

	return false
}
