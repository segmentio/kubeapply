package diff

import (
	"bytes"
	"fmt"

	"github.com/olekukonko/tablewriter"
)

func ResultsTable(results *Results) string {
	buf := &bytes.Buffer{}

	table := tablewriter.NewWriter(buf)
	table.SetHeader(
		[]string{
			"Kind",
			"Name",
			"Namespace",
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

	for _, result := range results.Results {
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
				kind,
				name,
				namespace,
				fmt.Sprintf("%d", max(result.NumAdded, result.NumRemoved)),
			},
		)
	}

	table.Render()
	return string(bytes.TrimRight(buf.Bytes(), "\n"))
}

func max(a int, b int) int {
	if a < b {
		return b
	}
	return a
}
