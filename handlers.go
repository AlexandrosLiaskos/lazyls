// ---- File: handlers.go ----
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jroimartin/gocui"
)

// setupKeybindings configures all application keybindings.
func setupKeybindings(g *gocui.Gui, state *AppState) error {
	// Quit (Global)
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		// Allow 'q' to close the file content view if it's open
		if state.IsFileContentViewVisible() {
			return handleCloseFileContentView(gui, view, state) // Use the updated handler
		}
		// Allow 'q' to close the action menu if it's open
		if state.IsActionMenuVisible() {
			return handleMenuClose(gui, view, state)
		}
		return quit(gui, view) // Otherwise, quit the app
	}); err != nil {
		return err
	}

	// Esc for closing menu or file view
	if err := g.SetKeybinding(viewActionMenu, gocui.KeyEsc, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleMenuClose(gui, view, state)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewFileContent, gocui.KeyEsc, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleCloseFileContentView(gui, view, state) // Use the updated handler
	}); err != nil {
		return err
	}

	// Toggle Hidden Files (Global)
	if err := g.SetKeybinding("", '.', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		// Don't toggle if file view or menu is open
		if state.IsFileContentViewVisible() || state.IsActionMenuVisible() {
			return nil
		}
		return handleToggleHidden(gui, state)
	}); err != nil {
		return err
	}

	// Focus Switching (Global - Tab)
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		// Don't switch focus if file view or menu is open
		if state.IsFileContentViewVisible() || state.IsActionMenuVisible() {
			return nil
		}
		return handleFocusSwitch(gui, state, true) // Forward
	}); err != nil {
		return err
	}

	// --- List Navigation Keybindings (Folders and Files views) ---
	viewsToNavigate := []string{viewFolders, viewFiles}
	for _, viewName := range viewsToNavigate {
		// --- Cursor Movement (Updates Cursor & Origin) ---
		bindMove := func(key interface{}, delta int) error {
			return g.SetKeybinding(viewName, key, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
				return handleMoveCursor(gui, view, delta, state)
			})
		}
		if err := bindMove(gocui.KeyArrowDown, 1); err != nil {
			return err
		}
		if err := bindMove('j', 1); err != nil {
			return err
		}
		if err := bindMove(gocui.KeyArrowUp, -1); err != nil {
			return err
		}
		if err := bindMove('k', -1); err != nil {
			return err
		}

		// Page move
		bindPageMove := func(key interface{}, multiplier int) error {
			return g.SetKeybinding(viewName, key, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
				_, maxY := view.Size()
				pageSize := maxY - 1
				if pageSize < 1 {
					pageSize = 1
				}
				return handleMoveCursor(gui, view, multiplier*pageSize, state)
			})
		}
		if err := bindPageMove(gocui.KeyPgdn, 1); err != nil {
			return err
		}
		if err := bindPageMove(gocui.KeySpace, 1); err != nil {
			return err
		}
		if err := bindPageMove(gocui.KeyPgup, -1); err != nil {
			return err
		}
		if err := bindPageMove('b', -1); err != nil {
			return err
		}

		// Go to Top/Bottom
		bindTopBottom := func(key interface{}, toTop bool) error {
			return g.SetKeybinding(viewName, key, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
				return handleGoTopBottom(gui, view, toTop, state)
			})
		}
		if err := bindTopBottom('g', true); err != nil {
			return err
		}
		if err := bindTopBottom(gocui.KeyHome, true); err != nil {
			return err
		}
		if err := bindTopBottom('G', false); err != nil {
			return err
		}
		if err := bindTopBottom(gocui.KeyEnd, false); err != nil {
			return err
		}

		// --- Action Trigger ---
		if err := g.SetKeybinding(viewName, gocui.KeyEnter, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
			return handleEnter(gui, view, state)
		}); err != nil {
			return err
		}
	}

	// --- File Content View Scroll Keybindings ---
	fileContentViewName := viewFileContent // Use the constant

	// Line Scroll
	if err := g.SetKeybinding(fileContentViewName, gocui.KeyArrowDown, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, 1, false)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, 'j', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, 1, false)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, gocui.KeyArrowUp, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, -1, false)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, 'k', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, -1, false)
	}); err != nil {
		return err
	}

	// Page Scroll
	if err := g.SetKeybinding(fileContentViewName, gocui.KeyPgdn, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		_, maxY := view.Size()
		pageSize := maxY - 1
		if pageSize < 1 {
			pageSize = 1
		}
		return handleScrollFileContentView(gui, view, state, pageSize, true)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, gocui.KeySpace, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		_, maxY := view.Size()
		pageSize := maxY - 1
		if pageSize < 1 {
			pageSize = 1
		}
		return handleScrollFileContentView(gui, view, state, pageSize, true)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, gocui.KeyPgup, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		_, maxY := view.Size()
		pageSize := maxY - 1
		if pageSize < 1 {
			pageSize = 1
		}
		return handleScrollFileContentView(gui, view, state, -pageSize, true)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, 'b', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		_, maxY := view.Size()
		pageSize := maxY - 1
		if pageSize < 1 {
			pageSize = 1
		}
		return handleScrollFileContentView(gui, view, state, -pageSize, true)
	}); err != nil {
		return err
	}

	// Go To Top/Bottom (Use large delta values as signal)
	if err := g.SetKeybinding(fileContentViewName, 'g', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, -999999, true)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, gocui.KeyHome, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, -999999, true)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, 'G', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, 999999, true)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(fileContentViewName, gocui.KeyEnd, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleScrollFileContentView(gui, view, state, 999999, true)
	}); err != nil {
		return err
	}

	// --- Action Menu Keybindings ---
	if err := g.SetKeybinding(viewActionMenu, gocui.KeyArrowDown, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleMenuNavigate(gui, view, 1, state)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewActionMenu, 'j', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleMenuNavigate(gui, view, 1, state)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewActionMenu, gocui.KeyArrowUp, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleMenuNavigate(gui, view, -1, state)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewActionMenu, 'k', gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleMenuNavigate(gui, view, -1, state)
	}); err != nil {
		return err
	}
	if err := g.SetKeybinding(viewActionMenu, gocui.KeyEnter, gocui.ModNone, func(gui *gocui.Gui, view *gocui.View) error {
		return handleMenuSelect(gui, view, state)
	}); err != nil {
		return err
	}

	return nil
}

