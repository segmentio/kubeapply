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

type TestStruct struct {
	Key    string
	Inner  TestStructInner
	Inner2 *TestStructInner
}
type TestStructInner struct {
	Map map[string]interface{}
}

func TestLookup(t *testing.T) {
	s := TestStruct{
		Key: "value0",
		Inner: TestStructInner{
			Map: map[string]interface{}{
				"key1": "value1",
				"key2": map[string]interface{}{
					"key3": map[string]interface{}{
						"key4": "value4",
					},
					"key5": 1234,
				},
			},
		},
		Inner2: &TestStructInner{
			Map: map[string]interface{}{
				"key6": "value6",
			},
		},
	}
	assert.Equal(t, "value0", lookup("Key", s))
	assert.Equal(t, nil, lookup("bad-key", s))
	assert.Equal(t, nil, lookup("", s))
	assert.Equal(t, "value1", lookup("Inner.Map.key1", s))
	assert.Equal(t, "value1", lookup("Inner.Map.key1", &s))
	assert.Equal(t, "value4", lookup("Inner.Map.key2.key3.key4", s))
	assert.Equal(t, 1234, lookup("Inner.Map.key2.key5", s))
	assert.Equal(t, nil, lookup("Inner.Map.non-existent-key", s))
	assert.Equal(t, nil, lookup("Inner.Map.key2.non-existent-key", s))
	assert.Equal(t, "value6", lookup("Inner2.Map.key6", s))
}
