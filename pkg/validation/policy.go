package validation

import (
	"context"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/open-policy-agent/opa/rego"
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
			fmt.Sprintf("result = %s.%s", module.Package, module.Result),
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

	allowed, ok := results[0].Bindings["result"].(bool)

	if !ok {
		result.Status = StatusError
		result.Message = fmt.Sprintf("Did not get exactly one result: %+v", results)
		return result
	} else if !allowed {
		result.Status = StatusInvalid
		result.Message = "Policy returned allowed = false"
		return result
	} else {
		result.Status = StatusValid
		result.Message = "Policy returned allowed = true"
		return result
	}
}
