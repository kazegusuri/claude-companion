package event

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
)

// SessionFileManager manages multiple SessionWatcher instances
type SessionFileManager struct {
	watchers       map[string]*ManagedWatcher
	mu             sync.RWMutex
	handlerBuilder *HandlerBuilder
	handlers       map[string]*Handler // Map of file path to handler

	// Configuration
	idleTimeout   time.Duration
	checkInterval time.Duration

	done chan struct{}
	wg   sync.WaitGroup
}

// ManagedWatcher wraps a SessionWatcher with metadata
type ManagedWatcher struct {
	watcher      *SessionWatcher
	handler      *Handler
	lastActivity time.Time
	filePath     string
}

// NewSessionFileManager creates a new session file manager
func NewSessionFileManager(handlerBuilder *HandlerBuilder) *SessionFileManager {
	return &SessionFileManager{
		watchers:       make(map[string]*ManagedWatcher),
		handlerBuilder: handlerBuilder,
		handlers:       make(map[string]*Handler),
		idleTimeout:    1 * time.Hour,   // Remove watchers after 1 hour of inactivity
		checkInterval:  1 * time.Minute, // Check for idle watchers every minute
		done:           make(chan struct{}),
	}
}

// Start begins the manager's cleanup routine
func (m *SessionFileManager) Start() {
	m.wg.Add(1)
	go m.cleanupRoutine()
}

// Stop stops the manager and all managed watchers
func (m *SessionFileManager) Stop() {
	close(m.done)
	m.wg.Wait()

	// Stop all watchers
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, mw := range m.watchers {
		mw.watcher.Stop()
	}
	m.watchers = make(map[string]*ManagedWatcher)
	m.handlers = make(map[string]*Handler)
}

// AddOrUpdateWatcher adds a new watcher or updates the activity time
func (m *SessionFileManager) AddOrUpdateWatcher(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if watcher already exists
	if mw, exists := m.watchers[filePath]; exists {
		mw.lastActivity = time.Now()
		if logger.IsDebugMode() {
			logger.LogInfo("Updated activity time for watcher: %s", filePath)
		}
		return nil
	}

	// Create new watcher with the builder (it will create handler and parser internally)
	watcher := NewSessionWatcher(filePath, m.handlerBuilder)

	// Store the handler for later reference
	m.handlers[filePath] = watcher.eventHandler
	if err := watcher.Start(); err != nil {
		return err
	}

	m.watchers[filePath] = &ManagedWatcher{
		watcher:      watcher,
		handler:      watcher.eventHandler,
		lastActivity: time.Now(),
		filePath:     filePath,
	}

	if logger.IsDebugMode() {
		logger.LogInfo("Created new session watcher for: %s", filePath)
	}
	return nil
}

// cleanupRoutine periodically removes idle watchers
func (m *SessionFileManager) cleanupRoutine() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupIdleWatchers()
		case <-m.done:
			return
		}
	}
}

// cleanupIdleWatchers removes watchers that have been idle for too long
func (m *SessionFileManager) cleanupIdleWatchers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	toRemove := []string{}

	for path, mw := range m.watchers {
		if now.Sub(mw.lastActivity) > m.idleTimeout {
			toRemove = append(toRemove, path)
		}
	}

	for _, path := range toRemove {
		if mw, exists := m.watchers[path]; exists {
			mw.watcher.Stop()
			delete(m.watchers, path)
			// Also remove the handler
			delete(m.handlers, path)
			if logger.IsDebugMode() {
				logger.LogInfo("Removed idle session watcher for: %s", path)
			}
		}
	}

	if logger.IsDebugMode() && len(toRemove) > 0 {
		logger.LogInfo("Cleaned up %d idle watchers", len(toRemove))
	}
}

// GetActiveWatcherCount returns the number of active watchers
func (m *SessionFileManager) GetActiveWatcherCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.watchers)
}

// ExtractSessionIDFromPath extracts the session ID from a session file path
// Example: /path/to/project/session-20240101-123456.jsonl -> session-20240101-123456
func ExtractSessionIDFromPath(filePath string) string {
	// Get the base filename without extension
	base := filepath.Base(filePath)
	// Remove the .jsonl extension
	sessionID := strings.TrimSuffix(base, ".jsonl")
	return sessionID
}
