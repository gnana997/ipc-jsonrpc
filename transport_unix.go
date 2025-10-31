//go:build !windows

package jsonrpcipc

import (
	"fmt"
	"net"
	"path/filepath"
	"strings"
)

// normalizeSocketPath converts simple socket names to full Unix socket paths
// Examples:
//   - "myapp" -> "/tmp/myapp.sock"
//   - "/tmp/myapp.sock" -> "/tmp/myapp.sock" (unchanged)
//   - "./myapp.sock" -> "./myapp.sock" (unchanged)
func normalizeSocketPath(socketPath string) string {
	// If it's already an absolute or relative path with directory separator, keep it
	if strings.Contains(socketPath, "/") {
		return socketPath
	}

	// If it already has .sock extension, just prepend /tmp/
	if strings.HasSuffix(socketPath, ".sock") {
		return filepath.Join("/tmp", socketPath)
	}

	// Simple name - convert to /tmp/{name}.sock
	return filepath.Join("/tmp", socketPath+".sock")
}

// Listen creates a Unix domain socket listener
func Listen(socketPath string) (net.Listener, error) {
	// Normalize socket path for Unix systems
	socketPath = normalizeSocketPath(socketPath)

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

// Dial creates a Unix domain socket client connection
func Dial(socketPath string) (net.Conn, error) {
	// Normalize socket path for Unix systems
	socketPath = normalizeSocketPath(socketPath)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", socketPath, err)
	}
	return conn, nil
}

// CleanupSocket removes the Unix socket file.
// This should be called when the server shuts down to clean up resources.
func CleanupSocket(socketPath string) error {
	return removeSocketFile(normalizeSocketPath(socketPath))
}

// GetSocketPath returns the actual socket path that will be used by Listen/Dial.
// This is useful for logging or displaying the socket path to users.
func GetSocketPath(socketPath string) string {
	return normalizeSocketPath(socketPath)
}
