package subcmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestDiff(t *testing.T) {
	if !util.KindEnabled() {
		t.Skipf("Skipping because kind is not enabled")
	}

	ctx := context.Background()

	namespace := fmt.Sprintf("test-diff-%d", time.Now().UnixNano()/1000)
	util.CreateNamespace(ctx, t, namespace, kubeConfigTestPath)

	tempDir, err := ioutil.TempDir("", "diff")
	require.Nil(t, err)
	defer os.RemoveAll(tempDir)

	clusterDir := filepath.Join(tempDir, "cluster")
	require.Nil(t, err)

	err = util.RecursiveCopy("testdata/clusters/apply-test", clusterDir)
	require.Nil(t, err)
	replaceNamespace(t, filepath.Join(clusterDir, "expanded"), namespace)

	// Full outputs
	diffFlagValues.kubeConfig = kubeConfigTestPath
	err = diffClusterPath(
		ctx,
		filepath.Join(clusterDir, "cluster.yaml"),
	)
	require.Nil(t, err)

	// Simple outputs
	diffFlagValues.simpleOutput = true
	err = diffClusterPath(
		ctx,
		filepath.Join(clusterDir, "cluster.yaml"),
	)
	require.Nil(t, err)
}
