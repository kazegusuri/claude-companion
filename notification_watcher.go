package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kazegusuri/claude-companion/narrator"
)

// NotificationEvent represents a notification event from the log file
type NotificationEvent struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Message        string `json:"message"`
}

// NotificationWatcher watches the notification log file for new events
type NotificationWatcher struct {
	filePath  string
	narrator  narrator.Narrator
	debugMode bool
}

// NewNotificationWatcher creates a new notification watcher
func NewNotificationWatcher(filePath string, narrator narrator.Narrator) *NotificationWatcher {
	return &NotificationWatcher{
		filePath:  filePath,
		narrator:  narrator,
		debugMode: false,
	}
}

// SetDebugMode enables or disables debug mode
func (w *NotificationWatcher) SetDebugMode(enabled bool) {
	w.debugMode = enabled
}

// Watch starts watching the notification log file
func (w *NotificationWatcher) Watch() error {
	file, err := os.Open(w.filePath)
	if err != nil {
		// If file doesn't exist, wait for it to be created
		if os.IsNotExist(err) {
			log.Printf("Notification log file %s does not exist, waiting for it to be created...", w.filePath)
			return w.waitForFileAndWatch()
		}
		return fmt.Errorf("failed to open notification log: %w", err)
	}
	defer file.Close()

	// Move to end of file to only watch new events
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	log.Printf("Watching notification log: %s", w.filePath)
	return w.tailFile(file)
}

// waitForFileAndWatch waits for the file to be created then starts watching
func (w *NotificationWatcher) waitForFileAndWatch() error {
	for {
		if _, err := os.Stat(w.filePath); err == nil {
			log.Printf("Notification log file created: %s", w.filePath)
			return w.Watch()
		}
		time.Sleep(1 * time.Second)
	}
}

// tailFile continuously reads new lines from the file
func (w *NotificationWatcher) tailFile(file *os.File) error {
	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No new data, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("error reading line: %w", err)
		}

		// Process the line
		if len(line) > 0 {
			w.processNotificationLine(line)
		}
	}
}

// processNotificationLine processes a single line from the notification log
func (w *NotificationWatcher) processNotificationLine(line string) {
	// Parse JSON notification event
	var event NotificationEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		if w.debugMode {
			log.Printf("Failed to parse notification event: %v", err)
		}
		return
	}

	// Format and display the notification
	w.formatNotificationEvent(&event)
}

// parsePermissionMessage parses permission messages to extract tool/MCP information
func (w *NotificationWatcher) parsePermissionMessage(message string) (isPermission bool, toolName string, mcpName string, operation string) {
	const permissionPrefix = "Claude needs your permission to use "

	if !strings.HasPrefix(message, permissionPrefix) {
		return false, "", "", ""
	}

	// Extract the tool/MCP part after the prefix
	toolPart := strings.TrimPrefix(message, permissionPrefix)

	// Check if it's an MCP operation (ends with "(MCP)")
	if strings.HasSuffix(toolPart, " (MCP)") {
		// Remove the " (MCP)" suffix
		toolPart = strings.TrimSuffix(toolPart, " (MCP)")

		// Split by " - " to get MCP name and operation
		parts := strings.Split(toolPart, " - ")
		if len(parts) == 2 {
			return true, "", parts[0], parts[1]
		}
	}

	// Regular tool use
	return true, toolPart, "", ""
}

// formatNotificationEvent formats and outputs a notification event
func (w *NotificationWatcher) formatNotificationEvent(event *NotificationEvent) {
	// Parse permission messages
	isPermission, toolName, mcpName, operation := w.parsePermissionMessage(event.Message)

	// Determine emoji based on message content
	emoji := "🔔"
	formattedMessage := event.Message
	displayToolName := ""

	if isPermission {
		emoji = "🔐"
		if mcpName != "" {
			// Format MCP tool name as mcp__{mcp_name}__{operation_name}
			displayToolName = fmt.Sprintf("mcp__%s__%s", mcpName, operation)
			formattedMessage = fmt.Sprintf("Permission request: Tool '%s' (MCP: %s - %s)", displayToolName, mcpName, operation)
		} else {
			// Regular tool permission
			displayToolName = toolName
			formattedMessage = fmt.Sprintf("Permission request: Tool '%s'", displayToolName)
		}
	} else if strings.Contains(event.Message, "waiting") {
		emoji = "⏳"
	} else if strings.Contains(event.Message, "error") || strings.Contains(event.Message, "failed") {
		emoji = "❌"
	} else if strings.Contains(event.Message, "success") || strings.Contains(event.Message, "completed") {
		emoji = "✅"
	}

	// Format the output
	output := fmt.Sprintf("\n[%s] %s %s", time.Now().Format("15:04:05"), emoji, event.HookEventName)

	// Add session info in debug mode
	if w.debugMode {
		output += fmt.Sprintf(" [Session: %s]", event.SessionID[:8])
	}

	output += fmt.Sprintf(": %s", formattedMessage)

	// Add debug info if enabled
	if w.debugMode {
		output += fmt.Sprintf("\n  [DEBUG] Original: %s", event.Message)
		output += fmt.Sprintf("\n  [DEBUG] CWD: %s", event.CWD)
		output += fmt.Sprintf("\n  [DEBUG] Transcript: %s", event.TranscriptPath)
	}

	fmt.Print(output)

	// Use narrator if available for tool permissions
	if w.narrator != nil && isPermission && displayToolName != "" {
		// Use NarrateToolUsePermission for permission requests
		narration := w.narrator.NarrateToolUsePermission(displayToolName)
		if narration != "" {
			fmt.Printf("\n  💬 %s", narration)
		}
	} else if w.narrator != nil && event.Message != "" {
		// Use NarrateText for other notifications
		narration := w.narrator.NarrateText(event.Message)
		if narration != "" {
			fmt.Printf("\n  💬 %s", narration)
		}
	}
}
