// ---- File: main.go ----
package main

import (
	"log"
	"os"

	"github.com/jroimartin/gocui"
)

func main() {
	// Setup logging
	logFile, err := os.OpenFile("lazyls.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, 0666)
	if err == nil {
		log.SetOutput(logFile)
		log.SetFlags(log.LstdFlags | log.Lshortfile) // Add line numbers to logs
		log.Println("--- Application Started ---")
	} else {
		log.Println("Could not open log file, logging to stderr:", err)
	}
	defer func() {
		if logFile != nil {
			log.Println("--- Application Ended ---")
			logFile.Close()
		}
	}()

	// Get CWD
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("FATAL: Failed to get current working directory: %v", err)
	}

	// Init State
	appState := NewAppState(cwd)

	// Initial Load
	err = loadDirectoryContents(appState)
	if err != nil {
		// Logged within loadDirectoryContents if using state.SetMessage
		log.Printf("Error: Failed to initially load directory contents: %v", err)
	}

	// Init gocui
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln("FATAL: Failed to initialize gocui:", err)
	}
	defer g.Close()

	g.Cursor = false // Disable cursor globally unless needed for input
	g.Mouse = false  // Disable mouse support for now
	// We don't set global SelFg/BgColor, it's per-view
	g.Highlight = true                // Enable highlighting globally (views can override)
	g.SelFgColor = gocui.ColorGreen   // Global default for selection foreground
	g.SelBgColor = gocui.ColorDefault // Global default for selection background
	// g.ASCII = true // Uncomment if Unicode icons cause issues

	// Set Layout Manager
	g.SetManagerFunc(func(gui *gocui.Gui) error {
		// The layout function now handles view creation, updates, and focus setting
		return layout(gui, appState) // Defined in ui.go
	})

	// Set Keybindings
	if err := setupKeybindings(g, appState); err != nil { // Defined in handlers.go
		log.Panicln("FATAL: Failed to set keybindings:", err)
	}

	// Start background tasks
	go calculateStats(g, appState)

	// Initial focus setting is now handled within the layout function's logic,
	// ensuring views exist before focus is set.

	// Start main loop
	log.Println("Starting main loop...")
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln("FATAL: Main loop error:", err)
	}
	log.Println("Main loop finished.")
}
