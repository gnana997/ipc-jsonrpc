package jsonrpcipc

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
)

func TestNewNotificationManager(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	nm := NewNotificationManager(codec)

	if nm == nil {
		t.Fatal("NewNotificationManager returned nil")
	}
	if nm.codec != codec {
		t.Error("NotificationManager codec not set correctly")
	}
	if nm.IsClosed() {
		t.Error("New NotificationManager should not be closed")
	}
}

func TestNotificationManager_Send(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	nm := NewNotificationManager(codec)

	// Read from peer in goroutine
	receivedCh := make(chan *Notification, 1)
	errorCh := make(chan error, 1)

	go func() {
		peerCodec := NewCodec(conn2)
		var notif Notification
		if err := peerCodec.ReadJSON(&notif); err != nil {
			errorCh <- err
			return
		}
		receivedCh <- &notif
	}()

	// Send notification
	params := map[string]interface{}{
		"message": "test notification",
		"value":   42,
	}

	err := nm.Send("test.method", params)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Verify notification was received
	select {
	case notif := <-receivedCh:
		if notif.JSONRPC != "2.0" {
			t.Errorf("JSONRPC = %q, want %q", notif.JSONRPC, "2.0")
		}
		if notif.Method != "test.method" {
			t.Errorf("Method = %q, want %q", notif.Method, "test.method")
		}
		// Verify params
		paramsJSON, _ := json.Marshal(notif.Params)
		var receivedParams map[string]interface{}
		json.Unmarshal(paramsJSON, &receivedParams)
		if receivedParams["message"] != "test notification" {
			t.Errorf("Params[message] = %v, want %v", receivedParams["message"], "test notification")
		}
	case err := <-errorCh:
		t.Fatalf("Failed to receive notification: %v", err)
	}
}

func TestNotificationManager_Send_AfterClose(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	nm := NewNotificationManager(codec)

	// Close the notification manager
	nm.Close()

	// Try to send after close
	err := nm.Send("test", nil)
	if err == nil {
		t.Error("Send() after Close() should return error")
	}
	if err.Error() != "notification manager is closed" {
		t.Errorf("Error message = %q, want %q", err.Error(), "notification manager is closed")
	}
}

func TestNotificationManager_Close(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	nm := NewNotificationManager(codec)

	if nm.IsClosed() {
		t.Error("New manager should not be closed")
	}

	nm.Close()

	if !nm.IsClosed() {
		t.Error("Manager should be closed after Close()")
	}

	// Multiple closes should be safe
	nm.Close()
	if !nm.IsClosed() {
		t.Error("Manager should still be closed")
	}
}

func TestNotificationManager_IsClosed(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	nm := NewNotificationManager(codec)

	// Initially not closed
	if nm.IsClosed() {
		t.Error("IsClosed() = true, want false initially")
	}

	// After close
	nm.Close()
	if !nm.IsClosed() {
		t.Error("IsClosed() = false, want true after Close()")
	}
}

