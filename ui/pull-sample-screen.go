package ui

import (
	"fmt"
	"strconv"
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

	// Load job data from Excel using the specific Lab file path
	jobData, err := pkg.ExcelToJSON(job.LabFilePath)

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
	// Each Lab file version gets its own directory (e.g., ex_project/25490/ and ex_project/25490_03/)
	moistureWriter, err := pkg.InitMoistureTestFile(job.ProjectNumber, job.LabFilePath)
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

	// Track last saved sample for edit feature
	var lastSampleData struct {
		boringNumber string
		depth        string
		canNumber    string
		canWeight    string
		wetWeight    string
		suctionCanNo string
		sampleIndex  int
	}
	lastSampleData.sampleIndex = -1 // -1 means no sample saved yet

	// Track timing
	startTime := time.Now()
	sampleStartTime := time.Now() // Track time for current sample (resets on save)

	// Get current sample info
	getCurrentSampleInfo := func() (string, string, string, bool, bool) {
		if currentSampleIndex < len(samples) {
			sample := samples[currentSampleIndex]
			hasSuction := false
			hasOtherTests := false

			for _, test := range sample.Tests {
				if strings.Contains(test, "Soil Suction") {
					hasSuction = true
				} else if !strings.Contains(test, "Moisture Content") && !strings.Contains(test, "Moisture") {
					// Check if there are tests other than Moisture Content and Soil Suction
					hasOtherTests = true
				}
			}
			return sample.BoringNumber, sample.Depth, strings.Join(sample.Tests, ", "), hasSuction, hasOtherTests
		}
		return "-", "-", "-", false, false
	}

	boringNumber, depth, tests, hasSuction, hasOtherTests := getCurrentSampleInfo()

	// ===== LEFT BOX - Input Fields =====
	form := tview.NewForm()

	// Declare saveSample and continueSaveSample early so they can be referenced
	var saveSample func()
	var continueSaveSample func(string, string, string, string)

	// Helper to rebuild form based on current sample's test requirements
	rebuildForm := func() {
		// Clear and rebuild form with empty values (true = also clear buttons)
		form.Clear(true)

		// Moisture Content fields (always present)
		form.AddTextView("", "━━━━━ Moisture Content ━━━━━", 0, 1, true, false)
		form.AddInputField("  Can #", "", 25, nil, nil)
		form.AddInputField("  Can Weight (g)", "", 25, nil, nil)
		form.AddInputField("  Wet Weight (g)", "", 25, nil, nil)

		// Soil Suction fields (only if current sample has Soil Suction test)
		_, _, _, hasSuction, hasOtherTests = getCurrentSampleInfo()
		if hasSuction {
			form.AddTextView("", "", 0, 1, false, false) // Spacer
			form.AddTextView("", "━━━━━ Soil Suction ━━━━━", 0, 1, true, false)
			form.AddInputField("  Suction Can #", "", 25, nil, nil)
		}

		// Add button with dynamic text based on tests
		buttonText := "Save Sample"
		if hasOtherTests {
			buttonText = "Get test and save sample"
		}
		form.AddButton(buttonText, saveSample)
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
		boringNumber, depth, tests, hasSuction, hasOtherTests = getCurrentSampleInfo()
		sampleProgress := fmt.Sprintf("%d of %d", currentSampleIndex+1, totalSamples)

		// Create visual progress bar
		progressBar := ""
		percentage := 0
		if totalSamples > 0 {
			percentage = (currentSampleIndex * 100) / totalSamples
			barLength := 20
			filledLength := (currentSampleIndex * barLength) / totalSamples
			if filledLength > barLength {
				filledLength = barLength
			}

			progressBar = "["
			for i := 0; i < barLength; i++ {
				if i < filledLength {
					progressBar += "█"
				} else {
					progressBar += "░"
				}
			}
			progressBar += fmt.Sprintf("] %d%%", percentage)
		}

		if currentSampleIndex >= totalSamples {
			sampleProgress = "COMPLETE"
			progressBar = "[████████████████████] 100%"
		}

		jobInfoText.SetText(fmt.Sprintf(
			"Job Number: %s\n\n"+
				"Progress: %s\n"+
				"%s\n\n"+
				"Boring: %s\n\n"+
				"Depth: %s\n\n"+
				"Tests: %s",
			job.ProjectNumber,
			sampleProgress,
			progressBar,
			boringNumber,
			depth,
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

	// Helper function to continue saving after validations pass
	continueSaveSample = func(canNum, canWeight, wetWeight, suctionNum string) {
		// Check for duplicate can numbers (if enabled in config)
		if pkg.CheckDuplicateCans {
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
		}

		logger.Info.Printf("Sample %d/%d saved - Boring: %s, Depth: %s, Can #: %s, Can Weight: %s, Wet Weight: %s, Suction #: %s",
			currentSampleIndex+1, totalSamples, boringNumber, depth, canNum, canWeight, wetWeight, suctionNum)

		// Mark can numbers as used (if duplicate checking is enabled)
		if pkg.CheckDuplicateCans {
			usedMoistureCans[canNum] = true
			if suctionNum != "" {
				usedSuctionCans[suctionNum] = true
			}
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

		// Save last sample data for edit feature
		lastSampleData.boringNumber = boringNumber
		lastSampleData.depth = depth
		lastSampleData.canNumber = canNum
		lastSampleData.canWeight = canWeight
		lastSampleData.wetWeight = wetWeight
		lastSampleData.suctionCanNo = suctionNum
		lastSampleData.sampleIndex = currentSampleIndex

		// Move to next sample
		currentSampleIndex++

		// Reset sample timer for next sample
		sampleStartTime = time.Now()

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
			showCompletionScreen(app, job, moistureWriter, container, onBack)
		}
	}

	// Save sample function (shared by button and keyboard shortcut)
	saveSample = func() {
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

		// Validate numeric values and minimum sample weight (100g)
		canWeightFloat, err := strconv.ParseFloat(canWeight, 64)
		if err != nil {
			logger.Error.Printf("Validation failed: Can Weight '%s' is not a valid number", canWeight)
			showErrorModal(fmt.Sprintf("Can Weight must be a valid number\n\nYou entered: %s", canWeight), form.GetFormItemByLabel("  Can Weight (g)"))
			return
		}
		wetWeightFloat, err := strconv.ParseFloat(wetWeight, 64)
		if err != nil {
			logger.Error.Printf("Validation failed: Wet Weight '%s' is not a valid number", wetWeight)
			showErrorModal(fmt.Sprintf("Wet Weight must be a valid number\n\nYou entered: %s", wetWeight), form.GetFormItemByLabel("  Wet Weight (g)"))
			return
		}

		// Check if wet weight > can weight
		if wetWeightFloat <= canWeightFloat {
			logger.Error.Printf("Validation failed: Wet Weight (%.2fg) must be greater than Can Weight (%.2fg)", wetWeightFloat, canWeightFloat)
			showErrorModal(fmt.Sprintf("Wet Weight must be greater than Can Weight\n\nCan Weight: %.2fg\nWet Weight: %.2fg", canWeightFloat, wetWeightFloat), form.GetFormItemByLabel("  Wet Weight (g)"))
			return
		}

		// Check minimum sample weight (100g difference)
		sampleWeight := wetWeightFloat - canWeightFloat
		if sampleWeight < 100.0 {
			logger.Info.Printf("Warning: Sample weight (%.2fg) is less than recommended 100g minimum", sampleWeight)

			// Show warning modal with override option
			modal := tview.NewModal().
				SetText(fmt.Sprintf("⚠️ Sample Weight Below Minimum\n\n"+
					"Can Weight: %.2fg\n"+
					"Wet Weight: %.2fg\n"+
					"Sample Weight: %.2fg\n\n"+
					"Recommended minimum: 100g\n"+
					"This sample is %.2fg under the minimum.\n\n"+
					"Do you want to proceed anyway?\n\n"+
					"[1] Override & Save    [2] Cancel",
					canWeightFloat, wetWeightFloat, sampleWeight, 100.0-sampleWeight)).
				AddButtons([]string{"Override & Save", "Cancel"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "Override & Save" {
						logger.Info.Printf("User overrode minimum sample weight warning for %.2fg sample", sampleWeight)
						// Continue with save - call the rest of saveSample logic
						continueSaveSample(canNum, canWeight, wetWeight, suctionNum)
					} else {
						// Cancel - go back to form
						app.SetRoot(container, true)
						app.SetFocus(form.GetFormItemByLabel("  Wet Weight (g)"))
					}
				})
			modal.SetBackgroundColor(tcell.ColorBlack)
			// Add keyboard shortcut support for 1 and 2
			modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Rune() == '1' {
					logger.Info.Printf("User overrode minimum sample weight warning for %.2fg sample", sampleWeight)
					continueSaveSample(canNum, canWeight, wetWeight, suctionNum)
					return nil
				} else if event.Rune() == '2' {
					app.SetRoot(container, true)
					app.SetFocus(form.GetFormItemByLabel("  Wet Weight (g)"))
					return nil
				}
				return event
			})
			app.SetRoot(modal, true)
			return
		}

		// If we get here, all validations passed - continue with save
		continueSaveSample(canNum, canWeight, wetWeight, suctionNum)
	}

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

		// Calculate current sample elapsed time
		sampleElapsed := time.Since(sampleStartTime)
		sampleSeconds := int(sampleElapsed.Seconds())

		// Calculate progress (0-180 seconds = 0-100%)
		targetSeconds := 180 // 3 minutes
		progressPercent := int((float64(sampleSeconds) / float64(targetSeconds)) * 100)
		if progressPercent > 100 {
			progressPercent = 100
		}

		// Create progress bar (20 characters wide)
		barWidth := 20
		filledBars := (progressPercent * barWidth) / 100
		if filledBars > barWidth {
			filledBars = barWidth
		}

		// Calculate color (green to red gradient)
		// At 0 sec: green, at 180 sec: red
		// We'll use color codes: green=#00FF00, yellow=#FFFF00, red=#FF0000
		var barColor string
		if sampleSeconds < 90 {
			// 0-90 seconds: green to yellow
			barColor = "green"
		} else if sampleSeconds < 150 {
			// 90-150 seconds: yellow to orange
			barColor = "yellow"
		} else if sampleSeconds < 180 {
			// 150-180 seconds: orange to red
			barColor = "orange"
		} else {
			// Over 180 seconds: red
			barColor = "red"
		}

		// Build progress bar string
		progressBar := "["
		for i := 0; i < barWidth; i++ {
			if i < filledBars {
				progressBar += "█"
			} else {
				progressBar += "░"
			}
		}
		progressBar += "]"

		// Format sample time as MM:SS
		sampleTimeStr := fmt.Sprintf("%02d:%02d", sampleSeconds/60, sampleSeconds%60)

		// Add color tag for the progress bar based on time
		coloredProgressBar := fmt.Sprintf("[%s]%s[white]", barColor, progressBar)

		timeDisplay.SetText(fmt.Sprintf(
			"Current Time: %s\n\n"+
				"Sample Time: %s / 03:00\n"+
				"%s\n\n"+
				"Time Pulling: %s\n\n"+
				"Avg Time/Sample: %s\n\n"+
				"Samples Done: %d",
			currentTime,
			sampleTimeStr,
			coloredProgressBar,
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
		SetText("Tab: Next Field  |  Enter: Save Sample  |  /: Reset Fields  |  -: Edit Last Sample  |  +: Back to Menu").
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

	// Input capture for back navigation and edit last sample
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '-' {
			// Edit last sample
			if lastSampleData.sampleIndex >= 0 {
				showEditLastSampleModal(app, job, &lastSampleData, moistureWriter, container, form)
			} else {
				// No samples saved yet
				modal := tview.NewModal().
					SetText("No samples have been saved yet.\n\nSave at least one sample before using edit feature.").
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						app.SetRoot(container, true)
						app.SetFocus(form)
					})
				modal.SetBackgroundColor(tcell.ColorBlack)
				app.SetRoot(modal, true)
			}
			return nil
		}
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
				// Job is complete, show completion screen
				logger.Info.Printf("All samples completed for job %s", job.ProjectNumber)
				showCompletionScreen(app, job, moistureWriter, container, onBack)
			}
			return nil
		}
		return event
	})

	return container
}

