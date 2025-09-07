package print

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/logger"
)

// NotificationPrinter formats events for terminal output
type NotificationPrinter struct {
	timeFunc func() time.Time
	writer   io.Writer
}

// NewNotificationPrinter creates a new printer
func NewNotificationPrinter() *NotificationPrinter {
	return NewNotificationPrinterWithWriter(os.Stdout)
}

// NewNotificationPrinterWithWriter creates a new printer with a custom writer
func NewNotificationPrinterWithWriter(w io.Writer) *NotificationPrinter {
	return &NotificationPrinter{
		timeFunc: time.Now,
		writer:   w,
	}
}

// Print outputs an event to the writer
func (p *NotificationPrinter) Print(e interface{}) {
	switch evt := e.(type) {
	case *event.NotificationEvent:
		output := p.formatNotificationEvent(evt)
		if output != "" {
			fmt.Fprint(p.writer, output)
		}
	default:
		// Unknown event types are silently ignored
	}
}

// formatNotificationEvent formats a notification event
func (p *NotificationPrinter) formatNotificationEvent(event *event.NotificationEvent) string {
	var output strings.Builder

	// Handle events based on HookEventName
	switch event.HookEventName {
	case "PreCompact":
		output.WriteString(p.formatPreCompactEvent(event))
	case "SessionStart":
		output.WriteString(p.formatSessionStartEvent(event))
	case "Notification":
		output.WriteString(p.formatGeneralNotificationEvent(event))
	default:
		// Return empty string for unknown event types
		return ""
	}

	// Ensure message ends with newline
	result := output.String()
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}

	return result
}

// formatPreCompactEvent formats PreCompact events
func (p *NotificationPrinter) formatPreCompactEvent(event *event.NotificationEvent) string {
	var output strings.Builder
	emoji := "🗜️"

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] %s %s", p.timeFunc().Format("15:04:05"), emoji, event.HookEventName)
	if logger.IsDebugMode() && len(event.SessionID) > 0 {
		sessionPrefix := event.SessionID
		if len(sessionPrefix) > 8 {
			sessionPrefix = sessionPrefix[:8]
		}
		header += fmt.Sprintf(" [Session: %s]", sessionPrefix)
	}
	output.WriteString(header + "\n")

	// Add formatted message from narration
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	}

	return output.String()
}

// formatSessionStartEvent formats SessionStart events
func (p *NotificationPrinter) formatSessionStartEvent(event *event.NotificationEvent) string {
	var output strings.Builder
	emoji := "🚀"

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] %s %s", p.timeFunc().Format("15:04:05"), emoji, event.HookEventName)
	if event.Source != "" {
		header += fmt.Sprintf(":%s", event.Source)
	}
	if logger.IsDebugMode() && len(event.SessionID) > 0 {
		sessionPrefix := event.SessionID
		if len(sessionPrefix) > 8 {
			sessionPrefix = sessionPrefix[:8]
		}
		header += fmt.Sprintf(" [Session: %s]", sessionPrefix)
	}
	output.WriteString(header + "\n")

	// Add formatted message from narration
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	}

	return output.String()
}

// formatGeneralNotificationEvent formats general Notification events
func (p *NotificationPrinter) formatGeneralNotificationEvent(event *event.NotificationEvent) string {
	var output strings.Builder

	// Parse permission messages for emoji and formatting
	emoji, formattedMessage, displayToolName := p.parseNotificationMessage(event.Message)

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] %s %s", p.timeFunc().Format("15:04:05"), emoji, event.HookEventName)
	if logger.IsDebugMode() && len(event.SessionID) > 0 {
		sessionPrefix := event.SessionID
		if len(sessionPrefix) > 8 {
			sessionPrefix = sessionPrefix[:8]
		}
		header += fmt.Sprintf(" [Session: %s]", sessionPrefix)
	}
	output.WriteString(header + "\n")

	// Add message
	if displayToolName != "" {
		output.WriteString(fmt.Sprintf("  %s: %s\n", formattedMessage, displayToolName))
	} else {
		output.WriteString(fmt.Sprintf("  %s\n", event.Message))
	}

	// Add narration if available
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	}

	return output.String()
}

// parseNotificationMessage determines emoji and formatting for notification messages
func (p *NotificationPrinter) parseNotificationMessage(message string) (emoji string, formattedMessage string, displayToolName string) {
	emoji = "🔔"
	formattedMessage = message

	// Check for permission messages
	if idx := strings.Index(message, "Claude will use "); idx == 0 {
		remaining := message[16:] // Skip "Claude will use "
		if colonIdx := strings.Index(remaining, ": "); colonIdx > 0 {
			toolNamePart := remaining[:colonIdx]
			operation := strings.ToLower(remaining[colonIdx+2:])

			if operation == "approve" {
				emoji = "✅"
			} else {
				emoji = "❌"
			}
			formattedMessage = fmt.Sprintf("Permission %s", operation)

			// Check if it's an MCP tool
			if strings.HasPrefix(toolNamePart, "mcp__") {
				parts := strings.SplitN(toolNamePart, "__", 3)
				if len(parts) >= 3 {
					displayToolName = fmt.Sprintf("%s (MCP: %s)", parts[2], parts[1])
				} else {
					displayToolName = toolNamePart
				}
			} else {
				displayToolName = toolNamePart
			}
			return
		}
	}

	// Check for other permission-related messages
	if strings.Contains(message, "Permission") {
		emoji = "🔑"
	}

	return emoji, formattedMessage, ""
}
