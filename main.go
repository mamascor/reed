package main

import (
	"lms-tui/logger"
	"lms-tui/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	// Initialize logging system
	logger.InitLogger("logs/lms.log")
	logger.Info.Println("Application starting...")

	app := tview.NewApplication()

	// Global input capture for numpad key mappings
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlJ {
			// Convert Ctrl+J (numpad Enter) to regular Enter
			return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
		}
		if event.Rune() == '*' {
			// Convert * to arrow up
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		}
		if event.Rune() == '-' {
			// Convert - to arrow down
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		}
		return event
	})

	loginScreen := ui.NewLoginScreen(app, func(userID, pin string) {
		 if userID == "1234" && pin == "0000" {
			logger.Info.Printf("User logged in: %s", userID)
			homescreen, homeList := ui.NewHomeScreen(app)
			app.SetRoot(homescreen, true)
			app.SetFocus(homeList)
		 } else {
			logger.Info.Printf("Failed login attempt for user: %s", userID)
		 }
	})


	if err := app.SetRoot(loginScreen, true).Run(); err != nil {
		panic(err)
	}
}