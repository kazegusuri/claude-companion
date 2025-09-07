package print

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/logger"
)

func TestNotificationPrinter_Print(t *testing.T) {
	// Save and restore debug mode
	originalDebugMode := logger.IsDebugMode()
	defer logger.SetDebugMode(originalDebugMode)

	// Create buffer for testing
	var buf bytes.Buffer
	printer := NewNotificationPrinterWithWriter(&buf)
	// Mock time for consistent output
	fixedTime := time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC)
	printer.timeFunc = func() time.Time { return fixedTime }

	// Group test cases by event type
	testGroups := map[string][]struct {
		name        string
		debugMode   bool
		event       interface{}
		wantOutput  string
		wantErr     bool
		description string
	}{
		"NotificationEvent": {
			// PreCompact events
			{
				name:      "precompact_event",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-123",
					},
					HookEventName: "PreCompact",
					Message:       "Compacting conversation...",
					Narration: &event.NarrationMessage{
						Text: "会話を圧縮しています",
					},
				},
				wantOutput: "[15:30:45] 🗜️ PreCompact\n" +
					"  💬 会話を圧縮しています\n",
				description: "PreCompact event should show emoji and narration",
			},
			{
				name:      "precompact_event_with_debug",
				debugMode: true,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-123456789abc",
					},
					HookEventName: "PreCompact",
					Message:       "Compacting conversation...",
					Narration: &event.NarrationMessage{
						Text: "会話を圧縮しています",
					},
				},
				wantOutput: "[15:30:45] 🗜️ PreCompact [Session: session-]\n" +
					"  💬 会話を圧縮しています\n",
				description: "PreCompact event with debug mode should show session ID",
			},
			{
				name:      "precompact_event_no_narration",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-123",
					},
					HookEventName: "PreCompact",
					Message:       "Compacting conversation...",
				},
				wantOutput:  "[15:30:45] 🗜️ PreCompact\n",
				description: "PreCompact event without narration should only show header",
			},

			// SessionStart events
			{
				name:      "sessionstart_startup",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-456",
					},
					HookEventName: "SessionStart",
					Source:        "startup",
					Message:       "Session started",
					Narration: &event.NarrationMessage{
						Text: "新しいセッションを開始しました",
					},
				},
				wantOutput: "[15:30:45] 🚀 SessionStart:startup\n" +
					"  💬 新しいセッションを開始しました\n",
				description: "SessionStart:startup event should show source and narration",
			},
			{
				name:      "sessionstart_clear",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-456",
					},
					HookEventName: "SessionStart",
					Source:        "clear",
					Message:       "Session cleared",
					Narration: &event.NarrationMessage{
						Text: "セッションをクリアしました",
					},
				},
				wantOutput: "[15:30:45] 🚀 SessionStart:clear\n" +
					"  💬 セッションをクリアしました\n",
				description: "SessionStart:clear event should show source and narration",
			},
			{
				name:      "sessionstart_resume",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-456",
					},
					HookEventName: "SessionStart",
					Source:        "resume",
					Message:       "Session resumed",
					Narration: &event.NarrationMessage{
						Text: "セッションを再開しました",
					},
				},
				wantOutput: "[15:30:45] 🚀 SessionStart:resume\n" +
					"  💬 セッションを再開しました\n",
				description: "SessionStart:resume event should show source and narration",
			},
			{
				name:      "sessionstart_no_source",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-456",
					},
					HookEventName: "SessionStart",
					Message:       "Session started",
					Narration: &event.NarrationMessage{
						Text: "セッションを開始しました",
					},
				},
				wantOutput: "[15:30:45] 🚀 SessionStart\n" +
					"  💬 セッションを開始しました\n",
				description: "SessionStart without source should not show colon",
			},
			{
				name:      "sessionstart_with_debug",
				debugMode: true,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-456789012abc",
					},
					HookEventName: "SessionStart",
					Source:        "startup",
					Message:       "Session started",
					Narration: &event.NarrationMessage{
						Text: "新しいセッションを開始しました",
					},
				},
				wantOutput: "[15:30:45] 🚀 SessionStart:startup [Session: session-]\n" +
					"  💬 新しいセッションを開始しました\n",
				description: "SessionStart with debug mode should show session ID",
			},

			// General Notification events
			{
				name:      "notification_general",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					HookEventName: "Notification",
					Message:       "Something happened",
					Narration: &event.NarrationMessage{
						Text: "何か起きました",
					},
				},
				wantOutput: "[15:30:45] 🔔 Notification\n" +
					"  Something happened\n" +
					"  💬 何か起きました\n",
				description: "General notification should show message and narration",
			},
			{
				name:      "notification_permission_approve",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					HookEventName: "Notification",
					Message:       "Claude will use WebSearch: approve",
					Narration: &event.NarrationMessage{
						Text: "WebSearchツールを使用します",
					},
				},
				wantOutput: "[15:30:45] ✅ Notification\n" +
					"  Permission approve: WebSearch\n" +
					"  💬 WebSearchツールを使用します\n",
				description: "Permission approve should show ✅ emoji",
			},
			{
				name:      "notification_permission_deny",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					HookEventName: "Notification",
					Message:       "Claude will use Bash: deny",
					Narration: &event.NarrationMessage{
						Text: "Bashツールの使用を拒否しました",
					},
				},
				wantOutput: "[15:30:45] ❌ Notification\n" +
					"  Permission deny: Bash\n" +
					"  💬 Bashツールの使用を拒否しました\n",
				description: "Permission deny should show ❌ emoji",
			},
			{
				name:      "notification_permission_mcp",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					HookEventName: "Notification",
					Message:       "Claude will use mcp__playwright__browser_click: approve",
					Narration: &event.NarrationMessage{
						Text: "MCPツールを使用します",
					},
				},
				wantOutput: "[15:30:45] ✅ Notification\n" +
					"  Permission approve: browser_click (MCP: playwright)\n" +
					"  💬 MCPツールを使用します\n",
				description: "MCP permission should parse MCP tool name",
			},
			{
				name:      "notification_contains_permission",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					HookEventName: "Notification",
					Message:       "Permission required for this action",
				},
				wantOutput: "[15:30:45] 🔑 Notification\n" +
					"  Permission required for this action\n",
				description: "Message containing 'Permission' should show 🔑 emoji",
			},
			{
				name:      "notification_no_narration",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					HookEventName: "Notification",
					Message:       "Simple notification",
				},
				wantOutput: "[15:30:45] 🔔 Notification\n" +
					"  Simple notification\n",
				description: "Notification without narration should only show message",
			},
			{
				name:      "notification_with_debug",
				debugMode: true,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-789012345abc",
					},
					HookEventName: "Notification",
					Message:       "Debug notification",
				},
				wantOutput: "[15:30:45] 🔔 Notification [Session: session-]\n" +
					"  Debug notification\n",
				description: "Notification with debug mode should show session ID",
			},

			// Unknown event type
			{
				name:      "unknown_hook_event",
				debugMode: false,
				event: &event.NotificationEvent{
					Session: event.Session{
						SessionID: "session-unknown",
					},
					HookEventName: "UnknownEvent",
					Message:       "Unknown event type",
				},
				wantOutput:  "",
				description: "Unknown hook event type should return empty string",
			},
		},
		"SystemMessage": {
			{
				name:      "system_message_simple",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "System notification",
					Level:   "info",
				},
				wantOutput: "[15:30:45] 📣 SYSTEM [info]:\n" +
					"  ℹ️ System notification\n",
				description: "Simple system message with info level",
			},
			{
				name:      "system_message_error",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "Error occurred",
					Level:   "error",
				},
				wantOutput: "[15:30:45] 📣 SYSTEM [error]:\n" +
					"  ❌ Error occurred\n",
				description: "System message with error level",
			},
			{
				name:      "system_message_warning",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "Warning message",
					Level:   "warning",
				},
				wantOutput: "[15:30:45] 📣 SYSTEM [warning]:\n" +
					"  ⚠️ Warning message\n",
				description: "System message with warning level",
			},
			{
				name:      "system_message_debug",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "Debug info",
					Level:   "debug",
				},
				wantOutput: "[15:30:45] 📣 SYSTEM [debug]:\n" +
					"  🐛 Debug info\n",
				description: "System message with debug level",
			},
			{
				name:      "system_message_with_narration",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "System notification",
					Level:   "info",
					Narration: &event.NarrationMessage{
						Text: "システム通知",
					},
				},
				wantOutput: "[15:30:45] 📣 SYSTEM [info]:\n" +
					"  ℹ️ System notification\n" +
					"  💬 システム通知\n",
				description: "System message with narration",
			},
			{
				name:      "system_message_meta_skip",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      true,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "Meta message",
					Level:   "info",
				},
				wantOutput:  "",
				description: "Meta system message should be skipped without debug mode",
			},
			{
				name:      "system_message_meta_debug",
				debugMode: true,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "uuid-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/dir",
						Timestamp:   fixedTime,
						IsMeta:      true,
					},
					Session: event.Session{
						SessionID: "session-789",
					},
					RawContent: "Meta message",
					Level:   "info",
					ToolUseID: "tool-456",
				},
				wantOutput: "[15:30:45] 📣 SYSTEM [info] [UUID: uuid-123, META, Tool: tool-456]:\n" +
					"  ℹ️ Meta message\n",
				description: "Meta system message with debug mode shows details",
			},
			{
				name:      "system_message_hook_event",
				debugMode: true,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "hook-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/workspace",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "hook-session",
					},
					Content: &event.HookSystemMessageContent{
						HookName: "PreCompact",
						Command:  "/usr/local/bin/hook.sh",
						Status:   "completed",
						Type:     "PreCompact",
						Message:  "Hook executed successfully",
					},
					RawContent: "Hook executed successfully",
					Level:      "info",
					ToolUseID:  "tool-456",
				},
				wantOutput: "[15:30:45] 🪝 HOOK [PreCompact] [UUID: hook-123, Tool: tool-456]\n" +
					"  📟 Command: /usr/local/bin/hook.sh\n" +
					"  ✅ Status: completed\n" +
					"  💬 Message: Hook executed successfully\n" +
					"  🏷️  Level: info\n" +
					"  📂 CWD: /test/workspace\n",
				description: "HookEvent displayed as SystemMessage in debug mode",
			},
			{
				name:      "system_message_hook_event_no_debug",
				debugMode: false,
				event: &event.SystemMessage{
					SessionMessageBase: event.SessionMessageBase{
						UUID:        "hook-123",
						Type:        event.MessageTypeSystem,
						IsSidechain: false,
						CWD:         "/test/workspace",
						Timestamp:   fixedTime,
						IsMeta:      false,
					},
					Session: event.Session{
						SessionID: "hook-session",
					},
					Content: &event.HookSystemMessageContent{
						HookName: "PreCompact",
						Type:     "PreCompact",
					},
					RawContent: "Hook executed successfully",
					Level:      "info",
				},
				wantOutput: "",
				description: "HookEvent should not display when debug mode is off",
			},
		},
		"SummaryEvent": {
			{
				name:      "summary_event_simple",
				debugMode: false,
				event: &event.SummaryEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					LeafUUID:  "leaf-uuid-123",
					Summary:   "Session summary text",
				},
				wantOutput:  "📋 [SUMMARY] Session summary text\n",
				description: "Simple summary event",
			},
			{
				name:      "summary_event_with_debug",
				debugMode: true,
				event: &event.SummaryEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					LeafUUID:  "leaf-uuid-123",
					Summary:   "Session summary text",
				},
				wantOutput:  "📋 [SUMMARY] Session summary text [LeafUUID: leaf-uuid-123]\n",
				description: "Summary event with debug mode shows UUID",
			},
			{
				name:      "summary_event_with_narration",
				debugMode: false,
				event: &event.SummaryEvent{
					Session: event.Session{
						SessionID: "session-789",
					},
					LeafUUID:  "leaf-uuid-123",
					Summary:   "Session summary text",
					Narration: &event.NarrationMessage{
						Text: "セッションの要約",
					},
				},
				wantOutput: "📋 [SUMMARY] Session summary text\n" +
					"  💬 セッションの要約\n",
				description: "Summary event with narration",
			},
		},
		"UnsupportedEventType": {
			{
				name:        "unsupported_event_type",
				debugMode:   false,
				event:       struct{ Name string }{Name: "test"},
				wantErr:     true,
				description: "Unsupported event type should return error",
			},
		},
	}

	// Run tests for each group
	for groupName, testCases := range testGroups {
		t.Run(groupName, func(t *testing.T) {
			for _, tt := range testCases {
				t.Run(tt.name, func(t *testing.T) {
					// Reset buffer for each test
					buf.Reset()

					// Set debug mode for this test
					logger.SetDebugMode(tt.debugMode)

					// Call Print method
					printer.Print(tt.event)

					// Get output from buffer
					output := buf.String()

					// For unsupported event types, Print doesn't write anything
					if tt.wantErr {
						if output != "" {
							t.Errorf("Print() should not output anything for unsupported event types, got: %q", output)
						}
						return
					}

					// Check output
					if output != tt.wantOutput {
						t.Errorf("Print() output mismatch:\ngot:\n%q\nwant:\n%q", output, tt.wantOutput)
						// Show differences for debugging
						gotLines := strings.Split(output, "\n")
						wantLines := strings.Split(tt.wantOutput, "\n")
						for i := 0; i < len(gotLines) || i < len(wantLines); i++ {
							var got, want string
							if i < len(gotLines) {
								got = gotLines[i]
							}
							if i < len(wantLines) {
								want = wantLines[i]
							}
							if got != want {
								t.Logf("Line %d differs:\n  got:  %q\n  want: %q", i+1, got, want)
							}
						}
					}

					// Check that output ends with exactly one newline (except for empty output)
					if output != "" {
						if !strings.HasSuffix(output, "\n") {
							t.Errorf("Print() output should end with a newline, got: %q", output)
						}
						// Check that it doesn't end with multiple newlines
						if strings.HasSuffix(output, "\n\n") {
							t.Errorf("Print() output should end with exactly one newline, got multiple newlines: %q", output)
						}
					}
				})
			}
		})
	}

	// Reset debug mode
	logger.SetDebugMode(false)
}
