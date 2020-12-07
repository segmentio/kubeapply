package subcmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/segmentio/kubeapply/pkg/config"
	"github.com/segmentio/kubeapply/pkg/star/expand"
	"github.com/spf13/cobra"
)

var star2yamlCmd = &cobra.Command{
	Use:   "star2yaml [star path]",
	Short: "star2yaml expands a kube starlark file to YAML",
	Args:  cobra.ExactArgs(1),
	RunE:  star2yamlRun,
}

type star2yamlFlags struct {
	clusterConfig string
	varsStr       string
}

var star2yamlFlagValues star2yamlFlags

func init() {
	star2yamlCmd.Flags().StringVar(
		&star2yamlFlagValues.clusterConfig,
		"cluster-config",
		"",
		"Path to a kubeapply-formatted YAML cluster config; used to set vars in ctx object",
	)
	star2yamlCmd.Flags().StringVar(
		&star2yamlFlagValues.varsStr,
		"vars",
		"",
		"Extra JSON-formatted vars to insert in ctx object",
	)

	RootCmd.AddCommand(star2yamlCmd)
}

func star2yamlRun(cmd *cobra.Command, args []string) error {
	var starParams map[string]interface{}

	if star2yamlFlagValues.clusterConfig != "" {
		clusterConfig, err := config.LoadClusterConfig(
			star2yamlFlagValues.clusterConfig,
			"",
		)
		if err != nil {
			return err
		}
		starParams = clusterConfig.StarParams()
	} else {
		starParams = map[string]interface{}{}
	}

	if star2yamlFlagValues.varsStr != "" {
		extraParams := map[string]interface{}{}

		if err := json.Unmarshal(
			[]byte(star2yamlFlagValues.varsStr),
			&extraParams,
		); err != nil {
			return err
		}

		for key, value := range extraParams {
			starParams[key] = value
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	result, err := expand.StarToYaml(args[0], cwd, starParams)
	if err != nil {
		return err
	}
	fmt.Println(result)

	return nil
}
