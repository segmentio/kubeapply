package apply

import (
	"bytes"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// ResultsTextTable returns a pretty table that summarizes the results of
// a kubectl apply run.
func ResultsTextTable(results []Result) string {
	buf := &bytes.Buffer{}

	table := tablewriter.NewWriter(buf)
	table.SetHeader(
		[]string{
			"Kind",
			"Name",
			"Namespace",
			"Created",
			"Old Version",
			"New Version",
		},
	)
	table.SetAutoWrapText(false)
	table.SetColumnAlignment(
		[]int{
			tablewriter.ALIGN_LEFT,
			tablewriter.ALIGN_LEFT,
			tablewriter.ALIGN_LEFT,
			tablewriter.ALIGN_LEFT,
			tablewriter.ALIGN_LEFT,
			tablewriter.ALIGN_LEFT,
		},
	)
	table.SetBorders(
		tablewriter.Border{
			Left:   false,
			Top:    true,
			Right:  false,
			Bottom: true,
		},
	)

	createdPrinter := color.New(color.FgYellow).SprintfFunc()
	updatedPrinter := color.New(color.FgBlue).SprintfFunc()
	unchangedPrinter := color.New(color.Faint).SprintfFunc()

	var printer func(f string, a ...interface{}) string

	for _, result := range results {
		if result.IsCreated() {
			printer = createdPrinter
		} else if result.IsUpdated() {
			printer = updatedPrinter
		} else {
			printer = unchangedPrinter
		}

		table.Append(
			[]string{
				printer("%s", result.Kind),
				printer("%s", result.Name),
				printer("%s", result.Namespace),
				printer("%s", result.CreatedTimestamp()),
				printer("%s", result.OldVersion),
				printer("%s", result.NewVersion),
			},
		)
	}

	table.Render()
	return string(bytes.TrimRight(buf.Bytes(), "\n"))
}
