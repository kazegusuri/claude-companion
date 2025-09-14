package event

import (
	"sync"
	"time"
)

// ToolStatus represents the status of a tool execution
type ToolStatus string

const (
	ToolStatusCreated  ToolStatus = "created"  // AssistantMessage でツール使用が提案された
	ToolStatusFinished ToolStatus = "finished" // 完了（成功/失敗/拒否）- tool_resultで判定
)

// BackgroundTaskInfo stores information about a background task
type BackgroundTaskInfo struct {
	BackgroundTaskID string // backgroundTaskId from tool_result
	ToolUseID        string // Associated tool_use_id
	Command          string // The command being executed
	Description      string // The description from the Bash tool
	IsTerminated     bool   // Whether the task has been terminated
	CreatedAt        time.Time
	TerminatedAt     *time.Time
}

// ToolInfo stores information about a tool execution
type ToolInfo struct {
	ToolUseID          string
	ToolName           string // Bash, Edit, etc.
	Status             ToolStatus
	CreatedAt          time.Time
	UpdatedAt          time.Time
	IsError            bool   // エラーで終了したか
	IsRejected         bool   // ユーザーに拒否されたか
	BackgroundTaskID   string // backgroundTaskId if this is a background task
}

// IsWaitingApproval returns true if the tool is waiting for approval
// Note: This state is now managed by Session, not ToolTracker
func (t *ToolInfo) IsWaitingApproval() bool {
	return false
}

// ToolTracker tracks tool executions by their tool_use_id
type ToolTracker struct {
	tools            map[string]*ToolInfo
	backgroundTasks  map[string]*BackgroundTaskInfo // key: backgroundTaskID
	toolDescriptions map[string]string              // temporary storage for tool descriptions (key: tool_use_id)
	mu               sync.RWMutex
}

// NewToolTracker creates a new ToolTracker
func NewToolTracker() *ToolTracker {
	return &ToolTracker{
		tools:            make(map[string]*ToolInfo),
		backgroundTasks:  make(map[string]*BackgroundTaskInfo),
		toolDescriptions: make(map[string]string),
	}
}

// TrackToolCreated stores information about a new tool execution
func (t *ToolTracker) TrackToolCreated(toolUseID, toolName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.tools[toolUseID] = &ToolInfo{
		ToolUseID:        toolUseID,
		ToolName:         toolName,
		Status:           ToolStatusCreated,
		CreatedAt:        now,
		UpdatedAt:        now,
		IsError:          false,
		IsRejected:       false,
		BackgroundTaskID: "",
	}
}

// UpdateToolStatus updates the status of a tool execution
// Ensures state transitions are one-way only (cannot go backwards)
func (t *ToolTracker) UpdateToolStatus(toolUseID string, newStatus ToolStatus) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	tool, exists := t.tools[toolUseID]
	if !exists {
		return false
	}

	// Check if transition is valid (one-way only)
	if !t.isValidTransition(tool.Status, newStatus) {
		return false
	}

	tool.Status = newStatus
	tool.UpdatedAt = time.Now()
	return true
}

// isValidTransition checks if a status transition is valid
func (t *ToolTracker) isValidTransition(from, to ToolStatus) bool {
	// Define valid transitions
	validTransitions := map[ToolStatus][]ToolStatus{
		ToolStatusCreated:  {ToolStatusFinished},
		ToolStatusFinished: {}, // Cannot transition from finished
	}

	allowed, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}

// GetActiveTool returns the currently active tool (not finished)
func (t *ToolTracker) GetActiveTool() (*ToolInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, tool := range t.tools {
		if tool.Status != ToolStatusFinished {
			// Return a copy to avoid race conditions
			toolCopy := *tool
			return &toolCopy, true
		}
	}
	return nil, false
}

// GetTool retrieves tool information by tool_use_id
func (t *ToolTracker) GetTool(toolUseID string) (*ToolInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tool, exists := t.tools[toolUseID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	toolCopy := *tool
	return &toolCopy, true
}


// FinishTool marks a tool as finished with optional error/rejection flags
func (t *ToolTracker) FinishTool(toolUseID string, isError, isRejected bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	tool, exists := t.tools[toolUseID]
	if !exists {
		return false
	}

	// Check if transition to finished is valid
	if !t.isValidTransition(tool.Status, ToolStatusFinished) {
		return false
	}

	tool.Status = ToolStatusFinished
	tool.UpdatedAt = time.Now()
	tool.IsError = isError
	tool.IsRejected = isRejected
	return true
}

// RemoveTool removes tool information after it's been processed
func (t *ToolTracker) RemoveTool(toolUseID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.tools, toolUseID)
}

