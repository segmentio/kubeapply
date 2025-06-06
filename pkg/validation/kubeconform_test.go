package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validKubeconformDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: test1
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80`

	invalidKubeconformDeployment = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: test1
  labels:
    app: nginx
spec:
  replicas: 3
  invalidField: invalid
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80`
)

func TestKubeconformChecker(t *testing.T) {
	ctx := context.Background()

	type testCase struct {
		resource Resource
		expected CheckResult
	}

	testCases := []testCase{
		{
			resource: MakeResource(
				"test/path",
				[]byte(validKubeconformDeployment),
				0,
			),
			expected: CheckResult{
				CheckType: CheckTypeKubeconform,
				CheckName: "kubeconform",
				Status:    StatusValid,
			},
		},
		{
			resource: MakeResource(
				"test/path",
				[]byte(invalidKubeconformDeployment),
				0,
			),
			expected: CheckResult{
				CheckType: CheckTypeKubeconform,
				CheckName: "kubeconform",
				Status:    StatusInvalid,
				Message:   "problem validating schema. Check JSON formatting: jsonschema: '/spec' does not validate with https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master/master-standalone-strict/deployment-apps-v1.json#/properties/spec/additionalProperties: additionalProperties 'invalidField' not allowed",
			},
		},
		{
			resource: MakeResource(
				"test/path",
				[]byte("xxx\nzzzz"),
				0,
			),
			expected: CheckResult{
				CheckType: CheckTypeKubeconform,
				CheckName: "kubeconform",
				Status:    StatusError,
				Message:   "error unmarshalling resource: error unmarshaling JSON: while decoding JSON: json: cannot unmarshal string into Go value of type map[string]interface {}",
			},
		},
	}

	checker, err := NewKubeconformChecker()
	require.NoError(t, err)

	for _, testCase := range testCases {
		result := checker.Check(ctx, testCase.resource)
		assert.Equal(t, testCase.expected, result)
	}
}
