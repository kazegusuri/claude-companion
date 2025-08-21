package event

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kazegusuri/claude-companion/handler"
	"github.com/kazegusuri/claude-companion/narrator"
)

// Formatter handles formatting of parsed events
type Formatter struct {
	narrator       narrator.Narrator
	debugMode      bool
	fileOperations []string
	currentTool    string
	emitter        handler.MessageEmitter
}

// NewFormatter creates a new Formatter instance
func NewFormatter(narrator narrator.Narrator) *Formatter {
	return &Formatter{
		narrator:       narrator,
		debugMode:      false,
		fileOperations: make([]string, 0),
	}
}

// SetMessageEmitter sets the message emitter for sending events
func (f *Formatter) SetMessageEmitter(emitter handler.MessageEmitter) {
	f.emitter = emitter
}

// SetDebugMode enables or disables debug mode
func (f *Formatter) SetDebugMode(enabled bool) {
	f.debugMode = enabled
}

// Format formats an event for display
func (f *Formatter) Format(event Event) (string, error) {
	switch e := event.(type) {
	case *UserMessage:
		return f.formatUserMessage(e)
	case *AssistantMessage:
		return f.formatAssistantMessage(e)
	case *SystemMessage:
		return f.formatSystemMessage(e)
	case *HookEvent:
		return f.formatHookEvent(e)
	case *SummaryEvent:
		return f.formatSummaryEvent(e)
	case *NotificationEvent:
		return f.formatNotificationEvent(e)
	case *TaskCompletionMessage:
		return f.formatTaskCompletionMessage(e)
	case *BaseEvent:
		return f.formatUnknownEvent(e)
	default:
		return "", fmt.Errorf("unknown event type: %T", event)
	}
}

func (f *Formatter) formatUserMessage(event *UserMessage) (string, error) {
	var output strings.Builder

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 👤 USER:", event.Timestamp.Format("15:04:05"))
	if f.debugMode {
		header += fmt.Sprintf(" [UUID: %s]", event.UUID)
	}
	output.WriteString(header + "\n")

	switch content := event.Message.Content.(type) {
	case string:
		// Truncate long messages
		lines := strings.Split(strings.TrimSpace(content), "\n")
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
		if f.debugMode && len(lines) > 3 {
			output.WriteString(fmt.Sprintf("  [DEBUG] Full content: %d lines, %d chars\n", len(lines), len(content)))
		}
	case []interface{}:
		for _, item := range content {
			if contentMap, ok := item.(map[string]interface{}); ok {
				if contentType, ok := contentMap["type"].(string); ok {
					switch contentType {
					case "text":
						if text, ok := contentMap["text"].(string); ok {
							// Check for special patterns
							if strings.Contains(text, "<command-name>") {
								output.WriteString("  🎯 Command execution\n")
							} else if strings.Contains(text, "<local-command-stdout>") {
								output.WriteString("  📤 Command output\n")
							} else {
								// Normal text - truncate if needed
								lines := strings.Split(strings.TrimSpace(text), "\n")
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
							}
						}
					case "tool_result":
						toolID := contentMap["tool_use_id"]
						// Check if it has error
						emoji := "✅"
						if isError, ok := contentMap["is_error"].(bool); ok && isError {
							emoji = "❌"
						}
						resultLine := fmt.Sprintf("  %s Tool Result: %v", emoji, toolID)
						output.WriteString(resultLine + "\n")
					}
				}
			}
		}
	default:
		output.WriteString(fmt.Sprintf("  %v\n", event.Message.Content))
		if f.debugMode {
			output.WriteString(fmt.Sprintf("  [DEBUG] Unknown content type: %T\n", event.Message.Content))
		}
	}

	// Ensure message ends with newline
	result := output.String()
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, nil
}