// quit is the keybinding handler for quitting the application.
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

// handleToggleHidden processes the toggle hidden keypress.
func handleToggleHidden(g *gocui.Gui, state *AppState) error {
	state.ToggleHidden()
	// Reset focus to folders view for consistency after toggle
	if _, err := g.SetCurrentView(viewFolders); err != nil {
		log.Printf("Warning: Failed to set focus to folders after toggle: %v", err)
	}
	// Explicitly update the view that will gain focus to reset its cursor display
	// Update: Calling g.Update is simpler and ensures layout handles everything
	// updateListView(g, state, viewFolders) // Force update
	g.Update(func(gui *gocui.Gui) error {
		return nil // Trigger layout update
	})

	// Let layout handle the rest
	return nil
}

// handleMoveCursor handles arrow keys, page up/down, space, j, k, etc. for list views.
func handleMoveCursor(g *gocui.Gui, v *gocui.View, delta int, state *AppState) error {
	if v == nil {
		return nil
	}
	_, viewHeight := v.Size()
	changed := state.moveCursorAndOrigin(v.Name(), delta, viewHeight)
	// Only trigger update if state actually changed
	if changed {
		g.Update(func(gui *gocui.Gui) error {
			return nil // Trigger layout update
		})
	}
	return nil
}

// handleGoTopBottom handles 'g', 'G', Home, End keys for list views.
func handleGoTopBottom(g *gocui.Gui, v *gocui.View, toTop bool, state *AppState) error {
	if v == nil {
		return nil
	}
	_, viewHeight := v.Size()
	list := state.GetCurrentList(v.Name())
	listLen := len(list)
	newCursorY := 0
	if !toTop {
		if listLen > 0 {
			newCursorY = listLen - 1
		} // else stays 0
	}

	changed := state.setCursorAndOrigin(v.Name(), newCursorY, viewHeight)
	if changed {
		g.Update(func(gui *gocui.Gui) error {
			return nil // Trigger layout update
		})
	}
	return nil
}

