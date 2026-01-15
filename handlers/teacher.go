package handlers

import (
	"encoding/json"
	"saber-websocket/models"
	"saber-websocket/server"
	"saber-websocket/utils"
)

func HandleTeacherConnect(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	client.ClientType = "teacher"
	client.ClientID = "teacher"
	client.Email = "Teacher Dashboard"
	hub.Register(client)
}

func HandleTeacherCommand(client *models.Client, msg models.Message, hub *server.Hub, logger *utils.Logger) {
	if client.ClientType != "teacher" { return }

	targetClientID, _ := msg.Data["targetClientId"].(string)
	command, _ := msg.Data["command"].(string)

	// Direct lookup for command routing
	student := hub.GetStudentSafe(targetClientID)
	
	if student == nil {
		// Notify teacher of failure
		failMsg, _ := json.Marshal(map[string]interface{}{
			"type": "command_failed",
			"data": map[string]interface{}{
				"targetClientId": targetClientID,
				"reason":         "Student not found",
			},
		})
		client.Send <- failMsg
		return
	}

	// Send command to student
	commandMsg, _ := json.Marshal(map[string]interface{}{
		"command": command,
		"data":    msg.Data["data"],
	})
	
	select {
	case student.Send <- commandMsg:
	default:
		logger.Warn("Command dropped, student buffer full")
	}
}