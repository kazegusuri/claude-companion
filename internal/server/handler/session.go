package handler

import (
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
)

// SessionGetter is an interface for getting session information
type SessionGetter interface {
	GetSession(sessionID string) (*Session, bool)
}

// Session represents an active session
type Session struct {
	SessionID      string    `json:"sessionId"`
	UUID           string    `json:"uuid"` // UUID of the event that created this session
	CWD            string    `json:"cwd"`
	TranscriptPath string    `json:"transcriptPath"` // Path to the transcript file
	StartTime      time.Time `json:"startTime"`      // When the session started
}

// SessionManager manages all active sessions
type SessionManager struct {
	sessions map[string]*Session // key: sessionID
	mu       sync.RWMutex
}

// NewSessionManager creates a new SessionManager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// CreateSession creates a new session and registers it
func (sm *SessionManager) CreateSession(sessionID, uuid, cwd, transcriptPath string) *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		SessionID:      sessionID,
		UUID:           uuid,
		CWD:            cwd,
		TranscriptPath: transcriptPath,
		StartTime:      time.Now(),
	}

	sm.sessions[sessionID] = session
	logger.LogInfo("New session created: %s (UUID: %s, CWD: %s, Transcript: %s)", sessionID, uuid, cwd, transcriptPath)

	return session
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(sessionID string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// RemoveSession removes a session from the manager
func (sm *SessionManager) RemoveSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.sessions[sessionID]; exists {
		delete(sm.sessions, sessionID)
		logger.LogInfo("Session removed: %s", sessionID)
		return true
	}
	return false
}

// GetAllSessions returns a copy of all sessions
func (sm *SessionManager) GetAllSessions() map[string]*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create a copy to avoid external modifications
	copy := make(map[string]*Session)
	for k, v := range sm.sessions {
		copy[k] = v
	}
	return copy
}
