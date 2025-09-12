package event

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// BufferInfo holds information about buffered events for a session
type BufferInfo struct {
	events      []Event
	timer       *time.Timer
	sessionName string
	startTime   time.Time
}

// FormatterInterface defines the interface for event formatters
type FormatterInterface interface {
	Format(event Event) (string, error)
	SetDebugMode(debug bool)
}

// CentralEventHandler is the interface for central event handler
type CentralEventHandler interface {
	SendEvent(event internalevent.Event)
}

// Handler processes events from multiple sources
type Handler struct {
	narrator       narrator.Narrator
	formatter      FormatterInterface
	eventChan      chan Event
	wg             sync.WaitGroup
	done           chan struct{}
	taskTracker    *TaskTracker
	sessionManager *handler.SessionManager
	centralHandler CentralEventHandler // Central event handler

	// Buffering support
	bufferMutex sync.Mutex
	buffers     map[string]*BufferInfo // key: session name
}

// NewHandler creates a new event handler
func NewHandler(narrator narrator.Narrator, sessionManager *handler.SessionManager, centralHandler CentralEventHandler) *Handler {
	formatter := NewFormatter(narrator)
	taskTracker := NewTaskTracker()

	return &Handler{
		narrator:       narrator,
		formatter:      formatter,
		eventChan:      make(chan Event, 100),
		done:           make(chan struct{}),
		taskTracker:    taskTracker,
		sessionManager: sessionManager,
		centralHandler: centralHandler,
		buffers:        make(map[string]*BufferInfo),
	}
}

// GetFormatter returns the handler's formatter
func (h *Handler) GetFormatter() *Formatter {
	if formatter, ok := h.formatter.(*Formatter); ok {
		return formatter
	}
	return nil
}

// GetSessionManager returns the handler's session manager
func (h *Handler) GetSessionManager() *handler.SessionManager {
	return h.sessionManager
}

// Start begins processing events
func (h *Handler) Start() {
	h.wg.Add(1)
	go h.processEvents()
}

// Stop stops the event handler
func (h *Handler) Stop() {
	close(h.done)
	// Don't close eventChan here to prevent panic on send
	// The processEvents goroutine will exit when done is closed
	h.wg.Wait()
}

// SendEvent sends an event to be processed
func (h *Handler) SendEvent(event Event) {
	select {
	case <-h.done:
		// Handler is stopping, discard event
		return
	default:
		// Try to send event
		select {
		case h.eventChan <- event:
		case <-h.done:
			// Handler stopped while sending, discard event
		}
	}
}

// HandleWarmupEvent processes warmup events to initialize session state
func (h *Handler) HandleWarmupEvent(event *BaseEvent) {
	// Extract session information from the base event
	if event.ParentUUID != nil && !event.IsSidechain && event.SessionID != "" && h.sessionManager != nil {
		// Get or create session
		session, exists := h.sessionManager.GetSession(event.SessionID)
		if !exists {
			h.sessionManager.CreateSession(event.SessionID, event.UUID, event.CWD, event.Session.Path)
			// If session doesn't exist during warmup, we might want to create it
			// but for now, just log it
			logger.DebugInfo("Warmup: Session %s not found for event type %s", event.SessionID, event.TypeString)
		} else {
			// Update session with warmup information if needed
			logger.DebugInfo("Warmup: Processing event type %s for session %s (CWD: %s)", event.TypeString, event.SessionID, session.CWD)
		}
	}

	// For warmup, we don't process the event through the normal pipeline
	// This is just to initialize state
}

// processEvents processes events from the channel
func (h *Handler) processEvents() {
	defer h.wg.Done()

	for {
		select {
		case event, ok := <-h.eventChan:
			if !ok {
				return
			}
			h.processEvent(event)
		case <-h.done:
			// Drain remaining events
			for {
				select {
				case event, ok := <-h.eventChan:
					if !ok {
						return
					}
					h.processEvent(event)
				default:
					return
				}
			}
		}
	}
}

