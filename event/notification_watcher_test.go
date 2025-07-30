package event

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestProcessNotificationLine(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantEvent   *NotificationEvent
		wantNoEvent bool
	}{
		{
			name: "SessionStart with source startup",
			line: `{"session_id":"8c70f7b7-5c83-4083-8930-f1fc33bf3dcd","transcript_path":"/tmp/test/projects/test-project/8c70f7b7-5c83-4083-8930-f1fc33bf3dcd.jsonl","cwd":"/tmp/test/project","hook_event_name":"SessionStart","source":"startup"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "8c70f7b7-5c83-4083-8930-f1fc33bf3dcd",
				TranscriptPath: "/tmp/test/projects/test-project/8c70f7b7-5c83-4083-8930-f1fc33bf3dcd.jsonl",
				CWD:            "/tmp/test/project",
				HookEventName:  "SessionStart",
				Source:         "startup",
			},
		},
		{
			name: "SessionStart with source clear",
			line: `{"session_id":"4e676915-7639-4dca-a41b-cf9684daaf50","transcript_path":"/tmp/test/projects/another-project/4e676915-7639-4dca-a41b-cf9684daaf50.jsonl","cwd":"/tmp/test/another","hook_event_name":"SessionStart","source":"clear"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "4e676915-7639-4dca-a41b-cf9684daaf50",
				TranscriptPath: "/tmp/test/projects/another-project/4e676915-7639-4dca-a41b-cf9684daaf50.jsonl",
				CWD:            "/tmp/test/another",
				HookEventName:  "SessionStart",
				Source:         "clear",
			},
		},
		{
			name: "SessionStart with source resume",
			line: `{"session_id":"07846160-cc2e-4dc1-9204-8c9817687f4b","transcript_path":"/tmp/test/projects/resume-project/07846160-cc2e-4dc1-9204-8c9817687f4b.jsonl","cwd":"/tmp/test/resume","hook_event_name":"SessionStart","source":"resume"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "07846160-cc2e-4dc1-9204-8c9817687f4b",
				TranscriptPath: "/tmp/test/projects/resume-project/07846160-cc2e-4dc1-9204-8c9817687f4b.jsonl",
				CWD:            "/tmp/test/resume",
				HookEventName:  "SessionStart",
				Source:         "resume",
			},
		},
		{
			name: "PreCompact notification with manual trigger",
			line: `{"session_id":"abc123","transcript_path":"/tmp/test/transcript.jsonl","cwd":"/tmp/test/project","hook_event_name":"PreCompact","trigger":"manual","custom_instructions":"Please summarize the conversation"}`,
			wantEvent: &NotificationEvent{
				SessionID:          "abc123",
				TranscriptPath:     "/tmp/test/transcript.jsonl",
				CWD:                "/tmp/test/project",
				HookEventName:      "PreCompact",
				Trigger:            "manual",
				CustomInstructions: "Please summarize the conversation",
			},
		},
		{
			name: "PreCompact notification with auto trigger",
			line: `{"session_id":"def456","transcript_path":"/tmp/test/transcript2.jsonl","cwd":"/tmp/test/project2","hook_event_name":"PreCompact","trigger":"auto"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "def456",
				TranscriptPath: "/tmp/test/transcript2.jsonl",
				CWD:            "/tmp/test/project2",
				HookEventName:  "PreCompact",
				Trigger:        "auto",
			},
		},
		{
			name: "Permission request for Bash",
			line: `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"Claude needs your permission to use Bash"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "test-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/tmp",
				HookEventName:  "Notification",
				Message:        "Claude needs your permission to use Bash",
			},
		},
		{
			name: "Permission request for Write",
			line: `{"session_id":"test-456","transcript_path":"/tmp/test2.jsonl","cwd":"/tmp/test","hook_event_name":"Notification","message":"Claude needs your permission to use Write"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "test-456",
				TranscriptPath: "/tmp/test2.jsonl",
				CWD:            "/tmp/test",
				HookEventName:  "Notification",
				Message:        "Claude needs your permission to use Write",
			},
		},
		{
			name: "MCP permission request",
			line: `{"session_id":"test-789","transcript_path":"/tmp/test3.jsonl","cwd":"/tmp/workspace","hook_event_name":"Notification","message":"Claude needs your permission to use filesystem - write (MCP)"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "test-789",
				TranscriptPath: "/tmp/test3.jsonl",
				CWD:            "/tmp/workspace",
				HookEventName:  "Notification",
				Message:        "Claude needs your permission to use filesystem - write (MCP)",
			},
		},
		{
			name: "Stop event",
			line: `{"session_id":"8c70f7b7-5c83-4083-8930-f1fc33bf3dcd","transcript_path":"/tmp/test/projects/stop-project/8c70f7b7-5c83-4083-8930-f1fc33bf3dcd.jsonl","cwd":"/tmp/test/stop","hook_event_name":"Stop","stop_hook_active":false}`,
			wantEvent: &NotificationEvent{
				SessionID:      "8c70f7b7-5c83-4083-8930-f1fc33bf3dcd",
				TranscriptPath: "/tmp/test/projects/stop-project/8c70f7b7-5c83-4083-8930-f1fc33bf3dcd.jsonl",
				CWD:            "/tmp/test/stop",
				HookEventName:  "Stop",
			},
		},
		{
			name: "General notification with error message",
			line: `{"session_id":"error-123","transcript_path":"/tmp/error.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"An error occurred while processing the request"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "error-123",
				TranscriptPath: "/tmp/error.jsonl",
				CWD:            "/tmp",
				HookEventName:  "Notification",
				Message:        "An error occurred while processing the request",
			},
		},
		{
			name: "Notification with waiting message",
			line: `{"session_id":"wait-123","transcript_path":"/tmp/wait.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"Claude is waiting for your input"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "wait-123",
				TranscriptPath: "/tmp/wait.jsonl",
				CWD:            "/tmp",
				HookEventName:  "Notification",
				Message:        "Claude is waiting for your input",
			},
		},
		{
			name: "Notification with success message",
			line: `{"session_id":"success-123","transcript_path":"/tmp/success.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"Operation completed successfully"}`,
			wantEvent: &NotificationEvent{
				SessionID:      "success-123",
				TranscriptPath: "/tmp/success.jsonl",
				CWD:            "/tmp",
				HookEventName:  "Notification",
				Message:        "Operation completed successfully",
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
			name:        "Plain text",
			line:        "This is not JSON",
			wantNoEvent: true,
		},
		{
			name:        "Empty JSON object",
			line:        `{}`,
			wantNoEvent: false, // Should create event with empty fields
			wantEvent:   &NotificationEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock event sender
			mockSender := NewMockEventSender()

			// Create watcher
			watcher := &NotificationWatcher{
				filePath:    "/test/path",
				eventSender: mockSender,
			}

			// Process line
			watcher.processNotificationLine(tt.line)

			// Get events immediately (no need to wait since it's synchronous)
			events := mockSender.GetEvents()

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
			notificationEvent, ok := events[0].(*NotificationEvent)
			if !ok {
				t.Fatalf("expected NotificationEvent, got %T", events[0])
			}

			// Compare events
			if diff := cmp.Diff(tt.wantEvent, notificationEvent); diff != "" {
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
			name: "Minimal notification",
			json: `{"session_id":"test-123","hook_event_name":"Notification"}`,
			want: &NotificationEvent{
				SessionID:     "test-123",
				HookEventName: "Notification",
			},
		},
		{
			name:    "Invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event NotificationEvent
			err := json.Unmarshal([]byte(tt.json), &event)

			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, &event); diff != "" {
					t.Errorf("NotificationEvent mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestNotificationWatcher tests the NotificationWatcher functionality
func TestNotificationWatcher(t *testing.T) {
	// Create a mock event sender
	mockSender := NewMockEventSender()

	// Create notification watcher
	watcher := NewNotificationWatcher("/tmp/test-notification.log", mockSender)

	// Test that watcher is created correctly
	if watcher.filePath != "/tmp/test-notification.log" {
		t.Errorf("expected filePath to be /tmp/test-notification.log, got %s", watcher.filePath)
	}

	// Test multiple events
	lines := []string{
		`{"session_id":"test-1","hook_event_name":"SessionStart","source":"startup"}`,
		`{"session_id":"test-2","hook_event_name":"PreCompact"}`,
		`{"session_id":"test-3","hook_event_name":"Notification","message":"Test"}`,
	}

	for _, line := range lines {
		watcher.processNotificationLine(line)
	}

	events := mockSender.GetEvents()
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}

	// Verify first event
	if event, ok := events[0].(*NotificationEvent); ok {
		if event.SessionID != "test-1" || event.HookEventName != "SessionStart" {
			t.Errorf("unexpected first event: %+v", event)
		}
	}

	// Verify second event
	if event, ok := events[1].(*NotificationEvent); ok {
		if event.SessionID != "test-2" || event.HookEventName != "PreCompact" {
			t.Errorf("unexpected second event: %+v", event)
		}
	}

	// Verify third event
	if event, ok := events[2].(*NotificationEvent); ok {
		if event.SessionID != "test-3" || event.HookEventName != "Notification" {
			t.Errorf("unexpected third event: %+v", event)
		}
	}
}

// TestMockEventSender tests the mock event sender functionality
func TestMockEventSender(t *testing.T) {
	sender := NewMockEventSender()

	// Test adding events
	event1 := &NotificationEvent{SessionID: "test1"}
	event2 := &NotificationEvent{SessionID: "test2"}

	sender.SendEvent(event1)
	sender.SendEvent(event2)

	events := sender.GetEvents()
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// Test clear
	sender.Clear()
	events = sender.GetEvents()
	if len(events) != 0 {
		t.Errorf("expected 0 events after clear, got %d", len(events))
	}
}
