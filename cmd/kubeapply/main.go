package main

import (
	"os"

	"github.com/segmentio/kubeapply/cmd/kubeapply/subcmd"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"k8s.io/klog"
)

var (
	// Set by LDFLAGS in build
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
