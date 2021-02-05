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
	"github.com/segmentio/kubeapply/pkg/star/expand"
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
	helmVersionConstraint = ">= 3.5"
)

var expandCmd = &cobra.Command{
	Use:   "expand [cluster configs]",
	Short: "expand expands all configs associated with the cluster config",
	Args:  cobra.MinimumNArgs(1),
	RunE:  expandRun,
}

type expandFlags struct {
	// Clean old configs in expanded directory before expanding
	clean bool

	// Number of helm instances to run in parallel when expanding out charts.
	helmParallelism int
}

var expandFlagsValues expandFlags

func init() {
	expandCmd.Flags().BoolVar(
		&expandFlagsValues.clean,
		"clean",
		false,
		"Clean out old configs in expanded directory",
	)
	expandCmd.Flags().IntVar(
		&expandFlagsValues.helmParallelism,
		"helm-parallelism",
		5,
		"Parallelism on helm expansions",
	)

	RootCmd.AddCommand(expandCmd)
}

func expandRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	for _, arg := range args {
		paths, err := filepath.Glob(arg)
		if err != nil {
			return err
		}

		for _, path := range paths {
			if err := expandClusterPath(ctx, path, expandFlagsValues.clean); err != nil {
				return err
			}
		}
	}

	return nil
}

func expandClusterPath(ctx context.Context, path string, clean bool) error {
	clusterConfig, err := config.LoadClusterConfig(path, "")
	if err != nil {
		return err
	}
	if err := clusterConfig.CheckVersion(version.Version); err != nil {
		return err
	}

	return expandCluster(ctx, clusterConfig, clean)
}

func expandCluster(
	ctx context.Context,
	clusterConfig *config.ClusterConfig,
	clean bool,
) error {
	tempDir, err := ioutil.TempDir("", "expand")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	log.Infof("Expanding cluster %s", clusterConfig.DescriptiveName())

	var chartsPath string
	var chartGlobalsPath string

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

		chartGlobalsPath = filepath.Join(tempDir, "globals/globals.yaml")
		err = writeGlobals(chartGlobalsPath, clusterConfig)
		if err != nil {
			return err
		}
	}

	if clean {
		log.Infof("Cleaning old version of expanded configs")
		os.RemoveAll(clusterConfig.ExpandedPath)
	}

	if len(clusterConfig.Profiles) > 0 {
		for _, profile := range clusterConfig.Profiles {
			expandedPath := filepath.Join(clusterConfig.ExpandedPath, profile.Name)

			err = util.RestoreData(
				ctx,
				filepath.Dir(clusterConfig.FullPath()),
				profile.URL,
				expandedPath,
			)
			if err != nil {
				return err
			}

			err = expandProfile(
				ctx,
				expandedPath,
				chartsPath,
				chartGlobalsPath,
				&profile,
				clusterConfig,
			)
			if err != nil {
				return err
			}
		}
	} else {
		log.Infof("Copying configs to %s", clusterConfig.ExpandedPath)
		err = util.RecursiveCopy(
			clusterConfig.ProfilePath,
			clusterConfig.ExpandedPath,
		)
		if err != nil {
			return err
		}

		err = expandProfile(
			ctx,
			clusterConfig.ExpandedPath,
			chartsPath,
			chartGlobalsPath,
			&config.Profile{
				Name: "main",
			},
			clusterConfig,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func expandProfile(
	ctx context.Context,
	expandedPath string,
	chartsPath string,
	chartGlobalsPath string,
	profile *config.Profile,
	clusterConfig *config.ClusterConfig,
) error {
	log.Infof("Expanding profile %s in %s", profile.Name, expandedPath)

	// TODO: Should probably wrap in another struct that has fields for both cluster config
	// and profile.
	clusterConfig.Profile = profile
	err := util.ApplyTemplate(
		expandedPath,
		clusterConfig,
		true,
		true,
	)
	if err != nil {
		return err
	}

	log.Infof("Removing extraneous directories in %s", expandedPath)
	err = util.RemoveDirs(
		expandedPath,
		noExpandFile,
	)
	if err != nil {
		return err
	}

	if chartsPath != "" {
		log.Infof("Applying helm to charts in %s", expandedPath)

		helmClient := helm.HelmClient{
			RootDir:          filepath.Dir(clusterConfig.FullPath()),
			GlobalValuesPath: chartGlobalsPath,
			Parallelism:      expandFlagsValues.helmParallelism,
		}
		err = helmClient.ExpandHelmTemplates(
			ctx,
			expandedPath,
			chartsPath,
		)
		if err != nil {
			return err
		}
	}

	log.Infof(
		"Running starlark interpreter for star files in %s",
		expandedPath,
	)
	err = expand.ExpandStar(
		expandedPath,
		filepath.Dir(clusterConfig.FullPath()),
		clusterConfig.StarParams(),
	)
	if err != nil {
		return err
	}

	log.Infof(
		"Adding header comments to all YAML files in %s",
		expandedPath,
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
