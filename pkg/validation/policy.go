package validation

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/open-policy-agent/opa/rego"
)

const (
	defaultPackage = "com.segment.kubeapply"
	defaultResult  = "deny"
	warnPrefix     = "warn:"
)

// Policy wraps a policy module and a prepared query.
type PolicyChecker struct {
	Module      PolicyModule
	Query       rego.PreparedEvalQuery
	ExtraFields map[string]interface{}
}

var _ Checker = (*PolicyChecker)(nil)

// PolicyModule contains information about a policy.
type PolicyModule struct {
	Name string

	// Contents is a string that stores the policy in rego format.
	Contents string

	// Package is the name of the package in the rego contents.
	Package string

	// Result is the variable that should be accessed to get the evaluation results.
	Result string

	// ExtraFields are added into the input and usable for policy evaluation.
	ExtraFields map[string]interface{}
}

// NewPolicyChecker creates a new PolicyChecker from the given module.
func NewPolicyChecker(ctx context.Context, module PolicyModule) (*PolicyChecker, error) {
	query, err := rego.New(
		rego.Query(
			fmt.Sprintf("result = data.%s.%s", module.Package, module.Result),
		),
		rego.Module(module.Package, module.Contents),
	).PrepareForEval(ctx)

	if err != nil {
		return nil, err
	}

	return &PolicyChecker{
		Module: module,
		Query:  query,
	}, nil
}

// DefaultPoliciesFromGlobs creates policy checkers from one or more file policy globs, using
// the default package and result values.
func DefaultPoliciesFromGlobs(
	ctx context.Context,
	globs []string,
	extraFields map[string]interface{},
) ([]*PolicyChecker, error) {
	checkers := []*PolicyChecker{}

	for _, glob := range globs {
		matches, err := filepath.Glob(glob)
		if err != nil {
			return nil, err
		}
		for _, match := range matches {
			contents, err := ioutil.ReadFile(match)
			if err != nil {
				return nil, err
			}

			checker, err := NewPolicyChecker(
				ctx,
				PolicyModule{
					Name:        filepath.Base(match),
					Contents:    string(contents),
					Package:     defaultPackage,
					Result:      defaultResult,
					ExtraFields: extraFields,
				},
			)
			if err != nil {
				return nil, err
			}
			checkers = append(checkers, checker)
		}
	}

	return checkers, nil
}

// Check runs a check against the argument resource using the current policy.
func (p *PolicyChecker) Check(ctx context.Context, resource Resource) CheckResult {
	result := CheckResult{
		CheckType: CheckTypeOPA,
		CheckName: p.Module.Name,
	}

	if resource.Name == "" {
		// Skip over resources that aren't likely to have any Kubernetes-related
		// structure.
		result.Status = StatusEmpty
		result.Message = fmt.Sprintf("No resource content")
		return result
	}

	data := map[string]interface{}{}

	if err := yaml.Unmarshal(resource.Contents, &data); err != nil {
		result.Status = StatusError
		result.Message = fmt.Sprintf("Error unmarshalling yaml: %+v", err)
		return result
	}

	for key, value := range p.Module.ExtraFields {
		data[key] = value
	}

	results, err := p.Query.Eval(ctx, rego.EvalInput(data))
	if err != nil {
		result.Status = StatusError
		result.Message = fmt.Sprintf("Error evaluating query: %+v", err)
		return result
	}

	if len(results) != 1 {
		result.Status = StatusError
		result.Message = fmt.Sprintf("Did not get exactly one result: %+v", results)
		return result
	}

	switch value := results[0].Bindings["result"].(type) {
	case bool:
		if value {
			result.Status = StatusValid
			result.Message = "Policy returned allowed = true"
		} else {
			result.Status = StatusInvalid
			result.Message = "Policy returned allowed = false"
		}
	case []interface{}:
		if len(value) == 0 {
			result.Status = StatusValid
			result.Message = "Policy returned 0 deny reasons"
		} else {
			invalidReasons := []string{}
			warnReasons := []string{}

			for _, subValue := range value {
				subValueStr := fmt.Sprintf("%v", subValue)

				if strings.HasPrefix(
					strings.ToLower(subValueStr),
					warnPrefix,
				) {
					// Treat this as a warning
					warnReasons = append(
						warnReasons,
						subValueStr,
					)
				} else {
					// Treat this as a denial
					invalidReasons = append(
						invalidReasons,
						subValueStr,
					)
				}
			}

			if len(invalidReasons) == 0 {
				result.Status = StatusWarning
				result.Message = fmt.Sprintf(
					"Policy returned %d warn reason(s)",
					len(warnReasons),
				)
				result.Reasons = warnReasons
			} else if len(warnReasons) == 0 {
				result.Status = StatusInvalid
				result.Message = fmt.Sprintf(
					"Policy returned %d deny reason(s)",
					len(invalidReasons),
				)
				result.Reasons = invalidReasons
			} else {
				result.Status = StatusInvalid
				result.Message = fmt.Sprintf(
					"Policy returned %d deny reason(s) and %d warn reason(s)",
					len(invalidReasons),
					len(warnReasons),
				)
				result.Reasons = append(invalidReasons, warnReasons...)
			}
		}
	default:
		result.Status = StatusError
		result.Message = fmt.Sprintf(
			"Got unexpected response type: %+v (%+v)",
			reflect.TypeOf(value),
			value,
		)
	}

	return result
}
