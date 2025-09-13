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
	case *event.AssistantMessage:
		output = p.formatAssistantMessage(evt)
	case *event.ResumeEvent:
		output = p.formatResumeEvent(evt)
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

	// Display narration if available, otherwise display message
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	} else {
		// Only show message if no narration
		if displayToolName != "" {
			output.WriteString(fmt.Sprintf("  %s: %s\n", formattedMessage, displayToolName))
		} else {
			output.WriteString(fmt.Sprintf("  %s\n", event.Message))
		}
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

	// Add subagent info if available
	if event.SubagentTask != nil {
		output.WriteString(fmt.Sprintf("  Subagent Events: %d (UUID: %s)\n",
			event.SubagentTask.EventCount, event.SubagentTask.UUID))
	}

	// Add narration if available
	if event.Narration != nil && event.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", event.Narration.Text))
	}

	return output.String()
}

// formatAssistantMessage formats an assistant message event
func (p *NotificationPrinter) formatAssistantMessage(msg *event.AssistantMessage) string {
	var output strings.Builder

	// Build header with timestamp and model
	timestamp := msg.Timestamp.Format("15:04:05")
	header := fmt.Sprintf("[%s] 🤖 ASSISTANT (%s):", timestamp, msg.Message.Model)

	// Add debug info if enabled
	if logger.IsDebugMode() {
		header += fmt.Sprintf(" [ID: %s, ReqID: %s]", msg.Message.ID, msg.RequestID)
		if msg.Message.StopReason != nil {
			header += fmt.Sprintf(" [Stop: %s]", *msg.Message.StopReason)
		}
	}
	output.WriteString(header + "\n")

	// Handle content based on type
	switch content := msg.Message.Content.(type) {
	case *event.APIErrorMessageContent:
		// Format API error - use narration if available, otherwise formatted error
		if msg.Narration != nil && msg.Narration.Text != "" {
			// Narration already generated by central handler
			output.WriteString(fmt.Sprintf("  ❌ %s\n", msg.Narration.Text))
		} else if content.StatusCode > 0 || content.Error.Type != "" {
			// Fallback to formatted error
			output.WriteString(fmt.Sprintf("  ❌ API Error %d: %s - %s\n",
				content.StatusCode, content.Error.Type, content.Error.Message))
		} else if content.RawText != "" {
			// Fallback to raw text
			output.WriteString(fmt.Sprintf("  ❌ %s\n", content.RawText))
		}

		// Return early since we already added narration above
		return output.String()

	case *event.AssistantMessageContentText:
		// Format single text content
		output.WriteString(p.formatAssistantTextContent(content))

	case *event.AssistantMessageContentList:
		// Format list of content items
		for _, item := range content.Items {
			switch c := item.(type) {
			case *event.AssistantMessageContentText:
				output.WriteString(p.formatAssistantTextContent(c))
			case *event.AssistantMessageContentToolUse:
				output.WriteString(p.formatAssistantToolUseContent(c))
			default:
				// Unknown content type
				if logger.IsDebugMode() {
					output.WriteString(fmt.Sprintf("  [DEBUG] Unknown content type: %T\n", c))
				}
			}
		}

	default:
		// For now, just indicate unsupported content type
		output.WriteString(fmt.Sprintf("  [Unsupported content type: %T]\n", content))
	}

	// Add narration if available
	if msg.Narration != nil && msg.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", msg.Narration.Text))
	}

	// Add token usage if available and in debug mode
	if logger.IsDebugMode() && msg.Message.Usage != nil {
		output.WriteString(fmt.Sprintf("  📊 Tokens - Input: %d, Output: %d\n",
			msg.Message.Usage.InputTokens, msg.Message.Usage.OutputTokens))
		if msg.Message.Usage.CacheReadInputTokens > 0 {
			output.WriteString(fmt.Sprintf("  📊 Cache Read: %d tokens\n", msg.Message.Usage.CacheReadInputTokens))
		}
		if msg.Message.Usage.CacheCreationInputTokens > 0 {
			output.WriteString(fmt.Sprintf("  📊 Cache Creation: %d tokens\n", msg.Message.Usage.CacheCreationInputTokens))
		}
	}

	return output.String()
}

