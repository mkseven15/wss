package utils

import (
	"fmt"
	"log"
	"os"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level  LogLevel
	logger *log.Logger
}

func NewLogger() *Logger {
	return &Logger{
		level:  INFO,
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) log(level LogLevel, levelStr string, message string) {
	if level >= l.level {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		l.logger.Printf("[%s] %s: %s", timestamp, levelStr, message)
	}
}

func (l *Logger) Debug(message string) {
	l.log(DEBUG, "DEBUG", message)
}

func (l *Logger) Info(message string) {
	l.log(INFO, "INFO", message)
}

func (l *Logger) Warn(message string) {
	l.log(WARN, "WARN", message)
}

func (l *Logger) Error(message string) {
	l.log(ERROR, "ERROR", message)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Warn(fmt.Sprintf(format, v...))
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Error(fmt.Sprintf(format, v...))
}