package pullreq

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var regenerateStr = os.Getenv("REGENERATE_TESTDATA")

func TestApplyComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.NoError(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpaths = []string{"test/subpath"}

	pullRequestClient := &FakePullRequestClient{
		ClusterConfigs: clusterConfigs,
		ApprovedVal:    true,
		Mergeable:      true,
		Merged:         false,
	}

	applies := []ClusterApply{
		{
			ClusterConfig: clusterConfigs[0],
			Results: []apply.Result{
				{
					Name:       "test-name",
					Namespace:  "test-namespace",
					Kind:       "test-kind",
					CreatedAt:  time.Unix(12345, 0),
					OldVersion: "1234",
					NewVersion: "3456",
				},
				{
					Name:       "test-name2",
					Namespace:  "test-namespace",
					Kind:       "test-kind",
					OldVersion: "1234",
					CreatedAt:  time.Unix(56778, 0),
					NewVersion: "1234",
				},
			},
		},
		{
			ClusterConfig: clusterConfigs[1],
			Results: []apply.Result{
				{
					Name:       "test-name3",
					Namespace:  "test-namespace3",
					Kind:       "test-kind3",
					CreatedAt:  time.Unix(12345, 0),
					OldVersion: "1234",
					NewVersion: "3456",
				},
				{
					Name:       "test-name4",
					Namespace:  "test-namespace5",
					Kind:       "test-kind",
					OldVersion: "1234",
					CreatedAt:  time.Unix(56778, 0),
					NewVersion: "1234",
				},
			},
		},
		{
			ClusterConfig: clusterConfigs[2],
			Results: []apply.Result{
				{
					Name:       "test-name3",
					Namespace:  "test-namespace3",
					Kind:       "test-kind3",
					CreatedAt:  time.Unix(12345, 0),
					OldVersion: "1234",
					NewVersion: "1234",
				},
			},
		},
	}

	commentData := ApplyCommentData{
		ClusterApplies:    applies,
		PullRequestClient: pullRequestClient,
		Env:               "stage",
	}

	result, err := FormatApplyComment(commentData)
	require.NoError(t, err)

	expectedOutput := "testdata/comments/apply.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.NoError(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.NoError(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestDiffComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.NoError(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpaths = []string{"test/subpath"}

	pullRequestClient := &FakePullRequestClient{
		ClusterConfigs: clusterConfigs,
		ApprovedVal:    true,
		Mergeable:      true,
		Merged:         false,
	}

	diffs := []ClusterDiff{
		{
			ClusterConfig: clusterConfigs[0],
			Results: []diff.Result{
				{
					Name:    "test1",
					RawDiff: "line1\nline2\nline3",
					Object: &apply.TypedKubeObj{
						Kind: "kind1",
						KubeMetadata: apply.KubeMetadata{
							Name:      "name1",
							Namespace: "namespace1",
						},
					},
					NumAdded:   1,
					NumRemoved: 2,
				},
				{
					Name:    "test2",
					RawDiff: "line1\nline2",
					Object: &apply.TypedKubeObj{
						Kind: "kind2",
						KubeMetadata: apply.KubeMetadata{
							Name:      "name2",
							Namespace: "namespace2",
						},
					},
					NumAdded:   1,
					NumRemoved: 2,
				},
				{
					Name:     "test3",
					RawDiff:  "line1\nline2",
					NumAdded: 10,
				},
			},
		},
		{
			ClusterConfig: clusterConfigs[1],
		},
	}

	commentData := DiffCommentData{
		ClusterDiffs:      diffs,
		PullRequestClient: pullRequestClient,
		Env:               "stage",
	}

	result, err := FormatDiffComment(commentData)
	require.NoError(t, err)

	expectedOutput := "testdata/comments/diffs.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.NoError(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.NoError(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestDiffCommentBehind(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.NoError(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpaths = []string{"test/subpath"}

	pullRequestClient := &FakePullRequestClient{
		ClusterConfigs: clusterConfigs,
		BehindByVal:    3,
		ApprovedVal:    true,
		Mergeable:      true,
		Merged:         false,
	}

	diffs := []ClusterDiff{
		{
			ClusterConfig: clusterConfigs[0],
			Results: []diff.Result{
				{
					Name:       "test",
					RawDiff:    "raw diff",
					NumAdded:   1,
					NumRemoved: 2,
				},
			},
		},
	}

	commentData := DiffCommentData{
		ClusterDiffs:      diffs,
		PullRequestClient: pullRequestClient,
		Env:               "stage",
	}

	result, err := FormatDiffComment(commentData)
	require.NoError(t, err)

	expectedOutput := "testdata/comments/diffs-behind.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.NoError(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.NoError(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestErrorComment(t *testing.T) {
	commentData := ErrorCommentData{
		Error: fmt.Errorf("This is an error!"),
		Env:   "stage",
	}

	result, err := FormatErrorComment(commentData)
	require.NoError(t, err)

	expectedOutput := "testdata/comments/error.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.NoError(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.NoError(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestHelpComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.NoError(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpaths = []string{"test/subpath"}

	commentData := HelpCommentData{
		ClusterConfigs: clusterConfigs,
		Env:            "stage",
	}

	result, err := FormatHelpComment(commentData)
	require.NoError(t, err)

	expectedOutput := "testdata/comments/help.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.NoError(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.NoError(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestStatusComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.NoError(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpaths = []string{"test/subpath"}

	pullRequestClient := &FakePullRequestClient{
		ClusterConfigs: clusterConfigs,
	}

	statuses := []ClusterStatus{
		{
			ClusterConfig: clusterConfigs[0],
			HealthSummary: "test-health-summary1",
		},
		{
			ClusterConfig: clusterConfigs[1],
			HealthSummary: "test-health-summary2",
		},
	}

	commentData := StatusCommentData{
		ClusterStatuses:   statuses,
		PullRequestClient: pullRequestClient,
		Env:               "stage",
	}

	result, err := FormatStatusComment(commentData)
	require.NoError(t, err)

	expectedOutput := "testdata/comments/statuses.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.NoError(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.NoError(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestCommentChunks(t *testing.T) {
	body := "0123456789abcdefghijABC\nDEFGHIJ\nKLMNO"

	assert.Equal(
		t,
		[]string{body},
		commentChunks(body, 5000),
	)
	assert.Equal(
		t,
		[]string{body},
		commentChunks(body, 40),
	)
	assert.Equal(
		t,
		[]string{
			"0123456789abcdefghijABC",
			"DEFGHIJ\nKLMNO",
		},
		commentChunks(body, 20),
	)
	assert.Equal(
		t,
		[]string{
			"0123456789abcdefghijABC",
			"DEFGHIJ\nKL",
			"MNO",
		},
		commentChunks(body, 10),
	)

	assert.Equal(
		t,
		[]string{
			"0123456789abcdefghijABC",
			"DEFGHIJKLMNO",
		},
		commentChunks("0123456789abcdefghijABC<!-- KUBEAPPLY_SPLIT -->DEFGHIJKLMNO", 20),
	)
}

func testClusterConfigs(t *testing.T, profileDir string) []*config.ClusterConfig {
	clusterConfigs := []*config.ClusterConfig{
		{
			Cluster:      "test-cluster1",
			Region:       "test-region",
			Env:          "test-env",
			ExpandedPath: "expanded",
			ProfilePath:  profileDir,
		},
		{
			Cluster:      "test-cluster2",
			Region:       "test-region",
			Env:          "test-env",
			ExpandedPath: "expanded",
			ProfilePath:  profileDir,
		},
		{
			Cluster:      "test-cluster3",
			Region:       "test-region",
			Env:          "test-env",
			ExpandedPath: "expanded",
			ProfilePath:  profileDir,
		},
	}

	for _, clusterConfig := range clusterConfigs {
		require.NoError(
			t,
			clusterConfig.SetDefaults(
				fmt.Sprintf(
					"/git/repo/clusters/%s.yaml",
					clusterConfig.Cluster,
				),
				"/git/repo",
			),
		)
	}

	clusterConfigs[2].Subpaths = []string{"subpath1/subpath2"}

	return clusterConfigs
}
