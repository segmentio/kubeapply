package events

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v30/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWebhookContext(t *testing.T) {
	type testCase struct {
		description       string
		input             interface{}
		webhookType       string
		expWebhookContext *WebhookContext
		expErr            bool
	}

	testCases := []testCase{
		{
			description:       "non-matching event type",
			input:             "not an event",
			expWebhookContext: nil,
			expErr:            true,
		},
		{
			description: "pull request opened",
			input: &github.PullRequestEvent{
				Action: aws.String("opened"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				PullRequest: &github.PullRequest{
					Number: aws.Int(50),
				},
			},
			webhookType: "pull_request",
			expWebhookContext: &WebhookContext{
				owner:          "segmentio",
				repo:           "test-repo",
				pullRequestNum: 50,
			},
			expErr: false,
		},
		{
			description: "pull request synchronized",
			input: &github.PullRequestEvent{
				Action: aws.String("synchronize"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				PullRequest: &github.PullRequest{
					Number: aws.Int(51),
				},
			},
			webhookType: "pull_request",
			expWebhookContext: &WebhookContext{
				owner:          "segmentio",
				repo:           "test-repo",
				pullRequestNum: 51,
			},
			expErr: false,
		},
		{
			description: "pull request closed",
			input: &github.PullRequestEvent{
				Action: aws.String("close"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				PullRequest: &github.PullRequest{
					Number: aws.Int(51),
				},
			},
			webhookType:       "pull_request",
			expWebhookContext: nil,
			expErr:            false,
		},
		{
			description: "comment created",
			input: &github.IssueCommentEvent{
				Action: aws.String("created"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: aws.String("https://github.com/segmentio/terracode-infra/pull/3119"),
					},
				},
				Comment: &github.IssueComment{
					Body: aws.String("kubeapply diff"),
				},
			},
			webhookType: "issue_comment",
			expWebhookContext: &WebhookContext{
				owner:          "segmentio",
				repo:           "test-repo",
				pullRequestNum: 3119,
			},
			expErr: false,
		},
		{
			description: "comment deleted",
			input: &github.IssueCommentEvent{
				Action: aws.String("deleted"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: aws.String("https://github.com/segmentio/terracode-infra/pull/3119"),
					},
				},
				Comment: &github.IssueComment{
					Body: aws.String("kubeapply diff"),
				},
			},
			webhookType:       "issue_comment",
			expWebhookContext: nil,
			expErr:            false,
		},
		{
			description: "non-kubeapply comment created",
			input: &github.IssueCommentEvent{
				Action: aws.String("created"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				Issue: &github.Issue{
					PullRequestLinks: &github.PullRequestLinks{
						URL: aws.String("https://github.com/segmentio/terracode-infra/pull/3119"),
					},
				},
				Comment: &github.IssueComment{
					Body: aws.String("non-kubeapply comment"),
				},
			},
			webhookType:       "issue_comment",
			expWebhookContext: nil,
			expErr:            false,
		},
		{
			description: "non-pull-request comment",
			input: &github.IssueCommentEvent{
				Action: aws.String("created"),
				Repo: &github.Repository{
					FullName: aws.String("segmentio/test-repo"),
				},
				Issue: &github.Issue{},
				Comment: &github.IssueComment{
					Body: aws.String("non-pull-request comment"),
				},
			},
			webhookType:       "issue_comment",
			expWebhookContext: nil,
			expErr:            false,
		},
	}

	for _, testCase := range testCases {
		inputBytes, err := json.Marshal(testCase.input)
		require.Nil(t, err)

		result, err := NewWebhookContext(
			testCase.webhookType,
			inputBytes,
			"test-github-token",
		)
		if testCase.expErr {
			assert.NotNil(t, err, testCase.description)
		} else if testCase.expWebhookContext == nil {
			assert.Nil(t, err, testCase.description)
			assert.Nil(t, result, testCase.description)
		} else {
			assert.Nil(t, err, testCase.description)
			assert.Equal(
				t,
				testCase.expWebhookContext.owner,
				result.owner,
				testCase.description,
			)
			assert.Equal(
				t,
				testCase.expWebhookContext.repo,
				result.repo,
				testCase.description,
			)
			assert.Equal(
				t,
				testCase.expWebhookContext.pullRequestNum,
				result.pullRequestNum,
				testCase.description,
			)

			pullRequestEvent, ok := testCase.input.(*github.PullRequestEvent)
			if ok {
				assert.Equal(
					t,
					pullRequestEvent,
					result.pullRequestEvent,
					testCase.description,
				)
			}

			issueCommentEvent, ok := testCase.input.(*github.IssueCommentEvent)
			if ok {
				assert.Equal(
					t,
					issueCommentEvent,
					result.issueCommentEvent,
					testCase.description,
				)
			}

			assert.NotNil(t, result.pullRequestClient, testCase.description)
		}
	}
}
