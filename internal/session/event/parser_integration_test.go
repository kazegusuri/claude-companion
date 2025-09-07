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
		"HookEventAsSystemMessage": {
			{
				name:  "hook_event_stop",
				input: `{"parentUuid":"c55f08ec-93cc-4e4e-9bfe-3be0035464f3","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"78f17a9d-d4da-4d94-ba71-18a48aac42a3","version":"1.0.64","gitBranch":"main","type":"system","content":"\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-07-31T15:42:02.113Z","uuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","toolUseID":"5a59f1ad-02af-4ddf-b129-3af63d9d0049","level":"info"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "ef16ec60-d3f6-4d59-bd99-d903bcddd8da",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/tmp/test/project",
						Timestamp:   mustParseTime("2025-07-31T15:42:02.113Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "78f17a9d-d4da-4d94-ba71-18a48aac42a3",
						TranscriptPath: "",
					},
					RawContent: "\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
					Content: &internalevent.HookSystemMessageContent{
						HookName: "Stop",
						Command:  "/usr/local/bin/claude-notification.sh",
						Status:   "completed successfully",
						Type:     "",
						Message:  "\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
					},
					Level:     "info",
					ToolUseID: "",
				},
			},
			{
				name:  "hook_event_session_start_resume",
				input: `{"parentUuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d99240fe-3539-438d-85c6-c51f5eb51902","version":"1.0.67","gitBranch":"feature/test","type":"system","content":"\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-08-03T13:09:46.461Z","uuid":"aa1fc221-60fc-4756-a892-93ffecbd47b9","toolUseID":"e51379a0-afd9-4434-bb3b-40cd178a0dc6","level":"info"}`,
				wantEvent: &internalevent.SystemMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "aa1fc221-60fc-4756-a892-93ffecbd47b9",
						Type:        internalevent.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/tmp/test/project",
						Timestamp:   mustParseTime("2025-08-03T13:09:46.461Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "d99240fe-3539-438d-85c6-c51f5eb51902",
						TranscriptPath: "",
					},
					RawContent: "\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
					Content: &internalevent.HookSystemMessageContent{
						HookName: "SessionStart",
						Command:  "/usr/local/bin/claude-notification.sh",
						Status:   "completed successfully",
						Type:     "resume",
						Message:  "\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
					},
					Level:     "info",
					ToolUseID: "",
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
					case *SystemMessage, *SummaryEvent, *NotificationEvent, *HookEvent:
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