func (f *Formatter) formatAssistantMessage(event *AssistantMessage) (string, error) {
	var output strings.Builder

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 🤖 ASSISTANT (%s):", event.Timestamp.Format("15:04:05"), event.Message.Model)
	if f.debugMode {
		header += fmt.Sprintf(" [ID: %s, ReqID: %s]", event.Message.ID, event.RequestID)
		if event.Message.StopReason != nil {
			header += fmt.Sprintf(" [Stop: %s]", *event.Message.StopReason)
		}
	}
	output.WriteString(header + "\n")

	// Track if we have any content to show summary for
	hasContent := false

	for i := range event.Message.Content {
		content := &event.Message.Content[i]
		hasContent = true
		switch content.Type {
		case "text":
			// Create EventMeta for the assistant message
			meta := &narrator.EventMeta{
				EventID:   event.Message.ID,
				SessionID: event.SessionID,
				CWD:       event.CWD,
				Timestamp: event.Timestamp,
			}
			formatted := f.FormatAssistantText(content.Text, false, meta)
			output.WriteString(formatted)
		case "thinking":
			// Create EventMeta for the thinking content
			meta := &narrator.EventMeta{
				EventID:   event.Message.ID,
				SessionID: event.SessionID,
				CWD:       event.CWD,
				Timestamp: event.Timestamp,
			}
			formatted := f.FormatAssistantText(content.Thinking, true, meta)
			output.WriteString(formatted)
		case "tool_use":
			// Convert input to map[string]interface{} for formatter
			inputMap := make(map[string]interface{})
			if content.Input != nil {
				if m, ok := content.Input.(map[string]interface{}); ok {
					inputMap = m
				} else {
					// Try to convert via JSON marshaling
					data, _ := json.Marshal(content.Input)
					json.Unmarshal(data, &inputMap)
				}
			}
			// Create EventMeta with tool ID and CWD
			meta := NewEventMeta(content.ID, event.CWD)
			formatted := f.FormatToolUse(content.Name, meta, inputMap)
			output.WriteString(formatted)
			// Add debug info showing tool use details
			if f.debugMode {
				output.WriteString(fmt.Sprintf("  [DEBUG] Tool Use: %s (id: %s)\n", content.Name, content.ID))
				if content.Input != nil {
					inputJSON, _ := json.MarshalIndent(content.Input, "    ", "  ")
					output.WriteString(fmt.Sprintf("    Input: %s\n", string(inputJSON)))
				}
			}
		}
	}

	// Show file operations summary first if we had any content
	if hasContent {
		summary := f.GetFileSummary()
		if summary != "" {
			output.WriteString(summary)
		}
		// Reset for next message
		f.Reset()
	}

	// Add token usage at the end if present
	if event.Message.Usage.OutputTokens > 0 {
		output.WriteString(fmt.Sprintf("  💰 Tokens: input=%d, output=%d, cache_read=%d, cache_creation=%d\n",
			event.Message.Usage.InputTokens,
			event.Message.Usage.OutputTokens,
			event.Message.Usage.CacheReadInputTokens,
			event.Message.Usage.CacheCreationInputTokens))
	}

	// Ensure message ends with newline
	result := output.String()
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, nil
}

func (f *Formatter) formatHookEvent(event *HookEvent) (string, error) {
	if event.IsMeta && !f.debugMode {
		return "", nil // Skip meta messages unless in debug mode
	}

	var output strings.Builder

	// Build header
	header := fmt.Sprintf("[%s] 🪝 HOOK [%s]", event.Timestamp.Format("15:04:05"), event.HookEventType)
	if f.debugMode {
		debugInfo := fmt.Sprintf(" [UUID: %s, Tool: %s]", event.UUID, event.ToolUseID)
		header += debugInfo
	}
	output.WriteString(header + "\n")

	// Show hook details
	output.WriteString(fmt.Sprintf("  📟 Command: %s\n", event.HookCommand))
	output.WriteString(fmt.Sprintf("  ✅ Status: %s\n", event.HookStatus))

	// Add debug info
	if f.debugMode {
		output.WriteString(fmt.Sprintf("  🏷️  Level: %s\n", event.Level))
		output.WriteString(fmt.Sprintf("  📂 CWD: %s\n", event.CWD))
		output.WriteString(fmt.Sprintf("  🌳 Branch: %s\n", event.GitBranch))
	}

	return output.String(), nil
}

