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
		diffs               []*github.CommitFile
		selectedClusterIDs  []string
		subpathOverride     string
		expectedClustersIDs []string
		expectedSubpaths    []string
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
					Filename: aws.String("clusters/clustertype/expanded/cluster1/subdir1/file3.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/subdir1/subdir3/file6.yaml"),
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
			},
		},
		{
			diffs: []*github.CommitFile{
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/file1.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster1/subdir1/file3.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster2/file2.yaml"),
				},
			},
			selectedClusterIDs: []string{
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
					Filename: aws.String("clusters/clustertype/expanded/cluster1/subdir1/file3.yaml"),
				},
				{
					Filename: aws.String("clusters/clustertype/expanded/cluster2/file2.yaml"),
				},
			},
			selectedClusterIDs: []string{
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
			testCase.selectedClusterIDs,
			testCase.subpathOverride,
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
