package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	denyPolicyStr = `
package example

deny[msg] {
	input.apiVersion == "badVersion"
	msg = "Cannot have bad api version"
}`

	allowPolicyStr = `
package example

default allow = true

allow = false {
	input.apiVersion == "badVersion"
}`
)

func TestPolicyChecker(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		policyModule PolicyModule
		resource     Resource
		expected     CheckResult
	}

	testCases := []testCase{
		{
			policyModule: PolicyModule{
				Name:     "testDenyPolicy",
				Contents: denyPolicyStr,
				Package:  "example",
				Result:   "deny",
			},
			resource: MakeResource("test/path", []byte("apiVersion: goodVersion"), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testDenyPolicy",
				Status:    StatusValid,
				Message:   "Policy returned 0 deny reasons",
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testDenyPolicy",
				Contents: denyPolicyStr,
				Package:  "example",
				Result:   "deny",
			},
			resource: MakeResource("test/path", []byte("apiVersion: badVersion"), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testDenyPolicy",
				Status:    StatusInvalid,
				Message:   "Policy returned 1 deny reason(s): [Cannot have bad api version]",
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testAllowPolicy",
				Contents: allowPolicyStr,
				Package:  "example",
				Result:   "allow",
			},
			resource: MakeResource("test/path", []byte("apiVersion: goodVersion"), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testAllowPolicy",
				Status:    StatusValid,
				Message:   "Policy returned allowed = true",
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testAllowPolicy",
				Contents: allowPolicyStr,
				Package:  "example",
				Result:   "allow",
			},
			resource: MakeResource("test/path", []byte("apiVersion: badVersion"), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testAllowPolicy",
				Status:    StatusInvalid,
				Message:   "Policy returned allowed = false",
			},
		},
	}

	for _, testCase := range testCases {
		checker, err := NewPolicyChecker(
			ctx,
			testCase.policyModule,
		)
		require.NoError(t, err)

		result := checker.Check(ctx, testCase.resource)
		assert.Equal(t, testCase.expected, result)
	}
}
