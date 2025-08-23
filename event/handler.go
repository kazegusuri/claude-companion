package event

import (
	"fmt"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/handler"
	"github.com/kazegusuri/claude-companion/logger"
	"github.com/kazegusuri/claude-companion/narrator"
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

// Handler processes events from multiple sources
type Handler struct {
	narrator       narrator.Narrator
	formatter      FormatterInterface
	debugMode      bool
	eventChan      chan Event
	wg             sync.WaitGroup
	done           chan struct{}
	taskTracker    *TaskTracker
	sessionManager *handler.SessionManager

	// Buffering support
	bufferMutex sync.Mutex
	buffers     map[string]*BufferInfo // key: session name
}

// NewHandler creates a new event handler
func NewHandler(narrator narrator.Narrator, sessionManager *handler.SessionManager, debugMode bool) *Handler {
	formatter := NewFormatter(narrator)
	formatter.SetDebugMode(debugMode)
	taskTracker := NewTaskTracker()

	return &Handler{
		narrator:       narrator,
		formatter:      formatter,
		debugMode:      debugMode,
		eventChan:      make(chan Event, 100),
		done:           make(chan struct{}),
		taskTracker:    taskTracker,
		sessionManager: sessionManager,
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
	close(h.eventChan)
	h.wg.Wait()
}

// SendEvent sends an event to be processed
func (h *Handler) SendEvent(event Event) {
	select {
	case h.eventChan <- event:
	case <-h.done:
		// Handler is stopping, discard event
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
			if h.debugMode {
				logger.LogInfo("Warmup: Session %s not found for event type %s", event.SessionID, event.TypeString)
			}
		} else {
			// Update session with warmup information if needed
			if h.debugMode {
				logger.LogInfo("Warmup: Processing event type %s for session %s (CWD: %s)", event.TypeString, event.SessionID, session.CWD)
			}
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
			if h.debugMode {
				logger.LogInfo("Ignoring sidechain UserMessage")
			}
			return
		}
	case *AssistantMessage:
		if e.IsSidechain {
			if h.debugMode {
				logger.LogInfo("Ignoring sidechain AssistantMessage")
			}
			return
		}
	case *SystemMessage:
		if e.IsSidechain {
			if h.debugMode {
				logger.LogInfo("Ignoring sidechain SystemMessage")
			}
			return
		}
	case *HookEvent:
		if e.IsSidechain {
			if h.debugMode {
				logger.LogInfo("Ignoring sidechain HookEvent")
			}
			return
		}
	case *BaseEvent:
		if e.IsSidechain {
			if h.debugMode {
				logger.LogInfo("Ignoring sidechain BaseEvent")
			}
			return
		}
	}

	switch e := event.(type) {
	case *NotificationEvent:
		// Handle SessionStart notification events
		if e.HookEventName == "SessionStart" {
			// NotificationEvent has TranscriptPath, so we can use it directly
			h.sessionManager.CreateSession(e.SessionID, "", e.CWD, e.TranscriptPath)
		}
		// Process notification events
		output, err := h.formatter.Format(e)
		if err != nil {
			logger.LogError("Error formatting NotificationEvent: %v", err)
			return
		}
		if output != "" {
			fmt.Print(output)
		}
	case *AssistantMessage:
		// Track Task tool uses
		h.trackTaskToolUses(e)
		// Format and display
		output, err := h.formatter.Format(e)
		if err != nil {
			logger.LogError("Error formatting AssistantMessage: %v", err)
			return
		}
		if output != "" {
			fmt.Print(output)
		}
	case *UserMessage:
		// Check if this is a Task result and create TaskCompletionMessage
		if taskCompletion := h.checkTaskResultFromUser(e); taskCompletion != nil {
			// Process the task completion event
			output, err := h.formatter.Format(taskCompletion)
			if err != nil {
				logger.LogError("Error formatting TaskCompletionMessage: %v", err)
			} else if output != "" {
				fmt.Print(output)
			}
		}
		// Normal formatting
		output, err := h.formatter.Format(e)
		if err != nil {
			logger.LogError("Error formatting UserMessage: %v", err)
			return
		}
		if output != "" {
			fmt.Print(output)
		}
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
		// Format and display the event
		output, err := h.formatter.Format(e)
		if err != nil {
			logger.LogError("Error formatting HookEvent: %v", err)
			return
		}
		if output != "" {
			fmt.Print(output)
		}
	case *SystemMessage, *SummaryEvent, *BaseEvent, *TaskCompletionMessage:
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
		if h.debugMode {
			logger.LogWarning("Unknown event type: %T", event)
		}
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

				if h.debugMode {
					logger.LogInfo("Tracking Task: ID=%s, Description=%s, Agent=%s",
						content.ID, description, subagentType)
				}
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

						if h.debugMode {
							logger.LogInfo("Task completed: ID=%s, Description=%s, Agent=%s",
								toolUseID, taskInfo.Description, taskInfo.SubagentType)
						}

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

			if !exists {
				// Case 1: No session exists - treat as completely new
				if h.debugMode {
					logger.LogInfo("New session (not registered): %s", baseEvent.SessionID)
				}
				return false // Process normally
			}

			if session.UUID == "" || session.UUID == baseEvent.UUID {
				// Case 2: Empty UUID or Same UUID - treat as new/normal start
				if h.debugMode {
					if session.UUID == "" {
						logger.LogInfo("Normal session start (empty UUID): %s", baseEvent.SessionID)
					} else {
						logger.LogInfo("Normal session start (UUID match): %s", baseEvent.SessionID)
					}
				}
				return false // Process normally
			} else {
				// Case 3: Different UUID - this is a resume scenario
				if h.debugMode {
					logger.LogInfo("Resume detected (UUID mismatch) for session: %s, stored UUID: %s, event UUID: %s",
						baseEvent.SessionID, session.UUID, baseEvent.UUID)
				}

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
		}

		// Not a SessionStart event with ParentUUID=nil - might be an issue
		if h.debugMode {
			logger.LogInfo("Non-SessionStart event with ParentUUID=nil: %T", event)
		}
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

	if h.debugMode {
		logger.LogInfo("Releasing buffer for session %s: %s (events: %d, duration: %v)",
			sessionName, reason, len(buffer.events), time.Since(buffer.startTime))
	}

	// Remove buffer and discard buffered events
	delete(h.buffers, sessionName)

	// Buffered events are discarded (not re-enqueued)
}
