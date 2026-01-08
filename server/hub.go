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
		register:   make(chan *models.Client, cfg.MessageBufferSize),
		unregister: make(chan *models.Client, cfg.MessageBufferSize),
		broadcast:  make(chan *models.BroadcastMessage, cfg.MessageBufferSize),
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

func (h *Hub) handleRegister(client *models.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.ClientType == "teacher" {
		if h.teacher != nil {
			h.logger.Warn("New teacher connecting, closing old teacher connection")
			close(h.teacher.Send)
		}
		h.teacher = client
		h.logger.Info("Teacher connected")

		// Send initial student list to teacher
		h.sendInitialStudentList()

	} else if client.ClientType == "student" {
		if len(h.students) >= h.config.MaxStudents {
			h.logger.Warn(fmt.Sprintf("Max students (%d) reached, rejecting %s", h.config.MaxStudents, client.ClientID))
			msg := map[string]interface{}{
				"type":    "error",
				"message": "Maximum student capacity reached",
			}
			if data, err := json.Marshal(msg); err == nil {
				client.Send <- data
			}
			close(client.Send)
			return
		}

		h.students[client.ClientID] = client
		h.logger.Info(fmt.Sprintf("Student connected: %s (ID: %s) - Total: %d", client.Email, client.ClientID, len(h.students)))

		// Notify teacher
		if h.teacher != nil {
			msg := map[string]interface{}{
				"type": "student_connected",
				"data": map[string]interface{}{
					"clientId": client.ClientID,
					"email":    client.Email,
				},
			}
			if data, err := json.Marshal(msg); err == nil {
				select {
				case h.teacher.Send <- data:
				default:
					h.logger.Warn("Teacher send channel full, dropping message")
				}
			}
		}

		// Send acknowledgment to student
		ack := map[string]interface{}{
			"type":    "server_ack",
			"message": "Connected successfully",
		}
		if data, err := json.Marshal(ack); err == nil {
			select {
			case client.Send <- data:
			default:
				h.logger.Warn("Student send channel full, dropping ack")
			}
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
			h.logger.Info(fmt.Sprintf("Student disconnected: %s (ID: %s) - Remaining: %d", client.Email, client.ClientID, len(h.students)))

			// Notify teacher
			if h.teacher != nil {
				msg := map[string]interface{}{
					"type": "student_disconnected",
					"data": map[string]interface{}{
						"clientId": client.ClientID,
					},
				}
				if data, err := json.Marshal(msg); err == nil {
					select {
					case h.teacher.Send <- data:
					default:
						h.logger.Warn("Teacher send channel full, dropping disconnect notification")
					}
				}
			}
		}
	}
}

func (h *Hub) handleBroadcast(message *models.BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	switch message.Target {
	case "teacher":
		if h.teacher != nil {
			select {
			case h.teacher.Send <- message.Message:
			default:
				h.logger.Warn("Teacher send channel full, dropping broadcast")
			}
		}

	case "student":
		// Broadcast to all students (rarely used)
		for _, student := range h.students {
			select {
			case student.Send <- message.Message:
			default:
				h.logger.Warn(fmt.Sprintf("Student %s send channel full, dropping broadcast", student.ClientID))
			}
		}

	default:
		// Specific client ID
		if student, ok := h.students[message.Target]; ok {
			select {
			case student.Send <- message.Message:
			default:
				h.logger.Warn(fmt.Sprintf("Student %s send channel full, dropping message", message.Target))
			}
		}
	}
}

func (h *Hub) sendInitialStudentList() {
	studentList := make([]map[string]interface{}, 0, len(h.students))
	for _, student := range h.students {
		studentList = append(studentList, map[string]interface{}{
			"clientId": student.ClientID,
			"email":    student.Email,
		})
	}

	msg := map[string]interface{}{
		"type": "initial_student_list",
		"data": studentList,
	}

	if data, err := json.Marshal(msg); err == nil {
		select {
		case h.teacher.Send <- data:
		default:
			h.logger.Warn("Teacher send channel full, dropping initial student list")
		}
	}
}

func (h *Hub) GetStudent(clientID string) *models.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.students[clientID]
}

func (h *Hub) GetTeacher() *models.Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.teacher
}

// Register adds a client to the hub
func (h *Hub) Register(client *models.Client) {
	h.register <- client
}

// Unregister removes a client from the hub
func (h *Hub) Unregister(client *models.Client) {
	h.unregister <- client
}

// Broadcast sends a message to clients
func (h *Hub) Broadcast(message *models.BroadcastMessage) {
	h.broadcast <- message
}
