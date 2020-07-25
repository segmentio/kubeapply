package util

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestoreData(t *testing.T) {
	type testCase struct {
		description   string
		url           string
		errExpected   bool
		destPath      string
		expectedFiles []string
	}

	tempDir, err := ioutil.TempDir("", "data")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	textInputsDir := filepath.Join(tempDir, "text_inputs")
	textOutputsDir := filepath.Join(tempDir, "text_outputs")

	WriteFiles(
		t,
		textInputsDir,
		map[string]string{
			"dir1/file1.txt": "file1 contents",
			"dir1/file2.txt": "file2 contents",
			"dir2/file3.txt": "file3 contents",
		},
	)

	tarInput := filepath.Join(tempDir, "tar_inputs.tar.gz")
	tarOutputDir := filepath.Join(tempDir, "tar_outputs")
	require.Nil(t, os.MkdirAll(tarOutputDir, 0755))

	cmd := exec.Command("tar", "-czvf", tarInput, "text_inputs")
	cmd.Dir = tempDir
	require.Nil(t, cmd.Run())

	testServer := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/test-file.tar.gz" {
					http.Error(w, "Not found", 404)
					return
				}

				input, err := os.Open(tarInput)
				defer input.Close()
				if err != nil {
					http.Error(w, "Can't open file", 500)
					return
				}
				info, err := input.Stat()
				if err != nil {
					http.Error(w, "Error stat'ing file", 500)
					return
				}

				w.Header().Set("Content-Disposition", "attachment; filename=tar_inputs.tar.gz")
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

				input.Seek(0, 0)
				io.Copy(w, input)
			},
		),
	)
	defer testServer.Close()

	httpOutputDir := filepath.Join(tempDir, "http_outputs")
	require.Nil(t, os.MkdirAll(httpOutputDir, 0755))

	testCases := []testCase{
		{
			description: "basic files",
			url:         textInputsDir,
			destPath:    textOutputsDir,
			expectedFiles: []string{
				"dir1/file1.txt",
				"dir1/file2.txt",
				"dir2/file3.txt",
			},
		},
		{
			description: "tar archive",
			url:         fmt.Sprintf("file://%s", tarInput),
			destPath:    tarOutputDir,
			expectedFiles: []string{
				"text_inputs/dir1/file1.txt",
				"text_inputs/dir1/file2.txt",
				"text_inputs/dir2/file3.txt",
			},
		},
		{
			description: "http archive",
			url:         fmt.Sprintf("%s/test-file.tar.gz", testServer.URL),
			destPath:    httpOutputDir,
			expectedFiles: []string{
				"text_inputs/dir1/file1.txt",
				"text_inputs/dir1/file2.txt",
				"text_inputs/dir2/file3.txt",
			},
		},
	}

	ctx := context.Background()

	for _, testCase := range testCases {
		err := RestoreData(ctx, ".", testCase.url, testCase.destPath)
		if testCase.errExpected {
			assert.NotNil(t, err, testCase.description)
		} else {
			require.Nil(t, err, testCase.description)
			require.Equal(
				t,
				testCase.expectedFiles,
				allSubpaths(t, testCase.destPath),
			)
		}
	}
}

func allSubpaths(t *testing.T, root string) []string {
	paths := []string{}

	err := filepath.Walk(
		root,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			paths = append(paths, relPath)
			return nil
		},
	)
	require.Nil(t, err)

	sort.Slice(paths, func(a, b int) bool {
		return paths[a] < paths[b]
	})

	return paths
}
