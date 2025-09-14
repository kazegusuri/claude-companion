package event

import (
	"sync"
	"time"
)

// ToolStatus represents the status of a tool execution
type ToolStatus string

const (
	ToolStatusCreated         ToolStatus = "created"          // AssistantMessage でツール使用が提案された
	ToolStatusWaitingApproval ToolStatus = "waiting_approval" // PreToolUse完了、承認待ち
	ToolStatusRunning         ToolStatus = "running"          // PostToolUse:Running または実行中
	ToolStatusFinished        ToolStatus = "finished"         // 完了（成功/失敗/拒否）
)

// ToolInfo stores information about a tool execution
type ToolInfo struct {
	ToolUseID  string
	ToolName   string // Bash, Edit, etc.
	Status     ToolStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	IsError    bool // エラーで終了したか
	IsRejected bool // ユーザーに拒否されたか
}

// ToolTracker tracks tool executions by their tool_use_id
type ToolTracker struct {
	tools map[string]*ToolInfo
	mu    sync.RWMutex
}

// NewToolTracker creates a new ToolTracker
func NewToolTracker() *ToolTracker {
	return &ToolTracker{
		tools: make(map[string]*ToolInfo),
	}
}

// TrackToolCreated stores information about a new tool execution
func (t *ToolTracker) TrackToolCreated(toolUseID, toolName string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.tools[toolUseID] = &ToolInfo{
		ToolUseID:  toolUseID,
		ToolName:   toolName,
		Status:     ToolStatusCreated,
		CreatedAt:  now,
		UpdatedAt:  now,
		IsError:    false,
		IsRejected: false,
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
		ToolStatusCreated:         {ToolStatusWaitingApproval, ToolStatusRunning, ToolStatusFinished},
		ToolStatusWaitingApproval: {ToolStatusRunning, ToolStatusFinished},
		ToolStatusRunning:         {ToolStatusFinished},
		ToolStatusFinished:        {}, // Cannot transition from finished
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
