//go:build windows

package jsonrpcipc

import (
	"fmt"
	"net"
	"strings"

	"github.com/Microsoft/go-winio"
)

// normalizeWindowsPipePath adds the \\.\pipe\ prefix if not already present.
func normalizeWindowsPipePath(path string) string {
	// If it's already a pipe path, return as-is
	if strings.HasPrefix(path, `\\.\pipe\`) || strings.HasPrefix(path, `\\?\pipe\`) {
		return path
	}

	// Otherwise, add the pipe prefix
	return fmt.Sprintf(`\\.\pipe\%s`, path)
}

// Listen creates a Windows Named Pipe listener
func Listen(socketPath string) (net.Listener, error) {
	addr := normalizeWindowsPipePath(socketPath)
	listener, err := winio.ListenPipe(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create named pipe listener on %s: %w", addr, err)
	}
	return listener, nil
}

// Dial creates a Windows Named Pipe client connection
func Dial(socketPath string) (net.Conn, error) {
	addr := normalizeWindowsPipePath(socketPath)
	conn, err := winio.DialPipe(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to named pipe %s: %w", addr, err)
	}
	return conn, nil
}

// CleanupSocket is a no-op on Windows as named pipes are automatically cleaned up.
func CleanupSocket(socketPath string) error {
	// Named pipes are automatically cleaned up by Windows
	return nil
}

// GetSocketPath returns the actual socket path that will be used by Listen/Dial.
// This is useful for logging or displaying the socket path to users.
func GetSocketPath(socketPath string) string {
	return normalizeWindowsPipePath(socketPath)
}
