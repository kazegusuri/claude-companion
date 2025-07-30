package event

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kazegusuri/claude-companion/narrator"
)

// Formatter handles formatting of parsed events
type Formatter struct {
	companion *CompanionFormatter
	debugMode bool
}

// NewFormatter creates a new Formatter instance
func NewFormatter(narrator narrator.Narrator) *Formatter {
	return &Formatter{
		companion: NewCompanionFormatter(narrator),
		debugMode: false,
	}
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
	case *SummaryEvent:
		return f.formatSummaryEvent(e)
	case *BaseEvent:
		return f.formatUnknownEvent(e)
	default:
		return "", fmt.Errorf("unknown event type: %T", event)
	}
}

func (f *Formatter) formatUserMessage(event *UserMessage) (string, error) {
	var output strings.Builder

	// Always use enhanced formatting with emojis
	output.WriteString(fmt.Sprintf("\n[%s] 👤 USER:", event.Timestamp.Format("15:04:05")))

	// Add debug info if enabled
	if f.debugMode {
		output.WriteString(fmt.Sprintf(" [UUID: %s]", event.UUID))
	}

	switch content := event.Message.Content.(type) {
	case string:
		// Truncate long messages
		lines := strings.Split(strings.TrimSpace(content), "\n")
		for i, line := range lines {
			if i < 3 {
				if i == 0 {
					output.WriteString(fmt.Sprintf("\n  💬 %s", line))
				} else {
					output.WriteString(fmt.Sprintf("\n  %s", line))
				}
			} else if i == 3 && len(lines) > 4 {
				output.WriteString(fmt.Sprintf("\n  ... (%d more lines)", len(lines)-3))
				break
			}
		}
		// Add full content in debug mode
		if f.debugMode && len(lines) > 3 {
			output.WriteString(fmt.Sprintf("\n  [DEBUG] Full content: %d lines, %d chars", len(lines), len(content)))
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
								output.WriteString("\n  🎯 Command execution")
							} else if strings.Contains(text, "<local-command-stdout>") {
								output.WriteString("\n  📤 Command output")
							} else {
								// Normal text - truncate if needed
								lines := strings.Split(strings.TrimSpace(text), "\n")
								for i, line := range lines {
									if i < 3 {
										if i == 0 {
											output.WriteString(fmt.Sprintf("\n  💬 %s", line))
										} else {
											output.WriteString(fmt.Sprintf("\n  %s", line))
										}
									} else if i == 3 && len(lines) > 4 {
										output.WriteString(fmt.Sprintf("\n  ... (%d more lines)", len(lines)-3))
										break
									}
								}
							}
						}
					case "tool_result":
						toolID := contentMap["tool_use_id"]
						output.WriteString(fmt.Sprintf("\n  ✅ Tool Result: %v", toolID))
						// Check if it has error
						if isError, ok := contentMap["is_error"].(bool); ok && isError {
							output.WriteString(" ❌ (error)")
						}
					}
				}
			}
		}
	default:
		output.WriteString(fmt.Sprintf("\n  %v", event.Message.Content))
		if f.debugMode {
			output.WriteString(fmt.Sprintf("\n  [DEBUG] Unknown content type: %T", event.Message.Content))
		}
	}

	return output.String(), nil
}

func (f *Formatter) formatAssistantMessage(event *AssistantMessage) (string, error) {
	var output strings.Builder

	// Always use enhanced formatting with emojis
	output.WriteString(fmt.Sprintf("\n[%s] 🤖 ASSISTANT (%s):", event.Timestamp.Format("15:04:05"), event.Message.Model))

	// Add debug info if enabled
	if f.debugMode {
		output.WriteString(fmt.Sprintf(" [ID: %s, ReqID: %s]", event.Message.ID, event.RequestID))
		if event.Message.StopReason != nil {
			output.WriteString(fmt.Sprintf(" [Stop: %s]", *event.Message.StopReason))
		}
	}

	// Track if we have any content to show summary for
	hasContent := false

	for i := range event.Message.Content {
		content := &event.Message.Content[i]
		hasContent = true
		switch content.Type {
		case "text":
			formatted := f.companion.FormatAssistantText(content.Text)
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
			formatted := f.companion.FormatToolUse(content.Name, content.ID, inputMap)
			output.WriteString(formatted)
			// Add debug info showing tool use details
			if f.debugMode {
				output.WriteString(fmt.Sprintf("\n  [DEBUG] Tool Use: %s (id: %s)", content.Name, content.ID))
				if content.Input != nil {
					inputJSON, _ := json.MarshalIndent(content.Input, "    ", "  ")
					output.WriteString(fmt.Sprintf("\n    Input: %s", string(inputJSON)))
				}
			}
		}
	}

	// Show file operations summary first if we had any content
	if hasContent {
		summary := f.companion.GetFileSummary()
		if summary != "" {
			output.WriteString(summary)
		}
		// Reset for next message
		f.companion.Reset()
	}

	// Add token usage at the end if present
	if event.Message.Usage.OutputTokens > 0 {
		output.WriteString(fmt.Sprintf("\n  💰 Tokens: input=%d, output=%d, cache_read=%d, cache_creation=%d",
			event.Message.Usage.InputTokens,
			event.Message.Usage.OutputTokens,
			event.Message.Usage.CacheReadInputTokens,
			event.Message.Usage.CacheCreationInputTokens))
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

	// Always use enhanced formatting with emojis
	emoji := "ℹ️"
	switch event.Level {
	case "error":
		emoji = "❌"
	case "warning":
		emoji = "⚠️"
	case "info":
		emoji = "ℹ️"
	case "debug":
		emoji = "🐛"
	}

	output := fmt.Sprintf("\n[%s] %s SYSTEM%s: %s", event.Timestamp.Format("15:04:05"), emoji, levelStr, event.Content)

	// Add debug info if enabled
	if f.debugMode {
		debugInfo := fmt.Sprintf(" [UUID: %s", event.UUID)
		if event.IsMeta {
			debugInfo += ", META"
		}
		if event.ToolUseID != "" {
			debugInfo += fmt.Sprintf(", Tool: %s", event.ToolUseID)
		}
		debugInfo += "]"
		output += debugInfo
	}

	return output, nil
}

func (f *Formatter) formatSummaryEvent(event *SummaryEvent) (string, error) {
	// Always use enhanced formatting with emojis
	output := fmt.Sprintf("\n📋 [SUMMARY] %s", event.Summary)

	// Add debug info if enabled
	if f.debugMode {
		output += fmt.Sprintf(" [LeafUUID: %s]", event.LeafUUID)
	}

	return output, nil
}

func (f *Formatter) formatUnknownEvent(event *BaseEvent) (string, error) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n[%s] %s event", event.Timestamp.Format("15:04:05"), event.TypeString))

	if f.debugMode {
		output.WriteString(fmt.Sprintf(" [UUID: %s]", event.UUID))
	}

	return output.String(), nil
}
