package validation

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func ResultTable(
	result ResourceResult,
	clusterName string,
	baseDir string,
	verbose bool,
) string {
	buf := &bytes.Buffer{}

	table := tablewriter.NewWriter(buf)
	table.SetHeader(
		[]string{
			"Property",
			"Value",
		},
	)
	table.SetAutoWrapText(false)
	table.SetColumnAlignment(
		[]int{
			tablewriter.ALIGN_RIGHT,
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

	if clusterName != "" {
		table.Append(
			[]string{
				"cluster",
				clusterName,
			},
		)
	}

	var displayPath string

	relPath, err := filepath.Rel(baseDir, result.Resource.Path)
	if err != nil {
		displayPath = result.Resource.Path
	} else {
		displayPath = relPath
	}

	table.Append(
		[]string{
			"path",
			displayPath,
		},
	)
	table.Append(
		[]string{
			"resource",
			result.Resource.PrettyName(),
		},
	)

	errorPrinter := color.New(color.FgHiRed).SprintfFunc()
	warnPrinter := color.New(color.FgHiYellow).SprintfFunc()
	standardPrinter := fmt.Sprintf

	for _, checkResult := range result.CheckResults {
		if !verbose && (checkResult.Status != StatusError &&
			checkResult.Status != StatusInvalid &&
			checkResult.Status != StatusWarning) {
			continue
		}

		reasonLines := []string{checkResult.Message}
		for r, reason := range checkResult.Reasons {
			reasonLines = append(reasonLines, fmt.Sprintf("(%d) %s", r+1, reason))
		}

		table.Append(
			[]string{
				"checkType",
				string(checkResult.CheckType),
			},
		)
		table.Append(
			[]string{
				"checkName",
				checkResult.CheckName,
			},
		)

		var printer func(f string, a ...interface{}) string
		switch checkResult.Status {
		case StatusError, StatusInvalid:
			printer = errorPrinter
		case StatusWarning:
			printer = warnPrinter
		default:
			printer = standardPrinter
		}

		table.Append(
			[]string{
				"checkStatus",
				printer(string(checkResult.Status)),
			},
		)

		table.Append(
			[]string{
				"checkMessage",
				strings.Join(reasonLines, "\n"),
			},
		)
	}

	table.Render()
	return string(bytes.TrimRight(buf.Bytes(), "\n"))
}

func WriteResultsCSV(
	results []ResourceResult,
	clusterName string,
	baseDir string,
	outputPath string,
) error {
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	csvWriter := csv.NewWriter(out)

	err = csvWriter.Write(
		[]string{
			"cluster",
			"path",
			"resource",
		},
	)
	if err != nil {
		return err
	}

	for _, result := range results {
		var displayPath string

		relPath, err := filepath.Rel(baseDir, result.Resource.Path)
		if err != nil {
			displayPath = result.Resource.Path
		} else {
			displayPath = relPath
		}

		err = csvWriter.Write(
			[]string{
				clusterName,
				displayPath,
				result.Resource.PrettyName(),
			},
		)
		if err != nil {
			return err
		}
	}

	csvWriter.Flush()
	return csvWriter.Error()
}
