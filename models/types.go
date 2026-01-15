package models

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket client
type Client struct {
	Hub        *Hub
	Conn       *websocket.Conn
	// Buffered channel for outbound messages
	Send       chan []byte
	ClientID   string
	ClientType string // "student" or "teacher"
	Email      string
	LastSeen   time.Time
	
	// We use a RWMutex specifically for client state to allow 
	// high-speed concurrent reads of client status
	mu         sync.RWMutex
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	Students   map[string]*Client
	Teacher    *Client
	
	// Lifecycle channels
	Register   chan *Client
	Unregister chan *Client
	
	// Broadcast channel (For low-priority Chat/Status events)
	Broadcast  chan *BroadcastMessage
	
	// Mutex to protect the maps
	mu         sync.RWMutex
}

// BroadcastMessage wraps a message with its target
type BroadcastMessage struct {
	Target  string // "teacher", "student", or specific clientID
	Message []byte
}

// Message represents the base WebSocket message structure
type Message struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data,omitempty"`
}

// --- Helper Methods ---

func (c *Client) UpdateLastSeen() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastSeen = time.Now()
}

func (c *Client) MarshalJSON() ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return json.Marshal(struct {
		ClientID string    `json:"clientId"`
		Email    string    `json:"email"`
		LastSeen time.Time `json:"lastSeen"`
	}{
		ClientID: c.ClientID,
		Email:    c.Email,
		LastSeen: c.LastSeen,
	})
}