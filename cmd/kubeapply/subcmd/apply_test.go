package subcmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const kubeConfigTestPath = "../../../.kube/kind-kubeapply-test.yaml"

func TestApply(t *testing.T) {
	if !util.KindEnabled() {
		t.Skipf("Skipping because kind is not enabled")
	}

	ctx := context.Background()

	namespace := fmt.Sprintf("test-apply-%d", time.Now().UnixNano()/1000)
	util.CreateNamespace(t, ctx, namespace, kubeConfigTestPath)

	tempDir, err := ioutil.TempDir("", "apply")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	clusterDir := filepath.Join(tempDir, "cluster")
	require.Nil(t, err)

	err = util.RecursiveCopy("testdata/clusters/apply-test", clusterDir)
	require.Nil(t, err)

	replaceNamespace(t, filepath.Join(clusterDir, "expanded"), namespace)

	applyFlagValues.noCheck = true
	applyFlagValues.yes = true
	applyFlagValues.kubeConfig = kubeConfigTestPath

	err = applyClusterPath(
		ctx,
		filepath.Join(clusterDir, "cluster.yaml"),
	)
	require.Nil(t, err)

	deployments := util.GetResources(t, ctx, "deployments", namespace, kubeConfigTestPath)
	assert.Equal(t, 1, len(deployments))

	services := util.GetResources(t, ctx, "services", namespace, kubeConfigTestPath)
	assert.Equal(t, 1, len(services))
}

func replaceNamespace(t *testing.T, root string, namespace string) {
	err := filepath.Walk(
		root,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(subPath, ".yaml") {
				return nil
			}

			contents, err := ioutil.ReadFile(subPath)
			if err != nil {
				return err
			}
			updatedContents := bytes.ReplaceAll(
				contents,
				[]byte("test-namespace"),
				[]byte(namespace),
			)
			return ioutil.WriteFile(subPath, updatedContents, 0644)
		},
	)
	require.Nil(t, err)
}
