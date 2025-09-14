package handler

import (
	"time"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// Core message types for chat
	MessageTypeSystem       MessageType = "system"
	MessageTypeUser         MessageType = "user"
	MessageTypeAssistant    MessageType = "assistant"
	MessageTypeNotification MessageType = "notification" // Notification messages (always delivered regardless of mode)

	// WebSocket control messages
	MessageTypePing            MessageType = "ping"
	MessageTypePong            MessageType = "pong"
	MessageTypeUserMessage     MessageType = "user_message"     // User input from client
	MessageTypeConfirmResponse MessageType = "confirm_response" // Tool permission response
	MessageTypeUpdateState     MessageType = "update_state"     // Update WebSocket state

	// Legacy types (kept for compatibility)
	MessageTypeAudio MessageType = "audio" // Deprecated: use MessageTypeAssistant with SubTypeAudio
	MessageTypeText  MessageType = "text"  // Deprecated: use MessageTypeAssistant with SubTypeText
	MessageTypeError MessageType = "error" // Deprecated: use MessageTypeSystem
)

// MessageRole represents the role of the message sender
type MessageRole string

const (
	MessageRoleSystem    MessageRole = "system"
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
)

// AssistantMessageSubType represents the subtype of assistant messages
type AssistantMessageSubType string

const (
	AssistantMessageSubTypeAudio AssistantMessageSubType = "audio"
	AssistantMessageSubTypeText  AssistantMessageSubType = "text"
)

// Metadata contains additional information about the message
type Metadata struct {
	EventType  string                  `json:"eventType"`
	ToolName   string                  `json:"toolName,omitempty"`
	Speaker    int                     `json:"speaker,omitempty"`
	SampleRate int                     `json:"sampleRate,omitempty"`
	Duration   float64                 `json:"duration,omitempty"`
	SessionID  string                  `json:"sessionId,omitempty"`
	AgentID    *int                    `json:"agentId,omitempty"` // Agent PID associated with this message
	Role       MessageRole             `json:"role,omitempty"`
	SubType    AssistantMessageSubType `json:"subType,omitempty"` // For assistant messages
}

// WebSocketConnectionState represents the current state of the WebSocket connection
type WebSocketConnectionState struct {
	Mode     string `json:"mode"`               // "timeline" or "agent"
	AgentPID *int   `json:"agentPid,omitempty"` // Agent PID when in agent mode
}

// ClientMessage represents a generic message from the client
type ClientMessage struct {
	Type      MessageType               `json:"type"`
	SessionID string                    `json:"sessionId"` // Session ID for the message
	Text      string                    `json:"text,omitempty"`
	Action    string                    `json:"action,omitempty"`    // For confirm_response: "permit" or "deny"
	MessageID string                    `json:"messageId,omitempty"` // For confirm_response: ID of the permission request message
	State     *WebSocketConnectionState `json:"state,omitempty"`     // For update_state: new state
	Timestamp string                    `json:"timestamp"`
}

// ChatMessage represents a unified message structure for chat display
type ChatMessage struct {
	Type      MessageType             `json:"type"`
	ID        string                  `json:"id"`
	Role      MessageRole             `json:"role"`
	Text      string                  `json:"text"`
	AudioData string                  `json:"audioData,omitempty"` // Base64 encoded WAV data for audio messages
	SubType   AssistantMessageSubType `json:"subType,omitempty"`   // For assistant messages
	Priority  int                     `json:"priority"`
	Timestamp time.Time               `json:"timestamp"`
	Metadata  Metadata                `json:"metadata,omitempty"`
}
