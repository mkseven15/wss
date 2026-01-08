package handlers

import (
	"encoding/json"
	"fmt"
	"saber-websocket/config"
	"saber-websocket/models"
	"saber-websocket/server"
	"saber-websocket/utils"
	"time"
)

func HandleStudentConnect(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	data, ok := msg.Data["clientId"]
	if !ok {
		logger.Warn("student_connect missing clientId")
		return
	}

	clientID, ok := data.(string)
	if !ok {
		logger.Warn("student_connect clientId is not a string")
		return
	}

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
	if client.ClientType != "student" {
		logger.Warn("Non-student sent tab update")
		return
	}

	// Store full tab list for tabs_update
	if msg.Type == "tabs_update" {
		if tabs, ok := msg.Data["tabs"].(map[string]interface{}); ok {
			client.SetCurrentTabs(tabs)
		}
	}

	// Relay to teacher
	teacher := hub.GetTeacher()
	if teacher != nil {
		relayMsg := map[string]interface{}{
			"type": "student_" + msg.Type,
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
}

func HandleScreenshot(client *models.Client, msg models.Message, hub *server.Hub, cfg *config.Config, logger *utils.Logger) {
	if client.ClientType != "student" {
		logger.Warn("Non-student sent screenshot")
		return
	}

	startTime := time.Now()

	// Extract screenshot data
	imageData, ok := msg.Data["imageData"].(string)
	if !ok {
		logger.Warn("Screenshot missing imageData")
		return
	}

	// Compress screenshot
	compressedData, err := utils.CompressScreenshot(imageData, cfg.ScreenshotQuality)
	if err != nil {
		logger.Warn(fmt.Sprintf("Screenshot compression failed: %v", err))
		compressedData = imageData // Use original if compression fails
	}

	// Log compression stats
	originalSize := len(imageData)
	compressedSize := len(compressedData)
	compressionRatio := float64(originalSize-compressedSize) / float64(originalSize) * 100
	processingTime := time.Since(startTime).Milliseconds()

	logger.Info(fmt.Sprintf("Screenshot from %s: %d -> %d bytes (%.1f%% reduction) in %dms",
		client.ClientID, originalSize, compressedSize, compressionRatio, processingTime))

	// Relay to teacher with compressed data
	teacher := hub.GetTeacher()
	if teacher != nil {
		relayMsg := map[string]interface{}{
			"type": "student_screenshot",
			"data": map[string]interface{}{
				"clientId": client.ClientID,
				"payload": map[string]interface{}{
					"tabId":     msg.Data["tabId"],
					"imageData": compressedData,
				},
			},
		}

		if data, err := json.Marshal(relayMsg); err == nil {
			hub.Broadcast(&models.BroadcastMessage{
				Target:  "teacher",
				Message: data,
			})
		}
	}
}

func HandleScreenshotError(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	if client.ClientType != "student" {
		logger.Warn("Non-student sent screenshot error")
		return
	}

	// Relay to teacher
	teacher := hub.GetTeacher()
	if teacher != nil {
		relayMsg := map[string]interface{}{
			"type": "student_" + msg.Type,
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
}

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
			logger.Warn("Client send channel full, dropping pong")
		}
	}
}
