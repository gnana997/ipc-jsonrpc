package jsonrpcipc

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"

	"github.com/Microsoft/go-winio"
)

// Listen creates a cross-platform IPC listener.
//
// On Unix/Linux/Mac, it creates a Unix domain socket.
// On Windows, it creates a Named Pipe.
//
// Socket path handling:
//   - Unix/Linux/Mac: Path is used as-is (e.g., "/tmp/myapp.sock")
//   - Windows: If path doesn't start with "\\.\pipe\" or "\\?\pipe\",
//     it's automatically prefixed with "\\.\pipe\" (e.g., "myapp" becomes "\\.\pipe\myapp")
//
// The function automatically handles cleanup:
//   - On Unix systems, it removes existing socket files before creating a new listener
//
// Example usage:
//
//	listener, err := Listen("/tmp/myapp.sock")  // Unix
//	listener, err := Listen("myapp")             // Windows (becomes \\.\pipe\myapp)
func Listen(socketPath string) (net.Listener, error) {
	if runtime.GOOS == "windows" {
		// Windows: Use winio for named pipes
		addr := normalizeWindowsPipePath(socketPath)
		listener, err := winio.ListenPipe(addr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create named pipe listener on %s: %w", addr, err)
		}
		return listener, nil
	}

	// Unix/Linux/Mac: Use Unix domain sockets
	// Remove existing socket file if it exists
	if err := removeSocketFile(socketPath); err != nil {
		return nil, fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create listener on %s: %w", socketPath, err)
	}

	return listener, nil
}

// Dial creates a cross-platform IPC client connection.
//
// This function uses the same path normalization as Listen().
//
// Example usage:
//
//	conn, err := Dial("/tmp/myapp.sock")  // Unix
//	conn, err := Dial("myapp")             // Windows (becomes \\.\pipe\myapp)
func Dial(socketPath string) (net.Conn, error) {
	if runtime.GOOS == "windows" {
		// Windows: Use winio for named pipes
		addr := normalizeWindowsPipePath(socketPath)
		conn, err := winio.DialPipe(addr, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to named pipe %s: %w", addr, err)
		}
		return conn, nil
	}

	// Unix/Linux/Mac: Use Unix domain sockets
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", socketPath, err)
	}

	return conn, nil
}

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
