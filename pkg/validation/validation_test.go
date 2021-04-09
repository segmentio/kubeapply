package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FakeChecker struct {
}

func TestKubeValidator(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		path     string
		expected []Result
	}

	testCases := []testCase{
		{
			path: "testdata/error",
			expected: []Result{
				{
					Filename:      "testdata/error/deployment.yaml",
					Kind:          "",
					Name:          "",
					Namespace:     "",
					Version:       "",
					SchemaStatus:  "error",
					SchemaMessage: "error unmarshalling resource: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type map[string]interface {}",
				},
			},
		},
		{
			path: "testdata/invalid",
			expected: []Result{
				{
					Filename:      "testdata/invalid/deployment.yaml",
					Kind:          "Deployment",
					Name:          "nginx-deployment",
					Namespace:     "test1",
					Version:       "apps/v1",
					SchemaStatus:  "valid",
					SchemaMessage: "",
				},
				{
					Filename:      "testdata/invalid/deployment.yaml",
					Kind:          "Deployment",
					Name:          "nginx-deployment2",
					Namespace:     "test1",
					Version:       "apps/v1",
					SchemaStatus:  "invalid",
					SchemaMessage: "For field spec: Additional property notAValidKey is not allowed",
				},
				{
					Filename:      "testdata/invalid/service.yaml",
					Kind:          "Service",
					Name:          "my-service",
					Namespace:     "test1",
					Version:       "v1",
					SchemaStatus:  "valid",
					SchemaMessage: "",
				},
			},
		},
		{
			path: "testdata/valid",
			expected: []Result{
				{
					Filename:      "testdata/valid/deployment.yaml",
					Kind:          "Deployment",
					Name:          "nginx-deployment",
					Namespace:     "test1",
					Version:       "apps/v1",
					SchemaStatus:  "valid",
					SchemaMessage: "",
				},
				{
					Filename:      "testdata/valid/deployment.yaml",
					Kind:          "Deployment",
					Name:          "nginx-deployment2",
					Namespace:     "test1",
					Version:       "apps/v1",
					SchemaStatus:  "valid",
					SchemaMessage: "",
				},
				{
					Filename:      "testdata/valid/service.yaml",
					Kind:          "Service",
					Name:          "my-service",
					Namespace:     "test1",
					Version:       "v1",
					SchemaStatus:  "valid",
					SchemaMessage: "",
				},
			},
		},
	}

	validator := NewKubeValidator(KubeValidatorConfig{NumWorkers: 1})

	for _, testCase := range testCases {
		results, err := validator.RunChecks(ctx, testCase.path)

		// Zero out indices so we don't need to check them
		for i := 0; i < len(results); i++ {
			results[i].Resource.index = 0
		}

		require.NoError(t, err)
		assert.Equal(t, testCase.expected, results)
	}
}
