package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lms-tui/logger"
	"lms-tui/models"
	"lms-tui/pkg"
)

// findNonEmptyColumns returns indices of columns that have at least one non-empty cell
func findNonEmptyColumns(rows [][]string) []int {
	if len(rows) == 0 {
		return []int{}
	}

	// Find max column count
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	// Check each column for non-empty content
	nonEmptyCols := []int{}
	for col := 0; col < maxCols; col++ {
		hasContent := false
		for _, row := range rows {
			if col < len(row) && strings.TrimSpace(row[col]) != "" {
				hasContent = true
				break
			}
		}
		if hasContent {
			nonEmptyCols = append(nonEmptyCols, col)
		}
	}
	return nonEmptyCols
}

// filterEmptyRows removes rows that are completely empty
func filterEmptyRows(rows [][]string) [][]string {
	filtered := [][]string{}
	for _, row := range rows {
		hasContent := false
		for _, cell := range row {
			if strings.TrimSpace(cell) != "" {
				hasContent = true
				break
			}
		}
		if hasContent {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func NewJobDetailScreen(app *tview.Application, job models.Job, onBack func()) tview.Primitive {
	// Build the Excel file path
	filePath := fmt.Sprintf("projects/%s/Lab_%s.xlsm", job.ProjectNumber, job.ProjectNumber)

	logger.Info.Printf("Opening job detail for: %s", job.ProjectNumber)

	// Create the table for sample data
	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	// Convert Excel to JSON and log it
	jobData, err := pkg.ExcelToJSON(filePath)
	if err != nil {
		logger.Error.Printf("Failed to parse Excel file: %v", err)
		table.SetCell(0, 0, tview.NewTableCell("Error loading job data").
			SetTextColor(tcell.ColorRed).
			SetAlign(tview.AlignCenter))
		table.SetCell(1, 0, tview.NewTableCell(err.Error()).
			SetTextColor(tcell.ColorYellow))
	} else {
		// Set up table headers
		headers := []string{"Boring", "Depth", "Tests Required"}
		for col, header := range headers {
			table.SetCell(0, col, tview.NewTableCell(header).
				SetTextColor(tcell.ColorWhite).
				SetAttributes(tcell.AttrBold).
				SetAlign(tview.AlignCenter).
				SetSelectable(false).
				SetExpansion(1))
		}

		// Populate table with sample data
		for row, sample := range jobData.Samples {
			// Boring Number
			boringCell := tview.NewTableCell(sample.BoringNumber).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter).
				SetAttributes(tcell.AttrBold)
			table.SetCell(row+1, 0, boringCell)

			// Depth
			depthCell := tview.NewTableCell(sample.Depth).
				SetTextColor(tcell.ColorWhite).
				SetAlign(tview.AlignCenter)
			table.SetCell(row+1, 1, depthCell)

			// Tests (joined as comma-separated list)
			testsStr := strings.Join(sample.Tests, ", ")
			if testsStr == "" {
				testsStr = "-"
			}
			testsCell := tview.NewTableCell(testsStr).
				SetTextColor(tcell.ColorWhite).
				SetExpansion(2)
			table.SetCell(row+1, 2, testsCell)
		}

		logger.Info.Printf("Displayed %d samples in table", len(jobData.Samples))
	}

	// Job info header with data from JSON
	var headerText string
	if jobData != nil {
		headerText = fmt.Sprintf(
			"Job: %s  Project: %s\n"+
				"Engineer: %s  Date: %s  Due: %s  Total Samples: %d",
			jobData.JobNumber,
			jobData.ProjectName,
			jobData.Engineer,
			jobData.Date,
			jobData.DueDate,
			jobData.TotalSamples)
	} else {
		headerText = fmt.Sprintf("Job: %s - %s", job.ProjectNumber, job.ProjectName)
	}

	jobInfo := tview.NewTextView().
		SetText(headerText).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Instructions
	instructions := tview.NewTextView().
		SetText("Up/Down: Navigate Samples  |  +: Back to Job List").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Container
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(jobInfo, 3, 0, false).
		AddItem(table, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	container.SetBorder(true).
		SetTitle(fmt.Sprintf(" Sample Test Requirements - Job %s ", job.ProjectNumber)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite)

	// Full screen layout (minimal margins for more space)
	vertical := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).
		AddItem(container, 0, 1, true).
		AddItem(nil, 1, 0, false)

	horizontal := tview.NewFlex().
		AddItem(nil, 2, 0, false).
		AddItem(vertical, 0, 1, true).
		AddItem(nil, 2, 0, false)

	// Input capture for back navigation
	horizontal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			logger.Info.Println("Returning from job detail to view jobs")
			onBack()
			return nil
		}
		return event
	})

	return horizontal
}
