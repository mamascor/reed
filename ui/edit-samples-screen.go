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

func NewEditSamplesScreen(app *tview.Application, job models.Job, onBack func()) tview.Primitive {
	logger.Info.Printf("Opening edit samples screen for Job: %s", job.ProjectNumber)

	// Load backup data
	backupFile := fmt.Sprintf("ex_project/%s/backup.json", job.ProjectNumber)
	backupData, err := pkg.LoadBackupData(backupFile)
	if err != nil {
		logger.Error.Printf("Failed to load backup data: %v", err)
		// Show error modal and go back
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Failed to load backup data:\n%v\n\nPress Enter to go back", err)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				onBack()
			})
		modal.SetBackgroundColor(tcell.ColorBlack)
		return modal
	}

	if len(backupData.Samples) == 0 {
		// No samples to edit
		modal := tview.NewModal().
			SetText("No samples found to edit.\n\nPress Enter to go back").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				onBack()
			})
		modal.SetBackgroundColor(tcell.ColorBlack)
		return modal
	}

	// Create table to show all samples
	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, false).
		SetFixed(1, 0)

	// Set headers
	headers := []string{"#", "Boring", "Depth", "Can #", "Can Wt", "Wet Wt", "Suction Can"}
	for col, header := range headers {
		table.SetCell(0, col, tview.NewTableCell(header).
			SetTextColor(tcell.ColorYellow).
			SetAlign(tview.AlignCenter).
			SetSelectable(false))
	}

	// Populate table with samples
	for i, sample := range backupData.Samples {
		row := i + 1
		table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", i+1)).SetAlign(tview.AlignCenter))
		table.SetCell(row, 1, tview.NewTableCell(sample.BoringNumber).SetAlign(tview.AlignCenter))
		table.SetCell(row, 2, tview.NewTableCell(sample.Depth).SetAlign(tview.AlignCenter))
		table.SetCell(row, 3, tview.NewTableCell(sample.CanNumber).SetAlign(tview.AlignCenter))
		table.SetCell(row, 4, tview.NewTableCell(sample.CanWeight).SetAlign(tview.AlignCenter))
		table.SetCell(row, 5, tview.NewTableCell(sample.WetWeight).SetAlign(tview.AlignCenter))
		table.SetCell(row, 6, tview.NewTableCell(sample.SuctionCanNo).SetAlign(tview.AlignCenter))
	}

	table.SetBorder(true).
		SetTitle(" Select Sample to Edit (↑/↓ to navigate, Enter to edit) ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// Info text
	infoText := tview.NewTextView().
		SetText(fmt.Sprintf("Job %s - %d samples in backup\n\nUse ↑/↓ to select, Enter to edit, + to go back",
			job.ProjectNumber, len(backupData.Samples))).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetBackgroundColor(tcell.ColorBlack)

	// Container
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(infoText, 3, 0, false).
		AddItem(table, 0, 1, true)

	// Handle selection
	table.SetSelectedFunc(func(row, col int) {
		if row == 0 {
			return // Header row
		}
		selectedIndex := row - 1
		if selectedIndex >= 0 && selectedIndex < len(backupData.Samples) {
			sample := backupData.Samples[selectedIndex]
			showEditSampleModal(app, job, sample, selectedIndex, backupData, table, container)
		}
	})

	container.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Samples - Job %s ", job.ProjectNumber)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// Handle back navigation
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			onBack()
			return nil
		}
		return event
	})

	return container
}

