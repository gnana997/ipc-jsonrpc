package jsonrpcipc

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestIsWindows(t *testing.T) {
	result := IsWindows()
	expected := runtime.GOOS == "windows"

	if result != expected {
		t.Errorf("IsWindows() = %v, want %v (runtime.GOOS = %s)", result, expected, runtime.GOOS)
	}
}

func TestIsUnix(t *testing.T) {
	result := IsUnix()
	expected := runtime.GOOS != "windows"

	if result != expected {
		t.Errorf("IsUnix() = %v, want %v (runtime.GOOS = %s)", result, expected, runtime.GOOS)
	}
}

func TestIsWindows_IsUnix_Opposite(t *testing.T) {
	if IsWindows() == IsUnix() {
		t.Error("IsWindows() and IsUnix() should return opposite values")
	}
}

func TestGetSocketPath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantWindows  string
		wantUnix     string
	}{
		{
			name:        "simple name",
			path:        "myapp",
			wantWindows: `\\.\pipe\myapp`,
			wantUnix:    "/tmp/myapp.sock",
		},
		{
			name:        "path with sock extension",
			path:        "myapp.sock",
			wantWindows: `\\.\pipe\myapp.sock`,
			wantUnix:    "/tmp/myapp.sock",
		},
		{
			name:        "absolute unix path",
			path:        "/tmp/myapp.sock",
			wantWindows: "/tmp/myapp.sock", // Windows treats this as-is (absolute path)
			wantUnix:    "/tmp/myapp.sock",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetSocketPath(tt.path)

			if runtime.GOOS == "windows" {
				if result != tt.wantWindows {
					t.Errorf("GetSocketPath(%q) on Windows = %q, want %q", tt.path, result, tt.wantWindows)
				}
			} else {
				if result != tt.wantUnix {
					t.Errorf("GetSocketPath(%q) on Unix = %q, want %q", tt.path, result, tt.wantUnix)
				}
			}
		})
	}
}

func TestRemoveSocketFile_NonExistent(t *testing.T) {
	// Test removing a file that doesn't exist
	nonExistent := filepath.Join(os.TempDir(), "non-existent-socket-file-12345.sock")

	err := removeSocketFile(nonExistent)
	if err != nil {
		t.Errorf("removeSocketFile(%q) error = %v, want nil for non-existent file", nonExistent, err)
	}
}

func TestRemoveSocketFile_Existing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix socket file test on Windows")
	}

	// Create a temporary socket file
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	// Create the file
	file, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Verify file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Fatal("Test file was not created")
	}

	// Remove the file
	err = removeSocketFile(socketPath)
	if err != nil {
		t.Errorf("removeSocketFile() error = %v, want nil", err)
	}

	// Verify file was removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("File was not removed")
	}
}

func TestCleanupSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		// On Windows, CleanupSocket should be a no-op
		err := CleanupSocket("anypath")
		if err != nil {
			t.Errorf("CleanupSocket() on Windows error = %v, want nil", err)
		}
		return
	}

	// On Unix, test actual cleanup
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "cleanup-test.sock")

	// Create a file to cleanup
	file, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	file.Close()

	// Cleanup the socket
	err = CleanupSocket(socketPath)
	if err != nil {
		t.Errorf("CleanupSocket() error = %v, want nil", err)
	}

	// Verify file was removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file was not cleaned up")
	}
}

