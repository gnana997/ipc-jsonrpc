//go:build windows

package jsonrpcipc

import (
	"fmt"
	"net"

	"github.com/Microsoft/go-winio"
)

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
