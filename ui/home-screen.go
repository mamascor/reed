package ui

import (
	"lms-tui/logger"
	"github.com/rivo/tview"
)

func NewHomeScreen(app *tview.Application) (tview.Primitive, *tview.List) {
	list := tview.NewList().
		AddItem("LMS", "Lab Management System", '1', func() {
			logger.Info.Println("Navigating to LMS screen")
			lmsScreen, lmsList := NewLMSScreen(app, func() {
				// This callback runs when '+' is pressed in LMS screen
				logger.Info.Println("Returning to home screen from LMS")
				homescreen, homeList := NewHomeScreen(app)
				app.SetRoot(homescreen, true)
				app.SetFocus(homeList)
			})
			app.SetRoot(lmsScreen, true)
			app.SetFocus(lmsList)
		})

	// Container with textview and list
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("Marco Mascorro").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(list, 0, 1, true)

	container.SetBorder(true).
		SetTitle(" Home ").
		SetTitleAlign(tview.AlignCenter)

	container.SetBorderPadding(1, 1, 1, 1)

	// Center it
	vertical := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(container, 10, 1, true).
		AddItem(nil, 0, 1, false)

	horizontal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(vertical, 50, 1, true).
		AddItem(nil, 0, 1, false)

	return horizontal, list
}