func TestNotificationManager_ConcurrentSend(t *testing.T) {
	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	codec := NewCodec(conn1)
	nm := NewNotificationManager(codec)

	const numGoroutines = 10

	// Read notifications in goroutine
	receivedCh := make(chan int, numGoroutines)
	go func() {
		peerCodec := NewCodec(conn2)
		for i := 0; i < numGoroutines; i++ {
			var notif Notification
			if err := peerCodec.ReadJSON(&notif); err != nil {
				return
			}
			receivedCh <- 1
		}
	}()

	// Send notifications concurrently
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			err := nm.Send("concurrent.test", map[string]int{"id": id})
			if err != nil {
				t.Errorf("Send() error: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all notifications were received
	received := 0
	for i := 0; i < numGoroutines; i++ {
		<-receivedCh
		received++
	}

	if received != numGoroutines {
		t.Errorf("Received %d notifications, want %d", received, numGoroutines)
	}
}

func TestNewBroadcastManager(t *testing.T) {
	bm := NewBroadcastManager()

	if bm == nil {
		t.Fatal("NewBroadcastManager returned nil")
	}

	if bm.Count() != 0 {
		t.Errorf("New BroadcastManager Count() = %d, want 0", bm.Count())
	}
}

func TestBroadcastManager_Add_Remove(t *testing.T) {
	bm := NewBroadcastManager()

	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := &Connection{conn: conn1}

	// Add connection
	bm.Add(connection)

	if bm.Count() != 1 {
		t.Errorf("Count() after Add = %d, want 1", bm.Count())
	}

	// Remove connection
	bm.Remove(connection)

	if bm.Count() != 0 {
		t.Errorf("Count() after Remove = %d, want 0", bm.Count())
	}
}

func TestBroadcastManager_Add_Multiple(t *testing.T) {
	bm := NewBroadcastManager()

	connections := make([]*Connection, 5)
	for i := 0; i < 5; i++ {
		conn1, conn2 := newMockConnPair()
		defer conn1.Close()
		defer conn2.Close()
		connections[i] = &Connection{conn: conn1}
		bm.Add(connections[i])
	}

	if bm.Count() != 5 {
		t.Errorf("Count() = %d, want 5", bm.Count())
	}

	// Remove one
	bm.Remove(connections[2])

	if bm.Count() != 4 {
		t.Errorf("Count() after removing one = %d, want 4", bm.Count())
	}
}

func TestBroadcastManager_Count(t *testing.T) {
	bm := NewBroadcastManager()

	// Initially empty
	if bm.Count() != 0 {
		t.Errorf("Initial Count() = %d, want 0", bm.Count())
	}

	// Add connections
	for i := 0; i < 3; i++ {
		conn1, conn2 := newMockConnPair()
		defer conn1.Close()
		defer conn2.Close()
		bm.Add(&Connection{conn: conn1})
	}

	if bm.Count() != 3 {
		t.Errorf("Count() = %d, want 3", bm.Count())
	}
}

func TestBroadcastManager_Clear(t *testing.T) {
	bm := NewBroadcastManager()

	// Add multiple connections
	for i := 0; i < 5; i++ {
		conn1, conn2 := newMockConnPair()
		defer conn1.Close()
		defer conn2.Close()
		bm.Add(&Connection{conn: conn1})
	}

	if bm.Count() != 5 {
		t.Fatalf("Count() before Clear = %d, want 5", bm.Count())
	}

	// Clear all
	bm.Clear()

	if bm.Count() != 0 {
		t.Errorf("Count() after Clear = %d, want 0", bm.Count())
	}
}

func TestBroadcastManager_Broadcast(t *testing.T) {
	bm := NewBroadcastManager()

	// Create connections with their notification managers
	const numConns = 3
	receivedCh := make(chan struct{}, numConns)

	for i := 0; i < numConns; i++ {
		conn1, conn2 := newMockConnPair()
		defer conn1.Close()
		defer conn2.Close()

		// Create connection with notification manager
		codec := NewCodec(conn1)
		nm := NewNotificationManager(codec)
		connection := &Connection{
			conn:     conn1,
			notifier: nm,
		}

		bm.Add(connection)

		// Read from peer in goroutine
		go func(peer net.Conn) {
			peerCodec := NewCodec(peer)
			var notif Notification
			if err := peerCodec.ReadJSON(&notif); err == nil {
				if notif.Method == "broadcast.test" {
					receivedCh <- struct{}{}
				}
			}
		}(conn2)
	}

	// Broadcast to all connections
	count := bm.Broadcast("broadcast.test", map[string]string{"message": "hello all"})

	if count != numConns {
		t.Errorf("Broadcast() count = %d, want %d", count, numConns)
	}

	// Verify all received
	received := 0
	for i := 0; i < numConns; i++ {
		<-receivedCh
		received++
	}

	if received != numConns {
		t.Errorf("Received %d notifications, want %d", received, numConns)
	}
}

func TestBroadcastManager_Broadcast_Empty(t *testing.T) {
	bm := NewBroadcastManager()

	// Broadcast with no connections
	count := bm.Broadcast("test", nil)

	if count != 0 {
		t.Errorf("Broadcast() with no connections count = %d, want 0", count)
	}
}

func TestBroadcastManager_Add_SameConnection(t *testing.T) {
	bm := NewBroadcastManager()

	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := &Connection{conn: conn1}

	// Add same connection twice
	bm.Add(connection)
	bm.Add(connection)

	// Should still count as one (sync.Map behavior)
	if bm.Count() != 1 {
		t.Errorf("Count() with duplicate add = %d, want 1", bm.Count())
	}
}

func TestBroadcastManager_Remove_NonExistent(t *testing.T) {
	bm := NewBroadcastManager()

	conn1, conn2 := newMockConnPair()
	defer conn1.Close()
	defer conn2.Close()

	connection := &Connection{conn: conn1}

	// Remove without adding (should not panic)
	bm.Remove(connection)

	if bm.Count() != 0 {
		t.Errorf("Count() = %d, want 0", bm.Count())
	}
}
