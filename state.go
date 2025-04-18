// ---- File: state.go ----
package main

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/jroimartin/gocui"
)

// FileInfo holds processed information about a file or directory.
type FileInfo struct {
	Name  string
	Path  string // Full path for size calculation/access
	IsDir bool
	Size  int64 // Only calculated for files during largest file scan
	Icon  string
}

// ActionMenuItem defines an option in the action menu.
type ActionMenuItem struct {
	Label    string
	ActionFn func(g *gocui.Gui, item FileInfo, state *AppState) error // Function to execute, now includes *gocui.Gui
}

// AppState holds the application's state.
type AppState struct {
	sync.RWMutex // Embed RWMutex for protecting state access

	cwd          string
	visibleFiles []FileInfo
	visibleDirs  []FileInfo
	hiddenFiles  []FileInfo
	hiddenDirs   []FileInfo
	showHidden   bool

	// Stats related fields
	totalSize      int64
	largestFile    FileInfo
	gitStatus      string
	isLoadingStats bool
	statsError     error // Store errors from background tasks

	// UI related fields - Separate origins and cursors for each list
	visibleFoldersOriginY int
	visibleFilesOriginY   int
	hiddenFoldersOriginY  int
	hiddenFilesOriginY    int
	visibleFoldersCursorY int // Absolute index in the list
	visibleFilesCursorY   int // Absolute index in the list
	hiddenFoldersCursorY  int // Absolute index in the list
	hiddenFilesCursorY    int // Absolute index in the list

	// Action Menu State
	isActionMenuVisible   bool
	actionMenuItemTarget  FileInfo         // The file/folder the menu is for
	actionMenuOptions     []ActionMenuItem // Options with actions
	actionMenuSelectedIdx int
	previousFocusView     string // View to return focus to after closing menu

	// File Content View State
	isFileContentViewVisible  bool
	fileContentViewFileName   string // Name of the file being viewed
	fileContentViewContent    string // Content to display (can be large)
	fileContentViewTotalLines int    // Total lines in content for scrolling limit
	fileContentViewOriginY    int    // Scroll position (top visible line index)
	fileContentViewPrevFocus  string // View to return focus to after closing content view

	// Help View State
	helpVisible bool

	// Confirm Delete State
	confirmDeleteVisible bool
	itemToDelete         *FileInfo // Store the item pending deletion

	// Message Bar State
	lastMessage string // For temporary messages (e.g., copy status)
	// messageTimer *sync.Mutex // Using mutex as a simple timer signal mechanism (needs improvement for real timer)
}

// NewAppState creates and initializes a new AppState.
func NewAppState(cwd string) *AppState {
	return &AppState{
		cwd:            cwd,
		showHidden:     false,
		isLoadingStats: true, // Start in loading state
		gitStatus:      "Checking...",
		totalSize:      -1, // Indicate not calculated yet
		// Initialize all origins and cursors to 0
		visibleFoldersOriginY: 0,
		visibleFilesOriginY:   0,
		hiddenFoldersOriginY:  0,
		hiddenFilesOriginY:    0,
		visibleFoldersCursorY: 0,
		visibleFilesCursorY:   0,
		hiddenFoldersCursorY:  0,
		hiddenFilesCursorY:    0,
		// Initialize File Content View state
		isFileContentViewVisible: false,
		fileContentViewOriginY:   0,
		// messageTimer:             &sync.Mutex{}, // Initialize the mutex // Removed timer for now
	}
}

// --- State Query Methods (Read operations) ---

func (s *AppState) Cwd() string {
	s.RLock()
	defer s.RUnlock()
	return s.cwd
}

func (s *AppState) BaseDir() string {
	s.RLock()
	defer s.RUnlock()
	return filepath.Base(s.cwd)
}

func (s *AppState) IsLoadingStats() bool {
	s.RLock()
	defer s.RUnlock()
	return s.isLoadingStats
}

