package event

import (
	"testing"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// mockFormatter is a simple formatter for testing
type mockFormatter struct{}

func (m *mockFormatter) Format(event Event) (string, error) {
	return "", nil
}

func (m *mockFormatter) SetDebugMode(debug bool) {}

// mockCentralHandler captures events sent to central handler
type mockCentralHandler struct {
	events []internalevent.Event
}

func (m *mockCentralHandler) SendEvent(event internalevent.Event) {
	m.events = append(m.events, event)
}

func TestCreateSessionFromEvents(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Handler, Event)
		validate func(t *testing.T, h *Handler, event Event)
	}{
		{
			name: "NotificationEvent is sent to central handler",
			setup: func() (*Handler, Event) {
				sessionManager := handler.NewSessionManager()
				// Use mock central handler to verify event is sent
				mockCentral := &mockCentralHandler{}
				h := &Handler{
					sessionManager: sessionManager,
					centralHandler: mockCentral,
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
				// Process the event (this will send to central handler)
				h.processEvent(event)

				// Check that event was sent to central handler
				mockCentral := h.centralHandler.(*mockCentralHandler)
				if len(mockCentral.events) != 1 {
					t.Errorf("Expected 1 event sent to central handler, got %d", len(mockCentral.events))
					return
				}

				// Verify the sent event
				sentEvent, ok := mockCentral.events[0].(*internalevent.NotificationEvent)
				if !ok {
					t.Errorf("Expected *internalevent.NotificationEvent, got %T", mockCentral.events[0])
					return
				}

				if sentEvent.Session.SessionID != e.SessionID {
					t.Errorf("SessionID mismatch: got %s, want %s", sentEvent.Session.SessionID, e.SessionID)
				}
				if sentEvent.Session.TranscriptPath != e.TranscriptPath {
					t.Errorf("TranscriptPath mismatch: got %s, want %s", sentEvent.Session.TranscriptPath, e.TranscriptPath)
				}
				if sentEvent.HookEventName != e.HookEventName {
					t.Errorf("HookEventName mismatch: got %s, want %s", sentEvent.HookEventName, e.HookEventName)
				}
				// NotificationEvent doesn't have TranscriptPath directly, it's in Session.TranscriptPath
				// For session/event NotificationEvent, e.TranscriptPath is the field
				// For internal/event NotificationEvent, it's in sentEvent.Session.TranscriptPath
			},
		},
		{
			name: "HookEvent with SessionStart creates session",
			setup: func() (*Handler, Event) {
				sessionManager := handler.NewSessionManager()
				mockNarr := &mockNarrator{}
				mockPrint := &mockPrinter{}
				centralHandler := internalevent.NewHandler(sessionManager, mockNarr, mockPrint, nil)
				h := &Handler{
					sessionManager: sessionManager,
					centralHandler: centralHandler,
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
				mockNarr := &mockNarrator{}
				mockPrint := &mockPrinter{}
				centralHandler := internalevent.NewHandler(sessionManager, mockNarr, mockPrint, nil)
				h := &Handler{
					sessionManager: sessionManager,
					centralHandler: centralHandler,
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
