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
	var output string
	switch evt := e.(type) {
	case *event.NotificationEvent:
		output = p.formatNotificationEvent(evt)
	case *event.SystemMessage:
		output = p.formatSystemMessage(evt)
	case *event.UserMessage:
		output = p.formatUserMessage(evt)
	case *event.SummaryEvent:
		output = p.formatSummaryEvent(evt)
	case *event.TaskCompletionMessage:
		output = p.formatTaskCompletionMessage(evt)
	default:
		// Unknown event types are silently ignored
		return
	}

	if output != "" {
		fmt.Fprint(p.writer, output)
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
	if logger.IsDebugMode() && len(event.Session.SessionID) > 0 {
		sessionPrefix := event.Session.SessionID
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
	if logger.IsDebugMode() && len(event.Session.SessionID) > 0 {
		sessionPrefix := event.Session.SessionID
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
	if logger.IsDebugMode() && len(event.Session.SessionID) > 0 {
		sessionPrefix := event.Session.SessionID
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

// formatSystemMessage formats a system message event
func (p *NotificationPrinter) formatSystemMessage(msg *event.SystemMessage) string {
	if msg.SessionMessageBase.IsMeta && !logger.IsDebugMode() {
		return "" // Skip meta messages unless in debug mode
	}

	// Check if this is a HookEvent (HookSystemMessageContent)
	switch content := msg.Content.(type) {
	case *event.HookSystemMessageContent:
		return p.formatHookSystemMessage(msg, *content)
	}

	var output strings.Builder

	levelStr := ""
	if msg.Level != "" {
		levelStr = fmt.Sprintf(" [%s]", msg.Level)
	}

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 📣 SYSTEM%s", p.timeFunc().Format("15:04:05"), levelStr)
	if logger.IsDebugMode() {
		debugInfo := fmt.Sprintf(" [UUID: %s", msg.SessionMessageBase.UUID)
		if msg.SessionMessageBase.IsMeta {
			debugInfo += ", META"
		}
		if msg.ToolUseID != "" {
			debugInfo += fmt.Sprintf(", Tool: %s", msg.ToolUseID)
		}
		debugInfo += "]"
		header += debugInfo
	}
	header += ":\n"

	// Get level emoji for content
	contentEmoji := ""
	switch msg.Level {
	case "error":
		contentEmoji = "❌ "
	case "warning":
		contentEmoji = "⚠️ "
	case "info":
		contentEmoji = "ℹ️ "
	case "debug":
		contentEmoji = "🐛 "
	}

	// Build message with content on new line
	output.WriteString(header)
	output.WriteString(fmt.Sprintf("  %s%s\n", contentEmoji, msg.RawContent))

	// Add narration if available
	if msg.Narration != nil && msg.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", msg.Narration.Text))
	}

	return output.String()
}

// formatHookSystemMessage formats a HookEvent presented as SystemMessage
func (p *NotificationPrinter) formatHookSystemMessage(msg *event.SystemMessage, hookContent event.HookSystemMessageContent) string {
	// Only display in debug mode
	if !logger.IsDebugMode() {
		return ""
	}

	var output strings.Builder

	// Build header
	header := fmt.Sprintf("[%s] 🪝 HOOK [%s]", p.timeFunc().Format("15:04:05"), hookContent.Type)
	debugInfo := fmt.Sprintf(" [UUID: %s", msg.SessionMessageBase.UUID)
	if msg.ToolUseID != "" {
		debugInfo += fmt.Sprintf(", Tool: %s", msg.ToolUseID)
	}
	debugInfo += "]"
	header += debugInfo
	output.WriteString(header + "\n")

	// Show hook details
	if hookContent.Command != "" {
		output.WriteString(fmt.Sprintf("  📟 Command: %s\n", hookContent.Command))
	}
	if hookContent.Status != "" {
		output.WriteString(fmt.Sprintf("  ✅ Status: %s\n", hookContent.Status))
	}
	if hookContent.Message != "" {
		output.WriteString(fmt.Sprintf("  💬 Message: %s\n", hookContent.Message))
	}

	// Add debug info
	output.WriteString(fmt.Sprintf("  🏷️  Level: %s\n", msg.Level))
	if msg.SessionMessageBase.CWD != "" {
		output.WriteString(fmt.Sprintf("  📂 CWD: %s\n", msg.SessionMessageBase.CWD))
	}

	return output.String()
}

// formatSummaryEvent formats a summary event
func (p *NotificationPrinter) formatSummaryEvent(event *event.SummaryEvent) string {
	var output strings.Builder

	// Build message with optional debug info
	message := fmt.Sprintf("📋 [SUMMARY] %s", event.Summary)
	if logger.IsDebugMode() {
		message += fmt.Sprintf(" [LeafUUID: %s]", event.LeafUUID)
	}
	output.WriteString(message + "\n")

	// Add narration if available
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	}

	return output.String()
}

// formatUserMessage formats a user message event
func (p *NotificationPrinter) formatUserMessage(msg *event.UserMessage) string {
	// Skip meta messages unless in debug mode
	if msg.SessionMessageBase.IsMeta && !logger.IsDebugMode() {
		return ""
	}

	var output strings.Builder

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 👤 USER:", p.timeFunc().Format("15:04:05"))
	if msg.SessionMessageBase.IsMeta {
		header += " [META]"
	}
	if logger.IsDebugMode() {
		header += fmt.Sprintf(" [UUID: %s]", msg.SessionMessageBase.UUID)
	}
	output.WriteString(header + "\n")

	// Format content based on type
	switch content := msg.Message.Content.(type) {
	case *event.UserMessageContentMessage:
		output.WriteString(p.formatUserMessageContentMessage(content))
	case *event.UserMessageContentCommand:
		output.WriteString(p.formatUserMessageContentCommand(content))
	case *event.UserMessageContentLocalCommand:
		output.WriteString(p.formatUserMessageContentLocalCommand(content))
	case *event.UserMessageContentList:
		output.WriteString(p.formatUserMessageContentList(content))
	case *event.UserMessageContentToolResult:
		output.WriteString(p.formatUserMessageContentToolResult(content))
	case *event.UserMessageContentUnknown:
		output.WriteString(p.formatUserMessageContentUnknown(content))
	default:
		// Fallback for unknown content types
		output.WriteString(fmt.Sprintf("  %v\n", msg.Message.Content))
		if logger.IsDebugMode() {
			output.WriteString(fmt.Sprintf("  [DEBUG] Unknown content type: %T\n", msg.Message.Content))
		}
	}

	// Add narration if available
	if msg.Narration != nil && msg.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", msg.Narration.Text))
	}

	// Ensure message ends with newline
	result := output.String()
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

// formatUserMessageContentMessage formats UserMessageContentMessage
func (p *NotificationPrinter) formatUserMessageContentMessage(content *event.UserMessageContentMessage) string {
	var output strings.Builder

	// Truncate long messages
	lines := strings.Split(strings.TrimSpace(content.Text), "\n")
	for i, line := range lines {
		if i < 3 {
			if i == 0 {
				output.WriteString(fmt.Sprintf("  💬 %s\n", line))
			} else {
				output.WriteString(fmt.Sprintf("  %s\n", line))
			}
		} else if i == 3 && len(lines) > 4 {
			output.WriteString(fmt.Sprintf("  ... (%d more lines)\n", len(lines)-3))
			break
		}
	}
	// Add full content in debug mode
	if logger.IsDebugMode() && len(lines) > 3 {
		output.WriteString(fmt.Sprintf("  [DEBUG] Full content: %d lines, %d chars\n", len(lines), len(content.Text)))
	}

	return output.String()
}

// formatUserMessageContentCommand formats UserMessageContentCommand
func (p *NotificationPrinter) formatUserMessageContentCommand(content *event.UserMessageContentCommand) string {
	var output strings.Builder

	output.WriteString(fmt.Sprintf("  🎯 Command: %s\n", content.CommandName))
	if content.CommandMessage != "" {
		output.WriteString(fmt.Sprintf("  📝 Message: %s\n", content.CommandMessage))
	}
	if content.CommandArgs != "" {
		output.WriteString(fmt.Sprintf("  📦 Args: %s\n", content.CommandArgs))
	}

	return output.String()
}

// formatUserMessageContentLocalCommand formats UserMessageContentLocalCommand
func (p *NotificationPrinter) formatUserMessageContentLocalCommand(content *event.UserMessageContentLocalCommand) string {
	var output strings.Builder

	if content.Output == "" {
		output.WriteString("  📤 Command output: (no content)\n")
	} else {
		// Truncate long output
		lines := strings.Split(strings.TrimSpace(content.Output), "\n")
		output.WriteString("  📤 Command output:\n")
		for i, line := range lines {
			if i < 5 {
				output.WriteString(fmt.Sprintf("    %s\n", line))
			} else if i == 5 && len(lines) > 6 {
				output.WriteString(fmt.Sprintf("    ... (%d more lines)\n", len(lines)-5))
				break
			}
		}
	}

	return output.String()
}

// formatUserMessageContentUnknown formats UserMessageContentUnknown
func (p *NotificationPrinter) formatUserMessageContentUnknown(content *event.UserMessageContentUnknown) string {
	var output strings.Builder

	if content.Type != "" {
		output.WriteString(fmt.Sprintf("  ❓ Unknown Content Type: %s\n", content.Type))
	} else {
		output.WriteString("  ❓ Unknown Content (no type field)\n")
	}

	// Show a preview of the raw data in debug mode
	if logger.IsDebugMode() && len(content.Data) > 0 {
		preview := string(content.Data)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		output.WriteString(fmt.Sprintf("  [DEBUG] Raw data: %s\n", preview))
	}

	return output.String()
}

// formatUserMessageContentToolResult formats UserMessageContentToolResult
func (p *NotificationPrinter) formatUserMessageContentToolResult(content *event.UserMessageContentToolResult) string {
	var output strings.Builder

	if content.IsError {
		output.WriteString(fmt.Sprintf("  🛠️❌ Tool Result (Error)\n"))
		output.WriteString(fmt.Sprintf("  Tool ID: %s\n", content.ToolUseID))
		// Show error content
		lines := strings.Split(strings.TrimSpace(content.Content), "\n")
		for i, line := range lines {
			if i < 3 {
				output.WriteString(fmt.Sprintf("  %s\n", line))
			} else if i == 3 && len(lines) > 4 {
				output.WriteString(fmt.Sprintf("  ... (%d more lines)\n", len(lines)-3))
				break
			}
		}
	} else {
		output.WriteString(fmt.Sprintf("  🛠️✅ Tool Result\n"))
		output.WriteString(fmt.Sprintf("  Tool ID: %s\n", content.ToolUseID))
		// Show result content (truncated)
		lines := strings.Split(strings.TrimSpace(content.Content), "\n")
		for i, line := range lines {
			if i < 3 {
				output.WriteString(fmt.Sprintf("  %s\n", line))
			} else if i == 3 && len(lines) > 4 {
				output.WriteString(fmt.Sprintf("  ... (%d more lines)\n", len(lines)-3))
				break
			}
		}
	}

	// Add full content in debug mode
	if logger.IsDebugMode() {
		lines := strings.Split(content.Content, "\n")
		output.WriteString(fmt.Sprintf("  [DEBUG] Full content: %d lines, %d chars\n", len(lines), len(content.Content)))
	}

	return output.String()
}

// formatUserMessageContentList formats UserMessageContentList
func (p *NotificationPrinter) formatUserMessageContentList(content *event.UserMessageContentList) string {
	var output strings.Builder

	if len(content.Items) == 0 {
		output.WriteString("  📋 Empty list\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("  📋 List (%d items):\n", len(content.Items)))

	for i, item := range content.Items {
		output.WriteString(fmt.Sprintf("  [%d] ", i+1))

		// Format each item based on its type
		switch c := item.(type) {
		case *event.UserMessageContentMessage:
			// Show only first line for list items
			lines := strings.Split(strings.TrimSpace(c.Text), "\n")
			if len(lines) > 0 {
				firstLine := lines[0]
				if len(firstLine) > 50 {
					firstLine = firstLine[:50] + "..."
				}
				output.WriteString(fmt.Sprintf("💬 %s\n", firstLine))
			}
		case *event.UserMessageContentCommand:
			output.WriteString(fmt.Sprintf("🎯 Command: %s\n", c.CommandName))
		case *event.UserMessageContentLocalCommand:
			output.WriteString("📤 Command output\n")
		case *event.UserMessageContentInterrupted:
			output.WriteString(fmt.Sprintf("⛔ %s\n", c.Reason))
		case *event.UserMessageContentToolResult:
			if c.IsError {
				output.WriteString(fmt.Sprintf("🛠️❌ Tool error: %s\n", c.ToolUseID))
			} else {
				output.WriteString(fmt.Sprintf("🛠️✅ Tool result: %s\n", c.ToolUseID))
			}
		case *event.UserMessageContentUnknown:
			if c.Type != "" {
				output.WriteString(fmt.Sprintf("❓ Unknown type: %s\n", c.Type))
			} else {
				output.WriteString("❓ Unknown content (no type)\n")
			}
		default:
			output.WriteString(fmt.Sprintf("Unknown type: %T\n", item))
		}

		// Limit to showing first 3 items
		if i >= 2 && len(content.Items) > 3 {
			output.WriteString(fmt.Sprintf("  ... (%d more items)\n", len(content.Items)-3))
			break
		}
	}

	return output.String()
}

// formatTaskCompletionMessage formats a task completion message
func (p *NotificationPrinter) formatTaskCompletionMessage(event *event.TaskCompletionMessage) string {
	var output strings.Builder

	// Build header with timestamp
	timestamp := event.Timestamp.Format("15:04:05")
	header := fmt.Sprintf("[%s] ✨ Task Completed: %s", timestamp, event.TaskInfo.Description)

	// Add debug info if enabled
	if logger.IsDebugMode() {
		header += fmt.Sprintf(" [Session: %s, ToolUse: %s]", event.Session.SessionID, event.TaskInfo.ToolUseID)
	}
	output.WriteString(header + "\n")

	// Add agent type if available
	if event.TaskInfo.SubagentType != "" {
		output.WriteString(fmt.Sprintf("  Agent: %s\n", event.TaskInfo.SubagentType))
	}

	// Add narration if available
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	}

	return output.String()
}