func (f *Formatter) formatSystemMessage(event *SystemMessage) (string, error) {
	if event.IsMeta && !f.debugMode {
		return "", nil // Skip meta messages unless in debug mode
	}

	levelStr := ""
	if event.Level != "" {
		levelStr = fmt.Sprintf(" [%s]", event.Level)
	}

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 📣 SYSTEM%s", event.Timestamp.Format("15:04:05"), levelStr)
	if f.debugMode {
		debugInfo := fmt.Sprintf(" [UUID: %s", event.UUID)
		if event.IsMeta {
			debugInfo += ", META"
		}
		if event.ToolUseID != "" {
			debugInfo += fmt.Sprintf(", Tool: %s", event.ToolUseID)
		}
		debugInfo += "]"
		header += debugInfo
	}
	header += ":\n"

	// Get level emoji for content
	contentEmoji := ""
	switch event.Level {
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
	message := header + fmt.Sprintf("  %s%s", contentEmoji, event.Content)

	return message + "\n", nil
}

func (f *Formatter) formatSummaryEvent(event *SummaryEvent) (string, error) {
	// Build message with optional debug info
	message := fmt.Sprintf("📋 [SUMMARY] %s", event.Summary)
	if f.debugMode {
		message += fmt.Sprintf(" [LeafUUID: %s]", event.LeafUUID)
	}
	return message + "\n", nil
}

func (f *Formatter) formatUnknownEvent(event *BaseEvent) (string, error) {
	// Build message with optional debug info
	message := fmt.Sprintf("[%s] %s event", event.Timestamp.Format("15:04:05"), event.TypeString)
	if f.debugMode {
		message += fmt.Sprintf(" [UUID: %s]", event.UUID)
	}
	return message + "\n", nil
}

// formatNotificationEvent formats a notification event
func (f *Formatter) formatNotificationEvent(event *NotificationEvent) (string, error) {
	var output strings.Builder

	// Handle events based on HookEventName
	switch event.HookEventName {
	case "PreCompact":
		output.WriteString(f.formatPreCompactEvent(event))
	case "SessionStart":
		output.WriteString(f.formatSessionStartEvent(event))
	case "Notification":
		output.WriteString(f.formatGeneralNotificationEvent(event))
	default:
		// Return empty string for unknown event types
		return "", nil
	}

	// Ensure message ends with newline
	result := output.String()
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, nil
}

// formatPreCompactEvent formats PreCompact events
func (f *Formatter) formatPreCompactEvent(event *NotificationEvent) string {
	var output strings.Builder
	emoji := "🗜️"

	// Use narrator to get the narration message
	formattedMessage, _ := f.narrator.NarrateNotification(narrator.NotificationTypeCompact)

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] %s %s", timeNow().Format("15:04:05"), emoji, event.HookEventName)
	if f.debugMode && len(event.SessionID) >= 8 {
		header += fmt.Sprintf(" [Session: %s]", event.SessionID[:8])
	}
	header += fmt.Sprintf(": %s\n", formattedMessage)
	output.WriteString(header)

	// Add debug info if enabled
	if f.debugMode {
		output.WriteString(fmt.Sprintf("  [DEBUG] Trigger: %s\n", event.Trigger))
		output.WriteString(fmt.Sprintf("  [DEBUG] CWD: %s\n", event.CWD))
		output.WriteString(fmt.Sprintf("  [DEBUG] Transcript: %s\n", event.TranscriptPath))
	}

	// Show narrator emoji
	if formattedMessage != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", formattedMessage))
	}

	return output.String()
}

// formatSessionStartEvent formats SessionStart events
func (f *Formatter) formatSessionStartEvent(event *NotificationEvent) string {
	var output strings.Builder
	emoji := "🚀"

	// Use narrator to get the narration message based on source
	var notificationType narrator.NotificationType
	switch event.Source {
	case "startup":
		notificationType = narrator.NotificationTypeSessionStartStartup
	case "clear":
		notificationType = narrator.NotificationTypeSessionStartClear
	case "resume":
		notificationType = narrator.NotificationTypeSessionStartResume
	case "compact":
		notificationType = narrator.NotificationTypeSessionStartCompact
	default:
		notificationType = narrator.NotificationTypeSessionStartStartup
	}
	formattedMessage, _ := f.narrator.NarrateNotification(notificationType)

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] %s %s", timeNow().Format("15:04:05"), emoji, event.HookEventName)
	if f.debugMode && len(event.SessionID) >= 8 {
		header += fmt.Sprintf(" [Session: %s]", event.SessionID[:8])
	}
	header += fmt.Sprintf(" (source: %s)\n", event.Source)
	output.WriteString(header)

	// Add debug info if enabled
	if f.debugMode {
		output.WriteString(fmt.Sprintf("  [DEBUG] Source: %s\n", event.Source))
		output.WriteString(fmt.Sprintf("  [DEBUG] CWD: %s\n", event.CWD))
		output.WriteString(fmt.Sprintf("  [DEBUG] Transcript: %s\n", event.TranscriptPath))
	}

	// Show narrator emoji
	if formattedMessage != "" {
		output.WriteString(fmt.Sprintf("  💬 %s\n", formattedMessage))
	}

	return output.String()
}

