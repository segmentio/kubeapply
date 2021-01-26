package cluster

import (
	"testing"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
	"github.com/stretchr/testify/assert"
)

func TestSortedDiffResults(t *testing.T) {
	diffResults := []diff.Result{
		{
			Object: &apply.TypedKubeObj{
				KubeMetadata: apply.KubeMetadata{
					Name:      "test-name",
					Namespace: "test-namespace-1",
				},
				Kind: "test-kind",
			},
			Name: "resource-0",
		},
		{
			Object: &apply.TypedKubeObj{
				KubeMetadata: apply.KubeMetadata{
					Name:      "test-name",
					Namespace: "test-namespace-10",
				},
				Kind: "test-kind",
			},
			Name: "resource-1",
		},
		{
			Object: &apply.TypedKubeObj{
				KubeMetadata: apply.KubeMetadata{
					Name:      "test-name",
					Namespace: "test-namespace-2",
				},
				Kind: "test-kind",
			},
			Name: "resource-2",
		},
		{
			Object: &apply.TypedKubeObj{
				KubeMetadata: apply.KubeMetadata{
					Name:      "test-name",
					Namespace: "another-namespace",
				},
				Kind: "test-kind",
			},
			Name: "resource-3",
		},
		{
			Object: &apply.TypedKubeObj{
				KubeMetadata: apply.KubeMetadata{
					Name:      "test-name",
					Namespace: "another-namespace",
				},
				Kind: "another-kind",
			},
			Name: "resource-4",
		},
		{
			Object: &apply.TypedKubeObj{
				KubeMetadata: apply.KubeMetadata{
					Name:      "test-name-x",
					Namespace: "another-namespace",
				},
				Kind: "test-kind",
			},
			Name: "resource-5",
		},
		{
			Name: "resource-6",
		},
	}

	sortedResults := sortedDiffResults(diffResults)
	assert.Equal(
		t,
		sortedResults,
		[]diff.Result{
			diffResults[4],
			diffResults[3],
			diffResults[5],
			diffResults[6],
			diffResults[0],
			diffResults[2],
			diffResults[1],
		},
	)
}

func TestSortedApplyResults(t *testing.T) {
	applyResults := []apply.Result{
		{
			Name:       "resource-5",
			Namespace:  "test-namespace-003",
			Kind:       "test-kind",
			OldVersion: "0",
		},
		{
			Name:       "resource-1",
			Namespace:  "test-namespace-10",
			Kind:       "test-kind",
			OldVersion: "1",
		},
		{
			Name:       "resource-2",
			Namespace:  "test-namespace-3",
			Kind:       "test-kind",
			OldVersion: "2",
		},
		{
			Name:       "resource-3",
			Namespace:  "another-namespace",
			Kind:       "test-kind",
			OldVersion: "3",
		},
		{
			Name:       "resource-10",
			Namespace:  "another-namespace",
			Kind:       "test-kind",
			OldVersion: "4",
		},
		{
			Name:       "resource-100",
			Namespace:  "another-namespace",
			Kind:       "another-kind",
			OldVersion: "5",
		},
	}

	sortedResults := sortedApplyResults(applyResults)
	assert.Equal(
		t,
		sortedResults,
		[]apply.Result{
			applyResults[5],
			applyResults[3],
			applyResults[4],
			applyResults[2],
			applyResults[0],
			applyResults[1],
		},
	)
}
