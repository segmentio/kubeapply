package convert

import (
	"io/ioutil"

	"github.com/segmentio/kubeapply/pkg/star/expand/skymod"
	"k8s.io/apimachinery/pkg/runtime"
)

// YamlToStar converts a YAML file into a starlark representation.
func YamlToStar(filePaths []string, config Config) (string, error) {
	fileStrs := []string{}

	for _, filePath := range filePaths {
		fileBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return "", err
		}
		fileStrs = append(fileStrs, string(fileBytes))
	}

	return YamlStrToStar(fileStrs, config)
}

// YamlStrToStar converts a YAML string into a starlark representation.
func YamlStrToStar(yamlStrs []string, config Config) (string, error) {
	allObjs := []runtime.Object{}

	for _, yamlStr := range yamlStrs {
		objs, err := skymod.YamlStrToObjs(yamlStr)
		if err != nil {
			return "", err
		}
		allObjs = append(allObjs, objs...)
	}

	return ObjsToStar(allObjs, config)
}
