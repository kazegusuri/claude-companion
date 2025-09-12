package event

import (
	"encoding/json"
	"time"
)

// Type represents the type of event
type Type string

// Event is the common interface for all events
type Event interface {
	Type() Type
}

// EventType constants
const (
	EventTypeUser         = "user"
	EventTypeAssistant    = "assistant"
	EventTypeSystem       = "system"
	EventTypeSummary      = "summary"
	EventTypeNotification = "notification"
	EventTypeHook         = "hook"
	EventTypeTaskComplete = "task_completion"
)

// MessageType constants for SessionMessageBase
const (
	MessageTypeSystem    = "system"
	MessageTypeUser      = "user"
	MessageTypeAssistant = "assistant"
)

// Session represents session information
type Session struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path,omitempty"`
}

// SessionMessageBase represents common fields for session messages
type SessionMessageBase struct {
	UUID        string    `json:"uuid"`
	Type        string    `json:"type"` // system, user, assistant
	IsSidechain bool      `json:"is_sidechain"`
	CWD         string    `json:"cwd,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	IsMeta      bool      `json:"is_meta"`
}

// NotificationEvent represents a notification event from the hook log
type NotificationEvent struct {
	Session            Session           `json:"session"`
	HookEventName      string            `json:"hook_event_name"`
	Message            string            `json:"message"`
	Trigger            string            `json:"trigger"`
	CustomInstructions string            `json:"custom_instructions"`
	Source             string            `json:"source"` // For SessionStart events: startup, clear, resume
	Narration          *NarrationMessage `json:"narration,omitempty"`
}

// Type returns the event type
func (e *NotificationEvent) Type() Type {
	return Type(EventTypeNotification)
}

// NarrationMessage represents a narration message to be displayed
type NarrationMessage struct {
	Text string `json:"text"`
}

// SystemMessageContent is the interface for system message content
type SystemMessageContent interface {
	// Marker method to ensure type safety
	isSystemMessageContent()
}

// DefaultSystemMessageContent represents default system message content
type DefaultSystemMessageContent struct {
	Text string `json:"text"`
}

// isSystemMessageContent implements SystemMessageContent interface
func (d *DefaultSystemMessageContent) isSystemMessageContent() {}

// HookSystemMessageContent represents hook-specific system message content
type HookSystemMessageContent struct {
	HookName string `json:"hook_name"`
	Command  string `json:"command,omitempty"`
	Status   string `json:"status,omitempty"`
	Type     string `json:"type,omitempty"`
	Message  string `json:"message,omitempty"`
}

// isSystemMessageContent implements SystemMessageContent interface
func (h *HookSystemMessageContent) isSystemMessageContent() {}

// SystemMessage represents a system message event
type SystemMessage struct {
	SessionMessageBase
	Session    Session              `json:"session"`
	RawContent string               `json:"raw_content"`
	Content    SystemMessageContent `json:"content,omitempty"`
	Level      string               `json:"level"` // error, warning, info, debug
	ToolUseID  string               `json:"tool_use_id,omitempty"`
	Narration  *NarrationMessage    `json:"narration,omitempty"`
}

// Type returns the event type
func (e *SystemMessage) Type() Type {
	return EventTypeSystem
}

// UserMessageContent is the interface for user message content
type UserMessageContent interface {
	// Marker method to ensure type safety
	isUserMessageContent()
}

// UserMessageContentItem is the interface for items that can appear in content arrays
type UserMessageContentItem interface {
	// Marker method to ensure type safety
	isUserMessageContentItem()
}

// UserMessageContentMessage represents user message content
type UserMessageContentMessage struct {
	Text string `json:"text"`
}

// isUserMessageContent implements UserMessageContent interface
func (u *UserMessageContentMessage) isUserMessageContent() {}

// isUserMessageContentItem implements UserMessageContentItem interface
func (u *UserMessageContentMessage) isUserMessageContentItem() {}

// UserMessageContentCommand represents a command execution content
type UserMessageContentCommand struct {
	CommandName    string `json:"command_name"`
	CommandMessage string `json:"command_message"`
	CommandArgs    string `json:"command_args"`
}

// isUserMessageContent implements UserMessageContent interface
func (u *UserMessageContentCommand) isUserMessageContent() {}

// isUserMessageContentItem implements UserMessageContentItem interface
func (u *UserMessageContentCommand) isUserMessageContentItem() {}

// UserMessageContentLocalCommand represents local command output content
type UserMessageContentLocalCommand struct {
	Output string `json:"output"`
}

// isUserMessageContent implements UserMessageContent interface
func (u *UserMessageContentLocalCommand) isUserMessageContent() {}

// isUserMessageContentItem implements UserMessageContentItem interface
func (u *UserMessageContentLocalCommand) isUserMessageContentItem() {}

// UserMessageContentList represents a list of user message content items
type UserMessageContentList struct {
	Items []UserMessageContentItem `json:"items"`
}

// isUserMessageContent implements UserMessageContent interface
func (u *UserMessageContentList) isUserMessageContent() {}

// UserMessageContentInterrupted represents an interrupted message (only appears in arrays)
type UserMessageContentInterrupted struct {
	Reason string `json:"reason"` // e.g., "Request interrupted by user" or "Request interrupted by user for tool use"
}

// isUserMessageContentItem implements UserMessageContentItem interface (not UserMessageContent)
func (u *UserMessageContentInterrupted) isUserMessageContentItem() {}

// UserMessageContentToolResult represents a tool result content
type UserMessageContentToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// isUserMessageContent implements UserMessageContent interface
func (u *UserMessageContentToolResult) isUserMessageContent() {}

// isUserMessageContentItem implements UserMessageContentItem interface
func (u *UserMessageContentToolResult) isUserMessageContentItem() {}

// UserMessageContentUnknown represents unknown content type with raw JSON data
type UserMessageContentUnknown struct {
	Type string          `json:"type,omitempty"` // The type field if present
	Data json.RawMessage `json:"data"`           // Raw JSON data
}

// isUserMessageContent implements UserMessageContent interface
func (u *UserMessageContentUnknown) isUserMessageContent() {}

// isUserMessageContentItem implements UserMessageContentItem interface
func (u *UserMessageContentUnknown) isUserMessageContentItem() {}

// UserMessageData represents the message data in a user message
type UserMessageData struct {
	Role    string             `json:"role"` // "user"
	Content UserMessageContent `json:"content"`
}

// UserMessage represents a user message event
type UserMessage struct {
	SessionMessageBase
	Session       Session           `json:"session"`
	Message       UserMessageData   `json:"message"`
	Narration     *NarrationMessage `json:"narration,omitempty"`
	ToolUseResult interface{}       `json:"toolUseResult,omitempty"` // Optional field for tool use result data
}

// Type returns the event type
func (e *UserMessage) Type() Type {
	return EventTypeUser
}

// AssistantMessageContent is the interface for assistant message content
type AssistantMessageContent interface {
	// Marker method to ensure type safety
	isAssistantMessageContent()
}

// AssistantMessageContentItem is the interface for items that can appear in content arrays
type AssistantMessageContentItem interface {
	// Marker method to ensure type safety
	isAssistantMessageContentItem()
}

// CodeBlock represents a code block extracted from text
type CodeBlock struct {
	Language string `json:"language"`
	Content  string `json:"content"`
}

// AssistantMessageContentText represents text or thinking content in an assistant message
type AssistantMessageContentText struct {
	Type          string            `json:"type"`                     // "text" or "thinking"
	Text          string            `json:"text"`                     // The actual text content
	IsThinking    bool              `json:"is_thinking,omitempty"`    // Whether this is thinking content
	ProcessedText string            `json:"processed_text,omitempty"` // Text with code blocks replaced by placeholders
	CodeBlocks    []CodeBlock       `json:"code_blocks,omitempty"`    // Extracted code blocks
	Narration     *NarrationMessage `json:"narration,omitempty"`      // Narration for this content
}

// isAssistantMessageContent implements AssistantMessageContent interface
func (a *AssistantMessageContentText) isAssistantMessageContent() {}

// isAssistantMessageContentItem implements AssistantMessageContentItem interface
func (a *AssistantMessageContentText) isAssistantMessageContentItem() {}

// AssistantMessageContentToolUse represents tool use content in an assistant message
type AssistantMessageContentToolUse struct {
	Type  string      `json:"type"` // "tool_use"
	ID    string      `json:"id"`
	Name  string      `json:"name"`
	Input interface{} `json:"input"`
}

// isAssistantMessageContentItem implements AssistantMessageContentItem interface
func (a *AssistantMessageContentToolUse) isAssistantMessageContentItem() {}

// AssistantMessageContentList represents a list of assistant message content items
type AssistantMessageContentList struct {
	Items []AssistantMessageContentItem `json:"items"`
}

// isAssistantMessageContent implements AssistantMessageContent interface
func (a *AssistantMessageContentList) isAssistantMessageContent() {}

// APIErrorDetail represents details of an API error
type APIErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// APIErrorMessageContent represents API error content in an assistant message
type APIErrorMessageContent struct {
	StatusCode int            `json:"status_code"`
	ErrorType  string         `json:"error_type"`
	Error      APIErrorDetail `json:"error"`
	RawText    string         `json:"raw_text"` // Original error text for fallback
}

// isAssistantMessageContent implements AssistantMessageContent interface
func (a *APIErrorMessageContent) isAssistantMessageContent() {}

// TokenUsage represents token usage information for assistant messages
type TokenUsage struct {
	InputTokens              int    `json:"input_tokens"`
	CacheCreationInputTokens int    `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int    `json:"cache_read_input_tokens,omitempty"`
	OutputTokens             int    `json:"output_tokens"`
	ServiceTier              string `json:"service_tier,omitempty"`
}

