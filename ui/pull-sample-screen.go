package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lms-tui/logger"
	"lms-tui/models"
	"lms-tui/pkg"
)

func NewPullSampleScreen(app *tview.Application, job models.Job, onBack func()) tview.Primitive {
	logger.Info.Printf("Starting pull sample for Job: %s", job.ProjectNumber)

	// Load job data from Excel
	filePath := fmt.Sprintf("projects/%s/Lab_%s.xlsm", job.ProjectNumber, job.ProjectNumber)
	jobData, err := pkg.ExcelToJSON(filePath)

	var samples []pkg.SampleData
	var totalSamples int = 0
	if err == nil && jobData != nil {
		samples = jobData.Samples
		totalSamples = len(samples)
		logger.Info.Printf("Loaded %d samples from job %s", totalSamples, job.ProjectNumber)
	} else {
		logger.Error.Printf("Failed to load job data: %v", err)
	}

	// Initialize moisture test writer - creates ex_project/[job_number]/ directory and Excel file
	moistureWriter, err := pkg.InitMoistureTestFile(job.ProjectNumber)
	if err != nil {
		logger.Error.Printf("Failed to initialize moisture test file: %v", err)
	} else {
		logger.Info.Printf("Initialized moisture test file for job %s", job.ProjectNumber)
	}

	// Initialize soil suction test writer - shares the same file handle as moisture writer
	var suctionWriter *pkg.SoilSuctionWriter
	if moistureWriter != nil {
		suctionWriter, err = pkg.InitSoilSuctionFile(job.ProjectNumber, moistureWriter.GetFile())
		if err != nil {
			logger.Error.Printf("Failed to initialize soil suction test file: %v", err)
		} else {
			logger.Info.Printf("Initialized soil suction test file for job %s", job.ProjectNumber)
		}
	}

	// Track current sample index (0-based) - load saved progress
	currentSampleIndex := 0
	savedIndex, err := pkg.LoadProgress(job.ProjectNumber)
	if err == nil && savedIndex > 0 {
		currentSampleIndex = savedIndex
		logger.Info.Printf("Resuming job %s from sample %d", job.ProjectNumber, currentSampleIndex+1)
	}

	// Track used can numbers to prevent duplicates
	usedMoistureCans := make(map[string]bool)
	usedSuctionCans := make(map[string]bool)

	// Track timing
	startTime := time.Now()

	// Get current sample info
	getCurrentSampleInfo := func() (string, string, string, bool) {
		if currentSampleIndex < len(samples) {
			sample := samples[currentSampleIndex]
			hasSuction := false
			for _, test := range sample.Tests {
				if strings.Contains(test, "Soil Suction") {
					hasSuction = true
					break
				}
			}
			return sample.BoringNumber, sample.Depth, strings.Join(sample.Tests, ", "), hasSuction
		}
		return "-", "-", "-", false
	}

	boringNumber, depth, tests, hasSuction := getCurrentSampleInfo()

	// ===== LEFT BOX - Input Fields =====
	form := tview.NewForm()

	// Helper to rebuild form based on current sample's test requirements
	rebuildForm := func() {
		// Clear and rebuild form with empty values
		form.Clear(false)

		// Moisture Content fields (always present)
		form.AddTextView("", "━━━━━ Moisture Content ━━━━━", 0, 1, true, false)
		form.AddInputField("  Can #", "", 25, nil, nil)
		form.AddInputField("  Can Weight (g)", "", 25, nil, nil)
		form.AddInputField("  Wet Weight (g)", "", 25, nil, nil)

		// Soil Suction fields (only if current sample has Soil Suction test)
		_, _, _, hasSuction = getCurrentSampleInfo()
		if hasSuction {
			form.AddTextView("", "", 0, 1, false, false) // Spacer
			form.AddTextView("", "━━━━━ Soil Suction ━━━━━", 0, 1, true, false)
			form.AddInputField("  Suction Can #", "", 25, nil, nil)
		}
	}

	// Initial form build
	rebuildForm()

	// ===== TOP RIGHT BOX - Job Info =====
	jobInfoText := tview.NewTextView()
	jobInfoText.SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetBackgroundColor(tcell.ColorBlack)

	// Update job info display
	updateJobInfo := func() {
		boringNumber, depth, tests, hasSuction = getCurrentSampleInfo()
		sampleProgress := fmt.Sprintf("%d of %d", currentSampleIndex+1, totalSamples)
		if currentSampleIndex >= totalSamples {
			sampleProgress = "COMPLETE"
		}
		jobInfoText.SetText(fmt.Sprintf(
			"Job Number: %s\n\n"+
				"Boring: %s\n\n"+
				"Depth: %s\n\n"+
				"Sample: %s\n\n"+
				"Tests: %s",
			job.ProjectNumber,
			boringNumber,
			depth,
			sampleProgress,
			tests))
	}

	// Initial update
	updateJobInfo()

	// Declare container early so it can be referenced in closures
	var container *tview.Flex

	// Helper to show error modal and focus back to a specific field
	showErrorModal := func(message string, focusField tview.FormItem) {
		modal := tview.NewModal().
			SetText(message).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.SetRoot(container, true)
				if focusField != nil {
					app.SetFocus(focusField)
				} else {
					app.SetFocus(form)
				}
			})
		modal.SetBackgroundColor(tcell.ColorBlack)
		modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Rune() == '1' || event.Key() == tcell.KeyEnter {
				app.SetRoot(container, true)
				if focusField != nil {
					app.SetFocus(focusField)
				} else {
					app.SetFocus(form)
				}
				return nil
			}
			return event
		})
		app.SetRoot(modal, true)
	}

	// Save sample function (shared by button and keyboard shortcut)
	saveSample := func() {
		if currentSampleIndex >= totalSamples {
			logger.Info.Println("All samples completed")
			return
		}

		canNum := strings.TrimSpace(form.GetFormItemByLabel("  Can #").(*tview.InputField).GetText())
		canWeight := strings.TrimSpace(form.GetFormItemByLabel("  Can Weight (g)").(*tview.InputField).GetText())
		wetWeight := strings.TrimSpace(form.GetFormItemByLabel("  Wet Weight (g)").(*tview.InputField).GetText())

		// Get suction can number only if the field exists
		suctionNum := ""
		if suctionItem := form.GetFormItemByLabel("  Suction Can #"); suctionItem != nil {
			suctionNum = strings.TrimSpace(suctionItem.(*tview.InputField).GetText())
		}

		// Validate required fields
		if canNum == "" {
			logger.Error.Println("Validation failed: Can # is required")
			showErrorModal("Can # is required", form.GetFormItemByLabel("  Can #"))
			return
		}
		if canWeight == "" {
			logger.Error.Println("Validation failed: Can Weight is required")
			showErrorModal("Can Weight is required", form.GetFormItemByLabel("  Can Weight (g)"))
			return
		}
		if wetWeight == "" {
			logger.Error.Println("Validation failed: Wet Weight is required")
			showErrorModal("Wet Weight is required", form.GetFormItemByLabel("  Wet Weight (g)"))
			return
		}
		// Validate suction can # if the sample requires it
		if hasSuction && suctionNum == "" {
			logger.Error.Println("Validation failed: Suction Can # is required for this sample")
			showErrorModal("Suction Can # is required for this sample", form.GetFormItemByLabel("  Suction Can #"))
			return
		}
		// Check for duplicate moisture can number (already used in this session)
		if usedMoistureCans[canNum] {
			logger.Error.Printf("Validation failed: Moisture Can # %s has already been used", canNum)
			showErrorModal(fmt.Sprintf("Moisture Can # %s has already been used in this session.\n\nPlease use a different can.", canNum), form.GetFormItemByLabel("  Can #"))
			return
		}
		// Check if moisture can is already in the oven
		if inOven, canData, _ := pkg.IsCanInOven(canNum); inOven {
			logger.Error.Printf("Validation failed: Moisture Can # %s is already in the oven", canNum)
			showErrorModal(fmt.Sprintf("Moisture Can # %s is already in the oven!\n\nJob: %s\nBoring: %s\nDepth: %s\nTime In: %s\n\nPlease recheck can number or use a different can.", canNum, canData.JobNumber, canData.BoringNumber, canData.Depth, canData.TimeIn), form.GetFormItemByLabel("  Can #"))
			return
		}
		// Check for duplicate suction can number (already used in this session)
		if hasSuction && usedSuctionCans[suctionNum] {
			logger.Error.Printf("Validation failed: Suction Can # %s has already been used", suctionNum)
			showErrorModal(fmt.Sprintf("Suction Can # %s has already been used in this session.\n\nPlease use a different can.", suctionNum), form.GetFormItemByLabel("  Suction Can #"))
			return
		}

		logger.Info.Printf("Sample %d/%d saved - Boring: %s, Depth: %s, Can #: %s, Can Weight: %s, Wet Weight: %s, Suction #: %s",
			currentSampleIndex+1, totalSamples, boringNumber, depth, canNum, canWeight, wetWeight, suctionNum)

		// Mark can numbers as used
		usedMoistureCans[canNum] = true
		if suctionNum != "" {
			usedSuctionCans[suctionNum] = true
		}

		// Write moisture data to Excel file
		if moistureWriter != nil {
			err := moistureWriter.WriteMoistureSample(boringNumber, depth, canNum, canWeight, wetWeight)
			if err != nil {
				logger.Error.Printf("Failed to write moisture sample to Excel: %v", err)
			}
		}

		// Write soil suction data to Excel file
		if suctionWriter != nil && suctionNum != "" {
			err := suctionWriter.WriteSoilSuctionSample(boringNumber, depth, suctionNum)
			if err != nil {
				logger.Error.Printf("Failed to write soil suction sample to Excel: %v", err)
			}
		}

		// Save backup to JSON file
		if err := pkg.SaveSampleBackup(job.ProjectNumber, boringNumber, depth, canNum, canWeight, wetWeight, suctionNum); err != nil {
			logger.Error.Printf("Failed to save sample backup: %v", err)
		}

		// Add moisture can to oven tracking
		if moistureWriter != nil {
			moistureSheet, moistureColumn, found := moistureWriter.GetSampleMapping(boringNumber, depth)
			if found {
				if err := pkg.AddCanToOven(canNum, job.ProjectNumber, boringNumber, depth, moistureSheet, moistureColumn); err != nil {
					logger.Error.Printf("Failed to add can to oven: %v", err)
				}
			} else {
				logger.Error.Printf("Could not find moisture sheet mapping for %s at %s", boringNumber, depth)
			}
		}

		// Move to next sample
		currentSampleIndex++

		// Save progress so user can resume later
		if err := pkg.SaveProgress(job.ProjectNumber, currentSampleIndex); err != nil {
			logger.Error.Printf("Failed to save progress: %v", err)
		}

		// Update the job info display
		updateJobInfo()

		// Rebuild form for next sample (this also clears the fields)
		rebuildForm()

		// Focus back to first input field (skip the text views)
		app.SetFocus(form.GetFormItem(1))

		// Check if all samples are done
		if currentSampleIndex >= totalSamples {
			logger.Info.Printf("All %d samples completed for job %s", totalSamples, job.ProjectNumber)
		}
	}

	// Submit button
	form.AddButton("Save Sample", saveSample)

	// Handle Enter key - move between fields, save when on button
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Check if focus is on the Save Sample button
			if form.GetButton(0) != nil && app.GetFocus() == form.GetButton(0) {
				saveSample()
				return nil
			}

			// Otherwise, move to next input field (skip text views/labels)
			currentIndex, _ := form.GetFocusedItemIndex()
			if currentIndex >= 0 {
				// Find next input field
				totalItems := form.GetFormItemCount()
				for nextIndex := currentIndex + 1; nextIndex < totalItems; nextIndex++ {
					item := form.GetFormItem(nextIndex)
					// Check if it's an input field (not a text view)
					if _, ok := item.(*tview.InputField); ok {
						app.SetFocus(item)
						return nil
					}
				}
				// If no more input fields, focus the button
				app.SetFocus(form.GetButton(0))
			}
			return nil
		}
		// Handle / key to reset all fields for current sample
		if event.Rune() == '/' {
			// Clear all input fields
			if canField := form.GetFormItemByLabel("  Can #"); canField != nil {
				canField.(*tview.InputField).SetText("")
			}
			if canWeightField := form.GetFormItemByLabel("  Can Weight (g)"); canWeightField != nil {
				canWeightField.(*tview.InputField).SetText("")
			}
			if wetWeightField := form.GetFormItemByLabel("  Wet Weight (g)"); wetWeightField != nil {
				wetWeightField.(*tview.InputField).SetText("")
			}
			if suctionField := form.GetFormItemByLabel("  Suction Can #"); suctionField != nil {
				suctionField.(*tview.InputField).SetText("")
			}
			// Focus back to first input field
			app.SetFocus(form.GetFormItem(1))
			logger.Info.Println("Reset all fields for current sample")
			return nil
		}
		return event
	})

	form.SetBorder(true).
		SetTitle(" ┃ Sample Input ┃ ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack).
		SetBorderPadding(1, 1, 2, 2)

	form.SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite).
		SetButtonBackgroundColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)
	form.SetItemPadding(1)

	jobInfoBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(jobInfoText, 0, 1, false)

	jobInfoBox.SetBorder(true).
		SetTitle(" Current Sample ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// ===== BOTTOM RIGHT BOX - Timing =====
	timeDisplay := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	timeDisplay.SetBackgroundColor(tcell.ColorBlack)

	// Update time display function
	updateTimeDisplay := func() {
		currentTime := time.Now().Format("3:04:05 PM")
		elapsed := time.Since(startTime)
		elapsedStr := fmt.Sprintf("%02d:%02d:%02d",
			int(elapsed.Hours()),
			int(elapsed.Minutes())%60,
			int(elapsed.Seconds())%60)

		avgTime := "-"
		if currentSampleIndex > 0 {
			avgDuration := elapsed / time.Duration(currentSampleIndex)
			avgTime = fmt.Sprintf("%02d:%02d",
				int(avgDuration.Minutes()),
				int(avgDuration.Seconds())%60)
		}

		timeDisplay.SetText(fmt.Sprintf(
			"Current Time: %s\n\n"+
				"Time Pulling: %s\n\n"+
				"Avg Time/Sample: %s\n\n"+
				"Samples Done: %d",
			currentTime,
			elapsedStr,
			avgTime,
			currentSampleIndex))
	}

	// Initial update
	updateTimeDisplay()

	// Update every second
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			app.QueueUpdateDraw(func() {
				updateTimeDisplay()
			})
		}
	}()

	timeBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(timeDisplay, 0, 1, false)

	timeBox.SetBorder(true).
		SetTitle(" Timing ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// ===== RIGHT SIDE - Stack job info and timing =====
	rightSide := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(jobInfoBox, 0, 1, false).
		AddItem(timeBox, 0, 1, false)

	// ===== MAIN LAYOUT - Left (form) and Right (info + timing) =====
	mainContent := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(form, 0, 1, true).
		AddItem(rightSide, 0, 1, false)

	// Instructions at bottom
	instructions := tview.NewTextView().
		SetText("Tab: Next Field  |  Enter: Save Sample  |  /: Reset Fields  |  +: Back to Menu").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetBackgroundColor(tcell.ColorBlack)

	// Container with instructions - FULLSCREEN
	container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainContent, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	container.SetBorder(true).
		SetTitle(fmt.Sprintf(" Pull Sample - Job %s ", job.ProjectNumber)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// Input capture for back navigation
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			// Check if job is not complete
			if currentSampleIndex < totalSamples {
				// Show confirmation modal
				modal := tview.NewModal().
					SetText(fmt.Sprintf("You have completed %d of %d samples.\n\nAre you sure you want to stop for now?\n\n[1] Yes, Stop    [2] No, Continue", currentSampleIndex, totalSamples)).
					AddButtons([]string{"Yes, Stop", "No, Continue"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonLabel == "Yes, Stop" {
							logger.Info.Printf("User confirmed stop - Samples completed: %d/%d, Total time: %v", currentSampleIndex, totalSamples, time.Since(startTime))
							// Close the moisture writer (this also closes the shared file)
							if moistureWriter != nil {
								moistureWriter.Close()
								logger.Info.Printf("Closed Lab file for job %s", job.ProjectNumber)
							}
							onBack()
						} else {
							// Go back to form
							app.SetRoot(container, true)
							app.SetFocus(form)
						}
					})
				modal.SetBackgroundColor(tcell.ColorBlack)
				// Add keyboard shortcut support for 1 and 2
				modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
					if event.Rune() == '1' {
						logger.Info.Printf("User confirmed stop - Samples completed: %d/%d, Total time: %v", currentSampleIndex, totalSamples, time.Since(startTime))
						// Close the moisture writer (this also closes the shared file)
						if moistureWriter != nil {
							moistureWriter.Close()
							logger.Info.Printf("Closed Lab file for job %s", job.ProjectNumber)
						}
						onBack()
						return nil
					} else if event.Rune() == '2' {
						// Go back to form
						app.SetRoot(container, true)
						app.SetFocus(form)
						return nil
					}
					return event
				})
				app.SetRoot(modal, true)
			} else {
				// Job is complete, just go back
				logger.Info.Printf("Finished pulling - Samples completed: %d/%d, Total time: %v", currentSampleIndex, totalSamples, time.Since(startTime))
				// Close the moisture writer (this also closes the shared file)
				if moistureWriter != nil {
					moistureWriter.Close()
					logger.Info.Printf("Closed Lab file for job %s", job.ProjectNumber)
				}
				// Note: suctionWriter shares the same file, so no need to close it separately
				onBack()
			}
			return nil
		}
		return event
	})

	return container
}
