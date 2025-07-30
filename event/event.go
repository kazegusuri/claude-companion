package event

import (
	"fmt"
)

// Type represents the type of event
type Type string

const (
	TypeSessionLog   Type = "session_log"
	TypeNotification Type = "notification"
)

// Event is the common interface for all events
type Event interface {
	Type() Type
	Process(handler *Handler) error
}

// EventSender is an interface for sending events
type EventSender interface {
	SendEvent(event Event)
}

// SessionLogEvent wraps a session log line
type SessionLogEvent struct {
	Line string
}

func (e *SessionLogEvent) Type() Type {
	return TypeSessionLog
}

func (e *SessionLogEvent) Process(handler *Handler) error {
	output, err := handler.parser.ParseAndFormat(e.Line)
	if err != nil {
		return fmt.Errorf("failed to parse event: %w", err)
	}
	if output != "" {
		fmt.Print(output)
	}
	return nil
}

// NotificationLogEvent wraps a notification event
type NotificationLogEvent struct {
	Event *NotificationEvent
}

func (e *NotificationLogEvent) Type() Type {
	return TypeNotification
}

func (e *NotificationLogEvent) Process(handler *Handler) error {
	handler.formatNotificationEvent(e.Event)
	return nil
}
