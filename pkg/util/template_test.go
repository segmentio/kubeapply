package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTemplate(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "templates")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	err = RecursiveCopy("testdata/templates", tempDir)
	require.Nil(t, err)

	err = ApplyTemplate(
		tempDir,
		map[string]string{
			"value1": "value1Data",
			"value2": "value2Data",
		},
		true,
		false,
	)
	require.Nil(t, err)

	allFiles := getAllFiles(t, tempDir)

	assert.Equal(
		t,
		[]string{
			"configs/test.json",
			"configs2/test.json",
			"configs2/test.yaml",
			"test.yaml",
			"test2.yaml",
		},
		allFiles,
	)

	assert.Equal(
		t,
		strings.TrimSpace(`
key1: value1Data
key2: value2Data
contents:
    {
        "key1": "value1"
    }

configMap:
  test.json: |
    {
        "key2": "value2"
    }
  test.yaml: |
    key1: value1

    key2: value2Data

configMap2:
  test.json: |
    {
        "key1": "value1"
    }
`),
		strings.TrimSpace(
			fileContents(t, filepath.Join(tempDir, "test.yaml")),
		),
	)

	assert.Equal(
		t,
		"key1: {{.value1}}",
		strings.TrimSpace(
			fileContents(t, filepath.Join(tempDir, "test2.yaml")),
		),
	)
}

func TestApplyTemplateStrict(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "templates")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	err = RecursiveCopy("testdata/templates", tempDir)
	require.Nil(t, err)

	err = ApplyTemplate(
		tempDir,
		map[string]string{
			"value1": "value1Data",
		},
		true,
		true,
	)
	require.Error(t, err)
}

func getAllFiles(t *testing.T, path string) []string {
	allFiles := []string{}

	err := filepath.Walk(
		path,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(path, subPath)
			if err != nil {
				return err
			}

			allFiles = append(allFiles, relPath)
			return nil
		},
	)
	require.Nil(t, err)

	return allFiles
}

func fileContents(t *testing.T, path string) string {
	contents, err := ioutil.ReadFile(path)
	require.Nil(t, err)

	return string(contents)
}

func TestLookup(t *testing.T) {
	m := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"key3": map[string]interface{}{
				"key4": "value4",
			},
			"key5": 1234,
		},
		"key6": nil,
	}

	type testCase struct {
		input          interface{}
		path           string
		expectedResult interface{}
		expectErr      bool
	}

	testCases := []testCase{
		{
			input:          m,
			path:           "bad-key",
			expectedResult: nil,
		},
		{
			input:          m,
			path:           "",
			expectedResult: nil,
		},
		{
			input:          nil,
			path:           "key1",
			expectedResult: nil,
		},
		{
			input:          "not a map",
			path:           "key1",
			expectedResult: nil,
			expectErr:      true,
		},
		{
			input:          m,
			path:           "key1",
			expectedResult: "value1",
		},
		{
			input:          &m,
			path:           "key1",
			expectedResult: "value1",
		},
		{
			input:          m,
			path:           "key1.not-a-map",
			expectedResult: nil,
			expectErr:      true,
		},
		{
			input:          m,
			path:           "key2.key3.key4",
			expectedResult: "value4",
		},
		{
			input:          m,
			path:           "key2.key5",
			expectedResult: 1234,
		},
		{
			input:          m,
			path:           "key6.nil-key",
			expectedResult: nil,
		},
	}

	for index, tc := range testCases {
		result, err := lookup(tc.input, tc.path)
		assert.Equal(t, tc.expectedResult, result, "Unexpected result for case %d", index)
		if tc.expectErr {
			assert.Error(t, err, "Did not get expected error in case %d", index)
		} else {
			assert.NoError(t, err, "Got unexpected error in case %d", index)
		}
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		desc        string
		values      []interface{}
		expect      interface{}
		expectError string
	}{
		{
			desc:   "empty",
			values: []interface{}{},
			expect: map[string]interface{}{},
		},
		{
			desc:   "nil",
			values: []interface{}{nil, nil},
			expect: map[string]interface{}{},
		},
		{
			desc:   "single",
			values: []interface{}{map[string]interface{}{"a": 1}},
			expect: map[string]interface{}{"a": 1},
		},
		{
			desc:        "invalid type",
			values:      []interface{}{1},
			expectError: "Argument 0: Expected map[string]interface{} or nil, got int",
		},
		{
			desc: "simple",
			values: []interface{}{
				map[string]interface{}{"a": 1, "c": 3},
				map[string]interface{}{"b": 2},
			},
			expect: map[string]interface{}{
				"a": 1,
				"b": 2,
				"c": 3,
			},
		},
		{
			desc: "overwrite",
			values: []interface{}{
				map[string]interface{}{"a": 1, "c": 3},
				map[string]interface{}{"a": 4, "b": 2},
			},
			expect: map[string]interface{}{
				"a": 4,
				"b": 2,
				"c": 3,
			},
		},
		{
			desc: "nested",
			values: []interface{}{
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{
							"c": 1,
						},
						"d": 2,
					},
				},
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{
							"c": 3,
						},
					},
					"e": 4,
				},
			},
			expect: map[string]interface{}{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": 3,
					},
					"d": 2,
				},
				"e": 4,
			},
		},
		{
			desc: "nested replace",
			values: []interface{}{
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{
							"c": 1,
						},
						"d": 2,
					},
				},
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": 1,
					},
				},
			},
			expect: map[string]interface{}{
				"a": map[string]interface{}{
					"b": 1,
					"d": 2,
				},
			},
		},
		{
			desc: "type error",
			values: []interface{}{
				map[string]interface{}{
					"a": 1,
				},
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": 2,
					},
				},
			},
			expectError: "a: Expected map[string]interface{}, got int",
		},
		{
			desc: "nested type error",
			values: []interface{}{
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": 1,
					},
				},
				map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{
							"c": 2,
						},
					},
				},
			},
			expectError: "a.b: Expected map[string]interface{}, got int",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, gotErr := merge(test.values...)
			if test.expectError != "" {
				require.Error(t, gotErr)
				assert.Equal(t, test.expectError, gotErr.Error())
			} else {
				require.NoError(t, gotErr)
				assert.Equal(t, test.expect, got)
			}
		})
	}
}
