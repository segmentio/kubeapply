package pullreq

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var regenerateStr = os.Getenv("REGENERATE_TESTDATA")

func TestApplyComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.Nil(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpath = "test/subpath"

	pullRequestClient := NewFakePullRequestClient(clusterConfigs, nil, 0, true)

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
	require.Nil(t, err)

	expectedOutput := "testdata/comments/apply.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.Nil(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.Nil(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestDiffComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.Nil(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpath = "test/subpath"

	pullRequestClient := NewFakePullRequestClient(clusterConfigs, nil, 0, true)

	diffs := []ClusterDiff{
		{
			ClusterConfig: clusterConfigs[0],
			RawDiffs: strings.Join(
				[]string{
					"something",
					"--- file1",
					"+++ file2",
					"+ diff1",
					"- diff2",
					"",
					"--- file3",
					"+++ file4",
					"+ diff1",
					"- diff2",
				},
				"\n",
			),
		},
		{
			ClusterConfig: clusterConfigs[1],
			RawDiffs:      "",
		},
	}

	commentData := DiffCommentData{
		ClusterDiffs:      diffs,
		PullRequestClient: pullRequestClient,
		Env:               "stage",
	}

	result, err := FormatDiffComment(commentData)
	require.Nil(t, err)

	expectedOutput := "testdata/comments/diffs.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.Nil(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.Nil(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestDiffCommentBehind(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.Nil(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpath = "test/subpath"

	pullRequestClient := NewFakePullRequestClient(clusterConfigs, nil, 3, true)

	diffs := []ClusterDiff{
		{
			ClusterConfig: clusterConfigs[0],
			RawDiffs:      "these are diffs",
		},
	}

	commentData := DiffCommentData{
		ClusterDiffs:      diffs,
		PullRequestClient: pullRequestClient,
		Env:               "stage",
	}

	result, err := FormatDiffComment(commentData)
	require.Nil(t, err)

	expectedOutput := "testdata/comments/diffs-behind.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.Nil(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.Nil(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestErrorComment(t *testing.T) {
	commentData := ErrorCommentData{
		Error: fmt.Errorf("This is an error!"),
		Env:   "stage",
	}

	result, err := FormatErrorComment(commentData)
	require.Nil(t, err)

	expectedOutput := "testdata/comments/error.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.Nil(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.Nil(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestHelpComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.Nil(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpath = "test/subpath"

	commentData := HelpCommentData{
		ClusterConfigs: clusterConfigs,
		Env:            "stage",
	}

	result, err := FormatHelpComment(commentData)
	require.Nil(t, err)

	expectedOutput := "testdata/comments/help.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.Nil(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.Nil(t, err)
		assert.Equal(t, string(contents), result)
	}
}

func TestStatusComment(t *testing.T) {
	profileDir, err := ioutil.TempDir("", "profile")
	require.Nil(t, err)
	defer os.RemoveAll(profileDir)

	clusterConfigs := testClusterConfigs(t, profileDir)
	clusterConfigs[0].Subpath = "test/subpath"

	pullRequestClient := NewFakePullRequestClient(clusterConfigs, nil, 0, true)

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
	require.Nil(t, err)

	expectedOutput := "testdata/comments/statuses.md"

	if strings.ToLower(regenerateStr) == "true" {
		err = ioutil.WriteFile(expectedOutput, []byte(result), 0644)
		require.Nil(t, err)
	} else {
		contents, err := ioutil.ReadFile(expectedOutput)
		require.Nil(t, err)
		assert.Equal(t, string(contents), result)
	}
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
		require.Nil(
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

	clusterConfigs[2].Subpath = "subpath1/subpath2"

	return clusterConfigs
}
