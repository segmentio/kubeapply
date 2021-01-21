package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// KubeValidator is a struct that validates the kube configs associated with a cell config.
type KubeValidator struct{}

// ValidationResult stores the results of validating a single file.
type ValidationResult struct {
	Filename string   `json:"filename"`
	Kind     string   `json:"kind"`
	Status   string   `json:"status"`
	Errors   []string `json:"errors"`
}

// NewKubeValidator returns a new KubeValidator instance.
func NewKubeValidator() *KubeValidator {
	return &KubeValidator{}
}

// CheckYAML checks that each file ending with ".yaml" is actually parseable YAML. This
// is done separately from the kubeval checks because these errors cause the latter tool
// to not output valid JSON.
func (k *KubeValidator) CheckYAML(paths []string) error {
	for _, path := range paths {
		err := filepath.Walk(
			path,
			func(subPath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() || !strings.HasSuffix(subPath, ".yaml") {
					return nil
				}

				yamlBytes, err := ioutil.ReadFile(subPath)
				if err != nil {
					return err
				}

				reader := bytes.NewReader(yamlBytes)
				decoder := yaml.NewDecoder(reader)

				for {
					err := decoder.Decode(&map[string]interface{}{})
					if err == io.EOF {
						return nil
					} else if err != nil {
						return fmt.Errorf(
							"File %s does not contain a valid YAML map: %+v",
							subPath,
							err,
						)
					}
				}
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunKubeval runs kubeval over all files in the provided path.
func (k *KubeValidator) RunKubeval(
	ctx context.Context,
	path string,
) ([]ValidationResult, error) {
	cmd := exec.CommandContext(
		ctx,
		"kubeval",
		// Need this because otherwise manifests referring to CRDs fail. See
		// https://github.com/instrumenta/kubeval/issues/47 for associated issue on the
		// kubeval side.
		"--ignore-missing-schemas",
		"--strict",
		"-d",
		path,
		"-o",
		"json",
		"--quiet",
	)
	cmd.Env = os.Environ()

	// Ignore the error here unless we can't parse json in the output
	bytes, runErr := cmd.CombinedOutput()

	results := []ValidationResult{}

	err := json.Unmarshal(bytes, &results)
	if err != nil {
		return nil, fmt.Errorf(
			"Could not parse json from kubeval results; cmdErr: %v, jsonErr: %+v, output: %s",
			err,
			runErr,
			string(bytes),
		)
	}

	var currDir string

	// Strip off leading path so that output is more readable
	for r := 0; r < len(results); r++ {
		if strings.HasPrefix(results[r].Filename, path) {
			currPath := results[r].Filename[(len(path) + 1):]
			results[r].Filename = currPath
			currDir = filepath.Dir(currPath)
		} else {
			// If the path isn't absolute, it's relative to the last seen absolute dir
			results[r].Filename = filepath.Join(currDir, results[r].Filename)
		}
	}

	return results, nil
}
