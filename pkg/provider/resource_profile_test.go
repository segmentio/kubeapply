package provider

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestResourceProfile(t *testing.T) {
	ctx := context.Background()
	tempDir, err := ioutil.TempDir("", "profile")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	config := config.ClusterConfig{
		Cluster: "testCluster",
		Region:  "testRegion",
		Env:     "testEnv",
	}
	clusterClient, err := cluster.NewFakeClusterClient(
		ctx,
		&cluster.ClusterClientConfig{
			ClusterConfig: &config,
		},
	)
	require.NoError(t, err)

	rawClient := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "testNamespace",
			},
		},
	)

	providerCtx := &providerContext{
		autoCreateNamespaces: true,
		clusterClient:        clusterClient,
		config:               config,
		keepExpanded:         true,
		rawClient:            rawClient,
		tempDir:              tempDir,
	}

	resource.Test(
		t,
		resource.TestCase{
			IsUnitTest: true,
			Providers: map[string]*schema.Provider{
				"kubeapply": Provider(providerCtx),
			},
			Steps: []resource.TestStep{
				// First, do create
				{
					Config: `
provider "kubeapply" {
  cluster_name = "testCluster"
  region = "testRegion"
  account_name = "testAccountName"
  host = "testHost"
  cluster_ca_certificate = "testCACertificate"
  token = "testToken"
}

resource "kubeapply_profile" "main_profile" {
  path = "testdata"

  parameters = {
    value1 = "Value1"
	value2 = "Value2"
  }
}`,
					Check: func(state *terraform.State) error {
						require.Equal(t, 1, len(state.Modules))
						module := state.Modules[0]
						resource := module.Resources["kubeapply_profile.main_profile"]
						require.NotNil(t, resource)
						return nil
					},
				},
				// Then, do update that actually makes changes
				{
					Config: `
provider "kubeapply" {
  cluster_name = "testCluster"
  region = "testRegion"
  account_name = "testAccountName"
  host = "testHost"
  cluster_ca_certificate = "testCACertificate"
  token = "testToken"
}

resource "kubeapply_profile" "main_profile" {
  path = "testdata"

  parameters = {
    value1 = "UpdatedValue1"
	value2 = "UpdatedValue2"
  }
}`,
				},
				// Then, finally, update that doesn't make any changes
				{
					Config: `
provider "kubeapply" {
  cluster_name = "testCluster"
  region = "testRegion"
  account_name = "testAccountName"
  host = "testHost"
  cluster_ca_certificate = "testCACertificate"
  token = "testToken"
}

resource "kubeapply_profile" "main_profile" {
  path = "testdata"

  parameters = {
    value1 = "UpdatedValue1"
	value2 = "UpdatedValue2"
  }
}`,
				},
				// Test some error cases
				{
					Config: `
provider "kubeapply" {
  host = "testHost"
  cluster_ca_certificate = "testCACertificate"
  token = "testToken"
}

resource "kubeapply_profile" "main_profile" {
  path = "testdata"

  parameters = {
    value1 = "UpdatedValue1"
	value2 = "UpdatedValue2"
  }
}`,
					ExpectError: regexp.MustCompile("is required, but no definition was found"),
				},
				{
					Config: `
provider "kubeapply" {
  cluster_name = "testCluster"
  region = "testRegion"
  account_name = "testAccountName"
  host = "testHost"
  cluster_ca_certificate = "testCACertificate"
  token = "testToken"
}

resource "kubeapply_profile" "main_profile" {
  path = "bad dir"

  parameters = {
    value1 = "UpdatedValue1"
	value2 = "UpdatedValue2"
  }
}`,
					ExpectError: regexp.MustCompile("stat bad dir: no such file or directory"),
				},
			},
		},
	)

	expandedRoot := filepath.Join(tempDir, "expanded")
	subDirs, err := ioutil.ReadDir(expandedRoot)
	require.NoError(t, err)
	require.Greater(t, len(subDirs), 0)

	// There are lots of expansions, just look at the last one
	lastSubDir := filepath.Join(
		expandedRoot,
		subDirs[len(subDirs)-1].Name(),
	)
	lastSubDirContents := util.GetContents(t, lastSubDir)
	assert.Equal(
		t,
		map[string][]string{
			"deployment.yaml": {
				"apiVersion: apps/v1",
				"kind: Deployment", "metadata:",
				"  labels:",
				"    key1: UpdatedValue1",
				"    cluster: testCluster",
				"  name: testName",
				"  namespace: testNamespace",
				"",
			},
			"service.yaml": {
				"apiVersion: v1",
				"kind: Service", "metadata:",
				"  labels:",
				"    key2: UpdatedValue2",
				"    env: testEnv",
				"  name: testName",
				"  namespace: testNamespace2",
				"",
			},
		},
		lastSubDirContents,
	)

	calls := clusterClient.(*cluster.FakeClusterClient).Calls
	callTypes := []string{}
	for _, call := range calls {
		callTypes = append(callTypes, call.CallType)
	}
	assert.Equal(
		t,
		[]string{
			// Initial create
			"DiffStructured",
			"Apply",
			// Update
			"DiffStructured",
			"Apply",
		},
		callTypes,
	)

	namespaces, err := rawClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	namespaceNames := []string{}
	for _, namespace := range namespaces.Items {
		namespaceNames = append(namespaceNames, namespace.Name)
	}
	assert.ElementsMatch(
		t,
		[]string{
			"testNamespace",
			// Extra namespace was created
			"testNamespace2",
		},
		namespaceNames,
	)
}
