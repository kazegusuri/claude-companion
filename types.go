package main

import "time"

// BaseEvent contains common fields for all event types
type BaseEvent struct {
	ParentUUID  *string   `json:"parentUuid"`
	IsSidechain bool      `json:"isSidechain"`
	UserType    string    `json:"userType"`
	CWD         string    `json:"cwd"`
	SessionID   string    `json:"sessionId"`
	Version     string    `json:"version"`
	GitBranch   string    `json:"gitBranch"`
	UUID        string    `json:"uuid"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
}

// UserMessage represents a user input
type UserMessage struct {
	BaseEvent
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

// AssistantMessage represents an assistant response
type AssistantMessage struct {
	BaseEvent
	RequestID string `json:"requestId"`
	Message   struct {
		ID           string  `json:"id"`
		Type         string  `json:"type"`
		Role         string  `json:"role"`
		Model        string  `json:"model"`
		StopReason   *string `json:"stop_reason"`
		StopSequence *string `json:"stop_sequence"`
		Content      []struct {
			Type  string      `json:"type"`
			Text  string      `json:"text,omitempty"`
			ID    string      `json:"id,omitempty"`
			Name  string      `json:"name,omitempty"`
			Input interface{} `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens              int    `json:"input_tokens"`
			CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
			OutputTokens             int    `json:"output_tokens"`
			ServiceTier              string `json:"service_tier"`
		} `json:"usage"`
	} `json:"message"`
}

// SystemMessage represents system messages
type SystemMessage struct {
	BaseEvent
	Content   string `json:"content"`
	IsMeta    bool   `json:"isMeta"`
	ToolUseID string `json:"toolUseID,omitempty"`
	Level     string `json:"level,omitempty"`
}

// ToolResultMessage represents tool execution results
type ToolResultMessage struct {
	BaseEvent
	Message struct {
		Role    string `json:"role"`
		Content []struct {
			ToolUseID string      `json:"tool_use_id"`
			Type      string      `json:"type"`
			Content   interface{} `json:"content"`
			IsError   bool        `json:"is_error,omitempty"`
		} `json:"content"`
	} `json:"message"`
	ToolUseResult interface{} `json:"toolUseResult,omitempty"`
}

// EventType constants
const (
	EventTypeUser      = "user"
	EventTypeAssistant = "assistant"
	EventTypeSystem    = "system"
)
