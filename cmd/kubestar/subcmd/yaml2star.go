package subcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/segmentio/kubeapply/pkg/star/convert"
	"github.com/spf13/cobra"
)

var yaml2starCmd = &cobra.Command{
	Use:   "yaml2star [YAML configs]",
	Short: "yaml2star converts one or more kube YAML manifests to starlark",
	Args:  cobra.MinimumNArgs(1),
	RunE:  yaml2starRun,
}

type yaml2StarFlags struct {
	args []string
}

var yaml2StarFlagValues yaml2StarFlags

func init() {
	yaml2starCmd.Flags().StringArrayVar(
		&yaml2StarFlagValues.args,
		"args",
		[]string{},
		"list of arguments to add to entrypoint, in key=value format",
	)

	RootCmd.AddCommand(yaml2starCmd)
}

func yaml2starRun(cmd *cobra.Command, args []string) error {
	filePaths := []string{}

	for _, arg := range args {
		paths, err := filepath.Glob(arg)
		if err != nil {
			return err
		}

		fmt.Println(paths)

		for _, path := range paths {
			info, err := os.Stat(path)
			if err != nil {
				return err
			}
			if info.IsDir() {
				continue
			}

			filePaths = append(filePaths, path)
		}
	}

	result, err := convert.YamlToStar(filePaths, convert.Config{})
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}
