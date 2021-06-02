package main

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"
	"github.com/segmentio/kubeapply/pkg/provider"
	log "github.com/sirupsen/logrus"
)

func main() {
	log.Infof("Starting kubeapply provider")

	opts := &plugin.ServeOpts{
		ProviderFunc: func() *schema.Provider {
			return provider.Provider()
		},
	}

	err := plugin.Debug(context.Background(), "segmentio/kubeapply/kubeapply", opts)
	if err != nil {
		log.Fatal(err.Error())
	}

	//plugin.Serve(opts)
}
