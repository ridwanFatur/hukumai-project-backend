package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Hub maintains active WebSocket connections keyed by user ID.
// Each user may have at most one active connection at a time.
type Hub struct {
	mu          sync.RWMutex
	connections map[uint]*websocket.Conn
}

// GlobalHub is the singleton hub used throughout the application.
var GlobalHub = &Hub{
	connections: make(map[uint]*websocket.Conn),
}

// Register associates a WebSocket connection with a user.
// If the user already has a connection, the old one is closed first.
func (h *Hub) Register(userID uint, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.connections[userID]; ok {
		existing.Close()
	}
	h.connections[userID] = conn
}

// Unregister removes the WebSocket connection for a user.
func (h *Hub) Unregister(userID uint) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.connections, userID)
}

// SendJSON serialises v as JSON and writes it to the user's connection.
// Returns nil (no-op) if the user is not currently connected.
func (h *Hub) SendJSON(userID uint, v interface{}) error {
	h.mu.RLock()
	conn, ok := h.connections[userID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	return conn.WriteJSON(v)
}
