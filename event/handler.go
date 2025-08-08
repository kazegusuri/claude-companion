package event

import (
	"fmt"
	"sync"
	"time"

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
	narrator    narrator.Narrator
	formatter   FormatterInterface
	debugMode   bool
	eventChan   chan Event
	wg          sync.WaitGroup
	done        chan struct{}
	taskTracker *TaskTracker

	// Buffering support
	bufferMutex sync.Mutex
	buffers     map[string]*BufferInfo // key: session name
}

// NewHandler creates a new event handler
func NewHandler(narrator narrator.Narrator, debugMode bool) *Handler {
	formatter := NewFormatter(narrator)
	formatter.SetDebugMode(debugMode)
	taskTracker := NewTaskTracker()

	return &Handler{
		narrator:    narrator,
		formatter:   formatter,
		debugMode:   debugMode,
		eventChan:   make(chan Event, 100),
		done:        make(chan struct{}),
		taskTracker: taskTracker,
		buffers:     make(map[string]*BufferInfo),
	}
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
	case *SystemMessage, *HookEvent, *SummaryEvent, *BaseEvent, *TaskCompletionMessage:
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
func (h *Handler) handleBuffering(event Event) bool {
	// Extract BaseEvent from different event types
	var baseEvent *BaseEvent
	var sessionName string

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
			sessionName = baseEvent.Session.Session
			h.releaseBuffer(sessionName, "SessionStart:resume received")
			// Process this event normally after releasing buffer
			return false
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
	if baseEvent != nil && baseEvent.Session != nil {
		sessionName = baseEvent.Session.Session
	}

	// Check if we need to buffer this event
	if baseEvent != nil && !baseEvent.IsSidechain && baseEvent.Session != nil && baseEvent.ParentUUID == nil {
		h.bufferMutex.Lock()
		defer h.bufferMutex.Unlock()

		if h.debugMode {
			logger.LogInfo("Buffering event (ParentUUID==nil) for session: %s, type: %T", sessionName, event)
		}

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

	// Check if this event is for a buffered session
	if sessionName != "" {
		h.bufferMutex.Lock()
		if buffer, exists := h.buffers[sessionName]; exists {
			// Add to buffer
			buffer.events = append(buffer.events, event)
			h.bufferMutex.Unlock()
			return true
		}
		h.bufferMutex.Unlock()
	}

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