// handleEnter opens the action menu for the selected item.
func handleEnter(g *gocui.Gui, v *gocui.View, state *AppState) error {
	if v == nil {
		return nil
	}

	viewName := v.Name()
	currentList := state.GetCurrentList(viewName)
	cursorY := state.GetCurrentCursorY(viewName)

	if len(currentList) == 0 {
		return nil // Cannot select anything from an empty list
	}

	if cursorY < 0 || cursorY >= len(currentList) {
		log.Printf("Enter pressed with invalid cursor index %d for list length %d", cursorY, len(currentList))
		return nil // Index out of bounds
	}

	selectedItem := currentList[cursorY]

	// Define menu options based on item type
	var options []ActionMenuItem
	options = append(options, ActionMenuItem{Label: "Copy Full Path", ActionFn: copyFullPath})
	options = append(options, ActionMenuItem{Label: "Copy Relative Path", ActionFn: copyRelativePath})
	if !selectedItem.IsDir {
		options = append(options, ActionMenuItem{Label: "View Content", ActionFn: viewFileContentAction})
		options = append(options, ActionMenuItem{Label: "Copy Content (UTF-8)", ActionFn: copyContent})
	}
	options = append(options, ActionMenuItem{Label: "Cancel", ActionFn: func(*gocui.Gui, FileInfo, *AppState) error { return nil }}) // No-op cancel

	if len(options) > 0 {
		state.OpenActionMenu(selectedItem, options, viewName)
		g.Update(func(gui *gocui.Gui) error {
			return nil // Trigger layout update to show menu
		})
	}

	return nil
}

// handleMenuNavigate moves the selection in the action menu.
func handleMenuNavigate(g *gocui.Gui, v *gocui.View, delta int, state *AppState) error {
	state.NavigateActionMenu(delta)
	g.Update(func(gui *gocui.Gui) error {
		return nil // Trigger layout update to redraw menu
	})
	return nil
}

