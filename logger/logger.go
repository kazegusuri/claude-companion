package logger

import (
	"fmt"
	"time"
)

// LogError logs an error message with consistent formatting
func LogError(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] ❌ ERROR: "+message+"\n", append([]interface{}{timestamp}, args...)...)
}

// LogInfo logs an info message with consistent formatting
func LogInfo(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] ℹ️ INFO: "+message+"\n", append([]interface{}{timestamp}, args...)...)
}

// LogWarning logs a warning message with consistent formatting
func LogWarning(message string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	fmt.Printf("[%s] ⚠️ WARNING: "+message+"\n", append([]interface{}{timestamp}, args...)...)
}
