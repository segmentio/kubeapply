package diff

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	log "github.com/sirupsen/logrus"
)

const (
	maxLineLen = 256
)

// DiffKube processes the results of a kubectl diff call in place of the default 'diff'
// command.
func DiffKube(oldRoot string, newRoot string, shortDiff bool) ([]Result, error) {
	oldNames, err := walkPaths(oldRoot)
	if err != nil {
		return nil, err
	}

	newNames, err := walkPaths(newRoot)
	if err != nil {
		return nil, err
	}

	allNames := map[string]struct{}{}
	for name := range oldNames {
		allNames[name] = struct{}{}
	}
	for name := range newNames {
		allNames[name] = struct{}{}
	}

	allNamesSlice := []string{}
	for name := range allNames {
		allNamesSlice = append(allNamesSlice, name)
	}

	sort.Slice(allNamesSlice, func(a, b int) bool {
		return allNamesSlice[a] < allNamesSlice[b]
	})

	results := []Result{}

	for _, name := range allNamesSlice {
		_, oldOk := oldNames[name]
		_, newOk := newNames[name]

		var diffResult *Result

		if oldOk && newOk {
			diffResult, err = evalDiffs(
				name,
				oldRoot,
				name,
				newRoot,
				name,
				shortDiff,
			)
		} else if oldOk {
			diffResult, err = evalDiffs(
				name,
				oldRoot,
				name,
				newRoot,
				"",
				shortDiff,
			)
		} else {
			diffResult, err = evalDiffs(
				name,
				oldRoot,
				"",
				newRoot,
				name,
				shortDiff,
			)
		}

		if err != nil {
			return nil, err
		}

		if diffResult != nil && diffResult.RawDiff != "" {
			results = append(
				results,
				*diffResult,
			)
		}
	}

	return results, nil
}

func walkPaths(root string) (map[string]struct{}, error) {
	relPaths := map[string]struct{}{}

	err := filepath.Walk(
		root,
		func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			relPath, err := filepath.Rel(root, subPath)
			if err != nil {
				return err
			}

			relPaths[relPath] = struct{}{}
			return nil
		},
	)

	return relPaths, err
}

func evalDiffs(
	name string,
	oldRoot string,
	oldName string,
	newRoot string,
	newName string,
	shortDiff bool,
) (*Result, error) {
	var oldLines []string
	var newLines []string
	var oldHash string
	var newHash string
	var obj *apply.TypedKubeObj
	var err error

	if oldName != "" {
		oldPath := filepath.Join(oldRoot, oldName)
		oldLines, oldHash, err = getFileLines(oldPath, shortDiff)
		if err != nil {
			return nil, err
		}
		obj, err = getFileObj(oldPath)
		if err != nil {
			log.Warnf("Error parsing path %s: %+v", oldPath, err)
		}
	}

	if newName != "" {
		newPath := filepath.Join(newRoot, newName)
		newLines, newHash, err = getFileLines(newPath, shortDiff)
		if err != nil {
			return nil, err
		}

		// If we already got the object, don't bother trying to get it again since
		// it's unlikely that the top-level fields (name, namespace, type, etc.) have
		// been changed.
		if obj == nil {
			obj, err = getFileObj(newPath)
			if err != nil {
				log.Warnf("Error parsing path %s: %+v", newPath, err)
			}
		}
	}

	if oldHash == newHash {
		return nil, nil
	}

	diff := difflib.UnifiedDiff{
		A:        oldLines,
		B:        newLines,
		FromFile: fmt.Sprintf("Server:%s", oldName),
		ToFile:   fmt.Sprintf("Local:%s", newName),
		Context:  3,
	}

	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return nil, err
	}

	numAdded, numRemoved := diffCounts(diffStr)

	return &Result{
		Object:     obj,
		Name:       name,
		RawDiff:    diffStr,
		NumAdded:   numAdded,
		NumRemoved: numRemoved,
	}, nil
}

func getFileLines(path string, shortDiff bool) ([]string, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	lines := []string{}

	// Hash the file contents so we can avoid diffing files with the same content.
	h := sha1.New()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	insideManagedFields := false
	insideAnnotation := false
	insideLabels := false
	for scanner.Scan() {
		keep := true
		line := scanner.Text()
		log.Debug("line: ", line)

		// Skip over managedFields chunk in metadata since it's constantly
		// changing and causing spurious diffs.
		if strings.HasPrefix(line, "  managedFields:") {
			insideManagedFields = true
			keep = false
		} else if insideManagedFields {
			if !(strings.HasPrefix(line, "  -") || strings.HasPrefix(line, "   ")) {
				insideManagedFields = false
			} else {
				keep = false
			}
		}

		if shortDiff && keep {
			trimedLine := strings.TrimLeft(line, " ")

			// Skip over k2 annotations chunk in metadata since it's constantly
			// changing and causing users to miss important changes.
			if strings.HasPrefix(trimedLine, "annotations:") {
				insideAnnotation = true
				keep = false
			} else if insideAnnotation {
				log.Debug("leading spaces: ", strconv.Itoa(countLeadingSpaces(line)))
				if countLeadingSpaces(line) == 2 {
					log.Debug("outside annotations: ", line)
					insideAnnotation = false
				} else {

					keep = false

				}
			}

			// Skip over k2 labels chunk in metadata since it's constantly
			// changing and causing users to miss important changes.
			if strings.HasPrefix(trimedLine, "labels") {
				insideLabels = true
				keep = true
			} else if insideLabels {
				if countLeadingSpaces(line) == 2 {
					log.Debug("outside labels: ", line)
					insideLabels = false
				} else {
					if strings.HasPrefix(trimedLine, "k2.segment.com") || strings.HasPrefix(trimedLine, "app.kubernetes.io/instance:") || strings.HasPrefix(trimedLine, "version: ") {
						keep = false
					}
				}
			}

			// Skip over generation in metadata since it's constantly
			// changing and causing users to miss important changes.
			if strings.HasPrefix(trimedLine, "generation") {
				keep = false
			}
		}

		if keep {
			if len(line) > maxLineLen {
				// Trim very long lines
				line = fmt.Sprintf(
					"%s... (%d chars omitted)",
					line[0:maxLineLen],
					len(line)-maxLineLen,
				)
			}

			lines = append(lines, line+"\n")
			h.Write([]byte(line))
		}
	}

	return lines, fmt.Sprintf("%x", h.Sum(nil)), scanner.Err()
}

func countLeadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func getFileObj(path string) (*apply.TypedKubeObj, error) {
	obj := apply.TypedKubeObj{}

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(contents, &obj); err != nil {
		return nil, err
	}

	return &obj, nil
}

func diffCounts(diffStr string) (int, int) {
	numAdded := 0
	numRemoved := 0

	lines := strings.Split(diffStr, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+ ") {
			numAdded++
		} else if strings.HasPrefix(line, "- ") {
			numRemoved++
		}
	}

	return numAdded, numRemoved
}