// handleMenuSelect executes the selected action from the menu.
func handleMenuSelect(g *gocui.Gui, v *gocui.View, state *AppState) error {
	options := state.GetActionMenuOptions()
	selectedIdx := state.GetActionMenuSelectedIdx()
	targetItem := state.GetActionMenuItemTarget()

	if selectedIdx < 0 || selectedIdx >= len(options) {
		log.Printf("Menu selection out of bounds: %d", selectedIdx)
		return handleMenuClose(g, v, state) // Close menu if selection is invalid
	}

	selectedOption := options[selectedIdx]
	actionLabel := selectedOption.Label // Store label before potential state change

	// Close the menu *before* executing the action (usually)
	// except for actions that open a new view like "View Content"
	closeMenuFirst := actionLabel != "View Content"
	if closeMenuFirst {
		// Need to close menu and trigger update *before* executing action
		state.CloseActionMenu()
		g.Update(func(gui *gocui.Gui) error { return nil }) // Ensure menu disappears
	}

	// Execute the action
	actionErr := error(nil)
	if selectedOption.ActionFn != nil {
		// Pass the Gui instance to the action function
		actionErr = selectedOption.ActionFn(g, targetItem, state)
	}

	// Post-action state/UI updates
	if actionErr != nil {
		log.Printf("Action '%s' failed for %s: %v", actionLabel, targetItem.Name, actionErr)
		errMsg := fmt.Sprintf("Error: %s - %v", actionLabel, actionErr)
		state.SetMessage(trimError(fmt.Errorf(errMsg)))
		// If the failed action was view content, we still need to ensure the menu closes.
		if actionLabel == "View Content" && state.IsActionMenuVisible() {
			state.CloseActionMenu() // Force close state
		}
		g.Update(func(gui *gocui.Gui) error { return nil }) // Update UI for error message and potential menu close
	} else if actionLabel == "View Content" {
		// View Content Action was successful, state.SetFileContentView was called by the action.
		// Now close the action menu *after* successfully preparing the content view state.
		state.CloseActionMenu()
		state.ClearMessage() // Clear message after opening viewer
		// Trigger layout update to show content view and hide menu
		g.Update(func(gui *gocui.Gui) error { return nil })
	} else if actionLabel != "Cancel" {
		// Successful action other than View Content or Cancel
		successMsg := fmt.Sprintf("'%s' copied to clipboard", actionLabel)
		if strings.HasPrefix(actionLabel, "Copy Content") {
			successMsg = fmt.Sprintf("Content of '%s' copied", targetItem.Name)
		}
		state.SetMessage(successMsg)
		// Menu was already closed and updated if closeMenuFirst was true.
		// If it wasn't (e.g. cancel action), we still need an update for the message.
		if !closeMenuFirst {
			g.Update(func(gui *gocui.Gui) error { return nil }) // Update UI for success message
		}
	} else {
		// Cancel action - menu should be closed if closeMenuFirst was true
		// If not (logic error?), ensure update happens.
		if !closeMenuFirst && state.IsActionMenuVisible() { // Should not happen for Cancel, but defensively...
			state.CloseActionMenu()
			g.Update(func(gui *gocui.Gui) error { return nil })
		}
	}

	return nil // Errors handled via state.SetMessage
}

// handleMenuClose closes the action menu and returns focus.
func handleMenuClose(g *gocui.Gui, v *gocui.View, state *AppState) error {
	prevFocus := state.GetPreviousFocusView() // Get focus target BEFORE clearing state
	state.CloseActionMenu()
	state.ClearMessage() // Clear any action-related messages when menu closes

	// Restore focus immediately
	targetFocusView := viewFolders // Default fallback
	if prevFocus != "" {
		// Quick check if the view still exists (it should)
		if _, err := g.View(prevFocus); err == nil {
			targetFocusView = prevFocus
		} else {
			log.Printf("Warning: Previous focus view '%s' not found, defaulting to '%s'", prevFocus, viewFolders)
		}
	} else {
		log.Println("Warning: Previous focus view unknown when closing menu, defaulting to folders")
	}

	if _, err := g.SetCurrentView(targetFocusView); err != nil {
		log.Printf("Error restoring focus to %s after closing menu: %v", targetFocusView, err)
		// Attempt final fallback if setting target failed
		if targetFocusView != viewFolders && g.CurrentView().Name() != viewFolders {
			if _, err := g.SetCurrentView(viewFolders); err != nil {
				log.Printf("Error setting final fallback focus to %s: %v", viewFolders, err)
			}
		}
	}

	// Trigger layout update AFTER setting focus
	g.Update(func(gui *gocui.Gui) error {
		return nil // Trigger layout update to hide menu and restore focus
	})
	return nil
}

