package diff

import (
	"fmt"
	"strings"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	log "github.com/sirupsen/logrus"
)

// Results contains all results from a given diff run. It's used for wrapping so that
// everything can be put in a single struct when exported by kubeapply kdiff.
type Results struct {
	Results []Result `json:"results"`
}

// Result contains the results of diffing a single object.
type Result struct {
	Object     *apply.TypedKubeObj `json:"object"`
	Name       string              `json:"name"`
	RawDiff    string              `json:"rawDiff"`
	NumAdded   int                 `json:"numAdded"`
	NumRemoved int                 `json:"numRemoved"`
}

func PrintFull(results []Result) {
	if len(results) == 0 {
		log.Infof("No diffs found")
		return
	}

	log.Infof("Diffs summary:\n%s", ResultsTable(results))
	log.Info("Raw diffs:")
	for _, result := range results {
		result.PrintRaw(true)
	}
}

func PrintSummary(results []Result) {
	if len(results) == 0 {
		log.Infof("No diffs found")
		return
	}

	log.Infof("Diffs summary:\n%s", ResultsTable(results))
}

func (r *Result) PrintRaw(useColors bool) {
	lines := strings.Split(r.RawDiff, "\n")
	for _, line := range lines {
		var prefix string
		if len(line) > 0 {
			prefix = line[0:1]
		}

		switch prefix {
		case "+":
			if useColors {
				printGreen(line)
			} else {
				fmt.Println(line)
			}
		case "-":
			if useColors {
				printRed(line)
			} else {
				fmt.Println(line)
			}
		default:
			if len(line) > 0 {
				fmt.Println(line)
			}
		}
	}
}

func (r *Result) ClippedRawDiff(maxLen int) string {
	if len(r.RawDiff) > maxLen {
		return fmt.Sprintf(
			"%s\n... (%d chars omitted)",
			r.RawDiff[0:maxLen],
			len(r.RawDiff)-maxLen,
		)
	}
	return r.RawDiff
}

func (r *Result) NumChangedLines() int {
	if r.NumAdded > r.NumRemoved {
		return r.NumAdded
	}
	return r.NumRemoved
}

func printRed(line string) {
	// Use escape codes directly instead of color library to force colors even if we're
	// not in a terminal
	fmt.Printf("\033[91m%s\033[0m\n", line)
}

func printGreen(line string) {
	// Use escape codes directly instead of color library to force colors even if we're
	// not in a terminal
	fmt.Printf("\033[92m%s\033[0m\n", line)
}
