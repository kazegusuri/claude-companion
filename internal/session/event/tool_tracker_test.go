package event

import (
	"testing"
	"time"
)

func TestToolTracker_TrackToolCreated(t *testing.T) {
	tracker := NewToolTracker()

	// Track a new tool
	tracker.TrackToolCreated("tool-1", "Bash")

	// Verify tool was created
	tool, exists := tracker.GetTool("tool-1")
	if !exists {
		t.Fatal("Tool should exist")
	}

	if tool.ToolUseID != "tool-1" {
		t.Errorf("Expected tool ID 'tool-1', got '%s'", tool.ToolUseID)
	}
	if tool.ToolName != "Bash" {
		t.Errorf("Expected tool name 'Bash', got '%s'", tool.ToolName)
	}
	if tool.Status != ToolStatusCreated {
		t.Errorf("Expected status 'created', got '%s'", tool.Status)
	}
	if tool.IsError {
		t.Error("IsError should be false")
	}
	if tool.IsRejected {
		t.Error("IsRejected should be false")
	}
}

func TestToolTracker_UpdateToolStatus(t *testing.T) {
	tracker := NewToolTracker()
	tracker.TrackToolCreated("tool-1", "Edit")

	// Valid transition: Created -> Finished
	if !tracker.UpdateToolStatus("tool-1", ToolStatusFinished) {
		t.Error("Should allow transition from Created to Finished")
	}

	tool, _ := tracker.GetTool("tool-1")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Expected status 'finished', got '%s'", tool.Status)
	}

	// Invalid transition: Finished -> Created (cannot go backwards)
	if tracker.UpdateToolStatus("tool-1", ToolStatusCreated) {
		t.Error("Should not allow transition from Finished to Created")
	}

	// Status should remain Finished
	tool, _ = tracker.GetTool("tool-1")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Status should remain 'finished', got '%s'", tool.Status)
	}
}

func TestToolTracker_InvalidTransitions(t *testing.T) {
	tracker := NewToolTracker()
	tracker.TrackToolCreated("tool-1", "Write")

	// Move to Finished
	tracker.UpdateToolStatus("tool-1", ToolStatusFinished)

	// Invalid transition: Finished -> Created (cannot go backwards)
	if tracker.UpdateToolStatus("tool-1", ToolStatusCreated) {
		t.Error("Should not allow transition from Finished to Created")
	}

	// Status should remain Finished
	tool, _ := tracker.GetTool("tool-1")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Status should remain 'finished', got '%s'", tool.Status)
	}
}

func TestToolTracker_GetActiveTool(t *testing.T) {
	tracker := NewToolTracker()

	// No active tool initially
	_, exists := tracker.GetActiveTool()
	if exists {
		t.Error("Should not have active tool initially")
	}

	// Add first tool
	tracker.TrackToolCreated("tool-1", "Bash")
	activeTool, exists := tracker.GetActiveTool()
	if !exists {
		t.Fatal("Should have active tool")
	}
	if activeTool.ToolUseID != "tool-1" {
		t.Errorf("Active tool should be 'tool-1', got '%s'", activeTool.ToolUseID)
	}

	// Add second tool (still first is active as it's not finished)
	tracker.TrackToolCreated("tool-2", "Edit")
	activeTool, exists = tracker.GetActiveTool()
	if !exists {
		t.Fatal("Should have active tool")
	}
	// Note: GetActiveTool returns the first non-finished tool it finds

	// Finish first tool
	tracker.FinishTool("tool-1", false, false)
	activeTool, exists = tracker.GetActiveTool()
	if !exists {
		t.Fatal("Should still have active tool (tool-2)")
	}
	if activeTool.ToolUseID != "tool-2" {
		t.Errorf("Active tool should be 'tool-2', got '%s'", activeTool.ToolUseID)
	}

	// Finish second tool
	tracker.FinishTool("tool-2", false, false)
	_, exists = tracker.GetActiveTool()
	if exists {
		t.Error("Should not have active tool when all are finished")
	}
}

