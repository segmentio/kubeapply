package provider

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceGet(t *testing.T) {
	ctx := context.Background()

	gitClientObj := &fakeGitClient{}

	sourceFetcherObj, err := newSourceFetcher(gitClientObj)
	require.NoError(t, err)
	defer sourceFetcherObj.cleanup()

	tempDir, err := ioutil.TempDir("", "sources")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = sourceFetcherObj.get(
		ctx,
		"git@github.com:segmentio/terracode-modules//profiles?ref=2021-03-18",
		filepath.Join(tempDir, "dest1"),
	)
	require.NoError(t, err)

	err = sourceFetcherObj.get(
		ctx,
		"git@github.com:segmentio/terracode-modules//profiles?ref=2021-03-18",
		filepath.Join(tempDir, "dest2"),
	)
	require.NoError(t, err)

	err = sourceFetcherObj.get(
		ctx,
		"git@github.com:segmentio/terracode-modules//profiles?ref=2021-05-01",
		filepath.Join(tempDir, "dest3"),
	)
	require.NoError(t, err)

	contents := util.GetContents(t, tempDir)
	assert.Equal(
		t,
		map[string][]string{
			"dest1/profile.txt": {
				"repoURL=git@github.com:segmentio/terracode-modules ref=2021-03-18",
			},
			"dest2/profile.txt": {
				"repoURL=git@github.com:segmentio/terracode-modules ref=2021-03-18",
			},
			"dest3/profile.txt": {
				"repoURL=git@github.com:segmentio/terracode-modules ref=2021-05-01",
			},
		},
		contents,
	)

	assert.Equal(
		t,
		[]string{
			"cloneRepo repoURL=git@github.com:segmentio/terracode-modules,ref=2021-03-18",
			"cloneRepo repoURL=git@github.com:segmentio/terracode-modules,ref=2021-05-01",
		},
		gitClientObj.calls,
	)
}
