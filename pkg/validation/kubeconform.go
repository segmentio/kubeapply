package validation

import (
	"context"

	"github.com/yannh/kubeconform/pkg/validator"
)

type KubeconformChecker struct {
	validatorObj validator.Validator
}

var _ Checker = (*KubeconformChecker)(nil)

func NewKubeconformChecker() (*KubeconformChecker, error) {
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

	return &KubeconformChecker{
		validatorObj: validatorObj,
	}, nil
}

func (k *KubeconformChecker) Check(_ context.Context, resource Resource) CheckResult {
	kResult := k.validatorObj.ValidateResource(resource.TokResource())

	var message string
	if kResult.Err != nil {
		message = kResult.Err.Error()
	}

	return CheckResult{
		CheckType: CheckTypeKubeconform,
		CheckName: "kubeconform",
		Status:    kStatusToStatus(kResult.Status),
		Message:   message,
	}
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
