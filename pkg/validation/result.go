package validation

// Status stores the result of validating a single file or resource.
type Status string

const (
	StatusValid   Status = "valid"
	StatusInvalid Status = "invalid"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
	StatusSkipped Status = "skipped"
	StatusEmpty   Status = "empty"
)

// CheckType represents the type of check that has been done.
type CheckType string

const (
	CheckTypeKubeconform CheckType = "kubeconform"
	CheckTypeOPA         CheckType = "opa"
)

// Result stores the results of validating a single resource in a single file, for all checks.
type ResourceResult struct {
	Resource     Resource
	CheckResults []CheckResult
}

// CheckResult contains the detailed results of a single check.
type CheckResult struct {
	CheckType CheckType
	CheckName string
	Status    Status
	Message   string
	Reasons   []string
}

// HasIssues returns whether a ResourceResult has at least one check result with an
// error or warning.
func (r ResourceResult) HasIssues() bool {
	for _, checkResult := range r.CheckResults {
		if checkResult.Status == StatusError || checkResult.Status == StatusInvalid ||
			checkResult.Status == StatusWarning {
			return true
		}
	}

	return false
}

// CountsByStatus returns the number of check results for each status type.
func CountsByStatus(results []ResourceResult) map[Status]int {
	counts := map[Status]int{}

	for _, result := range results {
		for _, checkResult := range result.CheckResults {
			counts[checkResult.Status]++
		}
	}

	return counts
}

// ResultsWithIssues filters the argument resource results to just those with potential
// issues.
func ResultsWithIssues(results []ResourceResult) []ResourceResult {
	filteredResults := []ResourceResult{}

	for _, result := range results {
		if result.HasIssues() {
			filteredResults = append(filteredResults, result)
		}
	}

	return filteredResults
}
