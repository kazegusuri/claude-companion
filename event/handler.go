package event

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/narrator"
)

// Handler processes events from multiple sources
type Handler struct {
	narrator  narrator.Narrator
	parser    *Parser
	debugMode bool
	eventChan chan Event
	wg        sync.WaitGroup
	done      chan struct{}
}

// NewHandler creates a new event handler
func NewHandler(narrator narrator.Narrator, debugMode bool) *Handler {
	parser := NewParser(narrator)
	parser.SetDebugMode(debugMode)

	return &Handler{
		narrator:  narrator,
		parser:    parser,
		debugMode: debugMode,
		eventChan: make(chan Event, 100),
		done:      make(chan struct{}),
	}
}

// Start begins processing events
func (h *Handler) Start() {
	h.wg.Add(1)
	go h.processEvents()
}

// Stop stops the event handler
func (h *Handler) Stop() {
	close(h.done)
	close(h.eventChan)
	h.wg.Wait()
}

// SendEvent sends an event to be processed
func (h *Handler) SendEvent(event Event) {
	select {
	case h.eventChan <- event:
	case <-h.done:
		// Handler is stopping, discard event
	}
}

// processEvents processes events from the channel
func (h *Handler) processEvents() {
	defer h.wg.Done()

	for {
		select {
		case event, ok := <-h.eventChan:
			if !ok {
				return
			}
			if err := event.Process(h); err != nil {
				if h.debugMode {
					log.Printf("Error processing %s event: %v", event.Type(), err)
				}
			}
		case <-h.done:
			// Drain remaining events
			for {
				select {
				case event, ok := <-h.eventChan:
					if !ok {
						return
					}
					event.Process(h)
				default:
					return
				}
			}
		}
	}
}

// formatNotificationEvent formats and outputs a notification event (moved from notification_watcher.go)
func (h *Handler) formatNotificationEvent(event *NotificationEvent) {
	// Handle PreCompact event
	if event.HookEventName == "PreCompact" {
		emoji := "🗜️"

		// Use narrator to get the narration message
		var formattedMessage string
		if h.narrator != nil {
			formattedMessage = h.narrator.NarrateNotification(narrator.NotificationTypeCompact)
		} else {
			formattedMessage = "コンテキストを圧縮しています"
		}

		// Format the output
		output := fmt.Sprintf("\n[%s] %s %s", timeNow().Format("15:04:05"), emoji, event.HookEventName)

		// Add session info in debug mode
		if h.debugMode && len(event.SessionID) >= 8 {
			output += fmt.Sprintf(" [Session: %s]", event.SessionID[:8])
		}

		output += fmt.Sprintf(": %s", formattedMessage)

		// Add debug info if enabled
		if h.debugMode {
			output += fmt.Sprintf("\n  [DEBUG] Trigger: %s", event.Trigger)
			output += fmt.Sprintf("\n  [DEBUG] CWD: %s", event.CWD)
			output += fmt.Sprintf("\n  [DEBUG] Transcript: %s", event.TranscriptPath)
		}

		fmt.Print(output)

		// Show narrator emoji
		if h.narrator != nil && formattedMessage != "" {
			fmt.Printf("\n  💬 %s", formattedMessage)
		}

		return
	}

	// Parse permission messages
	isPermission, toolName, mcpName, operation := h.parsePermissionMessage(event.Message)

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
	} else if containsAny(event.Message, "waiting") {
		emoji = "⏳"
	} else if containsAny(event.Message, "error", "failed") {
		emoji = "❌"
	} else if containsAny(event.Message, "success", "completed") {
		emoji = "✅"
	}

	// Format the output
	output := fmt.Sprintf("\n[%s] %s %s", timeNow().Format("15:04:05"), emoji, event.HookEventName)

	// Add session info in debug mode
	if h.debugMode && len(event.SessionID) >= 8 {
		output += fmt.Sprintf(" [Session: %s]", event.SessionID[:8])
	}

	output += fmt.Sprintf(": %s", formattedMessage)

	// Add debug info if enabled
	if h.debugMode {
		output += fmt.Sprintf("\n  [DEBUG] Original: %s", event.Message)
		output += fmt.Sprintf("\n  [DEBUG] CWD: %s", event.CWD)
		output += fmt.Sprintf("\n  [DEBUG] Transcript: %s", event.TranscriptPath)
	}

	fmt.Print(output)

	// Use narrator if available for tool permissions
	if h.narrator != nil && isPermission && displayToolName != "" {
		// Use NarrateToolUsePermission for permission requests
		narration := h.narrator.NarrateToolUsePermission(displayToolName)
		if narration != "" {
			fmt.Printf("\n  💬 %s", narration)
		}
	} else if h.narrator != nil && event.Message != "" {
		// Use NarrateText for other notifications
		narration := h.narrator.NarrateText(event.Message)
		if narration != "" {
			fmt.Printf("\n  💬 %s", narration)
		}
	}
}

// parsePermissionMessage parses permission messages to extract tool/MCP information
func (h *Handler) parsePermissionMessage(message string) (isPermission bool, toolName string, mcpName string, operation string) {
	const permissionPrefix = "Claude needs your permission to use "

	if !hasPrefix(message, permissionPrefix) {
		return false, "", "", ""
	}

	// Extract the tool/MCP part after the prefix
	toolPart := trimPrefix(message, permissionPrefix)

	// Check if it's an MCP operation (ends with "(MCP)")
	if hasSuffix(toolPart, " (MCP)") {
		// Remove the " (MCP)" suffix
		toolPart = trimSuffix(toolPart, " (MCP)")

		// Split by " - " to get MCP name and operation
		parts := splitN(toolPart, " - ", 2)
		if len(parts) == 2 {
			return true, "", parts[0], parts[1]
		}
	}

	// Regular tool use
	return true, toolPart, "", ""
}

// Helper functions to avoid importing strings package in this file
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func trimPrefix(s, prefix string) string {
	if hasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func trimSuffix(s, suffix string) string {
	if hasSuffix(s, suffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func splitN(s, sep string, n int) []string {
	if n == 0 {
		return nil
	}
	if n < 0 {
		n = countOccurrences(s, sep) + 1
	}

	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx == -1 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); {
		if s[i:i+len(substr)] == substr {
			count++
			i += len(substr)
		} else {
			i++
		}
	}
	return count
}

// timeNow is a helper function to get current time (for testing)
var timeNow = time.Now
