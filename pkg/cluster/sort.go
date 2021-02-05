package cluster

import (
	"sort"
	"strconv"
	"strings"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
)

type wrappedResult struct {
	nameBase       string
	nameIndex      int
	namespaceBase  string
	namespaceIndex int
	kind           string
	resultIndex    int
}

// sortedDiffResults sorts diff results by namespace, then resource type,
// then resource name. If a resource name or namespace is structured as
// [base string]-[number], then the number is used to break ties among all
// entities with the same base.
func sortedDiffResults(results []diff.Result) []diff.Result {
	wrappedResults := make([]wrappedResult, len(results))
	for r, result := range results {
		if result.Object != nil {
			wrappedResults[r].nameBase, wrappedResults[r].nameIndex =
				parseName(result.Object.Name)
			wrappedResults[r].namespaceBase, wrappedResults[r].namespaceIndex =
				parseName(result.Object.Namespace)
			wrappedResults[r].kind = result.Object.Kind
		} else {
			wrappedResults[r].namespaceBase = result.Name
		}
		wrappedResults[r].resultIndex = r
	}

	sortWrappedResults(wrappedResults)

	sortedResults := make([]diff.Result, len(results))
	for r, wrappedResult := range wrappedResults {
		sortedResults[r] = results[wrappedResult.resultIndex]
	}

	return sortedResults
}

// sortedApplyResults sorts apply results by namespace, then resource type,
// then resource name.  If a resource name or namespace is structured as
// [base string]-[number], then the number is used to break ties among all
// entities with the same base.
func sortedApplyResults(results []apply.Result) []apply.Result {
	wrappedResults := make([]wrappedResult, len(results))
	for r, result := range results {
		wrappedResults[r].nameBase, wrappedResults[r].nameIndex =
			parseName(result.Name)
		wrappedResults[r].namespaceBase, wrappedResults[r].namespaceIndex =
			parseName(result.Namespace)
		wrappedResults[r].kind = result.Kind
		wrappedResults[r].resultIndex = r
	}

	sortWrappedResults(wrappedResults)

	sortedResults := make([]apply.Result, len(results))
	for r, wrappedResult := range wrappedResults {
		sortedResults[r] = results[wrappedResult.resultIndex]
	}

	return sortedResults
}

func sortWrappedResults(wrappedResults []wrappedResult) {
	sort.Slice(wrappedResults, func(a, b int) bool {
		result1 := wrappedResults[a]
		result2 := wrappedResults[b]
		if result1.namespaceBase < result2.namespaceBase {
			return true
		} else if result1.namespaceBase > result2.namespaceBase {
			return false
		} else if result1.namespaceIndex < result2.namespaceIndex {
			return true
		} else if result1.namespaceIndex > result2.namespaceIndex {
			return false
		} else if result1.kind < result2.kind {
			return true
		} else if result1.kind > result2.kind {
			return false
		} else if result1.nameBase < result2.nameBase {
			return true
		} else if result1.nameBase > result2.nameBase {
			return false
		} else if result1.nameIndex < result2.nameIndex {
			return true
		}
		return false
	})
}

func parseName(name string) (string, int) {
	components := strings.Split(name, "-")
	if len(components) < 2 {
		return name, 0
	}
	index, err := strconv.Atoi(components[len(components)-1])
	if err != nil {
		return name, 0
	}
	return strings.Join(components[0:len(components)-1], "-"), index
}
