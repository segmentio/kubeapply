package subcmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/helm"
	"github.com/segmentio/kubeapply/pkg/star"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	// If a directory has this file in it, it will be deleted before running
	// helm and downstream steps.
	noExpandFile = ".noexpand"

	// Require a minimum helm version to ensure that expansion works properly
	helmVersionConstraint = ">= 3.2"
)

var expandCmd = &cobra.Command{
	Use:   "expand [cluster configs]",
	Short: "expand expands all configs associated with the cluster config",
	Args:  cobra.MinimumNArgs(1),
	RunE:  expandRun,
}

type expandFlags struct {
	// Number of helm instances to run in parallel when expanding out charts.
	helmParallelism int
}

var expandFlagsValues expandFlags

func init() {
	expandCmd.Flags().IntVar(
		&expandFlagsValues.helmParallelism,
		"parallelism",
		5,
		"Parallelism on helm expansions",
	)

	RootCmd.AddCommand(expandCmd)
}

func expandRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	for _, arg := range args {
		if err := expandClusterPath(ctx, arg); err != nil {
			return err
		}
	}

	return nil
}

func expandClusterPath(ctx context.Context, path string) error {
	clusterConfig, err := config.LoadClusterConfig(path, "")
	if err != nil {
		return err
	}
	if err := clusterConfig.CheckVersion(version.Version); err != nil {
		return err
	}

	return expandCluster(ctx, clusterConfig)
}

func expandCluster(ctx context.Context, clusterConfig *config.ClusterConfig) error {
	tempDir, err := ioutil.TempDir("", "expand")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	log.Infof("Expanding cluster %s", clusterConfig.DescriptiveName())

	var chartsPath string

	if clusterConfig.Charts != "" {
		log.Debugf(
			"Checking that helm statisfies version constraint %s",
			helmVersionConstraint,
		)
		if err := helm.CheckHelmVersion(ctx, helmVersionConstraint); err != nil {
			return err
		}

		chartsPath = filepath.Join(tempDir, "charts")
		err = util.RestoreData(
			ctx,
			filepath.Dir(clusterConfig.FullPath()),
			clusterConfig.Charts,
			chartsPath,
		)
		if err != nil {
			return err
		}
		log.Debugf("Local charts path is %s", chartsPath)
	}

	log.Infof("Copying configs to %s", clusterConfig.ExpandedPath)
	err = util.RecursiveCopy(
		clusterConfig.ProfilePath,
		clusterConfig.ExpandedPath,
	)
	if err != nil {
		return err
	}

	log.Infof("Applying templates in %s", clusterConfig.ExpandedPath)
	err = util.ApplyTemplate(
		clusterConfig.ExpandedPath,
		clusterConfig,
		true,
	)
	if err != nil {
		return err
	}

	log.Infof("Removing extraneous directories in %s", clusterConfig.ExpandedPath)
	err = util.RemoveDirs(
		clusterConfig.ExpandedPath,
		noExpandFile,
	)
	if err != nil {
		return err
	}

	if chartsPath != "" {
		log.Infof("Applying helm to charts in %s", clusterConfig.ExpandedPath)
		globalsPath := filepath.Join(tempDir, "globals/globals.yaml")
		err = writeGlobals(globalsPath, clusterConfig)
		if err != nil {
			return err
		}

		helmClient := helm.HelmClient{
			RootDir:          filepath.Dir(clusterConfig.FullPath()),
			GlobalValuesPath: globalsPath,
			Parallelism:      expandFlagsValues.helmParallelism,
		}
		err = helmClient.ExpandHelmTemplates(
			ctx,
			clusterConfig.ExpandedPath,
			chartsPath,
		)
		if err != nil {
			return err
		}
	}

	starParams := map[string]interface{}{
		"cluster":    clusterConfig.Cluster,
		"env":        clusterConfig.Env,
		"region":     clusterConfig.Region,
		"parameters": clusterConfig.Parameters,
	}
	for key, value := range clusterConfig.Parameters {
		starParams[key] = value
	}

	log.Infof(
		"Running starlark interpreter for star files in %s",
		clusterConfig.ExpandedPath,
	)
	err = star.ExpandStar(
		clusterConfig.ExpandedPath,
		filepath.Dir(clusterConfig.FullPath()),
		starParams,
	)
	if err != nil {
		return err
	}

	log.Infof(
		"Adding header comments to all YAML files in %s",
		clusterConfig.ExpandedPath,
	)
	err = util.AddHeaders(clusterConfig.ExpandedPath)
	if err != nil {
		return err
	}

	return nil
}

func writeGlobals(path string, clusterConfig *config.ClusterConfig) error {
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	globalsLines := []string{
		"---",
		"global:",
		fmt.Sprintf("  cluster: %s", clusterConfig.Cluster),
		fmt.Sprintf("  region: %s", clusterConfig.Region),
		fmt.Sprintf("  shortRegion: %s", clusterConfig.ShortRegion()),
	}

	return ioutil.WriteFile(
		path,
		[]byte(strings.Join(globalsLines, "\n")),
		0644,
	)
}
