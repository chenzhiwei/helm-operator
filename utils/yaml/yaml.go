package yaml

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"strconv"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	goyaml "sigs.k8s.io/yaml"
)

func YamlToObject(yamlContent []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	jsonSpec, err := goyaml.YAMLToJSON(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("Failed to convert yaml to json: %v", err)
	}

	if err := obj.UnmarshalJSON(jsonSpec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal resource: %v", err)
	}

	return obj, nil
}

func yamlToObjects(yamlContent []byte) ([]*unstructured.Unstructured, error) {
	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	var objects []*unstructured.Unstructured
	reader := json.YAMLFramer.NewFrameReader(ioutil.NopCloser(bytes.NewReader(yamlContent)))
	decoder := streaming.NewDecoder(reader, yamlDecoder)
	for {
		obj, _, err := decoder.Decode(nil, nil)
		if err != nil {
			if err == io.EOF {
				break
			}
			continue
		}

		switch t := obj.(type) {
		case *unstructured.Unstructured:
			objects = append(objects, t)
		default:
			return nil, fmt.Errorf("Failed to convert object %s", reflect.TypeOf(obj))
		}
	}

	return objects, nil
}

func getObject(obj *unstructured.Unstructured, reader client.Reader) (*unstructured.Unstructured, error) {
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())

	err := reader.Get(context.TODO(), types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, found)

	return found, err
}

func createObject(obj *unstructured.Unstructured, client client.Client) error {
	err := client.Create(context.TODO(), obj)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("Failed to create resource: %v", err)
	}

	return nil
}

func updateObject(obj *unstructured.Unstructured, client client.Client) error {
	if err := client.Update(context.TODO(), obj); err != nil {
		return fmt.Errorf("Failed to update resource: %v", err)
	}

	return nil
}

func deleteObject(obj *unstructured.Unstructured, client client.Client) error {
	if err := client.Delete(context.TODO(), obj); err != nil {
		return fmt.Errorf("Failed to delete resource: %v", err)
	}

	return nil
}

func CreateOrUpdateFromBytes(content []byte, client client.Client, reader client.Reader) error {
	objects, err := yamlToObjects(content)
	if err != nil {
		return err
	}

	var errMsg string

	for _, obj := range objects {
		// gvk := obj.GetObjectKind().GroupVersionKind()

		objInCluster, err := getObject(obj, reader)
		if errors.IsNotFound(err) {
			if err := createObject(obj, client); err != nil {
				// klog.Infof("create resource with name: %s, namespace: %s, kind: %s, apiversion: %s/%s\n", obj.GetName(), obj.GetNamespace(), gvk.Kind, gvk.Group, gvk.Version)
				errMsg = errMsg + err.Error()
			}
			continue
		} else if err != nil {
			errMsg = errMsg + err.Error()
			continue
		}

		annoVersion := obj.GetAnnotations()["version"]
		if annoVersion == "" {
			annoVersion = "0"
		}
		annoVersionInCluster := objInCluster.GetAnnotations()["version"]
		if annoVersionInCluster == "" {
			annoVersionInCluster = "0"
		}

		version, _ := strconv.Atoi(annoVersion)
		versionInCluster, _ := strconv.Atoi(annoVersionInCluster)
		if version > versionInCluster {
			// Deepin merge and update the object
		}
	}

	if errMsg != "" {
		return fmt.Errorf("Failed to create resource: %v", errMsg)
	}

	return nil
}

func CreateOrUpdateFromYaml(path string, client client.Client, reader client.Reader) error {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	return CreateOrUpdateFromBytes(content, client, reader)
}
