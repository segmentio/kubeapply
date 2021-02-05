package events

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v30/github"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/pullreq"
	"github.com/segmentio/kubeapply/pkg/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleWebhook(t *testing.T) {
	type commentMatch struct {
		contains       []string
		doesNotContain []string
	}

	type statusMatch struct {
		context string
		state   string
	}

	type testCase struct {
		description     string
		strictCheck     bool
		automerge       bool
		kubectlErr      bool
		input           *WebhookContext
		expRespStatus   int
		expMerged       bool
		expComments     []commentMatch
		expRepoStatuses []statusMatch
	}

	profileDir, err := ioutil.TempDir("", "profile")
	require.Nil(t, err)
	defer os.RemoveAll(profileDir)

	testClusterConfigs := []*config.ClusterConfig{
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
			Env:          "test-env2",
			ExpandedPath: "expanded",
			ProfilePath:  profileDir,
		},
	}

	for _, clusterConfig := range testClusterConfigs {
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

	testCases := []testCase{
		{
			description:   "nil input",
			input:         nil,
			expRespStatus: 200,
		},
		{
			description: "open new pull request",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				pullRequestEvent: &github.PullRequestEvent{
					Action: aws.String("opened"),
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply help (test-env)",
						"test-cluster1",
						"test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster3",
					},
				},
				{
					contains: []string{
						"Kubeapply diff result (test-env)",
						"diff result for test-cluster1",
						"diff result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "sync pull request",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				pullRequestEvent: &github.PullRequestEvent{
					Action: aws.String("synchronize"),
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply diff result (test-env)",
						"diff result for test-cluster1",
						"diff result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "open new pull request no clusters",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  []*config.ClusterConfig{},
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				pullRequestEvent: &github.PullRequestEvent{
					Action: aws.String("opened"),
				},
			},
			expRespStatus:   200,
			expComments:     []commentMatch{},
			expRepoStatuses: []statusMatch{},
		},
		{
			description: "kubeapply help",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply help"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply help",
					},
				},
			},
			expRepoStatuses: []statusMatch{},
		},
		{
			description: "kubeapply status",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply status"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply cluster status",
						"summary test-cluster1",
						"summary test-cluster2",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/status (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply diff",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply diff"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply diff result (test-env)",
						"diff result for test-cluster1",
						"diff result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply diff single cluster",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply diff test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply diff result (test-env)",
						"diff result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply diff kubectl error",
			kubectlErr:  true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply diff test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 500,
			expComments: []commentMatch{
				{
					contains: []string{
						"Error comment: Error diffing for cluster test-env:test-region:test-cluster2: kubectl error",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "failure",
				},
			},
		},
		{
			description: "kubeapply diff behind",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					BehindByVal:     5,
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply diff test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply diff result (test-env)",
						"diff result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply diff unapproved",
			strictCheck: true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply diff test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply diff result (test-env)",
						"diff result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply diff no clusters",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  []*config.ClusterConfig{},
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply diff"),
					},
				},
			},
			expRespStatus:   200,
			expComments:     []commentMatch{},
			expRepoStatuses: []statusMatch{},
		},
		{
			description: "kubeapply apply",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply apply result (test-env)",
						"apply result for test-cluster1",
						"apply result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply apply mark failure",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply --no-auto-merge"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply apply result (test-env)",
						"apply result for test-cluster1",
						"apply result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "failure",
				},
			},
		},
		{
			description: "kubeapply apply single cluster",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String(
							"kubeapply apply test-env:test-region:test-cluster2 --subpath=path1",
						),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply apply result (test-env)",
						"apply result for test-cluster2 with paths [/git/repo/clusters/expanded/path1]",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply apply kubectl error",
			kubectlErr:  true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 500,
			expComments: []commentMatch{
				{
					contains: []string{
						"Error comment: Error applying for cluster test-env:test-region:test-cluster2: kubectl error",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "failure",
				},
			},
		},
		{
			description: "kubeapply apply behind",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					BehindByVal:     5,
					ApprovedVal:     true,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 500,
			expComments: []commentMatch{
				{
					contains: []string{
						"Error comment: Cannot run apply",
						"branch is behind master by 5",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "failure",
				},
			},
		},
		{
			description: "kubeapply apply not approved (not strict)",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     false,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply apply result (test-env)",
						"apply result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply apply not approved (strict)",
			strictCheck: true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs:  testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{},
					ApprovedVal:     false,
					Mergeable:       true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 500,
			expComments: []commentMatch{
				{
					contains: []string{
						"Error comment: Cannot run apply",
						"is not approved",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "kubeapply/apply (test-env)",
					state:   "failure",
				},
			},
		},
		{
			description: "kubeapply bad status (not strict)",
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "failure",
						},
					},
					ApprovedVal: true,
					Mergeable:   true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply apply result (test-env)",
						"apply result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "failure",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "kubeapply bad status (strict)",
			strictCheck: true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "failure",
						},
					},
					ApprovedVal: true,
					Mergeable:   true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 500,
			expComments: []commentMatch{
				{
					contains: []string{
						"Error comment: Cannot run apply",
						"status is not green",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "failure",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "failure",
				},
			},
		},
		{
			description: "kubeapply apply bad kubeapply status (strict)",
			strictCheck: true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "success",
						},
						{
							Context: "kubeapply/apply (test-env)",
							State:   "failure",
						},
					},
					ApprovedVal: true,
					Mergeable:   true,
				},
				commentType: commentTypeCommand,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
					Comment: &github.IssueComment{
						Body: aws.String("kubeapply apply test-env:test-region:test-cluster2"),
					},
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Kubeapply apply result (test-env)",
						"apply result for test-cluster2",
					},
					doesNotContain: []string{
						"test-cluster1",
						"test-cluster3",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "success",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "apply result with automerge",
			strictCheck: true,
			automerge:   true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "success",
						},
						{
							Context:     "kubeapply/diff (test-env)",
							State:       "success",
							Description: "success for clusters cluster1,cluster2 and others",
						},
						{
							Context:     "kubeapply/apply (test-env)",
							State:       "success",
							Description: "success for clusters cluster1,cluster2",
						},
					},
					ApprovedVal: true,
					Mergeable:   true,
				},
				commentType: commentTypeApplyResult,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
				},
			},
			expRespStatus: 200,
			expComments: []commentMatch{
				{
					contains: []string{
						"Auto-merging",
					},
				},
			},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "success",
				},
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
			expMerged: true,
		},
		{
			description: "apply result with automerge not possible due to draft",
			strictCheck: true,
			automerge:   true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "success",
						},
						{
							Context:     "kubeapply/diff (test-env)",
							State:       "success",
							Description: "success for clusters cluster1,cluster2",
						},
						{
							Context:     "kubeapply/apply (test-env)",
							State:       "success",
							Description: "success for clusters cluster1,cluster2",
						},
					},
					ApprovedVal: true,
					Draft:       true,
					Mergeable:   true,
				},
				commentType: commentTypeApplyResult,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
				},
			},
			expRespStatus: 200,
			expComments:   []commentMatch{},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "success",
				},
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "apply result with automerge not possible due to bad status",
			strictCheck: true,
			automerge:   true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "failure",
						},
						{
							Context: "kubeapply/diff (test-env)",
							State:   "success",
						},
						{
							Context: "kubeapply/apply (test-env)",
							State:   "success",
						},
					},
					ApprovedVal: true,
					Mergeable:   true,
				},
				commentType: commentTypeApplyResult,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
				},
			},
			expRespStatus: 200,
			expComments:   []commentMatch{},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "failure",
				},
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
		{
			description: "apply result with automerge not possible due to incomplete workflow",
			strictCheck: true,
			automerge:   true,
			input: &WebhookContext{
				pullRequestClient: &pullreq.FakePullRequestClient{
					ClusterConfigs: testClusterConfigs,
					RequestStatuses: []pullreq.PullRequestStatus{
						{
							Context: "check",
							State:   "success",
						},
						{
							Context: "kubeapply/diff (test-env)",
							State:   "success",
						},
						{
							Context: "kubeapply/diff (production)",
							State:   "success",
						},
						{
							Context: "kubeapply/apply (test-env)",
							State:   "success",
						},
					},
					ApprovedVal: true,
					Mergeable:   true,
				},
				commentType: commentTypeApplyResult,
				issueCommentEvent: &github.IssueCommentEvent{
					Action: aws.String("created"),
				},
			},
			expRespStatus: 200,
			expComments:   []commentMatch{},
			expRepoStatuses: []statusMatch{
				{
					context: "check",
					state:   "success",
				},
				{
					context: "kubeapply/diff (test-env)",
					state:   "success",
				},
				{
					context: "kubeapply/diff (production)",
					state:   "success",
				},
				{
					context: "kubeapply/apply (test-env)",
					state:   "success",
				},
			},
		},
	}

	for _, testCase := range testCases {
		var generator cluster.ClusterClientGenerator

		if testCase.kubectlErr {
			generator = cluster.NewFakeClusterClientError
		} else {
			generator = cluster.NewFakeClusterClient
		}

		handler := NewWebhookHandler(
			stats.NewFakeStatsClient(),
			generator,
			WebhookHandlerSettings{
				LogsURL:     "test-url",
				Env:         "test-env",
				Version:     "1.2.3",
				StrictCheck: testCase.strictCheck,
				Automerge:   testCase.automerge,
				Debug:       false,
			},
		)

		ctx := context.Background()
		result := handler.HandleWebhook(ctx, testCase.input)

		assert.Equal(
			t,
			testCase.expRespStatus,
			result.StatusCode,
			testCase.description,
		)
		if testCase.input != nil {
			pullRequestClient := testCase.input.pullRequestClient.(*pullreq.FakePullRequestClient)

			assert.Equal(
				t,
				len(testCase.expComments),
				len(pullRequestClient.Comments),
				testCase.description,
			)
			for c, commentStr := range pullRequestClient.Comments {
				if c < len(testCase.expComments) {
					match := testCase.expComments[c]
					for _, containsStr := range match.contains {
						assert.True(
							t,
							strings.Contains(commentStr, containsStr),
							"%s: '%s' does not contain %s",
							testCase.description,
							commentStr,
							containsStr,
						)
					}
					for _, doesNotContainStr := range match.doesNotContain {
						assert.False(
							t,
							strings.Contains(commentStr, doesNotContainStr),
							"%s: '%s' contains %s",
							testCase.description,
							commentStr,
							doesNotContainStr,
						)
					}
				}
			}

			assert.Equal(
				t,
				len(testCase.expRepoStatuses),
				len(pullRequestClient.RequestStatuses),
				testCase.description,
			)
			statusMap := map[string]pullreq.PullRequestStatus{}
			for _, status := range pullRequestClient.RequestStatuses {
				statusMap[status.Context] = status
			}

			for _, expStatus := range testCase.expRepoStatuses {
				assert.Equal(
					t,
					expStatus.state,
					statusMap[expStatus.context].State,
					testCase.description,
				)
			}

			assert.Equal(
				t,
				testCase.expMerged,
				pullRequestClient.Merged,
				testCase.description,
			)
		}
	}
}
