package event

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// parseHookStatus converts string status to HookStatus enum
func parseHookStatus(status string) internalevent.HookStatus {
	switch status {
	case "started":
		return internalevent.HookStatusStarted
	case "finished":
		return internalevent.HookStatusFinished
	default:
		// Default to finished for unknown status
		return internalevent.HookStatusFinished
	}
}

// BufferInfo holds information about buffered events for a session
type BufferInfo struct {
	events        []Event
	timer         *time.Timer
	sessionName   string
	startTime     time.Time
	resumedFromID string // The sessionID that triggered the resume
}

// SubagentBufferInfo holds information about buffered events for a subagent
type SubagentBufferInfo struct {
	events       []Event
	taskInfo     TaskInfo
	startTime    time.Time
	uuid         string // UUID of the first subagent event
	parentTaskID string // ID of the parent task tool use
	lastMessage  string // Last text message from the subagent
	isCompleted  bool   // Whether this subagent has been matched to a task completion
}

// CentralEventHandler is the interface for central event handler
type CentralEventHandler interface {
	SendEvent(event internalevent.Event)
}

// Handler processes events from multiple sources
type Handler struct {
	narrator        narrator.Narrator
	taskTracker     *TaskTracker
	sessionManager  *handler.SessionManager
	centralHandler  CentralEventHandler   // Central event handler
	session         *SessionFile          // The session this handler is watching
	internalSession internalevent.Session // Pre-converted internal session for reuse
	isWarmup        bool                  // Whether we're in warmup mode

	// Buffering support
	bufferMutex sync.Mutex
	buffers     map[string]*BufferInfo // key: session name

	// Subagent buffering support
	subagentMutex   sync.Mutex
	subagentBuffers map[string]*SubagentBufferInfo // key: subagent UUID
}

// NewHandler creates a new event handler
func NewHandler(narrator narrator.Narrator, sessionManager *handler.SessionManager, centralHandler CentralEventHandler, session *SessionFile) *Handler {
	taskTracker := NewTaskTracker()

	// Convert SessionFile to internalevent.Session once at initialization
	internalSession := internalevent.Session{
		SessionID:      session.SessionID,
		TranscriptPath: session.TranscriptPath,
	}

	return &Handler{
		narrator:        narrator,
		taskTracker:     taskTracker,
		sessionManager:  sessionManager,
		centralHandler:  centralHandler,
		session:         session,
		internalSession: internalSession,
		buffers:         make(map[string]*BufferInfo),
		subagentBuffers: make(map[string]*SubagentBufferInfo),
	}
}

// SetWarmupMode sets whether the handler is in warmup mode
func (h *Handler) SetWarmupMode(isWarmup bool) {
	h.isWarmup = isWarmup
}

// sendEventToCentral sends an event to the central handler if not in warmup mode
func (h *Handler) sendEventToCentral(event internalevent.Event) {
	// Don't send events to central handler during warmup
	if h.isWarmup {
		return
	}

	h.centralHandler.SendEvent(event)
}

// Start begins processing events (no-op for synchronous handler)
func (h *Handler) Start() {
	// No longer needed for synchronous processing
}

// Stop stops the event handler
func (h *Handler) Stop() {
	// Clean up any remaining buffered events
	h.bufferMutex.Lock()
	defer h.bufferMutex.Unlock()

	for _, buffer := range h.buffers {
		if buffer.timer != nil {
			buffer.timer.Stop()
		}
	}
	h.buffers = make(map[string]*BufferInfo)

	// Clean up subagent buffers
	h.subagentMutex.Lock()
	defer h.subagentMutex.Unlock()
	h.subagentBuffers = make(map[string]*SubagentBufferInfo)
}

// SendEvent processes an event synchronously
func (h *Handler) SendEvent(event Event) {
	h.processEvent(event)
}

// HandleWarmupEvent processes warmup events to initialize session state
func (h *Handler) HandleWarmupEvent(event *BaseEvent) {
	// Set warmup mode to prevent sending events to central handler
	previousWarmupState := h.isWarmup
	h.isWarmup = true
	defer func() {
		h.isWarmup = previousWarmupState
	}()

	// Extract session information from the base event
	if event.ParentUUID != nil && !event.IsSidechain && event.SessionID != "" && h.sessionManager != nil {
		// Get or create session
		_, exists := h.sessionManager.GetSession(event.SessionID)
		if !exists {
			h.sessionManager.CreateSession(event.SessionID, event.UUID, event.CWD, event.Session.TranscriptPath)
		}
	}

	// For warmup, we don't process the event through the normal pipeline
	// This is just to initialize state
}

