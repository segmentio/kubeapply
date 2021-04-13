package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	denyPolicyStr = `
package com.segment.kubeapply

deny[msg] {
	input.apiVersion == "badVersion"
	msg = "Cannot have bad api version"
}

deny[msg] {
	input.extraKey == "extraBadValue"
	msg = "Cannot have bad extra key"
}

deny[msg] {
	input.extraKey2 == "warnValue"
	msg = "WARN: Cannot have warn value"
}`

	allowPolicyStr = `
package com.segment.kubeapply

default allow = true

allow = false {
	input.apiVersion == "badVersion"
}`

	goodVersionResourceStr = `
apiVersion: goodVersion
kind: Deployment
metadata:
  name: test`

	badVersionResourceStr = `
apiVersion: badVersion
kind: Deployment
metadata:
  name: test`
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
				Package:  "com.segment.kubeapply",
				Result:   "deny",
			},
			resource: MakeResource("test/path", []byte(goodVersionResourceStr), 0),
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
				Package:  "com.segment.kubeapply",
				Result:   "deny",
				ExtraFields: map[string]interface{}{
					"extraKey": "goodValue",
				},
			},
			resource: MakeResource("test/path", []byte(badVersionResourceStr), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testDenyPolicy",
				Status:    StatusInvalid,
				Message:   "Policy returned 1 deny reason(s)",
				Reasons: []string{
					"Cannot have bad api version",
				},
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testDenyPolicy",
				Contents: denyPolicyStr,
				Package:  "com.segment.kubeapply",
				Result:   "deny",
				ExtraFields: map[string]interface{}{
					"extraKey2": "warnValue",
				},
			},
			resource: MakeResource("test/path", []byte(goodVersionResourceStr), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testDenyPolicy",
				Status:    StatusWarning,
				Message:   "Policy returned 1 warn reason(s)",
				Reasons: []string{
					"WARN: Cannot have warn value",
				},
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testDenyPolicy",
				Contents: denyPolicyStr,
				Package:  "com.segment.kubeapply",
				Result:   "deny",
				ExtraFields: map[string]interface{}{
					"extraKey": "extraBadValue",
				},
			},
			resource: MakeResource("test/path", []byte(badVersionResourceStr), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testDenyPolicy",
				Status:    StatusInvalid,
				Message:   "Policy returned 2 deny reason(s)",
				Reasons: []string{
					"Cannot have bad extra key",
					"Cannot have bad api version",
				},
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testDenyPolicy",
				Contents: denyPolicyStr,
				Package:  "com.segment.kubeapply",
				Result:   "deny",
				ExtraFields: map[string]interface{}{
					"extraKey":  "extraBadValue",
					"extraKey2": "warnValue",
				},
			},
			resource: MakeResource("test/path", []byte(badVersionResourceStr), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testDenyPolicy",
				Status:    StatusInvalid,
				Message:   "Policy returned 2 deny reason(s) and 1 warn reason(s)",
				Reasons: []string{
					"Cannot have bad extra key",
					"Cannot have bad api version",
					"WARN: Cannot have warn value",
				},
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testAllowPolicy",
				Contents: allowPolicyStr,
				Package:  "com.segment.kubeapply",
				Result:   "allow",
			},
			resource: MakeResource("test/path", []byte(goodVersionResourceStr), 0),
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
				Package:  "com.segment.kubeapply",
				Result:   "allow",
			},
			resource: MakeResource("test/path", []byte(badVersionResourceStr), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testAllowPolicy",
				Status:    StatusInvalid,
				Message:   "Policy returned allowed = false",
			},
		},
		{
			policyModule: PolicyModule{
				Name:     "testAllowPolicy",
				Contents: allowPolicyStr,
				Package:  "com.segment.kubeapply",
				Result:   "allow",
			},
			resource: MakeResource("test/path", []byte(""), 0),
			expected: CheckResult{
				CheckType: CheckTypeOPA,
				CheckName: "testAllowPolicy",
				Status:    StatusEmpty,
				Message:   "No resource content",
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
