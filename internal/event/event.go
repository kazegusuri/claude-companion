package event

import "time"

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
	EventTypeEmitter      = "emitter"
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

// SystemMessage represents a system message event
type SystemMessage struct {
	SessionMessageBase
	Session   Session           `json:"session"`
	Content   string            `json:"content"`
	Level     string            `json:"level"` // error, warning, info, debug
	ToolUseID string            `json:"tool_use_id,omitempty"`
	Narration *NarrationMessage `json:"narration,omitempty"`
}

// Type returns the event type
func (e *SystemMessage) Type() Type {
	return EventTypeSystem
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

// EmitterEvent represents an event for emitting messages to WebSocket clients
type EmitterEvent struct {
	SessionID string      `json:"session_id"`
	Message   interface{} `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
}

// Type returns the event type
func (e *EmitterEvent) Type() Type {
	return EventTypeEmitter
}

// Printer is the interface for printing events
type Printer interface {
	Print(event interface{})
}
