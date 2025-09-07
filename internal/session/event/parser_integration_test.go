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
		name            string
		input           string
		wantEvent       interface{} // Expected event sent to central handler
		expectEventSent bool        // Whether event should be sent to central handler
	}{
		"SystemMessage": {
			{
				name:            "simple",
				input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test-session","cwd":"/test/dir","content":"Tool execution completed","isMeta":false}`,
				expectEventSent: true,
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
					Level:      "",
				},
			},
			{
				name:            "with_level",
				input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test-session","cwd":"/test/dir","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
				expectEventSent: true,
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
					Level:      "warning",
				},
			},
			{
				name:            "with_error",
				input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"456","sessionID":"test-session","cwd":"/test/dir","content":"API error occurred","isMeta":false,"level":"error"}`,
				expectEventSent: true,
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
					Level:      "error",
				},
			},
			{
				name:            "with_tooluse",
				input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"789","sessionID":"test-session","cwd":"/test/dir","content":"Tool execution started","isMeta":false,"toolUseID":"toolu_123"}`,
				expectEventSent: true,
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
					Level:      "",
					ToolUseID:  "toolu_123",
				},
			},
			{
				name:            "meta",
				input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"meta-123","sessionID":"test-session","cwd":"/test/dir","content":"Meta message","isMeta":true,"level":"info"}`,
				expectEventSent: true,
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
					Level:      "info",
				},
			},
			{
				name:            "debug_with_tool",
				input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"debug-456","sessionID":"test-session","cwd":"/test/workspace","content":"Debug info","isMeta":true,"level":"debug","toolUseID":"tool-456"}`,
				expectEventSent: true,
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
					Level:      "debug",
					ToolUseID:  "tool-456",
				},
			},
		},
		"SummaryEvent": {
			{
				name:            "basic",
				input:           `{"type":"summary","summary":"Summary text","leafUuid":"leaf_123"}`,
				expectEventSent: true,
				wantEvent: &internalevent.SummaryEvent{
					Session: internalevent.Session{
						SessionID:      "",
						TranscriptPath: "",
					},
					LeafUUID: "leaf_123",
					Summary:  "Summary text",
				},
			},
			{
				name:            "with_long_text",
				input:           `{"type":"summary","summary":"This is a longer summary text that contains multiple sentences. It describes what happened in the session.","leafUuid":"leaf_xyz"}`,
				expectEventSent: true,
				wantEvent: &internalevent.SummaryEvent{
					Session: internalevent.Session{
						SessionID:      "",
						TranscriptPath: "",
					},
					LeafUUID: "leaf_xyz",
					Summary:  "This is a longer summary text that contains multiple sentences. It describes what happened in the session.",
				},
			},
		},
		"UserMessage": {
			{
				name:            "simple_string_content",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-123","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Hello, this is a test message"},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-123",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentMessage{
							Text: "Hello, this is a test message",
						},
					},
				},
			},
			{
				name:            "multiline_content",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-456","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Line 1\nLine 2\nLine 3"},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-456",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentMessage{
							Text: "Line 1\nLine 2\nLine 3",
						},
					},
				},
			},
			{
				name:            "meta_message",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-789","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Meta message"},"isMeta":true}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-789",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      true,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentMessage{
							Text: "Meta message",
						},
					},
				},
			},
			{
				name:            "with_sidechain",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-side","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Sidechain message"},"isSidechain":true,"isMeta":false}`,
				expectEventSent: false, // Sidechain messages are ignored
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-side",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: true,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentMessage{
							Text: "Sidechain message",
						},
					},
				},
			},
			{
				name:            "empty_content",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-empty","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":""},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-empty",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentMessage{
							Text: "",
						},
					},
				},
			},
			{
				name:            "local_command_stdout_empty",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-empty","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<local-command-stdout></local-command-stdout>"},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-local-empty",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentLocalCommand{
							Output: "",
						},
					},
				},
			},
			{
				name:            "local_command_stdout_no_content",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-no-content","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<local-command-stdout>(no content)</local-command-stdout>"},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-local-no-content",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentLocalCommand{
							Output: "",
						},
					},
				},
			},
			{
				name:            "local_command_stdout_with_output",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-output","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<local-command-stdout>Command executed successfully\nOutput line 1\nOutput line 2</local-command-stdout>"},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-local-output",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentLocalCommand{
							Output: "Command executed successfully\nOutput line 1\nOutput line 2",
						},
					},
				},
			},
			{
				name:            "local_command_stdout_in_array",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-array","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"<local-command-stdout>Array output</local-command-stdout>"}]},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-local-array",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentList{
							Items: []internalevent.UserMessageContentItem{
								&internalevent.UserMessageContentLocalCommand{
									Output: "Array output",
								},
							},
						},
					},
				},
			},
			{
				name:            "command_clear",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-cmd-clear","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<command-name>/clear</command-name>\n<command-message>clear</command-message>\n<command-args></command-args>"},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-cmd-clear",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentCommand{
							CommandName:    "/clear",
							CommandMessage: "clear",
							CommandArgs:    "",
						},
					},
				},
			},
			{
				name:            "command_with_args",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-cmd-search","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"<command-name>/search</command-name>\n<command-message>search files</command-message>\n<command-args>*.go -type f</command-args>"}]},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-cmd-search",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentList{
							Items: []internalevent.UserMessageContentItem{
								&internalevent.UserMessageContentCommand{
									CommandName:    "/search",
									CommandMessage: "search files",
									CommandArgs:    "*.go -type f",
								},
							},
						},
					},
				},
			},
			{
				name:            "array_content_multiple_items",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-array-multi","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"Example text 1"},{"type":"text","text":"<command-name>/example</command-name>\n<command-message>example message</command-message>"},{"type":"text","text":"<local-command-stdout>Example output</local-command-stdout>"}]},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-array-multi",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentList{
							Items: []internalevent.UserMessageContentItem{
								&internalevent.UserMessageContentMessage{
									Text: "Example text 1",
								},
								&internalevent.UserMessageContentCommand{
									CommandName:    "/example",
									CommandMessage: "example message",
									CommandArgs:    "",
								},
								&internalevent.UserMessageContentLocalCommand{
									Output: "Example output",
								},
							},
						},
					},
				},
			},
			{
				name:            "interrupted_for_tool_use",
				input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-interrupted-tool","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]},"isMeta":false}`,
				expectEventSent: true,
				wantEvent: &internalevent.UserMessage{
					SessionMessageBase: internalevent.SessionMessageBase{
						UUID:        "user-interrupted-tool",
						Type:        internalevent.MessageTypeUser,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
						IsMeta:      false,
					},
					Session: internalevent.Session{
						SessionID:      "test-session",
						TranscriptPath: "",
					},
					Message: internalevent.UserMessageData{
						Role: "user",
						Content: &internalevent.UserMessageContentList{
							Items: []internalevent.UserMessageContentItem{
								&internalevent.UserMessageContentInterrupted{
									Reason: "[Request interrupted by user for tool use]",
								},
							},
						},
					},
				},
			},
		},
		"HookEventAsSystemMessage": {
			{
				name:            "hook_event_stop",
				input:           `{"parentUuid":"c55f08ec-93cc-4e4e-9bfe-3be0035464f3","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"78f17a9d-d4da-4d94-ba71-18a48aac42a3","version":"1.0.64","gitBranch":"main","type":"system","content":"\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-07-31T15:42:02.113Z","uuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","toolUseID":"5a59f1ad-02af-4ddf-b129-3af63d9d0049","level":"info"}`,
				expectEventSent: true,
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
				name:            "hook_event_session_start_resume",
				input:           `{"parentUuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d99240fe-3539-438d-85c6-c51f5eb51902","version":"1.0.67","gitBranch":"feature/test","type":"system","content":"\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-08-03T13:09:46.461Z","uuid":"aa1fc221-60fc-4756-a892-93ffecbd47b9","toolUseID":"e51379a0-afd9-4434-bb3b-40cd178a0dc6","level":"info"}`,
				expectEventSent: true,
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

					// Check events sent to central handler based on expectEventSent flag
					if tt.expectEventSent {
						// This event should be sent to central handler
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
							cmpopts.IgnoreFields(internalevent.UserMessage{}, "Narration"),
						}

						if diff := cmp.Diff(tt.wantEvent, gotEvent, opts...); diff != "" {
							t.Errorf("%s mismatch (-want +got):\n%s", groupName, diff)
						}
					} else {
						// This event should not be sent to central handler
						if len(mockCentral.events) != 0 {
							t.Errorf("Expected no events sent to central handler, got %d", len(mockCentral.events))
						}
					}
				})
			}
		})
	}
}