func TestListen_Dial_Integration(t *testing.T) {
	var socketPath string

	if runtime.GOOS == "windows" {
		// Windows: Use a unique pipe name
		socketPath = "test-pipe-" + time.Now().Format("20060102150405")
	} else {
		// Unix: Use a temporary directory
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "test.sock")
	}

	// Start listener
	listener, err := Listen(socketPath)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()
	defer CleanupSocket(socketPath)

	// Accept connections in goroutine
	acceptDone := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err
			return
		}
		acceptDone <- conn
	}()

	// Dial the listener
	clientConn, err := Dial(socketPath)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer clientConn.Close()

	// Get server connection
	var serverConn net.Conn
	select {
	case serverConn = <-acceptDone:
	case err := <-acceptErr:
		t.Fatalf("Accept() error = %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Accept() timeout")
	}
	defer serverConn.Close()

	// Test bidirectional communication
	testMessage := []byte("Hello, IPC!")

	// Client writes, server reads
	go func() {
		_, err := clientConn.Write(testMessage)
		if err != nil {
			t.Errorf("Client write error: %v", err)
		}
	}()

	buf := make([]byte, len(testMessage))
	n, err := io.ReadFull(serverConn, buf)
	if err != nil {
		t.Fatalf("Server read error: %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("Server read %d bytes, want %d", n, len(testMessage))
	}
	if string(buf) != string(testMessage) {
		t.Errorf("Server received %q, want %q", string(buf), string(testMessage))
	}

	// Server writes, client reads
	response := []byte("Hello back!")
	go func() {
		_, err := serverConn.Write(response)
		if err != nil {
			t.Errorf("Server write error: %v", err)
		}
	}()

	buf2 := make([]byte, len(response))
	_, err = io.ReadFull(clientConn, buf2)
	if err != nil {
		t.Fatalf("Client read error: %v", err)
	}
	if string(buf2) != string(response) {
		t.Errorf("Client received %q, want %q", string(buf2), string(response))
	}
}

func TestListen_Dial_MultipleConnections(t *testing.T) {
	var socketPath string

	if runtime.GOOS == "windows" {
		socketPath = "test-multi-pipe-" + time.Now().Format("20060102150405")
	} else {
		tmpDir := t.TempDir()
		socketPath = filepath.Join(tmpDir, "multi-test.sock")
	}

	listener, err := Listen(socketPath)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()
	defer CleanupSocket(socketPath)

	const numConns = 3

	// Accept connections in goroutine
	serverConns := make(chan net.Conn, numConns)
	go func() {
		for i := 0; i < numConns; i++ {
			conn, err := listener.Accept()
			if err != nil {
				t.Errorf("Accept() error: %v", err)
				return
			}
			serverConns <- conn
		}
	}()

	// Create multiple client connections
	var clientConns []net.Conn
	for i := 0; i < numConns; i++ {
		conn, err := Dial(socketPath)
		if err != nil {
			t.Fatalf("Dial() %d error = %v", i, err)
		}
		clientConns = append(clientConns, conn)
		defer clientConns[i].Close()
	}

	// Verify all connections are accepted
	timeout := time.After(2 * time.Second)
	for i := 0; i < numConns; i++ {
		select {
		case conn := <-serverConns:
			conn.Close()
		case <-timeout:
			t.Fatalf("Timeout waiting for connection %d", i)
		}
	}
}

func TestDial_NoListener(t *testing.T) {
	var socketPath string

	if runtime.GOOS == "windows" {
		socketPath = "nonexistent-pipe-12345"
	} else {
		socketPath = "/tmp/nonexistent-socket-12345.sock"
	}

	// Try to dial without a listener
	conn, err := Dial(socketPath)
	if err == nil {
		conn.Close()
		t.Error("Dial() to non-existent socket succeeded, want error")
	}
}

func TestListen_CloseAndCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Socket cleanup test not applicable on Windows (named pipes auto-cleanup)")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "cleanup.sock")

	// Create and close listener
	listener, err := Listen(socketPath)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	// Verify socket file exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Error("Socket file was not created")
	}

	listener.Close()

	// Cleanup
	err = CleanupSocket(socketPath)
	if err != nil {
		t.Errorf("CleanupSocket() error = %v", err)
	}

	// Verify socket file is removed
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		t.Error("Socket file was not cleaned up")
	}
}

func TestListen_RemoveStaleSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Stale socket test not applicable on Windows")
	}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "stale.sock")

	// Create a stale socket file
	file, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("Failed to create stale socket: %v", err)
	}
	file.Close()

	// Listen should remove the stale socket and create a new listener
	listener, err := Listen(socketPath)
	if err != nil {
		t.Fatalf("Listen() with stale socket error = %v", err)
	}
	defer listener.Close()
	defer CleanupSocket(socketPath)

	// Verify we can dial the new listener
	conn, err := Dial(socketPath)
	if err != nil {
		t.Errorf("Dial() after removing stale socket error = %v", err)
	} else {
		conn.Close()
	}
}
