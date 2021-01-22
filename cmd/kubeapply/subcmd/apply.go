package subcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/segmentio/kubeapply/pkg/cluster"
	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	"github.com/segmentio/kubeapply/pkg/cluster/kube"
	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply [cluster configs]",
	Short: "apply runs kubectl apply over the resources associated with a cluster config",
	Args:  cobra.MinimumNArgs(1),
	RunE:  applyRun,
}

type applyFlags struct {
	// Whether to expand before applying.
	expand bool

	// Whether to keep around temporary, intermediate configs that are used
	// for actual apply.
	keepConfigs bool

	// Path to kubeconfig. If unset, tries to fetch from the environment.
	kubeConfig string

	// Whether to just apply without checking anything
	noCheck bool

	// Whether to just run "kubectl apply" with the default output options
	simpleOutput bool

	// Run operatation in just a subset of the subdirectories of the expanded configs
	// (typically maps to namespace). If unset, considers all configs.
	subpaths []string

	// Whether to accept all prompts automatically; does not apply if
	// noCheck is enabled
	yes bool
}

var applyFlagValues applyFlags

func init() {
	applyCmd.Flags().BoolVar(
		&applyFlagValues.expand,
		"expand",
		false,
		"Expand before applying",
	)
	applyCmd.Flags().BoolVar(
		&applyFlagValues.keepConfigs,
		"keep-configs",
		false,
		"Whether to keep around intermediate configs for easier debugging",
	)
	applyCmd.Flags().StringVar(
		&applyFlagValues.kubeConfig,
		"kubeconfig",
		"",
		"Path to kubeconfig",
	)
	applyCmd.Flags().BoolVar(
		&applyFlagValues.noCheck,
		"no-check",
		false,
		"Skip all checks and just apply",
	)
	applyCmd.Flags().BoolVar(
		&applyFlagValues.simpleOutput,
		"simple-output",
		false,
		"Run kubectl apply without any special output options",
	)
	applyCmd.Flags().StringArrayVar(
		&applyFlagValues.subpaths,
		"subpath",
		[]string{},
		"Apply for expanded configs in the provided subpath(s) only",
	)
	applyCmd.Flags().BoolVarP(
		&applyFlagValues.yes,
		"yes",
		"y",
		false,
		"Skip all prompts; has no effect if --no-check is true",
	)

	RootCmd.AddCommand(applyCmd)
}

func applyRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	for _, arg := range args {
		paths, err := filepath.Glob(arg)
		if err != nil {
			return err
		}

		for _, path := range paths {
			if err := applyClusterPath(ctx, path); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyClusterPath(ctx context.Context, path string) error {
	clusterConfig, err := config.LoadClusterConfig(path, "")
	if err != nil {
		return err
	}
	if err := clusterConfig.CheckVersion(version.Version); err != nil {
		return err
	}

	if applyFlagValues.expand {
		if err := expandCluster(ctx, clusterConfig, false); err != nil {
			return err
		}
	}

	log.Infof("Applying cluster %s", clusterConfig.DescriptiveName())

	ok, err := util.DirExists(clusterConfig.ExpandedPath)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf(
			"Expanded path %s does not exist",
			clusterConfig.ExpandedPath,
		)
	}

	kubeConfig := applyFlagValues.kubeConfig

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
	clusterConfig.Subpaths = applyFlagValues.subpaths

	if !applyFlagValues.noCheck {
		err := execValidation(ctx, clusterConfig)
		if err != nil {
			return err
		}

		_, rawResult, err := execDiff(ctx, clusterConfig, true)
		if err != nil {
			log.Errorf("Error running diff: %+v", err)
			log.Info(
				"Try re-running with --debug to see verbose output. Note that diffs will not work if target namespace(s) don't exist yet.",
			)
			return err
		}
		printDiff(rawResult)

		if !applyFlagValues.yes {
			fmt.Print("Are you sure? (yes/no) ")
			var response string
			_, err = fmt.Scanln(&response)
			if err != nil {
				return err
			}
			if strings.TrimSpace(response) != "yes" {
				log.Infof("Not continuing")
				return nil
			}
		} else {
			log.Warn("Automatically continuing because --yes is true")
		}
	} else {
		log.Warn("Skipping checks because --no-check is true")
	}

	kubeClient, err := cluster.NewKubeClusterClient(
		ctx,
		&cluster.ClusterClientConfig{
			CheckApplyConsistency: false,
			ClusterConfig:         clusterConfig,
			Debug:                 debug,
			KeepConfigs:           applyFlagValues.keepConfigs,
			StreamingOutput:       applyFlagValues.simpleOutput,
			// TODO: Make locking an option
			UseLocks: false,
		},
	)
	if err != nil {
		return err
	}
	defer kubeClient.Close()

	if applyFlagValues.simpleOutput {
		results, err := kubeClient.Apply(
			ctx,
			clusterConfig.AbsSubpaths(),
			clusterConfig.ServerSideApply,
		)
		if err != nil {
			return fmt.Errorf("Error running apply: %s", string(results))
		}
	} else {
		results, err := kubeClient.ApplyStructured(
			ctx,
			clusterConfig.AbsSubpaths(),
			clusterConfig.ServerSideApply,
		)
		if err != nil {
			return err
		}

		log.Infof("Apply results:\n%s", apply.ResultsTextTable(results))
	}

	return nil
}
