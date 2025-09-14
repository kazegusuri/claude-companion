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
	EventTypeResume       = "resume"
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

// NotificationMessageContent represents the content of a notification message
type NotificationMessageContent interface {
	isNotificationMessageContent()
}

// NotificationPermissionMessage represents a tool permission request notification
type NotificationPermissionMessage struct {
	ToolName  string `json:"tool_name"`
	MCPServer string `json:"mcp_server,omitempty"` // For MCP tools
	Operation string `json:"operation,omitempty"`  // For MCP operations
}

// isNotificationMessageContent implements NotificationMessageContent interface
func (n *NotificationPermissionMessage) isNotificationMessageContent() {}

// NotificationGeneralMessage represents a general notification message
type NotificationGeneralMessage struct {
	Text string `json:"text"`
}

// isNotificationMessageContent implements NotificationMessageContent interface
func (n *NotificationGeneralMessage) isNotificationMessageContent() {}

// NotificationEvent represents a notification event from the hook log
type NotificationEvent struct {
	Session            Session                    `json:"session"`
	HookEventName      string                     `json:"hook_event_name"`
	RawMessage         string                     `json:"raw_message"`       // Original message string
	Message            NotificationMessageContent `json:"message,omitempty"` // Parsed message content
	Trigger            string                     `json:"trigger"`
	CustomInstructions string                     `json:"custom_instructions"`
	Source             string                     `json:"source"` // For SessionStart events: startup, clear, resume
	Narration          *NarrationMessage          `json:"narration,omitempty"`
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

// AssistantMessageContentToolUseInput is the interface for tool-specific input
type AssistantMessageContentToolUseInput interface {
	isToolUseInput()
}

// AssistantMessageContentToolUse represents tool use content in an assistant message
type AssistantMessageContentToolUse struct {
	Type      string                              `json:"type"` // "tool_use"
	ID        string                              `json:"id"`
	Name      string                              `json:"name"`
	Input     AssistantMessageContentToolUseInput `json:"input"`
	Narration *NarrationMessage                   `json:"narration,omitempty"` // Narration for this tool use
}

// isAssistantMessageContentItem implements AssistantMessageContentItem interface
func (a *AssistantMessageContentToolUse) isAssistantMessageContentItem() {}

// ToolUseTodoWrite represents input for TodoWrite tool
type ToolUseTodoWrite struct {
	Todos []TodoItem `json:"todos"`
}

// TodoItem represents a single todo item
type TodoItem struct {
	Content    string `json:"content"`
	Status     string `json:"status"` // "pending", "in_progress", "completed"
	ActiveForm string `json:"activeForm,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseTodoWrite) isToolUseInput() {}

// ToolUseBash represents input for Bash tool
type ToolUseBash struct {
	Command         string `json:"command"`
	Description     string `json:"description,omitempty"`
	RunInBackground bool   `json:"run_in_background,omitempty"`
	Timeout         int    `json:"timeout,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseBash) isToolUseInput() {}

// ToolUseRead represents input for Read tool
type ToolUseRead struct {
	FilePath string `json:"file_path"`
	Limit    int    `json:"limit,omitempty"`
	Offset   int    `json:"offset,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseRead) isToolUseInput() {}

// ToolUseWrite represents input for Write tool
type ToolUseWrite struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseWrite) isToolUseInput() {}

// ToolUseEdit represents input for Edit tool
type ToolUseEdit struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseEdit) isToolUseInput() {}

// ToolUseMultiEdit represents input for MultiEdit tool
type ToolUseMultiEdit struct {
	FilePath string     `json:"file_path"`
	Edits    []EditItem `json:"edits"`
}

// EditItem represents a single edit operation
type EditItem struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseMultiEdit) isToolUseInput() {}

// ToolUseGrep represents input for Grep tool
type ToolUseGrep struct {
	Pattern         string `json:"pattern"`
	Path            string `json:"path,omitempty"`
	Glob            string `json:"glob,omitempty"`
	Type            string `json:"type,omitempty"`
	OutputMode      string `json:"output_mode,omitempty"`
	ContextAfter    int    `json:"-A,omitempty"`
	ContextBefore   int    `json:"-B,omitempty"`
	Context         int    `json:"-C,omitempty"`
	CaseInsensitive bool   `json:"-i,omitempty"`
	ShowLineNumbers bool   `json:"-n,omitempty"`
	HeadLimit       int    `json:"head_limit,omitempty"`
	Multiline       bool   `json:"multiline,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseGrep) isToolUseInput() {}

// ToolUseGlob represents input for Glob tool
type ToolUseGlob struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseGlob) isToolUseInput() {}

// ToolUseTask represents input for Task tool
type ToolUseTask struct {
	Description  string `json:"description"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseTask) isToolUseInput() {}

// ToolUseWebFetch represents input for WebFetch tool
type ToolUseWebFetch struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseWebFetch) isToolUseInput() {}

// ToolUseWebSearch represents input for WebSearch tool
type ToolUseWebSearch struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitempty"`
	BlockedDomains []string `json:"blocked_domains,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseWebSearch) isToolUseInput() {}

// ToolUseNotebookEdit represents input for NotebookEdit tool
type ToolUseNotebookEdit struct {
	NotebookPath string `json:"notebook_path"`
	CellID       string `json:"cell_id,omitempty"`
	CellType     string `json:"cell_type,omitempty"`
	EditMode     string `json:"edit_mode,omitempty"`
	NewSource    string `json:"new_source"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseNotebookEdit) isToolUseInput() {}

// ToolUseExitPlanMode represents input for ExitPlanMode tool
type ToolUseExitPlanMode struct {
	Plan string `json:"plan"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseExitPlanMode) isToolUseInput() {}

// ToolUseBashOutput represents input for BashOutput tool
type ToolUseBashOutput struct {
	BashID string `json:"bash_id"`
	Filter string `json:"filter,omitempty"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseBashOutput) isToolUseInput() {}

// ToolUseKillBash represents input for KillBash tool
type ToolUseKillBash struct {
	ShellID string `json:"shell_id"`
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseKillBash) isToolUseInput() {}

// ToolUseMCP represents input for MCP (Model Context Protocol) tools
type ToolUseMCP struct {
	Server string                 `json:"server,omitempty"` // MCP server name
	Tool   string                 `json:"tool,omitempty"`   // MCP tool name
	Data   map[string]interface{} `json:"-"`                // Captures all MCP-specific fields
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseMCP) isToolUseInput() {}

// ToolUseGeneric represents generic tool input for unknown tools
type ToolUseGeneric struct {
	Data map[string]interface{} `json:"-"` // Captures all fields
}

// isToolUseInput implements AssistantMessageContentToolUseInput interface
func (t *ToolUseGeneric) isToolUseInput() {}

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

// SubagentTask contains information about a subagent execution
type SubagentTask struct {
	UUID       string `json:"uuid"`        // UUID of the first subagent event
	EventCount int    `json:"event_count"` // Number of events in the subagent thread
}

// TaskCompletionMessage represents a task completion event
type TaskCompletionMessage struct {
	Session      Session           `json:"session"`
	TaskInfo     TaskInfo          `json:"task_info"`
	SubagentTask *SubagentTask     `json:"subagent_task,omitempty"` // Associated subagent execution (nil if no match)
	Timestamp    time.Time         `json:"timestamp"`
	Narration    *NarrationMessage `json:"narration,omitempty"`
}

// Type returns the event type
func (e *TaskCompletionMessage) Type() Type {
	return EventTypeTaskComplete
}

// ResumeEvent represents a resume event when buffered events are released
type ResumeEvent struct {
	Session       Session           `json:"session"`
	ResumedFromID string            `json:"resumed_from_id"` // The sessionID that was being resumed from
	BufferedCount int               `json:"buffered_count"`  // Number of events that were buffered
	Timestamp     time.Time         `json:"timestamp"`
	Reason        string            `json:"reason"` // Reason for release (e.g., "SessionStart:resume received")
	Narration     *NarrationMessage `json:"narration,omitempty"`
}

// Type returns the event type
func (e *ResumeEvent) Type() Type {
	return EventTypeResume
}

// Printer is the interface for printing events
type Printer interface {
	Print(event interface{})
}