func showEditSampleModal(app *tview.Application, job models.Job, sample pkg.SampleBackupData,
	sampleIndex int, backupData *pkg.BackupData, table *tview.Table, container tview.Primitive) {

	// Create edit form
	form := tview.NewForm()
	form.AddInputField("Can #", sample.CanNumber, 25, nil, nil)
	form.AddInputField("Can Weight (g)", sample.CanWeight, 25, nil, nil)
	form.AddInputField("Wet Weight (g)", sample.WetWeight, 25, nil, nil)
	form.AddInputField("Suction Can #", sample.SuctionCanNo, 25, nil, nil)

	form.AddButton("Save Changes", func() {
		// Get updated values
		newCanNo := strings.TrimSpace(form.GetFormItemByLabel("Can #").(*tview.InputField).GetText())
		newCanWeight := strings.TrimSpace(form.GetFormItemByLabel("Can Weight (g)").(*tview.InputField).GetText())
		newWetWeight := strings.TrimSpace(form.GetFormItemByLabel("Wet Weight (g)").(*tview.InputField).GetText())
		newSuctionCanNo := strings.TrimSpace(form.GetFormItemByLabel("Suction Can #").(*tview.InputField).GetText())

		// Validate
		if newCanNo == "" || newCanWeight == "" || newWetWeight == "" {
			showErrorModal(app, "Can #, Can Weight, and Wet Weight are required", table, container)
			return
		}

		logger.Info.Printf("Updating sample %d: %s|%s - Can#: %s->%s, CanWt: %s->%s, WetWt: %s->%s, SuctionCan: %s->%s",
			sampleIndex+1, sample.BoringNumber, sample.Depth,
			sample.CanNumber, newCanNo,
			sample.CanWeight, newCanWeight,
			sample.WetWeight, newWetWeight,
			sample.SuctionCanNo, newSuctionCanNo)

		// Update backup data
		backupData.Samples[sampleIndex].CanNumber = newCanNo
		backupData.Samples[sampleIndex].CanWeight = newCanWeight
		backupData.Samples[sampleIndex].WetWeight = newWetWeight
		backupData.Samples[sampleIndex].SuctionCanNo = newSuctionCanNo

		// Save backup
		backupFile := fmt.Sprintf("ex_project/%s/backup.json", job.ProjectNumber)
		if err := pkg.SaveBackupDataToFile(backupData, backupFile); err != nil {
			logger.Error.Printf("Failed to save backup: %v", err)
			showErrorModal(app, fmt.Sprintf("Failed to save backup:\n%v", err), table, container)
			return
		}

		// Update Excel file - moisture data
		moistureWriter, err := pkg.InitMoistureTestFile(job.ProjectNumber, job.LabFilePath)
		if err != nil {
			logger.Error.Printf("Failed to initialize moisture writer: %v", err)
			showErrorModal(app, fmt.Sprintf("Failed to update Excel:\n%v", err), table, container)
			return
		}
		defer moistureWriter.Close()

		err = moistureWriter.WriteMoistureSample(sample.BoringNumber, sample.Depth, newCanNo, newCanWeight, newWetWeight)
		if err != nil {
			logger.Error.Printf("Failed to write moisture sample: %v", err)
			showErrorModal(app, fmt.Sprintf("Failed to update moisture data:\n%v", err), table, container)
			return
		}

		// Update Excel file - suction data if present
		if newSuctionCanNo != "" {
			suctionWriter, err := pkg.InitSoilSuctionFile(job.ProjectNumber, moistureWriter.GetFile())
			if err != nil {
				logger.Error.Printf("Failed to initialize suction writer: %v", err)
			} else {
				defer suctionWriter.Close()
				err = suctionWriter.WriteSoilSuctionSample(sample.BoringNumber, sample.Depth, newSuctionCanNo)
				if err != nil {
					logger.Error.Printf("Failed to write suction sample: %v", err)
				}
			}
		}

		// Update table display
		table.SetCell(sampleIndex+1, 3, tview.NewTableCell(newCanNo).SetAlign(tview.AlignCenter))
		table.SetCell(sampleIndex+1, 4, tview.NewTableCell(newCanWeight).SetAlign(tview.AlignCenter))
		table.SetCell(sampleIndex+1, 5, tview.NewTableCell(newWetWeight).SetAlign(tview.AlignCenter))
		table.SetCell(sampleIndex+1, 6, tview.NewTableCell(newSuctionCanNo).SetAlign(tview.AlignCenter))

		logger.Info.Printf("Successfully updated sample %d", sampleIndex+1)

		// Show success message
		successModal := tview.NewModal().
			SetText("Sample updated successfully!").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.SetRoot(container, true)
				app.SetFocus(table)
			})
		successModal.SetBackgroundColor(tcell.ColorBlack)
		app.SetRoot(successModal, true)
	})

	form.AddButton("Cancel", func() {
		app.SetRoot(container, true)
		app.SetFocus(table)
	})

	form.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Sample - %s | %s ", sample.BoringNumber, sample.Depth)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	form.SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite).
		SetButtonBackgroundColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// Center the form
	modal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(form, 15, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	modal.SetBackgroundColor(tcell.ColorBlack)
	app.SetRoot(modal, true)
	app.SetFocus(form)
}

func showErrorModal(app *tview.Application, message string, returnTo tview.Primitive, container tview.Primitive) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(container, true)
			app.SetFocus(returnTo)
		})
	modal.SetBackgroundColor(tcell.ColorBlack)
	app.SetRoot(modal, true)
}
