// ---- File: core.go ----
package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jroimartin/gocui"
)

// --- Icon Mapping (Requires Nerd Fonts) ---
var iconMap = map[string]string{
	// Folders
	"dir":          "", // Default folder icon
	"node_modules": "", // npm icon
	".git":         "", // Git icon

	// Files by extension
	".go":   "",
	".py":   "",
	".js":   "",
	".ts":   "", // Added TypeScript
	".jsx":  "", // Added JSX
	".tsx":  "", // Added TSX
	".json": "",
	".html": "",
	".css":  "",
	".scss": "", // Added SCSS
	".md":   "",
	".sh":   "",
	".bash": "",
	".zsh":  "",
	".yml":  "",
	".yaml": "",
	".toml": "", // Added TOML
	".zip":  "",
	".tar":  "",
	".gz":   "",
	".bz2":  "",
	".xz":   "",
	".rar":  "", // Added RAR
	".7z":   "", // Added 7z
	".log":  "",
	".env":  "", // Env files similar to config
	".sql":  "", // Added SQL
	".pdf":  "", // Added PDF
	".png":  "", // Added Image icons
	".jpg":  "",
	".jpeg": "",
	".gif":  "",
	".svg":  "", // Added SVG

	// Files by name
	"makefile":   "",
	"dockerfile": "",
	"license":    "",
	"readme.md":  "", // Prioritize Readme icon
}

var defaultFileIcon = "" // Default file icon
var defaultDirIcon = ""  // Default directory icon

func getIcon(name string, isDir bool) string {
	lowerName := strings.ToLower(name)

	// Check full name first (e.g., "README.md")
	if icon, ok := iconMap[lowerName]; ok {
		return icon
	}

	if isDir {
		if icon, ok := iconMap[lowerName]; ok { // Check dir name (e.g., ".git")
			return icon
		}
		if strings.HasPrefix(name, ".") { // Check hidden dir name without dot? Less common.
			if icon, ok := iconMap[name]; ok {
				return icon
			}
		}
		// Check if name is in the map specifically for dirs (if we add any like "vendor": "...")
		// if icon, ok := iconMap["dir:"+lowerName]; ok { return icon }
		return defaultDirIcon // Default dir icon
	}

	// Check extension for files
	ext := strings.ToLower(filepath.Ext(name))
	if ext != "" {
		if icon, ok := iconMap[ext]; ok {
			return icon
		}
	}

	// Fallback to default file icon
	return defaultFileIcon
}

// loadDirectoryContents reads CWD, filters, sorts, and updates appState.
func loadDirectoryContents(state *AppState) error {
	state.ClearMessage() // Clear any previous messages on reload
	cwd := state.Cwd()

	visibleFiles := []FileInfo{}
	visibleDirs := []FileInfo{}
	hiddenFiles := []FileInfo{}
	hiddenDirs := []FileInfo{}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		state.SetMessage(fmt.Sprintf("Error reading dir: %s", trimError(err)))
		// Return nil to allow UI to update with the error message, but don't stop the app
		return nil // Was: fmt.Errorf("reading directory %s: %w", cwd, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		// Simple check for hidden (can be platform specific)
		isHidden := strings.HasPrefix(name, ".") && name != "." && name != ".."

		// Use os.Stat to check if it's a directory or symlink etc.
		// Using Lstat to not follow symlinks for the IsDir check itself.
		info, err := os.Lstat(filepath.Join(cwd, name))
		if err != nil {
			log.Printf("Warning: Could not stat entry %s: %v", name, err)
			// Skip this entry or mark it as an error? Let's skip for now.
			continue
		}

		isDir := info.IsDir()
		// TODO: Handle Symlinks visually? (Maybe add a different icon or suffix)
		// isSymlink := info.Mode()&os.ModeSymlink != 0

		fullPath := filepath.Join(cwd, name) // Needed for actions

		fi := FileInfo{
			Name:  name,
			Path:  fullPath,
			IsDir: isDir,
			Icon:  getIcon(name, isDir), // Pass isDir here
			// Size is populated by calculateStats for largestFile
		}

		if isHidden {
			if isDir {
				hiddenDirs = append(hiddenDirs, fi)
			} else {
				hiddenFiles = append(hiddenFiles, fi)
			}
		} else {
			if isDir {
				visibleDirs = append(visibleDirs, fi)
			} else {
				visibleFiles = append(visibleFiles, fi)
			}
		}
	}

	// Sort alphabetically (case-insensitive)
	sortFunc := func(a, b FileInfo) bool {
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	}
	sort.Slice(visibleDirs, func(i, j int) bool { return sortFunc(visibleDirs[i], visibleDirs[j]) })
	sort.Slice(visibleFiles, func(i, j int) bool { return sortFunc(visibleFiles[i], visibleFiles[j]) })
	sort.Slice(hiddenDirs, func(i, j int) bool { return sortFunc(hiddenDirs[i], hiddenDirs[j]) })
	sort.Slice(hiddenFiles, func(i, j int) bool { return sortFunc(hiddenFiles[i], hiddenFiles[j]) })

	// Update state using the method (this also resets cursors/origins)
	state.SetDirectoryContents(visibleDirs, visibleFiles, hiddenDirs, hiddenFiles)

	return nil
}

