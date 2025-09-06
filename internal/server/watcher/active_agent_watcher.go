package watcher

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/server/db"
)

// ActiveAgentWatcher monitors active Claude agents and tracks PIDs
type ActiveAgentWatcher struct {
	db       *db.DB
	interval time.Duration
	logger   *slog.Logger

	mu         sync.RWMutex
	activePIDs map[int]struct{}

	// Callbacks for notifications (to be implemented later)
	onAdded   func(pid int, agent db.ClaudeAgent)
	onRemoved func(pid int)
}

// NewActiveAgentWatcher creates a new active agent watcher
func NewActiveAgentWatcher(database *db.DB, interval time.Duration, logger *slog.Logger) *ActiveAgentWatcher {
	return &ActiveAgentWatcher{
		db:         database,
		interval:   interval,
		logger:     logger,
		activePIDs: make(map[int]struct{}),
	}
}

// setOnAdded sets the callback for when a new agent is detected
func (w *ActiveAgentWatcher) setOnAdded(fn func(pid int, agent db.ClaudeAgent)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onAdded = fn
}

// setOnRemoved sets the callback for when an agent is removed
func (w *ActiveAgentWatcher) setOnRemoved(fn func(pid int)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.onRemoved = fn
}

// getActivePIDs returns a copy of the current active PIDs
func (w *ActiveAgentWatcher) getActivePIDs() map[int]struct{} {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[int]struct{}, len(w.activePIDs))
	for pid := range w.activePIDs {
		result[pid] = struct{}{}
	}
	return result
}

// Start begins monitoring active agents
func (w *ActiveAgentWatcher) Start(ctx context.Context) {
	w.logger.Info("Starting active agent watcher", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Initial scan
	w.updateActivePIDs()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping active agent watcher")
			return
		case <-ticker.C:
			w.updateActivePIDs()
		}
	}
}

// updateActivePIDs fetches current agents and updates the active PID list
func (w *ActiveAgentWatcher) updateActivePIDs() {
	agents, err := w.db.ListClaudeAgents()
	if err != nil {
		w.logger.Error("Failed to list claude agents", "error", err)
		return
	}

	// Create new PID map from current agents
	newPIDs := make(map[int]struct{})
	agentMap := make(map[int]db.ClaudeAgent)
	for _, agent := range agents {
		newPIDs[agent.PID] = struct{}{}
		agentMap[agent.PID] = agent
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Find newly added PIDs
	for pid := range newPIDs {
		if _, exists := w.activePIDs[pid]; !exists {
			w.logger.Info("New agent detected",
				"pid", pid,
				"project_dir", agentMap[pid].ProjectDir,
				"session_id", agentMap[pid].SessionID)

			// Call callback if set
			if w.onAdded != nil {
				go w.onAdded(pid, agentMap[pid])
			}
		}
	}

	// Find removed PIDs
	for pid := range w.activePIDs {
		if _, exists := newPIDs[pid]; !exists {
			w.logger.Info("Agent removed", "pid", pid)

			// Call callback if set
			if w.onRemoved != nil {
				go w.onRemoved(pid)
			}
		}
	}

	// Update the active PIDs map
	w.activePIDs = newPIDs

	w.logger.Debug("Active PIDs updated", "count", len(w.activePIDs))
}

// isActive checks if a PID is currently active
func (w *ActiveAgentWatcher) isActive(pid int) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	_, exists := w.activePIDs[pid]
	return exists
}

// getActiveCount returns the number of active agents
func (w *ActiveAgentWatcher) getActiveCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return len(w.activePIDs)
}
