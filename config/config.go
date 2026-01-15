package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port              string
	MaxStudents       int
	MaxMessageSize    int64
	WriteTimeout      time.Duration
	PongTimeout       time.Duration
	PingInterval      time.Duration
	// Increased buffer to handle burst traffic, but logic will drop packets if full
	MessageBufferSize int 
}

func LoadConfig() *Config {
	return &Config{
		Port:           getEnv("PORT", "8080"),
		MaxStudents:    getEnvInt("MAX_STUDENTS", 100), // Increased default
		MaxMessageSize: 10 * 1024 * 1024, // 10MB
		WriteTimeout:   5 * time.Second,  // Tighter timeout to detect lag quickly
		PongTimeout:    60 * time.Second,
		PingInterval:   50 * time.Second,
		MessageBufferSize: 128, 
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}