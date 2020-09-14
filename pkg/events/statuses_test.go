package events

import (
	"context"
	"testing"

	"github.com/segmentio/kubeapply/pkg/pullreq"
	"github.com/stretchr/testify/assert"
)

func TestStatusOK(t *testing.T) {
	type testCase struct {
		statuses             []pullreq.PullRequestStatus
		expAllGreen          bool
		expOKToApply         bool
		expWorkflowCompleted bool
	}

	testCases := []testCase{
		{
			statuses:             []pullreq.PullRequestStatus{},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context:     "check",
					State:       "failure",
					Description: "check failed",
				},
			},
			expAllGreen:          false,
			expOKToApply:         false,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context:     "check",
					State:       "success",
					Description: "check succeeded",
				},
			},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context:     "check",
					State:       "success",
					Description: "check succeeded",
				},
				{
					Context:     "kubeapply/diff (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
			},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context:     "check",
					State:       "success",
					Description: "check succeeded",
				},
				{
					Context:     "kubeapply/diff (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
				{
					Context:     "kubeapply/apply (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
				{
					Context:     "kubeapply/diff (production)",
					State:       "success",
					Description: "successful for clusters cluster3,cluster4",
				},
				{
					Context:     "kubeapply/apply (production)",
					State:       "failure",
					Description: "failure for clusters cluster3,cluster4",
				},
			},
			expAllGreen:          false,
			expOKToApply:         true,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context: "check",
					State:   "success",
				},
				{
					Context:     "kubeapply/diff (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
				{
					Context:     "kubeapply/apply (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
				{
					Context:     "kubeapply/diff (production)",
					State:       "success",
					Description: "successful for clusters cluster3,cluster4",
				},
				{
					Context:     "kubeapply/apply (production)",
					State:       "success",
					Description: "successful for clusters cluster3,cluster4",
				},
			},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: true,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context: "check",
					State:   "success",
				},
				{
					Context:     "kubeapply/diff (stage)",
					State:       "success",
					Description: "successful for clusters cluster1",
				},
				{
					Context:     "kubeapply/apply (stage)",
					State:       "success",
					Description: "successful for clusters cluster1",
				},
			},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: true,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context: "check",
					State:   "success",
				},
				{
					Context:     "kubeapply/diff (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
				{
					Context:     "kubeapply/apply (stage)",
					State:       "success",
					Description: "successful for clusters cluster2",
				},
			},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context: "check",
					State:   "success",
				},
				{
					Context:     "kubeapply/diff (stage)",
					State:       "success",
					Description: "successful for clusters cluster1,cluster2",
				},
				{
					Context:     "kubeapply/apply (stage)",
					State:       "success",
					Description: "successful for clusters cluster1",
				},
				{
					Context:     "kubeapply/apply (stage)",
					State:       "success",
					Description: "successful for clusters cluster2",
				},
			},
			expAllGreen:          true,
			expOKToApply:         true,
			expWorkflowCompleted: true,
		},
	}

	ctx := context.Background()

	for index, testCase := range testCases {
		pullRequestClient := &pullreq.FakePullRequestClient{
			RequestStatuses: testCase.statuses,
		}

		okToApply := statusOKToApply(ctx, pullRequestClient)
		workflowCompleted := statusWorkflowCompleted(ctx, pullRequestClient)

		assert.Equal(
			t,
			testCase.expOKToApply,
			okToApply,
			"Ok to apply, test case %d", index,
		)

		assert.Equal(
			t,
			testCase.expWorkflowCompleted,
			workflowCompleted,
			"Workflow completed, test case %d", index,
		)
	}
}