// formatAssistantToolUseContent formats tool use content from assistant message
func (p *NotificationPrinter) formatAssistantToolUseContent(content *event.AssistantMessageContentToolUse) string {
	var output strings.Builder

	// Display narration if available
	if content.Narration != nil && content.Narration.Text != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", content.Narration.Text))

		// Special handling for specific tools even when narrator is used
		switch content.Name {
		case "TodoWrite":
			if todoInput, ok := content.Input.(*event.ToolUseTodoWrite); ok {
				for i, todo := range todoInput.Todos {
					emoji := ""
					switch todo.Status {
					case "completed":
						emoji = "✅"
					case "in_progress":
						emoji = "🔄"
					case "pending":
						emoji = "⏳"
					}
					output.WriteString(fmt.Sprintf("    %d. %s %s\n", i+1, emoji, todo.Content))
				}
			}
		case "Bash":
			// Show command after narration for Bash
			if bashInput, ok := content.Input.(*event.ToolUseBash); ok {
				output.WriteString(fmt.Sprintf("  $ %s\n", bashInput.Command))
			}
		default:
			// For MCP tools, display the tool name after narration
			if strings.HasPrefix(content.Name, "mcp__") {
				output.WriteString(fmt.Sprintf("  🔧 %s\n", content.Name))
			}
		}
	} else {
		// Fallback: display tool name and basic info
		output.WriteString(fmt.Sprintf("  🛠️ Tool Use: %s\n", content.Name))

		// Display tool-specific details based on type
		switch toolInput := content.Input.(type) {
		case *event.ToolUseTodoWrite:
			for i, todo := range toolInput.Todos {
				emoji := ""
				switch todo.Status {
				case "completed":
					emoji = "✅"
				case "in_progress":
					emoji = "🔄"
				case "pending":
					emoji = "⏳"
				}
				output.WriteString(fmt.Sprintf("    %d. %s %s\n", i+1, emoji, todo.Content))
			}
		case *event.ToolUseBash:
			output.WriteString(fmt.Sprintf("    Command: %s\n", toolInput.Command))
			if toolInput.Description != "" {
				output.WriteString(fmt.Sprintf("    Description: %s\n", toolInput.Description))
			}
		case *event.ToolUseRead:
			output.WriteString(fmt.Sprintf("    File: %s\n", toolInput.FilePath))
			if toolInput.Limit > 0 {
				output.WriteString(fmt.Sprintf("    Limit: %d lines\n", toolInput.Limit))
			}
		case *event.ToolUseWrite:
			output.WriteString(fmt.Sprintf("    File: %s\n", toolInput.FilePath))
			lines := strings.Count(toolInput.Content, "\n") + 1
			output.WriteString(fmt.Sprintf("    Content: %d lines\n", lines))
		case *event.ToolUseEdit:
			output.WriteString(fmt.Sprintf("    File: %s\n", toolInput.FilePath))
			if toolInput.ReplaceAll {
				output.WriteString("    Mode: Replace all occurrences\n")
			}
		case *event.ToolUseMultiEdit:
			output.WriteString(fmt.Sprintf("    File: %s\n", toolInput.FilePath))
			output.WriteString(fmt.Sprintf("    Edits: %d changes\n", len(toolInput.Edits)))
		case *event.ToolUseGrep:
			output.WriteString(fmt.Sprintf("    Pattern: %s\n", toolInput.Pattern))
			if toolInput.Path != "" {
				output.WriteString(fmt.Sprintf("    Path: %s\n", toolInput.Path))
			}
		case *event.ToolUseGlob:
			output.WriteString(fmt.Sprintf("    Pattern: %s\n", toolInput.Pattern))
			if toolInput.Path != "" {
				output.WriteString(fmt.Sprintf("    Path: %s\n", toolInput.Path))
			}
		case *event.ToolUseTask:
			output.WriteString(fmt.Sprintf("    Description: %s\n", toolInput.Description))
			output.WriteString(fmt.Sprintf("    Agent: %s\n", toolInput.SubagentType))
		case *event.ToolUseWebFetch:
			output.WriteString(fmt.Sprintf("    URL: %s\n", toolInput.URL))
		case *event.ToolUseWebSearch:
			output.WriteString(fmt.Sprintf("    Query: %s\n", toolInput.Query))
		case *event.ToolUseMCP:
			output.WriteString(fmt.Sprintf("    Server: %s\n", toolInput.Server))
			output.WriteString(fmt.Sprintf("    Tool: %s\n", toolInput.Tool))
		}
	}

	return output.String()
}

