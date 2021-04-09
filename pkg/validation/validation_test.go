package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FakeChecker struct{}

func (f *FakeChecker) Check(_ context.Context, resource Resource) CheckResult {
	return CheckResult{
		CheckType: CheckTypeKubeconform,
		Status:    StatusValid,
		Message:   resource.PrettyName(),
	}
}

func TestKubeValidator(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		path     string
		expected []Result
	}

	testCases := []testCase{
		{
			path: "testdata/configs",
			expected: []Result{
				{
					Resource: Resource{
						Path:      "testdata/configs/deployment.yaml",
						Kind:      "Deployment",
						Name:      "nginx-deployment",
						Namespace: "test1",
						Version:   "apps/v1",
					},
					CheckResults: []CheckResult{
						{
							CheckType: CheckTypeKubeconform,
							Status:    StatusValid,
							Message:   "Deployment.apps/v1.test1.nginx-deployment",
						},
					},
				},
				{
					Resource: Resource{
						Path:      "testdata/configs/deployment.yaml",
						Kind:      "Deployment",
						Name:      "nginx-deployment2",
						Namespace: "test1",
						Version:   "apps/v1",
					},
					CheckResults: []CheckResult{
						{
							CheckType: CheckTypeKubeconform,
							Status:    StatusValid,
							Message:   "Deployment.apps/v1.test1.nginx-deployment2",
						},
					},
				},
				{
					Resource: Resource{
						Path:      "testdata/configs/service.yaml",
						Kind:      "Service",
						Name:      "my-service",
						Namespace: "test1",
						Version:   "v1",
					},
					CheckResults: []CheckResult{
						{
							CheckType: CheckTypeKubeconform,
							Status:    StatusValid,
							Message:   "Service.v1.test1.my-service",
						},
					},
				},
			},
		},
	}

	validator := NewKubeValidator(
		KubeValidatorConfig{
			NumWorkers: 1,
			Checkers: []Checker{
				&FakeChecker{},
			},
		},
	)

	for _, testCase := range testCases {
		results, err := validator.RunChecks(ctx, testCase.path)

		// Zero out contents and indices so we don't need to check them
		for i := 0; i < len(results); i++ {
			results[i].Resource.Contents = nil
			results[i].Resource.index = 0
		}

		require.NoError(t, err)
		assert.Equal(t, testCase.expected, results)
	}
}
