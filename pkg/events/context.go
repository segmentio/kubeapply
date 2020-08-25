package events

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/go-github/v30/github"
	"github.com/segmentio/kubeapply/pkg/pullreq"
	log "github.com/sirupsen/logrus"
)

const (
	webhookTypeHeader = "X-Github-Event"
)

// WebhookContext represents the full context of a webhook call from Github. It includes
// details on the pull request, comment, repo, etc.
type WebhookContext struct {
	pullRequestClient pullreq.PullRequestClient
	owner             string
	repo              string
	pullRequestNum    int
	commentType       commentType
	pullRequestEvent  *github.PullRequestEvent
	issueCommentEvent *github.IssueCommentEvent
}

// NewWebhookContext converts a webhook object into a WebhookContext, if possible.
func NewWebhookContext(
	webhookType string,
	webhookBody []byte,
	githubToken string,
) (*WebhookContext, error) {
	webhookObj, err := github.ParseWebHook(webhookType, webhookBody)
	if err != nil {
		return nil, fmt.Errorf("Could not parse webhook: %+v", err)
	}

	webhookBytes, err := json.MarshalIndent(webhookObj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Could not marshal webhook json: %+v", err)
	}

	log.Infof(
		"Got webhook of type %s (%+v): %s",
		webhookType,
		reflect.TypeOf(webhookObj),
		string(webhookBytes),
	)

	switch event := webhookObj.(type) {
	case *github.PullRequestEvent:
		action := event.GetAction()

		if action != "opened" && action != "synchronize" {
			log.Infof("Got non-matching pull request event action: %s", action)
			return nil, nil
		}

		owner, repoName := parseRepoName(event.GetRepo())
		pullRequestNum := event.GetPullRequest().GetNumber()

		client := pullreq.NewGHPullRequestClient(
			githubToken,
			owner,
			repoName,
			pullRequestNum,
		)
		return &WebhookContext{
			pullRequestClient: client,
			owner:             owner,
			repo:              repoName,
			pullRequestNum:    pullRequestNum,
			pullRequestEvent:  event,
		}, nil
	case *github.IssueCommentEvent:
		issue := event.GetIssue()

		if !issue.IsPullRequest() {
			log.Info("Issue is not for a pull request")
			return nil, nil
		} else if event.GetAction() != "created" {
			log.Infof("Issue comment action is not created: %s", event.GetAction())
			return nil, nil
		}

		owner, repoName := parseRepoName(event.GetRepo())

		pullRequestURL := issue.GetPullRequestLinks().GetURL()
		urlComponents := strings.Split(pullRequestURL, "/")
		pullRequestNum, err := strconv.Atoi(urlComponents[len(urlComponents)-1])
		if err != nil {
			return nil, err
		}

		commentBody := event.GetComment().GetBody()
		commentType := commentBodyToType(commentBody)
		if commentType == commentTypeOther {
			log.Info("Comment body is neither a command nor an apply result, returning")
			return nil, nil
		}

		client := pullreq.NewGHPullRequestClient(
			githubToken,
			owner,
			repoName,
			pullRequestNum,
		)
		return &WebhookContext{
			pullRequestClient: client,
			owner:             owner,
			repo:              repoName,
			pullRequestNum:    pullRequestNum,
			commentType:       commentType,
			issueCommentEvent: event,
		}, nil
	default:
		log.Infof("Got irrelevant event type: %+v", reflect.TypeOf(event))
		return nil, nil
	}
}

// Close closes the underlying clients associated with this WebhookContext.
func (w *WebhookContext) Close() error {
	return w.pullRequestClient.Close()
}

func parseRepoName(repo *github.Repository) (string, string) {
	repoFullName := repo.GetFullName()
	repoComponents := strings.Split(repoFullName, "/")

	return repoComponents[0], repoComponents[1]
}

// GetWebhookTypeLambdaHeaders gets the webhook type from lambda-type
// headers.
func GetWebhookTypeLambdaHeaders(headers map[string]string) string {
	return headers[strings.ToLower(webhookTypeHeader)]
}

// GetWebhookTypeHTTPHeaders gets the webhook type from http-type headers.
func GetWebhookTypeHTTPHeaders(headers http.Header) string {
	return headers.Get(webhookTypeHeader)
}
