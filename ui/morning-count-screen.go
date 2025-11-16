package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"lms-tui/logger"
	"lms-tui/pkg"
)

func NewMorningCountScreen(app *tview.Application, onBack func()) tview.Primitive {
	logger.Info.Println("Opening Morning Count screen")

	// Load cans currently in oven
	cansInOven, err := pkg.GetCansInOven()
	if err != nil {
		logger.Error.Printf("Failed to load oven tracking: %v", err)
		cansInOven = []pkg.OvenCanData{}
	}

	// ===== LEFT BOX - List of cans in oven =====
	canListText := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(false)

	updateCanList := func() {
		var listContent strings.Builder
		if len(cansInOven) == 0 {
			listContent.WriteString("[gray]No cans in oven[-]")
		} else {
			for i, can := range cansInOven {
				listContent.WriteString(fmt.Sprintf("[yellow]%d.[-] Can #[white]%s[-]\n", i+1, can.CanNumber))
				listContent.WriteString(fmt.Sprintf("   Job: %s\n", can.JobNumber))
				listContent.WriteString(fmt.Sprintf("   Boring: %s\n", can.BoringNumber))
				listContent.WriteString(fmt.Sprintf("   Depth: %s\n", can.Depth))
				listContent.WriteString(fmt.Sprintf("   Time In: %s\n", can.TimeIn))
				if i < len(cansInOven)-1 {
					listContent.WriteString("\n")
				}
			}
		}
		canListText.SetText(listContent.String())
	}
	updateCanList()

	canListBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(canListText, 0, 1, false)

	canListBox.SetBorder(true).
		SetTitle(fmt.Sprintf(" Cans in Oven (%d) ", len(cansInOven))).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// ===== RIGHT BOX - Input form =====
	form := tview.NewForm()

	// Status text to show results
	statusText := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	statusText.SetBackgroundColor(tcell.ColorBlack)

	// Track completed cans
	completedCount := 0

	updateStatus := func(message string) {
		statusText.SetText(fmt.Sprintf("%s\n\nCompleted: %d / %d", message, completedCount, len(cansInOven)))
	}
	updateStatus("Enter can number and dry weight")

	// Declare container early for modal references
	var container *tview.Flex

	// Helper to show error modal
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
		app.SetRoot(modal, true)
	}

	// Save function
	saveDryWeight := func() {
		canNumField := form.GetFormItemByLabel("Can #").(*tview.InputField)
		dryWeightField := form.GetFormItemByLabel("Dry Weight (g)").(*tview.InputField)

		canNum := strings.TrimSpace(canNumField.GetText())
		dryWeight := strings.TrimSpace(dryWeightField.GetText())

		// Validate inputs
		if canNum == "" {
			showErrorModal("Can # is required", canNumField)
			return
		}
		if dryWeight == "" {
			showErrorModal("Dry Weight is required", dryWeightField)
			return
		}

		// Find the can in the oven
		var foundCan *pkg.OvenCanData
		for i := range cansInOven {
			if cansInOven[i].CanNumber == canNum {
				foundCan = &cansInOven[i]
				break
			}
		}

		if foundCan == nil {
			showErrorModal(fmt.Sprintf("Can # %s is not in the oven.\n\nPlease check the can number.", canNum), canNumField)
			return
		}

		// Write dry weight to moisture sheet
		if err := pkg.WriteDryWeightToMoistureSheet(*foundCan, dryWeight); err != nil {
			logger.Error.Printf("Failed to write dry weight to moisture sheet: %v", err)
			showErrorModal(fmt.Sprintf("Failed to save dry weight:\n%v", err), nil)
			return
		}

		// Remove can from oven
		if _, err := pkg.RemoveCanFromOven(canNum); err != nil {
			logger.Error.Printf("Failed to remove can from oven: %v", err)
		}

		logger.Info.Printf("Saved dry weight for can %s: %s g (Job: %s, Boring: %s, Depth: %s)",
			canNum, dryWeight, foundCan.JobNumber, foundCan.BoringNumber, foundCan.Depth)

		// Update the cans list
		newCans := []pkg.OvenCanData{}
		for _, can := range cansInOven {
			if can.CanNumber != canNum {
				newCans = append(newCans, can)
			}
		}
		cansInOven = newCans
		updateCanList()
		canListBox.SetTitle(fmt.Sprintf(" Cans in Oven (%d) ", len(cansInOven)))

		// Clear inputs for next entry
		canNumField.SetText("")
		dryWeightField.SetText("")

		// Update status
		completedCount++
		updateStatus(fmt.Sprintf("[green]Saved Can #%s: %s g[-]", canNum, dryWeight))

		// Focus back to can number field
		app.SetFocus(canNumField)
	}

	// Add input fields
	form.AddInputField("Can #", "", 20, nil, nil)
	form.AddInputField("Dry Weight (g)", "", 20, tview.InputFieldFloat, nil)
	form.AddButton("Save", saveDryWeight)

	// Handle Enter key to move between fields
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			// Check if focus is on the Save button
			if form.GetButton(0) != nil && app.GetFocus() == form.GetButton(0) {
				saveDryWeight()
				return nil
			}

			// Move to next field
			currentIndex, _ := form.GetFocusedItemIndex()
			if currentIndex >= 0 {
				totalItems := form.GetFormItemCount()
				for nextIndex := currentIndex + 1; nextIndex < totalItems; nextIndex++ {
					item := form.GetFormItem(nextIndex)
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
		return event
	})

	form.SetBorder(false).
		SetBackgroundColor(tcell.ColorBlack)
	form.SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite).
		SetButtonBackgroundColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorBlack).
		SetLabelColor(tcell.ColorWhite)
	form.SetItemPadding(1)

	// Right box containing form and status
	rightBox := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(form, 10, 0, true).
		AddItem(statusText, 0, 1, false)

	rightBox.SetBorder(true).
		SetTitle(" Enter Dry Weight ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// ===== MAIN LAYOUT =====
	mainContent := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(canListBox, 0, 1, false).
		AddItem(rightBox, 0, 1, true)

	// Instructions
	instructions := tview.NewTextView().
		SetText("Tab: Next Field  |  Enter: Save  |  +: Back to Menu").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetBackgroundColor(tcell.ColorBlack)

	// Container
	container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(mainContent, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	container.SetBorder(true).
		SetTitle(" Morning Count - Dry Weights ").
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorBlack)

	// Back navigation
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == '+' {
			logger.Info.Println("Returning from Morning Count screen")
			onBack()
			return nil
		}
		return event
	})

	return container
}
