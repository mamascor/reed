package main

import (
	"lms-tui/logger"
	"lms-tui/pkg"
	"lms-tui/ui"
	"os/exec"
	"time"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	// Initialize logging system
	logger.InitLogger("logs/lms.log")
	logger.Info.Println("Application starting...")

	// Load configuration from config.json
	if err := pkg.LoadConfig("config.json"); err != nil {
		logger.Info.Printf("Failed to load config, using defaults: %v", err)
	}

	// Prevent screen from sleeping while app is running (Wayland/GNOME)
	inhibitCmd := exec.Command("gnome-session-inhibit", "--inhibit", "idle", "--reason", "LMS TUI Application Active", "sleep", "infinity")
	if err := inhibitCmd.Start(); err != nil {
		logger.Info.Printf("Warning: Could not inhibit screen sleep: %v", err)
	} else {
		logger.Info.Println("Screen sleep prevention enabled")
	}

	// Kill the inhibit process when app exits
	defer func() {
		if inhibitCmd.Process != nil {
			inhibitCmd.Process.Kill()
			inhibitCmd.Wait()
			logger.Info.Println("Screen sleep prevention disabled")
		}
	}()

	// Keep Num Lock ON while app is running
	exec.Command("numlockx", "on").Run()
	logger.Info.Println("Num Lock enabled")

	// Monitor and keep Num Lock on (in case user accidentally toggles it)
	stopNumLock := make(chan bool)
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stopNumLock:
				return
			case <-ticker.C:
				exec.Command("numlockx", "on").Run()
			}
		}
	}()
	defer func() {
		stopNumLock <- true
	}()

	// Give the inhibit process time to start
	time.Sleep(100 * time.Millisecond)

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