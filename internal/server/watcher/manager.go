package watcher

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/server/db"
)

// Manager manages all watchers
type Manager struct {
	db                 *db.DB
	logger             *slog.Logger
	staleAgentWatcher  *StaleAgentWatcher
	activeAgentWatcher *ActiveAgentWatcher

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewManager creates a new watcher manager
func NewManager(database *db.DB, logger *slog.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		db:     database,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start starts all watchers
func (m *Manager) Start() {
	m.logger.Info("Starting watcher manager")

	// Initialize and start stale agent watcher (cleanup every 5 minutes)
	m.staleAgentWatcher = NewStaleAgentWatcher(m.db, 5*time.Minute, m.logger)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.staleAgentWatcher.Start(m.ctx)
	}()

	// Initialize and start active agent watcher (monitor every 5 seconds)
	m.activeAgentWatcher = NewActiveAgentWatcher(m.db, 5*time.Second, m.logger)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.activeAgentWatcher.Start(m.ctx)
	}()

	m.logger.Info("All watchers started")
}

// Stop stops all watchers gracefully
func (m *Manager) Stop() {
	m.logger.Info("Stopping watcher manager")

	// Cancel context to signal all watchers to stop
	m.cancel()

	// Wait for all watchers to finish
	m.wg.Wait()

	m.logger.Info("All watchers stopped")
}

// SetOnAgentAdded sets the callback for when a new agent is detected
func (m *Manager) SetOnAgentAdded(fn func(pid int, agent db.ClaudeAgent)) {
	if m.activeAgentWatcher != nil {
		m.activeAgentWatcher.setOnAdded(fn)
	}
}

// SetOnAgentRemoved sets the callback for when an agent is removed
func (m *Manager) SetOnAgentRemoved(fn func(pid int)) {
	if m.activeAgentWatcher != nil {
		m.activeAgentWatcher.setOnRemoved(fn)
	}
}
