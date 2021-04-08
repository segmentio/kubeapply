package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKubeValidator(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		path     string
		expected []ValidationResult
	}

	testCases := []testCase{
		{
			path: "testdata/error",
			expected: []ValidationResult{
				{
					Filename:  "testdata/error/deployment.yaml",
					Kind:      "",
					Name:      "",
					Namespace: "",
					Version:   "",
					Status:    "error",
					Message:   "error unmarshalling resource: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type map[string]interface {}",
				},
			},
		},
		{
			path: "testdata/invalid",
			expected: []ValidationResult{
				{
					Filename:  "testdata/invalid/deployment.yaml",
					Kind:      "Deployment",
					Name:      "nginx-deployment",
					Namespace: "test1",
					Version:   "apps/v1",
					Status:    "valid",
					Message:   "",
				},
				{
					Filename:  "testdata/invalid/deployment.yaml",
					Kind:      "Deployment",
					Name:      "nginx-deployment2",
					Namespace: "test1",
					Version:   "apps/v1",
					Status:    "invalid",
					Message:   "For field spec: Additional property notAValidKey is not allowed",
				},
				{
					Filename:  "testdata/invalid/service.yaml",
					Kind:      "Service",
					Name:      "my-service",
					Namespace: "test1",
					Version:   "v1",
					Status:    "valid",
					Message:   "",
				},
			},
		},
		{
			path: "testdata/valid",
			expected: []ValidationResult{
				{
					Filename:  "testdata/valid/deployment.yaml",
					Kind:      "Deployment",
					Name:      "nginx-deployment",
					Namespace: "test1",
					Version:   "apps/v1",
					Status:    "valid",
					Message:   "",
				},
				{
					Filename:  "testdata/valid/deployment.yaml",
					Kind:      "Deployment",
					Name:      "nginx-deployment2",
					Namespace: "test1",
					Version:   "apps/v1",
					Status:    "valid",
					Message:   "",
				},
				{
					Filename:  "testdata/valid/service.yaml",
					Kind:      "Service",
					Name:      "my-service",
					Namespace: "test1",
					Version:   "v1",
					Status:    "valid",
					Message:   "",
				},
			},
		},
	}

	validator, err := NewKubeValidator()
	require.NoError(t, err)

	for _, testCase := range testCases {
		results, err := validator.RunValidation(ctx, testCase.path)

		// Zero out indices so we don't need to check them
		for i := 0; i < len(results); i++ {
			results[i].index = 0
		}

		require.NoError(t, err)
		assert.Equal(t, testCase.expected, results)
	}
}