func (s *AppState) Stats() (totalSize int64, largestFile FileInfo, gitStatus string, statsErr error) {
	s.RLock()
	defer s.RUnlock()
	return s.totalSize, s.largestFile, s.gitStatus, s.statsError
}

func (s *AppState) IsShowingHidden() bool {
	s.RLock()
	defer s.RUnlock()
	return s.showHidden
}

func (s *AppState) VisibleDirs() []FileInfo {
	s.RLock()
	defer s.RUnlock()
	dirs := make([]FileInfo, len(s.visibleDirs))
	copy(dirs, s.visibleDirs)
	return dirs
}

func (s *AppState) VisibleFiles() []FileInfo {
	s.RLock()
	defer s.RUnlock()
	files := make([]FileInfo, len(s.visibleFiles))
	copy(files, s.visibleFiles)
	return files
}

func (s *AppState) HiddenDirs() []FileInfo {
	s.RLock()
	defer s.RUnlock()
	dirs := make([]FileInfo, len(s.hiddenDirs))
	copy(dirs, s.hiddenDirs)
	return dirs
}

func (s *AppState) HiddenFiles() []FileInfo {
	s.RLock()
	defer s.RUnlock()
	files := make([]FileInfo, len(s.hiddenFiles))
	copy(files, s.hiddenFiles)
	return files
}

// --- Getters for UI state ---

func (s *AppState) VisibleFoldersOriginY() int {
	s.RLock()
	defer s.RUnlock()
	return s.visibleFoldersOriginY
}

func (s *AppState) VisibleFilesOriginY() int {
	s.RLock()
	defer s.RUnlock()
	return s.visibleFilesOriginY
}

func (s *AppState) HiddenFoldersOriginY() int {
	s.RLock()
	defer s.RUnlock()
	return s.hiddenFoldersOriginY
}

func (s *AppState) HiddenFilesOriginY() int {
	s.RLock()
	defer s.RUnlock()
	return s.hiddenFilesOriginY
}

func (s *AppState) VisibleFoldersCursorY() int {
	s.RLock()
	defer s.RUnlock()
	return s.visibleFoldersCursorY
}

func (s *AppState) VisibleFilesCursorY() int {
	s.RLock()
	defer s.RUnlock()
	return s.visibleFilesCursorY
}

func (s *AppState) HiddenFoldersCursorY() int {
	s.RLock()
	defer s.RUnlock()
	return s.hiddenFoldersCursorY
}

func (s *AppState) HiddenFilesCursorY() int {
	s.RLock()
	defer s.RUnlock()
	return s.hiddenFilesCursorY
}

// GetCurrentCursorY returns the absolute cursor index for the currently relevant list.
func (s *AppState) GetCurrentCursorY(viewName string) int {
	s.RLock()
	defer s.RUnlock()
	isHidden := s.showHidden
	switch viewName {
	case viewFolders:
		if isHidden {
			return s.hiddenFoldersCursorY
		}
		return s.visibleFoldersCursorY
	case viewFiles:
		if isHidden {
			return s.hiddenFilesCursorY
		}
		return s.visibleFilesCursorY
	}
	return 0 // Should not happen
}

// GetCurrentOriginY returns the origin Y for the currently relevant list.
func (s *AppState) GetCurrentOriginY(viewName string) int {
	s.RLock()
	defer s.RUnlock()
	isHidden := s.showHidden
	switch viewName {
	case viewFolders:
		if isHidden {
			return s.hiddenFoldersOriginY
		}
		return s.visibleFoldersOriginY
	case viewFiles:
		if isHidden {
			return s.hiddenFilesOriginY
		}
		return s.visibleFilesOriginY
	}
	return 0 // Should not happen
}

