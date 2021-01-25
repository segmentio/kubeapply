package kube

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/briandowns/spinner"
	"github.com/segmentio/kubeapply/data"
	"github.com/segmentio/kubeapply/pkg/util"
	log "github.com/sirupsen/logrus"
)

const (
	diffScript = "kdiff-wrapper.sh"
)

// TODO: Switch to a YAML library that supports doing this splitting for us.
var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

// OrderedClient is a kubectl-wrapped client that tries to be clever about the order
// in which resources are created or destroyed.
type OrderedClient struct {
	kubeConfigPath string
	keepConfigs    bool
	extraEnv       []string
	debug          bool
	serverSide     bool
}

// NewOrderedClient returns a new OrderedClient instance.
func NewOrderedClient(
	kubeConfigPath string,
	keepConfigs bool,
	extraEnv []string,
	debug bool,
	serverSide bool,
) *OrderedClient {
	return &OrderedClient{
		kubeConfigPath: kubeConfigPath,
		keepConfigs:    keepConfigs,
		extraEnv:       extraEnv,
		debug:          debug,
		serverSide:     serverSide,
	}
}

// Apply runs kubectl apply on the manifests in the argument path. The apply is done
// in the optimal order based on resource type.
func (k *OrderedClient) Apply(
	ctx context.Context,
	applyPaths []string,
	output bool,
	format string,
	dryRun bool,
) ([]byte, error) {
	tempDir, err := ioutil.TempDir("", "manifests")
	if err != nil {
		return nil, err
	}
	defer func() {
		if k.keepConfigs {
			log.Infof("Keeping temporary configs in %s", tempDir)
		} else {
			os.RemoveAll(tempDir)
		}
	}()

	manifests, err := GetManifests(applyPaths)
	if err != nil {
		return nil, err
	}
	SortManifests(manifests)

	for m, manifest := range manifests {
		// kubectl applies resources in their lexicographic ordering, so this naming scheme
		// should force it to apply the manifests in the order we want.

		var name string
		var namespace string

		if manifest.Head.Metadata != nil {
			name = manifest.Head.Metadata.Name
			namespace = manifest.Head.Metadata.Namespace
		}

		tempPath := filepath.Join(
			tempDir,
			fmt.Sprintf(
				"%06d_%s_%s_%s.yaml",
				m,
				name,
				namespace,
				manifest.Head.Kind,
			),
		)

		err = ioutil.WriteFile(tempPath, []byte(manifest.Contents), 0644)
		if err != nil {
			return nil, err
		}
	}

	args := []string{
		"apply",
		"--field-manager=''",
		"--kubeconfig",
		k.kubeConfigPath,
		"-R",
		"-f",
		tempDir,
	}
	if k.serverSide {
		args = append(args, "--server-side", "true")
	}
	if k.debug {
		args = append(args, "-v", "8")
	}
	if format != "" {
		args = append(args, "-o", format)
	}
	if dryRun {
		args = append(args, "--dry-run")
	}

	if output {
		return runKubectlOutput(
			ctx,
			args,
			k.extraEnv,
			nil,
		)
	}
	return nil, runKubectl(
		ctx,
		args,
		k.extraEnv,
	)
}

// Diff runs kubectl diff for the configs at the argument path.
func (k *OrderedClient) Diff(
	ctx context.Context,
	configPaths []string,
	structured bool,
	spinner *spinner.Spinner,
) ([]byte, error) {
	var diffCmd string

	tempDir, err := ioutil.TempDir("", "diff")
	if err != nil {
		return nil, err
	}
	defer func() {
		if k.keepConfigs {
			log.Infof("Keeping temporary configs in %s", tempDir)
		} else {
			os.RemoveAll(tempDir)
		}
	}()

	args := []string{
		"--field-manager=''",
		"--kubeconfig",
		k.kubeConfigPath,
		"diff",
		"-R",
	}

	for _, configPath := range configPaths {
		args = append(args, "-f", configPath)
	}

	if k.serverSide {
		args = append(args, "--server-side", "true")
	}
	if k.debug {
		args = append(args, "-v", "8")
	}

	envVars := []string{}

	if structured {
		diffCmd = filepath.Join(tempDir, diffScript)
		err = ioutil.WriteFile(
			diffCmd,
			data.MustAsset(fmt.Sprintf("scripts/%s", diffScript)),
			0755,
		)
		if err != nil {
			return nil, err
		}

		envVars = append(
			envVars,
			fmt.Sprintf("KUBECTL_EXTERNAL_DIFF=%s", diffCmd),
		)
	}

	return runKubectlOutput(
		ctx,
		args,
		envVars,
		spinner,
	)
}

// Summary returns a pretty summary of the current cluster state.
func (k *OrderedClient) Summary(
	ctx context.Context,
) (string, error) {
	tempDir, err := ioutil.TempDir("", "cluster-summary")
	if err != nil {
		return "", err
	}
	defer func() {
		if k.keepConfigs {
			log.Infof("Keeping temporary configs in %s", tempDir)
		} else {
			os.RemoveAll(tempDir)
		}
	}()

	err = data.RestoreAssets(tempDir, "scripts/cluster-summary")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(
		"python",
		filepath.Join(tempDir, "scripts/cluster-summary/cluster_summary.py"),
		"--no-color",
		"--kubeconfig",
		k.kubeConfigPath,
	)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

// GetNamespaceUID returns the kubernetes identifier for a given namespace in this cluster.
func (k *OrderedClient) GetNamespaceUID(ctx context.Context, namespace string) (string, error) {
	if namespace == "" {
		return "", errors.New("expected a valid kubernetes namespace")
	}

	args := []string{
		"--kubeconfig",
		k.kubeConfigPath,
		"get",
		"namespace",
		namespace,
		"-o",
		"json",
	}

	out, err := runKubectlOutput(ctx, args, nil, nil)
	if err != nil {
		return "", err
	}

	var j struct {
		Metadata struct {
			UID string `json:"uid"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(out, &j); err != nil {
		return "", err
	}

	return j.Metadata.UID, nil
}

func runKubectl(ctx context.Context, args []string, extraEnv []string) error {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return err
	}

	return util.RunCmdWithPrinters(
		ctx,
		kubectlPath,
		args,
		extraEnv,
		nil,
		util.LogrusInfoPrinter("[kubectl]"),
		util.LogrusInfoPrinter("[kubectl]"),
	)
}

func runKubectlOutput(
	ctx context.Context,
	args []string,
	extraEnv []string,
	spinner *spinner.Spinner,
) ([]byte, error) {
	kubectlPath, err := exec.LookPath("kubectl")
	if err != nil {
		return nil, err
	}

	log.Infof("Running kubectl with args %+v", args)

	if spinner != nil {
		spinner.Start()
		defer spinner.Stop()
	}

	cmd := exec.CommandContext(ctx, kubectlPath, args...)

	envVars := os.Environ()
	envVars = append(envVars, extraEnv...)
	cmd.Env = envVars

	return cmd.CombinedOutput()
}
