package logger

import (
	"fmt"
	"sync"
	"time"
)

var (
	debugMode bool
	mu        sync.RWMutex
)

// SetDebugMode sets the global debug mode
func SetDebugMode(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	debugMode = enabled
}

// IsDebugMode returns the current debug mode status
func IsDebugMode() bool {
	mu.RLock()
	defer mu.RUnlock()
	return debugMode
}

// LogError logs an error message with consistent formatting
func LogError(message string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	fmt.Printf("[%s] ❌ ERROR: %s\n", timestamp, formattedMessage)
}

// LogInfo logs an info message with consistent formatting
func LogInfo(message string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	fmt.Printf("[%s] ℹ️ INFO: %s\n", timestamp, formattedMessage)
}

// LogWarning logs a warning message with consistent formatting
func LogWarning(message string, args ...any) {
	timestamp := time.Now().Format("15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)
	fmt.Printf("[%s] ⚠️ WARNING: %s\n", timestamp, formattedMessage)
}

// DebugInfo logs an info message only when debug mode is enabled
func DebugInfo(message string, args ...any) {
	mu.RLock()
	enabled := debugMode
	mu.RUnlock()

	if enabled {
		LogInfo(message, args...)
	}
}

// DebugWarning logs a warning message only when debug mode is enabled
func DebugWarning(message string, args ...any) {
	mu.RLock()
	enabled := debugMode
	mu.RUnlock()

	if enabled {
		LogWarning(message, args...)
	}
}
