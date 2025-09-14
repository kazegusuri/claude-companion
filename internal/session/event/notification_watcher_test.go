package event

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

func TestProcessNotificationLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantEvent   *internalevent.NotificationEvent
		wantNoEvent bool
	}{
		{
			name: "SessionStart with source startup",
			line: `{"session_id":"8c70f7b7-5c83-4083-8930-f1fc33bf3dcd","transcript_path":"/tmp/test/projects/test-project/8c70f7b7-5c83-4083-8930-f1fc33bf3dcd.jsonl","cwd":"/tmp/test/project","hook_event_name":"SessionStart","source":"startup"}`,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{
					SessionID:      "8c70f7b7-5c83-4083-8930-f1fc33bf3dcd",
					TranscriptPath: "/tmp/test/projects/test-project/8c70f7b7-5c83-4083-8930-f1fc33bf3dcd.jsonl",
				},
				HookEventName: "SessionStart",
				Source:        "startup",
			},
		},
		{
			name: "SessionStart with source clear",
			line: `{"session_id":"4e676915-7639-4dca-a41b-cf9684daaf50","transcript_path":"/tmp/test/projects/another-project/4e676915-7639-4dca-a41b-cf9684daaf50.jsonl","cwd":"/tmp/test/another","hook_event_name":"SessionStart","source":"clear"}`,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{
					SessionID:      "4e676915-7639-4dca-a41b-cf9684daaf50",
					TranscriptPath: "/tmp/test/projects/another-project/4e676915-7639-4dca-a41b-cf9684daaf50.jsonl",
				},
				HookEventName: "SessionStart",
				Source:        "clear",
			},
		},
		{
			name: "PreCompact notification with manual trigger",
			line: `{"session_id":"abc123","transcript_path":"/tmp/test/transcript.jsonl","cwd":"/tmp/test/project","hook_event_name":"PreCompact","trigger":"manual","custom_instructions":"Please summarize the conversation"}`,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{
					SessionID:      "abc123",
					TranscriptPath: "/tmp/test/transcript.jsonl",
				},
				HookEventName:      "PreCompact",
				Trigger:            "manual",
				CustomInstructions: "Please summarize the conversation",
			},
		},
		{
			name: "Permission request for Bash",
			line: `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"Claude needs your permission to use Bash"}`,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{
					SessionID:      "test-123",
					TranscriptPath: "/tmp/test.jsonl",
				},
				HookEventName: "Notification",
				RawMessage:    "Claude needs your permission to use Bash",
				Message: &internalevent.NotificationPermissionMessage{
					ToolName: "Bash",
				},
			},
		},
		{
			name: "MCP permission request",
			line: `{"session_id":"test-789","transcript_path":"/tmp/test3.jsonl","cwd":"/tmp/workspace","hook_event_name":"Notification","message":"Claude needs your permission to use filesystem - write (MCP)"}`,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{
					SessionID:      "test-789",
					TranscriptPath: "/tmp/test3.jsonl",
				},
				HookEventName: "Notification",
				RawMessage:    "Claude needs your permission to use filesystem - write (MCP)",
				Message: &internalevent.NotificationPermissionMessage{
					ToolName:  "filesystem",
					MCPServer: "filesystem",
					Operation: "write",
				},
			},
		},
		{
			name: "General notification with error message",
			line: `{"session_id":"error-123","transcript_path":"/tmp/error.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"An error occurred while processing the request"}`,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{
					SessionID:      "error-123",
					TranscriptPath: "/tmp/error.jsonl",
				},
				HookEventName: "Notification",
				RawMessage:    "An error occurred while processing the request",
				Message: &internalevent.NotificationGeneralMessage{
					Text: "An error occurred while processing the request",
				},
			},
		},
		{
			name:        "Invalid JSON",
			line:        `{invalid json}`,
			wantNoEvent: true,
		},
		{
			name:        "Empty line",
			line:        "",
			wantNoEvent: true,
		},
		{
			name:        "Empty JSON object",
			line:        `{}`,
			wantNoEvent: false,
			wantEvent: &internalevent.NotificationEvent{
				Session: internalevent.Session{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock central handler
			mockHandler := NewMockCentralHandler()

			// Create watcher
			watcher := &NotificationWatcher{
				filePath:       "/test/path",
				centralHandler: mockHandler,
			}

			// Process line
			watcher.processNotificationLine(tt.line)

			// Get events from central handler
			events := mockHandler.GetEvents()

			if tt.wantNoEvent {
				if len(events) > 0 {
					t.Errorf("expected no events, got %d events", len(events))
				}
				return
			}

			// Should have exactly one event
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d events", len(events))
			}

			// Check event type
			notificationEvent, ok := events[0].(*internalevent.NotificationEvent)
			if !ok {
				t.Fatalf("expected internalevent.NotificationEvent, got %T", events[0])
			}

			// Compare events with cmp.Diff
			opts := []cmp.Option{
				cmpopts.IgnoreFields(internalevent.NotificationEvent{}, "Narration"),
			}
			if diff := cmp.Diff(tt.wantEvent, notificationEvent, opts...); diff != "" {
				t.Errorf("NotificationEvent mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestParseNotificationJSON tests parsing of notification JSON directly
func TestParseNotificationJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    *NotificationEvent
		wantErr bool
	}{
		{
			name: "PreCompact with custom instructions",
			json: `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"PreCompact","trigger":"manual","custom_instructions":"Please summarize"}`,
			want: &NotificationEvent{
				SessionID:          "test-123",
				TranscriptPath:     "/tmp/test.jsonl",
				CWD:                "/tmp",
				HookEventName:      "PreCompact",
				Trigger:            "manual",
				CustomInstructions: "Please summarize",
			},
		},
		{
			name: "Notification with message",
			json: `{"session_id":"notif-123","transcript_path":"/tmp/notif.jsonl","cwd":"/tmp/notif","hook_event_name":"Notification","message":"Test message"}`,
			want: &NotificationEvent{
				SessionID:      "notif-123",
				TranscriptPath: "/tmp/notif.jsonl",
				CWD:            "/tmp/notif",
				HookEventName:  "Notification",
				Message:        "Test message",
			},
		},
		{
			name:    "Invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name: "Empty object",
			json: `{}`,
			want: &NotificationEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got NotificationEvent
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, &got); diff != "" {
					t.Errorf("NotificationEvent mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}
