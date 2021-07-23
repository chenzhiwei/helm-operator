package helm

import (
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/releaseutil"
	"sigs.k8s.io/yaml"
)

func getChart(path string) (*chart.Chart, error) {
	client := action.NewInstall(&action.Configuration{})
	settings := cli.New()
	cp, err := client.ChartPathOptions.LocateChart(path, settings)
	if err != nil {
		return nil, err
	}

	chart, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	return chart, nil
}

func getValues(name, namespace string, bytes []byte, chart *chart.Chart) (chartutil.Values, error) {
	rawMap := map[string]interface{}{}
	if err := yaml.Unmarshal(bytes, &rawMap); err != nil {
		return nil, err
	}

	options := chartutil.ReleaseOptions{
		Name:      name,
		Namespace: namespace,
	}

	values, err := chartutil.ToRenderValues(chart, rawMap, options, nil)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func GetManifests(name, namespace, path string, bytes []byte) ([]releaseutil.Manifest, error) {
	chart, err := getChart(path)
	if err != nil {
		return nil, err
	}

	values, err := getValues(name, namespace, bytes, chart)
	if err != nil {
		return nil, err
	}

	files, err := engine.Render(chart, values)
	if err != nil {
		return nil, err
	}

	for k := range files {
		if strings.HasSuffix(k, ".txt") || strings.HasPrefix(k, "_") {
			delete(files, k)
		}
	}

	// Don't handle hooks
	_, manifests, err := releaseutil.SortManifests(files, nil, releaseutil.InstallOrder)
	if err != nil {
		return nil, err
	}

	return manifests, err
}
