// ---- File: utils.go ----

package main

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard" // Import clipboard library
)

// formatSize converts bytes to a human-readable string (KB, MB, GB).
func formatSize(sizeBytes int64) string {
	const (
		_          = iota // ignore first value by assigning to blank identifier
		KB float64 = 1 << (10 * iota)
		MB
		GB
		TB // Added Terabyte
	)

	switch {
	case sizeBytes == -1: // Initial calculating state
		return "Calculating..."
	case sizeBytes == -2: // Error state
		return "Error"
	case sizeBytes < 0: // Other negative shouldn't happen, but fallback
		return "Invalid Size"
	case sizeBytes == 0:
		// Show 0 B only if it's a file, maybe implicit for folders?
		// For now, let's be explicit.
		return "0 B"
	}

	size := float64(sizeBytes)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TiB", size/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GiB", size/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MiB", size/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KiB", size/KB)
	default:
		return fmt.Sprintf("%d B", sizeBytes)
	}
}

// trimError provides a shorter version of an error message.
func trimError(err error) string {
	if err == nil {
		return ""
	}
	errMsg := err.Error()

	// Try to remove path prefixes or common verbose parts
	// This version prioritizes the *last* part after ": "
	lastColon := strings.LastIndex(errMsg, ": ")
	if lastColon != -1 && lastColon < len(errMsg)-2 {
		errMsg = errMsg[lastColon+2:]
	}

	// Example: Remove specific common prefixes/suffixes (less needed if we take last part)
	// errMsg = strings.TrimPrefix(errMsg, "stat ")
	// errMsg = strings.TrimPrefix(errMsg, "open ")
	// errMsg = strings.TrimPrefix(errMsg, "read ")
	// errMsg = strings.TrimSuffix(errMsg, "no such file or directory")
	// errMsg = strings.TrimSuffix(errMsg, "permission denied")
	// errMsg = strings.TrimSuffix(errMsg, "is a directory")

	// Limit overall length
	maxLen := 60 // Adjusted max length for potentially wider message bar
	if len(errMsg) > maxLen {
		errMsg = errMsg[:maxLen-3] + "..."
	}

	return errMsg
}

// copyToClipboard writes the given text to the system clipboard.
func copyToClipboard(text string) error {
	err := clipboard.WriteAll(text)
	if err != nil {
		// Log the detailed error, but return a simpler one potentially
		// log.Printf("Clipboard write error: %v", err)
		return fmt.Errorf("clipboard unavailable") // Or return original error
	}
	return nil
}