// formatGeneralNotificationEvent formats general Notification events
func (f *Formatter) formatGeneralNotificationEvent(event *NotificationEvent) string {
	var output strings.Builder

	// Parse permission messages
	isPermission, toolName, mcpName, operation := f.parsePermissionMessage(event.Message)

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

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] %s %s", timeNow().Format("15:04:05"), emoji, event.HookEventName)
	if f.debugMode && len(event.SessionID) >= 8 {
		header += fmt.Sprintf(" [Session: %s]", event.SessionID[:8])
	}
	header += fmt.Sprintf(": %s\n", formattedMessage)
	output.WriteString(header)

	// Add debug info if enabled
	if f.debugMode {
		output.WriteString(fmt.Sprintf("  [DEBUG] Original: %s\n", event.Message))
		output.WriteString(fmt.Sprintf("  [DEBUG] CWD: %s\n", event.CWD))
		output.WriteString(fmt.Sprintf("  [DEBUG] Transcript: %s\n", event.TranscriptPath))
	}

	// Use narrator for tool permissions
	if isPermission && displayToolName != "" {
		// Use NarrateToolUsePermission for permission requests
		narration, _ := f.narrator.NarrateToolUsePermission(displayToolName)
		if narration != "" {
			output.WriteString(fmt.Sprintf("  💬 %s\n", narration))
		}

		// Send ConfirmEvent to WebSocket clients
		if f.emitter != nil {
			confirmMsg := &handler.AudioMessage{
				Type:      handler.MessageTypeText,
				ID:        uuid.New().String(),
				Text:      fmt.Sprintf("Tool permission requested: %s", displayToolName),
				Timestamp: time.Now(),
				Metadata: handler.Metadata{
					EventType: "tool_permission",
					ToolName:  displayToolName,
					SessionID: event.SessionID,
				},
			}
			f.emitter.Broadcast(confirmMsg)
		}
	} else if event.Message != "" {
		// Use NarrateText for other notifications
		// Create EventMeta for the notification
		meta := &narrator.EventMeta{
			SessionID: event.SessionID,
			CWD:       event.CWD,
			Timestamp: time.Now(), // NotificationEvent doesn't have timestamp, use current time
		}
		narration, _ := f.narrator.NarrateText(event.Message, false, meta)
		if narration != "" {
			output.WriteString(fmt.Sprintf("  💬 %s\n", narration))
		}
	}

	return output.String()
}

// parsePermissionMessage parses permission messages to extract tool/MCP information
func (f *Formatter) parsePermissionMessage(message string) (isPermission bool, toolName string, mcpName string, operation string) {
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

// Helper functions to avoid importing strings package
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
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
	if n == 1 {
		return []string{s}
	}

	var result []string
	for n > 1 && len(s) > 0 {
		idx := -1
		for i := 0; i <= len(s)-len(sep); i++ {
			if s[i:i+len(sep)] == sep {
				idx = i
				break
			}
		}
		if idx == -1 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
		n--
	}
	if len(s) > 0 {
		result = append(result, s)
	}
	return result
}

// formatTaskCompletionMessage formats a task completion message
func (f *Formatter) formatTaskCompletionMessage(event *TaskCompletionMessage) (string, error) {
	var output strings.Builder

	// Use narrator to build and narrate the task completion message
	narration, _ := f.narrator.NarrateTaskCompletion(
		event.TaskInfo.Description,
		event.TaskInfo.SubagentType,
	)

	// Format the output
	output.WriteString(fmt.Sprintf("[%s] 💬 %s\n",
		event.Timestamp.Format("15:04:05"),
		narration))

	return output.String(), nil
}

// timeNow is a helper function to get current time (for testing)
var timeNow = time.Now

const (
	// MaxMainTextLines is the maximum number of lines to show for main text with placeholders
	MaxMainTextLines = 30
	// MaxCodePreviewLines is the maximum number of lines to show in code block preview
	MaxCodePreviewLines = 5
	// MaxNormalTextLines is the maximum number of lines to show for normal text without code blocks
	MaxNormalTextLines = 30
)

// CodeBlock represents a code block extracted from text
type CodeBlock struct {
	Language string
	Content  string
}

// toRelativePath converts an absolute path to a relative path from cwd
func toRelativePath(cwd, path string) string {
	if cwd == "" || path == "" {
		return path
	}

	// Try to make the path relative to cwd
	relPath, err := filepath.Rel(cwd, path)
	if err != nil {
		// If failed, return the original path
		return path
	}

	// If the relative path starts with "..", it's outside cwd, so return absolute
	if strings.HasPrefix(relPath, "..") {
		return path
	}

	return relPath
}

