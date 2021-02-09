package subcmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v30/github"
	"github.com/segmentio/kubeapply/pkg/cluster"
	kaevents "github.com/segmentio/kubeapply/pkg/events"
	"github.com/segmentio/kubeapply/pkg/pullreq"
	"github.com/segmentio/kubeapply/pkg/stats"
	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var pullRequestCmd = &cobra.Command{
	Use:     "pull-request",
	Short:   "pull-request mimics the behavior of the webhook lambda (for testing only)",
	PreRunE: pullRequestPreRun,
	RunE:    pullRequestRun,
}

type pullRequestFlags struct {
	// Whether to automerge if applies in all clusters have completed successfully
	automerge bool

	// The body of the comment in the webhook
	commentBody string

	// Environment to evaluate hook in
	env string

	// Type of event in github webhook
	eventType string

	// Key for requests to github API (via github app private key)
	githubAppKey string

	// Github app ID
	githubAppID string

	// Github app installation ID
	githubAppInstallationID string

	// Token for requests to github API (via personal token)
	githubToken string

	// Number of the pull request in the argument repo
	pullRequestNum int

	// Full name of the repo, in [owner]/[name] format
	repo string

	// Whether to be strict about checking for approvals and green github status.
	//
	// Deprecated, to be replaced by the values below.
	strictCheck bool

	// Whether green CI is required to apply
	greenCIRequired bool

	// Whether a review is required to apply
	reviewRequired bool
}

var pullRequestFlagValues pullRequestFlags

func init() {
	pullRequestCmd.Flags().BoolVar(
		&pullRequestFlagValues.automerge,
		"automerge",
		false,
		"Automerge value for kubeapply lambda",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.commentBody,
		"comment-body",
		"kubeapply help",
		"Comment in pull request",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.env,
		"env",
		"stage",
		"Environment for kubeapply lambda",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.eventType,
		"event-type",
		"comment",
		"Event type",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.githubToken,
		"github-token",
		"",
		"Token for github posts",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.githubAppKey,
		"github-app-key",
		"",
		"App key to use for github posts",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.githubAppID,
		"github-app-id",
		"",
		"App ID for github app",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.githubAppInstallationID,
		"github-app-installation-id",
		"",
		"Installation ID for github app",
	)
	pullRequestCmd.Flags().BoolVar(
		&pullRequestFlagValues.greenCIRequired,
		"green-ci-required",
		false,
		"Whether a green CI is required to apply",
	)
	pullRequestCmd.Flags().IntVar(
		&pullRequestFlagValues.pullRequestNum,
		"pull-request",
		0,
		"Pull request number",
	)
	pullRequestCmd.Flags().StringVar(
		&pullRequestFlagValues.repo,
		"repo",
		"",
		"Repo to post comment in, in format [owner]/[name]",
	)
	pullRequestCmd.Flags().BoolVar(
		&pullRequestFlagValues.reviewRequired,
		"review-required",
		false,
		"Whether a review is required to apply",
	)
	pullRequestCmd.Flags().BoolVar(
		&pullRequestFlagValues.strictCheck,
		"strict-check",
		false,
		"Strict-check value for kubeapply lambda",
	)

	pullRequestCmd.MarkFlagRequired("repo")

	RootCmd.AddCommand(pullRequestCmd)
}

func pullRequestPreRun(cmd *cobra.Command, args []string) error {
	matches, err := regexp.MatchString(
		"[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+",
		pullRequestFlagValues.repo,
	)
	if err != nil {
		return err
	}
	if !matches {
		return errors.New("Repo must be in format [owner]/[name]")
	}

	if pullRequestFlagValues.githubToken == "" &&
		(pullRequestFlagValues.githubAppKey == "" ||
			pullRequestFlagValues.githubAppID == "" ||
			pullRequestFlagValues.githubAppInstallationID == "") {
		return errors.New("Must set either github token or app key, id, and installation")
	}

	return nil
}

func pullRequestRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	var webhookType string
	var webhookObj interface{}

	var accessToken string

	if pullRequestFlagValues.githubToken == "" {
		jwt, err := pullreq.GenerateJWT(
			pullRequestFlagValues.githubAppKey,
			pullRequestFlagValues.githubAppID,
		)
		if err != nil {
			return err
		}

		appAccessToken, err := pullreq.GenerateAccessToken(
			ctx,
			jwt,
			pullRequestFlagValues.githubAppInstallationID,
		)
		if err != nil {
			return err
		}
		accessToken = appAccessToken.Token
	} else {
		accessToken = pullRequestFlagValues.githubToken
	}

	switch pullRequestFlagValues.eventType {
	case "comment":
		webhookType = "issue_comment"
		webhookObj = &github.IssueCommentEvent{
			Action: aws.String("created"),
			Issue: &github.Issue{
				PullRequestLinks: &github.PullRequestLinks{
					URL: aws.String(
						fmt.Sprintf(
							"https://github.com/%s/pull/%d",
							pullRequestFlagValues.repo,
							pullRequestFlagValues.pullRequestNum,
						),
					),
				},
			},
			Repo: &github.Repository{
				FullName: aws.String(pullRequestFlagValues.repo),
			},
			Comment: &github.IssueComment{
				Body: aws.String(pullRequestFlagValues.commentBody),
			},
		}
	case "create-pull-request":
		webhookType = "pull_request"
		webhookObj = &github.PullRequestEvent{
			Action: aws.String("opened"),
			PullRequest: &github.PullRequest{
				Number: aws.Int(pullRequestFlagValues.pullRequestNum),
			},
			Repo: &github.Repository{
				FullName: aws.String(pullRequestFlagValues.repo),
			},
		}
	case "synchronize-pull-request":
		webhookType = "pull_request"
		webhookObj = &github.PullRequestEvent{
			Action: aws.String("synchronized"),
			PullRequest: &github.PullRequest{
				Number: aws.Int(pullRequestFlagValues.pullRequestNum),
			},
			Repo: &github.Repository{
				FullName: aws.String(pullRequestFlagValues.repo),
			},
		}
	default:
		return fmt.Errorf("Unrecognized event type: %s", pullRequestFlagValues.eventType)
	}

	webhookBytes, err := json.Marshal(webhookObj)
	if err != nil {
		return err
	}

	webhookContext, err := kaevents.NewWebhookContext(
		webhookType,
		webhookBytes,
		accessToken,
	)
	if err != nil {
		return err
	} else if webhookContext == nil {
		return errors.New("Webhook context is nil")
	}
	defer webhookContext.Close()

	statsClient := stats.NewFakeStatsClient()

	webhookHandler := kaevents.NewWebhookHandler(
		statsClient,
		cluster.NewKubeClusterClient,
		kaevents.WebhookHandlerSettings{
			LogsURL:               "https://github.com/segmentio/kubeapply",
			Env:                   pullRequestFlagValues.env,
			Version:               version.Version,
			UseLocks:              true,
			ApplyConsistencyCheck: false,
			Automerge:             pullRequestFlagValues.automerge,
			StrictCheck:           pullRequestFlagValues.strictCheck,
			GreenCIRequired:       pullRequestFlagValues.greenCIRequired,
			ReviewRequired:        pullRequestFlagValues.reviewRequired,
			Debug:                 debug,
		},
	)
	resp := webhookHandler.HandleWebhook(
		ctx,
		webhookContext,
	)
	log.Infof("Webhook response: %+v", resp)

	return nil
}
