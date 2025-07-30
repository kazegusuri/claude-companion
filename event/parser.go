package event

import (
	"encoding/json"
	"fmt"
)

// Parser handles parsing of JSONL events
type Parser struct{}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// Parse parses a JSON line and returns the appropriate event type
func (p *Parser) Parse(line string) (Event, error) {
	// First, parse to get the event type
	var baseEvent BaseEvent
	if err := json.Unmarshal([]byte(line), &baseEvent); err != nil {
		return nil, fmt.Errorf("failed to parse base event: %w", err)
	}

	// Parse into specific event type based on Type field
	switch baseEvent.TypeString {
	case EventTypeUser:
		var event UserMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse user message: %w", err)
		}
		return &event, nil
	case EventTypeAssistant:
		var event AssistantMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse assistant message: %w", err)
		}
		return &event, nil
	case EventTypeSystem:
		var event SystemMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse system message: %w", err)
		}
		return &event, nil
	case EventTypeSummary:
		var event SummaryEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse summary event: %w", err)
		}
		return &event, nil
	default:
		// Return base event for unknown types
		return &baseEvent, nil
	}
}
