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
	Merged          bool
}

func NewFakePullRequestClient(
	clusterConfigs []*config.ClusterConfig,
	statuses []PullRequestStatus,
	behindBy int,
	approved bool,
) *FakePullRequestClient {
	return &FakePullRequestClient{
		ClusterConfigs:  clusterConfigs,
		Comments:        []string{},
		RequestStatuses: statuses,
		BehindByVal:     behindBy,
		ApprovedVal:     approved,
	}
}

func (prc *FakePullRequestClient) Init(ctx context.Context) error {
	return nil
}

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

func (prc *FakePullRequestClient) PostComment(ctx context.Context, body string) error {
	prc.Comments = append(prc.Comments, fmt.Sprintf("Normal comment: %s", body))
	return nil
}

func (prc *FakePullRequestClient) PostErrorComment(
	ctx context.Context,
	env string,
	err error,
) error {
	prc.Comments = append(prc.Comments, fmt.Sprintf("Error comment: %+v", err))
	return nil
}

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

func (prc *FakePullRequestClient) Statuses(
	ctx context.Context,
) ([]PullRequestStatus, error) {
	return prc.RequestStatuses, nil
}

func (prc *FakePullRequestClient) Merge(
	ctx context.Context,
) error {
	prc.Merged = true
	return nil
}

func (prc *FakePullRequestClient) Approved(ctx context.Context) bool {
	return prc.ApprovedVal
}

func (prc *FakePullRequestClient) Base() string {
	return "master"
}

func (prc *FakePullRequestClient) BehindBy() int {
	return prc.BehindByVal
}

func (prc *FakePullRequestClient) HeadSHA() string {
	return "test-sha"
}

func (prc *FakePullRequestClient) Close() error {
	return nil
}
