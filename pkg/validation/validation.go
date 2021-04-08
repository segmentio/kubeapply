package validation

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yannh/kubeconform/pkg/validator"
)

// KubeValidator is a struct that validates the kube configs associated with a cell config.
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

// RunKubeconform runs kubeconform over all files in the provided path.
func (k *KubeValidator) RunKubeconform(
	ctx context.Context,
	path string,
) ([]ValidationResult, error) {
	results := []ValidationResult{}

	err := filepath.Walk(
		path,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.HasSuffix(subPath, ".yaml") {
				return nil
			}

			file, err := os.Open(subPath)
			if err != nil {
				return err
			}

			for _, kResult := range k.validatorObj.Validate(subPath, file) {
				result := ValidationResult{
					Filename: kResult.Resource.Path,
					Status:   kubeconformStatusToStatus(kResult.Status),
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

			return nil
		},
	)

	return results, err
}

func kubeconformStatusToStatus(kStatus validator.Status) Status {
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