// ExtractCodeBlocks extracts code blocks from text content
func (f *Formatter) ExtractCodeBlocks(text string) []CodeBlock {
	blocks := []CodeBlock{}

	// Match fenced code blocks with optional language
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		language := match[1]
		if language == "" {
			language = "text"
		}
		blocks = append(blocks, CodeBlock{
			Language: language,
			Content:  match[2],
		})
	}

	return blocks
}

// FormatToolUse formats tool usage for companion display
func (f *Formatter) FormatToolUse(toolName string, meta EventMeta, input map[string]interface{}) string {
	f.currentTool = toolName

	var output strings.Builder

	// Create a copy of input for potential modifications
	modifiedInput := make(map[string]interface{})
	for k, v := range input {
		modifiedInput[k] = v
	}

	// Convert paths to relative for specific tools
	if meta.CWD != "" && (toolName == "Grep" || toolName == "Glob" || toolName == "LS") {
		if path, ok := modifiedInput["path"].(string); ok && path != "" {
			modifiedInput["path"] = toRelativePath(meta.CWD, path)
		}
	}

	// Use narrator with potentially modified input
	narration, _ := f.narrator.NarrateToolUse(toolName, modifiedInput)
	if narration != "" {
		output.WriteString(fmt.Sprintf("  💬 %s", narration))
		// Track file operations for summary
		if toolName == "Read" || toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit" {
			if path, ok := input["file_path"].(string); ok {
				f.fileOperations = append(f.fileOperations, fmt.Sprintf("%s: %s", toolName, path))
			}
		}

		// Special handling for TodoWrite - show details even when narrator is used
		if toolName == "TodoWrite" {
			if todos, ok := input["todos"].([]interface{}); ok {
				for i, todo := range todos {
					if todoMap, ok := todo.(map[string]interface{}); ok {
						content := ""
						if c, ok := todoMap["content"].(string); ok {
							content = c
						}
						if status, ok := todoMap["status"].(string); ok {
							emoji := ""
							switch status {
							case "completed":
								emoji = "✅"
							case "in_progress":
								emoji = "🔄"
							case "pending":
								emoji = "⏳"
							}
							output.WriteString(fmt.Sprintf("\n    %d. %s %s", i+1, emoji, content))
						}
					}
				}
			}
		}

		return output.String() + "\n"
	}

	// Fallback to emoji-based formatting if narrator is not available
	// Use emojis and formatting based on tool type
	switch toolName {
	case "Read", "mcp__ide__read":
		if filePath, ok := input["file_path"].(string); ok {
			f.fileOperations = append(f.fileOperations, fmt.Sprintf("Read: %s", filePath))
			output.WriteString(fmt.Sprintf("  📄 Reading file: %s", filePath))
		}
	case "Write":
		if filePath, ok := input["file_path"].(string); ok {
			f.fileOperations = append(f.fileOperations, fmt.Sprintf("Write: %s", filePath))
			output.WriteString(fmt.Sprintf("  ✏️  Writing file: %s", filePath))
		}
	case "Edit", "MultiEdit":
		if filePath, ok := input["file_path"].(string); ok {
			f.fileOperations = append(f.fileOperations, fmt.Sprintf("Edit: %s", filePath))
			output.WriteString(fmt.Sprintf("  ✂️  Editing file: %s", filePath))
		}
	case "Bash":
		if command, ok := input["command"].(string); ok {
			output.WriteString(fmt.Sprintf("  🖥️  Running command: %s", command))
		}
	case "Grep":
		if pattern, ok := input["pattern"].(string); ok {
			path, _ := input["path"].(string)
			if path == "" {
				path = "current directory"
			}
			output.WriteString(fmt.Sprintf("  🔍 Searching for '%s' in %s", pattern, path))
		}
	case "WebFetch":
		if url, ok := input["url"].(string); ok {
			output.WriteString(fmt.Sprintf("  🌐 Fetching: %s", url))
		}
	case "Task":
		if desc, ok := input["description"].(string); ok {
			output.WriteString(fmt.Sprintf("  🤖 Launching agent: %s", desc))
		}
	case "TodoWrite":
		output.WriteString("  ✅ Updating todo list")
		// Display todo list details
		if todos, ok := input["todos"].([]interface{}); ok {
			for i, todo := range todos {
				if todoMap, ok := todo.(map[string]interface{}); ok {
					content := ""
					if c, ok := todoMap["content"].(string); ok {
						content = c
					}
					if status, ok := todoMap["status"].(string); ok {
						emoji := ""
						switch status {
						case "completed":
							emoji = "✅"
						case "in_progress":
							emoji = "🔄"
						case "pending":
							emoji = "⏳"
						}
						output.WriteString(fmt.Sprintf("\n    %d. %s %s", i+1, emoji, content))
					}
				}
			}
		}
	default:
		if strings.HasPrefix(toolName, "mcp__") {
			// MCP tools
			output.WriteString(fmt.Sprintf("  🔧 MCP Tool: %s", toolName))
		} else {
			output.WriteString(fmt.Sprintf("  🔧 Tool: %s", toolName))
		}
	}

	// Show detailed input for debugging (optional)
	if len(input) > 0 && toolName != "TodoWrite" {
		output.WriteString(fmt.Sprintf(" (id: %s)", meta.ToolID))
	}

	return output.String() + "\n"
}

