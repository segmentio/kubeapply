package validation

import "context"

// Checker is an interface that checks a resource and then returns a CheckResult.
type Checker interface {
	Check(context.Context, Resource) CheckResult
}
