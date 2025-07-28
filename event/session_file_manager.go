package event

import (
	"log"
	"sync"
	"time"
)

// SessionFileManager manages multiple SessionWatcher instances
type SessionFileManager struct {
	watchers map[string]*ManagedWatcher
	mu       sync.RWMutex
	handler  *Handler

	// Configuration
	idleTimeout   time.Duration
	checkInterval time.Duration

	done chan struct{}
	wg   sync.WaitGroup
}

// ManagedWatcher wraps a SessionWatcher with metadata
type ManagedWatcher struct {
	watcher      *SessionWatcher
	lastActivity time.Time
	filePath     string
}

// NewSessionFileManager creates a new session file manager
func NewSessionFileManager(handler *Handler) *SessionFileManager {
	return &SessionFileManager{
		watchers:      make(map[string]*ManagedWatcher),
		handler:       handler,
		idleTimeout:   1 * time.Hour,   // Remove watchers after 1 hour of inactivity
		checkInterval: 1 * time.Minute, // Check for idle watchers every minute
		done:          make(chan struct{}),
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
}

// AddOrUpdateWatcher adds a new watcher or updates the activity time
func (m *SessionFileManager) AddOrUpdateWatcher(filePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if watcher already exists
	if mw, exists := m.watchers[filePath]; exists {
		mw.lastActivity = time.Now()
		log.Printf("Updated activity time for watcher: %s", filePath)
		return nil
	}

	// Create new watcher
	watcher := NewSessionWatcher(filePath, m.handler)
	if err := watcher.Start(); err != nil {
		return err
	}

	m.watchers[filePath] = &ManagedWatcher{
		watcher:      watcher,
		lastActivity: time.Now(),
		filePath:     filePath,
	}

	log.Printf("Created new session watcher for: %s", filePath)
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
			log.Printf("Removed idle session watcher for: %s", path)
		}
	}

	if len(toRemove) > 0 {
		log.Printf("Cleaned up %d idle watchers", len(toRemove))
	}
}

// GetActiveWatcherCount returns the number of active watchers
func (m *SessionFileManager) GetActiveWatcherCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.watchers)
}
