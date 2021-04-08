package validation

import (
	"context"
	"fmt"
	"sync"

	"github.com/yannh/kubeconform/pkg/resource"
	"github.com/yannh/kubeconform/pkg/validator"
)

const (
	numWorkers = 6
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

// ValidationResult stores the results of validating a single resource in a single file.
type ValidationResult struct {
	Filename  string `json:"filename"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
	Status    Status `json:"status"`
	Message   string `json:"msg"`
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

// RunSchemaValidation runs kubeconform over all files in the provided path and returns the result.
func (k *KubeValidator) RunSchemaValidation(
	ctx context.Context,
	path string,
) ([]ValidationResult, error) {
	kResults := []validator.Result{}
	resourcesChan, errChan := resource.FromFiles(ctx, []string{path}, nil)
	mut := sync.Mutex{}
	wg := sync.WaitGroup{}

	// Based on implementation in
	// https://github.com/yannh/kubeconform/blob/master/cmd/kubeconform/main.go.
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			for res := range resourcesChan {
				mut.Lock()
				kResults = append(kResults, k.validatorObj.ValidateResource(res))
				mut.Unlock()
			}
			wg.Done()
		}()
	}

	wg.Add(1)
	go func() {
		// Process errors while discovering resources
		for err := range errChan {
			if err == nil {
				continue
			}

			var kResult validator.Result

			if err, ok := err.(resource.DiscoveryError); ok {
				kResult = validator.Result{
					Resource: resource.Resource{Path: err.Path},
					Err:      err.Err,
					Status:   validator.Error,
				}
			} else {
				kResult = validator.Result{
					Resource: resource.Resource{},
					Err:      err,
					Status:   validator.Error,
				}
			}
			mut.Lock()
			kResults = append(kResults, kResult)
			mut.Unlock()
		}
		wg.Done()
	}()
	wg.Wait()

	results := []ValidationResult{}

	for _, kResult := range kResults {
		if kResult.Status == validator.Empty {
			// Skip over empty results
			continue
		}

		result := ValidationResult{
			Filename: kResult.Resource.Path,
			Status:   kStatusToStatus(kResult.Status),
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