// processEvent processes a single event based on its type
func (h *Handler) processEvent(event Event) {
	// Check if event should be buffered or if it releases buffered events
	if h.handleBuffering(event) {
		return // Event was buffered or handled
	}

	// Handle subagent events (sidechain events)
	if h.handleSubagentEvent(event) {
		return // Event was handled by subagent buffering
	}

	switch e := event.(type) {
	case *AssistantMessage:
		// Track Task tool uses
		h.trackTaskToolUses(e)

		// Convert and send to central handler
		// Central handler will handle all content types including tool_use
		centralAssistant := h.convertAssistantMessage(e)
		if centralAssistant != nil {
			h.sendEventToCentral(centralAssistant)
		}
	case *UserMessage:
		// Check if this is a Task result and create TaskCompletionMessage
		if taskCompletion := h.checkTaskResultFromUser(e); taskCompletion != nil {
			// Send TaskCompletionMessage to central handler
			centralTaskCompletion := &internalevent.TaskCompletionMessage{
				Session: h.internalSession,
				TaskInfo: internalevent.TaskInfo{
					ToolUseID:    taskCompletion.TaskInfo.ToolUseID,
					Description:  taskCompletion.TaskInfo.Description,
					SubagentType: taskCompletion.TaskInfo.SubagentType,
				},
				SubagentTask: nil,
				Timestamp:    taskCompletion.Timestamp,
			}

			// Convert SubagentTask if present
			if taskCompletion.SubagentTask != nil {
				centralTaskCompletion.SubagentTask = &internalevent.SubagentTask{
					UUID:       taskCompletion.SubagentTask.UUID,
					EventCount: taskCompletion.SubagentTask.EventCount,
				}
			}
			h.sendEventToCentral(centralTaskCompletion)
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
				Session: h.internalSession,
				Message: internalevent.UserMessageData{
					Role:    "user",
					Content: content,
				},
				// Copy optional fields from the parsed event
				ToolUseResult: e.ToolUseResult,
			}
			h.sendEventToCentral(centralEvent)
		}

		// UserMessage is now handled by central event handler
		// WebSocket broadcast is done in central handler for UserMessageContentMessage
	case *HookEvent:
		// Handle SessionStart event
		if e.HookEventType == "SessionStart" {
			// Create a new session
			h.sessionManager.CreateSession(e.SessionID, e.UUID, e.CWD, h.session.TranscriptPath)
		}

		// Convert HookEvent to SystemMessage and send to central handler
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
			Session:    h.internalSession,
			RawContent: e.Content,
			Content: &internalevent.HookSystemMessageContent{
				HookName: hookName,
				Command:  e.HookCommand,
				Status:   parseHookStatus(e.HookStatus),
				Type:     hookType,
				Message:  e.Content,
			},
			Level:     "info", // Default level for hook events
			ToolUseID: e.ToolUseID,
		}

		h.sendEventToCentral(centralEvent)
	case *SystemMessage:
		// Convert to internal/event.SystemMessage and forward to central handler
		centralEvent := &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        e.UUID,
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: e.IsSidechain,
				CWD:         e.CWD,
				Timestamp:   e.Timestamp,
				IsMeta:      e.IsMeta,
			},
			Session:    h.internalSession,
			RawContent: e.Content,
			Level:      e.Level,
			ToolUseID:  e.ToolUseID,
		}
		h.sendEventToCentral(centralEvent)
		// SystemMessage display is now handled by central handler's printer
	case *SummaryEvent:
		// Convert to internal/event.SummaryEvent and forward to central handler
		centralEvent := &internalevent.SummaryEvent{
			Session:  h.internalSession,
			LeafUUID: e.LeafUUID,
			Summary:  e.Summary,
		}
		h.sendEventToCentral(centralEvent)
		// SummaryEvent display is now handled by central handler's printer
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
							BaseEvent:    msg.BaseEvent, // Use BaseEvent from UserMessage
							TaskInfo:     taskInfo,
							SubagentTask: nil, // Will be set below if we find a matching subagent
						}

						// Try to match with a subagent based on the result content
						if msg.ToolUseResult != nil {
							if resultMap, ok := msg.ToolUseResult.(map[string]interface{}); ok {
								if content, ok := resultMap["content"].([]interface{}); ok && len(content) > 0 {
									if firstContent, ok := content[0].(map[string]interface{}); ok {
										if contentType, ok := firstContent["type"].(string); ok && contentType == "text" {
											if text, ok := firstContent["text"].(string); ok && text != "" {
												// Find matching subagent by last message
												subagentTask, matchedUUID := h.matchSubagentByMessage(text)
												if subagentTask != nil {
													taskCompletion.SubagentTask = subagentTask
													logger.DebugInfo("Matched subagent with task: ToolUseID=%s, SubagentUUID=%s, EventCount=%d",
														toolUseID, matchedUUID, subagentTask.EventCount)
												}
											}
										}
									}
								}
							}
						}

						eventCount := 0
						if taskCompletion.SubagentTask != nil {
							eventCount = taskCompletion.SubagentTask.EventCount
						}
						logger.DebugInfo("Task completed: ID=%s, Description=%s, Agent=%s, SubagentEvents=%d",
							toolUseID, taskInfo.Description, taskInfo.SubagentType, eventCount)

						return taskCompletion
					}
				}
			}
		}
	}
	return nil
}

