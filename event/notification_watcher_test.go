package event

import (
	"testing"
	"time"

	"github.com/kazegusuri/claude-companion/narrator"
)

// MockNarrator implements narrator.Narrator for testing
type MockNarrator struct {
	LastToolPermission string
	LastText           string
}

func (m *MockNarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	return ""
}

func (m *MockNarrator) NarrateToolUsePermission(toolName string) string {
	m.LastToolPermission = toolName
	return toolName + "の使用許可を求めています"
}

func (m *MockNarrator) NarrateText(text string) string {
	m.LastText = text
	return text
}

func (m *MockNarrator) NarrateNotification(notificationType narrator.NotificationType) string {
	switch notificationType {
	case narrator.NotificationTypeCompact:
		return "コンテキストを圧縮しています"
	default:
		return ""
	}
}

// TODO: These tests need to be refactored to work with the new event package structure
// The parsePermissionMessage and formatNotificationEvent methods are now private in the event package

func TestProcessNotificationLine(t *testing.T) {
	mockNarrator := &MockNarrator{}
	handler := NewHandler(mockNarrator, false)
	// Start the event handler
	handler.Start()
	defer handler.Stop()

	watcher := &NotificationWatcher{
		filePath:     "/test/path",
		eventHandler: handler,
	}

	tests := []struct {
		name          string
		line          string
		wantNarration bool
		wantTool      string
	}{
		{
			name:          "Valid JSON notification",
			line:          `{"session_id":"test-123","transcript_path":"/tmp/test.jsonl","cwd":"/tmp","hook_event_name":"Notification","message":"Claude needs your permission to use Write"}`,
			wantNarration: true,
			wantTool:      "Write",
		},
		{
			name:          "Invalid JSON",
			line:          `{invalid json}`,
			wantNarration: false,
			wantTool:      "",
		},
		{
			name:          "Empty line",
			line:          "",
			wantNarration: false,
			wantTool:      "",
		},
		{
			name:          "Plain text",
			line:          "This is not JSON",
			wantNarration: false,
			wantTool:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockNarrator.LastToolPermission = ""
			mockNarrator.LastText = ""

			// Process line
			watcher.processNotificationLine(tt.line)

			// Give the event handler time to process
			time.Sleep(10 * time.Millisecond)

			if tt.wantNarration && mockNarrator.LastToolPermission != tt.wantTool && mockNarrator.LastText == "" {
				t.Errorf("processNotificationLine() expected narration but got none")
			}
			if !tt.wantNarration && (mockNarrator.LastToolPermission != "" || mockNarrator.LastText != "") {
				t.Errorf("processNotificationLine() expected no narration but got narration")
			}
			if tt.wantTool != "" && mockNarrator.LastToolPermission != tt.wantTool {
				t.Errorf("processNotificationLine() expected tool %q but got %q", tt.wantTool, mockNarrator.LastToolPermission)
			}
		})
	}
}
