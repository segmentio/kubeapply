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
					Context: "check",
					State:   "failure",
				},
			},
			expAllGreen:          false,
			expOKToApply:         false,
			expWorkflowCompleted: false,
		},
		{
			statuses: []pullreq.PullRequestStatus{
				{
					Context: "check",
					State:   "success",
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
					Context: "kubeapply/diff (stage)",
					State:   "success",
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
					Context: "kubeapply/diff (stage)",
					State:   "success",
				},
				{
					Context: "kubeapply/apply (stage)",
					State:   "success",
				},
				{
					Context: "kubeapply/diff (production)",
					State:   "success",
				},
				{
					Context: "kubeapply/apply (production)",
					State:   "failure",
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
					Context: "kubeapply/diff (stage)",
					State:   "success",
				},
				{
					Context: "kubeapply/apply (stage)",
					State:   "success",
				},
				{
					Context: "kubeapply/diff (production)",
					State:   "success",
				},
				{
					Context: "kubeapply/apply (production)",
					State:   "success",
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
					Context: "kubeapply/diff (stage)",
					State:   "success",
				},
				{
					Context: "kubeapply/apply (stage)",
					State:   "success",
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
			"Test case %d", index,
		)

		assert.Equal(
			t,
			testCase.expWorkflowCompleted,
			workflowCompleted,
			"Test case %d", index,
		)
	}
}