// processEvent processes a single event based on its type
func (h *Handler) processEvent(event Event) {
	// Check if event should be buffered or if it releases buffered events
	if h.handleBuffering(event) {
		return // Event was buffered or handled
	}

	// Check if the event should be ignored (sidechain events)
	switch e := event.(type) {
	case *UserMessage:
		if e.IsSidechain {
			logger.DebugInfo("Ignoring sidechain UserMessage")
			return
		}
	case *AssistantMessage:
		if e.IsSidechain {
			logger.DebugInfo("Ignoring sidechain AssistantMessage")
			return
		}
	case *SystemMessage:
		if e.IsSidechain {
			logger.DebugInfo("Ignoring sidechain SystemMessage")
			return
		}
	case *HookEvent:
		if e.IsSidechain {
			logger.DebugInfo("Ignoring sidechain HookEvent")
			return
		}
	case *BaseEvent:
		if e.IsSidechain {
			logger.DebugInfo("Ignoring sidechain BaseEvent")
			return
		}
	}

	switch e := event.(type) {
	case *NotificationEvent:
		// Convert to internal/event.NotificationEvent and forward to central handler
		centralEvent := &internalevent.NotificationEvent{
			Session: internalevent.Session{
				SessionID:      e.SessionID,
				TranscriptPath: e.TranscriptPath,
			},
			HookEventName:      e.HookEventName,
			Message:            e.Message,
			Trigger:            e.Trigger,
			CustomInstructions: e.CustomInstructions,
			Source:             e.Source,
		}
		if h.centralHandler != nil {
			h.centralHandler.SendEvent(centralEvent)
		}
		// NotificationEvent display is now handled by central handler's printer
	case *AssistantMessage:
		// Track Task tool uses
		h.trackTaskToolUses(e)

		// Convert and send to central handler
		if h.centralHandler != nil {
			centralAssistant := h.convertAssistantMessage(e)
			if centralAssistant != nil {
				h.centralHandler.SendEvent(centralAssistant)
			}

			// For tool_use content, still use formatter for display
			for _, content := range e.Message.Content {
				if content.Type == "tool_use" {
					// Format only tool_use content
					toolUseMsg := &AssistantMessage{
						BaseEvent: e.BaseEvent,
						RequestID: e.RequestID,
						Message: AssistantMessageContent{
							ID:           e.Message.ID,
							Type:         e.Message.Type,
							Role:         e.Message.Role,
							Model:        e.Message.Model,
							Content:      []AssistantContent{content},
							StopReason:   e.Message.StopReason,
							StopSequence: e.Message.StopSequence,
							Usage:        e.Message.Usage,
						},
						IsApiErrorMessage: false,
					}

					output, err := h.formatter.Format(toolUseMsg)
					if err != nil {
						logger.LogError("Error formatting tool_use: %v", err)
						continue
					}
					if output != "" {
						fmt.Print(output)
					}
				}
			}
		} else {
			// Fallback to local formatting if no central handler
			// Only format tool_use content types
			hasNonToolUse := false
			for _, content := range e.Message.Content {
				if content.Type != "tool_use" {
					hasNonToolUse = true
					break
				}
			}

			if hasNonToolUse {
				logger.LogWarning("Cannot display non-tool_use content without central handler")
			}

			// Format tool_use content
			for _, content := range e.Message.Content {
				if content.Type == "tool_use" {
					toolUseMsg := &AssistantMessage{
						BaseEvent: e.BaseEvent,
						RequestID: e.RequestID,
						Message: AssistantMessageContent{
							ID:           e.Message.ID,
							Type:         e.Message.Type,
							Role:         e.Message.Role,
							Model:        e.Message.Model,
							Content:      []AssistantContent{content},
							StopReason:   e.Message.StopReason,
							StopSequence: e.Message.StopSequence,
							Usage:        e.Message.Usage,
						},
						IsApiErrorMessage: false,
					}

					output, err := h.formatter.Format(toolUseMsg)
					if err != nil {
						logger.LogError("Error formatting tool_use: %v", err)
						continue
					}
					if output != "" {
						fmt.Print(output)
					}
				}
			}
		}
	case *UserMessage:
		// Check if this is a Task result and create TaskCompletionMessage
		if taskCompletion := h.checkTaskResultFromUser(e); taskCompletion != nil {
			// Send TaskCompletionMessage to central handler
			if h.centralHandler != nil {
				// Get session info from the event
				sessionID := ""
				transcriptPath := ""
				if e.SessionID != "" {
					sessionID = e.SessionID
				}

				centralTaskCompletion := &internalevent.TaskCompletionMessage{
					Session: internalevent.Session{
						SessionID:      sessionID,
						TranscriptPath: transcriptPath,
					},
					TaskInfo: internalevent.TaskInfo{
						ToolUseID:    taskCompletion.TaskInfo.ToolUseID,
						Description:  taskCompletion.TaskInfo.Description,
						SubagentType: taskCompletion.TaskInfo.SubagentType,
					},
					Timestamp: taskCompletion.Timestamp,
				}
				h.centralHandler.SendEvent(centralTaskCompletion)
			}
		}

		// Convert to internal/event.UserMessage and send to central handler
		var content internalevent.UserMessageContent

		switch c := e.Message.Content.(type) {
		case string:
			// Parse the string to check for special formats
			content = h.parseUserMessageContent(c)
		case []interface{}:
			// Handle array content - create UserMessageContentList
			content = h.parseUserMessageContentArray(c)
		}

		// Only send if we successfully parsed content
		if content != nil {
			centralEvent := &internalevent.UserMessage{
				SessionMessageBase: internalevent.SessionMessageBase{
					UUID:        e.UUID,
					Type:        internalevent.MessageTypeUser,
					IsSidechain: e.IsSidechain,
					CWD:         e.CWD,
					Timestamp:   e.Timestamp,
					IsMeta:      e.IsMeta,
				},
				Session: internalevent.Session{
					SessionID:      e.SessionID,
					TranscriptPath: "", // UserMessage doesn't have TranscriptPath
				},
				Message: internalevent.UserMessageData{
					Role:    "user",
					Content: content,
				},
				// Copy optional fields from the parsed event
				ToolUseResult: e.ToolUseResult,
			}
			if h.centralHandler != nil {
				h.centralHandler.SendEvent(centralEvent)
			}
		}

		// UserMessage is now handled by central event handler
		// WebSocket broadcast is done in central handler for UserMessageContentMessage
	case *HookEvent:
		// Handle SessionStart event
		if e.HookEventType == "SessionStart" {
			// Use SessionFile.Path as TranscriptPath
			transcriptPath := ""
			if e.Session != nil {
				transcriptPath = e.Session.Path
			}
			// Create a new session
			h.sessionManager.CreateSession(e.SessionID, e.UUID, e.CWD, transcriptPath)
		}

		// Convert HookEvent to SystemMessage and send to central handler
		transcriptPath := ""
		if e.Session != nil {
			transcriptPath = e.Session.Path
		}

		// Parse hook type and status from HookEventType (e.g., "SessionStart:resume" or "Stop")
		hookName := e.HookEventType
		hookType := ""
		if colonIdx := strings.Index(e.HookEventType, ":"); colonIdx > 0 {
			hookName = e.HookEventType[:colonIdx]
			hookType = e.HookEventType[colonIdx+1:]
		}

		centralEvent := &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        e.UUID,
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: e.IsSidechain,
				CWD:         e.CWD,
				Timestamp:   e.Timestamp,
				IsMeta:      e.IsMeta,
			},
			Session: internalevent.Session{
				SessionID:      e.SessionID,
				TranscriptPath: transcriptPath,
			},
			RawContent: e.Content,
			Content: &internalevent.HookSystemMessageContent{
				HookName: hookName,
				Command:  e.HookCommand,
				Status:   e.HookStatus,
				Type:     hookType,
				Message:  e.Content,
			},
			Level: "info", // Default level for hook events
		}

		if h.centralHandler != nil {
			h.centralHandler.SendEvent(centralEvent)
		}
	case *SystemMessage:
		// Convert to internal/event.SystemMessage and forward to central handler
		// Get transcript path from session if available
		transcriptPath := ""
		if e.Session != nil {
			transcriptPath = e.Session.Path
		}
		centralEvent := &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        e.UUID,
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: e.IsSidechain,
				CWD:         e.CWD,
				Timestamp:   e.Timestamp,
				IsMeta:      e.IsMeta,
			},
			Session: internalevent.Session{
				SessionID:      e.SessionID,
				TranscriptPath: transcriptPath,
			},
			RawContent: e.Content,
			Level:      e.Level,
			ToolUseID:  e.ToolUseID,
		}
		if h.centralHandler != nil {
			h.centralHandler.SendEvent(centralEvent)
		}
		// SystemMessage display is now handled by central handler's printer
	case *SummaryEvent:
		// Convert to internal/event.SummaryEvent and forward to central handler
		// Get SessionID and TranscriptPath from Session if available
		sessionID := ""
		transcriptPath := ""
		if e.Session != nil {
			sessionID = e.Session.Session
			transcriptPath = e.Session.Path
		}
		centralEvent := &internalevent.SummaryEvent{
			Session: internalevent.Session{
				SessionID:      sessionID,
				TranscriptPath: transcriptPath,
			},
			LeafUUID: e.LeafUUID,
			Summary:  e.Summary,
		}
		if h.centralHandler != nil {
			h.centralHandler.SendEvent(centralEvent)
		}
		// SummaryEvent display is now handled by central handler's printer
	case *BaseEvent, *TaskCompletionMessage:
		// Format and display parsed events
		output, err := h.formatter.Format(e)
		if err != nil {
			logger.LogError("Error formatting %T: %v", e, err)
			return
		}
		if output != "" {
			fmt.Print(output)
		}
	default:
		logger.DebugWarning("Unknown event type: %T", event)
	}
}

