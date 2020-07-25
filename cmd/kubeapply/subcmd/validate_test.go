package subcmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	testClusterDir := "testdata/clusters/expand-test"

	err := validateRun(nil, []string{filepath.Join(testClusterDir, "cluster2.yaml")})
	require.Nil(t, err)
}
