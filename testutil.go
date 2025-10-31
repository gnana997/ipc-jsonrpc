package jsonrpcipc

import (
	"io"
	"net"
	"sync"
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
