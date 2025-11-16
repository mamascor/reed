package ui

import (
	"lms-tui/logger"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)


func NewLMSScreen(app *tview.Application, onBack func()) (tview.Primitive, *tview.List) {
	list := tview.NewList().
		AddItem("View Available Jobs", "View all available jobs", '1', func() {
			logger.Info.Println("Navigating to View Jobs screen")
			newJobScreen, newJobTable := NewViewJobScreen(app, func() {
				// Go back to LMS screen
				logger.Info.Println("Returning to LMS screen from View Jobs")
				lmsScreen, lmsList := NewLMSScreen(app, onBack)
				app.SetRoot(lmsScreen, true)
				app.SetFocus(lmsList)
			})
			app.SetRoot(newJobScreen, true)
			app.SetFocus(newJobTable)
		}).
		AddItem("Pull Job", "Pull a job from the queue", '2', func() {
			logger.Info.Println("Navigating to Pull Job List screen")
			pullJobScreen, pullJobTable := NewPullJobListScreen(app, func() {
				// Go back to LMS screen
				logger.Info.Println("Returning to LMS screen from Pull Job List")
				lmsScreen, lmsList := NewLMSScreen(app, onBack)
				app.SetRoot(lmsScreen, true)
				app.SetFocus(lmsList)
			})
			app.SetRoot(pullJobScreen, true)
			app.SetFocus(pullJobTable)
		}).
		AddItem("Morning Count", "Measure can weights in the morning", '3', func() {
			logger.Info.Println("Navigating to Morning Count screen")
			morningCountScreen := NewMorningCountScreen(app, func() {
				// Go back to LMS screen
				logger.Info.Println("Returning to LMS screen from Morning Count")
				lmsScreen, lmsList := NewLMSScreen(app, onBack)
				app.SetRoot(lmsScreen, true)
				app.SetFocus(lmsList)
			})
			app.SetRoot(morningCountScreen, true)
		})

	// Container with textview and list
	container := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetText("LMS Screen").SetTextAlign(tview.AlignCenter), 1, 0, false).
		AddItem(list, 0, 1, true)

	container.SetBorder(true).
		SetTitle(" LMS ").
		SetTitleAlign(tview.AlignCenter)

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

	horizontal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
      	if event.Rune() == '+' {
          	onBack()  // Call the callback
        	return nil  // Consume the event
      	}
      	return event  // Pass through other keys
  	})

	return horizontal, list
}

