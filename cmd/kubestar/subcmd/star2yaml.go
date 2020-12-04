package subcmd

import (
	"encoding/json"
	"fmt"
	"os"

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
	varsStr string
}

var star2yamlFlagValues star2yamlFlags

func init() {
	star2yamlCmd.Flags().StringVar(
		&star2yamlFlagValues.varsStr,
		"vars",
		"",
		"JSON-formatted vars to insert in ctx object",
	)

	RootCmd.AddCommand(star2yamlCmd)
}

func star2yamlRun(cmd *cobra.Command, args []string) error {
	vars := map[string]interface{}{}

	if star2yamlFlagValues.varsStr != "" {
		if err := json.Unmarshal([]byte(star2yamlFlagValues.varsStr), &vars); err != nil {
			return err
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	result, err := expand.StarToYaml(args[0], cwd, vars)
	if err != nil {
		return err
	}
	fmt.Println(result)

	return nil
}
