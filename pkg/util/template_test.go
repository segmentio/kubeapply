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
