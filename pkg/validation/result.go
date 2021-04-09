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

// CheckType represents the type of check that has been done.
type CheckType string

const (
	CheckTypeKubeconform CheckType = "kubeconform"
	CheckTypeOPA         CheckType = "opa"
)

// Result stores the results of validating a single resource in a single file, for all checks.
type Result struct {
	Resource     Resource
	CheckResults []CheckResult
}

// CheckResult contains the detailed results of a single check.
type CheckResult struct {
	CheckType CheckType
	CheckName string
	Status    Status
	Message   string
}