// matchSubagentByMessage finds a subagent buffer by its last message and marks it as completed
func (h *Handler) matchSubagentByMessage(message string) (*SubagentTask, string) {
	h.subagentMutex.Lock()
	defer h.subagentMutex.Unlock()

	for uuid, buffer := range h.subagentBuffers {
		// Skip already completed subagents
		if buffer.isCompleted {
			continue
		}

		// Check if the last message matches
		if buffer.lastMessage == message {
			// Mark as completed
			buffer.isCompleted = true
			eventCount := len(buffer.events)

			logger.DebugInfo("Matched subagent by message: UUID=%s, EventCount=%d, Message=%s",
				uuid, eventCount, message)

			return &SubagentTask{
				UUID:       buffer.uuid,
				EventCount: eventCount,
			}, uuid
		}
	}

	return nil, ""
}

// handleBuffering checks if an event should be buffered or if it releases buffered events
// Returns true if the event was handled (buffered or triggered release)
//
// Resume handling:
// When the resume command is executed, past events from the resumed session are passed through,
// so we need to ignore these historical events.
//
// Resume Start Detection:
//   - Condition: parentUUID == null AND handler.sessionID != event.sessionID
//   - Description: When reading from the beginning of a session file, if an event with null parentUUID
//     appears and its sessionID differs from the handler's expected sessionID,
//     this indicates events from a past session and marks the start of resume processing
//   - Action: Start buffering events from this point onwards
//
// Resume End Detection:
//   - Condition: HookEvent with SessionStart:resume AND handler.sessionID == event.sessionID
//   - Description: When a SessionStart:resume HookEvent arrives with a sessionID matching
//     the handler's sessionID, this indicates resume processing is complete
//   - Action: Discard buffered events and resume normal processing from this event
//
// Timeout-based Auto-release:
// - Condition: 1 second elapsed since resume start
// - Description: Even if SessionStart:resume doesn't arrive, automatically release buffer after 1 second
// - Action: Discard buffered events, send ResumeEvent, and return to normal processing
//
// Normal Session Start:
// - Condition: parentUUID == null AND handler.sessionID == event.sessionID
// - Description: When parentUUID is null but sessionID matches, this is a normal session start
// - Action: Continue normal processing without buffering
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
		// Check for Resume End: Check SessionStart:resume event FIRST before buffering check
		// This must be done before checking if event should be buffered
		if e.HookEventType == "SessionStart:resume" && baseEvent.Session != nil {
			sessionName := baseEvent.Session.SessionID
			// Resume End Condition: SessionStart:resume AND handler.sessionID == event.sessionID
			// When this condition is met, resume processing is complete and normal processing resumes
			if baseEvent.SessionID == h.session.SessionID {
				// Release buffer (buffered events are discarded)
				h.releaseBuffer(sessionName, "SessionStart:resume received with matching SessionID")
				// This SessionStart:resume event itself is processed normally
				return false
			}
			// If SessionID doesn't match, this resume event belongs to another session,
			// so continue to check if it should be buffered
		}
	case *BaseEvent:
		baseEvent = e
	case *TaskCompletionMessage:
		baseEvent = &e.BaseEvent
	case *SummaryEvent:
		// SummaryEvent typically doesn't have session info in original JSON,
		// but parser adds Session field. However, SessionID is empty.
		// Skip buffering for SummaryEvent as it's informational
		return false
	default:
		// Event doesn't have BaseEvent (e.g., NotificationEvent)
		return false
	}

	// Get session name if we have it
	if baseEvent == nil || baseEvent.Session == nil {
		return false
	}
	sessionName := baseEvent.Session.SessionID

	// Check if we need to buffer this event
	if !baseEvent.IsSidechain && baseEvent.ParentUUID == nil {
		// This is a session start event (parentUUID is null)
		// Check if the SessionID matches the one this handler is watching
		if baseEvent.SessionID != h.session.SessionID {
			// Resume Start Detected: parentUUID is null AND SessionID differs from handler's expected ID
			// This indicates we're reading events from a past session that was resumed
			// All events until SessionStart:resume with matching sessionID should be buffered (discarded)
			logger.DebugInfo("Resume detected: Expected SessionID=%s, got SessionID=%s",
				h.session.SessionID, baseEvent.SessionID)

			h.bufferMutex.Lock()
			defer h.bufferMutex.Unlock()

			// Check if we already have a buffer for this session
			// Multiple parentUUID=null events can appear during resume
			if buffer, exists := h.buffers[sessionName]; exists {
				// Add to existing buffer
				buffer.events = append(buffer.events, event)
				return true
			}

			// Create new buffer for this session
			buffer := &BufferInfo{
				events:        []Event{event},
				sessionName:   sessionName,
				startTime:     time.Now(),
				resumedFromID: baseEvent.SessionID, // Store the sessionID that triggered the resume
				// Timeout: Auto-release buffer after 1 second if SessionStart:resume doesn't arrive
				// This prevents indefinite buffering in case resume event is lost
				timer: time.AfterFunc(1*time.Second, func() {
					h.releaseBuffer(sessionName, "timeout")
				}),
			}
			h.buffers[sessionName] = buffer
			return true
		}

		// Normal Session Start: parentUUID is null AND SessionID matches handler's expected ID
		// This is a regular session start, not a resume scenario
		logger.DebugInfo("Normal session start for SessionID: %s", baseEvent.SessionID)
		return false // Process normally
	}

	// Check if this event is for a buffered session
	// If a buffer exists for this session, continue buffering subsequent events
	// until SessionStart:resume with matching sessionID arrives
	h.bufferMutex.Lock()
	if buffer, exists := h.buffers[sessionName]; exists {
		// Add to buffer - this event is part of the resumed session's history
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

	// Send ResumeEvent to central handler
	resumeEvent := &internalevent.ResumeEvent{
		Session:       h.internalSession,
		ResumedFromID: buffer.resumedFromID,
		BufferedCount: len(buffer.events),
		Timestamp:     time.Now(),
		Reason:        reason,
	}
	h.sendEventToCentral(resumeEvent)

	// Remove buffer and discard buffered events
	delete(h.buffers, sessionName)

	// Buffered events are discarded (not re-enqueued)
}

