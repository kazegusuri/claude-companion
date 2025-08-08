package event

import (
	"fmt"
	"sync"

	"github.com/kazegusuri/claude-companion/logger"
	"github.com/kazegusuri/claude-companion/narrator"
)

// Handler processes events from multiple sources
type Handler struct {
	narrator    narrator.Narrator
	formatter   *Formatter
	debugMode   bool
	eventChan   chan Event
	wg          sync.WaitGroup
	done        chan struct{}
	taskTracker *TaskTracker
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
