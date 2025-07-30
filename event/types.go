package event

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
	TypeString  string    `json:"type"`
}

// Type returns the event type
func (e *BaseEvent) Type() Type {
	return Type(e.TypeString)
}

// UserMessageContent represents the content of a user message
type UserMessageContent struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // Can be string or array
}

// UserMessage represents a user input
type UserMessage struct {
	BaseEvent
	Message UserMessageContent `json:"message"`
}

// AssistantContent represents a content item in an assistant message
type AssistantContent struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	Input    interface{} `json:"input,omitempty"`
	Thinking string      `json:"thinking,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens              int    `json:"input_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens"`
	OutputTokens             int    `json:"output_tokens"`
	ServiceTier              string `json:"service_tier"`
}

// AssistantMessageContent represents the content of an assistant message
type AssistantMessageContent struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Model        string             `json:"model"`
	StopReason   *string            `json:"stop_reason"`
	StopSequence *string            `json:"stop_sequence"`
	Content      []AssistantContent `json:"content"`
	Usage        Usage              `json:"usage"`
}

// AssistantMessage represents an assistant response
type AssistantMessage struct {
	BaseEvent
	RequestID string                  `json:"requestId"`
	Message   AssistantMessageContent `json:"message"`
}

// SystemMessage represents system messages
type SystemMessage struct {
	BaseEvent
	Content           string `json:"content"`
	IsMeta            bool   `json:"isMeta"`
	IsApiErrorMessage bool   `json:"isApiErrorMessage,omitempty"`
	IsCompactSummary  bool   `json:"isCompactSummary,omitempty"`
	ToolUseID         string `json:"toolUseID,omitempty"`
	Level             string `json:"level,omitempty"`
}

// ToolResultContent represents a tool result content item
type ToolResultContent struct {
	ToolUseID string      `json:"tool_use_id"`
	Type      string      `json:"type"`
	Content   interface{} `json:"content"`
	IsError   bool        `json:"is_error,omitempty"`
}

// ToolResultMessageContent represents the content of a tool result message
type ToolResultMessageContent struct {
	Role    string              `json:"role"`
	Content []ToolResultContent `json:"content"`
}

// ToolResultMessage represents tool execution results
type ToolResultMessage struct {
	BaseEvent
	Message       ToolResultMessageContent `json:"message"`
	ToolUseResult interface{}              `json:"toolUseResult,omitempty"`
}

// SummaryEvent represents a session summary
type SummaryEvent struct {
	EventType string `json:"type"`
	Summary   string `json:"summary"`
	LeafUUID  string `json:"leafUuid"`
}

// Type returns the event type
func (e *SummaryEvent) Type() Type {
	return Type(EventTypeSummary)
}

// NotificationEvent represents a notification event from the hook log
type NotificationEvent struct {
	SessionID          string `json:"session_id"`
	TranscriptPath     string `json:"transcript_path"`
	CWD                string `json:"cwd"`
	HookEventName      string `json:"hook_event_name"`
	Message            string `json:"message"`
	Trigger            string `json:"trigger"`
	CustomInstructions string `json:"custom_instructions"`
	Source             string `json:"source"` // For SessionStart events: startup, clear, resume
}

// Type returns the event type
func (e *NotificationEvent) Type() Type {
	return Type(EventTypeNotification)
}

// EventType constants
const (
	EventTypeUser         = "user"
	EventTypeAssistant    = "assistant"
	EventTypeSystem       = "system"
	EventTypeSummary      = "summary"
	EventTypeNotification = "notification"
)
