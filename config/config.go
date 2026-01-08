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
	ScreenshotQuality int
	MessageBufferSize int
}

func LoadConfig() *Config {
	return &Config{
		Port:              getEnv("PORT", "8080"),
		MaxStudents:       getEnvInt("MAX_STUDENTS", 50),
		MaxMessageSize:    10 * 1024 * 1024, // 10MB for screenshots
		WriteTimeout:      10 * time.Second,
		PongTimeout:       60 * time.Second,
		PingInterval:      54 * time.Second, // Slightly less than pong timeout
		ScreenshotQuality: getEnvInt("SCREENSHOT_QUALITY", 60),
		MessageBufferSize: 256, // Buffered channel size
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