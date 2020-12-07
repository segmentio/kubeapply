package main

import (
	"os"

	"github.com/segmentio/kubeapply/cmd/kubestar/subcmd"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"k8s.io/klog"
)

var (
	// VersionRef is the kubeapply git SHA reference. It's set at build-time.
	VersionRef = "dev"
)

func init() {
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	klog.SetOutput(os.Stderr)
}

func main() {
	subcmd.Execute(VersionRef)
}
