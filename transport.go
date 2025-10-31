package jsonrpcipc

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Listen and Dial are implemented in platform-specific files:
// - transport_windows.go (for Windows)
// - transport_unix.go (for Unix/Linux/Mac)

// normalizeWindowsPipePath adds the \\.\pipe\ prefix if not already present.
func normalizeWindowsPipePath(path string) string {
	// If it's already a pipe path, return as-is
	if strings.HasPrefix(path, `\\.\pipe\`) || strings.HasPrefix(path, `\\?\pipe\`) {
		return path
	}

	// Otherwise, add the pipe prefix
	return fmt.Sprintf(`\\.\pipe\%s`, path)
}

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

// CleanupSocket removes the Unix socket file.
// This should be called when the server shuts down to clean up resources.
// On Windows (named pipes), this is a no-op as pipes are automatically cleaned up.
func CleanupSocket(socketPath string) error {
	if runtime.GOOS == "windows" {
		// Named pipes are automatically cleaned up by Windows
		return nil
	}
	return removeSocketFile(socketPath)
}

// GetSocketPath returns the actual socket path that will be used by Listen/Dial.
// This is useful for logging or displaying the socket path to users.
//
// Example:
//
//	path := GetSocketPath("myapp")
//	// On Windows: returns "\\.\pipe\myapp"
//	// On Unix: returns "myapp"
func GetSocketPath(socketPath string) string {
	if runtime.GOOS == "windows" {
		return normalizeWindowsPipePath(socketPath)
	}
	return socketPath
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsUnix returns true if running on a Unix-like system (Linux, Mac, BSD, etc.).
func IsUnix() bool {
	return !IsWindows()
}
