package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/segmentio/kubeapply/pkg/provider"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	debug       bool
	debugServer bool
	debugLogs   bool

	rootCmd = &cobra.Command{
		Use:   "terraform-provider-kubeapply",
		Short: "Terraform provider for kubeapply",
		RunE:  runProvider,
	}
)

func init() {
	rootCmd.Flags().BoolVar(
		&debugServer,
		"debug",
		getEnvDefault("KUBEAPPLY_DEBUG"),
		"Run both server and logs in debug mode; if set, debug-server and debug-logs flags are ignored",
	)
	rootCmd.Flags().BoolVar(
		&debugServer,
		"debug-server",
		getEnvDefault("KUBEAPPLY_DEBUG_SERVER"),
		"Run server in debug mode",
	)
	rootCmd.Flags().BoolVar(
		&debugServer,
		"debug-logs",
		getEnvDefault("KUBEAPPLY_DEBUG_LOGS"),
		"Log at debug level",
	)

	// Terraform requires a very simple log output format; see
	// https://www.terraform.io/docs/extend/debugging.html#inserting-log-lines-into-a-provider
	// for more details.
	log.SetFormatter(&simpleFormatter{})
	log.SetLevel(log.InfoLevel)
}

func runProvider(cmd *cobra.Command, args []string) error {
	if debug || debugLogs {
		log.SetLevel(log.DebugLevel)
	}

	log.Info("Starting kubeapply provider")
	ctx := context.Background()

	opts := &plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return provider.Provider(nil)
		},
	}

	if debug || debugServer {
		log.Info("Running server in debug mode")
		return plugin.Debug(ctx, "segment.io/kubeapply/kubeapply", opts)
	}

	plugin.Serve(opts)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

type simpleFormatter struct {
}

func (s *simpleFormatter) Format(entry *log.Entry) ([]byte, error) {
	return []byte(
		fmt.Sprintf(
			"[%s] %s\n",
			strings.ToUpper(entry.Level.String()),
			entry.Message,
		),
	), nil
}

func getEnvDefault(name string) bool {
	envVal := os.Getenv(name)
	return strings.ToUpper(envVal) == "TRUE"
}
