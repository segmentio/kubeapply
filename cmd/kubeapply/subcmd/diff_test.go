package subcmd

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/require"
)

func TestDiff(t *testing.T) {
	if !util.KindEnabled() {
		t.Skipf("Skipping because kind is not enabled")
	}

	ctx := context.Background()

	testClusterDir := "testdata/clusters/basic"

	diffFlagValues.kubeConfig = kubeConfigTestPath

	err := diffClusterPath(
		ctx,
		filepath.Join(testClusterDir, "cluster.yaml"),
	)
	require.Nil(t, err)
}
