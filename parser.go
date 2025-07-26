package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EventParser handles parsing and formatting of JSONL events
type EventParser struct{}

// NewEventParser creates a new EventParser instance
func NewEventParser() *EventParser {
	return &EventParser{}
}

// ParseAndFormat parses a JSON line and returns formatted output
func (p *EventParser) ParseAndFormat(line string) (string, error) {
	// First, parse to get the event type
	var baseEvent BaseEvent
	if err := json.Unmarshal([]byte(line), &baseEvent); err != nil {
		return "", fmt.Errorf("failed to parse base event: %w", err)
	}

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
		return p.formatUnknownEvent(line, baseEvent)
	}
}

func (p *EventParser) formatUserMessage(line string) (string, error) {
	var event UserMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse user message: %w", err)
	}

	var output strings.Builder

	// Handle content as either string or array
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

func (p *EventParser) formatAssistantMessage(line string) (string, error) {
	var event AssistantMessage
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse assistant message: %w", err)
	}

	var output strings.Builder
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
	return fmt.Sprintf("\n[%s] SYSTEM%s: %s", event.Timestamp.Format("15:04:05"), levelStr, event.Content), nil
}

func (p *EventParser) formatSummaryEvent(line string) (string, error) {
	var event SummaryEvent
	if err := json.Unmarshal([]byte(line), &event); err != nil {
		return "", fmt.Errorf("failed to parse summary event: %w", err)
	}
	return fmt.Sprintf("\n[SUMMARY] %s", event.Summary), nil
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
