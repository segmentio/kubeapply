package cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/config"
)

var _ ClusterClient = (*FakeClusterClient)(nil)

// FakeClusterClient is a fake implementation of a ClusterClient. For testing purposes only.
type FakeClusterClient struct {
	clusterConfig   *config.ClusterConfig
	subpathOverride string
	store           map[string]string
	kubectlErr      error
}

func NewFakeClusterClient(
	ctx context.Context,
	config *ClusterClientConfig,
) (ClusterClient, error) {
	return &FakeClusterClient{
		clusterConfig: config.ClusterConfig,
		store:         map[string]string{},
	}, nil
}

func NewFakeClusterClientError(
	ctx context.Context,
	config *ClusterClientConfig,
) (ClusterClient, error) {
	return &FakeClusterClient{
		clusterConfig: config.ClusterConfig,
		store:         map[string]string{},
		kubectlErr:    errors.New("kubectl error!"),
	}, nil
}

func (cc *FakeClusterClient) Apply(
	ctx context.Context,
	path string,
) ([]byte, error) {
	return []byte(
			fmt.Sprintf(
				"apply result for %s with path %s",
				cc.clusterConfig.Cluster,
				path,
			),
		),
		cc.kubectlErr
}

func (cc *FakeClusterClient) ApplyStructured(
	ctx context.Context,
	path string,
) ([]apply.Result, error) {
	return []apply.Result{
		{
			Kind: "Deployment",
			Name: fmt.Sprintf(
				"apply result for %s with path %s",
				cc.clusterConfig.Cluster,
				path,
			),
			Namespace:  "test-namespace",
			OldVersion: "1234",
			NewVersion: "5678",
		},
	}, cc.kubectlErr
}

func (cc *FakeClusterClient) Diff(ctx context.Context, path string) ([]byte, error) {
	return []byte(
			fmt.Sprintf(
				"diff result for %s with path %s",
				cc.clusterConfig.Cluster,
				path,
			),
		),
		cc.kubectlErr
}

func (cc *FakeClusterClient) Summary(ctx context.Context) (string, error) {
	return fmt.Sprintf("summary %s", cc.clusterConfig.Cluster), cc.kubectlErr
}

func (cc *FakeClusterClient) GetStoreValue(key string) (string, error) {
	return cc.store[key], nil
}

func (cc *FakeClusterClient) SetStoreValue(key string, value string) error {
	cc.store[key] = value
	return nil
}

func (cc *FakeClusterClient) Config() *config.ClusterConfig {
	return cc.clusterConfig
}

func (cc *FakeClusterClient) Close() error {
	return nil
}
