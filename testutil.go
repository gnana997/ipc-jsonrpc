package jsonrpcipc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// mockConn implements net.Conn for testing without real sockets
type mockConn struct {
	reader     *io.PipeReader
	writer     *io.PipeWriter
	peerReader *io.PipeReader
	peerWriter *io.PipeWriter
	closeOnce  sync.Once
	closed     bool
	localAddr  mockAddr
	remoteAddr mockAddr
}

type mockAddr struct {
	network string
	address string
}

func (a mockAddr) Network() string { return a.network }
func (a mockAddr) String() string  { return a.address }

// newMockConnPair creates a pair of connected mock connections for testing
func newMockConnPair() (*mockConn, *mockConn) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()

	conn1 := &mockConn{
		reader:     r1,
		writer:     w2,
		peerReader: r2,
		peerWriter: w1,
		localAddr:  mockAddr{"mock", "client"},
		remoteAddr: mockAddr{"mock", "server"},
	}

	conn2 := &mockConn{
		reader:     r2,
		writer:     w1,
		peerReader: r1,
		peerWriter: w2,
		localAddr:  mockAddr{"mock", "server"},
		remoteAddr: mockAddr{"mock", "client"},
	}

	return conn1, conn2
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.reader.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writer.Write(b)
}

func (m *mockConn) Close() error {
	m.closeOnce.Do(func() {
		m.closed = true
		m.reader.Close()
		m.writer.Close()
		if m.peerWriter != nil {
			m.peerWriter.Close()
		}
		if m.peerReader != nil {
			m.peerReader.Close()
		}
	})
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return m.localAddr }
func (m *mockConn) RemoteAddr() net.Addr               { return m.remoteAddr }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// mockHandler is a test handler that records calls
type mockHandler struct {
	calls  []mockHandlerCall
	mu     sync.Mutex
	result interface{}
	err    error
}

type mockHandlerCall struct {
	ctx    context.Context
	params json.RawMessage
}

func newMockHandler(result interface{}, err error) *mockHandler {
	return &mockHandler{
		calls:  []mockHandlerCall{},
		result: result,
		err:    err,
	}
}

func (h *mockHandler) Handle(ctx context.Context, params json.RawMessage) (interface{}, error) {
	h.mu.Lock()
	h.calls = append(h.calls, mockHandlerCall{ctx, params})
	h.mu.Unlock()
	return h.result, h.err
}

func (h *mockHandler) CallCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.calls)
}

func (h *mockHandler) LastCall() *mockHandlerCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.calls) == 0 {
		return nil
	}
	return &h.calls[len(h.calls)-1]
}

func (h *mockHandler) Calls() []mockHandlerCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	result := make([]mockHandlerCall, len(h.calls))
	copy(result, h.calls)
	return result
}

// testLogger is a logger that captures log messages for testing
type testLogger struct {
	logs []logEntry
	mu   sync.Mutex
}

type logEntry struct {
	method   string
	duration time.Duration
	err      error
}

func newTestLogger() *testLogger {
	return &testLogger{logs: []logEntry{}}
}

func (l *testLogger) Log(method string, duration time.Duration, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, logEntry{method, duration, err})
}

func (l *testLogger) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.logs)
}

func (l *testLogger) Entries() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	result := make([]logEntry, len(l.logs))
	copy(result, l.logs)
	return result
}

// createTestSocketPath creates a unique socket path for testing
func createTestSocketPath(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	return tmpDir + "/test.sock"
}

// waitForServer waits for a server to start listening
func waitForServer(socketPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := Dial(socketPath)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("server did not start within timeout")
}

// sendRequest sends a JSON-RPC request to a connection
func sendRequest(t *testing.T, conn net.Conn, method string, params interface{}, id interface{}) {
	t.Helper()
	req := &Request{
		JSONRPC: "2.0",
		Method:  method,
		ID:      id,
	}
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		req.Params = data
	}

	codec := NewCodec(conn)
	if err := codec.WriteJSON(req); err != nil {
		t.Fatal(err)
	}
}

// readResponse reads a JSON-RPC response from a connection
func readResponse(t *testing.T, conn net.Conn) *Message {
	t.Helper()
	codec := NewCodec(conn)
	var msg Message
	if err := codec.ReadJSON(&msg); err != nil {
		t.Fatal(err)
	}
	return &msg
}

// readNotification reads a JSON-RPC notification from a connection
func readNotification(t *testing.T, conn net.Conn) *Notification {
	t.Helper()
	codec := NewCodec(conn)
	var notif Notification
	if err := codec.ReadJSON(&notif); err != nil {
		t.Fatal(err)
	}
	return &notif
}

// sendNotification sends a JSON-RPC notification to a connection
func sendNotification(t *testing.T, conn net.Conn, method string, params interface{}) {
	t.Helper()
	notif := &Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	codec := NewCodec(conn)
	if err := codec.WriteJSON(notif); err != nil {
		t.Fatal(err)
	}
}

// compareIDs compares two ID values, accounting for JSON number conversion
func compareIDs(a, b interface{}) bool {
	// Handle nil
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Handle string IDs
	aStr, aIsStr := a.(string)
	bStr, bIsStr := b.(string)
	if aIsStr && bIsStr {
		return aStr == bStr
	}

	// Handle numeric IDs (JSON unmarshals to float64)
	aNum, aIsNum := toFloat64(a)
	bNum, bIsNum := toFloat64(b)
	if aIsNum && bIsNum {
		return aNum == bNum
	}

	return false
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}