// calculateStats runs in a goroutine to get size, largest file, and git status.
func calculateStats(g *gocui.Gui, state *AppState) {
	state.SetStatsLoading() // Mark as loading

	// Trigger UI update immediately to show "Calculating..."
	g.Update(func(gui *gocui.Gui) error { return nil })

	cwd := state.Cwd()

	var totalSize int64 = 0                       // Start at 0, handle errors explicitly
	var largestFile FileInfo = FileInfo{Size: -1} // Size -1 indicates none found yet
	var gitStatus string
	var firstWalkErr error // Store the first significant error encountered

	// Use WalkDir for potentially better performance and error handling per entry
	err := filepath.WalkDir(cwd, func(path string, d fs.DirEntry, walkError error) error {
		// --- Handle Walk Errors ---
		if walkError != nil {
			// Log the error but try to continue if possible
			log.Printf("Warning: Walk error accessing %s: %v", path, walkError)
			if firstWalkErr == nil { // Store the first error
				// Try to get a more user-friendly name if possible
				entryName := "entry"
				if d != nil {
					entryName = d.Name()
				}
				firstWalkErr = fmt.Errorf("accessing %s: %w", entryName, walkError)
			}
			// If it's an error on a directory, skip its contents
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			// Otherwise, skip just this file/entry
			return nil // Returning nil allows WalkDir to continue
		}

		// Skip the root directory itself for size calculation
		if path == cwd {
			return nil
		}

		// --- Process Entry ---
		if !d.IsDir() {
			info, infoErr := d.Info()
			if infoErr != nil {
				log.Printf("Warning: Could not get info for %s: %v", path, infoErr)
				if firstWalkErr == nil {
					firstWalkErr = fmt.Errorf("info for %s: %w", d.Name(), infoErr)
				}
				return nil // Skip this entry
			}
			fileSize := info.Size()
			totalSize += fileSize

			// Update largest file found so far
			if fileSize > largestFile.Size {
				largestFile = FileInfo{
					Name:  d.Name(),
					Path:  path, // Store full path for potential actions later if needed
					IsDir: false,
					Size:  fileSize,
					Icon:  getIcon(d.Name(), false), // Get icon for the largest file
				}
			}
		}
		return nil // Continue walking
	})

	// Handle error returned directly by WalkDir (e.g., initial access error)
	if err != nil && firstWalkErr == nil {
		firstWalkErr = fmt.Errorf("walking %s: %w", filepath.Base(cwd), err)
	}

	// --- Update State Based on Walk Results ---
	finalTotalSize := totalSize
	finalLargestFile := largestFile
	if firstWalkErr != nil {
		log.Printf("Warning: Stats calculation encountered errors: %v", firstWalkErr)
		finalTotalSize = -2         // Indicate error state for size
		if largestFile.Size == -1 { // If no file was ever successfully processed
			finalLargestFile = FileInfo{Name: "Error during scan", Size: -2}
		} else {
			// Keep the largest file found, but maybe indicate the total size is partial?
			// For now, just marking total size as error is sufficient.
			// No name change needed here, error is indicated by totalSize = -2
		}

	} else if largestFile.Size == -1 { // Walk completed without error, but no files found
		finalLargestFile = FileInfo{} // Represents "no files" correctly
	}

	// 2. Check Git Status (runs regardless of walk errors)
	// Use IsGitRepo and GetGitBranch functions for clarity
	isRepo, repoCheckErr := IsGitRepo(cwd)
	if repoCheckErr != nil {
		log.Printf("Warning: Git check failed for %s: %v", cwd, repoCheckErr)
		gitStatus = "Status Unknown (Error)" // More specific error
	} else if !isRepo {
		gitStatus = "Inactive"
	} else {
		branchName, branchErr := GetGitBranch(cwd)
		if branchErr != nil {
			log.Printf("Warning: Could not get git branch for %s: %v", cwd, branchErr)
			gitStatus = "Active: (Branch Error)" // Specific error for branch issue
		} else if branchName == "" {
			// This might happen in detached HEAD state
			gitStatus = "Active: (Detached HEAD?)"
		} else {
			gitStatus = fmt.Sprintf("Active: (%s)", branchName)
		}
		// Optional: Check for modifications (adds overhead)
		// modified, modCheckErr := HasGitModifications(cwd)
		// if modCheckErr == nil && modified {
		// 	gitStatus += " *" // Add indicator if modified
		// }
	}

	// --- Update state safely ---
	state.SetStatsResults(finalTotalSize, finalLargestFile, gitStatus, firstWalkErr)

	// Trigger UI update from the goroutine
	g.Update(func(gui *gocui.Gui) error {
		return nil // Layout manager will call update*View functions
	})
}

