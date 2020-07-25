package subcmd

import (
	"crypto/sha1"
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

var regenerateStr = os.Getenv("REGENERATE_TESTDATA")

func TestExpand(t *testing.T) {
	testClusterDir := "testdata/clusters/expand-test"

	require.Nil(t, os.RemoveAll(filepath.Join(testClusterDir, "expanded")))

	err := expandRun(nil, []string{filepath.Join(testClusterDir, "cluster1.yaml")})
	require.Nil(t, err)

	if strings.ToLower(regenerateStr) == "true" {
		require.Nil(t, os.RemoveAll(filepath.Join(testClusterDir, "expanded_expected")))
		require.Nil(
			t,
			util.RecursiveCopy(
				filepath.Join(testClusterDir, "expanded"),
				filepath.Join(testClusterDir, "expanded_expected"),
			),
		)
	} else {
		assert.Equal(
			t,
			getContents(t, filepath.Join(testClusterDir, "expanded_expected")),
			getContents(t, filepath.Join(testClusterDir, "expanded")),
		)
	}
}

func getContents(t *testing.T, root string) map[string]string {
	contentsMap := map[string]string{}

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

			hash := sha1.New()
			_, err = hash.Write(contents)
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(root, subPath)
			if err != nil {
				return err
			}

			contentsMap[relPath] = fmt.Sprintf("%x", hash.Sum(nil))
			return nil
		},
	)
	require.Nil(t, err)

	return contentsMap
}
