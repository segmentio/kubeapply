package subcmd

import (
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "star2yaml [cluster configs]",
	Short: "apply runs kubectl apply over the resources associated with a cluster config",
	Args:  cobra.MinimumNArgs(1),
	RunE:  applyRun,
}
