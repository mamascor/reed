package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lms-tui/logger"
	"lms-tui/models"
	"lms-tui/pkg"
)

// NewPullJobListScreen displays a list of jobs for the user to select for pulling samples
func NewPullJobListScreen(app *tview.Application, onBack func()) (tview.Primitive, *tview.Table) {
	// Dynamically discover jobs from projects folder
	jobs, err := pkg.DiscoverJobs()
	if err != nil {
		logger.Error.Printf("Failed to discover jobs: %v", err)
		jobs = []models.Job{}
	}

	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0)

	// Set headers
	headers := []string{"Project #", "Project Name", "Engineer", "Assigned", "Due Date"}
	for col, header := range headers {
		cell := tview.NewTableCell(header).
			SetTextColor(tcell.ColorWhite).
			SetAlign(tview.AlignCenter).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		table.SetCell(0, col, cell)
	}

	// Populate table with job data
	for row, job := range jobs {
		table.SetCell(row+1, 0, tview.NewTableCell(job.ProjectNumber).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))

		table.SetCell(row+1, 1, tview.NewTableCell(job.ProjectName).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(2))

		table.SetCell(row+1, 2, tview.NewTableCell(job.EngineerInitials).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))

		table.SetCell(row+1, 3, tview.NewTableCell(job.FormatDateAssigned()).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))

		table.SetCell(row+1, 4, tview.NewTableCell(job.FormatDueDate()).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))
	}

	// Handle job selection function
	selectJob := func() {
		row, _ := table.GetSelection()
		if row == 0 {
			return
		}
		selectedJob := jobs[row-1]
		logger.Info.Printf("Job selected for pulling: %s - %s", selectedJob.ProjectNumber, selectedJob.ProjectName)

		// Navigate directly to pull sample screen
		pullScreen := NewPullSampleScreen(app, selectedJob, func() {
			// Go back to pull job list screen
			pullJobScreen, pullJobTable := NewPullJobListScreen(app, onBack)
			app.SetRoot(pullJobScreen, true)
			app.SetFocus(pullJobTable)
		})
		app.SetRoot(pullScreen, true)
	}

	// Handle job selection - navigate directly to pull sample screen
	table.SetSelectedFunc(func(row, column int) {
		selectJob()
	})


	// Title text
	titleText := tview.NewTextView().
		SetText("Select Job to Pull").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorWhite)

	// Instructions text
	instructions := tview.NewTextView().
		SetText("Up/Down: Navigate  |  +: Back to LMS  |  Enter: Select Job").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorWhite).
		SetDynamicColors(true)

	// Container
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(titleText, 1, 0, false).
		AddItem(table, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	container.SetBorder(true).
		SetTitle(" Pull Job - Select Project ").
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

	horizontal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			onBack()
			return nil
		}
		return event
	})

	return horizontal, table
}
