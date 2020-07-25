package subcmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/segmentio/kubeapply/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "check verifies that all dependencies are installed at their proper versions",
	Args:  cobra.MaximumNArgs(0),
	RunE:  checkRun,
}

func init() {
	RootCmd.AddCommand(checkCmd)
}

func checkRun(cmd *cobra.Command, args []string) error {
	log.Infof("kubeapply version:\n>>> %s", version.Version)

	if err := checkDep("helm", "version"); err != nil {
		return err
	}
	if err := checkDep("kubectl", "version"); err != nil {
		return err
	}
	if err := checkDep("kubeval", "--version"); err != nil {
		return err
	}

	return nil
}

func checkDep(name string, versionArg string) error {
	log.Infof("Looking for %s", name)
	depPath, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("Could not find %s in your PATH", name)
	}
	log.Infof("Found %s at %s", name, depPath)

	versionCmd := exec.Command(name, versionArg)
	output, err := versionCmd.CombinedOutput()
	if err != nil {
		return err
	}
	log.Infof("%s version:\n%s", name, prettyLines(output))
	return nil
}

func prettyLines(content []byte) string {
	inputLines := strings.Split(strings.TrimSpace(string(content)), "\n")
	outputLines := []string{}

	for _, line := range inputLines {
		outputLines = append(outputLines, fmt.Sprintf(">>> %s", line))
	}

	return strings.Join(outputLines, "\n")
}
