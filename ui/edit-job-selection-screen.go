package ui

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lms-tui/logger"
	"lms-tui/models"
	"lms-tui/pkg"
)

func NewEditJobSelectionScreen(app *tview.Application, onBack func()) (tview.Primitive, *tview.Table) {
	// Discover jobs that have been pulled (have backup data)
	jobs, err := pkg.DiscoverJobs()
	if err != nil {
		logger.Error.Printf("Failed to discover jobs: %v", err)
	}

	// Filter jobs that have backup data
	jobsWithSamples := []struct {
		Job          models.Job
		SampleCount  int
	}{}

	for _, job := range jobs {
		backupFile := fmt.Sprintf("ex_project/%s/backup.json", job.ProjectNumber)
		if _, err := os.Stat(backupFile); err == nil {
			// Backup file exists, load it to get sample count
			backupData, err := pkg.LoadBackupData(backupFile)
			if err == nil && len(backupData.Samples) > 0 {
				jobsWithSamples = append(jobsWithSamples, struct {
					Job          models.Job
					SampleCount  int
				}{
					Job:         job,
					SampleCount: len(backupData.Samples),
				})
			}
		}
	}

	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	// Set headers
	headers := []string{"Job #", "Project Name", "Samples"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, col, cell)
	}

	// Populate table
	if len(jobsWithSamples) == 0 {
		table.SetCell(1, 0, tview.NewTableCell("No jobs with samples found").
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter))
	} else {
		for row, jobInfo := range jobsWithSamples {
			table.SetCell(row+1, 0, tview.NewTableCell(jobInfo.Job.ProjectNumber).
				SetAlign(tview.AlignCenter).
				SetTextColor(tcell.ColorWhite))
			table.SetCell(row+1, 1, tview.NewTableCell(jobInfo.Job.ProjectName).
				SetTextColor(tcell.ColorWhite).
				SetExpansion(2))
			table.SetCell(row+1, 2, tview.NewTableCell(fmt.Sprintf("%d", jobInfo.SampleCount)).
				SetAlign(tview.AlignCenter).
				SetTextColor(tcell.ColorWhite))
		}
	}

	// Handle job selection
	table.SetSelectedFunc(func(row, column int) {
		if row == 0 || len(jobsWithSamples) == 0 {
			return
		}
		selectedJobInfo := jobsWithSamples[row-1]
		logger.Info.Printf("Selected job %s for editing samples", selectedJobInfo.Job.ProjectNumber)

		// Navigate to edit samples screen
		editSamplesScreen := NewEditSamplesScreen(app, selectedJobInfo.Job, func() {
			// Go back to job selection
			editJobScreen, editJobTable := NewEditJobSelectionScreen(app, onBack)
			app.SetRoot(editJobScreen, true)
			app.SetFocus(editJobTable)
		})
		app.SetRoot(editSamplesScreen, true)
	})

	// Instructions
	instructions := tview.NewTextView().
		SetText("Up/Down: Navigate  |  Enter: Select Job  |  +: Back to LMS").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	// Container
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	container.SetBorder(true).
		SetTitle(" Select Job to Edit Samples ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite)

	// Center it
	vertical := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(container, 0, 4, true).
		AddItem(nil, 0, 1, false)

	horizontal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(vertical, 0, 3, true).
		AddItem(nil, 0, 1, false)

	// Input capture
	horizontal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			onBack()
			return nil
		}
		return event
	})

	return horizontal, table
}
