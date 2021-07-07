package cluster

import (
	"context"
	"time"

	"github.com/briandowns/spinner"
	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
	"github.com/segmentio/kubeapply/pkg/config"
)

const (
	lockAcquistionTimeout = 30 * time.Second
)

// ClusterClient is an interface that interacts with the API of a single Kubernetes cluster.
type ClusterClient interface {
	// Apply applies all of the configs at the given path.
	Apply(ctx context.Context, paths []string, serverSide bool) ([]byte, error)

	// ApplyStructured applies all of the configs at the given path and returns structured,
	// as opposed to raw, outputs
	ApplyStructured(ctx context.Context, paths []string, serverSide bool) ([]apply.Result, error)

	// Diff gets the diffs between the configs at the given path and the actual state of resources
	// in the cluster.
	Diff(ctx context.Context, paths []string, serverSide bool) ([]byte, error)

	// Diff gets the diffs between the configs at the given path and the actual state of resources
	// in the cluster.
	//
	// The diffCommand argument can be set to use a custom diff command in place of the default
	// (kubeapply kdiff).
	DiffStructured(
		ctx context.Context,
		paths []string,
		serverSide bool,
		diffCommand string,
	) ([]diff.Result, error)

	// Summary returns a summary of all workloads in the cluster.
	Summary(ctx context.Context) (string, error)

	// GetStoreValue gets the value of the given key.
	GetStoreValue(ctx context.Context, key string) (string, error)

	// SetStoreValue sets the given key/value pair in the cluster.
	SetStoreValue(ctx context.Context, key string, value string) error

	// Config returns the config for this cluster.
	Config() *config.ClusterConfig

	// GetNamespaceUID returns the kubernetes identifier for a given namespace in this cluster.
	GetNamespaceUID(ctx context.Context, namespace string) (string, error)

	// Close cleans up this client.
	Close() error
}

// ClusterClientConfig stores the configuration necessary to create a ClusterClient.
type ClusterClientConfig struct {
	// CheckApplyConsistency indicates whether we should check whether an apply is done with
	// the same SHA as the last diff in the cluster.
	CheckApplyConsistency bool

	// ClusterConfig is the config for the cluster that we are communicating with.
	ClusterConfig *config.ClusterConfig

	// Debug indicates whether commands should be run with debug-level logging.
	Debug bool

	// KeepConfigs indicates whether kube client should keep around intermediate
	// yaml manifests. These are useful for debugging when there are apply errors.
	KeepConfigs bool

	// HeadSHA is the SHA of the current branch. Used for consistency checking, can be omitted
	// if that option is false.
	HeadSHA string

	// UseColors indicates whether output should include colors. Currently only applies to diff
	// operations.
	UseColors bool

	// UseLocks indicates whether a cluster-specific lock should be acquired before any diff
	// and expand operations.
	UseLocks bool

	// SpinnerObj is a pointer to a Spinner instance. If unset, no spinner is used. Currently
	// only applies to diff operations.
	SpinnerObj *spinner.Spinner

	// StreamingOutput indicates whether results should be streamed out to stdout and stderr.
	// Currently only applies to apply operations.
	StreamingOutput bool
}

// ClusterClientGenerator generates a ClusterClient from a config.
type ClusterClientGenerator func(
	ctx context.Context,
	config *ClusterClientConfig,
) (ClusterClient, error)
