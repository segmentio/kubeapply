package subcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/cluster/kube"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	spinnerCharSet  = 32
	spinnerDuration = 200 * time.Millisecond
)

var diffCmd = &cobra.Command{
	Use:   "diff [cluster configs]",
	Short: "diff shows the difference between the local configs and the API state",
	Args:  cobra.MinimumNArgs(1),
	RunE:  diffRun,
}

type diffFlags struct {
	// Expand before running diff.
	expand bool

	// Path to kubeconfig. If unset, tries to fetch from the environment.
	kubeConfig string

	// Run operatation in just one subdirectory of the expanded configs
	// (typically maps to namespace). If unset, considers all configs.
	subpath string
}

var diffFlagValues diffFlags

func init() {
	diffCmd.Flags().BoolVar(
		&diffFlagValues.expand,
		"expand",
		false,
		"Expand before running diff",
	)
	diffCmd.Flags().StringVar(
		&diffFlagValues.kubeConfig,
		"kubeconfig",
		"",
		"Path to kubeconfig",
	)
	diffCmd.Flags().StringVar(
		&diffFlagValues.subpath,
		"subpath",
		"",
		"Diff for expanded configs in the provided subpath only",
	)

	RootCmd.AddCommand(diffCmd)
}

func diffRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	for _, arg := range args {
		if err := diffClusterPath(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

func diffClusterPath(ctx context.Context, path string) error {
	clusterConfig, err := config.LoadClusterConfig(path, "")
	if err != nil {
		return err
	}
	if err := clusterConfig.CheckVersion(version.Version); err != nil {
		return err
	}

	if diffFlagValues.expand {
		if err := expandCluster(ctx, clusterConfig); err != nil {
			return err
		}
	}

	log.Infof("Diffing cluster %s", clusterConfig.DescriptiveName())

	ok, err := util.DirExists(clusterConfig.ExpandedPath)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf(
			"Expanded path %s does not exist",
			clusterConfig.ExpandedPath,
		)
	}

	kubeConfig := diffFlagValues.kubeConfig

	if kubeConfig == "" {
		kubeConfig = os.Getenv("KUBECONFIG")
		if kubeConfig == "" {
			return errors.New("Must either set --kubeconfig flag or KUBECONFIG env variable")
		}
	}

	matches := kube.KubeconfigMatchesCluster(kubeConfig, clusterConfig.Cluster)
	if !matches {
		return fmt.Errorf(
			"Kubeconfig in %s does not appear to reference cluster %s",
			kubeConfig,
			clusterConfig.Cluster,
		)
	}

	clusterConfig.KubeConfigPath = kubeConfig
	clusterConfig.Subpath = diffFlagValues.subpath

	diffResult, err := execDiff(ctx, clusterConfig)
	if err != nil {
		log.Errorf("Error running diff: %s, %+v", diffResult, err)
		log.Info(
			"Try re-running with --debug to see verbose output. Note that diffs will not work if target namespace(s) don't exist yet.",
		)
		return err
	}
	printDiff(diffResult)

	return nil
}

func execDiff(
	ctx context.Context,
	clusterConfig *config.ClusterConfig,
) (string, error) {
	log.Info("Generating diff against versions in Kube API")

	spinnerObj := spinner.New(
		spinner.CharSets[spinnerCharSet],
		spinnerDuration,
		spinner.WithWriter(os.Stderr),
		spinner.WithHiddenCursor(true),
	)
	spinnerObj.Prefix = "Running: "

	kubeClient, err := cluster.NewKubeClusterClient(
		ctx,
		&cluster.ClusterClientConfig{
			CheckApplyConsistency: false,
			ClusterConfig:         clusterConfig,
			Debug:                 debug,
			SpinnerObj:            spinnerObj,
			UseColors:             true,
			// TODO: Make locking an option
			UseLocks: false,
		},
	)
	if err != nil {
		return "", err
	}
	defer kubeClient.Close()

	results, err := kubeClient.Diff(ctx, clusterConfig.AbsSubpath())
	return strings.TrimSpace(string(results)), err
}

func printDiff(diffStr string) {
	if diffStr == "" {
		fmt.Println("No diffs found.")
	} else {
		fmt.Println(diffStr)
	}
}