// trackTaskToolUses tracks Task tool uses from AssistantMessage
func (h *Handler) trackTaskToolUses(msg *AssistantMessage) {
	for _, content := range msg.Message.Content {
		if content.Type == "tool_use" && content.Name == "Task" {
			// Extract Task parameters from input
			if inputMap, ok := content.Input.(map[string]interface{}); ok {
				description := ""
				subagentType := ""

				if desc, ok := inputMap["description"].(string); ok {
					description = desc
				}
				if agent, ok := inputMap["subagent_type"].(string); ok {
					subagentType = agent
				}

				// Track the Task execution
				h.taskTracker.TrackTask(content.ID, description, subagentType)

				logger.DebugInfo("Tracking Task: ID=%s, Description=%s, Agent=%s",
					content.ID, description, subagentType)
			}
		}
	}
}

// checkTaskResultFromUser checks if a UserMessage contains Task results and creates TaskCompletionMessage
func (h *Handler) checkTaskResultFromUser(msg *UserMessage) *TaskCompletionMessage {
	// Check if content is an array (tool results are in array format)
	contentArray, ok := msg.Message.Content.([]interface{})
	if !ok {
		return nil
	}

	// Look for tool_result items
	for _, item := range contentArray {
		if contentMap, ok := item.(map[string]interface{}); ok {
			if contentType, ok := contentMap["type"].(string); ok && contentType == "tool_result" {
				if toolUseID, ok := contentMap["tool_use_id"].(string); ok {
					if taskInfo, exists := h.taskTracker.GetTask(toolUseID); exists {
						// This is a Task result
						h.taskTracker.RemoveTask(toolUseID)

						// Create TaskCompletionMessage
						taskCompletion := &TaskCompletionMessage{
							BaseEvent: msg.BaseEvent, // Use BaseEvent from UserMessage
							TaskInfo:  taskInfo,
						}

						logger.DebugInfo("Task completed: ID=%s, Description=%s, Agent=%s",
							toolUseID, taskInfo.Description, taskInfo.SubagentType)

						return taskCompletion
					}
				}
			}
		}
	}
	return nil
}

