package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lms-tui/logger"
	"lms-tui/models"
	"lms-tui/pkg"
)

func NewViewJobScreen(app *tview.Application, onBack func()) (tview.Primitive, *tview.Table) {

	// Dynamically discover jobs from projects folder
	jobs, err := pkg.DiscoverJobs()
	if err != nil {
		logger.Error.Printf("Failed to discover jobs: %v", err)
		jobs = []models.Job{}
	}

	table := tview.NewTable().
		SetBorders(true).
		SetSelectable(true, false).
		SetFixed(1, 0) // Fix header row so it doesn't scroll

	// Set headers with better styling
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
		// Project Number
		table.SetCell(row+1, 0, tview.NewTableCell(job.ProjectNumber).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))

		// Project Name
		table.SetCell(row+1, 1, tview.NewTableCell(job.ProjectName).
			SetTextColor(tcell.ColorWhite).
			SetExpansion(2)) // Give more space to project name

		// Engineer Initials
		table.SetCell(row+1, 2, tview.NewTableCell(job.EngineerInitials).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))

		// Date Assigned
		table.SetCell(row+1, 3, tview.NewTableCell(job.FormatDateAssigned()).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))

		// Due Date
		table.SetCell(row+1, 4, tview.NewTableCell(job.FormatDueDate()).
			SetAlign(tview.AlignCenter).
			SetTextColor(tcell.ColorWhite))
	}

	// Handle job selection function
	selectJob := func() {
		row, _ := table.GetSelection()
		// Skip header row
		if row == 0 {
			return
		}
		// Get the selected job
		selectedJob := jobs[row-1]
		logger.Info.Printf("Job selected: %s - %s", selectedJob.ProjectNumber, selectedJob.ProjectName)

		// Navigate to job detail screen
		detailScreen := NewJobDetailScreen(app, selectedJob, func() {
			// Go back to view jobs screen
			viewJobScreen, viewJobTable := NewViewJobScreen(app, onBack)
			app.SetRoot(viewJobScreen, true)
			app.SetFocus(viewJobTable)
		})
		app.SetRoot(detailScreen, true)
	}

	// Handle job selection
	table.SetSelectedFunc(func(row, column int) {
		selectJob()
	})


	// Title text
	titleText := tview.NewTextView().
		SetText("View Jobs").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorWhite)

	// Instructions text
	instructions := tview.NewTextView().
		SetText("Up/Down: Navigate  |  +: Back to Home  |  Enter: Select").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorWhite).
		SetDynamicColors(true)

	// Container with title, table, and instructions
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(titleText, 1, 0, false).
		AddItem(table, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	container.SetBorder(true).
		SetTitle(" Job Management System ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite)

	// Center it with dynamic sizing
	vertical := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(container, 0, 4, true). // Takes 4/6 of vertical space
		AddItem(nil, 0, 1, false)

	horizontal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(vertical, 0, 3, true). // Takes 3/5 of horizontal space
		AddItem(nil, 0, 1, false)

	// Input capture for navigation
	horizontal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			onBack()
			return nil
		}
		return event
	})

	return horizontal, table
}