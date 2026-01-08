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
	Send       chan []byte
	ClientID   string
	ClientType string // "student" or "teacher"
	Email      string
	LastSeen   time.Time
	CurrentTabs map[string]interface{}
	mu         sync.RWMutex
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	Students   map[string]*Client
	Teacher    *Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *BroadcastMessage
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

// StudentConnectData for student_connect message
type StudentConnectData struct {
	ClientID string `json:"clientId"`
	Email    string `json:"email"`
}

// TabsUpdateData for tabs_update message
type TabsUpdateData struct {
	Tabs interface{} `json:"tabs"`
}

// ScreenshotData for screenshot message
type ScreenshotData struct {
	TabID     string `json:"tabId"`
	ImageData string `json:"imageData"`
}

// TeacherCommandData for teacher_command message
type TeacherCommandData struct {
	TargetClientID string                 `json:"targetClientId"`
	Command        string                 `json:"command"`
	Data           map[string]interface{} `json:"data,omitempty"`
}

// UpdateLastSeen updates the client's last seen timestamp
func (c *Client) UpdateLastSeen() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.LastSeen = time.Now()
}

// GetLastSeen returns the client's last seen timestamp
func (c *Client) GetLastSeen() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.LastSeen
}

// SetCurrentTabs updates the client's current tabs
func (c *Client) SetCurrentTabs(tabs map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.CurrentTabs = tabs
}

// GetCurrentTabs returns the client's current tabs
func (c *Client) GetCurrentTabs() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.CurrentTabs
}

// MarshalJSON creates a JSON representation of the client (without sensitive data)
func (c *Client) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ClientID string    `json:"clientId"`
		Email    string    `json:"email"`
		LastSeen time.Time `json:"lastSeen"`
	}{
		ClientID: c.ClientID,
		Email:    c.Email,
		LastSeen: c.GetLastSeen(),
	})
}