package event

import (
	"time"

	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// syncToolInfoToSession synchronizes tool information from ToolTracker to SessionManager
func (h *Handler) syncToolInfoToSession() {
	if h.sessionManager == nil || h.session == nil {
		return
	}

	session, exists := h.sessionManager.GetSession(h.session.SessionID)
	if !exists {
		return
	}

	// Sync active tool
	if activeTool, exists := h.toolTracker.GetActiveTool(); exists {
		toolInfo := &handler.ToolInfo{
			ToolUseID:         activeTool.ToolUseID,
			ToolName:          activeTool.ToolName,
			Status:            string(activeTool.Status),
			CreatedAt:         activeTool.CreatedAt,
			UpdatedAt:         activeTool.UpdatedAt,
			IsError:           activeTool.IsError,
			IsRejected:        activeTool.IsRejected,
			IsWaitingApproval: activeTool.IsWaitingApproval(),
		}
		session.UpdateActiveTool(toolInfo)
	} else {
		session.UpdateActiveTool(nil)
	}

	// Sync background tasks
	allBgTasks := h.toolTracker.GetAllBackgroundTasks()
	for _, bgTask := range allBgTasks {
		bgInfo := &handler.BackgroundTaskInfo{
			BackgroundTaskID: bgTask.BackgroundTaskID,
			ToolUseID:        bgTask.ToolUseID,
			Command:          bgTask.Command,
			Description:      bgTask.Description,
			IsTerminated:     bgTask.IsTerminated,
			CreatedAt:        bgTask.CreatedAt,
			TerminatedAt:     bgTask.TerminatedAt,
		}
		session.AddBackgroundTask(bgInfo)
	}

	// Remove tasks that no longer exist in ToolTracker
	sessionBgTasks := session.GetBackgroundTasks()
	for bgID := range sessionBgTasks {
		if _, exists := allBgTasks[bgID]; !exists {
			session.RemoveBackgroundTask(bgID)
		}
	}
}

// trackBackgroundTaskFromToolResult checks if the tool result contains a background task ID and tracks it
func (h *Handler) trackBackgroundTaskFromToolResult(userMsg *UserMessage) {
	if userMsg.ToolUseResult == nil {
		return
	}

	// Parse ToolUseResult as a map
	if resultMap, ok := userMsg.ToolUseResult.(map[string]interface{}); ok {
		// Case 1: Bash tool creates a new background task
		if bgTaskID, ok := resultMap["backgroundTaskId"].(string); ok && bgTaskID != "" {
			// Find the tool_use_id from the message content
			var toolUseID string
			if contents, ok := userMsg.Message.Content.([]interface{}); ok {
				for _, content := range contents {
					if contentMap, ok := content.(map[string]interface{}); ok {
						if contentMap["type"] == "tool_result" {
							if id, ok := contentMap["tool_use_id"].(string); ok {
								toolUseID = id
								break
							}
						}
					}
				}
			}

			if toolUseID != "" {
				// Extract command from the result if available
				command := ""
				if cmdVal, ok := resultMap["command"].(string); ok {
					command = cmdVal
				}

				// Get stored description if available
				description := ""
				if desc, exists := h.toolTracker.GetToolDescription(toolUseID); exists {
					description = desc
				}

				h.toolTracker.TrackBackgroundTask(bgTaskID, toolUseID, command, description)
				h.syncToolInfoToSession()
			}
		}

		// Case 2: BashOutput tool provides command information for existing background task
		if shellID, ok := resultMap["shellId"].(string); ok && shellID != "" {
			if command, ok := resultMap["command"].(string); ok && command != "" {
				// Update the command for existing background task
				h.toolTracker.UpdateBackgroundTaskCommand(shellID, command)
				h.syncToolInfoToSession()
			}
		}
	}
}

// terminateBackgroundTaskFromKillShell checks if the tool result is from KillShell and terminates the task
func (h *Handler) terminateBackgroundTaskFromKillShell(userMsg *UserMessage) {
	if userMsg.ToolUseResult == nil {
		return
	}

	// Parse ToolUseResult as a map
	if resultMap, ok := userMsg.ToolUseResult.(map[string]interface{}); ok {
		if shellID, ok := resultMap["shell_id"].(string); ok && shellID != "" {
			if h.toolTracker.TerminateBackgroundTask(shellID) {
				h.syncToolInfoToSession()
			}
		}
	}
}

// syncTaskInfoToSession synchronizes task information from TaskTracker to SessionManager
func (h *Handler) syncTaskInfoToSession() {
	if h.sessionManager == nil || h.session == nil {
		return
	}

	session, exists := h.sessionManager.GetSession(h.session.SessionID)
	if !exists {
		return
	}

	// Sync all active tasks
	allTasks := h.taskTracker.GetAllTasks()
	for taskID, task := range allTasks {
		taskInfo := &handler.TaskInfo{
			TaskID:      taskID,
			ToolUseID:   task.ToolUseID,
			TaskName:    task.SubagentType,
			Description: task.Description,
			Status:      "in_progress", // Tasks are always in progress until removed
			CreatedAt:   time.Now(),    // Use current time as TaskInfo doesn't have timestamps
			UpdatedAt:   time.Now(),
		}
		session.AddActiveTask(taskInfo)
	}

	// Remove tasks that no longer exist in TaskTracker
	sessionTasks := session.GetActiveTasks()
	for taskID := range sessionTasks {
		if _, exists := allTasks[taskID]; !exists {
			session.RemoveActiveTask(taskID)
		}
	}
}
