package ui

import (
	"lms-tui/logger"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewLoginScreen(app *tview.Application, onLogin func(userID, pin string)) tview.Primitive {

	var userID, pin string

	// Create a form which handles input fields better
	form := tview.NewForm().
		AddInputField("UserID", "", 30, func(textToCheck string, lastChar rune) bool {
			// Only accept digits
			return lastChar >= '0' && lastChar <= '9'
		}, func(text string) {

			userID = text
		}).
		AddPasswordField("PIN", "", 30, '*', func(text string) {
			pin = text
		}).
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldTextColor(tcell.ColorWhite).
		SetButtonTextColor(tcell.ColorWhite).
		SetLabelColor(tcell.ColorWhite)

	form.SetBorder(true).
		SetTitle(" LMS Login ").
		SetTitleAlign(tview.AlignCenter)

		form.GetFormItem(1).(*tview.InputField).SetAcceptanceFunc(func(textToCheck string, lastChar rune) bool {
			return lastChar >= '0' && lastChar <= '9'
		})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Log all key presses
		logger.Info.Printf("Key pressed - Key: %v, Rune: %c (%d), Name: %s, Modifiers: %v",
			event.Key(), event.Rune(), event.Rune(), event.Name(), event.Modifiers())

		if event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab {
			return nil
		}

		// Handle Enter key
		if event.Key() == tcell.KeyEnter {
			logger.Info.Println("Enter key detected!")
			// Get current focus index
			focusIndex := -1
			for i := 0; i < form.GetFormItemCount(); i++ {
				if app.GetFocus() == form.GetFormItem(i) {
					focusIndex = i
					break
				}
			}
			logger.Info.Printf("Current focus index: %d", focusIndex)

			// If focus is on first field (UserID), move to second field
			if focusIndex == 0 {
				if userID == "" {
					logger.Info.Println("UserID is empty, not moving to next field")
					return nil
				}
				logger.Info.Println("Moving to PIN field")
				app.SetFocus(form.GetFormItem(1))
				return nil
			}

			// If focus is on second field (PIN), attempt login
			if focusIndex == 1 {
				logger.Info.Println("Attempting login")
				onLogin(userID, pin)
				return nil
			}
		}

		return event
	})

	// Add instructions below the form
	instructions := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Click ENTER to continue").
		SetTextColor(tcell.ColorWhite)

	// Container for form and instructions
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(instructions, 1, 0, false)

	// Center vertically
	vertical := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(container, 11, 1, true).
		AddItem(nil, 0, 1, false)

	// Center horizontally
	horizontal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(vertical, 50, 1, true).
		AddItem(nil, 0, 1, false)

	return horizontal
}