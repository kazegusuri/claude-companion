package handler

import (
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
)

// ToolInfo represents information about a tool
type ToolInfo struct {
	ToolUseID         string    `json:"toolUseId"`
	ToolName          string    `json:"toolName"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
	IsError           bool      `json:"isError"`
	IsRejected        bool      `json:"isRejected"`
	IsWaitingApproval bool      `json:"isWaitingApproval"`
}

// BackgroundTaskInfo represents information about a background task
type BackgroundTaskInfo struct {
	BackgroundTaskID string     `json:"backgroundTaskId"`
	ToolUseID        string     `json:"toolUseId"`
	Command          string     `json:"command"`
	Description      string     `json:"description"`
	IsTerminated     bool       `json:"isTerminated"`
	CreatedAt        time.Time  `json:"createdAt"`
	TerminatedAt     *time.Time `json:"terminatedAt,omitempty"`
}

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

	// Tool and background task information
	activeTool      *ToolInfo                      `json:"activeTool,omitempty"`
	backgroundTasks map[string]*BackgroundTaskInfo `json:"backgroundTasks,omitempty"`

	// Mutex for thread-safe updates
	mu sync.RWMutex
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
		SessionID:       sessionID,
		UUID:            uuid,
		CWD:             cwd,
		TranscriptPath:  transcriptPath,
		StartTime:       time.Now(),
		backgroundTasks: make(map[string]*BackgroundTaskInfo),
	}

	sm.sessions[sessionID] = session
	logger.DebugInfo("New session created: %s (UUID: %s, CWD: %s, Transcript: %s)", sessionID, uuid, cwd, transcriptPath)

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

// UpdateActiveTool updates the active tool information for a session
func (s *Session) UpdateActiveTool(tool *ToolInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activeTool = tool
}

// GetActiveTool returns the active tool information
func (s *Session) GetActiveTool() *ToolInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.activeTool == nil {
		return nil
	}
	// Return a copy to avoid race conditions
	toolCopy := *s.activeTool
	return &toolCopy
}

// AddBackgroundTask adds or updates a background task
func (s *Session) AddBackgroundTask(task *BackgroundTaskInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backgroundTasks == nil {
		s.backgroundTasks = make(map[string]*BackgroundTaskInfo)
	}
	s.backgroundTasks[task.BackgroundTaskID] = task
}

// RemoveBackgroundTask removes a background task by ID
func (s *Session) RemoveBackgroundTask(backgroundTaskID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.backgroundTasks, backgroundTaskID)
}

// UpdateBackgroundTask updates an existing background task
func (s *Session) UpdateBackgroundTask(backgroundTaskID string, isTerminated bool, terminatedAt *time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if task, exists := s.backgroundTasks[backgroundTaskID]; exists {
		task.IsTerminated = isTerminated
		if terminatedAt != nil {
			task.TerminatedAt = terminatedAt
		}
	}
}

// GetBackgroundTasks returns a copy of all background tasks
func (s *Session) GetBackgroundTasks() map[string]*BackgroundTaskInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid race conditions
	copy := make(map[string]*BackgroundTaskInfo)
	for k, v := range s.backgroundTasks {
		taskCopy := *v
		if v.TerminatedAt != nil {
			terminatedTime := *v.TerminatedAt
			taskCopy.TerminatedAt = &terminatedTime
		}
		copy[k] = &taskCopy
	}
	return copy
}

// GetActiveBackgroundTasks returns only active (non-terminated) background tasks
func (s *Session) GetActiveBackgroundTasks() map[string]*BackgroundTaskInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeTasks := make(map[string]*BackgroundTaskInfo)
	for k, v := range s.backgroundTasks {
		if !v.IsTerminated {
			taskCopy := *v
			activeTasks[k] = &taskCopy
		}
	}
	return activeTasks
}
