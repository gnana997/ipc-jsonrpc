package jsonrpcipc

import (
	"fmt"
	"os"
	"runtime"
)

// Listen and Dial are implemented in platform-specific files:
// - transport_windows.go (for Windows)
// - transport_unix.go (for Unix/Linux/Mac)

// removeSocketFile deletes stale socket files before binding.
// Binding to an existing socket file will fail, so cleanup is required.
func removeSocketFile(path string) error {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, nothing to remove
			return nil
		}
		// Some other error occurred
		return fmt.Errorf("failed to stat socket file: %w", err)
	}

	// Remove the file
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove socket file: %w", err)
	}

	return nil
}

// CleanupSocket, GetSocketPath are implemented in platform-specific files

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsUnix returns true if running on a Unix-like system (Linux, Mac, BSD, etc.).
func IsUnix() bool {
	return !IsWindows()
}
