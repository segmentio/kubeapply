package validation

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	_ "github.com/open-policy-agent/opa/rego"
)

// numWorkers is the number of workers that we will use in parallel to run the validation process.
const numWorkers = 4

var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

// KubeValidator is a struct that validates the kube configs associated with a cluster.
type KubeValidator struct {
	config KubeValidatorConfig
}

type KubeValidatorConfig struct {
	NumWorkers int
	Checkers   []Checker
}

// NewKubeValidator returns a new KubeValidator instance.
func NewKubeValidator(config KubeValidatorConfig) *KubeValidator {
	return &KubeValidator{
		config: config,
	}
}

// RunChecks runs all checks over all resources in the path and returns the results.
func (k *KubeValidator) RunChecks(
	ctx context.Context,
	path string,
) ([]Result, error) {
	resources := []Resource{}
	index := 0

	// First, get all of the resources.
	err := filepath.Walk(
		path,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(subPath, ".yaml") {
				return nil
			}

			contents, err := ioutil.ReadFile(subPath)
			if err != nil {
				return err
			}
			manifestStrs := sep.Split(string(contents), -1)

			for _, manifestStr := range manifestStrs {
				trimmedManifest := strings.TrimSpace(manifestStr)
				if len(trimmedManifest) == 0 {
					continue
				}

				resources = append(
					resources,
					MakeResource(subPath, []byte(trimmedManifest), index),
				)
				index++
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	resourcesChan := make(chan Resource, len(resources))
	for _, resource := range resources {
		resourcesChan <- resource
	}
	defer close(resourcesChan)

	resultsChan := make(chan Result, len(resources))

	for i := 0; i < numWorkers; i++ {
		go func() {
			for resource := range resourcesChan {
				result := Result{
					Resource: resource,
				}
				for _, checker := range k.config.Checkers {
					result.CheckResults = append(
						result.CheckResults,
						checker.Check(ctx, resource),
					)
				}

				resultsChan <- result
			}
		}()
	}

	results := []Result{}
	for i := 0; i < len(resources); i++ {
		results = append(results, <-resultsChan)
	}

	sort.Slice(results, func(a, b int) bool {
		return results[a].Resource.index < results[b].Resource.index
	})

	return results, nil
}
