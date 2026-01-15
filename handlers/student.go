package handlers

import (
	"encoding/json"
	"saber-websocket/config"
	"saber-websocket/models"
	"saber-websocket/server"
	"saber-websocket/utils"
	"time"
)

func HandleStudentConnect(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	data, ok := msg.Data["clientId"]
	if !ok { return }
	
	clientID, ok := data.(string)
	if !ok { return }

	email := "N/A"
	if emailData, ok := msg.Data["email"].(string); ok {
		email = emailData
	}

	client.ClientID = clientID
	client.Email = email
	client.ClientType = "student"

	hub.Register(client)
}

func HandleTabUpdate(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	if client.ClientType != "student" { return }

	// Store tabs in memory if it's a full update
	if msg.Type == "tabs_update" {
		if tabs, ok := msg.Data["tabs"].(map[string]interface{}); ok {
			client.SetCurrentTabs(tabs)
		}
	}

	// 1. Relay payload preparation
	relayMsg := map[string]interface{}{
		"type": "student_" + msg.Type,
		"data": map[string]interface{}{
			"clientId": client.ClientID,
			"payload":  msg.Data,
		},
	}
	
	// 2. Control Message -> Use Standard Broadcast Channel
	if data, err := json.Marshal(relayMsg); err == nil {
		hub.Broadcast(&models.BroadcastMessage{
			Target:  "teacher",
			Message: data,
		})
	}
}

// HandleScreenshot implements the FAST-PATH Relay
func HandleScreenshot(client *models.Client, msg models.Message, hub *server.Hub, cfg *config.Config, logger *utils.Logger) {
	// 1. Validation
	if client.ClientType != "student" { return }

	// 2. Data Extraction & Relay Construction
	relayMsg := map[string]interface{}{
		"type": "student_screenshot",
		"data": map[string]interface{}{
			"clientId": client.ClientID,
			"payload":  msg.Data, // Contains {tabId, imageData}
		},
	}
	
	finalBytes, err := json.Marshal(relayMsg)
	if err != nil { return }

	// 3. FAST-PATH: Direct Stream Injection
	teacher := hub.GetTeacherSafe()
	if teacher != nil {
		select {
		case teacher.Send <- finalBytes:
			// Success
		default:
			// Drop frame if teacher is lagging (Backpressure)
		}
	}
}

// Added missing HandlePing
func HandlePing(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	pongMsg := map[string]interface{}{
		"type": "pong",
		"data": map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		},
	}

	if data, err := json.Marshal(pongMsg); err == nil {
		select {
		case client.Send <- data:
		default:
			// Optimization: Don't log dropped pings
		}
	}
}

// Added missing HandleScreenshotError
func HandleScreenshotError(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	if client.ClientType != "student" { return }

	// Just relay the error to the teacher so they know why the screen is black
	relayMsg := map[string]interface{}{
		"type": "student_" + msg.Type, // e.g., student_screenshot_error
		"data": map[string]interface{}{
			"clientId": client.ClientID,
			"payload":  msg.Data,
		},
	}

	if data, err := json.Marshal(relayMsg); err == nil {
		hub.Broadcast(&models.BroadcastMessage{
			Target:  "teacher",
			Message: data,
		})
	}
}