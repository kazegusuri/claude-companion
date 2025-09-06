package event

import (
	"sync"
)

// TaskInfo stores information about a Task execution
type TaskInfo struct {
	ToolUseID    string
	Description  string
	SubagentType string
}

// TaskTracker tracks Task tool executions by their tool_use_id
type TaskTracker struct {
	tasks map[string]TaskInfo
	mu    sync.RWMutex
}

// NewTaskTracker creates a new TaskTracker
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		tasks: make(map[string]TaskInfo),
	}
}

// TrackTask stores information about a Task execution
func (t *TaskTracker) TrackTask(toolUseID, description, subagentType string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tasks[toolUseID] = TaskInfo{
		ToolUseID:    toolUseID,
		Description:  description,
		SubagentType: subagentType,
	}
}

// GetTask retrieves Task information by tool_use_id
func (t *TaskTracker) GetTask(toolUseID string) (TaskInfo, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	info, exists := t.tasks[toolUseID]
	return info, exists
}

// RemoveTask removes Task information after it's been used
func (t *TaskTracker) RemoveTask(toolUseID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.tasks, toolUseID)
}
