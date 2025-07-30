package event

// Type represents the type of event
type Type string

const (
	TypeNotification Type = "notification"
)

// Event is the common interface for all events
type Event interface {
	Type() Type
}

// EventSender is an interface for sending events
type EventSender interface {
	SendEvent(event Event)
}

// NotificationLogEvent wraps a notification event
type NotificationLogEvent struct {
	Event *NotificationEvent
}

func (e *NotificationLogEvent) Type() Type {
	return TypeNotification
}
