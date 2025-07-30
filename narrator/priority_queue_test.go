package narrator

import (
	"context"
	"testing"
	"time"
)

func TestPriorityQueue_BasicOperations(t *testing.T) {
	pq := NewPriorityQueue()
	ctx := context.Background()

	// Test enqueue
	item := NarrationItem{
		Text:      "Test text",
		Type:      NarrationTypeText,
		Priority:  priorityMap[NarrationTypeText],
		Timestamp: time.Now(),
		ID:        "test-1",
	}

	if !pq.Enqueue(item) {
		t.Error("Failed to enqueue item")
	}

	// Test dequeue
	dequeuedItem := pq.Dequeue(ctx)
	if dequeuedItem == nil {
		t.Error("Failed to dequeue item")
	}
	if dequeuedItem.Text != item.Text {
		t.Errorf("Dequeued item text mismatch: got %s, want %s", dequeuedItem.Text, item.Text)
	}

	// Test size
	if pq.Size() != 0 {
		t.Errorf("Queue size should be 0, got %d", pq.Size())
	}
}

func TestPriorityQueue_ShouldSkip(t *testing.T) {
	pq := NewPriorityQueue()

	// Add items with different priorities
	lowPriority := NarrationItem{
		Text:     "Low priority",
		Type:     NarrationTypeToolUse,
		Priority: priorityMap[NarrationTypeToolUse],
		ID:       "low-1",
	}

	highPriority := NarrationItem{
		Text:     "High priority",
		Type:     NarrationTypeText,
		Priority: priorityMap[NarrationTypeText],
		ID:       "high-1",
	}

	// Enqueue high priority item
	pq.Enqueue(highPriority)

	// Check if low priority item should be skipped
	if !pq.ShouldSkip(lowPriority) {
		t.Error("Low priority item should be skipped when high priority item is in queue")
	}

	// Dequeue high priority item
	ctx := context.Background()
	pq.Dequeue(ctx)

	// Now low priority item should not be skipped
	if pq.ShouldSkip(lowPriority) {
		t.Error("Low priority item should not be skipped when queue is empty")
	}
}

func TestPriorityQueue_MultipleSkips(t *testing.T) {
	pq := NewPriorityQueue()

	// Create items: low, low, high
	low1 := NarrationItem{
		Text:     "Low 1",
		Type:     NarrationTypeToolUse,
		Priority: priorityMap[NarrationTypeToolUse],
		ID:       "low-1",
	}

	low2 := NarrationItem{
		Text:     "Low 2",
		Type:     NarrationTypeToolUseMCP,
		Priority: priorityMap[NarrationTypeToolUseMCP],
		ID:       "low-2",
	}

	high := NarrationItem{
		Text:     "High",
		Type:     NarrationTypeText,
		Priority: priorityMap[NarrationTypeText],
		ID:       "high-1",
	}

	// Enqueue in order: low1, low2, high
	pq.Enqueue(low1)
	pq.Enqueue(low2)
	pq.Enqueue(high)

	ctx := context.Background()

	// Dequeue low1 - it should be skipped
	item1 := pq.Dequeue(ctx)
	if item1 == nil {
		t.Fatal("Expected to dequeue low1")
	}
	if !pq.ShouldSkip(*item1) {
		t.Error("low1 should be skipped due to high priority item in queue")
	}

	// Dequeue low2 - it should also be skipped
	item2 := pq.Dequeue(ctx)
	if item2 == nil {
		t.Fatal("Expected to dequeue low2")
	}
	if !pq.ShouldSkip(*item2) {
		t.Error("low2 should be skipped due to high priority item in queue")
	}

	// Dequeue high - it should not be skipped
	item3 := pq.Dequeue(ctx)
	if item3 == nil {
		t.Fatal("Expected to dequeue high")
	}
	if pq.ShouldSkip(*item3) {
		t.Error("high priority item should not be skipped")
	}
}

func TestPriorityQueue_ContextCancellation(t *testing.T) {
	pq := NewPriorityQueue()
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Try to dequeue with cancelled context
	item := pq.Dequeue(ctx)
	if item != nil {
		t.Error("Dequeue should return nil when context is cancelled")
	}
}

func TestPriorityQueue_Close(t *testing.T) {
	pq := NewPriorityQueue()
	ctx := context.Background()

	// Close the queue
	pq.Close()

	// Try to enqueue after close
	item := NarrationItem{
		Text: "Test",
		ID:   "test-1",
	}
	if pq.Enqueue(item) {
		t.Error("Enqueue should fail after queue is closed")
	}

	// Try to dequeue after close
	dequeuedItem := pq.Dequeue(ctx)
	if dequeuedItem != nil {
		t.Error("Dequeue should return nil after queue is closed")
	}
}

func TestNarrationMetrics(t *testing.T) {
	metrics := NewNarrationMetrics()

	// Test increment operations
	metrics.IncrementQueued()
	metrics.IncrementQueued()
	metrics.IncrementSkipped()
	metrics.IncrementPlayed()
	metrics.IncrementErrors()

	stats := metrics.GetStats()

	if stats["total_queued"] != int64(2) {
		t.Errorf("Expected total_queued to be 2, got %v", stats["total_queued"])
	}
	if stats["total_skipped"] != int64(1) {
		t.Errorf("Expected total_skipped to be 1, got %v", stats["total_skipped"])
	}
	if stats["total_played"] != int64(1) {
		t.Errorf("Expected total_played to be 1, got %v", stats["total_played"])
	}
	if stats["total_errors"] != int64(1) {
		t.Errorf("Expected total_errors to be 1, got %v", stats["total_errors"])
	}

	// Test skip rate
	skipRate := stats["skip_rate"].(float64)
	expectedSkipRate := 50.0 // 1 skipped out of 2 queued
	if skipRate != expectedSkipRate {
		t.Errorf("Expected skip_rate to be %f, got %f", expectedSkipRate, skipRate)
	}
}

func TestPriorityMapping(t *testing.T) {
	// Verify priority order (higher number = higher priority)
	if priorityMap[NarrationTypeToolUse] >= priorityMap[NarrationTypeToolUseMCP] {
		t.Error("ToolUse should have lower priority than ToolUseMCP")
	}
	if priorityMap[NarrationTypeToolUseMCP] >= priorityMap[NarrationTypeToolUsePermission] {
		t.Error("ToolUseMCP should have lower priority than ToolUsePermission")
	}
	if priorityMap[NarrationTypeToolUsePermission] >= priorityMap[NarrationTypeNotification] {
		t.Error("ToolUsePermission should have lower priority than Notification")
	}
	if priorityMap[NarrationTypeNotification] >= priorityMap[NarrationTypeText] {
		t.Error("Notification should have lower priority than Text")
	}
}