// GetCurrentList returns the currently relevant list based on view name and hidden state.
func (s *AppState) GetCurrentList(viewName string) []FileInfo {
	s.RLock()
	defer s.RUnlock()
	isHidden := s.showHidden
	switch viewName {
	case viewFolders:
		if isHidden {
			// Return copy
			dirs := make([]FileInfo, len(s.hiddenDirs))
			copy(dirs, s.hiddenDirs)
			return dirs
		}
		dirs := make([]FileInfo, len(s.visibleDirs))
		copy(dirs, s.visibleDirs)
		return dirs
	case viewFiles:
		if isHidden {
			files := make([]FileInfo, len(s.hiddenFiles))
			copy(files, s.hiddenFiles)
			return files
		}
		files := make([]FileInfo, len(s.visibleFiles))
		copy(files, s.visibleFiles)
		return files
	}
	return nil // Should not happen
}

// --- Action Menu Getters ---
func (s *AppState) IsActionMenuVisible() bool {
	s.RLock()
	defer s.RUnlock()
	return s.isActionMenuVisible
}

func (s *AppState) GetActionMenuItemTarget() FileInfo {
	s.RLock()
	defer s.RUnlock()
	return s.actionMenuItemTarget
}

func (s *AppState) GetActionMenuOptions() []ActionMenuItem {
	s.RLock()
	defer s.RUnlock()
	// Return a copy
	opts := make([]ActionMenuItem, len(s.actionMenuOptions))
	copy(opts, s.actionMenuOptions)
	return opts
}

func (s *AppState) GetActionMenuSelectedIdx() int {
	s.RLock()
	defer s.RUnlock()
	return s.actionMenuSelectedIdx
}

func (s *AppState) GetPreviousFocusView() string {
	s.RLock()
	defer s.RUnlock()
	return s.previousFocusView
}

// --- File Content View Getters ---
func (s *AppState) IsFileContentViewVisible() bool {
	s.RLock()
	defer s.RUnlock()
	return s.isFileContentViewVisible
}

func (s *AppState) GetFileContentViewFileName() string {
	s.RLock()
	defer s.RUnlock()
	return s.fileContentViewFileName
}

func (s *AppState) GetFileContentViewContent() string {
	s.RLock()
	defer s.RUnlock()
	return s.fileContentViewContent
}

func (s *AppState) GetFileContentViewOriginY() int {
	s.RLock()
	defer s.RUnlock()
	return s.fileContentViewOriginY
}

func (s *AppState) GetFileContentViewPrevFocus() string {
	s.RLock()
	defer s.RUnlock()
	return s.fileContentViewPrevFocus
}

func (s *AppState) GetFileContentViewTotalLines() int {
	s.RLock()
	defer s.RUnlock()
	return s.fileContentViewTotalLines
}

// --- Help View Getters ---
func (s *AppState) IsHelpVisible() bool {
	s.RLock()
	defer s.RUnlock()
	return s.helpVisible
}

// --- Confirm Delete Getters ---
func (s *AppState) IsConfirmDeleteVisible() bool {
	s.RLock()
	defer s.RUnlock()
	return s.confirmDeleteVisible
}

func (s *AppState) GetItemToDelete() *FileInfo {
	s.RLock()
	defer s.RUnlock()
	return s.itemToDelete
}

// --- Message Bar Getters ---
func (s *AppState) GetLastMessage() string {
	s.RLock()
	defer s.RUnlock()
	return s.lastMessage
}

// --- State Modification Methods (Write operations) ---

// SetDirectoryContents updates the file/dir lists and resets cursors/origins.
func (s *AppState) SetDirectoryContents(visibleDirs, visibleFiles, hiddenDirs, hiddenFiles []FileInfo) {
	s.Lock()
	defer s.Unlock()
	s.visibleDirs = visibleDirs
	s.visibleFiles = visibleFiles
	s.hiddenDirs = hiddenDirs
	s.hiddenFiles = hiddenFiles

	// Reset scrolls and cursors whenever content changes
	s.visibleFoldersOriginY = 0
	s.visibleFilesOriginY = 0
	s.hiddenFoldersOriginY = 0
	s.hiddenFilesOriginY = 0
	s.visibleFoldersCursorY = 0
	s.visibleFilesCursorY = 0
	s.hiddenFoldersCursorY = 0
	s.hiddenFilesCursorY = 0
}