// formatAssistantTextContent formats text content from assistant message
func (p *NotificationPrinter) formatAssistantTextContent(content *event.AssistantMessageContentText) string {
	var output strings.Builder

	// Prepare processed text for display
	processedText := strings.TrimSpace(content.Text)
	if len(content.CodeBlocks) > 0 {
		// Replace code blocks with placeholders
		for i, block := range content.CodeBlocks {
			placeholder := fmt.Sprintf("[CODE BLOCK %d: %s]", i+1, block.Language)
			// Find and replace the original code block
			original := fmt.Sprintf("```%s\n%s```", block.Language, block.Content)
			if block.Language == "text" || block.Language == "" {
				original = fmt.Sprintf("```\n%s```", block.Content)
			}
			processedText = strings.Replace(processedText, original, placeholder, 1)
		}
	}

	// Display narration if available
	if content.Narration != nil && content.Narration.Text != "" {
		if content.IsThinking {
			output.WriteString(fmt.Sprintf("  🤔 %s\n", content.Narration.Text))
		} else {
			output.WriteString(fmt.Sprintf("  💬 %s\n", content.Narration.Text))
		}
	}

	// Show the main text (only if multiple lines)
	lines := strings.Split(strings.TrimSpace(processedText), "\n")

	// Filter out code block placeholders if any
	var displayLines []string
	if len(content.CodeBlocks) > 0 {
		for _, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "[CODE BLOCK") || !strings.HasSuffix(strings.TrimSpace(line), "]") {
				displayLines = append(displayLines, line)
			}
		}
	} else {
		displayLines = lines
	}

	// Display text lines with 📝 or 🤔 emoji (only if multiple lines or if no narration)
	if len(displayLines) > 1 || (content.Narration == nil || content.Narration.Text == "") {
		maxLines := 5
		for i, line := range displayLines {
			if i < maxLines {
				if i == 0 {
					if content.IsThinking {
						output.WriteString(fmt.Sprintf("  🤔 %s\n", line))
					} else {
						output.WriteString(fmt.Sprintf("  📝 %s\n", line))
					}
				} else {
					output.WriteString(fmt.Sprintf("  %s\n", line))
				}
			} else if i == maxLines && len(displayLines) > maxLines+1 {
				output.WriteString(fmt.Sprintf("  ... (%d more lines)\n", len(displayLines)-maxLines))
				break
			}
		}
	}

	// Display code blocks if any
	if len(content.CodeBlocks) > 0 {
		for i, block := range content.CodeBlocks {
			if len(displayLines) > 0 || i > 0 {
				output.WriteString("\n")
			}
			language := block.Language
			if language == "" {
				language = "text"
			}
			output.WriteString(fmt.Sprintf("  📝 Code Block %d (%s):\n", i+1, language))
			output.WriteString("    ```\n")

			// Display code content with indentation
			lines := strings.Split(strings.TrimRight(block.Content, "\n"), "\n")
			maxLines := 10
			for j, line := range lines {
				if j < maxLines {
					output.WriteString(fmt.Sprintf("    %s\n", line))
				} else if j == maxLines && len(lines) > maxLines+1 {
					output.WriteString(fmt.Sprintf("    ... (%d more lines)\n", len(lines)-maxLines))
					break
				}
			}
			output.WriteString("    ```\n")
		}
	}

	return output.String()
}

// formatResumeEvent formats a resume event
func (p *NotificationPrinter) formatResumeEvent(event *event.ResumeEvent) string {
	var output strings.Builder

	// Build header with timestamp
	timestamp := event.Timestamp.Format("15:04:05")
	header := fmt.Sprintf("[%s] 🔄 RESUME", timestamp)

	// Add debug info if enabled
	if logger.IsDebugMode() {
		sessionPrefix := event.Session.SessionID
		if len(sessionPrefix) > 8 {
			sessionPrefix = sessionPrefix[:8]
		}
		header += fmt.Sprintf(" [Session: %s]", sessionPrefix)
		if event.ResumedFromID != "" {
			resumedPrefix := event.ResumedFromID
			if len(resumedPrefix) > 8 {
				resumedPrefix = resumedPrefix[:8]
			}
			header += fmt.Sprintf(" [From: %s]", resumedPrefix)
		}
	}
	output.WriteString(header + "\n")

	// Display buffered events count
	if event.BufferedCount > 0 {
		output.WriteString(fmt.Sprintf("  📦 Released %d buffered events\n", event.BufferedCount))
	} else {
		output.WriteString("  📦 Resume completed (no buffered events)\n")
	}

	// Display reason if available
	if event.Reason != "" {
		output.WriteString(fmt.Sprintf("  💭 Reason: %s\n", event.Reason))
	}

	return output.String()
}
