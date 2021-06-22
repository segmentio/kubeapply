package util

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/ghodss/yaml"
)

var (
	extraTemplateFuncs = template.FuncMap{
		"lookup":     lookup,
		"pathLookup": pathLookup,
		"toYaml":     toYaml,
		"urlEncode":  url.QueryEscape,
		"merge":      merge,
	}
)

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
				// Wrap the error so that we can provide more context
				return fmt.Errorf("Error expanding path %s: %+v", subPath, err)
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

// lookup does a dot-separated path lookup on the input map. If a key on the path is
// not found, it returns nil. If the input or any of its children on the lookup path is not
// a map, it returns an error.
func lookup(input interface{}, path string) (interface{}, error) {
	obj := reflect.ValueOf(input)
	components := strings.Split(path, ".")

	for i := 0; i < len(components); {
		switch obj.Kind() {
		case reflect.Map:
			obj = obj.MapIndex(reflect.ValueOf(components[i]))
			i++
		case reflect.Ptr, reflect.Interface:
			if obj.IsNil() {
				return nil, nil
			}

			// Get the thing being pointed to or interfaced, don't advance index
			obj = obj.Elem()
		default:
			if obj.IsValid() {
				// An object was found, but it's not a map. Return an error.
				return nil, fmt.Errorf(
					"Tried to traverse a value that's not a map (kind=%s)",
					obj.Kind(),
				)
			}

			// An intermediate key wasn't found
			return nil, nil
		}
	}

	if !obj.IsValid() {
		// The last key wasn't found
		return nil, nil
	}
	return obj.Interface(), nil
}

// pathLookup is the same as lookup, but with the arguments flipped.
func pathLookup(path string, input interface{}) (interface{}, error) {
	return lookup(input, path)
}

func toYaml(input interface{}) (string, error) {
	bytes, err := yaml.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// merge recursively merges one or more string-keyed maps into one map. It
// always returns a map[string]interface{} or an error.
func merge(values ...interface{}) (interface{}, error) {
	merged := map[string]interface{}{}
	var err error

	for i, val := range values {
		switch v := val.(type) {
		case map[string]interface{}:
			merged, err = mergeMap("", merged, v)
		case nil:
			continue
		default:
			err = fmt.Errorf("Argument %d: Expected map[string]interface{} or nil, got %s", i, typeLabel(val))
		}

		if err != nil {
			return nil, err
		}
	}

	return merged, nil
}

func mergeMap(path string, l, r map[string]interface{}) (map[string]interface{}, error) {
	var err error

	for k, v := range r {
		rMap, ok := v.(map[string]interface{})
		if !ok || l[k] == nil {
			l[k] = v
			continue
		}

		lMap, ok := l[k].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%s: Expected map[string]interface{}, got %s", joinPath(path, k), typeLabel(l[k]))
		}

		l[k], err = mergeMap(joinPath(path, k), lMap, rMap)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func typeLabel(v interface{}) string {
	typ := reflect.TypeOf(v)
	if typ == nil {
		return "nil"
	}
	return typ.String()
}

func joinPath(l, r string) string {
	if l == "" {
		return r
	}
	return l + "." + r
}
