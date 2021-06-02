package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/segmentio/kubeapply/pkg/provider"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var (
	debug bool

	rootCmd = &cobra.Command{
		Use:   "terraform-provider-kubeapply",
		Short: "Terraform provider for kubeapply",
		RunE:  runProvider,
	}
)

func init() {
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Run in debug mode")

	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	log.SetLevel(log.InfoLevel)
}

func runProvider(cmd *cobra.Command, args []string) error {
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	log.Info("Starting kubeapply provider")
	ctx := context.Background()

	opts := &plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return provider.Provider(nil)
		},
	}

	if debug {
		log.Info("Running in debug mode")
		return plugin.Debug(ctx, "segmentio/kubeapply/kubeapply", opts)
	}

	plugin.Serve(opts)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
