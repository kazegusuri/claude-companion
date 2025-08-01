package logger

import (
	"fmt"
	"log"
	"time"
)

// LogError logs an error message with consistent formatting
func LogError(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	log.Printf("[%s] ❌ ERROR: %s", timestamp, formattedMessage)
}

// LogInfo logs an info message with consistent formatting
func LogInfo(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	log.Printf("[%s] ℹ️ INFO: %s", timestamp, formattedMessage)
}

// LogWarning logs a warning message with consistent formatting
func LogWarning(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	log.Printf("[%s] ⚠️ WARNING: %s", timestamp, formattedMessage)
}