// handleBuffering checks if an event should be buffered or if it releases buffered events
// Returns true if the event was handled (buffered or triggered release)
//
// Resume handling:
// When the resume command is executed, past events from the resumed session are passed through,
// so we need to ignore these historical events.
// When an event with null ParentUUID is received mid-stream, it's identified as a resume event.
// Events between the resume detection and the SessionStart event generated by the resume are ignored.
func (h *Handler) handleBuffering(event Event) bool {
	// Extract BaseEvent from different event types
	var baseEvent *BaseEvent

	switch e := event.(type) {
	case *UserMessage:
		baseEvent = &e.BaseEvent
	case *AssistantMessage:
		baseEvent = &e.BaseEvent
	case *SystemMessage:
		baseEvent = &e.BaseEvent
	case *HookEvent:
		baseEvent = &e.BaseEvent
		// Check if this is a SessionStart:resume event FIRST before buffering check
		if e.HookEventType == "SessionStart:resume" && baseEvent.Session != nil {
			sessionName := baseEvent.Session.Session
			// Only release buffer if session name matches SessionID
			if sessionName == baseEvent.SessionID {
				h.releaseBuffer(sessionName, "SessionStart:resume received")
				// Process this event normally after releasing buffer
				return false
			}
			// If session name doesn't match, continue to buffering check
		}
	case *BaseEvent:
		baseEvent = e
	case *TaskCompletionMessage:
		baseEvent = &e.BaseEvent
	default:
		// Event doesn't have BaseEvent (e.g., NotificationEvent, SummaryEvent)
		return false
	}

	// Get session name if we have it
	if baseEvent == nil || baseEvent.Session == nil {
		return false
	}
	sessionName := baseEvent.Session.Session

	// Check if we need to buffer this event
	if !baseEvent.IsSidechain && baseEvent.ParentUUID == nil {
		// Check if this is a resume scenario using SessionManager
		if h.sessionManager != nil {
			session, exists := h.sessionManager.GetSession(baseEvent.SessionID)

			// if !exists {
			// 	// Case 1: No session exists - treat as completely new
			// 	logger.DebugInfo("New session (not registered): %s", baseEvent.SessionID)
			// 	return false // Process normally
			// }
			var sessionUUID string
			if exists {
				sessionUUID = session.UUID
			}

			if exists && sessionUUID == "" || sessionUUID == baseEvent.UUID {
				// Case 2: Empty UUID or Same UUID - treat as new/normal start
				if sessionUUID == "" {
					logger.DebugInfo("Normal session start (empty UUID): %s", baseEvent.SessionID)
				} else {
					logger.DebugInfo("Normal session start (UUID match): %s", baseEvent.SessionID)
				}
				return false // Process normally
			}

			// Case 3: Different UUID - this is a resume scenario
			logger.DebugInfo("Resume detected (UUID mismatch) for session: %s, stored UUID: %s, event UUID: %s",
				baseEvent.SessionID, sessionUUID, baseEvent.UUID)

			h.bufferMutex.Lock()
			defer h.bufferMutex.Unlock()

			// Check if we already have a buffer for this session
			if buffer, exists := h.buffers[sessionName]; exists {
				// Add to existing buffer
				buffer.events = append(buffer.events, event)
				return true
			}

			// Create new buffer for this session
			buffer := &BufferInfo{
				events:      []Event{event},
				sessionName: sessionName,
				startTime:   time.Now(),
				timer: time.AfterFunc(1*time.Second, func() {
					h.releaseBuffer(sessionName, "timeout")
				}),
			}
			h.buffers[sessionName] = buffer
			return true
		}

		// Not a SessionStart event with ParentUUID=nil - might be an issue
		logger.DebugInfo("Non-SessionStart event with ParentUUID=nil: %T", event)
		return false // Process normally for backward compatibility
	}

	// Check if this event is for a buffered session
	h.bufferMutex.Lock()
	if buffer, exists := h.buffers[sessionName]; exists {
		// Add to buffer
		buffer.events = append(buffer.events, event)
		h.bufferMutex.Unlock()
		return true
	}
	h.bufferMutex.Unlock()

	return false
}

