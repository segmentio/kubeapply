package helm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/segmentio/kubeapply/pkg/util"
	log "github.com/sirupsen/logrus"
)

var (
	sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

	// Header comments that can be set in helm chart values files to change default behavior
	chartOverrideHeaders     = []string{"kubeapply__chart", "kubeapply__charts", "charts"}
	chartNameHeaders         = []string{"kubeapply__chartName", "chartName"}
	chartsSubDirHeaders      = []string{"kubeapply__chartsSubdir"}
	disabledHeaders          = []string{"kubeapply__disabled", "disabled"}
	namespaceOverrideHeaders = []string{"kubeapply__namespace", "namespace"}
	releaseNameHeaders       = []string{"kubeapply__releaseName", "releaseName"}
)

// HelmClient represents a client that can be used for expanding out Helm templates into full
// Kubernetes configs.
type HelmClient struct {
	// Debug indicates whether helm should be run with debug logging.
	Debug bool

	// GlobalValuesPath is an optional path to a set of "global" values that will be used to
	// supplement the chart-specific values.
	GlobalValuesPath string

	// Parallelism is the number of helm processes that should be run in parallel.
	Parallelism int

	// RootDir is the root relative to which file URLs will be fetched. Only applies for charts
	// that override their sources with a file URL.
	RootDir string
}

type helmContext struct {
	chartsPath    string
	namespace     string
	part          int
	totalParts    int
	valuesPath    string
	valuesContent []byte
}

// ExpandHelmTemplates expands out all of the helm values files in the provided configPath.
// Charts are sourced from either the provided chartsPath or from the override location
// in the value file yaml.
func (c *HelmClient) ExpandHelmTemplates(
	ctx context.Context,
	configPath string,
	chartsPath string,
) error {
	helmContexts := []helmContext{}

	err := filepath.Walk(
		configPath,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(subPath, ".helm.yaml") {
				return nil
			}

			var namespace string
			if s := strings.Split(strings.TrimPrefix(subPath, configPath), "/"); len(s) > 1 {
				namespace = s[1]
			}

			if namespace == "" {
				return fmt.Errorf("Resource %s is defined out of a namespace.", subPath)
			}

			contents, err := ioutil.ReadFile(subPath)
			if err != nil {
				return err
			}
			subParts := sep.Split(string(contents), -1)

			for s, subPart := range subParts {
				helmContexts = append(
					helmContexts,
					helmContext{
						namespace:     namespace,
						chartsPath:    chartsPath,
						part:          s,
						totalParts:    len(subParts),
						valuesPath:    subPath,
						valuesContent: []byte(subPart),
					},
				)
			}
			return nil
		},
	)
	if err != nil {
		return err
	}

	helmContextsChan := make(chan helmContext, len(helmContexts))
	defer close(helmContextsChan)

	for _, helmContext := range helmContexts {
		helmContextsChan <- helmContext
	}

	errChan := make(chan error, len(helmContexts))

	runnerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := 0; i < c.Parallelism; i++ {
		go func() {
			for {
				hctx, ok := <-helmContextsChan
				if !ok {
					return
				}

				err := c.generateHelmTemplates(
					runnerCtx,
					hctx,
				)
				if err != nil {
					errChan <- err
				}

				if hctx.part == 0 {
					// Only delete values file once
					errChan <- os.RemoveAll(hctx.valuesPath)
				} else {
					errChan <- nil
				}
			}
		}()
	}

	for i := 0; i < len(helmContexts); i++ {
		err := <-errChan
		if err != nil {
			cancel()
			return err
		}
	}

	return nil
}

