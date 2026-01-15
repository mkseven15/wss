package handlers

import (
	"encoding/json"
	"saber-websocket/config"
	"saber-websocket/models"
	"saber-websocket/server"
	"saber-websocket/utils"
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

	// 1. Relay payload preparation
	relayMsg := map[string]interface{}{
		"type": "student_" + msg.Type,
		"data": map[string]interface{}{
			"clientId": client.ClientID,
			"payload":  msg.Data,
		},
	}
	
	// 2. Control Message -> Use Standard Broadcast Channel
	// Because tab updates are low frequency and reliable delivery matters more than speed.
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

	// 2. Data Extraction
	// NOTE: We do NOT process/compress the image here. We act as a blind relay.
	// The Client (Chrome Ext) MUST compress before sending.
	
	// Construct the relay message
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
	// Instead of dumping this into the Hub.Broadcast channel (which waits in line),
	// we grab the teacher connection immediately and push.
	
	teacher := hub.GetTeacherSafe()
	if teacher != nil {
		select {
		case teacher.Send <- finalBytes:
			// Success
		default:
			// Teacher is lagging or disconnected.
			// ACTION: DROP THE FRAME.
			// Why? Because sending a screenshot from 2 seconds ago is useless.
			// We want them to see the *next* frame immediately when they catch up.
			// logger.Warn("Dropped frame for " + client.ClientID)
		}
	}
}