// handleCloseFileContentView updates the state to hide the content view.
func handleCloseFileContentView(g *gocui.Gui, v *gocui.View, state *AppState) error {
	prevFocus := state.GetFileContentViewPrevFocus() // Get focus target BEFORE clearing state
	state.CloseFileContentView()
	state.ClearMessage() // Clear any messages when closing the viewer

	// Restore focus immediately
	targetFocusView := viewFolders // Default fallback
	if prevFocus != "" {
		// Quick check if the view still exists (it should)
		if _, err := g.View(prevFocus); err == nil {
			targetFocusView = prevFocus
		} else {
			log.Printf("Warning: Previous focus view '%s' not found, defaulting to '%s'", prevFocus, viewFolders)
		}
	} else {
		log.Println("Warning: Previous focus view unknown when closing file content, defaulting to folders")
	}

	if _, err := g.SetCurrentView(targetFocusView); err != nil {
		log.Printf("Error restoring focus to %s after closing file view: %v", targetFocusView, err)
		// Attempt final fallback if setting target failed (no need to check if targetFocusView != viewFolders as it's already the fallback)
		if g.CurrentView().Name() != viewFolders { // Prevent unnecessary SetCurrentView if already on fallback
			if _, err := g.SetCurrentView(viewFolders); err != nil {
				log.Printf("Error setting final fallback focus to %s: %v", viewFolders, err)
			}
		}
	}

	// Trigger layout update AFTER setting focus
	g.Update(func(gui *gocui.Gui) error {
		return nil
	})
	return nil
}

