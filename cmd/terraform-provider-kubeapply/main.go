package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	// Terraform doesn't provide any sort of "cleanup" notification for providers, so there's
	// no way to do any cleanup when the provider exits. Doing the below as a hacky alternative
	// to ensure that the disk doesn't fill up with accumulated junk.
	threshold := time.Now().UTC().Add(-1 * time.Hour)

	tempDirs, _ := filepath.Glob(
		filepath.Join(os.TempDir(), "kubeapply_*"),
	)
	for _, tempDir := range tempDirs {
		info, err := os.Stat(tempDir)
		if err != nil {
			log.Warnf("Error getting info for %s: %+v", tempDir, err)
			continue
		}
		if info.ModTime().Before(threshold) {
			log.Infof("Cleaning temp dir %s", tempDir)
			err = os.RemoveAll(tempDir)
			if err != nil {
				log.Warnf("Error deleting %s: %+v", tempDir, err)
			}
		}
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
	if len(entry.Data) > 0 {
		fieldsJSON, _ := json.Marshal(entry.Data)

		return []byte(
			fmt.Sprintf(
				"[%s] %s (%+v)\n",
				strings.ToUpper(entry.Level.String()),
				entry.Message,
				string(fieldsJSON),
			),
		), nil
	} else {
		return []byte(
			fmt.Sprintf(
				"[%s] %s\n",
				strings.ToUpper(entry.Level.String()),
				entry.Message,
			),
		), nil
	}
}

func getEnvDefault(name string) bool {
	envVal := os.Getenv(name)
	return strings.ToUpper(envVal) == "TRUE"
}