// --- Git Helper Functions ---

// IsGitRepo checks if a directory is part of a git repository.
func IsGitRepo(dir string) (bool, error) {
	// `git rev-parse --is-inside-work-tree` is reliable
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	output, err := cmd.Output()
	if err != nil {
		// This often means 'git' command not found or it's not a repo.
		// Check if it's the specific "not a git repository" error.
		if exitErr, ok := err.(*exec.ExitError); ok {
			// stderr output often contains "fatal: not a git repository"
			if strings.Contains(string(exitErr.Stderr), "not a git repository") {
				return false, nil // Not an error, just not a repo
			}
		}
		// Otherwise, it's a different error (e.g., git not installed)
		return false, fmt.Errorf("git check failed: %w", err)
	}
	return strings.TrimSpace(string(output)) == "true", nil
}

// GetGitBranch returns the current branch name.
func GetGitBranch(dir string) (string, error) {
	// Use `git branch --show-current` as it's simpler
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		// Check if it's detached HEAD state (often returns exit code 1, but no output on stdout)
		// If it's an ExitError and output is empty, likely detached HEAD. We don't need exitErr itself.
		if _, ok := err.(*exec.ExitError); ok && len(output) == 0 {
			// Could try `git rev-parse --short HEAD` to get commit hash as indicator
			// For now, just return empty string indicating unknown/detached branch
			return "", nil
		}
		return "", fmt.Errorf("git branch check failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// HasGitModifications checks for uncommitted changes or untracked files.
func HasGitModifications(dir string) (bool, error) {
	// `git status --porcelain` is fast and output is empty if clean
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status check failed: %w", err)
	}
	return len(output) > 0, nil
}
