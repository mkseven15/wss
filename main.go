package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"saber-websocket/config"
	"saber-websocket/handlers"
	"saber-websocket/server"
	"saber-websocket/utils"
	"syscall"
	"time"
)

func main() {
	// Initialize logger
	logger := utils.NewLogger()
	logger.Info("Starting Saber WebSocket Server...")

	// Load configuration
	cfg := config.LoadConfig()
	logger.Info(fmt.Sprintf("Configuration loaded: Port=%s, MaxStudents=%d", cfg.Port, cfg.MaxStudents))

	// Create the hub (central message router)
	hub := server.NewHub(cfg, logger)
	go hub.Run()

	// Setup HTTP server with WebSocket endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handlers.ServeWs(hub, w, r, cfg, logger)
	})

	// Health check endpoint for Render
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info(fmt.Sprintf("Server listening on port %s", cfg.Port))
		logger.Info(fmt.Sprintf("WebSocket endpoint: ws://localhost:%s/", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Sprintf("Server error: %v", err))
			os.Exit(1)
		}
	}()

	// Graceful shutdown handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(fmt.Sprintf("Server forced to shutdown: %v", err))
	}

	logger.Info("Server stopped")
}