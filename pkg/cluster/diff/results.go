package diff

import (
	"fmt"
	"strings"

	"github.com/segmentio/kubeapply/pkg/cluster/apply"
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

func (r *Result) Print(useColors bool) {
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
