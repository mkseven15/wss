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

	// 1. OPTIMIZATION: Extract data immediately to free up the reader
	imageData, ok := msg.Data["imageData"].(string)
	if !ok {
		logger.Warn("Screenshot missing imageData")
		return
	}
	tabID := msg.Data["tabId"]
	clientID := client.ClientID

	// 2. OPTIMIZATION: Run compression asynchronously!
	// This prevents the read pump from blocking, so we don't get network backpressure.
	go func() {
		startTime := time.Now()

		// Compress screenshot
		compressedData, err := utils.CompressScreenshot(imageData, cfg.ScreenshotQuality)
		if err != nil {
			logger.Warn(fmt.Sprintf("Screenshot compression failed: %v", err))
			compressedData = imageData // Use original if compression fails
		}

		// Log compression stats (Optional: comment out in production to save I/O)
		// originalSize := len(imageData)
		// compressedSize := len(compressedData)
		// processingTime := time.Since(startTime).Milliseconds()
		// logger.Info(fmt.Sprintf("Screenshot processed for %s in %dms", clientID, processingTime))

		// Check if too much time passed (stale frame)
		if time.Since(startTime) > 2000*time.Millisecond {
			logger.Warn("Dropping stale screenshot (>2s old)")
			return
		}

		// Relay to teacher with compressed data
		teacher := hub.GetTeacher()
		if teacher != nil {
			relayMsg := map[string]interface{}{
				"type": "student_screenshot",
				"data": map[string]interface{}{
					"clientId": clientID,
					"payload": map[string]interface{}{
						"tabId":     tabID,
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
	}()
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
			// Optimization: Don't log on every dropped ping, it spams logs
		}
	}
}
