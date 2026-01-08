package handlers

import (
	"encoding/json"
	"fmt"
	"saber-websocket/models"
	"saber-websocket/server"
	"saber-websocket/utils"
)

func HandleTeacherConnect(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	client.ClientType = "teacher"
	client.ClientID = "teacher"
	client.Email = "Teacher Dashboard"

	hub.Register <- client
}

func HandleTeacherCommand(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	if client.ClientType != "teacher" {
		logger.Warn("Non-teacher tried to send command")
		return
	}

	targetClientID, ok := msg.Data["targetClientId"].(string)
	if !ok {
		logger.Warn("teacher_command missing targetClientId")
		return
	}

	command, ok := msg.Data["command"].(string)
	if !ok {
		logger.Warn("teacher_command missing command")
		return
	}

	// Get the target student
	student := hub.GetStudent(targetClientID)
	if student == nil {
		logger.Warn(fmt.Sprintf("Command target student %s not found", targetClientID))

		// Notify teacher of failure
		failMsg := map[string]interface{}{
			"type": "command_failed",
			"data": map[string]interface{}{
				"targetClientId": targetClientID,
				"reason":         "Student not found",
			},
		}

		if data, err := json.Marshal(failMsg); err == nil {
			select {
			case client.Send <- data:
			default:
				logger.Warn("Teacher send channel full, dropping command_failed")
			}
		}
		return
	}

	logger.Info(fmt.Sprintf("Relaying command '%s' to student %s", command, targetClientID))

	// Relay command to student
	commandMsg := map[string]interface{}{
		"command": command,
		"data":    msg.Data["data"],
	}

	if data, err := json.Marshal(commandMsg); err == nil {
		hub.Broadcast <- &models.BroadcastMessage{
			Target:  targetClientID,
			Message: data,
		}
	}
}