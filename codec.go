package jsonrpcipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

// LineDelimitedCodec handles encoding and decoding of line-delimited JSON messages.
// Each JSON message is terminated with a newline character ('\n').
//
// This codec is compatible with the Node.js node-ipc-jsonrpc package
// which uses the same line-delimited JSON format.
//
// Thread-safety: This codec is safe for concurrent use. Reads and writes are
// protected by separate mutexes to allow concurrent read/write operations.
type LineDelimitedCodec struct {
	reader *bufio.Reader
	writer *bufio.Writer
	conn   io.ReadWriteCloser

	// Separate mutexes for reading and writing to allow concurrent operations
	readMu  sync.Mutex
	writeMu sync.Mutex
}

// NewCodec creates a new LineDelimitedCodec for the given connection.
//
// The codec uses buffered I/O for efficient reading and writing of line-delimited
// JSON messages.
func NewCodec(conn io.ReadWriteCloser) *LineDelimitedCodec {
	return &LineDelimitedCodec{
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		conn:   conn,
	}
}

// ReadMessage reads a single line-delimited JSON message from the connection.
//
// The message is read until a newline character ('\n') is encountered.
// Empty lines are skipped automatically.
//
// Returns:
//   - The raw JSON bytes (without the newline)
//   - An error if reading fails or if EOF is reached
//
// Thread-safety: This method is safe to call concurrently with WriteMessage.
func (c *LineDelimitedCodec) ReadMessage() ([]byte, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	for {
		// Read until newline
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				// If we have data before EOF, try to parse it
				if len(line) > 0 {
					return line, nil
				}
			}
			return nil, fmt.Errorf("read error: %w", err)
		}

		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		// Handle Windows line endings (\r\n)
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) == 0 {
			continue
		}

		return line, nil
	}
}

// WriteMessage writes a line-delimited JSON message to the connection.
//
// The message is written with a newline character ('\n') appended.
// The write is buffered and flushed immediately to ensure the message is sent.
//
// Parameters:
//   - data: The raw JSON bytes to write (newline will be added automatically)
//
// Thread-safety: This method is safe to call concurrently with ReadMessage.
func (c *LineDelimitedCodec) WriteMessage(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if _, err := c.writer.Write(data); err != nil {
		return fmt.Errorf("write error: %w", err)
	}

	if err := c.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write newline error: %w", err)
	}

	if err := c.writer.Flush(); err != nil {
		return fmt.Errorf("flush error: %w", err)
	}

	return nil
}

// ReadJSON reads and unmarshals a JSON-RPC message from the connection.
//
// This is a convenience method that combines ReadMessage and json.Unmarshal.
//
// Parameters:
//   - v: Pointer to the value to unmarshal into (typically *Message)
//
// Thread-safety: This method is safe to call concurrently with WriteJSON.
func (c *LineDelimitedCodec) ReadJSON(v interface{}) error {
	data, err := c.ReadMessage()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json unmarshal error: %w", err)
	}

	return nil
}

// WriteJSON marshals and writes a JSON-RPC message to the connection.
//
// This is a convenience method that combines json.Marshal and WriteMessage.
//
// Parameters:
//   - v: The value to marshal and write (typically a Response or Notification)
//
// Thread-safety: This method is safe to call concurrently with ReadJSON.
func (c *LineDelimitedCodec) WriteJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	return c.WriteMessage(data)
}

// Close closes the underlying connection.
func (c *LineDelimitedCodec) Close() error {
	return c.conn.Close()
}
