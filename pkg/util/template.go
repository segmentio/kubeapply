package util

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/ghodss/yaml"
)

var extraTemplateFuncs = template.FuncMap{
	"urlEncode": url.QueryEscape,
	"toYaml":    toYaml,
}

// ApplyTemplate runs golang templating on all files in the provided path,
// replacing them in-place with their templated versions.
func ApplyTemplate(
	dir string,
	data interface{},
	deleteSources bool,
	strict bool,
) error {
	return filepath.Walk(
		dir,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() || !strings.Contains(subPath, ".gotpl.") {
				return nil
			}

			outFile, err := os.Create(
				strings.ReplaceAll(subPath, ".gotpl.", "."),
			)
			if err != nil {
				return err
			}
			defer outFile.Close()

			err = applyTemplateFile(subPath, data, true, strict, outFile)
			if err != nil {
				return err
			}

			if deleteSources {
				err := os.Remove(subPath)
				if err != nil {
					return err
				}
			}

			return nil
		},
	)
}

func applyTemplateFile(
	path string,
	data interface{},
	allowContents bool,
	strict bool,
	out io.Writer,
) error {
	templateFuncs := sprig.TxtFuncMap()

	for key, value := range extraTemplateFuncs {
		templateFuncs[key] = value
	}

	if allowContents {
		templateFuncs["fileContents"] = fileContentsGenerator(path, data, strict)
		templateFuncs["configMapEntry"] = configMapEntryGenerator(path, data, strict)
		templateFuncs["configMapEntries"] = configMapEntriesGenerator(path, data, strict)
	}

	tmpl := template.New(filepath.Base(path)).Funcs(templateFuncs)
	if strict {
		tmpl = tmpl.Option("missingkey=error")
	}

	var err error
	tmpl, err = tmpl.ParseFiles(path)
	if err != nil {
		return err
	}

	return tmpl.Execute(out, data)
}

func fileContentsGenerator(
	templatePath string,
	data interface{},
	strict bool,
) func(string) (string, error) {
	return func(relPath string) (string, error) {
		configPath := filepath.Join(
			filepath.Dir(templatePath),
			relPath,
		)
		buf := &bytes.Buffer{}
		err := applyTemplateFile(configPath, data, false, strict, buf)
		if err != nil {
			return "", err
		}

		return strings.TrimSpace(string(buf.Bytes())), nil
	}
}

func configMapEntryGenerator(
	templatePath string,
	data interface{},
	strict bool,
) func(string) (string, error) {
	return func(relPath string) (string, error) {
		configPath := filepath.Join(
			filepath.Dir(templatePath),
			relPath,
		)
		buf := &bytes.Buffer{}
		err := applyTemplateFile(configPath, data, false, strict, buf)
		if err != nil {
			return "", err
		}

		contentLines := strings.Split(
			strings.TrimSpace(string(buf.Bytes())),
			"\n",
		)

		outputLines := []string{
			fmt.Sprintf("  %s: |", filepath.Base(relPath)),
		}
		for _, line := range contentLines {
			if len(line) > 0 {
				outputLines = append(outputLines, fmt.Sprintf("    %s", line))
			} else {
				outputLines = append(outputLines, "")
			}
		}

		return strings.Join(outputLines, "\n"), nil
	}
}

func configMapEntriesGenerator(
	templatePath string,
	data interface{},
	strict bool,
) func(string) (string, error) {
	return func(relPath string) (string, error) {
		outputLines := []string{}

		dirPath := filepath.Join(
			filepath.Dir(templatePath),
			relPath,
		)

		dirFiles, err := ioutil.ReadDir(dirPath)
		if err != nil {
			return "", err
		}

		for _, dirFile := range dirFiles {
			if dirFile.IsDir() {
				continue
			} else if dirFile.Name()[0] == '.' {
				// Ignore "dot files"
				continue
			}

			configPath := filepath.Join(
				dirPath,
				dirFile.Name(),
			)
			buf := &bytes.Buffer{}
			err := applyTemplateFile(configPath, data, false, strict, buf)
			if err != nil {
				return "", err
			}

			contentLines := strings.Split(
				strings.TrimSpace(string(buf.Bytes())),
				"\n",
			)

			outputLines = append(
				outputLines,
				fmt.Sprintf(
					"  %s: |",
					strings.ReplaceAll(dirFile.Name(), ".gotpl.", "."),
				),
			)
			for _, line := range contentLines {
				if len(line) > 0 {
					outputLines = append(outputLines, fmt.Sprintf("    %s", line))
				} else {
					outputLines = append(outputLines, "")
				}
			}
		}

		return strings.Join(outputLines, "\n"), nil
	}
}

func toYaml(input interface{}) (string, error) {
	bytes, err := yaml.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
