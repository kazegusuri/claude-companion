package event

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// mustParseTime is a helper function to parse time in tests
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestIntegration_ParseAndSendToCentral tests parsing JSON input and sending to central handler
func TestIntegration_ParseAndSendToCentral(t *testing.T) {
	testGroups := map[string][]struct {
		name      string
		input     string
		wantEvent interface{} // Expected event sent to central handler
	}{
		"SystemMessage": {
			{
				name:  "simple",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test-session","cwd":"/test/dir","content":"Tool execution completed","isMeta":false}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "123",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					RawContent: "Tool execution completed",
					Level:   "",
				},
			},
			{
				name:  "with_level",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test-session","cwd":"/test/dir","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "123",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					RawContent: "Rate limit warning",
					Level:   "warning",
				},
			},
			{
				name:  "with_error",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"456","sessionID":"test-session","cwd":"/test/dir","content":"API error occurred","isMeta":false,"level":"error"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "456",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					RawContent: "API error occurred",
					Level:   "error",
				},
			},
			{
				name:  "with_tooluse",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"789","sessionID":"test-session","cwd":"/test/dir","content":"Tool execution started","isMeta":false,"toolUseID":"toolu_123"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "789",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					RawContent: "Tool execution started",
					Level:     "",
					ToolUseID: "toolu_123",
				},
			},
			{
				name:  "meta",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"meta-123","sessionID":"test-session","cwd":"/test/dir","content":"Meta message","isMeta":true,"level":"info"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "meta-123",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      true,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					RawContent: "Meta message",
					Level:   "info",
				},
			},
			{
				name:  "debug_with_tool",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"debug-456","sessionID":"test-session","cwd":"/test/workspace","content":"Debug info","isMeta":true,"level":"debug","toolUseID":"tool-456"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "debug-456",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/workspace",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      true,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					RawContent: "Debug info",
					Level:     "debug",
					ToolUseID: "tool-456",
				},
			},
		},
		"SummaryEvent": {
			{
				name:  "basic",
				input: `{"type":"summary","summary":"Summary text","leafUuid":"leaf_123"}`,
				wantEvent: &internalevent.SummaryEvent{
					Session: internalevent.Session{
						SessionID:      "",
						TranscriptPath: "",
					},
					LeafUUID:  "leaf_123",
					Summary:   "Summary text",
				},
			},
			{
				name:  "with_long_text",
				input: `{"type":"summary","summary":"This is a longer summary text that contains multiple sentences. It describes what happened in the session.","leafUuid":"leaf_xyz"}`,
				wantEvent: &internalevent.SummaryEvent{
					Session: internalevent.Session{
						SessionID:      "",
						TranscriptPath: "",
					},
					LeafUUID:  "leaf_xyz",
					Summary:   "This is a longer summary text that contains multiple sentences. It describes what happened in the session.",
				},
			},
		},
		"NotificationEvent": {
			{
				name:  "session_start",
				input: `{"type":"notification","sessionID":"notif-session","transcriptPath":"/test/transcript.jsonl","cwd":"/test/dir","hookEventName":"SessionStart","message":"Session started","trigger":"startup","customInstructions":"","source":"startup"}`,
				wantEvent: &internalevent.NotificationEvent{
					Session: internalevent.Session{
						SessionID:      "notif-session",
						TranscriptPath: "/test/transcript.jsonl",
					},
					HookEventName:      "SessionStart",
					Message:            "Session started",
					Trigger:            "startup",
					CustomInstructions: "",
					Source:             "startup",
				},
			},
			{
				name:  "precompact",
				input: `{"type":"notification","sessionID":"compact-session","transcriptPath":"/logs/session.jsonl","cwd":"/workspace","hookEventName":"PreCompact","message":"Compacting conversation","trigger":"","customInstructions":""}`,
				wantEvent: &internalevent.NotificationEvent{
					Session: internalevent.Session{
						SessionID:      "compact-session",
						TranscriptPath: "/logs/session.jsonl",
					},
					HookEventName:      "PreCompact",
					Message:            "Compacting conversation",
					Trigger:            "",
					CustomInstructions: "",
					Source:             "",
				},
			},
			{
				name:  "permission",
				input: `{"type":"notification","sessionID":"perm-session","transcriptPath":"","cwd":"/test","hookEventName":"Notification","message":"Claude will use WebSearch: approve","trigger":"","customInstructions":""}`,
				wantEvent: &internalevent.NotificationEvent{
					Session: internalevent.Session{
						SessionID:      "perm-session",
						TranscriptPath: "",
					},
					HookEventName:      "Notification",
					Message:            "Claude will use WebSearch: approve",
					Trigger:            "",
					CustomInstructions: "",
					Source:             "",
				},
			},
		},
		"HookEvent": {
			{
				name:  "hook_event_precompact",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"hook-123","sessionID":"hook-session","cwd":"/test/workspace","content":"PreCompact [/usr/local/bin/hook.sh] completed","isMeta":false,"toolUseID":"tool-456","level":"info"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "hook-123",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/workspace",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "hook-session",
						TranscriptPath: "",
					},
					RawContent: "PreCompact [/usr/local/bin/hook.sh] completed",
					Content: &internalevent.HookSystemMessageContent{
						HookName: "PreCompact",
						Command:  "/usr/local/bin/hook.sh",
						Status:   "completed",
						Type:     "PreCompact",
						Message:  "PreCompact [/usr/local/bin/hook.sh] completed",
					},
					Level:     "info",
					ToolUseID: "tool-456",
				},
			},
			{
				name:  "hook_event_sessionstart_resume",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"hook-456","sessionID":"session-start","cwd":"/workspace","content":"SessionStart:resume [/bin/start.sh] completed","isMeta":true,"level":"debug"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "hook-456",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/workspace",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      true,
					},
					Session: internalevent.Session{
						SessionID:      "session-start",
						TranscriptPath: "",
					},
					RawContent: "SessionStart:resume [/bin/start.sh] completed",
					Content: &internalevent.HookSystemMessageContent{
						HookName: "SessionStart",
						Command:  "/bin/start.sh",
						Status:   "completed",
						Type:     "SessionStart:resume",
						Message:  "SessionStart:resume [/bin/start.sh] completed",
					},
					Level: "debug",
				},
			},
			{
				name:  "hook_event_stop",
				input: `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"hook-789","sessionID":"stop-session","cwd":"/tmp","content":"Stop [/usr/bin/cleanup.sh] failed","isMeta":false,"level":"error","toolUseID":"tool-cleanup"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "hook-789",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/tmp",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "stop-session",
						TranscriptPath: "",
					},
					RawContent: "Stop [/usr/bin/cleanup.sh] failed",
					Content: &internalevent.HookSystemMessageContent{
						HookName: "Stop",
						Command:  "/usr/bin/cleanup.sh",
						Status:   "failed",
						Type:     "Stop",
						Message:  "Stop [/usr/bin/cleanup.sh] failed",
					},
					Level:     "error",
					ToolUseID: "tool-cleanup",
				},
			},
		},
	}

	for groupName, tests := range testGroups {
		t.Run(groupName, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// Create parser
					parser := NewParser()

					// Parse the event
					event, err := parser.Parse(tt.input)
					if err != nil {
						t.Fatalf("Parse() error = %v", err)
					}

					// Create mock central handler
					mockCentral := &mockCentralHandler{}

					// Create session manager
					sessionManager := handler.NewSessionManager()

					// Create handler with mock central handler
					h := &Handler{
						sessionManager: sessionManager,
						centralHandler: mockCentral,
						formatter:      &mockFormatter{},
						buffers:        make(map[string]*BufferInfo),
					}

					// Process the event
					h.processEvent(event)

					// Check events sent to central handler based on event type
					switch event.(type) {
					case *SystemMessage, *SummaryEvent, *NotificationEvent:
						// These should be sent to central handler
						if len(mockCentral.events) != 1 {
							t.Errorf("Expected 1 event sent to central handler, got %d", len(mockCentral.events))
							return
						}

						gotEvent := mockCentral.events[0]

						// Compare using cmp.Diff based on expected type
						opts := []cmp.Option{
							cmpopts.IgnoreFields(internalevent.SystemMessage{}, "Narration"),
							cmpopts.IgnoreFields(internalevent.SummaryEvent{}, "Narration"),
							cmpopts.IgnoreFields(internalevent.NotificationEvent{}, "Narration"),
						}

						if diff := cmp.Diff(tt.wantEvent, gotEvent, opts...); diff != "" {
							t.Errorf("%s mismatch (-want +got):\n%s", groupName, diff)
						}

					default:
						// Other event types should not be sent to central handler
						if len(mockCentral.events) != 0 {
							t.Errorf("Expected no events sent to central handler for %T, got %d", event, len(mockCentral.events))
						}
					}
				})
			}
		})
	}
}


