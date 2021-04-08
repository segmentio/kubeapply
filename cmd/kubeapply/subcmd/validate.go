package subcmd

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/util"
	"github.com/segmentio/kubeapply/pkg/validation"
	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [cluster configs]",
	Short: "validate checks the cluster configs using kubeval",
	Args:  cobra.MinimumNArgs(1),
	RunE:  validateRun,
}

type validateFlags struct {
	// Expand before validating.
	expand bool
}

var validateFlagValues validateFlags

func init() {
	validateCmd.Flags().BoolVar(
		&validateFlagValues.expand,
		"expand",
		false,
		"Expand before validating",
	)

	RootCmd.AddCommand(validateCmd)
}

func validateRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	for _, arg := range args {
		paths, err := filepath.Glob(arg)
		if err != nil {
			return err
		}

		for _, path := range paths {
			if err := validateClusterPath(ctx, path); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateClusterPath(ctx context.Context, path string) error {
	clusterConfig, err := config.LoadClusterConfig(path, "")
	if err != nil {
		return err
	}
	if err := clusterConfig.CheckVersion(version.Version); err != nil {
		return err
	}

	if validateFlagValues.expand {
		if err := expandCluster(ctx, clusterConfig, false); err != nil {
			return err
		}
	}

	ok, err := util.DirExists(clusterConfig.ExpandedPath)
	if err != nil {
		return err
	} else if !ok {
		return fmt.Errorf(
			"Expanded path %s does not exist",
			clusterConfig.ExpandedPath,
		)
	}

	return execValidation(ctx, clusterConfig)
}

func execValidation(ctx context.Context, clusterConfig *config.ClusterConfig) error {
	log.Infof("Validating cluster %s", clusterConfig.DescriptiveName())

	kubeValidator, err := validation.NewKubeValidator()
	if err != nil {
		return err
	}

	log.Infof("Running kubeconform on configs in %+v", clusterConfig.AbsSubpaths())
	results, err := kubeValidator.RunSchemaValidation(ctx, clusterConfig.AbsSubpaths()[0])
	if err != nil {
		return err
	}

	numInvalidResources := 0

	for _, result := range results {
		switch result.Status {
		case validation.StatusValid:
			log.Infof("Resource %s in file %s OK", result.PrettyName(), result.Filename)
		case validation.StatusSkipped:
			log.Debugf("Resource %s in file %s was skipped", result.PrettyName(), result.Filename)
		case validation.StatusError:
			numInvalidResources++
			log.Errorf("File %s could not be validated: %+v", result.Filename, result.Message)
		case validation.StatusInvalid:
			numInvalidResources++
			log.Errorf(
				"Resource %s in file %s is invalid: %s",
				result.PrettyName(),
				result.Filename,
				result.Message,
			)
		case validation.StatusEmpty:
		default:
			log.Infof("Unrecognized result type: %+v", result)
		}
	}

	if numInvalidResources > 0 {
		return fmt.Errorf(
			"Validation failed for %d resources",
			numInvalidResources,
		)
	}

	log.Infof("Validation of cluster %s passed", clusterConfig.DescriptiveName())
	return nil
}