func showEditLastSampleModal(app *tview.Application, job models.Job, lastSample *struct {
	boringNumber string
	depth        string
	canNumber    string
	canWeight    string
	wetWeight    string
	suctionCanNo string
	sampleIndex  int
}, moistureWriter *pkg.MoistureTestWriter, returnContainer tview.Primitive, returnFocus tview.Primitive) {

	logger.Info.Printf("Opening edit last sample modal for %s | %s", lastSample.boringNumber, lastSample.depth)

	// Create edit form
	editForm := tview.NewForm()
	editForm.AddInputField("Can #", lastSample.canNumber, 25, nil, nil)
	editForm.AddInputField("Can Weight (g)", lastSample.canWeight, 25, nil, nil)
	editForm.AddInputField("Wet Weight (g)", lastSample.wetWeight, 25, nil, nil)
	if lastSample.suctionCanNo != "" {
		editForm.AddInputField("Suction Can #", lastSample.suctionCanNo, 25, nil, nil)
	}

	editForm.AddButton("Save Changes", func() {
		// Get updated values
		newCanNo := strings.TrimSpace(editForm.GetFormItemByLabel("Can #").(*tview.InputField).GetText())
		newCanWeight := strings.TrimSpace(editForm.GetFormItemByLabel("Can Weight (g)").(*tview.InputField).GetText())
		newWetWeight := strings.TrimSpace(editForm.GetFormItemByLabel("Wet Weight (g)").(*tview.InputField).GetText())
		newSuctionCanNo := ""
		if suctionField := editForm.GetFormItemByLabel("Suction Can #"); suctionField != nil {
			newSuctionCanNo = strings.TrimSpace(suctionField.(*tview.InputField).GetText())
		}

		// Validate
		if newCanNo == "" || newCanWeight == "" || newWetWeight == "" {
			showEditErrorModal(app, "Can #, Can Weight, and Wet Weight are required", returnContainer, returnFocus)
			return
		}

		logger.Info.Printf("Updating last sample: %s|%s - Can#: %s->%s, CanWt: %s->%s, WetWt: %s->%s, SuctionCan: %s->%s",
			lastSample.boringNumber, lastSample.depth,
			lastSample.canNumber, newCanNo,
			lastSample.canWeight, newCanWeight,
			lastSample.wetWeight, newWetWeight,
			lastSample.suctionCanNo, newSuctionCanNo)

		// Load backup data
		backupFile := fmt.Sprintf("ex_project/%s/backup.json", job.ProjectNumber)
		backupData, err := pkg.LoadBackupData(backupFile)
		if err != nil {
			logger.Error.Printf("Failed to load backup data: %v", err)
			showEditErrorModal(app, fmt.Sprintf("Failed to load backup:\n%v", err), returnContainer, returnFocus)
			return
		}

		// Find and update the sample in backup
		sampleFound := false
		for i := range backupData.Samples {
			if backupData.Samples[i].BoringNumber == lastSample.boringNumber &&
				backupData.Samples[i].Depth == lastSample.depth {
				backupData.Samples[i].CanNumber = newCanNo
				backupData.Samples[i].CanWeight = newCanWeight
				backupData.Samples[i].WetWeight = newWetWeight
				backupData.Samples[i].SuctionCanNo = newSuctionCanNo
				sampleFound = true
				break
			}
		}

		if !sampleFound {
			logger.Error.Printf("Could not find sample in backup: %s|%s", lastSample.boringNumber, lastSample.depth)
			showEditErrorModal(app, "Sample not found in backup file", returnContainer, returnFocus)
			return
		}

		// Save backup
		if err := pkg.SaveBackupDataToFile(backupData, backupFile); err != nil {
			logger.Error.Printf("Failed to save backup: %v", err)
			showEditErrorModal(app, fmt.Sprintf("Failed to save backup:\n%v", err), returnContainer, returnFocus)
			return
		}

		// Update Excel file - moisture data
		err = moistureWriter.WriteMoistureSample(lastSample.boringNumber, lastSample.depth, newCanNo, newCanWeight, newWetWeight)
		if err != nil {
			logger.Error.Printf("Failed to write moisture sample: %v", err)
			showEditErrorModal(app, fmt.Sprintf("Failed to update moisture data:\n%v", err), returnContainer, returnFocus)
			return
		}

		// Update Excel file - suction data if present
		if newSuctionCanNo != "" {
			suctionWriter, err := pkg.InitSoilSuctionFile(job.ProjectNumber, moistureWriter.GetFile())
			if err != nil {
				logger.Error.Printf("Failed to initialize suction writer: %v", err)
			} else {
				defer suctionWriter.Close()
				err = suctionWriter.WriteSoilSuctionSample(lastSample.boringNumber, lastSample.depth, newSuctionCanNo)
				if err != nil {
					logger.Error.Printf("Failed to write suction sample: %v", err)
				}
			}
		}

		// Update the lastSample data with new values
		lastSample.canNumber = newCanNo
		lastSample.canWeight = newCanWeight
		lastSample.wetWeight = newWetWeight
		lastSample.suctionCanNo = newSuctionCanNo

		logger.Info.Printf("Successfully updated last sample")

		// Show success message
		successModal := tview.NewModal().
			SetText("Last sample updated successfully!").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.SetRoot(returnContainer, true)
				app.SetFocus(returnFocus)
			})
		successModal.SetBackgroundColor(tcell.ColorBlack)
		app.SetRoot(successModal, true)
	})

	editForm.AddButton("Cancel", func() {
		app.SetRoot(returnContainer, true)
		app.SetFocus(returnFocus)
	})

	editForm.SetBorder(true).
		SetTitle(fmt.Sprintf(" Edit Last Sample - %s | %s ", lastSample.boringNumber, lastSample.depth)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorBlack)

	editForm.SetFieldBackgroundColor(tcell.ColorBlack).
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
			AddItem(editForm, 15, 0, true).
			AddItem(nil, 0, 1, false), 60, 0, true).
		AddItem(nil, 0, 1, false)

	modal.SetBackgroundColor(tcell.ColorBlack)
	app.SetRoot(modal, true)
	app.SetFocus(editForm)
}

