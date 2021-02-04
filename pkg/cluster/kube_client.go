package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/briandowns/spinner"
	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/diff"
	"github.com/segmentio/kubeapply/pkg/cluster/kube"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/store"
	log "github.com/sirupsen/logrus"
)

var _ ClusterClient = (*KubeClusterClient)(nil)

// KubeClusterClient is an implementation of a ClusterClient that hits an actual Kubernetes API.
// It's backed by a kube.OrderedClient which, in turn, wraps kubectl.
type KubeClusterClient struct {
	clusterConfig *config.ClusterConfig

	headSHA               string
	clusterKey            string
	lockID                string
	useLocks              bool
	checkApplyConsistency bool
	spinnerObj            *spinner.Spinner
	streamingOutput       bool

	tempDir        string
	kubeConfigPath string
	kubeClient     *kube.OrderedClient
	kubeLocker     store.Locker
	kubeStore      store.Store
}

// kubeapplyDiffEvent is used for storing the last successful diff in the kubeStore.
// This value is checked before applying to ensure that the SHAs match.
type kubeapplyDiffEvent struct {
	SHA string `json:"sha"`

	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

// NewKubeClusterClient creates a new ClusterClient instance for a real
// Kubernetes cluster.
func NewKubeClusterClient(
	ctx context.Context,
	config *ClusterClientConfig,
) (ClusterClient, error) {
	clusterKey := fmt.Sprintf(
		"%s__%s__%s",
		config.ClusterConfig.Cluster,
		config.ClusterConfig.Region,
		config.ClusterConfig.Env,
	)

	var err error
	var tempDir string
	var kubeConfigPath string

	if config.ClusterConfig.KubeConfigPath != "" {
		kubeConfigPath = config.ClusterConfig.KubeConfigPath
	} else {
		// Generate a kubeconfig via the EKS API.
		tempDir, err = ioutil.TempDir("", "kubeconfigs")
		if err != nil {
			return nil, err
		}

		kubeConfigPath = filepath.Join(
			tempDir,
			fmt.Sprintf(
				"kubeconfig_%s_%s.yaml",
				config.ClusterConfig.Cluster,
				config.ClusterConfig.Region,
			),
		)

		sess := session.Must(session.NewSession())
		err = kube.CreateKubeconfigViaAPI(
			ctx,
			sess,
			config.ClusterConfig.Cluster,
			config.ClusterConfig.Region,
			kubeConfigPath,
		)
		if err != nil {
			return nil, err
		}
	}

	kubeClient := kube.NewOrderedClient(
		kubeConfigPath,
		config.KeepConfigs,
		nil,
		config.Debug,
		config.ClusterConfig.ServerSideApply,
	)

	kubeStore, err := store.NewKubeStore(
		kubeConfigPath,
		"kubeapply-store",
		"kube-system",
	)
	if err != nil {
		return nil, err
	}

	hostName, err := os.Hostname()
	if err != nil {
		log.Warnf("Error getting hostname, using generic string: %s", hostName)
		hostName = "kubeapply"
	}

	// Ensure that lock ID is identifiable and unique
	lockID := fmt.Sprintf("%s-%d", hostName, time.Now().UnixNano()/int64(1000))

	kubeLocker, err := store.NewKubeLocker(
		kubeConfigPath,
		lockID,
		"kube-system",
	)
	if err != nil {
		return nil, err
	}

	return &KubeClusterClient{
		clusterConfig:         config.ClusterConfig,
		headSHA:               config.HeadSHA,
		useLocks:              config.UseLocks,
		checkApplyConsistency: config.CheckApplyConsistency,
		spinnerObj:            config.SpinnerObj,
		streamingOutput:       config.StreamingOutput,
		clusterKey:            clusterKey,
		lockID:                lockID,
		tempDir:               tempDir,
		kubeConfigPath:        kubeConfigPath,
		kubeClient:            kubeClient,
		kubeStore:             kubeStore,
		kubeLocker:            kubeLocker,
	}, nil
}

// Apply does a kubectl apply for the resources at the argument path.
func (cc *KubeClusterClient) Apply(
	ctx context.Context,
	paths []string,
	serverSide bool,
) ([]byte, error) {
	return cc.execApply(ctx, paths, "", false)
}

// ApplyStructured does a structured kubectl apply for the resources at the
// argument path.
func (cc *KubeClusterClient) ApplyStructured(
	ctx context.Context,
	paths []string,
	serverSide bool,
) ([]apply.Result, error) {
	oldContents, err := cc.execApply(ctx, paths, "json", true)
	if err != nil {
		return nil,
			fmt.Errorf(
				"Error running apply dry-run: %+v; output: %s",
				err,
				string(oldContents),
			)
	}

	oldObjs, err := apply.KubeJSONToObjects(oldContents)
	if err != nil {
		return nil, err
	}

	newContents, err := cc.execApply(ctx, paths, "json", false)
	if err != nil {
		return nil,
			fmt.Errorf(
				"Error running apply: %+v; output: %s",
				err,
				string(newContents),
			)
	}
	newObjs, err := apply.KubeJSONToObjects(newContents)
	if err != nil {
		return nil, err
	}

	results, err := apply.ObjsToResults(oldObjs, newObjs)
	if err != nil {
		return nil, err
	}
	return sortedApplyResults(results), nil
}

// Diff runs a kubectl diff between the configs at the argument path and the associated
// resources in the cluster. It returns raw output that can be immediately printed to the
// console.
func (cc *KubeClusterClient) Diff(
	ctx context.Context,
	paths []string,
	serverSide bool,
) ([]byte, error) {
	rawResults, err := cc.execDiff(ctx, paths, false)
	if err != nil {
		return nil, fmt.Errorf(
			"Error running diff: %+v (output: %s)",
			err,
			string(rawResults),
		)
	}

	return rawResults, nil
}

// DiffStructured runs a kubectl diff between the configs at the argument path and the associated
// resources in the cluster. It returns a structured result that can be used printed to a table.
func (cc *KubeClusterClient) DiffStructured(
	ctx context.Context,
	paths []string,
	serverSide bool,
) ([]diff.Result, error) {
	rawResults, err := cc.execDiff(ctx, paths, true)
	if err != nil {
		return nil, fmt.Errorf(
			"Error running diff: %+v (output: %s)",
			err,
			string(rawResults),
		)
	}

	results := diff.Results{}
	if err := json.Unmarshal(rawResults, &results); err != nil {
		return nil, err
	}
	return sortedDiffResults(results.Results), nil
}

// Summary returns a summary of the current cluster state.
func (cc *KubeClusterClient) Summary(ctx context.Context) (string, error) {
	return cc.kubeClient.Summary(ctx)
}

// GetStoreValue gets the value of the argument key.
func (cc *KubeClusterClient) GetStoreValue(key string) (string, error) {
	return cc.kubeStore.Get(key)
}

// SetStoreValue sets the value of the argument key to the argument value.
func (cc *KubeClusterClient) SetStoreValue(key string, value string) error {
	return cc.kubeStore.Set(key, value)
}

// Config returns this client's cluster config.
func (cc *KubeClusterClient) Config() *config.ClusterConfig {
	return cc.clusterConfig
}

// GetNamespaceUID returns the kubernetes identifier for a given namespace in this cluster.
func (cc *KubeClusterClient) GetNamespaceUID(
	ctx context.Context,
	namespace string,
) (string, error) {
	return cc.kubeClient.GetNamespaceUID(ctx, namespace)
}

// Close closes the client and cleans up all of the associated resources.
func (cc *KubeClusterClient) Close() error {
	if cc.tempDir != "" {
		return os.RemoveAll(cc.tempDir)
	}
	return nil
}

func (cc *KubeClusterClient) execApply(
	ctx context.Context,
	paths []string,
	format string,
	dryRun bool,
) ([]byte, error) {
	if cc.useLocks {
		acquireCtx, cancel := context.WithTimeout(ctx, lockAcquistionTimeout)
		defer cancel()

		err := cc.kubeLocker.Acquire(acquireCtx, cc.clusterConfig.Cluster)
		if err != nil {
			return nil, fmt.Errorf("Error acquiring lock: %+v. Try again later.", err)
		}
		defer func() {
			err := cc.kubeLocker.Release(cc.clusterConfig.Cluster)
			if err != nil {
				log.Warnf(
					"Error releasing lock for %s: %+v",
					cc.clusterConfig.Cluster,
					err,
				)
			}
		}()
	} else {
		log.Debug("Skipping over locking")
	}

	if cc.checkApplyConsistency {
		log.Infof("Fetching diff event for key %s", cc.clusterKey)
		storeValue, err := cc.GetStoreValue(cc.clusterKey)
		if err != nil {
			return nil, err
		}
		diffEvent := kubeapplyDiffEvent{}
		if err := json.Unmarshal([]byte(storeValue), &diffEvent); err != nil {
			return nil, err
		}

		if diffEvent.SHA != cc.headSHA {
			return nil, fmt.Errorf(
				"Last diff was applied at a different SHA (%s).\nPlease run kubeapply diff again.",
				diffEvent.SHA,
			)
		}
	} else {
		log.Debug("Skipping over apply consistency check")
	}

	return cc.kubeClient.Apply(
		ctx,
		paths,
		!cc.streamingOutput,
		format,
		dryRun,
	)
}

func (cc *KubeClusterClient) execDiff(
	ctx context.Context,
	paths []string,
	structured bool,
) ([]byte, error) {
	if cc.useLocks {
		acquireCtx, cancel := context.WithTimeout(ctx, lockAcquistionTimeout)
		defer cancel()

		err := cc.kubeLocker.Acquire(acquireCtx, cc.clusterConfig.Cluster)
		if err != nil {
			return nil, fmt.Errorf("Error acquiring lock: %+v. Try again later.", err)
		}
		defer func() {
			err := cc.kubeLocker.Release(cc.clusterConfig.Cluster)
			if err != nil {
				log.Warnf(
					"Error releasing lock for %s: %+v",
					cc.clusterConfig.Cluster,
					err,
				)
			}
		}()
	} else {
		log.Debug("Skipping over locking")
	}

	diffResult, err := cc.kubeClient.Diff(
		ctx,
		paths,
		structured,
		cc.spinnerObj,
	)
	if err != nil || !cc.checkApplyConsistency {
		return diffResult, err
	}

	diffEvent := kubeapplyDiffEvent{
		SHA:       cc.headSHA,
		UpdatedAt: time.Now(),
		UpdatedBy: cc.lockID,
	}
	diffEventBytes, err := json.Marshal(diffEvent)
	if err != nil {
		return diffResult, err
	}
	diffEventStr := string(diffEventBytes)

	log.Infof("Setting store key value: %s, %s", cc.clusterKey, diffEventStr)
	return diffResult, cc.kubeStore.Set(cc.clusterKey, diffEventStr)
}
