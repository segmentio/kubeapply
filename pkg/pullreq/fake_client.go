package pullreq

import (
	"context"
	"fmt"

	"github.com/segmentio/kubeapply/pkg/config"
)

var _ PullRequestClient = (*FakePullRequestClient)(nil)

// FakePullRequestClient is a fake implementation of a PullRequestClient. For testing purposes
// only.
type FakePullRequestClient struct {
	ClusterConfigs  []*config.ClusterConfig
	Comments        []string
	RequestStatuses []PullRequestStatus
	BehindByVal     int
	ApprovedVal     bool
	Draft           bool
	Mergeable       bool
	Merged          bool
}

// Init initializes this client.
func (prc *FakePullRequestClient) Init(ctx context.Context) error {
	return nil
}

// GetCoveredClusters returns the configs of the clusters that are affected by
// this pull request.
func (prc *FakePullRequestClient) GetCoveredClusters(
	env string,
	selectedClusterIDs []string,
	subpathOverride string,
) ([]*config.ClusterConfig, error) {
	selectedClusterIDsMap := map[string]struct{}{}
	for _, selectedClusterID := range selectedClusterIDs {
		selectedClusterIDsMap[selectedClusterID] = struct{}{}
	}

	coveredClusters := []*config.ClusterConfig{}

	for _, clusterConfig := range prc.ClusterConfigs {
		if len(selectedClusterIDsMap) > 0 {
			if _, ok := selectedClusterIDsMap[clusterConfig.DescriptiveName()]; !ok {
				continue
			}
		}

		if env != "" && clusterConfig.Env != env {
			continue
		}

		if subpathOverride != "" {
			clusterConfig.Subpath = subpathOverride
		} else {
			clusterConfig.Subpath = "."
		}

		coveredClusters = append(coveredClusters, clusterConfig)
	}

	return coveredClusters, nil
}

// PostComment posts a fake comment.
func (prc *FakePullRequestClient) PostComment(ctx context.Context, body string) error {
	prc.Comments = append(prc.Comments, fmt.Sprintf("Normal comment: %s", body))
	return nil
}

// PostErrorComment posts a fake error comment.
func (prc *FakePullRequestClient) PostErrorComment(
	ctx context.Context,
	env string,
	err error,
) error {
	prc.Comments = append(prc.Comments, fmt.Sprintf("Error comment: %+v", err))
	return nil
}

// UpdateStatus updates a status in this pull request.
func (prc *FakePullRequestClient) UpdateStatus(
	ctx context.Context,
	state string,
	stateContext string,
	description string,
	url string,
) error {
	for s, status := range prc.RequestStatuses {
		if status.Context == stateContext {
			// Update in-place
			prc.RequestStatuses[s].Description = description
			prc.RequestStatuses[s].State = state
			return nil
		}
	}

	prc.RequestStatuses = append(
		prc.RequestStatuses,
		PullRequestStatus{
			Context:     stateContext,
			Description: description,
			State:       state,
		},
	)

	return nil
}

// Statuses returns all of the statuses associated with this pull request.
func (prc *FakePullRequestClient) Statuses(
	ctx context.Context,
) ([]PullRequestStatus, error) {
	return prc.RequestStatuses, nil
}

// Merge does a fake merge of this pull request.
func (prc *FakePullRequestClient) Merge(
	ctx context.Context,
) error {
	prc.Merged = true
	return nil
}

// IsDraft returns whether this pull request is a draft.
func (prc *FakePullRequestClient) IsDraft(ctx context.Context) bool {
	return prc.Draft
}

// IsMerged returns whether this pull request has been merged.
func (prc *FakePullRequestClient) IsMerged(ctx context.Context) bool {
	return prc.Merged
}

// IsMergeable returns whether this pull request is mergeable.
func (prc *FakePullRequestClient) IsMergeable(ctx context.Context) bool {
	return prc.Mergeable
}

// Approved returns whether this pull request is approved.
func (prc *FakePullRequestClient) Approved(ctx context.Context) bool {
	return prc.ApprovedVal
}

// Base returns the base branch for this pull request.
func (prc *FakePullRequestClient) Base() string {
	return "master"
}

// BehindBy returns the number of commits that this pull request is behind
// the base by.
func (prc *FakePullRequestClient) BehindBy() int {
	return prc.BehindByVal
}

// HeadSHA returns the git SHA of the HEAD of the branch on which this
// pull request is based.
func (prc *FakePullRequestClient) HeadSHA() string {
	return "test-sha"
}

// Close closes the client.
func (prc *FakePullRequestClient) Close() error {
	return nil
}
