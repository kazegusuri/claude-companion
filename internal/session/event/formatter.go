package event

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// Formatter handles formatting of parsed events
type Formatter struct {
	narrator       narrator.Narrator
	fileOperations []string
	currentTool    string
	emitter        handler.MessageEmitter
}

// NewFormatter creates a new Formatter instance
func NewFormatter(narrator narrator.Narrator) *Formatter {
	return &Formatter{
		narrator:       narrator,
		fileOperations: make([]string, 0),
	}
}

// SetMessageEmitter sets the message emitter for sending events
func (f *Formatter) SetMessageEmitter(emitter handler.MessageEmitter) {
	f.emitter = emitter
}

// SetDebugMode enables or disables debug mode (deprecated - uses global logger debug mode)
func (f *Formatter) SetDebugMode(enabled bool) {
	// This method is kept for compatibility but now uses global logger debug mode
	logger.SetDebugMode(enabled)
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
	case *TaskCompletionMessage:
		return f.formatTaskCompletionMessage(e)
	case *BaseEvent:
		return f.formatUnknownEvent(e)
	default:
		return "", fmt.Errorf("unknown event type: %T", event)
	}
}

func (f *Formatter) formatUserMessage(event *UserMessage) (string, error) {
	// Skip meta messages unless in debug mode
	if event.IsMeta && !logger.IsDebugMode() {
		return "", nil
	}

	// Note: We don't skip tool_result only messages anymore
	// They should be formatted and displayed

	var output strings.Builder

	// Extract text content
	textContent := f.extractTextFromUserContent(event.Message.Content)

	// Check if this is a user command (XML tag format)
	isUserCommand := f.isUserCommand(textContent)

	// Send to WebSocket if emitter is available and it's not a user command, meta message, or tool_result only
	if f.emitter != nil && !isUserCommand && !event.IsMeta && !f.hasOnlyToolResult(event.Message.Content) {
		chatMsg := &handler.ChatMessage{
			Type:      handler.MessageTypeUser,
			ID:        event.UUID,
			Role:      handler.MessageRoleUser,
			Text:      textContent,
			Priority:  1,
			Timestamp: event.Timestamp,
			Metadata: handler.Metadata{
				EventType: "user_message",
				SessionID: event.SessionID,
				Role:      handler.MessageRoleUser,
			},
		}
		f.emitter.BroadcastChat(chatMsg)
	}

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 👤 USER:", event.Timestamp.Format("15:04:05"))
	if isUserCommand {
		header += " [COMMAND]"
	}
	if event.IsMeta {
		header += " [META]"
	}
	if logger.IsDebugMode() {
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
		if logger.IsDebugMode() && len(lines) > 3 {
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
		if logger.IsDebugMode() {
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

// extractTextFromUserContent extracts text from user message content
func (f *Formatter) extractTextFromUserContent(content interface{}) string {
	switch c := content.(type) {
	case string:
		return c
	case []interface{}:
		var texts []string
		for _, item := range c {
			if contentMap, ok := item.(map[string]interface{}); ok {
				if contentType, ok := contentMap["type"].(string); ok && contentType == "text" {
					if text, ok := contentMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
		}
		return strings.Join(texts, "\n")
	default:
		return ""
	}
}

// hasOnlyToolResult checks if the user message content contains only tool_result type
func (f *Formatter) hasOnlyToolResult(content interface{}) bool {
	contentArray, ok := content.([]interface{})
	if !ok || len(contentArray) == 0 {
		return false
	}

	// Check if all items are tool_result type
	for _, item := range contentArray {
		if contentMap, ok := item.(map[string]interface{}); ok {
			if contentType, ok := contentMap["type"].(string); ok {
				if contentType != "tool_result" {
					return false
				}
			} else {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

// isUserCommand checks if the text content is a user command in XML tag format
func (f *Formatter) isUserCommand(text string) bool {
	if text == "" {
		return false
	}

	// Get the first line
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return false
	}

	firstLine := strings.TrimSpace(lines[0])

	// Check if the first line looks like an XML tag: <tag>content</tag>
	// Pattern: starts with <tag> and ends with </tag>
	if len(firstLine) > 0 && firstLine[0] == '<' {
		// Find the end of the opening tag
		endIdx := strings.Index(firstLine, ">")
		if endIdx > 1 { // Must have at least one character for tag name
			// Extract tag name (between < and >)
			tagName := firstLine[1:endIdx]

			// Check if the line ends with the corresponding closing tag
			expectedClosingTag := "</" + tagName + ">"
			if strings.HasSuffix(firstLine, expectedClosingTag) {
				// This is a valid XML command format
				return true
			}
		}
	}

	return false
}

func (f *Formatter) formatAssistantMessage(event *AssistantMessage) (string, error) {
	var output strings.Builder

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 🤖 ASSISTANT (%s):", event.Timestamp.Format("15:04:05"), event.Message.Model)
	if logger.IsDebugMode() {
		header += fmt.Sprintf(" [ID: %s, ReqID: %s]", event.Message.ID, event.RequestID)
		if event.Message.StopReason != nil {
			header += fmt.Sprintf(" [Stop: %s]", *event.Message.StopReason)
		}
	}
	output.WriteString(header + "\n")

	// Check if this is an API error message
	if event.IsApiErrorMessage {
		// Parse the API error from the content
		for _, content := range event.Message.Content {
			if content.Type == "text" {
				// Extract status code and JSON from the error message
				// Format: "API Error: 500 {\"type\":\"error\",\"error\":{...}}"
				statusCode := 0
				jsonStartIdx := strings.Index(content.Text, "{")

				// Extract status code if present
				if strings.HasPrefix(content.Text, "API Error: ") {
					parts := strings.SplitN(content.Text, " ", 4)
					if len(parts) >= 3 {
						if code, err := strconv.Atoi(parts[2]); err == nil {
							statusCode = code
						}
					}
				}

				if jsonStartIdx != -1 {
					jsonStr := content.Text[jsonStartIdx:]
					var apiError APIError
					if err := json.Unmarshal([]byte(jsonStr), &apiError); err == nil {
						// Pass the parsed API error to the narrator
						narration, _ := f.narrator.NarrateAPIError(statusCode, apiError.Error.Type, apiError.Error.Message)
						if narration != "" {
							output.WriteString(fmt.Sprintf("  ❌ %s\n", narration))
						} else {
							// Fallback to formatted error
							output.WriteString(fmt.Sprintf("  ❌ API Error %d: %s - %s\n", statusCode, apiError.Error.Type, apiError.Error.Message))
						}
					} else {
						// Fallback to raw text if JSON parsing fails
						output.WriteString(fmt.Sprintf("  ❌ %s\n", content.Text))
					}
				} else {
					// No JSON found, use raw text
					output.WriteString(fmt.Sprintf("  ❌ %s\n", content.Text))
				}
			}
		}
		result := output.String()
		if result != "" && !strings.HasSuffix(result, "\n") {
			result += "\n"
		}
		return result, nil
	}

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
			if logger.IsDebugMode() {
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
	if event.IsMeta && !logger.IsDebugMode() {
		return "", nil // Skip meta messages unless in debug mode
	}

	var output strings.Builder

	// Build header
	header := fmt.Sprintf("[%s] 🪝 HOOK [%s]", event.Timestamp.Format("15:04:05"), event.HookEventType)
	if logger.IsDebugMode() {
		debugInfo := fmt.Sprintf(" [UUID: %s, Tool: %s]", event.UUID, event.ToolUseID)
		header += debugInfo
	}
	output.WriteString(header + "\n")

	// Show hook details
	output.WriteString(fmt.Sprintf("  📟 Command: %s\n", event.HookCommand))
	output.WriteString(fmt.Sprintf("  ✅ Status: %s\n", event.HookStatus))

	// Add debug info
	if logger.IsDebugMode() {
		output.WriteString(fmt.Sprintf("  🏷️  Level: %s\n", event.Level))
		output.WriteString(fmt.Sprintf("  📂 CWD: %s\n", event.CWD))
		output.WriteString(fmt.Sprintf("  🌳 Branch: %s\n", event.GitBranch))
	}

	return output.String(), nil
}

func (f *Formatter) formatSystemMessage(event *SystemMessage) (string, error) {
	if event.IsMeta && !logger.IsDebugMode() {
		return "", nil // Skip meta messages unless in debug mode
	}

	// Send to WebSocket if emitter is available and it's not a meta message
	if f.emitter != nil && !event.IsMeta {
		chatMsg := &handler.ChatMessage{
			Type:      handler.MessageTypeSystem,
			ID:        event.UUID,
			Role:      handler.MessageRoleSystem,
			Text:      event.Content,
			Priority:  1,
			Timestamp: event.Timestamp,
			Metadata: handler.Metadata{
				EventType: "system_message",
				SessionID: event.SessionID,
				Role:      handler.MessageRoleSystem,
			},
		}
		f.emitter.BroadcastChat(chatMsg)
	}

	levelStr := ""
	if event.Level != "" {
		levelStr = fmt.Sprintf(" [%s]", event.Level)
	}

	// Build header with optional debug info
	header := fmt.Sprintf("[%s] 📣 SYSTEM%s", event.Timestamp.Format("15:04:05"), levelStr)
	if logger.IsDebugMode() {
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
	if logger.IsDebugMode() {
		message += fmt.Sprintf(" [LeafUUID: %s]", event.LeafUUID)
	}
	return message + "\n", nil
}

func (f *Formatter) formatUnknownEvent(event *BaseEvent) (string, error) {
	// Build message with optional debug info
	message := fmt.Sprintf("[%s] %s event", event.Timestamp.Format("15:04:05"), event.TypeString)
	if logger.IsDebugMode() {
		message += fmt.Sprintf(" [UUID: %s]", event.UUID)
	}
	return message + "\n", nil
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