// handleSubagentEvent handles subagent (sidechain) events
// Returns true if the event was handled as a subagent event
func (h *Handler) handleSubagentEvent(event Event) bool {
	// Extract BaseEvent to check IsSidechain
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
	case *BaseEvent:
		baseEvent = e
	case *TaskCompletionMessage:
		baseEvent = &e.BaseEvent
	default:
		return false
	}

	// Only handle sidechain events
	if !baseEvent.IsSidechain {
		return false
	}

	h.subagentMutex.Lock()
	defer h.subagentMutex.Unlock()

	// Check if this is the start of a new subagent thread (parentUUID == null && IsSidechain == true)
	if baseEvent.ParentUUID == nil {
		// This is the start of a new subagent thread
		logger.DebugInfo("Subagent thread started: UUID=%s", baseEvent.UUID)

		// Get task info from the tracker using the current tracked tasks
		// We need to find which task this subagent belongs to
		var taskInfo TaskInfo
		var parentTaskID string

		// Try to find the most recent Task tool use that hasn't been completed yet
		for id, info := range h.taskTracker.GetAllTasks() {
			// Use the most recent task (this is a simplification - in production you might want better matching)
			taskInfo = info
			parentTaskID = id
			break
		}

		// Create new subagent buffer
		buffer := &SubagentBufferInfo{
			events:       []Event{event},
			taskInfo:     taskInfo,
			startTime:    baseEvent.Timestamp,
			uuid:         baseEvent.UUID,
			parentTaskID: parentTaskID,
		}

		h.subagentBuffers[baseEvent.UUID] = buffer
		return true
	}

	// Check if this event belongs to an existing subagent buffer
	// We need to find which buffer this event belongs to by checking parent chain
	for uuid, buffer := range h.subagentBuffers {
		// Skip completed subagents
		if buffer.isCompleted {
			continue
		}

		// Check if this event is part of this subagent's chain
		if h.isEventInSubagentChain(event, uuid) {
			// Add event to buffer
			buffer.events = append(buffer.events, event)

			// Check if this is an assistant message with text content
			if assistantMsg, ok := event.(*AssistantMessage); ok {
				if assistantMsg.Message.Type == "message" && len(assistantMsg.Message.Content) > 0 {
					// Check if the first content is text
					firstContent := assistantMsg.Message.Content[0]
					if firstContent.Type == "text" && firstContent.Text != "" {
						buffer.lastMessage = firstContent.Text
						logger.DebugInfo("Updated subagent last message: UUID=%s, Message=%s",
							uuid, buffer.lastMessage)
					}
				}
			}

			logger.DebugInfo("Buffered subagent event: UUID=%s, BufferSize=%d",
				baseEvent.UUID, len(buffer.events))
			return true
		}
	}

	// This sidechain event doesn't belong to any tracked subagent
	logger.DebugInfo("Untracked sidechain event: UUID=%s", baseEvent.UUID)
	return true
}

