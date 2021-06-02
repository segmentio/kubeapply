package util

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kruntime "k8s.io/apimachinery/pkg/runtime"
)

// CompareJSONObjs compares two objects via their JSON representations. This
// is much easier to debug that comparing the objects directly.
func CompareJSONObjs(t *testing.T, exp kruntime.Object, actual kruntime.Object) {
	expBytes, err := json.Marshal(exp)
	if err != nil {
		assert.FailNow(t, "Error marshalling expected object to JSON", err)
	}

	actualBytes, err := json.Marshal(actual)
	if err != nil {
		assert.FailNow(t, "Error marshalling actual object to JSON", err)
	}

	assert.JSONEq(t, string(expBytes), string(actualBytes))
}

// WriteFiles takes a map of paths to file contents and uses this to write out files to
// the file system.
func WriteFiles(t *testing.T, baseDir string, files map[string]string) {
	for path, contents := range files {
		fullPath := filepath.Join(baseDir, path)
		fullPathDir := filepath.Dir(fullPath)

		ok, err := DirExists(fullPathDir)
		if err != nil {
			assert.FailNow(t, "Error checking dir: %+v", err)
		}

		if !ok {
			err = os.MkdirAll(fullPathDir, 0755)
			if err != nil {
				assert.FailNow(t, "Error creating dir: %+v", err)
			}
		}

		err = ioutil.WriteFile(fullPath, []byte(contents), 0644)
		if err != nil {
			assert.FailNow(t, "Error creating file: %+v", err)
		}
	}
}

// GetContents returns the contents of a directory as a map from
// file name to string content lines.
func GetContents(t *testing.T, root string) map[string][]string {
	contentsMap := map[string][]string{}

	// Process as a directory
	err := filepath.Walk(
		root,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			contents, err := ioutil.ReadFile(subPath)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(root, subPath)
			if err != nil {
				return err
			}

			lines := []string{}
			for _, line := range bytes.Split(contents, []byte("\n")) {
				lines = append(lines, string(line))
			}

			contentsMap[relPath] = lines
			return nil
		},
	)
	require.Nil(t, err)

	return contentsMap
}

// GetFileContents gets the string contents of a single file.
func GetFileContents(t *testing.T, path string) string {
	contents, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	return string(contents)
}

// KindEnabled returns whether testing with kind is enabled. This is generally true locally
// but false in CI (for now).
func KindEnabled() bool {
	return strings.ToLower(os.Getenv("KIND_ENABLED")) == "true"
}

// CreateNamespace creates a namespace in a test cluster.
func CreateNamespace(
	ctx context.Context,
	t *testing.T,
	namespace string,
	kubeconfig string,
) {
	cmd := exec.CommandContext(
		ctx,
		"kubectl",
		"create",
		"namespace",
		namespace,
		"--kubeconfig",
		kubeconfig,
	)
	result, err := cmd.CombinedOutput()
	require.Nil(t, err, "Error running kubectl: %+v", string(result))
}

// DeleteNamespace deletes a namespace in a test cluster.
func DeleteNamespace(
	ctx context.Context,
	t *testing.T,
	namespace string,
	kubeconfig string,
) {
	cmd := exec.CommandContext(
		ctx,
		"kubectl",
		"delete",
		"namespace",
		namespace,
		"--kubeconfig",
		kubeconfig,
	)
	result, err := cmd.CombinedOutput()
	require.Nil(t, err, "Error running kubectl: %+v", string(result))
	require.Nil(t, err)
}

// GetResources gets the objects with the given kind in the argument namespace.
func GetResources(
	ctx context.Context,
	t *testing.T,
	kind string,
	namespace string,
	kubeconfig string,
) []apply.TypedKubeObj {
	cmd := exec.CommandContext(
		ctx,
		"kubectl",
		"get",
		kind,
		"-n",
		namespace,
		"--kubeconfig",
		kubeconfig,
		"-o",
		"json",
	)
	result, err := cmd.CombinedOutput()
	require.Nil(t, err, "Error getting resources: %+v", string(result))
	objs, err := apply.KubeJSONToObjects(result)
	require.Nil(t, err)

	return objs
}
