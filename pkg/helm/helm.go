package helm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/segmentio/kubeapply/pkg/util"
	log "github.com/sirupsen/logrus"
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
	chartsPath string
	namespace  string
	valuesPath string
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

			var namespace string
			if s := strings.Split(strings.TrimPrefix(subPath, configPath), "/"); len(s) > 1 {
				namespace = s[1]
			}

			if namespace == "" {
				return fmt.Errorf("Resource %s is defined out of a namespace.", subPath)
			}

			if strings.HasSuffix(subPath, ".helm.yaml") {
				helmContexts = append(
					helmContexts,
					helmContext{
						namespace:  namespace,
						chartsPath: chartsPath,
						valuesPath: subPath,
					},
				)
				return nil
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
				errChan <- os.RemoveAll(hctx.valuesPath)
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
		"Processing helm chart with values %s in %s",
		filepath.Base(hctx.valuesPath),
		hctx.namespace,
	)

	// Skip over empty placeholder values files
	contents, err := ioutil.ReadFile(hctx.valuesPath)
	if err != nil {
		return err
	}
	trimmedContents := bytes.TrimSpace(contents)

	if len(trimmedContents) == 0 {
		log.Warnf(
			"Values file %s in %s is empty, skipping",
			filepath.Base(hctx.valuesPath),
			hctx.namespace,
		)
		return nil
	}

	var contentsMap map[string]interface{}
	if err := yaml.Unmarshal(trimmedContents, &contentsMap); err != nil {
		return fmt.Errorf(
			"File %s is not valid YAML. If it's a template, be sure to add '.gotpl' in its name",
			hctx.valuesPath,
		)
	}

	headerComments := getHeaderComments(trimmedContents)

	if getValue(headerComments, "kubeapply__disabled", "disabled") == "true" {
		log.Warnf(
			"Skipping values file %s in %s because it has disabled: true",
			filepath.Base(hctx.valuesPath),
			hctx.namespace,
		)
		return nil
	}

	var localChartsPath string
	chartsOverride := getValue(headerComments, "kubeapply__chart", "kubeapply__charts", "charts")

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

		chartsSubdir := getValue(headerComments, "kubeapply__chartsSubdir")
		if chartsSubdir != "" {
			localChartsPath = filepath.Join(localChartsPath, chartsSubdir)
		}

		log.Debugf("Using charts override path of %s", localChartsPath)
	} else {
		localChartsPath = hctx.chartsPath
	}

	baseName := filepath.Base(hctx.valuesPath)
	nameComponents := strings.Split(baseName, ".")

	chartNameOverride := getValue(headerComments, "kubeapply__chartName", "chartName")
	var chartNamePath string
	if chartNameOverride != "" {
		chartNamePath = chartNameOverride
	} else {
		chartNamePath = nameComponents[0]
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

	err = runHelm(ctx, depArgs)
	if err != nil {
		return err
	}

	releaseName := getValue(headerComments, "kubeapply__releaseName", "releaseName")

	templateArgs := []string{
		"template",
		fmt.Sprintf("--namespace=%s", hctx.namespace),
		fmt.Sprintf("--values=%s", hctx.valuesPath),
		fmt.Sprintf("--output-dir=%s", filepath.Dir(hctx.valuesPath)),
	}
	if releaseName != "" {
		templateArgs = append(templateArgs, fmt.Sprintf("--name-template=%s", releaseName))
	}
	if c.GlobalValuesPath != "" {
		templateArgs = append(templateArgs, fmt.Sprintf("--values=%s", c.GlobalValuesPath))
	}
	if c.Debug {
		templateArgs = append(templateArgs, "--debug")
	}

	nameOverride := getValue(headerComments, "kubeapply__releaseName")

	if nameOverride != "" {
		templateArgs = append(
			templateArgs,
			fmt.Sprintf("--name-template=%s", nameOverride),
		)
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
