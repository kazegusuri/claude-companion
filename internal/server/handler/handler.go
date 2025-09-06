package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"time"

	"github.com/google/uuid"
)

// ClaudeCommand represents the command types for claude-code-send
type ClaudeCommand string

const (
	ClaudeCommandSend    ClaudeCommand = "send"
	ClaudeCommandProceed ClaudeCommand = "proceed"
	ClaudeCommandStop    ClaudeCommand = "stop"

	// ClaudeCodeSendPath is the path to the claude-code-send command
	ClaudeCodeSendPath = "/usr/local/bin/claude-code-send"
)

// MessageSender defines the interface for sending messages
type MessageSender interface {
	// Send sends a message to the specific client
	Send(msg *ChatMessage) error
	// Broadcast sends a message to all connected clients
	Broadcast(msg *ChatMessage)
}

// MessageEmitter defines the interface for sending messages
type MessageEmitter interface {
	// BroadcastChat sends a chat message to all connected clients
	BroadcastChat(msg *ChatMessage)
}

// ClaudeCommandRequest represents the request to execute claude-code-send command
type ClaudeCommandRequest struct {
	CWD       string        `json:"cwd"`
	SessionID string        `json:"sessionId"`
	Command   ClaudeCommand `json:"command"`
	Message   string        `json:"message"`
}

// ClaudeCommandResponse represents the response from claude-code-send command
type ClaudeCommandResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

// EventHandler handles different types of messages from WebSocket clients
type EventHandler struct {
	clientID      string
	sender        MessageSender
	sessionGetter SessionGetter
}

// NewEventHandler creates a new EventHandler
func NewEventHandler(clientID string, sender MessageSender, sessionGetter SessionGetter) *EventHandler {
	return &EventHandler{
		clientID:      clientID,
		sender:        sender,
		sessionGetter: sessionGetter,
	}
}

// HandleMessage routes messages to appropriate handlers based on type
func (h *EventHandler) HandleMessage(message ClientMessage) {
	switch message.Type {
	case MessageTypePing:
		h.HandlePing(message)
	case MessageTypeUserMessage:
		h.HandleUserMessage(message)
	case MessageTypeConfirmResponse:
		h.HandleConfirmResponse(message)
	default:
		// Unhandled message type
	}
}

// HandlePing handles ping messages from the client
func (h *EventHandler) HandlePing(message ClientMessage) {
	pong := &ChatMessage{
		Type:      MessageTypePong,
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
	}
	// Send pong directly to the client (not broadcast)
	if err := h.sender.Send(pong); err != nil {
		log.Printf("Failed to send pong: %v", err)
	}
}

// HandleUserMessage handles user messages from the client
func (h *EventHandler) HandleUserMessage(message ClientMessage) {

	// Get session from SessionGetter
	if h.sessionGetter == nil {
		return
	}

	session, exists := h.sessionGetter.GetSession(message.SessionID)
	if !exists {
		return
	}

	// Execute claude-code-send command asynchronously
	req := &ClaudeCommandRequest{
		Command:   ClaudeCommandSend,
		Message:   message.Text,
		CWD:       session.CWD,
		SessionID: message.SessionID,
	}
	go h.executeClaudeCommand(req)
}

// HandleConfirmResponse handles permission confirmation responses from the client
func (h *EventHandler) HandleConfirmResponse(message ClientMessage) {

	// Get session from SessionGetter
	if h.sessionGetter == nil {
		return
	}

	session, exists := h.sessionGetter.GetSession(message.SessionID)
	if !exists {
		return
	}

	// Execute claude-code-send command based on action
	var command ClaudeCommand
	switch message.Action {
	case "permit":
		command = ClaudeCommandProceed
	case "deny":
		command = ClaudeCommandStop
	default:
		return
	}

	// Execute command asynchronously
	req := &ClaudeCommandRequest{
		Command:   command,
		Message:   fmt.Sprintf("Permission %s by user", message.Action),
		CWD:       session.CWD,
		SessionID: message.SessionID,
	}
	go h.executeClaudeCommand(req)

	// Send feedback message to the requesting client only
	feedbackMsg := &ChatMessage{
		Type:      MessageTypeSystem,
		Role:      MessageRoleSystem,
		ID:        uuid.New().String(),
		Text:      fmt.Sprintf("Permission %s - executing %s command", message.Action, command),
		Timestamp: time.Now(),
		Metadata: Metadata{
			EventType: "permission_response",
		},
	}
	if err := h.sender.Send(feedbackMsg); err != nil {
		log.Printf("Failed to send feedback to client %s: %v", h.clientID, err)
	}
}

// executeClaudeCommand executes the claude-code-send command
func (h *EventHandler) executeClaudeCommand(req *ClaudeCommandRequest) {
	// Prepare the command request for JSON marshaling
	request := struct {
		CWD       string `json:"cwd"`
		SessionID string `json:"sessionId"`
		Command   string `json:"command"`
		Message   string `json:"message"`
	}{
		CWD:       req.CWD,
		SessionID: req.SessionID,
		Command:   string(req.Command),
		Message:   req.Message,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		log.Printf("Error marshaling JSON for claude-code-send: %v", err)
		return
	}

	// Execute the command
	cmd := exec.Command(ClaudeCodeSendPath)
	cmd.Stdin = bytes.NewReader(jsonData)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing claude-code-send (%s): %v", req.Command, err)

		// Send error message to client
		errorMsg := &ChatMessage{
			Type:      MessageTypeSystem,
			Role:      MessageRoleSystem,
			ID:        uuid.New().String(),
			Text:      fmt.Sprintf("Failed to execute %s command: %v", req.Command, err),
			Timestamp: time.Now(),
			Metadata: Metadata{
				EventType: "command_error",
			},
		}
		if err := h.sender.Send(errorMsg); err != nil {
			log.Printf("Failed to send error message to client %s: %v", h.clientID, err)
		}
		return
	}

	// Parse the response
	var response ClaudeCommandResponse
	if err := json.Unmarshal(output, &response); err != nil {
		log.Printf("Error parsing claude-code-send response: %v", err)
		return
	}

	// Check the response
	if response.Success {

		// Send success message to client
		successMsg := &ChatMessage{
			Type:      MessageTypeSystem,
			Role:      MessageRoleSystem,
			ID:        uuid.New().String(),
			Text:      fmt.Sprintf("%s command executed: %s", req.Command, response.Message),
			Timestamp: time.Now(),
			Metadata: Metadata{
				EventType: "command_success",
			},
		}
		if err := h.sender.Send(successMsg); err != nil {
			log.Printf("Failed to send success message to client %s: %v", h.clientID, err)
		}
	} else {

		// Send error message to client
		errorMsg := &ChatMessage{
			Type:      MessageTypeSystem,
			Role:      MessageRoleSystem,
			ID:        uuid.New().String(),
			Text:      fmt.Sprintf("%s command failed: %s", req.Command, response.Error),
			Timestamp: time.Now(),
			Metadata: Metadata{
				EventType: "command_error",
			},
		}
		if err := h.sender.Send(errorMsg); err != nil {
			log.Printf("Failed to send error message to client %s: %v", h.clientID, err)
		}
	}
}
