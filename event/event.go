package event

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Type represents the type of event
type Type string

// Event is the common interface for all events
type Event interface {
	Type() Type
}

// EventMeta contains metadata about the event context
type EventMeta struct {
	EventID   string // Unique identifier for the event
	SessionID string // Session identifier
	ToolID    string
	CWD       string
	Timestamp time.Time // When the event was created
}

// NewEventMeta creates a new EventMeta with a generated UUID and timestamp
func NewEventMeta(toolID, cwd string) EventMeta {
	return EventMeta{
		EventID:   uuid.New().String(),
		SessionID: uuid.New().String(), // Generate a new session ID
		ToolID:    toolID,
		CWD:       cwd,
		Timestamp: time.Now(),
	}
}

// EventSender is an interface for sending events
type EventSender interface {
	SendEvent(event Event)
}

// EventType constants
const (
	EventTypeUser         = "user"
	EventTypeAssistant    = "assistant"
	EventTypeSystem       = "system"
	EventTypeSummary      = "summary"
	EventTypeNotification = "notification"
)

// SessionFile represents the project and session information extracted from the log file path
type SessionFile struct {
	Path    string `json:"path"` // Full path to the session file
	Project string `json:"project"`
	Session string `json:"session"`
}

// BaseEvent contains common fields for all event types
type BaseEvent struct {
	ParentUUID  *string      `json:"parentUuid"`
	IsSidechain bool         `json:"isSidechain"`
	UserType    string       `json:"userType"`
	CWD         string       `json:"cwd"`
	SessionID   string       `json:"sessionId"`
	Session     *SessionFile `json:"session,omitempty"`
	Version     string       `json:"version"`
	GitBranch   string       `json:"gitBranch"`
	UUID        string       `json:"uuid"`
	Timestamp   time.Time    `json:"timestamp"`
	TypeString  string       `json:"type"`
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
	IsMeta  bool               `json:"isMeta,omitempty"`
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

// TaskCompletionMessage represents the completion of a Task tool execution
type TaskCompletionMessage struct {
	BaseEvent
	TaskInfo TaskInfo
}

// Type returns the event type
func (e *TaskCompletionMessage) Type() Type {
	return Type("task_completion")
}

// HookEvent represents a hook execution event from Claude
type HookEvent struct {
	BaseEvent
	Content   string `json:"content"`
	IsMeta    bool   `json:"isMeta"`
	ToolUseID string `json:"toolUseID"`
	Level     string `json:"level"`

	// Parsed fields from content
	HookName      string
	HookCommand   string
	HookStatus    string
	HookEventType string // SessionStart:resume, Stop, etc.
}

// ParseHookContent parses the content field to extract hook information
func (h *HookEvent) ParseHookContent() error {
	// Remove ANSI escape codes
	cleanContent := stripANSI(h.Content)

	// Pattern 1: "SessionStart:resume [/usr/local/bin/claude-notification.sh] completed successfully"
	// Pattern 2: "Stop [/usr/local/bin/claude-notification.sh] completed successfully"
	pattern := regexp.MustCompile(`^(\w+(?::\w+)?)\s+\[([^\]]+)\]\s+(.+)$`)
	matches := pattern.FindStringSubmatch(cleanContent)

	if len(matches) == 4 {
		h.HookEventType = matches[1]
		h.HookCommand = matches[2]
		h.HookStatus = matches[3]

		// Extract hook name from command path
		parts := strings.Split(h.HookCommand, "/")
		if len(parts) > 0 {
			h.HookName = parts[len(parts)-1]
		}

		return nil
	}

	return fmt.Errorf("unable to parse hook content: %s", cleanContent)
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(str string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(str, "")
}

// extractSessionFromPath extracts project and session information from a log file path
// Expected format: {project}/{session}.jsonl
func extractSessionFromPath(path string) *SessionFile {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Extract the directory and filename
	dir := filepath.Dir(cleanPath)
	filename := filepath.Base(cleanPath)

	// Remove .jsonl extension if present
	if strings.HasSuffix(filename, ".jsonl") {
		filename = strings.TrimSuffix(filename, ".jsonl")
	}

	// Extract project name from the parent directory
	projectDir := filepath.Base(dir)

	// Return session info from parent directory and filename
	return &SessionFile{
		Path:    cleanPath,
		Project: projectDir,
		Session: filename,
	}
}
