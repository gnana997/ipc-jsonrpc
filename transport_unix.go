//go:build !windows

package jsonrpcipc

import (
	"fmt"
	"net"
)

// Listen creates a Unix domain socket listener
func Listen(socketPath string) (net.Listener, error) {
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
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", socketPath, err)
	}
	return conn, nil
}