// AssistantMessageData represents the message data in an assistant message
type AssistantMessageData struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"` // "message"
	Role       string                  `json:"role"` // "assistant"
	Model      string                  `json:"model"`
	Content    AssistantMessageContent `json:"content"`
	StopReason *string                 `json:"stop_reason,omitempty"`
	StopSeq    *string                 `json:"stop_seq,omitempty"`
	Usage      *TokenUsage             `json:"usage,omitempty"` // Token usage information
}

// AssistantMessage represents an assistant message event
type AssistantMessage struct {
	SessionMessageBase
	Session           Session              `json:"session"`
	RequestID         string               `json:"request_id,omitempty"`
	Message           AssistantMessageData `json:"message"`
	IsApiErrorMessage bool                 `json:"is_api_error_message,omitempty"`
	Narration         *NarrationMessage    `json:"narration,omitempty"`
}

// Type returns the event type
func (e *AssistantMessage) Type() Type {
	return EventTypeAssistant
}

// SummaryEvent represents a summary event
type SummaryEvent struct {
	Session   Session           `json:"session"`
	LeafUUID  string            `json:"leaf_uuid"`
	Summary   string            `json:"summary"`
	Narration *NarrationMessage `json:"narration,omitempty"`
}

// Type returns the event type
func (e *SummaryEvent) Type() Type {
	return EventTypeSummary
}

// TaskInfo contains information about a Task tool execution
type TaskInfo struct {
	ToolUseID    string `json:"tool_use_id"`
	Description  string `json:"description"`
	SubagentType string `json:"subagent_type"`
}

// TaskCompletionMessage represents a task completion event
type TaskCompletionMessage struct {
	Session   Session           `json:"session"`
	TaskInfo  TaskInfo          `json:"task_info"`
	Timestamp time.Time         `json:"timestamp"`
	Narration *NarrationMessage `json:"narration,omitempty"`
}

// Type returns the event type
func (e *TaskCompletionMessage) Type() Type {
	return EventTypeTaskComplete
}

// Printer is the interface for printing events
type Printer interface {
	Print(event interface{})
}