// GetAllTools returns a copy of all tracked tools
func (t *ToolTracker) GetAllTools() map[string]ToolInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a copy to avoid race conditions
	toolsCopy := make(map[string]ToolInfo)
	for k, v := range t.tools {
		toolsCopy[k] = *v
	}
	return toolsCopy
}

// TrackBackgroundTask tracks a new background task
func (t *ToolTracker) TrackBackgroundTask(backgroundTaskID, toolUseID, command, description string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.backgroundTasks[backgroundTaskID] = &BackgroundTaskInfo{
		BackgroundTaskID: backgroundTaskID,
		ToolUseID:        toolUseID,
		Command:          command,
		Description:      description,
		IsTerminated:     false,
		CreatedAt:        now,
		TerminatedAt:     nil,
	}

	// Update associated tool with background task ID
	if tool, exists := t.tools[toolUseID]; exists {
		tool.BackgroundTaskID = backgroundTaskID
	}
}

// UpdateBackgroundTaskCommand updates the command and optionally description of an existing background task
func (t *ToolTracker) UpdateBackgroundTaskCommand(backgroundTaskID string, command string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	task, exists := t.backgroundTasks[backgroundTaskID]
	if !exists {
		return false
	}

	// Update command if it was empty or different
	if task.Command == "" || task.Command != command {
		task.Command = command
	}
	return true
}

// UpdateBackgroundTaskDescription updates the description of an existing background task
func (t *ToolTracker) UpdateBackgroundTaskDescription(backgroundTaskID string, description string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	task, exists := t.backgroundTasks[backgroundTaskID]
	if !exists {
		return false
	}

	// Update description if it was empty
	if task.Description == "" && description != "" {
		task.Description = description
	}
	return true
}

// TerminateBackgroundTask marks a background task as terminated
func (t *ToolTracker) TerminateBackgroundTask(backgroundTaskID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	task, exists := t.backgroundTasks[backgroundTaskID]
	if !exists {
		return false
	}

	if !task.IsTerminated {
		now := time.Now()
		task.IsTerminated = true
		task.TerminatedAt = &now
	}
	return true
}

// GetBackgroundTask retrieves background task information by ID
func (t *ToolTracker) GetBackgroundTask(backgroundTaskID string) (*BackgroundTaskInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	task, exists := t.backgroundTasks[backgroundTaskID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	taskCopy := *task
	if task.TerminatedAt != nil {
		terminatedTime := *task.TerminatedAt
		taskCopy.TerminatedAt = &terminatedTime
	}
	return &taskCopy, true
}

// GetActiveBackgroundTasks returns all active (non-terminated) background tasks
func (t *ToolTracker) GetActiveBackgroundTasks() map[string]BackgroundTaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activeTasks := make(map[string]BackgroundTaskInfo)
	for id, task := range t.backgroundTasks {
		if !task.IsTerminated {
			taskCopy := *task
			if task.TerminatedAt != nil {
				terminatedTime := *task.TerminatedAt
				taskCopy.TerminatedAt = &terminatedTime
			}
			activeTasks[id] = taskCopy
		}
	}
	return activeTasks
}

// GetAllBackgroundTasks returns a copy of all background tasks
func (t *ToolTracker) GetAllBackgroundTasks() map[string]BackgroundTaskInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	tasksCopy := make(map[string]BackgroundTaskInfo)
	for id, task := range t.backgroundTasks {
		taskCopy := *task
		if task.TerminatedAt != nil {
			terminatedTime := *task.TerminatedAt
			taskCopy.TerminatedAt = &terminatedTime
		}
		tasksCopy[id] = taskCopy
	}
	return tasksCopy
}

// RemoveBackgroundTask removes a background task from tracking
func (t *ToolTracker) RemoveBackgroundTask(backgroundTaskID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear background task ID from associated tool
	if task, exists := t.backgroundTasks[backgroundTaskID]; exists {
		if tool, toolExists := t.tools[task.ToolUseID]; toolExists {
			tool.BackgroundTaskID = ""
		}
	}

	delete(t.backgroundTasks, backgroundTaskID)
}

// StoreToolDescription temporarily stores a tool description by tool_use_id
// This is used when we get the description from AssistantMessage before the backgroundTaskId
func (t *ToolTracker) StoreToolDescription(toolUseID, description string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.toolDescriptions[toolUseID] = description
}

// GetToolDescription retrieves and removes a stored tool description
func (t *ToolTracker) GetToolDescription(toolUseID string) (string, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	desc, exists := t.toolDescriptions[toolUseID]
	if exists {
		delete(t.toolDescriptions, toolUseID)
	}
	return desc, exists
}
