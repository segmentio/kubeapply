package provider

import (
	"context"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestResource(t *testing.T) {
	ctx := context.Background()
	config := config.ClusterConfig{}
	clusterClient, err := cluster.NewFakeClusterClient(
		ctx,
		&cluster.ClusterClientConfig{
			ClusterConfig: &config,
		},
	)
	require.NoError(t, err)

	providerCtx := &providerContext{
		config:        config,
		clusterClient: clusterClient,
	}

	resource.Test(t, resource.TestCase{
		IsUnitTest: true,
		Providers: map[string]*schema.Provider{
			"kubeapply": Provider(providerCtx),
		},
		Steps: []resource.TestStep{
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
    version = "v1.9.5"
  }
}
				`,
				ExpectError: regexp.MustCompile("error"),
				//ExpectNonEmptyPlan: true,
			},
		},
	})
}
