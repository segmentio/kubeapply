package validation

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	_ "github.com/open-policy-agent/opa/rego"
	"github.com/yannh/kubeconform/pkg/resource"
	"github.com/yannh/kubeconform/pkg/validator"
)

// KubeValidator is a struct that validates the kube configs associated with a cluster.
type KubeValidator struct {
	validatorObj validator.Validator
}

// Status stores the result of validating a single file or resource.
type Status string

const (
	StatusValid   Status = "valid"
	StatusInvalid Status = "invalid"
	StatusError   Status = "error"
	StatusSkipped Status = "skipped"
	StatusEmpty   Status = "empty"
	StatusOther   Status = "other"
)

const (
	numWorkers = 4
)

var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

// ValidationResult stores the results of validating a single resource in a single file.
type ValidationResult struct {
	Filename  string `json:"filename"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Status    Status `json:"status"`
	Message   string `json:"msg"`

	index int
}

// PrettyName returns a pretty, compact name for the resource associated with a validation result.
func (v ValidationResult) PrettyName() string {
	if v.Kind != "" && v.Version != "" && v.Name != "" && v.Namespace != "" {
		// Namespaced resources
		return fmt.Sprintf("%s.%s.%s.%s", v.Kind, v.Version, v.Namespace, v.Name)
	} else if v.Kind != "" && v.Version != "" && v.Name != "" {
		// Non-namespaced resources
		return fmt.Sprintf("%s.%s.%s", v.Kind, v.Version, v.Name)
	} else {
		return v.Name
	}
}

// NewKubeValidator returns a new KubeValidator instance.
func NewKubeValidator() (*KubeValidator, error) {
	validatorObj, err := validator.New(
		nil,
		validator.Opts{
			IgnoreMissingSchemas: true,
			Strict:               true,
		},
	)
	if err != nil {
		return nil, err
	}

	return &KubeValidator{
		validatorObj: validatorObj,
	}, nil
}

type wrappedResource struct {
	resource.Resource
	index int
}

type wrappedResult struct {
	validator.Result
	index int
}

// RunValidation runs kubeconform over all files in the provided path and returns the result.
func (k *KubeValidator) RunValidation(
	ctx context.Context,
	path string,
) ([]ValidationResult, error) {
	resources := []wrappedResource{}
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

			trimmedFile := strings.TrimSpace(string(contents))
			manifestStrs := sep.Split(trimmedFile, -1)

			for _, manifestStr := range manifestStrs {
				resources = append(
					resources,
					wrappedResource{
						Resource: resource.Resource{
							Path:  subPath,
							Bytes: []byte(manifestStr),
						},
						index: index,
					},
				)
				index++
			}

			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	resourcesChan := make(chan wrappedResource, len(resources))
	for _, resource := range resources {
		resourcesChan <- resource
	}
	defer close(resourcesChan)

	kResultsChan := make(chan wrappedResult, len(resources))

	for i := 0; i < numWorkers; i++ {
		go func() {
			for resource := range resourcesChan {
				kResultsChan <- wrappedResult{
					Result: k.validatorObj.ValidateResource(resource.Resource),
					index:  resource.index,
				}
			}
		}()
	}

	results := []ValidationResult{}
	for i := 0; i < len(resources); i++ {
		kResult := <-kResultsChan

		if kResult.Status == validator.Empty {
			// Skip over empty results
			continue
		}

		result := ValidationResult{
			Filename: kResult.Resource.Path,
			Status:   kStatusToStatus(kResult.Status),
			index:    kResult.index,
		}

		if kResult.Err != nil {
			result.Message = kResult.Err.Error()
		}

		sig, err := kResult.Resource.Signature()
		if err == nil && sig != nil {
			result.Kind = sig.Kind
			result.Name = sig.Name
			result.Namespace = sig.Namespace
			result.Version = sig.Version
		}
		results = append(results, result)
	}

	sort.Slice(results, func(a, b int) bool {
		return results[a].index < results[b].index
	})

	return results, nil
}

func kStatusToStatus(kStatus validator.Status) Status {
	switch kStatus {
	case validator.Valid:
		return StatusValid
	case validator.Invalid:
		return StatusInvalid
	case validator.Error:
		return StatusError
	case validator.Skipped:
		return StatusSkipped
	case validator.Empty:
		return StatusEmpty
	default:
		return StatusOther
	}
}