// FormatAssistantText formats assistant text content with code block extraction
func (f *Formatter) FormatAssistantText(text string, isThinking bool, meta *narrator.EventMeta) string {
	var output strings.Builder

	// Extract code blocks
	codeBlocks := f.ExtractCodeBlocks(text)

	// Prepare text for narration
	processedText := strings.TrimSpace(text)
	if len(codeBlocks) > 0 {
		// Replace code blocks with placeholders
		for i, block := range codeBlocks {
			placeholder := fmt.Sprintf("[CODE BLOCK %d: %s]", i+1, block.Language)
			// Find and replace the original code block
			original := fmt.Sprintf("```%s\n%s```", block.Language, block.Content)
			if block.Language == "text" || block.Language == "" {
				original = fmt.Sprintf("```\n%s```", block.Content)
			}
			processedText = strings.Replace(processedText, original, placeholder, 1)
		}
	}

	// Narrate the text
	narrated, _ := f.narrator.NarrateText(processedText, isThinking, meta)
	output.WriteString(fmt.Sprintf("  💬 %s\n", narrated))

	// Show the main text (only if multiple lines)
	lines := strings.Split(strings.TrimSpace(processedText), "\n")

	// Filter out code block placeholders if any
	var displayLines []string
	if len(codeBlocks) > 0 {
		for _, line := range lines {
			if !strings.HasPrefix(strings.TrimSpace(line), "[CODE BLOCK") || !strings.HasSuffix(strings.TrimSpace(line), "]") {
				displayLines = append(displayLines, line)
			}
		}
	} else {
		displayLines = lines
	}

	// Display text lines with 📝 emoji (only if multiple lines)
	if len(displayLines) > 1 {
		for i, line := range displayLines {
			if i < MaxNormalTextLines {
				if i == 0 {
					output.WriteString(fmt.Sprintf("  📝 %s\n", line))
				} else {
					output.WriteString(fmt.Sprintf("  %s\n", line))
				}
			} else if i == MaxNormalTextLines && len(displayLines) > MaxNormalTextLines+1 {
				output.WriteString(fmt.Sprintf("  ... (%d more lines)\n", len(displayLines)-MaxNormalTextLines))
				break
			}
		}
	}

	// Show code blocks separately if any
	if len(codeBlocks) > 0 {
		for i, block := range codeBlocks {
			if len(displayLines) > 0 || i > 0 {
				output.WriteString("\n")
			}
			output.WriteString(fmt.Sprintf("  📝 Code Block %d (%s):\n", i+1, block.Language))
			output.WriteString("    ```\n")
			// Show first few lines of code
			codeLines := strings.Split(strings.TrimSpace(block.Content), "\n")
			for j, line := range codeLines {
				if j < MaxCodePreviewLines {
					output.WriteString(fmt.Sprintf("    %s\n", line))
				} else if j == MaxCodePreviewLines && len(codeLines) > MaxCodePreviewLines+1 {
					output.WriteString(fmt.Sprintf("    ... (%d more lines)\n", len(codeLines)-MaxCodePreviewLines))
					break
				}
			}
			output.WriteString("    ```\n")
		}
	}

	return output.String()
}

// GetFileSummary returns a summary of file operations performed
func (f *Formatter) GetFileSummary() string {
	if len(f.fileOperations) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("  📁 File Operations Summary:\n")
	for _, op := range f.fileOperations {
		output.WriteString(fmt.Sprintf("    - %s\n", op))
	}

	return output.String()
}

// Reset clears the formatter state
func (f *Formatter) Reset() {
	f.fileOperations = []string{}
	f.currentTool = ""
}
