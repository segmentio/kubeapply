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
	Short: "validate checks the cluster configs using kubeconform and (optionally) opa policies",
	Args:  cobra.MinimumNArgs(1),
	RunE:  validateRun,
}

type validateFlags struct {
	// Expand before validating.
	expand bool

	// Number of worker goroutines to use for validation.
	numWorkers int

	// Paths to OPA policy rego files that will be run against kube resources.
	// See https://www.openpolicyagent.org/ for more details.
	policies []string
}

var validateFlagValues validateFlags

func init() {
	validateCmd.Flags().BoolVar(
		&validateFlagValues.expand,
		"expand",
		false,
		"Expand before validating",
	)
	validateCmd.Flags().IntVar(
		&validateFlagValues.numWorkers,
		"num-workers",
		4,
		"Number of workers to use for validation",
	)
	validateCmd.Flags().StringArrayVar(
		&validateFlagValues.policies,
		"policy",
		[]string{},
		"Paths to OPA policies",
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

	kubeconformChecker, err := validation.NewKubeconformChecker()
	if err != nil {
		return err
	}

	policies, err := validation.DefaultPoliciesFromGlobs(
		ctx,
		validateFlagValues.policies,
		map[string]interface{}{
			// TODO: Add more parameters here (or entire config)?
			"cluster": clusterConfig.Cluster,
			"region":  clusterConfig.Region,
			"env":     clusterConfig.Env,
		},
	)
	if err != nil {
		return err
	}

	checkers := []validation.Checker{kubeconformChecker}
	for _, policy := range policies {
		checkers = append(checkers, policy)
	}

	validator := validation.NewKubeValidator(
		validation.KubeValidatorConfig{
			NumWorkers: validateFlagValues.numWorkers,
			Checkers:   checkers,
		},
	)

	log.Infof("Running validator on configs in %+v", clusterConfig.AbsSubpaths())
	results, err := validator.RunChecks(ctx, clusterConfig.AbsSubpaths()[0])
	if err != nil {
		return err
	}

	numInvalidResourceChecks := 0
	numValidResourceChecks := 0
	numSkippedResourceChecks := 0

	for _, result := range results {
		for _, checkResult := range result.CheckResults {
			switch checkResult.Status {
			case validation.StatusValid:
				numValidResourceChecks++
				log.Debugf(
					"Resource %s in file %s OK according to check %s",
					result.Resource.PrettyName(),
					result.Resource.Path,
					checkResult.CheckName,
				)
			case validation.StatusSkipped:
				numSkippedResourceChecks++
				log.Debugf(
					"Resource %s in file %s was skipped by check %s",
					result.Resource.PrettyName(),
					result.Resource.Path,
					checkResult.CheckName,
				)
			case validation.StatusError:
				numInvalidResourceChecks++
				log.Errorf(
					"Resource %s in file %s could not be processed by check %s: %s",
					result.Resource.PrettyName(),
					result.Resource.Path,
					checkResult.CheckName,
					checkResult.Message,
				)
			case validation.StatusInvalid:
				numInvalidResourceChecks++
				log.Errorf(
					"Resource %s in file %s is invalid according to check %s: %s",
					result.Resource.PrettyName(),
					result.Resource.Path,
					checkResult.CheckName,
					checkResult.Message,
				)
			case validation.StatusEmpty:
			default:
				log.Infof("Unrecognized result type: %+v", result)
			}
		}
	}

	if numInvalidResourceChecks > 0 {
		return fmt.Errorf(
			"Validation failed for %d resources in cluster %s (%d checks valid, %d skipped)",
			numInvalidResourceChecks,
			clusterConfig.DescriptiveName(),
			numValidResourceChecks,
			numSkippedResourceChecks,
		)
	}

	log.Infof(
		"Validation of cluster %s passed (%d checks valid, %d skipped)",
		clusterConfig.DescriptiveName(),
		numValidResourceChecks,
		numSkippedResourceChecks,
	)
	return nil
}