// ToggleHidden flips the hidden file visibility and resets scrolls/cursors for the activated views.
func (s *AppState) ToggleHidden() bool {
	s.Lock()
	defer s.Unlock()
	s.showHidden = !s.showHidden
	// Reset scroll and cursor for *both* sets of views for simplicity
	s.visibleFoldersOriginY = 0
	s.visibleFilesOriginY = 0
	s.hiddenFoldersOriginY = 0
	s.hiddenFilesOriginY = 0
	s.visibleFoldersCursorY = 0
	s.visibleFilesCursorY = 0
	s.hiddenFoldersCursorY = 0
	s.hiddenFilesCursorY = 0

	return s.showHidden // Return new state
}

// SetStatsLoading marks the application as loading stats.
func (s *AppState) SetStatsLoading() {
	s.Lock()
	defer s.Unlock()
	s.isLoadingStats = true
	s.gitStatus = "Calculating..." // Provide immediate feedback
	s.totalSize = -1               // Reset size indicator
	s.largestFile = FileInfo{}
	s.statsError = nil
}

// SetStatsResults updates the state after stats calculation finishes.
func (s *AppState) SetStatsResults(totalSize int64, largestFile FileInfo, gitStatus string, err error) {
	s.Lock()
	defer s.Unlock()
	s.totalSize = totalSize
	s.largestFile = largestFile
	s.gitStatus = gitStatus
	s.isLoadingStats = false
	s.statsError = err
	if err != nil && s.totalSize != -2 { // Ensure error state if err is present
		s.totalSize = -2
	}
}

// SetMessage temporarily sets a message to be displayed (e.g., in status bar).
func (s *AppState) SetMessage(msg string) {
	s.Lock()
	defer s.Unlock()
	s.lastMessage = msg
	// TODO: Implement a timer to clear the message after a delay
}

// ClearMessage clears the temporary message.
func (s *AppState) ClearMessage() {
	s.Lock()
	defer s.Unlock()
	s.lastMessage = ""
}

// --- List View Scrolling and Cursor Movement ---

// moveCursorAndOrigin updates the cursor and origin for the relevant list view.
// Returns true if the state changed.
func (s *AppState) moveCursorAndOrigin(viewName string, delta int, viewHeight int) bool {
	s.Lock()
	defer s.Unlock()

	var currentList []FileInfo
	var pOriginY *int
	var pCursorY *int

	// Select the correct state variables based on viewName and showHidden
	isHidden := s.showHidden
	switch viewName {
	case viewFolders:
		if isHidden {
			currentList = s.hiddenDirs
			pOriginY = &s.hiddenFoldersOriginY
			pCursorY = &s.hiddenFoldersCursorY
		} else {
			currentList = s.visibleDirs
			pOriginY = &s.visibleFoldersOriginY
			pCursorY = &s.visibleFoldersCursorY
		}
	case viewFiles:
		if isHidden {
			currentList = s.hiddenFiles
			pOriginY = &s.hiddenFilesOriginY
			pCursorY = &s.hiddenFilesCursorY
		} else {
			currentList = s.visibleFiles
			pOriginY = &s.visibleFilesOriginY
			pCursorY = &s.visibleFilesCursorY
		}
	default:
		return false // Invalid view name
	}

	listLen := len(currentList)
	if listLen <= 0 {
		changed := *pOriginY != 0 || *pCursorY != 0
		*pOriginY = 0
		*pCursorY = 0
		return changed
	}

	oldOriginY := *pOriginY
	oldCursorY := *pCursorY

	// 1. Calculate new cursor position
	newCursorY := oldCursorY + delta
	if newCursorY < 0 {
		newCursorY = 0
	}
	if newCursorY >= listLen {
		newCursorY = listLen - 1
	}

	// 2. Calculate new origin based on cursor position
	newOriginY := oldOriginY
	if newCursorY < newOriginY { // Cursor moved above the visible area
		newOriginY = newCursorY
	} else if newCursorY >= newOriginY+viewHeight { // Cursor moved below the visible area
		newOriginY = newCursorY - viewHeight + 1
	}

	// 3. Validate and clamp origin (in case of page jumps or short lists)
	maxOriginY := listLen - viewHeight
	if maxOriginY < 0 {
		maxOriginY = 0
	}
	if newOriginY > maxOriginY {
		newOriginY = maxOriginY
	}
	if newOriginY < 0 {
		newOriginY = 0
	}

	// 4. Update state if changed
	changed := oldCursorY != newCursorY || oldOriginY != newOriginY
	if changed {
		*pCursorY = newCursorY
		*pOriginY = newOriginY
	}

	return changed
}

