package diff

import (
	"bytes"
	"fmt"

	"github.com/olekukonko/tablewriter"
)

// ResultsTable returns a table that summarizes a slice of result diffs.
func ResultsTable(results []Result) string {
	buf := &bytes.Buffer{}

	table := tablewriter.NewWriter(buf)
	table.SetHeader(
		[]string{
			"Namespace",
			"Kind",
			"Name",
			"Changed Lines",
		},
	)
	table.SetAutoWrapText(false)
	table.SetColumnAlignment(
		[]int{
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

	for _, result := range results {
		var kind string
		var name string
		var namespace string

		if result.Object != nil {
			kind = result.Object.Kind
			name = result.Object.KubeMetadata.Name
			namespace = result.Object.KubeMetadata.Namespace
		} else {
			name = result.Name
		}

		table.Append(
			[]string{
				namespace,
				kind,
				name,
				fmt.Sprintf("%d", result.NumChangedLines()),
			},
		)
	}

	table.Render()
	return string(bytes.TrimRight(buf.Bytes(), "\n"))
}
