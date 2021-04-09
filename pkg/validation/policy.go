package validation

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/open-policy-agent/opa/rego"
)

const (
	defaultPackage = "com.segment.kubeapply"
	defaultResult  = "deny"
)

// Policy wraps a policy module and a prepared query.
type PolicyChecker struct {
	Module PolicyModule
	Query  rego.PreparedEvalQuery
}

var _ Checker = (*PolicyChecker)(nil)

// PolicyModule contains information about a policy.
type PolicyModule struct {
	Name     string
	Contents string
	Package  string
	Result   string
}

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

func DefaultPoliciesFromGlobs(
	ctx context.Context,
	globs []string,
) ([]PolicyChecker, error) {
	checkers := []PolicyChecker{}

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
					Name:     filepath.Base(match),
					Contents: string(contents),
					Package:  defaultPackage,
					Result:   defaultResult,
				},
			)
			if err != nil {
				return nil, err
			}
			checkers = append(checkers, *checker)
		}
	}

	return checkers, nil
}

func (p *PolicyChecker) Check(ctx context.Context, resource Resource) CheckResult {
	data := map[string]interface{}{}
	result := CheckResult{
		CheckType: CheckTypeOPA,
		CheckName: p.Module.Name,
	}

	if err := yaml.Unmarshal(resource.Contents, &data); err != nil {
		result.Status = StatusError
		result.Message = fmt.Sprintf("Error unmarshalling yaml: %+v", err)
		return result
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
			result.Status = StatusInvalid
			result.Message = fmt.Sprintf(
				"Policy returned %d deny reason(s): %+v",
				len(value),
				value,
			)
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