// handleFocusSwitch switches focus between folders and files views using Tab.
func handleFocusSwitch(g *gocui.Gui, state *AppState, forward bool) error {
	// Don't switch focus if the action menu or file view is visible
	if state.IsActionMenuVisible() || state.IsFileContentViewVisible() {
		return nil
	}

	currentView := g.CurrentView()
	if currentView == nil {
		_, err := g.SetCurrentView(viewFolders) // Default to folders if no focus
		// Trigger UI update to reflect focus change (highlighting)
		if err == nil {
			g.Update(func(gui *gocui.Gui) error { return nil })
		}
		return err
	}

	views := []string{viewFolders, viewFiles} // The views we cycle through
	currentIdx := -1
	for i, name := range views {
		if name == currentView.Name() {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 { // Current view is not one of the cyclable views
		_, err := g.SetCurrentView(viewFolders) // Default to folders
		if err == nil {
			g.Update(func(gui *gocui.Gui) error { return nil })
		}
		return err
	}

	nextIdx := 0
	if forward {
		nextIdx = (currentIdx + 1) % len(views)
	} else {
		// This case is currently unreachable as Shift+Tab is not bound
		nextIdx = (currentIdx - 1 + len(views)) % len(views)
	}

	nextViewName := views[nextIdx]

	_, err := g.SetCurrentView(nextViewName)
	if err != nil {
		log.Printf("Error switching focus to %s: %v", nextViewName, err)
	} else {
		// Trigger UI update to reflect focus change (highlighting)
		g.Update(func(gui *gocui.Gui) error {
			return nil // Trigger layout update
		})
	}

	// Title highlighting is handled dynamically in updateListView based on g.CurrentView()
	return err // Return potential error from SetCurrentView
}

// --- Action Implementations ---

// copyFullPath copies the item's absolute path to the clipboard.
func copyFullPath(g *gocui.Gui, item FileInfo, state *AppState) error {
	return copyToClipboard(item.Path)
}

// copyRelativePath copies the item's path relative to CWD to the clipboard.
func copyRelativePath(g *gocui.Gui, item FileInfo, state *AppState) error {
	relPath, err := filepath.Rel(state.Cwd(), item.Path)
	if err != nil {
		log.Printf("Error getting relative path for '%s' from '%s': %v", item.Path, state.Cwd(), err)
		return fmt.Errorf("could not determine relative path")
	}
	return copyToClipboard(relPath)
}

// copyContent reads a file's content and copies it to the clipboard.
func copyContent(g *gocui.Gui, item FileInfo, state *AppState) error {
	if item.IsDir {
		return fmt.Errorf("cannot copy content of a directory")
	}

	// Use the shared ReadFileWithLimit function
	content, err := ReadFileWithLimit(item.Path, maxCopySize)
	if err != nil {
		return err // Error already formatted by ReadFileWithLimit
	}

	if content == nil { // File was empty
		return copyToClipboard("")
	}

	// Clipboard interaction might fail with non-UTF8, but let the clipboard library handle it.
	return copyToClipboard(string(content))
}

// viewFileContentAction reads a file and updates the state to show the content view.
// NOTE: This function now only updates the state. The menu closing and UI update
// are handled in handleMenuSelect *after* this function returns successfully.
func viewFileContentAction(g *gocui.Gui, item FileInfo, state *AppState) error {
	if item.IsDir {
		return fmt.Errorf("cannot view content of a directory")
	}

	// Use the shared ReadFileWithLimit function
	contentBytes, err := ReadFileWithLimit(item.Path, maxViewSize) // Use maxViewSize limit
	if err != nil {
		return err // Return the formatted error
	}

	var content string
	if contentBytes != nil {
		// Naive check for binary content (look for null bytes).
		isNullTerminated := false
		for _, b := range contentBytes {
			if b == 0 {
				isNullTerminated = true
				break
			}
		}

		if isNullTerminated {
			return fmt.Errorf("cannot display binary file content")
		} else {
			content = string(contentBytes)
			// Replace tabs with spaces for consistent rendering
			content = strings.ReplaceAll(content, "\t", "    ")
		}

	} else {
		content = "[Empty File]" // Indicate empty file explicitly
	}

	// Get the current focus *before* the menu closes in handleMenuSelect
	// This requires knowing the focus *before* the action menu was opened.
	currentFocus := state.GetPreviousFocusView() // Focus from before menu opened
	if currentFocus == "" {                      // Fallback if state wasn't set correctly
		log.Println("Warning: Previous focus view unknown when opening file content, defaulting to folders")
		currentFocus = viewFolders
	}

	// Prepare state for the content view
	state.SetFileContentView(item.Name, content, currentFocus)

	// IMPORTANT: Do NOT trigger g.Update here.
	// It will be triggered in handleMenuSelect after this function returns successfully,
	// ensuring the menu closes *and* the content view appears in one layout pass.
	return nil
}

// --- File Content View Handlers ---

// handleScrollFileContentView scrolls the content view by delta lines.
func handleScrollFileContentView(g *gocui.Gui, v *gocui.View, state *AppState, delta int, isPageScroll bool) error {
	if v == nil || !state.IsFileContentViewVisible() {
		return nil
	}
	_, viewHeight := v.Size()
	totalLines := state.GetFileContentViewTotalLines()

	// Disable scrolling if content fits in view
	if totalLines <= viewHeight {
		return nil
	}

	// Adjust delta for Go To Top/Bottom based on current origin
	if isPageScroll {
		currentOrigin := state.GetFileContentViewOriginY()
		if delta <= -totalLines { // Request to go to top ('g', Home)
			delta = -currentOrigin
		} else if delta >= totalLines { // Request to go to bottom ('G', End)
			maxOriginY := totalLines - viewHeight
			if maxOriginY < 0 {
				maxOriginY = 0
			}
			delta = maxOriginY - currentOrigin
		}
	}

	// Update state's originY - the ScrollFileContentView method handles bounds checking
	state.ScrollFileContentView(delta, viewHeight)

	g.Update(func(gui *gocui.Gui) error {
		return nil
	})
	return nil
}

// --- Helper for Reading Files ---

const maxCopySize = 5 * 1024 * 1024  // 5 MB limit for copying
const maxViewSize = 20 * 1024 * 1024 // 20 MB limit for viewing

// ReadFileWithLimit reads a file up to a specified size limit.
// Returns the content as bytes, or nil if empty, or an error.
func ReadFileWithLimit(path string, limitBytes int64) ([]byte, error) {
	info, err := os.Stat(path) // Use Stat, not Lstat, to get size of actual file if symlink
	if err != nil {
		return nil, fmt.Errorf("could not stat file: %w", err)
	}
	if info.IsDir() { // Add explicit check for directory
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > limitBytes {
		limitMB := limitBytes / (1024 * 1024)
		return nil, fmt.Errorf("file too large (> %d MiB)", limitMB)
	}
	if info.Size() == 0 {
		return nil, nil // Return nil for empty file, no error
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %w", err)
	}
	return content, nil
}
