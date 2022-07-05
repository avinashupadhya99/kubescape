package cautils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/armosec/k8s-interface/workloadinterface"
	"github.com/armosec/opa-utils/objectsenvelopes/localworkload"
	helmchart "helm.sh/helm/v3/pkg/chart"
	helmloader "helm.sh/helm/v3/pkg/chart/loader"
	helmchartutil "helm.sh/helm/v3/pkg/chartutil"
	helmengine "helm.sh/helm/v3/pkg/engine"
)

type HelmChart struct {
	chart *helmchart.Chart
	path  string
}

func IsHelmDirectory(path string) (bool, error) {
	return helmchartutil.IsChartDir(path)
}

func NewHelmChart(path string) (*HelmChart, error) {
	chart, err := helmloader.Load(path)
	if err != nil {
		return nil, err
	}

	return &HelmChart{
		chart: chart,
		path:  path,
	}, nil
}

func (hc *HelmChart) GetName() string {
	return hc.chart.Name()
}

func (hc *HelmChart) GetDefaultValues() map[string]interface{} {
	return hc.chart.Values
}

// GetWorkloads renders chart template using the default values and returns a map of source file to its workloads
func (hc *HelmChart) GetWorkloadsWithDefaultValues() (map[string][]workloadinterface.IMetadata, []error) {
	return hc.GetWorkloads(hc.GetDefaultValues())
}

// GetWorkloads renders chart template using the provided values and returns a map of source (absolute) file path to its workloads
func (hc *HelmChart) GetWorkloads(values map[string]interface{}) (map[string][]workloadinterface.IMetadata, []error) {
	vals, err := helmchartutil.ToRenderValues(hc.chart, values, helmchartutil.ReleaseOptions{}, nil)
	if err != nil {
		return nil, []error{err}
	}

	sourceToFile, err := helmengine.Render(hc.chart, vals)
	if err != nil {
		return nil, []error{err}
	}

	workloads := make(map[string][]workloadinterface.IMetadata, 0)
	errs := []error{}

	for path, renderedYaml := range sourceToFile {
		if !IsYaml(strings.ToLower(path)) {
			continue
		}

		wls, e := ReadFile([]byte(renderedYaml), YAML_FILE_FORMAT)
		errs = append(errs, e...)
		if len(wls) == 0 {
			continue
		}
		if firstPathSeparatorIndex := strings.Index(path, string(os.PathSeparator)); firstPathSeparatorIndex != -1 {
			absPath := filepath.Join(hc.path, path[firstPathSeparatorIndex:])

			workloads[absPath] = []workloadinterface.IMetadata{}
			for i := range wls {
				lw := localworkload.NewLocalWorkload(wls[i].GetObject())
				lw.SetPath(absPath)
				workloads[absPath] = append(workloads[absPath], lw)
			}
		}

	}
	return workloads, errs
}