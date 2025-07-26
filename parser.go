package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EventParser handles parsing and formatting of JSONL events
type EventParser struct {
	companion     *CompanionFormatter
	companionMode bool
}

// NewEventParser creates a new EventParser instance
func NewEventParser() *EventParser {
	return &EventParser{
		companion:     NewCompanionFormatter(),
		companionMode: true, // Default to companion mode
	}
}

// SetNarrator sets the narrator for the companion formatter
func (p *EventParser) SetNarrator(n Narrator) {
	if p.companion != nil {
		p.companion.SetNarrator(n)
	}
}

// SetCompanionMode enables or disables companion mode
func (p *EventParser) SetCompanionMode(enabled bool) {
	p.companionMode = enabled
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

	if !p.companionMode {
		// Simple format for non-companion mode
		switch content := event.Message.Content.(type) {
		case string:
			output.WriteString(fmt.Sprintf("\n[%s] USER: %s", event.Timestamp.Format("15:04:05"), content))
		case []interface{}:
			output.WriteString(fmt.Sprintf("\n[%s] USER:", event.Timestamp.Format("15:04:05")))
			for _, item := range content {
				if contentMap, ok := item.(map[string]interface{}); ok {
					if contentType, ok := contentMap["type"].(string); ok {
						switch contentType {
						case "text":
							if text, ok := contentMap["text"].(string); ok {
								output.WriteString(fmt.Sprintf("\n  Text: %s", text))
							}
						case "tool_result":
							output.WriteString(fmt.Sprintf("\n  Tool Result: %v", contentMap["tool_use_id"]))
						}
					}
				}
			}
		default:
			output.WriteString(fmt.Sprintf("\n[%s] USER: %v", event.Timestamp.Format("15:04:05"), event.Message.Content))
		}
		return output.String(), nil
	}

	// Companion mode formatting
	switch content := event.Message.Content.(type) {
	case string:
		output.WriteString(fmt.Sprintf("\n[%s] 👤 USER:", event.Timestamp.Format("15:04:05")))
		// Truncate long messages
		lines := strings.Split(strings.TrimSpace(content), "\n")
		for i, line := range lines {
			if i < 3 {
				output.WriteString(fmt.Sprintf("\n  %s", line))
			} else if i == 3 && len(lines) > 4 {
				output.WriteString(fmt.Sprintf("\n  ... (%d more lines)", len(lines)-3))
				break
			}
		}
	case []interface{}:
		output.WriteString(fmt.Sprintf("\n[%s] 👤 USER:", event.Timestamp.Format("15:04:05")))
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
										output.WriteString(fmt.Sprintf("\n  %s", line))
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
		output.WriteString(fmt.Sprintf("\n[%s] 👤 USER: %v", event.Timestamp.Format("15:04:05"), event.Message.Content))
	}

	return output.String(), nil
}

func (p *EventParser) formatAssistantMessage(line string) (string, error) {
	var event AssistantMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse assistant message: %w", err)
	}

	var output strings.Builder

	if !p.companionMode {
		// Simple format for non-companion mode
		output.WriteString(fmt.Sprintf("\n[%s] ASSISTANT (%s):", event.Timestamp.Format("15:04:05"), event.Message.Model))

		for _, content := range event.Message.Content {
			switch content.Type {
			case "text":
				output.WriteString(fmt.Sprintf("\n  Text: %s", content.Text))
			case "tool_use":
				output.WriteString(fmt.Sprintf("\n  Tool Use: %s (id: %s)", content.Name, content.ID))
				if content.Input != nil {
					inputJSON, _ := json.MarshalIndent(content.Input, "    ", "  ")
					output.WriteString(fmt.Sprintf("\n    Input: %s", string(inputJSON)))
				}
			}
		}

		if event.Message.Usage.OutputTokens > 0 {
			output.WriteString(fmt.Sprintf("\n  Tokens: input=%d, output=%d, cache_read=%d, cache_creation=%d",
				event.Message.Usage.InputTokens,
				event.Message.Usage.OutputTokens,
				event.Message.Usage.CacheReadInputTokens,
				event.Message.Usage.CacheCreationInputTokens))
		}

		return output.String(), nil
	}

	// Companion mode formatting
	output.WriteString(fmt.Sprintf("\n[%s] 🤖 ASSISTANT (%s):", event.Timestamp.Format("15:04:05"), event.Message.Model))

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
		}
	}

	// Add token usage if present
	if event.Message.Usage.OutputTokens > 0 {
		output.WriteString(fmt.Sprintf("\n  💰 Tokens: in=%d, out=%d, cache=%d",
			event.Message.Usage.InputTokens,
			event.Message.Usage.OutputTokens,
			event.Message.Usage.CacheReadInputTokens))
	}

	// Show file operations summary at end of message if we had any content
	if hasContent {
		summary := p.companion.GetFileSummary()
		if summary != "" {
			output.WriteString(summary)
		}
		// Reset for next message
		p.companion.Reset()
	}

	return output.String(), nil
}

func (p *EventParser) formatSystemMessage(line string) (string, error) {
	var event SystemMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse system message: %w", err)
	}

	if event.IsMeta {
		return "", nil // Skip meta messages
	}

	levelStr := ""
	if event.Level != "" {
		levelStr = fmt.Sprintf(" [%s]", event.Level)
	}

	if !p.companionMode {
		// Simple format
		return fmt.Sprintf("\n[%s] SYSTEM%s: %s", event.Timestamp.Format("15:04:05"), levelStr, event.Content), nil
	}

	// Companion mode - choose emoji based on level
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

	return fmt.Sprintf("\n[%s] %s SYSTEM%s: %s", event.Timestamp.Format("15:04:05"), emoji, levelStr, event.Content), nil
}

func (p *EventParser) formatSummaryEvent(line string) (string, error) {
	var event SummaryEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse summary event: %w", err)
	}

	if !p.companionMode {
		return fmt.Sprintf("\n[SUMMARY] %s", event.Summary), nil
	}

	return fmt.Sprintf("\n📋 [SUMMARY] %s", event.Summary), nil
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
