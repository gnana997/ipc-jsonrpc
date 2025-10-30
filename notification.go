package jsonrpcipc

import (
	"fmt"
	"sync"
)

// NotificationManager handles sending notifications to clients.
//
// Notifications are JSON-RPC messages without an ID field, meaning they
// don't expect a response from the client.
type NotificationManager struct {
	codec   *LineDelimitedCodec
	mu      sync.Mutex // Protects writes to codec
	closed  bool
	closeMu sync.RWMutex
}

// NewNotificationManager creates a new notification manager.
func NewNotificationManager(codec *LineDelimitedCodec) *NotificationManager {
	return &NotificationManager{
		codec: codec,
	}
}

// Send sends a notification to the client.
//
// Parameters:
//   - method: The notification method name
//   - params: The notification parameters (will be JSON-marshaled)
//
// Returns an error if the notification cannot be sent.
//
// Thread-safety: This method is safe to call concurrently.
func (nm *NotificationManager) Send(method string, params interface{}) error {
	nm.closeMu.RLock()
	if nm.closed {
		nm.closeMu.RUnlock()
		return fmt.Errorf("notification manager is closed")
	}
	nm.closeMu.RUnlock()

	notification := &Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	nm.mu.Lock()
	defer nm.mu.Unlock()

	return nm.codec.WriteJSON(notification)
}

// Close marks the notification manager as closed.
// After calling Close, Send will return an error.
func (nm *NotificationManager) Close() {
	nm.closeMu.Lock()
	defer nm.closeMu.Unlock()
	nm.closed = true
}

// IsClosed returns true if the notification manager has been closed.
func (nm *NotificationManager) IsClosed() bool {
	nm.closeMu.RLock()
	defer nm.closeMu.RUnlock()
	return nm.closed
}

// BroadcastManager manages broadcasting notifications to multiple connections.
type BroadcastManager struct {
	connections sync.Map // map[*Connection]bool
}

// NewBroadcastManager creates a new broadcast manager.
func NewBroadcastManager() *BroadcastManager {
	return &BroadcastManager{}
}

// Add adds a connection to the broadcast list.
func (bm *BroadcastManager) Add(conn *Connection) {
	bm.connections.Store(conn, true)
}

// Remove removes a connection from the broadcast list.
func (bm *BroadcastManager) Remove(conn *Connection) {
	bm.connections.Delete(conn)
}

// Broadcast sends a notification to all connected clients.
//
// Parameters:
//   - method: The notification method name
//   - params: The notification parameters
//
// Returns the number of connections the notification was sent to.
// Errors sending to individual connections are logged but don't stop the broadcast.
func (bm *BroadcastManager) Broadcast(method string, params interface{}) int {
	count := 0

	bm.connections.Range(func(key, value interface{}) bool {
		conn := key.(*Connection)
		if err := conn.Notify(method, params); err == nil {
			count++
		}
		return true // Continue iteration
	})

	return count
}

// Count returns the number of active connections.
func (bm *BroadcastManager) Count() int {
	count := 0
	bm.connections.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Clear removes all connections from the broadcast list.
func (bm *BroadcastManager) Clear() {
	bm.connections.Range(func(key, value interface{}) bool {
		bm.connections.Delete(key)
		return true
	})
}
