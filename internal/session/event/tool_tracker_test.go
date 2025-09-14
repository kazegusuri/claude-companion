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

	// Valid transition: Created -> WaitingApproval
	if !tracker.UpdateToolStatus("tool-1", ToolStatusWaitingApproval) {
		t.Error("Should allow transition from Created to WaitingApproval")
	}

	tool, _ := tracker.GetTool("tool-1")
	if tool.Status != ToolStatusWaitingApproval {
		t.Errorf("Expected status 'waiting_approval', got '%s'", tool.Status)
	}

	// Valid transition: WaitingApproval -> Running
	if !tracker.UpdateToolStatus("tool-1", ToolStatusRunning) {
		t.Error("Should allow transition from WaitingApproval to Running")
	}

	tool, _ = tracker.GetTool("tool-1")
	if tool.Status != ToolStatusRunning {
		t.Errorf("Expected status 'running', got '%s'", tool.Status)
	}

	// Valid transition: Running -> Finished
	if !tracker.UpdateToolStatus("tool-1", ToolStatusFinished) {
		t.Error("Should allow transition from Running to Finished")
	}

	tool, _ = tracker.GetTool("tool-1")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Expected status 'finished', got '%s'", tool.Status)
	}

	// Invalid transition: Finished -> Running (cannot go backwards)
	if tracker.UpdateToolStatus("tool-1", ToolStatusRunning) {
		t.Error("Should not allow transition from Finished to Running")
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

	// Move to Running
	tracker.UpdateToolStatus("tool-1", ToolStatusRunning)

	// Invalid transition: Running -> Created (cannot go backwards)
	if tracker.UpdateToolStatus("tool-1", ToolStatusCreated) {
		t.Error("Should not allow transition from Running to Created")
	}

	// Invalid transition: Running -> WaitingApproval (cannot go backwards)
	if tracker.UpdateToolStatus("tool-1", ToolStatusWaitingApproval) {
		t.Error("Should not allow transition from Running to WaitingApproval")
	}

	// Status should remain Running
	tool, _ := tracker.GetTool("tool-1")
	if tool.Status != ToolStatusRunning {
		t.Errorf("Status should remain 'running', got '%s'", tool.Status)
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

	// Move to WaitingApproval
	tracker.UpdateToolStatus("tool-1", ToolStatusWaitingApproval)

	// Finish with rejection
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

	// PreToolUse completed - waiting for approval
	tracker.UpdateToolStatus("tool-reject", ToolStatusWaitingApproval)

	// User rejects - finish immediately
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

	// PreToolUse completed - waiting for approval
	tracker.UpdateToolStatus("tool-approve", ToolStatusWaitingApproval)

	// Tool executed (approved) - running
	tracker.UpdateToolStatus("tool-approve", ToolStatusRunning)

	// PostToolUse completed - finished
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
	tracker.UpdateToolStatus("tool-time", ToolStatusRunning)

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
			tracker.UpdateToolStatus(string(rune(i)), ToolStatusRunning)
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