// isEventInSubagentChain checks if an event belongs to a specific subagent chain
func (h *Handler) isEventInSubagentChain(event Event, subagentUUID string) bool {
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
	case *BaseEvent:
		baseEvent = e
	case *TaskCompletionMessage:
		baseEvent = &e.BaseEvent
	default:
		return false
	}

	// Check if this event's parent UUID matches any event in the buffer
	if baseEvent.ParentUUID == nil {
		return false
	}

	// Check if parent UUID matches the subagent's initial UUID
	if *baseEvent.ParentUUID == subagentUUID {
		return true
	}

	// Check if parent UUID matches any event in the buffer
	buffer := h.subagentBuffers[subagentUUID]
	if buffer != nil {
		for _, bufEvent := range buffer.events {
			var bufBase *BaseEvent
			switch e := bufEvent.(type) {
			case *UserMessage:
				bufBase = &e.BaseEvent
			case *AssistantMessage:
				bufBase = &e.BaseEvent
			case *SystemMessage:
				bufBase = &e.BaseEvent
			case *HookEvent:
				bufBase = &e.BaseEvent
			case *BaseEvent:
				bufBase = e
			case *TaskCompletionMessage:
				bufBase = &e.BaseEvent
			}
			if bufBase != nil && bufBase.UUID == *baseEvent.ParentUUID {
				return true
			}
		}
	}

	return false
}

// GetSubagentBuffers returns a snapshot of current subagent buffers for debugging
func (h *Handler) GetSubagentBuffers() map[string]*SubagentBufferInfo {
	h.subagentMutex.Lock()
	defer h.subagentMutex.Unlock()

	// Create a copy to avoid race conditions
	buffersCopy := make(map[string]*SubagentBufferInfo)
	for k, v := range h.subagentBuffers {
		buffersCopy[k] = &SubagentBufferInfo{
			events:       v.events, // Note: this is a shallow copy
			taskInfo:     v.taskInfo,
			startTime:    v.startTime,
			uuid:         v.uuid,
			parentTaskID: v.parentTaskID,
			lastMessage:  v.lastMessage,
			isCompleted:  v.isCompleted,
		}
	}
	return buffersCopy
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
