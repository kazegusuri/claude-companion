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
