package subcmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
	args       []string
	entrypoint string
}

var yaml2StarFlagValues yaml2StarFlags

func init() {
	yaml2starCmd.Flags().StringArrayVar(
		&yaml2StarFlagValues.args,
		"args",
		[]string{},
		"List of arguments to add to custom (non-main) entrypoint, in key=value format",
	)
	yaml2starCmd.Flags().StringVar(
		&yaml2StarFlagValues.entrypoint,
		"entrypoint",
		"main",
		"Name of entrypoint",
	)

	RootCmd.AddCommand(yaml2starCmd)
}

func yaml2starRun(cmd *cobra.Command, args []string) error {
	config := convert.Config{
		Entrypoint: yaml2StarFlagValues.entrypoint,
	}

	for _, arg := range yaml2StarFlagValues.args {
		components := strings.SplitN(arg, "=", 2)
		if len(components) != 2 {
			return fmt.Errorf("Argument not in format key=value: %s", arg)
		}
		config.Args = append(
			config.Args,
			convert.Arg{
				Name: components[0],
				// For now, treat all default values as (non-required) strings
				DefaultValue: components[1],
			},
		)
	}

	sort.Slice(config.Args, func(a, b int) bool {
		return config.Args[a].Name < config.Args[b].Name
	})

	filePaths := []string{}

	for _, arg := range args {
		paths, err := filepath.Glob(arg)
		if err != nil {
			return err
		}

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

	result, err := convert.YamlToStar(filePaths, config)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}
