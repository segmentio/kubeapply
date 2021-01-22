package diff

import (
	"fmt"
	"strings"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
	log "github.com/sirupsen/logrus"
)

type Results struct {
	Results []Result `json:"results"`
}

type Result struct {
	Object     *apply.TypedKubeObj `json:"object"`
	Name       string              `json:"name"`
	RawDiff    string              `json:"rawDiff"`
	NumAdded   int                 `json:"numAdded"`
	NumRemoved int                 `json:"numRemoved"`
}

func (r *Results) PrintFull() {
	if len(r.Results) == 0 {
		log.Infof("No diffs found")
		return
	}

	log.Infof("Diffs summary:\n%s", ResultsTable(r))
	log.Info("Raw diffs:")
	for _, result := range r.Results {
		result.PrintRaw(true)
	}
}

func (r *Results) PrintSummary() {
	if len(r.Results) == 0 {
		log.Infof("No diffs found")
		return
	}

	log.Infof("Diffs summary:\n%s", ResultsTable(r))
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
