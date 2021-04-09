package validation

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

type CheckType string

const (
	CheckTypeKubeconform CheckType = "kubeconform"
	CheckTypeOPA         CheckType = "opa"
)

// Result stores the results of validating a single resource in a single file.
type Result struct {
	Resource     Resource
	CheckResults []CheckResult
}

type CheckResult struct {
	CheckType CheckType
	CheckName string
	Status    Status
	Message   string
}
