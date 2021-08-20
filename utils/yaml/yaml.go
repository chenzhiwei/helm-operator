package yaml

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/chenzhiwei/helm-operator/utils/pointer"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func YamlToObject(yamlContent []byte) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	jsonSpec, err := yaml.YAMLToJSON(yamlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to convert yaml to json: %v", err)
	}

	if err := obj.UnmarshalJSON(jsonSpec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal resource: %v", err)
	}

	return obj, nil
}

func CreateOrUpdateFromFiles(c client.Client, files []string) error {
	ctx := context.TODO()

	var errMsg []string
	for _, path := range files {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			errMsg = append(errMsg, err.Error())
			continue
		}

		if err := serverSideApply(ctx, c, content); err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}

	if len(errMsg) > 0 {
		return fmt.Errorf("failed to create resources: %s", strings.Join(errMsg, ";"))
	}

	return nil
}

func serverSideApply(ctx context.Context, c client.Client, content []byte) error {
	obj, err := YamlToObject(content)
	if err != nil {
		return err
	}

	patchOptions := &client.PatchOptions{
		FieldManager: "helmchart-controller",
		Force:        pointer.Bool(true),
	}

	return c.Patch(ctx, obj, client.Apply, patchOptions)
}