// releaseBuffer releases buffered events for a session
func (h *Handler) releaseBuffer(sessionName string, reason string) {
	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()

	buffer, exists := h.buffers[sessionName]
	if !exists {
		return
	}

	// Stop the timer if it's still running
	if buffer.timer != nil {
		buffer.timer.Stop()
	}

	logger.DebugInfo("Releasing buffer for session %s: %s (events: %d, duration: %v)",
		sessionName, reason, len(buffer.events), time.Since(buffer.startTime))

	// Remove buffer and discard buffered events
	delete(h.buffers, sessionName)

	// Buffered events are discarded (not re-enqueued)
}

// parseUserMessageContentArray parses array content and returns UserMessageContentList
func (h *Handler) parseUserMessageContentArray(contentArray []interface{}) internalevent.UserMessageContent {
	var items []internalevent.UserMessageContentItem

	for _, item := range contentArray {
		if contentMap, ok := item.(map[string]interface{}); ok {
			contentType, hasType := contentMap["type"].(string)

			if !hasType {
				// No type field - create UserMessageContentUnknown
				rawData, err := json.Marshal(contentMap)
				if err == nil {
					unknown := &internalevent.UserMessageContentUnknown{
						Data: json.RawMessage(rawData),
					}
					items = append(items, unknown)
				}
				continue
			}

			switch contentType {
			case "text":
				if text, ok := contentMap["text"].(string); ok {
					// Parse each text item as array item
					parsedContent := h.parseUserMessageContentItem(text)
					if parsedContent != nil {
						items = append(items, parsedContent)
					}
				}

			case "tool_result":
				toolResult := &internalevent.UserMessageContentToolResult{}

				// Extract tool_use_id
				if toolUseID, ok := contentMap["tool_use_id"].(string); ok {
					toolResult.ToolUseID = toolUseID
				}

				// Extract content
				if content, ok := contentMap["content"].(string); ok {
					toolResult.Content = content
				}

				// Extract is_error (optional)
				if isError, ok := contentMap["is_error"].(bool); ok {
					toolResult.IsError = isError
				}

				items = append(items, toolResult)

			default:
				// Unknown type - create UserMessageContentUnknown
				rawData, err := json.Marshal(contentMap)
				if err == nil {
					unknown := &internalevent.UserMessageContentUnknown{
						Type: contentType,
						Data: json.RawMessage(rawData),
					}
					items = append(items, unknown)
				}
			}
		}
	}

	// If we found items, return UserMessageContentList
	if len(items) > 0 {
		return &internalevent.UserMessageContentList{
			Items: items,
		}
	}

	return nil
}

