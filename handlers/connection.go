package handlers

import (
	"encoding/json"
	"net/http"
	"saber-websocket/config"
	"saber-websocket/models"
	"saber-websocket/server"
	"saber-websocket/utils"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins - in production, restrict this
		return true
	},
}

func ServeWs(hub *server.Hub, w http.ResponseWriter, r *http.Request, cfg *config.Config, logger *utils.Logger) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("WebSocket upgrade failed: " + err.Error())
		return
	}

	client := &models.Client{
		Conn:        conn,
		Send:        make(chan []byte, cfg.MessageBufferSize),
		LastSeen:    time.Now(),
		CurrentTabs: make(map[string]interface{}),
	}

	// Start read and write pumps
	go writePump(client, cfg, logger)
	go readPump(client, hub, cfg, logger)
}

func readPump(client *models.Client, hub *server.Hub, cfg *config.Config, logger *utils.Logger) {
	defer func() {
		hub.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(cfg.MaxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(cfg.PongTimeout))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(cfg.PongTimeout))
		return nil
	})

	for {
		_, messageBytes, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Warn("Unexpected close: " + err.Error())
			}
			break
		}

		client.UpdateLastSeen()

		var msg models.Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			logger.Warn("Invalid JSON message: " + err.Error())
			continue
		}

		// Route message based on type
		switch msg.Type {
		case "student_connect":
			HandleStudentConnect(client, msg, hub, logger)
		case "tabs_update", "tab_created", "tab_updated", "tab_removed":
			HandleTabUpdate(client, msg, hub, logger)
		case "screenshot":
			HandleScreenshot(client, msg, hub, cfg, logger)
		case "screenshot_error", "screenshot_skipped":
			HandleScreenshotError(client, msg, hub, logger)
		case "ping":
			HandlePing(client, msg, hub, logger)
		case "teacher_connect":
			HandleTeacherConnect(client, msg, hub, logger)
		case "teacher_command":
			HandleTeacherCommand(client, msg, hub, logger)
		default:
			logger.Warn("Unknown message type: " + msg.Type)
		}
	}
}

func writePump(client *models.Client, cfg *config.Config, logger *utils.Logger) {
	ticker := time.NewTicker(cfg.PingInterval)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			if !ok {
				// Hub closed the channel
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current write
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(cfg.WriteTimeout))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
