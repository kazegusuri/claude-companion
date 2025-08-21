package narrator

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// NarrationType represents the type of narration
type NarrationType int

const (
	NarrationTypeToolUse NarrationType = iota
	NarrationTypeToolUseMCP
	NarrationTypeToolUsePermission
	NarrationTypeNotification
	NarrationTypeText
)

// Priority mapping for each narration type (higher number = higher priority)
var priorityMap = map[NarrationType]int{
	NarrationTypeToolUse:           1, // Lowest priority
	NarrationTypeToolUseMCP:        2,
	NarrationTypeToolUsePermission: 3,
	NarrationTypeNotification:      4,
	NarrationTypeText:              5, // Highest priority
}

// NarrationItem represents an item in the narration queue
type NarrationItem struct {
	Text         string // Normalized text for TTS
	OriginalText string // Original text before normalization
	Type         NarrationType
	Priority     int
	Timestamp    time.Time
	ID           string
	Meta         *EventMeta // Event metadata for context
}

// PriorityQueue manages narration items with priority-based skipping
type PriorityQueue struct {
	items    []NarrationItem
	mu       sync.Mutex
	notEmpty *sync.Cond
	closed   bool
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]NarrationItem, 0),
	}
	pq.notEmpty = sync.NewCond(&pq.mu)
	return pq
}

// Enqueue adds an item to the queue
func (pq *PriorityQueue) Enqueue(item NarrationItem) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.closed {
		return false
	}

	pq.items = append(pq.items, item)
	pq.notEmpty.Signal()
	return true
}

// Dequeue removes and returns the next item from the queue
// Returns nil if the context is cancelled or queue is closed
func (pq *PriorityQueue) Dequeue(ctx context.Context) *NarrationItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for len(pq.items) == 0 && !pq.closed {
		// Wait for items or cancellation
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				pq.mu.Lock()
				pq.notEmpty.Broadcast()
				pq.mu.Unlock()
			case <-done:
			}
		}()

		pq.notEmpty.Wait()
		select {
		case <-done:
		default:
			close(done)
		}

		if ctx.Err() != nil {
			return nil
		}
	}

	if pq.closed || len(pq.items) == 0 {
		return nil
	}

	item := pq.items[0]
	pq.items = pq.items[1:]
	return &item
}

// ShouldSkip determines if an item should be skipped based on queue priorities
func (pq *PriorityQueue) ShouldSkip(item NarrationItem) bool {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	// Find the highest priority in the remaining queue
	maxPriorityAfter := 0
	for _, queuedItem := range pq.items {
		if queuedItem.Priority > maxPriorityAfter {
			maxPriorityAfter = queuedItem.Priority
		}
	}

	// Skip if current item's priority is lower than any item in the queue
	return item.Priority < maxPriorityAfter
}

// Size returns the current queue size
func (pq *PriorityQueue) Size() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return len(pq.items)
}

// Close closes the queue
func (pq *PriorityQueue) Close() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.closed = true
	pq.notEmpty.Broadcast()
}

// NarrationMetrics tracks performance metrics
type NarrationMetrics struct {
	totalQueued  int64
	totalSkipped int64
	totalPlayed  int64
	totalErrors  int64
	startTime    time.Time
}

// NewNarrationMetrics creates a new metrics tracker
func NewNarrationMetrics() *NarrationMetrics {
	return &NarrationMetrics{
		startTime: time.Now(),
	}
}

// IncrementQueued increments the queued counter
func (m *NarrationMetrics) IncrementQueued() {
	atomic.AddInt64(&m.totalQueued, 1)
}

// IncrementSkipped increments the skipped counter
func (m *NarrationMetrics) IncrementSkipped() {
	atomic.AddInt64(&m.totalSkipped, 1)
}

// IncrementPlayed increments the played counter
func (m *NarrationMetrics) IncrementPlayed() {
	atomic.AddInt64(&m.totalPlayed, 1)
}

// IncrementErrors increments the error counter
func (m *NarrationMetrics) IncrementErrors() {
	atomic.AddInt64(&m.totalErrors, 1)
}

// GetStats returns current statistics
func (m *NarrationMetrics) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_queued":  atomic.LoadInt64(&m.totalQueued),
		"total_skipped": atomic.LoadInt64(&m.totalSkipped),
		"total_played":  atomic.LoadInt64(&m.totalPlayed),
		"total_errors":  atomic.LoadInt64(&m.totalErrors),
		"uptime":        time.Since(m.startTime).String(),
		"skip_rate":     m.getSkipRate(),
	}
}

func (m *NarrationMetrics) getSkipRate() float64 {
	queued := atomic.LoadInt64(&m.totalQueued)
	skipped := atomic.LoadInt64(&m.totalSkipped)
	if queued == 0 {
		return 0
	}
	return float64(skipped) / float64(queued) * 100
}
