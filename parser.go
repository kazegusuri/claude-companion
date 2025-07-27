package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EventParser handles parsing and formatting of JSONL events
type EventParser struct {
	companion *CompanionFormatter
	debugMode bool
}

// NewEventParser creates a new EventParser instance
func NewEventParser(narrator Narrator) *EventParser {
	return &EventParser{
		companion: NewCompanionFormatter(narrator),
		debugMode: false, // Default to normal mode
	}
}

// SetDebugMode enables or disables debug mode
func (p *EventParser) SetDebugMode(enabled bool) {
	p.debugMode = enabled
}

// ParseAndFormat parses a JSON line and returns formatted output
func (p *EventParser) ParseAndFormat(line string) (string, error) {
	// First, parse to get the event type
	var baseEvent BaseEvent
	if err := json.Unmarshal([]byte(line), &baseEvent); err != nil {
		return "", fmt.Errorf("failed to parse base event: %w", err)
	}

	// Route to appropriate formatter based on event type
	// Each formatter handles both companion and non-companion modes
	switch baseEvent.Type {
	case EventTypeUser:
		return p.formatUserMessage(line)
	case EventTypeAssistant:
		return p.formatAssistantMessage(line)
	case EventTypeSystem:
		return p.formatSystemMessage(line)
	case EventTypeSummary:
		return p.formatSummaryEvent(line)
	default:
		// Handle any unknown event types gracefully
		return p.formatUnknownEvent(line, baseEvent)
	}
}

func (p *EventParser) formatUserMessage(line string) (string, error) {
	var event UserMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse user message: %w", err)
	}

	var output strings.Builder

	// Always use enhanced formatting with emojis
	output.WriteString(fmt.Sprintf("\n[%s] 👤 USER:", event.Timestamp.Format("15:04:05")))

	// Add debug info if enabled
	if p.debugMode {
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
		if p.debugMode && len(lines) > 3 {
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
		if p.debugMode {
			output.WriteString(fmt.Sprintf("\n  [DEBUG] Unknown content type: %T", event.Message.Content))
		}
	}

	return output.String(), nil
}

func (p *EventParser) formatAssistantMessage(line string) (string, error) {
	var event AssistantMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse assistant message: %w", err)
	}

	var output strings.Builder

	// Always use enhanced formatting with emojis
	output.WriteString(fmt.Sprintf("\n[%s] 🤖 ASSISTANT (%s):", event.Timestamp.Format("15:04:05"), event.Message.Model))

	// Add debug info if enabled
	if p.debugMode {
		output.WriteString(fmt.Sprintf(" [ID: %s, ReqID: %s]", event.Message.ID, event.RequestID))
		if event.Message.StopReason != nil {
			output.WriteString(fmt.Sprintf(" [Stop: %s]", *event.Message.StopReason))
		}
	}

	// Track if we have any content to show summary for
	hasContent := false

	for _, content := range event.Message.Content {
		hasContent = true
		switch content.Type {
		case "text":
			formatted := p.companion.FormatAssistantText(content.Text)
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
			formatted := p.companion.FormatToolUse(content.Name, content.ID, inputMap)
			output.WriteString(formatted)
			// Add debug info showing tool use details
			if p.debugMode {
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
		summary := p.companion.GetFileSummary()
		if summary != "" {
			output.WriteString(summary)
		}
		// Reset for next message
		p.companion.Reset()
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

func (p *EventParser) formatSystemMessage(line string) (string, error) {
	var event SystemMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse system message: %w", err)
	}

	if event.IsMeta && !p.debugMode {
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
	if p.debugMode {
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

func (p *EventParser) formatSummaryEvent(line string) (string, error) {
	var event SummaryEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse summary event: %w", err)
	}

	// Always use enhanced formatting with emojis
	output := fmt.Sprintf("\n📋 [SUMMARY] %s", event.Summary)

	// Add debug info if enabled
	if p.debugMode {
		output += fmt.Sprintf(" [LeafUUID: %s]", event.LeafUUID)
	}

	return output, nil
}

func (p *EventParser) formatUnknownEvent(line string, baseEvent BaseEvent) (string, error) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("\n[%s] %s event", baseEvent.Timestamp.Format("15:04:05"), baseEvent.Type))

	// Also show raw JSON for unknown types
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err == nil {
		prettyJSON, _ := json.MarshalIndent(data, "  ", "  ")
		output.WriteString(fmt.Sprintf("\n  Raw: %s", string(prettyJSON)))
	}

	return output.String(), nil
}