// parseUserMessageContentItem parses text that appears in array and returns UserMessageContentItem
func (h *Handler) parseUserMessageContentItem(text string) internalevent.UserMessageContentItem {
	// Check for interrupted message patterns (only in arrays)
	if text == "[Request interrupted by user]" || text == "[Request interrupted by user for tool use]" {
		return &internalevent.UserMessageContentInterrupted{
			Reason: text,
		}
	}

	// Check for local command stdout pattern
	if strings.Contains(text, "<local-command-stdout>") && strings.Contains(text, "</local-command-stdout>") {
		// Extract content between tags
		start := strings.Index(text, "<local-command-stdout>")
		end := strings.Index(text, "</local-command-stdout>")
		if start != -1 && end != -1 && end > start {
			output := text[start+len("<local-command-stdout>") : end]
			// Handle special case "(no content)"
			if output == "(no content)" {
				output = ""
			}
			return &internalevent.UserMessageContentLocalCommand{
				Output: output,
			}
		}
	}

	// Check for command pattern
	if strings.Contains(text, "<command-name>") && strings.Contains(text, "</command-name>") {
		commandName := ""
		commandMessage := ""
		commandArgs := ""

		// Extract command name
		if start := strings.Index(text, "<command-name>"); start != -1 {
			if end := strings.Index(text, "</command-name>"); end != -1 && end > start {
				commandName = text[start+len("<command-name>") : end]
			}
		}

		// Extract command message
		if start := strings.Index(text, "<command-message>"); start != -1 {
			if end := strings.Index(text, "</command-message>"); end != -1 && end > start {
				commandMessage = text[start+len("<command-message>") : end]
			}
		}

		// Extract command args
		if start := strings.Index(text, "<command-args>"); start != -1 {
			if end := strings.Index(text, "</command-args>"); end != -1 && end > start {
				commandArgs = text[start+len("<command-args>") : end]
			}
		}

		return &internalevent.UserMessageContentCommand{
			CommandName:    commandName,
			CommandMessage: commandMessage,
			CommandArgs:    commandArgs,
		}
	}

	// Default to message content
	return &internalevent.UserMessageContentMessage{
		Text: text,
	}
}

// parseUserMessageContent parses the user message content and returns the appropriate type
func (h *Handler) parseUserMessageContent(text string) internalevent.UserMessageContent {

	// Check for local command stdout pattern
	if strings.Contains(text, "<local-command-stdout>") && strings.Contains(text, "</local-command-stdout>") {
		// Extract content between tags
		start := strings.Index(text, "<local-command-stdout>")
		end := strings.Index(text, "</local-command-stdout>")
		if start != -1 && end != -1 && end > start {
			output := text[start+len("<local-command-stdout>") : end]
			// Handle special case "(no content)"
			if output == "(no content)" {
				output = ""
			}
			return &internalevent.UserMessageContentLocalCommand{
				Output: output,
			}
		}
	}

	// Check for command pattern
	if strings.Contains(text, "<command-name>") && strings.Contains(text, "</command-name>") {
		commandName := ""
		commandMessage := ""
		commandArgs := ""

		// Extract command name
		if start := strings.Index(text, "<command-name>"); start != -1 {
			if end := strings.Index(text, "</command-name>"); end != -1 && end > start {
				commandName = text[start+len("<command-name>") : end]
			}
		}

		// Extract command message
		if start := strings.Index(text, "<command-message>"); start != -1 {
			if end := strings.Index(text, "</command-message>"); end != -1 && end > start {
				commandMessage = text[start+len("<command-message>") : end]
			}
		}

		// Extract command args
		if start := strings.Index(text, "<command-args>"); start != -1 {
			if end := strings.Index(text, "</command-args>"); end != -1 && end > start {
				commandArgs = text[start+len("<command-args>") : end]
			}
		}

		return &internalevent.UserMessageContentCommand{
			CommandName:    commandName,
			CommandMessage: commandMessage,
			CommandArgs:    commandArgs,
		}
	}

	// Default to message content
	return &internalevent.UserMessageContentMessage{
		Text: text,
	}
}
