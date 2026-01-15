package server

import (
	"encoding/json"
	"fmt"
	"saber-websocket/config"
	"saber-websocket/models"
	"saber-websocket/utils"
	"sync"
)

type Hub struct {
	students   map[string]*models.Client
	teacher    *models.Client
	register   chan *models.Client
	unregister chan *models.Client
	broadcast  chan *models.BroadcastMessage
	config     *config.Config
	logger     *utils.Logger
	mu         sync.RWMutex
}

func NewHub(cfg *config.Config, logger *utils.Logger) *Hub {
	return &Hub{
		students:   make(map[string]*models.Client),
		teacher:    nil,
		register:   make(chan *models.Client),
		unregister: make(chan *models.Client),
		broadcast:  make(chan *models.BroadcastMessage, 256), // Larger buffer for control messages
		config:     cfg,
		logger:     logger,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.handleRegister(client)

		case client := <-h.unregister:
			h.handleUnregister(client)

		case message := <-h.broadcast:
			h.handleBroadcast(message)
		}
	}
}

// GetTeacherSafe returns the teacher client pointer safely.
// This allows handlers to bypass the main Hub loop for streaming.
func (h *Hub) GetTeacherSafe() *models.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.teacher
}

// GetStudentSafe returns a student client pointer safely.
func (h *Hub) GetStudentSafe(clientID string) *models.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.students[clientID]
}

func (h *Hub) handleRegister(client *models.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.ClientType == "teacher" {
		if h.teacher != nil {
			h.logger.Warn("New teacher connecting, closing old session")
			close(h.teacher.Send)
		}
		h.teacher = client
		h.logger.Info("Teacher connected")
		
		// Push initial state immediately
		go h.sendInitialStudentList(client)

	} else if client.ClientType == "student" {
		if len(h.students) >= h.config.MaxStudents {
			h.sendError(client, "Class is full")
			close(client.Send)
			return
		}

		h.students[client.ClientID] = client
		h.logger.Info(fmt.Sprintf("Student + : %s (%s)", client.Email, client.ClientID))

		// Notify Teacher (Control Message)
		if h.teacher != nil {
			h.sendToTeacherInternal(map[string]interface{}{
				"type": "student_connected",
				"data": map[string]interface{}{
					"clientId": client.ClientID,
					"email":    client.Email,
				},
			})
		}
	}
}

func (h *Hub) handleUnregister(client *models.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.ClientType == "teacher" {
		if h.teacher == client {
			h.teacher = nil
			close(client.Send)
			h.logger.Info("Teacher disconnected")
		}
	} else if client.ClientType == "student" {
		if _, ok := h.students[client.ClientID]; ok {
			delete(h.students, client.ClientID)
			close(client.Send)
			h.logger.Info(fmt.Sprintf("Student - : %s", client.ClientID))

			if h.teacher != nil {
				h.sendToTeacherInternal(map[string]interface{}{
					"type": "student_disconnected",
					"data": map[string]interface{}{
						"clientId": client.ClientID,
					},
				})
			}
		}
	}
}

// handleBroadcast processes low-priority control messages (chat, commands)
func (h *Hub) handleBroadcast(message *models.BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if message.Target == "teacher" && h.teacher != nil {
		h.trySend(h.teacher, message.Message)
	} else if message.Target == "student" {
		for _, s := range h.students {
			h.trySend(s, message.Message)
		}
	} else if client, ok := h.students[message.Target]; ok {
		h.trySend(client, message.Message)
	}
}

// trySend attempts to send a message. If buffer is full, it drops it (Backpressure).
func (h *Hub) trySend(client *models.Client, msg []byte) {
	select {
	case client.Send <- msg:
	default:
		// Buffer full - Drop message to prevent server blocking
		// This is acceptable for real-time systems
	}
}

// Internal helper to send map as json
func (h *Hub) sendToTeacherInternal(msg interface{}) {
	if data, err := json.Marshal(msg); err == nil {
		if h.teacher != nil {
			h.trySend(h.teacher, data)
		}
	}
}

func (h *Hub) sendError(client *models.Client, errorMsg string) {
	msg := map[string]interface{}{"type": "error", "message": errorMsg}
	if data, err := json.Marshal(msg); err == nil {
		client.Send <- data
	}
}

func (h *Hub) sendInitialStudentList(teacher *models.Client) {
	// Need lock to read map, but we're already inside a lock in handleRegister?
	// No, this is called as a goroutine, so we need RLock
	h.mu.RLock()
	defer h.mu.RUnlock()

	list := make([]map[string]interface{}, 0, len(h.students))
	for _, s := range h.students {
		list = append(list, map[string]interface{}{
			"clientId": s.ClientID,
			"email":    s.Email,
		})
	}

	msg := map[string]interface{}{
		"type": "initial_student_list",
		"data": list,
	}
	if data, err := json.Marshal(msg); err == nil {
		h.trySend(teacher, data)
	}
}

// API Methods
func (h *Hub) Register(c *models.Client)   { h.register <- c }
func (h *Hub) Unregister(c *models.Client) { h.unregister <- c }
func (h *Hub) Broadcast(m *models.BroadcastMessage) { h.broadcast <- m }