func showEditErrorModal(app *tview.Application, message string, returnContainer tview.Primitive, returnFocus tview.Primitive) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(returnContainer, true)
			app.SetFocus(returnFocus)
		})
	modal.SetBackgroundColor(tcell.ColorBlack)
	app.SetRoot(modal, true)
}

func showCompletionScreen(app *tview.Application, job models.Job, moistureWriter *pkg.MoistureTestWriter, returnContainer tview.Primitive, onBack func()) {
	// Completion message
	completionText := tview.NewTextView().
		SetText(fmt.Sprintf("[green]✓ All samples completed for Job %s![white]\n\nWhat would you like to do next?", job.ProjectNumber)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetBackgroundColor(tcell.ColorBlack)

	// Container for completion screen - declare early so it can be referenced in closures
	var completionContainer *tview.Flex
	var menu *tview.List

	// Create menu options
	menu = tview.NewList().
		AddItem("Finish Job", "Close files and return to main menu", '1', func() {
			logger.Info.Printf("Finishing job %s", job.ProjectNumber)
			// Close the moisture writer
			if moistureWriter != nil {
				moistureWriter.Close()
				logger.Info.Printf("Closed Lab file for job %s", job.ProjectNumber)
			}
			onBack()
		}).
		AddItem("Print Suction Sheet", "Print the soil suction test sheet", '2', func() {
			logger.Info.Printf("Printing suction sheet for job %s", job.ProjectNumber)
			// TODO: Implement print suction sheet functionality
			showInfoModal(app, "Print Suction Sheet feature is coming soon!\n\nPress Enter to continue", completionContainer, menu)
		}).
		AddItem("Print Moisture Content Sheet", "Print the moisture content test sheet", '3', func() {
			logger.Info.Printf("Printing moisture content sheet for job %s", job.ProjectNumber)
			// TODO: Implement print moisture content sheet functionality
			showInfoModal(app, "Print Moisture Content Sheet feature is coming soon!\n\nPress Enter to continue", completionContainer, menu)
		})

	// Create container
	completionContainer = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(completionText, 3, 0, false).
		AddItem(menu, 0, 1, true)

	completionContainer.SetBorder(true).
		SetTitle(" Job Complete ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorGreen).
		SetBackgroundColor(tcell.ColorBlack)

	app.SetRoot(completionContainer, true)
	app.SetFocus(menu)
}

func showInfoModal(app *tview.Application, message string, returnTo tview.Primitive, focusTo tview.Primitive) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(returnTo, true)
			if focusTo != nil {
				app.SetFocus(focusTo)
			}
		})
	modal.SetBackgroundColor(tcell.ColorBlack)
	app.SetRoot(modal, true)
}
