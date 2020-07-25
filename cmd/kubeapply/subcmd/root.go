package subcmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// Errors returned from frontend commands
	ErrCommandMissing   = errors.New("must specify command to run")
	ErrTooManyArguments = errors.New("too many arguments")
	ErrTooFewArguments  = errors.New("too few arguments")

	debug bool
)

var RootCmd = &cobra.Command{
	Use:               "kubeapply",
	Short:             "kubeapply automates onebox workflows",
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: prerunE,
	PersistentPostRun: postrun,
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(
		&debug,
		"debug",
		"d",
		false,
		"Enable debug logging",
	)
}

func Execute(versionRef string) {
	RootCmd.Version = fmt.Sprintf("v%s (ref:%s)", version.Version, versionRef)

	if err := RootCmd.Execute(); err != nil {
		log.Error(err)
		switch err {
		case ErrTooFewArguments, ErrTooManyArguments:
			RootCmd.Usage()
		}
		os.Exit(1)
	}
}

func prerunE(cmd *cobra.Command, args []string) error {
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func postrun(cmd *cobra.Command, args []string) {}
