package handler

import (
	"time"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeAudio           MessageType = "audio"
	MessageTypeText            MessageType = "text"
	MessageTypePing            MessageType = "ping"
	MessageTypePong            MessageType = "pong"
	MessageTypeError           MessageType = "error"
	MessageTypeUserMessage     MessageType = "user_message"
	MessageTypeConfirmResponse MessageType = "confirm_response"
)

// AudioMessage represents a message containing audio data
type AudioMessage struct {
	Type      MessageType `json:"type"`
	ID        string      `json:"id"`
	Text      string      `json:"text"`
	AudioData string      `json:"audioData,omitempty"` // Base64 encoded WAV data
	Priority  int         `json:"priority"`
	Timestamp time.Time   `json:"timestamp"`
	Metadata  Metadata    `json:"metadata,omitempty"`
}

// Metadata contains additional information about the message
type Metadata struct {
	EventType  string  `json:"eventType"`
	ToolName   string  `json:"toolName,omitempty"`
	Speaker    int     `json:"speaker,omitempty"`
	SampleRate int     `json:"sampleRate,omitempty"`
	Duration   float64 `json:"duration,omitempty"`
	SessionID  string  `json:"sessionId,omitempty"`
}

// ClientMessage represents a generic message from the client
type ClientMessage struct {
	Type      MessageType `json:"type"`
	SessionID string      `json:"sessionId"` // Session ID for the message
	Text      string      `json:"text,omitempty"`
	Action    string      `json:"action,omitempty"`    // For confirm_response: "permit" or "deny"
	MessageID string      `json:"messageId,omitempty"` // For confirm_response: ID of the permission request message
	Timestamp string      `json:"timestamp"`
}
