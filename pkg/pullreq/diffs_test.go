package pullreq

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/google/go-github/v30/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCoveredClusters(t *testing.T) {
	type clusterTestCase struct {
		diffs                   []*github.CommitFile
		selectedClusterGlobStrs []string
		subpathOverride         string
		multiSubpaths           bool
		expectedClustersIDs     []string
		expectedSubpaths        []string
	}

	testCases := []clusterTestCase{
		{
			diffs:               []*github.CommitFile{},
			expectedClustersIDs: []string{},
			expectedSubpaths:    []string{},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/cluster1.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/profile/file1.yaml"),
				},
			},
			expectedClustersIDs: []string{},
			expectedSubpaths:    []string{},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/cluster1.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/profile/file1.yaml"),
				},
			},
			expectedClustersIDs: []string{},
			expectedSubpaths:    []string{},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/file1.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/file2.yaml"),
				},
				{
					Filename: aws.String("other/expanded/file1.yaml"),
				},
			},
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster1",
			},
			expectedSubpaths: []string{
				".",
			},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/file3.yaml",
					),
				},
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/subdir3/file6.yaml",
					),
				},
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir2/file4.yaml",
					),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster2/file2.yaml"),
				},
			},
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster1",
			},
			expectedSubpaths: []string{
				"subdir1",
				"subdir2",
			},
			multiSubpaths: true,
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/file1.yaml"),
				},
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/file3.yaml",
					),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster2/file2.yaml"),
				},
			},
			selectedClusterGlobStrs: []string{
				"stage:us-west-2:cluster1",
				"production:us-west-2:cluster2",
				"stage:us-west-2:cluster3",
			},
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster1",
				"stage:us-west-2:cluster3",
			},
			expectedSubpaths: []string{
				".",
				".",
			},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/file1.yaml"),
				},
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/file3.yaml",
					),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster2/file2.yaml"),
				},
			},
			selectedClusterGlobStrs: []string{
				"stage:*1",
			},
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster1",
			},
			expectedSubpaths: []string{
				".",
			},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/file3.yaml",
					),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster3/file1.yaml"),
				},
			},
			selectedClusterGlobStrs: []string{
				"stage:*",
			},
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster1",
				"stage:us-west-2:cluster3",
			},
			expectedSubpaths: []string{
				"subdir1",
				".",
			},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/file3.yaml",
					),
				},
			},
			selectedClusterGlobStrs: []string{
				"stage:*",
			},
			// Cluster 3 is pruned from selection list because it doesn't have any files in the
			// git diff.
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster1",
			},
			expectedSubpaths: []string{
				"subdir1",
			},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/file1.yaml"),
				},
				{
					Filename: aws.String(
						"clusters/clustertype/expanded/cluster1/subdir1/file3.yaml",
					),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster2/file2.yaml"),
				},
			},
			selectedClusterGlobStrs: []string{
				"stage:us-west-2:cluster3",
			},
			expectedClustersIDs: []string{
				"stage:us-west-2:cluster3",
			},
			subpathOverride: "subdir1",
			expectedSubpaths: []string{
				"subdir1",
			},
		},
	}

	for index, testCase := range testCases {
		coveredClusters, err := GetCoveredClusters(
			"testdata/repo",
			testCase.diffs,
			"stage",
			testCase.selectedClusterGlobStrs,
			testCase.subpathOverride,
			testCase.multiSubpaths,
		)
		assert.Nil(t, err, "Test case %d", index)

		coveredClusterIDs := []string{}
		subpaths := []string{}

		for _, coveredCluster := range coveredClusters {
			coveredClusterIDs = append(coveredClusterIDs, coveredCluster.DescriptiveName())
			subpaths = append(subpaths, coveredCluster.Subpaths...)
		}

		assert.Equal(t, testCase.expectedClustersIDs, coveredClusterIDs, "Test case %d", index)
		assert.Equal(t, testCase.expectedSubpaths, subpaths, "Test case %d", index)
	}
}

func TestLowestParent(t *testing.T) {
	type parentTestCase struct {
		root      string
		paths     []string
		expResult string
		expErr    bool
	}

	testCases := []parentTestCase{
		{
			root:      "root/cluster/expanded",
			paths:     []string{},
			expResult: ".",
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace/file1.txt",
				"root/cluster/expanded/namespace/file2.txt",
			},
			expResult: "namespace",
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace/file1.txt",
				"root/cluster/expanded/namespace/child/file2.txt",
			},
			expResult: "namespace",
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace/child1/child2/file1.txt",
				"root/cluster/expanded/namespace/child1/child2/file2.txt",
				"root/cluster/expanded/namespace/child1/child2/file3.txt",
			},
			expResult: "namespace/child1/child2",
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace1/file1.txt",
				"root/cluster/expanded/namespace2/child/file2.txt",
				"root/cluster/expanded/namespace2/child/file3.txt",
			},
			expResult: ".",
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/file1.txt",
				"root/cluster/expanded/namespace2/child/file2.txt",
				"root/cluster/expanded/namespace2/child/file3.txt",
			},
			expResult: ".",
		},
	}

	for index, testCase := range testCases {
		result, err := lowestParent(testCase.root, testCase.paths)
		if testCase.expErr {
			assert.NotNil(t, err, "test case %d", index)
		} else {
			require.Nil(t, err, "test case %d", index)
			assert.Equal(t, testCase.expResult, result, "test case %d", index)
		}
	}
}

func TestLowestParents(t *testing.T) {
	type parentTestCase struct {
		root      string
		paths     []string
		expResult []string
		expErr    bool
	}

	testCases := []parentTestCase{
		{
			root:      "root/cluster/expanded",
			paths:     []string{},
			expResult: []string{"."},
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace/file1.txt",
				"root/cluster/expanded/namespace/file2.txt",
			},
			expResult: []string{"namespace"},
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace/file1.txt",
				"root/cluster/expanded/namespace/child/file2.txt",
			},
			expResult: []string{"namespace"},
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace/child1/child2/file1.txt",
				"root/cluster/expanded/namespace/child1/child2/file2.txt",
				"root/cluster/expanded/namespace/child1/child2/file3.txt",
			},
			expResult: []string{"namespace/child1/child2"},
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/expanded/namespace1/child1/child1b/file1.txt",
				"root/cluster/expanded/namespace2/child2/file2.txt",
				"root/cluster/expanded/namespace2/child2/file3.txt",
				"root/cluster/expanded/namespace2/child2/child2b/file4.txt",
				"root/cluster/expanded/namespace3/child3/file5.txt",
			},
			expResult: []string{
				"namespace1/child1/child1b",
				"namespace2/child2",
				"namespace3/child3",
			},
		},
		{
			root: "root/cluster/expanded",
			paths: []string{
				"root/cluster/file1.txt",
				"root/cluster/expanded/namespace2/child/file2.txt",
				"root/cluster/expanded/namespace2/child/file3.txt",
			},
			expResult: []string{"."},
		},
	}

	for index, testCase := range testCases {
		result, err := lowestParents(testCase.root, testCase.paths)
		if testCase.expErr {
			assert.NotNil(t, err, "test case %d", index)
		} else {
			require.Nil(t, err, "test case %d", index)
			assert.Equal(
				t,
				testCase.expResult,
				result,
				"test case %d",
				index,
			)
		}
	}
}
