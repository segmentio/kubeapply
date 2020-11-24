package helm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const globals = `
---
global:
  cluster: "test-cluster"
  region: "us-west-2"
  shortRegion: "usw2"
`

func TestExpandHelmTemplates(t *testing.T) {
	type testCase struct {
		description     string
		configPath      string
		expError        bool
		expContents     map[string][]string
		expDoesNotExist []string
	}

	ctx := context.Background()

	tempDir, err := ioutil.TempDir("", "helm")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	globalsPath := filepath.Join(tempDir, "globals.yaml")
	err = ioutil.WriteFile(
		globalsPath,
		[]byte(globals),
		0644,
	)
	require.Nil(t, err)

	testCases := []testCase{
		{
			description: "normal chart",
			configPath:  "testdata/configs",
			expDoesNotExist: []string{
				"kube-system/alb-ingress-controller.helm.yaml",
			},
			expContents: map[string][]string{
				"kube-system/alb-ingress-controller/templates/alb-ingress-controller.yaml": {
					"name: alb-ingress-controller-RELEASE-NAME",
					"namespace: kube-system",
					"image: test-image1",
					"helm/test: normal",
				},
			},
		},
		{
			description: "chart override",
			configPath:  "testdata/configs-charts-override",
			expDoesNotExist: []string{
				"kube-system/alb-ingress-controller.helm.yaml",
			},
			expContents: map[string][]string{
				"kube-system/name-override/alb-ingress-controller/templates/alb-ingress-controller.yaml": {
					"name: alb-ingress-controller-name-override",
					"namespace: kube-system",
					"image: test-image2",
					"helm/test: override",
				},
			},
		},
		{
			description: "chart multi",
			configPath:  "testdata/configs-charts-multi",
			expDoesNotExist: []string{
				"kube-system/alb-ingress-controller.helm.yaml",
			},
			expContents: map[string][]string{
				"kube-system/namespace1/alb-ingress-controller/templates/alb-ingress-controller.yaml": {
					"name: alb-ingress-controller-namespace1",
					"namespace: namespace1",
					"image: test-image1",
					"helm/test: normal",
				},
				"kube-system/namespace2/alb-ingress-controller/templates/alb-ingress-controller.yaml": {
					"name: alb-ingress-controller-namespace2",
					"namespace: namespace2",
					"image: test-image2",
					"helm/test: normal",
				},
			},
		},
		{
			description: "chart disabled",
			configPath:  "testdata/configs-disabled",
			expDoesNotExist: []string{
				"kube-system/alb-ingress-controller.helm.yaml",
				"kube-system/alb-ingress-controller/templates/alb-ingress-controller.yaml",
			},
		},
		{
			description: "helm error",
			configPath:  "testdata/configs-bad",
			expError:    true,
		},
	}

	client := &HelmClient{
		GlobalValuesPath: globalsPath,
		Parallelism:      2,
	}

	for index, testCase := range testCases {
		testCaseDir := filepath.Join(tempDir, fmt.Sprintf("%d", index))
		err = os.MkdirAll(testCaseDir, 0755)
		require.Nil(t, err, testCase.description)

		err := util.RecursiveCopy(
			testCase.configPath,
			testCaseDir,
		)
		require.Nil(t, err, testCase.description)

		err = client.ExpandHelmTemplates(
			ctx,
			testCaseDir,
			"testdata/charts",
		)

		if testCase.expError {
			require.NotNil(t, err, testCase.description)
		} else {
			require.Nil(t, err, testCase.description)

			for _, doesNotExistSubPath := range testCase.expDoesNotExist {
				fullPath := filepath.Join(testCaseDir, doesNotExistSubPath)
				exists, err := util.FileExists(fullPath)

				require.Nil(t, err, testCase.description)
				assert.False(
					t,
					exists,
					"file %s for test case %s exists",
					doesNotExistSubPath,
					testCase.description,
				)
			}

			for existsSubPath, values := range testCase.expContents {
				fullPath := filepath.Join(testCaseDir, existsSubPath)
				exists, err := util.FileExists(fullPath)

				require.Nil(t, err, testCase.description)
				require.True(t, exists, testCase.description)

				contents, err := ioutil.ReadFile(fullPath)
				require.Nil(t, err, testCase.description)
				contentsStr := string(contents)

				for _, value := range values {
					assert.True(
						t,
						strings.Contains(contentsStr, value),
						"file %s for test case %s does not contain %s: %s",
						existsSubPath,
						testCase.description,
						value,
						contentsStr,
					)
				}
			}
		}
	}
}

func TestGetHeaderComments(t *testing.T) {
	type testCase struct {
		contentsLines   []string
		expectedHeaders map[string]string
	}

	testCases := []testCase{
		{
			contentsLines: []string{
				"",
				"# some comment without a colon",
				"something",
			},
			expectedHeaders: map[string]string{},
		},
		{
			contentsLines: []string{
				"",
				"# disabled: false",
				"# another comment",
				"",
				"something",
				"# charts: 'charts'",
			},
			expectedHeaders: map[string]string{
				"disabled": "false",
			},
		},
		{
			contentsLines: []string{
				"",
				"#charts: \"charts-url\"",
				"# disabled: true",
				"",
				"something",
				"# charts: 'charts'",
			},
			expectedHeaders: map[string]string{
				"charts":   "charts-url",
				"disabled": "true",
			},
		},
	}

	for _, testCase := range testCases {
		contentsStr := strings.Join(testCase.contentsLines, "\n")

		headers := getHeaderComments([]byte(contentsStr))
		assert.Equal(t, testCase.expectedHeaders, headers)
	}
}
