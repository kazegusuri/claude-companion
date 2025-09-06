package event

import (
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

// SessionEvent represents an event related to session creation/updates
type SessionEvent struct {
	SessionID      string
	UUID           string
	CWD            string
	TranscriptPath string
	EventType      string // "create", "update", "delete"
	Timestamp      time.Time
}

// Type returns the event type
func (e *SessionEvent) Type() Type {
	return Type("session")
}

// FrontendEvent represents an event from the frontend
type FrontendEvent struct {
	EventType string                 `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
}

// Type returns the event type
func (e *FrontendEvent) Type() Type {
	return Type("frontend")
}