func (c *HelmClient) generateHelmTemplates(
	ctx context.Context,
	hctx helmContext,
) error {
	log.Infof(
		"Processing helm chart with values %s in %s (part %d/%d)",
		filepath.Base(hctx.valuesPath),
		hctx.namespace,
		hctx.part+1,
		hctx.totalParts,
	)

	trimmedContents := bytes.TrimSpace(hctx.valuesContent)

	if len(trimmedContents) == 0 {
		log.Warnf(
			"Values file %s in %s (part %d/%d) is empty, skipping",
			filepath.Base(hctx.valuesPath),
			hctx.namespace,
			hctx.part+1,
			hctx.totalParts,
		)
		return nil
	}

	var contentsMap map[string]interface{}
	if err := yaml.Unmarshal(trimmedContents, &contentsMap); err != nil {
		return fmt.Errorf(
			"File %s (part %d/%d) is not valid YAML. If it's a template, be sure to add '.gotpl' in its name",
			hctx.valuesPath,
			hctx.part+1,
			hctx.totalParts,
		)
	}

	headerComments := getHeaderComments(trimmedContents)

	if getValue(headerComments, disabledHeaders...) == "true" {
		log.Warnf(
			"Skipping values file %s in %s (part %d/%d) because it has disabled: true",
			filepath.Base(hctx.valuesPath),
			hctx.namespace,
			hctx.part+1,
			hctx.totalParts,
		)
		return nil
	}

	var localChartsPath string
	chartsOverride := getValue(headerComments, chartOverrideHeaders...)

	if chartsOverride != "" {
		log.Debugf("Found charts override: %s", chartsOverride)

		tempDir, err := ioutil.TempDir("", "charts")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tempDir)

		localChartsPath = filepath.Join(tempDir, "charts")
		err = util.RestoreData(
			ctx,
			c.RootDir,
			chartsOverride,
			localChartsPath,
		)
		if err != nil {
			return err
		}

		chartsSubdir := getValue(headerComments, chartsSubDirHeaders...)
		if chartsSubdir != "" {
			localChartsPath = filepath.Join(localChartsPath, chartsSubdir)
		}

		log.Debugf("Using charts override path of %s", localChartsPath)
	} else {
		localChartsPath = hctx.chartsPath
	}

	baseName := filepath.Base(hctx.valuesPath)
	nameComponents := strings.Split(baseName, ".")

	chartNameOverride := getValue(headerComments, chartNameHeaders...)
	var chartNamePath string
	if chartNameOverride != "" {
		chartNamePath = chartNameOverride
	} else {
		chartNamePath = nameComponents[0]
	}

	namespaceOverride := getValue(headerComments, namespaceOverrideHeaders...)
	var templateNamespace string
	if namespaceOverride != "" {
		templateNamespace = namespaceOverride
	} else {
		templateNamespace = hctx.namespace
	}

	chartPath := filepath.Join(localChartsPath, chartNamePath)

	depArgs := []string{
		"dep",
		"update",
		chartPath,
	}
	if c.Debug {
		depArgs = append(depArgs, "--debug")
	}

	err := runHelm(ctx, depArgs)
	if err != nil {
		return err
	}

	releaseName := getValue(headerComments, releaseNameHeaders...)

	tempValuesDir, err := ioutil.TempDir("", "helm")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempValuesDir)
	tempValuesPath := filepath.Join(tempValuesDir, "values.yaml")
	if err := ioutil.WriteFile(tempValuesPath, trimmedContents, 0644); err != nil {
		return err
	}

	templateArgs := []string{
		"template",
		fmt.Sprintf("--namespace=%s", templateNamespace),
		fmt.Sprintf("--values=%s", tempValuesPath),
		fmt.Sprintf("--output-dir=%s", filepath.Dir(hctx.valuesPath)),
	}
	if releaseName != "" {
		templateArgs = append(templateArgs, fmt.Sprintf("--name-template=%s", releaseName))

		// Include release name in output paths so different ones don't clobber each other
		templateArgs = append(templateArgs, "--release-name")
	}
	if c.GlobalValuesPath != "" {
		templateArgs = append(templateArgs, fmt.Sprintf("--values=%s", c.GlobalValuesPath))
	}
	if c.Debug {
		templateArgs = append(templateArgs, "--debug")
	}

	templateArgs = append(templateArgs, chartPath)

	return runHelm(ctx, templateArgs)
}

func runHelm(ctx context.Context, args []string) error {
	return util.RunCmdWithPrinters(
		ctx,
		"helm",
		args,
		nil,
		nil,
		util.LogrusDebugPrinter("[helm]"),
		util.LogrusWarnPrinter("[helm]"),
	)
}

func getHeaderComments(contents []byte) map[string]string {
	var contentsStr string
	if len(contents) > 1000 {
		contentsStr = string(contents[0:1000])
	} else {
		contentsStr = string(contents)
	}

	lines := strings.Split(contentsStr, "\n")

	headerComments := map[string]string{}

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		} else if !strings.HasPrefix(trimmedLine, "#") {
			// End of header comments
			break
		}

		components := strings.SplitN(trimmedLine, ":", 2)
		if len(components) < 2 {
			continue
		}

		key := strings.TrimSpace(components[0][1:])

		value := strings.TrimSpace(components[1])
		value = strings.ReplaceAll(value, "\"", "")
		value = strings.ReplaceAll(value, "'", "")

		headerComments[key] = value
	}

	return headerComments
}

func getValue(input map[string]string, keys ...string) string {
	for _, key := range keys {
		if input[key] != "" {
			return input[key]
		}
	}

	return ""
}