// setCursorAndOrigin sets the cursor to a specific index and adjusts the origin for list views.
func (s *AppState) setCursorAndOrigin(viewName string, newCursorY int, viewHeight int) bool {
	s.Lock()
	defer s.Unlock()

	var currentList []FileInfo
	var pOriginY *int
	var pCursorY *int

	isHidden := s.showHidden
	switch viewName {
	case viewFolders:
		if isHidden {
			currentList = s.hiddenDirs
			pOriginY = &s.hiddenFoldersOriginY
			pCursorY = &s.hiddenFoldersCursorY
		} else {
			currentList = s.visibleDirs
			pOriginY = &s.visibleFoldersOriginY
			pCursorY = &s.visibleFoldersCursorY
		}
	case viewFiles:
		if isHidden {
			currentList = s.hiddenFiles
			pOriginY = &s.hiddenFilesOriginY
			pCursorY = &s.hiddenFilesCursorY
		} else {
			currentList = s.visibleFiles
			pOriginY = &s.visibleFilesOriginY
			pCursorY = &s.visibleFilesCursorY
		}
	default:
		return false
	}

	listLen := len(currentList)
	if listLen <= 0 {
		changed := *pOriginY != 0 || *pCursorY != 0
		*pOriginY = 0
		*pCursorY = 0
		return changed
	}

	oldOriginY := *pOriginY
	oldCursorY := *pCursorY

	// 1. Clamp new cursor position
	if newCursorY < 0 {
		newCursorY = 0
	}
	if newCursorY >= listLen {
		newCursorY = listLen - 1
	}

	// 2. Calculate new origin
	newOriginY := *pOriginY
	if newCursorY < newOriginY || newCursorY >= newOriginY+viewHeight {
		// Cursor is outside the current view, center it if possible
		newOriginY = newCursorY - viewHeight/2
	}

	// 3. Validate and clamp origin
	maxOriginY := listLen - viewHeight
	if maxOriginY < 0 {
		maxOriginY = 0
	}
	if newOriginY > maxOriginY {
		newOriginY = maxOriginY
	}
	if newOriginY < 0 {
		newOriginY = 0
	}

	// 4. Update state if changed
	changed := oldCursorY != newCursorY || oldOriginY != newOriginY
	if changed {
		*pCursorY = newCursorY
		*pOriginY = newOriginY
	}

	return changed
}

// --- Action Menu State Management ---

func (s *AppState) OpenActionMenu(item FileInfo, options []ActionMenuItem, currentFocusView string) {
	s.Lock()
	defer s.Unlock()
	s.isActionMenuVisible = true
	s.actionMenuItemTarget = item
	s.actionMenuOptions = options
	s.actionMenuSelectedIdx = 0 // Start at the first option
	s.previousFocusView = currentFocusView
	s.lastMessage = "" // Clear any previous message
}

func (s *AppState) CloseActionMenu() {
	s.Lock()
	defer s.Unlock()
	s.isActionMenuVisible = false
	s.actionMenuItemTarget = FileInfo{} // Clear target
	s.actionMenuOptions = nil           // Clear options
	s.actionMenuSelectedIdx = -1
	// previousFocusView remains until next menu open
}

