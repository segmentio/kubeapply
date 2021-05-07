package subcmd

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	// Directory to write CSV summaries of OPA results to.
	csvOutDir string

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
	validateCmd.Flags().StringVar(
		&validateFlagValues.csvOutDir,
		"csv-out-dir",
		"",
		"Directory to write CSV results to",
	)
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

	if validateFlagValues.csvOutDir != "" {
		if err := os.MkdirAll(validateFlagValues.csvOutDir, 0755); err != nil {
			log.Fatal(err)
		}
	}

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

func execValidation(
	ctx context.Context,
	clusterConfig *config.ClusterConfig,
) error {
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

	counts := validation.CountsByStatus(results)
	resultsWithIssues := validation.ResultsWithIssues(results)

	if len(resultsWithIssues) > 0 {
		log.Warnf("Found %d resources with potential issues", len(resultsWithIssues))
		for _, result := range resultsWithIssues {
			fmt.Println(
				validation.ResultTable(
					result,
					clusterConfig.DescriptiveName(),
					clusterConfig.ExpandedPath,
					debug,
				),
			)
		}

		if validateFlagValues.csvOutDir != "" {
			outputPath := filepath.Join(
				validateFlagValues.csvOutDir,
				fmt.Sprintf("%s.csv", clusterConfig.DescriptiveName()),
			)
			log.Infof("Writing resource issues to %s", outputPath)

			err = validation.WriteResultsCSV(
				resultsWithIssues,
				clusterConfig.DescriptiveName(),
				clusterConfig.ExpandedPath,
				outputPath,
			)
			if err != nil {
				return err
			}
		}
	}

	if counts[validation.StatusError]+counts[validation.StatusInvalid] > 0 {
		return errors.New("Validation failed")
	}

	log.Infof("Validation passed")
	return nil
}
