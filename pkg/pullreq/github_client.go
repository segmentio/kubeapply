package pullreq

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v30/github"
	"github.com/segmentio/kubeapply/pkg/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

var _ PullRequestClient = (*GHPullRequestClient)(nil)

// GHPullRequestClient is an implementation of PullRequestClient that hits the Github API. The
// actual work of communicating with Github is handled by a go-github client instance.
type GHPullRequestClient struct {
	*github.Client

	token          string
	owner          string
	repo           string
	pullRequestNum int

	pullRequest *github.PullRequest
	repoInfo    *github.Repository
	reviews     []*github.PullRequestReview

	branch     string
	base       string
	comparison *github.CommitsComparison
	issueNum   int
	issue      *github.Issue
	files      []*github.CommitFile
	status     *github.CombinedStatus

	clonePath string
}

func NewGHPullRequestClient(
	token string,
	owner string,
	repo string,
	pullRequestNum int,
) *GHPullRequestClient {
	return &GHPullRequestClient{
		token:          token,
		owner:          owner,
		repo:           repo,
		pullRequestNum: pullRequestNum,
	}
}

func (prc *GHPullRequestClient) Init(ctx context.Context) error {
	var err error

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: prc.token,
		},
	)
	tc := oauth2.NewClient(ctx, ts)
	prc.Client = github.NewClient(tc)

	log.Info("Getting pull request from github API")
	prc.pullRequest, _, err = prc.Client.PullRequests.Get(
		ctx,
		prc.owner,
		prc.repo,
		prc.pullRequestNum,
	)
	if err != nil {
		return err
	}

	prc.branch = prc.pullRequest.GetHead().GetRef()
	prc.base = prc.pullRequest.GetBase().GetRef()

	issueURL := prc.pullRequest.GetIssueURL()
	issueComponents := strings.Split(issueURL, "/")
	issueStr := issueComponents[len(issueComponents)-1]
	prc.issueNum, err = strconv.Atoi(issueStr)
	if err != nil {
		return err
	}

	log.Info("Getting pull request files from github API")
	currPage := 0

	for {
		currFiles, resp, err := prc.Client.PullRequests.ListFiles(
			ctx,
			prc.owner,
			prc.repo,
			prc.pullRequestNum,
			&github.ListOptions{
				Page:    currPage,
				PerPage: 50,
			},
		)
		if err != nil {
			return err
		}
		prc.files = append(prc.files, currFiles...)
		log.Infof(
			"Got %d changed files in page %d; next page is %d",
			len(currFiles),
			currPage,
			resp.NextPage,
		)

		if resp.NextPage <= currPage {
			break
		}

		currPage = resp.NextPage
	}

	log.Info("Getting pull request issue from github API")
	prc.issue, _, err = prc.Client.Issues.Get(
		ctx,
		prc.owner,
		prc.repo,
		prc.issueNum,
	)
	if err != nil {
		return err
	}

	log.Info("Getting pull request reviews from github API")
	prc.reviews, _, err = prc.Client.PullRequests.ListReviews(
		ctx,
		prc.owner,
		prc.repo,
		prc.pullRequestNum,
		&github.ListOptions{
			PerPage: 500,
		},
	)
	if err != nil {
		return err
	}
	log.Infof("Got %d existing reviews", len(prc.reviews))

	log.Info("Getting repo information from github API")
	prc.repoInfo, _, err = prc.Client.Repositories.Get(
		ctx,
		prc.owner,
		prc.repo,
	)
	if err != nil {
		return err
	}

	log.Info("Getting combined status from github API")
	prc.status, _, err = prc.Client.Repositories.GetCombinedStatus(
		ctx,
		prc.owner,
		prc.repo,
		prc.pullRequest.GetHead().GetSHA(),
		&github.ListOptions{
			PerPage: 500,
		},
	)

	log.Info("Getting up-to-date diff with base")
	prc.comparison, _, err = prc.Client.Repositories.CompareCommits(
		ctx,
		prc.owner,
		prc.repo,
		prc.base,
		prc.branch,
	)
	if err != nil {
		return err
	}

	prc.clonePath, err = ioutil.TempDir("", "kubeapply")
	if err != nil {
		return err
	}

	log.Infof(
		"Doing shallow clone of repo at branch %s in %s",
		prc.branch,
		prc.clonePath,
	)
	_, err = git.PlainClone(
		prc.clonePath,
		false,
		&git.CloneOptions{
			URL: fmt.Sprintf(
				"https://github.com/%s/%s.git",
				prc.owner,
				prc.repo,
			),
			Progress:      os.Stdout,
			ReferenceName: plumbing.NewBranchReferenceName(prc.branch),
			Auth: &http.BasicAuth{
				Username: "abc123", // This can be anything except an empty string
				Password: prc.token,
			},
			SingleBranch: true,
			Depth:        1,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (prc *GHPullRequestClient) GetCoveredClusters(
	env string,
	selectedClusterIDs []string,
	subpathOverride string,
) ([]*config.ClusterConfig, error) {
	return GetCoveredClusters(
		prc.clonePath,
		prc.files,
		env,
		selectedClusterIDs,
		subpathOverride,
	)
}

func (prc *GHPullRequestClient) PostComment(ctx context.Context, body string) error {
	log.Infof("Posting comment via github API: %s", body)
	_, _, err := prc.Client.Issues.CreateComment(
		ctx,
		prc.owner,
		prc.repo,
		prc.issueNum,
		&github.IssueComment{
			Body: aws.String(body),
		},
	)

	return err
}

func (prc *GHPullRequestClient) PostErrorComment(
	ctx context.Context,
	env string,
	err error,
) error {
	commentBody, err := FormatErrorComment(ErrorCommentData{Error: err, Env: env})
	if err != nil {
		return err
	}

	return prc.PostComment(
		ctx,
		commentBody,
	)
}

// UpdateStatus updates the status of the HEAD SHA of the branch in the pull request.
// Note that the "state" argument must be one of error, failure, pending, or success.
func (prc *GHPullRequestClient) UpdateStatus(
	ctx context.Context,
	state string,
	stateContext string,
	description string,
	url string,
) error {
	ref := prc.pullRequest.GetHead().GetSHA()

	log.Infof(
		"Updating status for ref %s via github API: %s %s %s %s",
		ref,
		state,
		stateContext,
		description,
		url,
	)

	_, _, err := prc.Client.Repositories.CreateStatus(
		ctx,
		prc.owner,
		prc.repo,
		ref,
		&github.RepoStatus{
			State:       aws.String(state),
			Context:     aws.String(stateContext),
			Description: aws.String(description),
			TargetURL:   aws.String(url),
		},
	)
	return err
}

func (prc *GHPullRequestClient) Merge(
	ctx context.Context,
) error {
	_, _, err := prc.Client.PullRequests.Merge(
		ctx,
		prc.owner,
		prc.repo,
		prc.pullRequestNum,
		fmt.Sprintf("Merged by kubeapply (pull request %d)", prc.pullRequestNum),
		&github.PullRequestOptions{
			MergeMethod: "squash",
		},
	)

	return err
}

func (prc *GHPullRequestClient) Statuses(
	ctx context.Context,
) ([]PullRequestStatus, error) {
	statuses := []PullRequestStatus{}

	for _, status := range prc.status.Statuses {
		statuses = append(
			statuses,
			PullRequestStatus{
				Context:     aws.StringValue(status.Context),
				State:       aws.StringValue(status.State),
				Description: aws.StringValue(status.Description),
			},
		)
	}

	return statuses, nil
}

func (prc *GHPullRequestClient) IsDraft(ctx context.Context) bool {
	return aws.BoolValue(prc.pullRequest.Draft)
}

func (prc *GHPullRequestClient) IsMerged(ctx context.Context) bool {
	return aws.BoolValue(prc.pullRequest.Merged)
}

func (prc *GHPullRequestClient) IsMergeable(ctx context.Context) bool {
	return aws.BoolValue(prc.pullRequest.Mergeable)
}

func (prc *GHPullRequestClient) Approved(ctx context.Context) bool {
	for _, review := range prc.reviews {
		if strings.ToLower(review.GetState()) == "approved" {
			return true
		}
	}

	return false
}

func (prc *GHPullRequestClient) Base() string {
	return prc.base
}

func (prc *GHPullRequestClient) BehindBy() int {
	if prc.comparison != nil {
		return aws.IntValue(prc.comparison.BehindBy)
	}

	return 0
}

func (prc *GHPullRequestClient) HeadSHA() string {
	if prc.pullRequest.Head.SHA != nil {
		return *prc.pullRequest.Head.SHA
	}

	return "unknown"
}

func (prc *GHPullRequestClient) Close() error {
	if prc.clonePath != "" {
		return os.RemoveAll(prc.clonePath)
	}

	return nil
}