func (s *AppState) NavigateActionMenu(delta int) {
	s.Lock()
	defer s.Unlock()
	if !s.isActionMenuVisible || len(s.actionMenuOptions) == 0 {
		return
	}
	s.actionMenuSelectedIdx += delta
	if s.actionMenuSelectedIdx < 0 {
		s.actionMenuSelectedIdx = len(s.actionMenuOptions) - 1 // Wrap around top
	}
	if s.actionMenuSelectedIdx >= len(s.actionMenuOptions) {
		s.actionMenuSelectedIdx = 0 // Wrap around bottom
	}
}

// --- File Content View State Management ---

// SetFileContentView prepares the state for showing the file content.
func (s *AppState) SetFileContentView(filename, content, prevFocus string) {
	s.Lock()
	defer s.Unlock()
	s.isFileContentViewVisible = true
	s.fileContentViewFileName = filename
	s.fileContentViewContent = content
	// Calculate total lines (handle potential trailing newline)
	s.fileContentViewTotalLines = strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") && len(content) > 0 {
		s.fileContentViewTotalLines++
	} else if len(content) == 0 {
		s.fileContentViewTotalLines = 1 // Treat empty file as 1 line for display
	}

	s.fileContentViewOriginY = 0 // Reset scroll to top
	s.fileContentViewPrevFocus = prevFocus
}

// CloseFileContentView resets the state to hide the file content view.
func (s *AppState) CloseFileContentView() {
	s.Lock()
	defer s.Unlock()
	s.isFileContentViewVisible = false
	s.fileContentViewFileName = ""
	s.fileContentViewContent = ""
	s.fileContentViewTotalLines = 0
	s.fileContentViewOriginY = 0
	// s.fileContentViewPrevFocus remains for layout to use
}

// ScrollFileContentView updates the origin (scroll position) of the file content view.
func (s *AppState) ScrollFileContentView(delta int, viewHeight int) {
	s.Lock()
	defer s.Unlock()
	if !s.isFileContentViewVisible {
		return
	}

	newOriginY := s.fileContentViewOriginY + delta

	// Calculate max possible origin
	// Max origin is total lines - view height, but must be >= 0
	maxOriginY := s.fileContentViewTotalLines - viewHeight
	if maxOriginY < 0 {
		maxOriginY = 0
	}

	// Clamp new origin
	if newOriginY < 0 {
		newOriginY = 0
	}
	if newOriginY > maxOriginY {
		newOriginY = maxOriginY
	}

	s.fileContentViewOriginY = newOriginY
}

// --- Help View State Management ---

func (s *AppState) SetHelpVisible(visible bool) {
	s.Lock()
	defer s.Unlock()
	s.helpVisible = visible
}

// --- Confirm Delete State Management ---

func (s *AppState) SetConfirmDeleteVisible(visible bool) {
	s.Lock()
	defer s.Unlock()
	s.confirmDeleteVisible = visible
}

func (s *AppState) SetItemToDelete(item *FileInfo) {
	s.Lock()
	defer s.Unlock()
	s.itemToDelete = item
}

// --- Setters for UI state ---

func (s *AppState) SetVisibleFoldersOriginY(y int) {
	s.Lock()
	defer s.Unlock()
	s.visibleFoldersOriginY = y
}

func (s *AppState) SetVisibleFilesOriginY(y int) {
	s.Lock()
	defer s.Unlock()
	s.visibleFilesOriginY = y
}

func (s *AppState) SetHiddenFoldersOriginY(y int) {
	s.Lock()
	defer s.Unlock()
	s.hiddenFoldersOriginY = y
}

func (s *AppState) SetHiddenFilesOriginY(y int) {
	s.Lock()
	defer s.Unlock()
	s.hiddenFilesOriginY = y
}
