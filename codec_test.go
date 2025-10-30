package jsonrpcipc

import (
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestNewCodec(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)

	if codec == nil {
		t.Fatal("NewCodec returned nil")
	}

	if codec.reader == nil {
		t.Error("Codec reader is nil")
	}

	if codec.writer == nil {
		t.Error("Codec writer is nil")
	}

	if codec.conn != conn1 {
		t.Error("Codec conn does not match provided connection")
	}
}

func TestCodec_ReadMessage(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "single line",
			input: `{"jsonrpc":"2.0","method":"test","id":1}` + "\n",
			want:  `{"jsonrpc":"2.0","method":"test","id":1}`,
		},
		{
			name:  "line with spaces",
			input: `  {"jsonrpc":"2.0"}  ` + "\n",
			want:  `  {"jsonrpc":"2.0"}  `,
		},
		{
			name:  "Windows line ending",
			input: `{"jsonrpc":"2.0","method":"test"}` + "\r\n",
			want:  `{"jsonrpc":"2.0","method":"test"}`,
		},
		{
			name:  "multiple empty lines before content",
			input: "\n\n\n" + `{"jsonrpc":"2.0"}` + "\n",
			want:  `{"jsonrpc":"2.0"}`,
		},
		{
			name:  "empty lines between",
			input: `{"first":1}` + "\n\n\n" + `{"second":2}` + "\n",
			want:  `{"first":1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn1, conn2 := newMockConnPair()
			defer conn1.Close()
			defer conn2.Close()

			codec := NewCodec(conn1)

			// Write test input to peer connection in goroutine
			// (io.Pipe blocks until read, so we need async write)
			go func() {
				conn2.Write([]byte(tt.input))
			}()

			got, err := codec.ReadMessage()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("ReadMessage() = %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestCodec_ReadMessage_MultipleMessages(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)

	messages := []string{
		`{"id":1}`,
		`{"id":2}`,
		`{"id":3}`,
	}

	// Write all messages in goroutine
	go func() {
		for _, msg := range messages {
			conn2.Write([]byte(msg + "\n"))
		}
	}()

	// Read all messages
	for i, want := range messages {
		got, err := codec.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage() %d error: %v", i, err)
		}
		if string(got) != want {
			t.Errorf("ReadMessage() %d = %q, want %q", i, string(got), want)
		}
	}
}

func TestCodec_ReadMessage_EOF(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	codec := NewCodec(conn1)

	// Close the peer connection to trigger EOF
	conn2.Close()

	_, err := codec.ReadMessage()
	if err == nil {
		t.Error("Expected error on EOF, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "read error") {
		t.Errorf("Expected read error, got: %v", err)
	}
}

func TestCodec_WriteMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "simple message",
			message: `{"jsonrpc":"2.0","result":"ok","id":1}`,
			want:    `{"jsonrpc":"2.0","result":"ok","id":1}` + "\n",
		},
		{
			name:    "empty object",
			message: `{}`,
			want:    `{}` + "\n",
		},
		{
			name:    "message with spaces",
			message: `  {"test": "value"}  `,
			want:    `  {"test": "value"}  ` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn1, conn2 := newMockConnPair()
			defer conn1.Close()
			defer conn2.Close()

			codec := NewCodec(conn1)

			// Read from peer in goroutine
			resultCh := make(chan string, 1)
			errorCh := make(chan error, 1)

			go func() {
				buf := make([]byte, len(tt.want))
				n, err := io.ReadFull(conn2, buf)
				if err != nil {
					errorCh <- err
					return
				}
				resultCh <- string(buf[:n])
			}()

			err := codec.WriteMessage([]byte(tt.message))
			if err != nil {
				t.Fatalf("WriteMessage() error: %v", err)
			}

			// Get result from goroutine
			select {
			case got := <-resultCh:
				if got != tt.want {
					t.Errorf("WriteMessage() wrote %q, want %q", got, tt.want)
				}
			case err := <-errorCh:
				t.Fatalf("Failed to read from peer: %v", err)
			}
		})
	}
}

func TestCodec_WriteMessage_Multiple(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	peerCodec := NewCodec(conn2)

	messages := []string{
		`{"id":1}`,
		`{"id":2}`,
		`{"id":3}`,
	}

	// Read all messages from peer in goroutine
	resultCh := make(chan []string, 1)
	errorCh := make(chan error, 1)

	go func() {
		var results []string
		for range messages {
			got, err := peerCodec.ReadMessage()
			if err != nil {
				errorCh <- err
				return
			}
			results = append(results, string(got))
		}
		resultCh <- results
	}()

	// Write all messages
	for _, msg := range messages {
		if err := codec.WriteMessage([]byte(msg)); err != nil {
			t.Fatalf("WriteMessage() error: %v", err)
		}
	}

	// Verify results
	select {
	case results := <-resultCh:
		for i, want := range messages {
			if results[i] != want {
				t.Errorf("Message %d = %q, want %q", i, results[i], want)
			}
		}
	case err := <-errorCh:
		t.Fatalf("ReadMessage() error: %v", err)
	}
}

func TestCodec_ReadJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "valid request",
			input: `{"jsonrpc":"2.0","method":"test","params":{"key":"value"},"id":1}` + "\n",
		},
		{
			name:  "valid response",
			input: `{"jsonrpc":"2.0","result":"success","id":1}` + "\n",
		},
		{
			name:  "valid notification",
			input: `{"jsonrpc":"2.0","method":"notify","params":{"data":"test"}}` + "\n",
		},
		{
			name:    "invalid JSON",
			input:   `{invalid json}` + "\n",
			wantErr: true,
		},
		{
			name:    "incomplete JSON",
			input:   `{"jsonrpc":"2.0"` + "\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn1, conn2 := newMockConnPair()
			defer conn1.Close()
			defer conn2.Close()

			codec := NewCodec(conn1)

			// Write test input in goroutine
			go func() {
				conn2.Write([]byte(tt.input))
			}()

			var msg Message
			err := codec.ReadJSON(&msg)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReadJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if msg.JSONRPC != "2.0" {
					t.Errorf("ReadJSON() JSONRPC = %q, want %q", msg.JSONRPC, "2.0")
				}
			}
		})
	}
}

func TestCodec_ReadJSON_DifferentTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFn  func(t *testing.T, msg *Message)
	}{
		{
			name:  "request type",
			input: `{"jsonrpc":"2.0","method":"test","id":1}` + "\n",
			checkFn: func(t *testing.T, msg *Message) {
				if !msg.IsRequest() {
					t.Error("Expected request type")
				}
			},
		},
		{
			name:  "notification type",
			input: `{"jsonrpc":"2.0","method":"notify"}` + "\n",
			checkFn: func(t *testing.T, msg *Message) {
				if !msg.IsNotification() {
					t.Error("Expected notification type")
				}
			},
		},
		{
			name:  "response type",
			input: `{"jsonrpc":"2.0","result":"ok","id":1}` + "\n",
			checkFn: func(t *testing.T, msg *Message) {
				if !msg.IsResponse() {
					t.Error("Expected response type")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn1, conn2 := newMockConnPair()
			defer conn1.Close()
			defer conn2.Close()

			codec := NewCodec(conn1)

			// Write in goroutine
			go func() {
				conn2.Write([]byte(tt.input))
			}()

			var msg Message
			if err := codec.ReadJSON(&msg); err != nil {
				t.Fatalf("ReadJSON() error: %v", err)
			}

			tt.checkFn(t, &msg)
		})
	}
}

func TestCodec_WriteJSON(t *testing.T) {
	tests := []struct {
		name    string
		message interface{}
		wantErr bool
	}{
		{
			name: "request",
			message: &Request{
				JSONRPC: "2.0",
				Method:  "test",
				Params:  json.RawMessage(`{"key":"value"}`),
				ID:      1,
			},
		},
		{
			name: "response",
			message: &Response{
				JSONRPC: "2.0",
				Result:  "success",
				ID:      1,
			},
		},
		{
			name: "notification",
			message: &Notification{
				JSONRPC: "2.0",
				Method:  "notify",
				Params:  map[string]string{"key": "value"},
			},
		},
		{
			name: "error response",
			message: &ErrorResponse{
				JSONRPC: "2.0",
				Error: &RPCError{
					Code:    -32600,
					Message: "Invalid Request",
				},
				ID: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn1, conn2 := newMockConnPair()
			defer conn1.Close()
			defer conn2.Close()

			codec := NewCodec(conn1)
			peerCodec := NewCodec(conn2)

			// Read from peer in goroutine if expecting success
			errorCh := make(chan error, 1)
			msgCh := make(chan Message, 1)

			if !tt.wantErr {
				go func() {
					var msg Message
					if err := peerCodec.ReadJSON(&msg); err != nil {
						errorCh <- err
						return
					}
					msgCh <- msg
				}()
			}

			err := codec.WriteJSON(tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Get result from goroutine
				select {
				case msg := <-msgCh:
					if msg.JSONRPC != "2.0" {
						t.Errorf("JSONRPC = %q, want %q", msg.JSONRPC, "2.0")
					}
				case err := <-errorCh:
					t.Fatalf("Failed to read back: %v", err)
				}
			}
		})
	}
}

func TestCodec_WriteJSON_InvalidType(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)

	// Try to marshal an unmarshalable type
	invalidData := make(chan int)

	err := codec.WriteJSON(invalidData)
	if err == nil {
		t.Error("Expected error when marshaling invalid type, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "json marshal error") {
		t.Errorf("Expected json marshal error, got: %v", err)
	}
}

func TestCodec_ConcurrentReadWrite(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec1 := NewCodec(conn1)
	codec2 := NewCodec(conn2)

	const numMessages = 10

	var wg sync.WaitGroup
	wg.Add(4)

	errors := make(chan error, 4)

	// Codec1 writes
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			msg := &Request{
				JSONRPC: "2.0",
				Method:  "test",
				ID:      i,
			}
			if err := codec1.WriteJSON(msg); err != nil {
				errors <- err
				return
			}
		}
	}()

	// Codec1 reads
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			var msg Response
			if err := codec1.ReadJSON(&msg); err != nil {
				errors <- err
				return
			}
		}
	}()

	// Codec2 writes
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			msg := &Response{
				JSONRPC: "2.0",
				Result:  "ok",
				ID:      i,
			}
			if err := codec2.WriteJSON(msg); err != nil {
				errors <- err
				return
			}
		}
	}()

	// Codec2 reads
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			var msg Request
			if err := codec2.ReadJSON(&msg); err != nil {
				errors <- err
				return
			}
		}
	}()

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
	}
}

func TestCodec_Close(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn2.Close()

	codec := NewCodec(conn1)

	if err := codec.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Verify connection is closed by trying to write
	_, err := conn1.Write([]byte("test"))
	if err == nil {
		t.Error("Expected error writing to closed connection")
	}
}

func TestCodec_ReadAfterClose(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn2.Close()

	codec := NewCodec(conn1)

	// Close the codec
	if err := codec.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Try to read after close
	_, err := codec.ReadMessage()
	if err == nil {
		t.Error("Expected error reading from closed connection")
	}
}

func TestCodec_WriteAfterClose(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn2.Close()

	codec := NewCodec(conn1)

	// Close the codec
	if err := codec.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	// Try to write after close
	err := codec.WriteMessage([]byte(`{"test":"data"}`))
	if err == nil {
		t.Error("Expected error writing to closed connection")
	}
}