func TestToolTracker_FinishTool(t *testing.T) {
	tracker := NewToolTracker()
	tracker.TrackToolCreated("tool-1", "Grep")

	// Finish with rejection directly from Created
	if !tracker.FinishTool("tool-1", true, true) {
		t.Error("Should be able to finish tool")
	}

	tool, _ := tracker.GetTool("tool-1")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Expected status 'finished', got '%s'", tool.Status)
	}
	if !tool.IsError {
		t.Error("IsError should be true")
	}
	if !tool.IsRejected {
		t.Error("IsRejected should be true")
	}
}

func TestToolTracker_RejectionFlow(t *testing.T) {
	tracker := NewToolTracker()

	// Tool created
	tracker.TrackToolCreated("tool-reject", "Bash")

	// User rejects via tool_result - finish immediately
	if !tracker.FinishTool("tool-reject", true, true) {
		t.Error("Should be able to finish tool with rejection")
	}

	tool, _ := tracker.GetTool("tool-reject")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Tool should be finished after rejection, got '%s'", tool.Status)
	}
	if !tool.IsRejected {
		t.Error("Tool should be marked as rejected")
	}
}

func TestToolTracker_ApprovalFlow(t *testing.T) {
	tracker := NewToolTracker()

	// Tool created
	tracker.TrackToolCreated("tool-approve", "Edit")

	// Tool executed - finished directly (tool_result received)
	tracker.FinishTool("tool-approve", false, false)

	tool, _ := tracker.GetTool("tool-approve")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Tool should be finished, got '%s'", tool.Status)
	}
	if tool.IsError {
		t.Error("Tool should not have error")
	}
	if tool.IsRejected {
		t.Error("Tool should not be rejected")
	}
}

func TestToolTracker_GetAllTools(t *testing.T) {
	tracker := NewToolTracker()

	// Add multiple tools
	tracker.TrackToolCreated("tool-1", "Bash")
	tracker.TrackToolCreated("tool-2", "Edit")
	tracker.TrackToolCreated("tool-3", "Write")

	allTools := tracker.GetAllTools()
	if len(allTools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(allTools))
	}

	// Verify all tools are present
	if _, exists := allTools["tool-1"]; !exists {
		t.Error("tool-1 should exist")
	}
	if _, exists := allTools["tool-2"]; !exists {
		t.Error("tool-2 should exist")
	}
	if _, exists := allTools["tool-3"]; !exists {
		t.Error("tool-3 should exist")
	}
}

func TestToolTracker_RemoveTool(t *testing.T) {
	tracker := NewToolTracker()

	tracker.TrackToolCreated("tool-remove", "Read")

	// Verify tool exists
	_, exists := tracker.GetTool("tool-remove")
	if !exists {
		t.Fatal("Tool should exist")
	}

	// Remove the tool
	tracker.RemoveTool("tool-remove")

	// Verify tool no longer exists
	_, exists = tracker.GetTool("tool-remove")
	if exists {
		t.Error("Tool should not exist after removal")
	}
}

func TestToolTracker_UpdatedTime(t *testing.T) {
	tracker := NewToolTracker()
	tracker.TrackToolCreated("tool-time", "Bash")

	tool, _ := tracker.GetTool("tool-time")
	createdAt := tool.CreatedAt
	firstUpdate := tool.UpdatedAt

	// Sleep a bit to ensure time difference
	time.Sleep(10 * time.Millisecond)

	// Update status
	tracker.UpdateToolStatus("tool-time", ToolStatusFinished)

	tool, _ = tracker.GetTool("tool-time")
	if !tool.UpdatedAt.After(firstUpdate) {
		t.Error("UpdatedAt should be updated after status change")
	}
	if !tool.CreatedAt.Equal(createdAt) {
		t.Error("CreatedAt should not change")
	}
}

func TestToolTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewToolTracker()

	// Test concurrent operations
	done := make(chan bool)

	// Goroutine 1: Create tools
	go func() {
		for i := 0; i < 100; i++ {
			tracker.TrackToolCreated(string(rune(i)), "TestTool")
		}
		done <- true
	}()

	// Goroutine 2: Update status
	go func() {
		for i := 0; i < 100; i++ {
			tracker.UpdateToolStatus(string(rune(i)), ToolStatusFinished)
		}
		done <- true
	}()

	// Goroutine 3: Get active tool
	go func() {
		for i := 0; i < 100; i++ {
			tracker.GetActiveTool()
		}
		done <- true
	}()

	// Goroutine 4: Get all tools
	go func() {
		for i := 0; i < 100; i++ {
			tracker.GetAllTools()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}

func TestToolTracker_BackgroundTask(t *testing.T) {
	tracker := NewToolTracker()

	// Create a tool first
	tracker.TrackToolCreated("tool-1", "Bash")

	// Track a background task
	tracker.TrackBackgroundTask("bg-123", "tool-1", "cd web && bun run dev --port 3001", "")

	// Verify background task was created
	task, exists := tracker.GetBackgroundTask("bg-123")
	if !exists {
		t.Fatal("Background task should exist")
	}

	if task.BackgroundTaskID != "bg-123" {
		t.Errorf("Expected background task ID 'bg-123', got '%s'", task.BackgroundTaskID)
	}
	if task.ToolUseID != "tool-1" {
		t.Errorf("Expected tool ID 'tool-1', got '%s'", task.ToolUseID)
	}
	if task.Command != "cd web && bun run dev --port 3001" {
		t.Errorf("Expected command 'cd web && bun run dev --port 3001', got '%s'", task.Command)
	}
	if task.IsTerminated {
		t.Error("IsTerminated should be false initially")
	}
	if task.TerminatedAt != nil {
		t.Error("TerminatedAt should be nil initially")
	}

	// Verify tool has background task ID
	tool, _ := tracker.GetTool("tool-1")
	if tool.BackgroundTaskID != "bg-123" {
		t.Errorf("Expected tool to have background task ID 'bg-123', got '%s'", tool.BackgroundTaskID)
	}
}

func TestToolTracker_TerminateBackgroundTask(t *testing.T) {
	tracker := NewToolTracker()
	tracker.TrackToolCreated("tool-1", "Bash")
	tracker.TrackBackgroundTask("bg-456", "tool-1", "watch -n 1 ls", "")

	// Verify task is active initially
	task, _ := tracker.GetBackgroundTask("bg-456")
	if task.IsTerminated {
		t.Error("Task should not be terminated initially")
	}

	// Terminate the task
	if !tracker.TerminateBackgroundTask("bg-456") {
		t.Error("Should be able to terminate the background task")
	}

	// Verify task is now terminated
	task, exists := tracker.GetBackgroundTask("bg-456")
	if !exists {
		t.Fatal("Background task should still exist")
	}
	if !task.IsTerminated {
		t.Error("Task should be terminated")
	}
	if task.TerminatedAt == nil {
		t.Error("TerminatedAt should not be nil")
	}

	// Try to terminate again (should still return true)
	if !tracker.TerminateBackgroundTask("bg-456") {
		t.Error("Should still return true when terminating already terminated task")
	}

	// Try to terminate non-existent task
	if tracker.TerminateBackgroundTask("non-existent") {
		t.Error("Should return false for non-existent task")
	}
}

func TestToolTracker_GetActiveBackgroundTasks(t *testing.T) {
	tracker := NewToolTracker()

	// No active tasks initially
	activeTasks := tracker.GetActiveBackgroundTasks()
	if len(activeTasks) != 0 {
		t.Error("Should have no active tasks initially")
	}

	// Add some background tasks
	tracker.TrackToolCreated("tool-1", "Bash")
	tracker.TrackToolCreated("tool-2", "Bash")
	tracker.TrackToolCreated("tool-3", "Bash")

	tracker.TrackBackgroundTask("bg-1", "tool-1", "command1", "")
	tracker.TrackBackgroundTask("bg-2", "tool-2", "command2", "")
	tracker.TrackBackgroundTask("bg-3", "tool-3", "command3", "")

	// All should be active
	activeTasks = tracker.GetActiveBackgroundTasks()
	if len(activeTasks) != 3 {
		t.Errorf("Expected 3 active tasks, got %d", len(activeTasks))
	}

	// Terminate one task
	tracker.TerminateBackgroundTask("bg-2")

	// Should have 2 active tasks
	activeTasks = tracker.GetActiveBackgroundTasks()
	if len(activeTasks) != 2 {
		t.Errorf("Expected 2 active tasks, got %d", len(activeTasks))
	}

	// Verify correct tasks are active
	if _, exists := activeTasks["bg-1"]; !exists {
		t.Error("bg-1 should be active")
	}
	if _, exists := activeTasks["bg-2"]; exists {
		t.Error("bg-2 should not be active")
	}
	if _, exists := activeTasks["bg-3"]; !exists {
		t.Error("bg-3 should be active")
	}
}

func TestToolTracker_GetAllBackgroundTasks(t *testing.T) {
	tracker := NewToolTracker()

	// Add some background tasks
	tracker.TrackToolCreated("tool-1", "Bash")
	tracker.TrackToolCreated("tool-2", "Bash")

	tracker.TrackBackgroundTask("bg-1", "tool-1", "command1", "")
	tracker.TrackBackgroundTask("bg-2", "tool-2", "command2", "")

	// Terminate one task
	tracker.TerminateBackgroundTask("bg-1")

	// Get all tasks (both active and terminated)
	allTasks := tracker.GetAllBackgroundTasks()
	if len(allTasks) != 2 {
		t.Errorf("Expected 2 tasks total, got %d", len(allTasks))
	}

	// Verify both tasks exist
	if _, exists := allTasks["bg-1"]; !exists {
		t.Error("bg-1 should exist")
	}
	if _, exists := allTasks["bg-2"]; !exists {
		t.Error("bg-2 should exist")
	}

	// Verify termination status
	if !allTasks["bg-1"].IsTerminated {
		t.Error("bg-1 should be terminated")
	}
	if allTasks["bg-2"].IsTerminated {
		t.Error("bg-2 should not be terminated")
	}
}

func TestToolTracker_RemoveBackgroundTask(t *testing.T) {
	tracker := NewToolTracker()
	tracker.TrackToolCreated("tool-1", "Bash")
	tracker.TrackBackgroundTask("bg-remove", "tool-1", "test command", "")

	// Verify task exists
	_, exists := tracker.GetBackgroundTask("bg-remove")
	if !exists {
		t.Fatal("Background task should exist")
	}

	// Verify tool has background task ID
	tool, _ := tracker.GetTool("tool-1")
	if tool.BackgroundTaskID != "bg-remove" {
		t.Error("Tool should have background task ID")
	}

	// Remove the task
	tracker.RemoveBackgroundTask("bg-remove")

	// Verify task no longer exists
	_, exists = tracker.GetBackgroundTask("bg-remove")
	if exists {
		t.Error("Background task should not exist after removal")
	}

	// Verify tool's background task ID was cleared
	tool, _ = tracker.GetTool("tool-1")
	if tool.BackgroundTaskID != "" {
		t.Error("Tool's background task ID should be cleared")
	}
}

func TestToolTracker_BackgroundTaskConcurrentAccess(t *testing.T) {
	tracker := NewToolTracker()
	done := make(chan bool)

	// Create some tools first
	for i := 0; i < 10; i++ {
		tracker.TrackToolCreated(string(rune('a'+i)), "Bash")
	}

	// Goroutine 1: Track background tasks
	go func() {
		for i := 0; i < 10; i++ {
			tracker.TrackBackgroundTask(string(rune('1'+i)), string(rune('a'+i)), "command", "")
		}
		done <- true
	}()

	// Goroutine 2: Terminate background tasks
	go func() {
		for i := 0; i < 10; i++ {
			tracker.TerminateBackgroundTask(string(rune('1' + i)))
		}
		done <- true
	}()

	// Goroutine 3: Get active tasks
	go func() {
		for i := 0; i < 50; i++ {
			tracker.GetActiveBackgroundTasks()
		}
		done <- true
	}()

	// Goroutine 4: Get all tasks
	go func() {
		for i := 0; i < 50; i++ {
			tracker.GetAllBackgroundTasks()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
