package pullreq

import (
	"context"

	"github.com/segmentio/kubeapply/pkg/config"
)

// PullRequestClient is an interface for communicating with a pull request management system,
// i.e. Github, for a single pull request.
type PullRequestClient interface {
	// Init initializes the client by cloning the repo, etc.
	Init(ctx context.Context) error

	// GetCoveredClusters gets the configs for all clusters "covered" by a pull request.
	GetCoveredClusters(
		env string,
		selectedClusterGlobStrs []string,
		subpathOverride string,
	) ([]*config.ClusterConfig, error)

	// PostComment posts a non-error comment in the discussion stream for a pull request.
	PostComment(ctx context.Context, body string) error

	// PostErrorComment posts an error comment in the discussion stream for a pull request.
	PostErrorComment(
		ctx context.Context,
		env string,
		err error,
	) error

	// UpdateStatus updates the status of pull request.
	UpdateStatus(
		ctx context.Context,
		state string,
		stateContext string,
		description string,
		url string,
	) error

	// Merge merges the client's pull request into the base branch.
	Merge(ctx context.Context) error

	// Statuses gets all statuses for the pull request.
	Statuses(ctx context.Context) ([]PullRequestStatus, error)

	// IsDraft returns whether the pull request is a draft.
	IsDraft(ctx context.Context) bool

	// IsMerged returns whether the pull request is already merged.
	IsMerged(ctx context.Context) bool

	// IsMergeable returns whether the pull request is mergeble.
	IsMergeable(ctx context.Context) bool

	// Approved returns whether the client pull request is approved.
	Approved(ctx context.Context) bool

	// Base returns the base branch for the pull request.
	Base() string

	// BehindBy returns the number of commits that the pull request is behind the base branch by.
	BehindBy() int

	// HeadSHA returns the SHA of the head of this pull request.
	HeadSHA() string

	// Close cleans up the resources behind this pull request.
	Close() error
}

// PullRequestStatus represents the status of a single check on a pull request.
type PullRequestStatus struct {
	Context     string
	Description string
	State       string
}

// IsSuccess returns whether this PullRequestStatus is successful/green.
func (prc PullRequestStatus) IsSuccess() bool {
	return prc.State == "success"
}
