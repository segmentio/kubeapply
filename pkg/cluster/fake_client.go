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

// NewFakeClusterClient returns a FakeClusterClient that works without errors.
func NewFakeClusterClient(
	ctx context.Context,
	config *ClusterClientConfig,
) (ClusterClient, error) {
	return &FakeClusterClient{
		clusterConfig: config.ClusterConfig,
		store:         map[string]string{},
	}, nil
}

// NewFakeClusterClientError returns a FakeClusterClient that simulates an error when
// running kubectl.
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

// Apply runs a fake apply using the configs in the argument path.
func (cc *FakeClusterClient) Apply(
	ctx context.Context,
	path string,
	serverSide bool,
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

// ApplyStructured runs a fake structured apply using the configs in the argument
// path.
func (cc *FakeClusterClient) ApplyStructured(
	ctx context.Context,
	path string,
	serverSide bool,
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

// Diff runs a fake diff using the configs in the argument path.
func (cc *FakeClusterClient) Diff(
	ctx context.Context,
	path string,
	serverSide bool,
) ([]byte, error) {
	return []byte(
			fmt.Sprintf(
				"diff result for %s with path %s",
				cc.clusterConfig.Cluster,
				path,
			),
		),
		cc.kubectlErr
}

// Summary creates a fake summary output of the current cluster state.
func (cc *FakeClusterClient) Summary(ctx context.Context) (string, error) {
	return fmt.Sprintf("summary %s", cc.clusterConfig.Cluster), cc.kubectlErr
}

// GetStoreValue gets the value of the argument key.
func (cc *FakeClusterClient) GetStoreValue(key string) (string, error) {
	return cc.store[key], nil
}

// SetStoreValue sets the argument key to the argument value.
func (cc *FakeClusterClient) SetStoreValue(key string, value string) error {
	cc.store[key] = value
	return nil
}

// Config returns this client's cluster config.
func (cc *FakeClusterClient) Config() *config.ClusterConfig {
	return cc.clusterConfig
}

// GetNamespaceUID returns the kubernetes identifier for a given namespace in this cluster.
func (cc *FakeClusterClient) GetNamespaceUID(ctx context.Context, namespace string) (string, error) {
	return fmt.Sprintf("ns-%s", namespace), cc.kubectlErr
}

// Close closes the client.
func (cc *FakeClusterClient) Close() error {
	return nil
}
