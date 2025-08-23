package event

import (
	"testing"

	"github.com/kazegusuri/claude-companion/handler"
)

// mockFormatter is a simple formatter for testing
type mockFormatter struct{}

func (m *mockFormatter) Format(event Event) (string, error) {
	return "", nil
}

func (m *mockFormatter) SetDebugMode(debug bool) {}

func TestCreateSessionFromEvents(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Handler, Event)
		validate func(t *testing.T, h *Handler, event Event)
	}{
		{
			name: "NotificationEvent with SessionStart creates session",
			setup: func() (*Handler, Event) {
				sessionManager := handler.NewSessionManager()
				h := &Handler{
					sessionManager: sessionManager,
					formatter:      &mockFormatter{},
					buffers:        make(map[string]*BufferInfo),
				}

				event := &NotificationEvent{
					SessionID:      "session-1",
					CWD:            "/test/dir",
					HookEventName:  "SessionStart",
					TranscriptPath: "/test/transcript.jsonl",
				}

				return h, event
			},
			validate: func(t *testing.T, h *Handler, event Event) {
				e := event.(*NotificationEvent)
				// Process the event
				h.processEvent(event)

				// Check if session was created
				session, exists := h.sessionManager.GetSession(e.SessionID)
				if !exists {
					t.Errorf("Session %s was not created", e.SessionID)
					return
				}
				// UUID should be empty for NotificationEvent
				if session.UUID != "" {
					t.Errorf("Session UUID should be empty for NotificationEvent, got %s", session.UUID)
				}
				if session.CWD != e.CWD {
					t.Errorf("Session CWD mismatch: got %s, want %s", session.CWD, e.CWD)
				}
				if session.TranscriptPath != e.TranscriptPath {
					t.Errorf("Session TranscriptPath mismatch: got %s, want %s", session.TranscriptPath, e.TranscriptPath)
				}
			},
		},
		{
			name: "HookEvent with SessionStart creates session",
			setup: func() (*Handler, Event) {
				sessionManager := handler.NewSessionManager()
				h := &Handler{
					sessionManager: sessionManager,
					formatter:      &mockFormatter{},
					buffers:        make(map[string]*BufferInfo),
				}

				parentUUID := "parent-hook-uuid"
				event := &HookEvent{
					BaseEvent: BaseEvent{
						UUID:       "test-uuid-2",
						SessionID:  "session-2",
						CWD:        "/test/dir2",
						TypeString: "hook",
						ParentUUID: &parentUUID,
						Session: &SessionFile{
							Path: "/test/transcript2.jsonl",
						},
					},
					HookEventType: "SessionStart",
				}

				return h, event
			},
			validate: func(t *testing.T, h *Handler, event Event) {
				e := event.(*HookEvent)
				// Process the event
				h.processEvent(event)

				// Check if session was created
				session, exists := h.sessionManager.GetSession(e.SessionID)
				if !exists {
					t.Errorf("Session %s was not created", e.SessionID)
					return
				}
				if session.UUID != e.UUID {
					t.Errorf("Session UUID mismatch: got %s, want %s", session.UUID, e.UUID)
				}
				if session.CWD != e.CWD {
					t.Errorf("Session CWD mismatch: got %s, want %s", session.CWD, e.CWD)
				}
				if session.TranscriptPath != e.Session.Path {
					t.Errorf("Session TranscriptPath mismatch: got %s, want %s", session.TranscriptPath, e.Session.Path)
				}
			},
		},
		{
			name: "HandleWarmupEvent creates session",
			setup: func() (*Handler, Event) {
				sessionManager := handler.NewSessionManager()
				h := &Handler{
					sessionManager: sessionManager,
					formatter:      &mockFormatter{},
					buffers:        make(map[string]*BufferInfo),
				}

				parentUUID := "parent-uuid"
				event := &BaseEvent{
					UUID:        "test-uuid-3",
					SessionID:   "session-3",
					CWD:         "/test/dir3",
					TypeString:  "warmup",
					ParentUUID:  &parentUUID,
					IsSidechain: false,
					Session: &SessionFile{
						Path: "/test/transcript3.jsonl",
					},
				}

				return h, event
			},
			validate: func(t *testing.T, h *Handler, event Event) {
				e := event.(*BaseEvent)

				// Call HandleWarmupEvent directly
				h.HandleWarmupEvent(e)

				// Check if session was created
				session, exists := h.sessionManager.GetSession(e.SessionID)
				if !exists {
					t.Errorf("Session %s was not created by HandleWarmupEvent", e.SessionID)
					return
				}
				if session.UUID != e.UUID {
					t.Errorf("Session UUID mismatch: got %s, want %s", session.UUID, e.UUID)
				}
				if session.CWD != e.CWD {
					t.Errorf("Session CWD mismatch: got %s, want %s", session.CWD, e.CWD)
				}
				if session.TranscriptPath != e.Session.Path {
					t.Errorf("Session TranscriptPath mismatch: got %s, want %s", session.TranscriptPath, e.Session.Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, event := tt.setup()
			tt.validate(t, handler, event)
		})
	}
}
