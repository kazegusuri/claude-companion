package main

import (
	"testing"
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

func TestParsePermissionMessage(t *testing.T) {
	watcher := &NotificationWatcher{}

	tests := []struct {
		name          string
		message       string
		wantPermission bool
		wantTool      string
		wantMCP       string
		wantOperation string
	}{
		{
			name:          "Regular tool permission - Write",
			message:       "Claude needs your permission to use Write",
			wantPermission: true,
			wantTool:      "Write",
			wantMCP:       "",
			wantOperation: "",
		},
		{
			name:          "Regular tool permission - Bash",
			message:       "Claude needs your permission to use Bash",
			wantPermission: true,
			wantTool:      "Bash",
			wantMCP:       "",
			wantOperation: "",
		},
		{
			name:          "Regular tool permission - Read",
			message:       "Claude needs your permission to use Read",
			wantPermission: true,
			wantTool:      "Read",
			wantMCP:       "",
			wantOperation: "",
		},
		{
			name:          "MCP permission - gopls go_package_api",
			message:       "Claude needs your permission to use gopls - go_package_api (MCP)",
			wantPermission: true,
			wantTool:      "",
			wantMCP:       "gopls",
			wantOperation: "go_package_api",
		},
		{
			name:          "MCP permission - gopls go_symbol_references",
			message:       "Claude needs your permission to use gopls - go_symbol_references (MCP)",
			wantPermission: true,
			wantTool:      "",
			wantMCP:       "gopls",
			wantOperation: "go_symbol_references",
		},
		{
			name:          "MCP permission - linear-remote create_comment",
			message:       "Claude needs your permission to use linear-remote - create_comment (MCP)",
			wantPermission: true,
			wantTool:      "",
			wantMCP:       "linear-remote",
			wantOperation: "create_comment",
		},
		{
			name:          "Non-permission message - waiting",
			message:       "Claude is waiting for your input",
			wantPermission: false,
			wantTool:      "",
			wantMCP:       "",
			wantOperation: "",
		},
		{
			name:          "Non-permission message - completed",
			message:       "Task completed successfully",
			wantPermission: false,
			wantTool:      "",
			wantMCP:       "",
			wantOperation: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPermission, toolName, mcpName, operation := watcher.parsePermissionMessage(tt.message)

			if isPermission != tt.wantPermission {
				t.Errorf("parsePermissionMessage() isPermission = %v, want %v", isPermission, tt.wantPermission)
			}
			if toolName != tt.wantTool {
				t.Errorf("parsePermissionMessage() toolName = %v, want %v", toolName, tt.wantTool)
			}
			if mcpName != tt.wantMCP {
				t.Errorf("parsePermissionMessage() mcpName = %v, want %v", mcpName, tt.wantMCP)
			}
			if operation != tt.wantOperation {
				t.Errorf("parsePermissionMessage() operation = %v, want %v", operation, tt.wantOperation)
			}
		})
	}
}

func TestFormatNotificationEvent(t *testing.T) {
	mockNarrator := &MockNarrator{}
	watcher := &NotificationWatcher{
		narrator:  mockNarrator,
		debugMode: false,
	}

	tests := []struct {
		name               string
		event              *NotificationEvent
		wantToolPermission string
		wantTextNarration  string
	}{
		{
			name: "Regular tool permission - Write",
			event: &NotificationEvent{
				SessionID:      "test-session-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/home/test",
				HookEventName:  "Notification",
				Message:        "Claude needs your permission to use Write",
			},
			wantToolPermission: "Write",
			wantTextNarration:  "",
		},
		{
			name: "MCP permission - gopls",
			event: &NotificationEvent{
				SessionID:      "test-session-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/home/test",
				HookEventName:  "Notification",
				Message:        "Claude needs your permission to use gopls - go_package_api (MCP)",
			},
			wantToolPermission: "mcp__gopls__go_package_api",
			wantTextNarration:  "",
		},
		{
			name: "Waiting notification",
			event: &NotificationEvent{
				SessionID:      "test-session-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/home/test",
				HookEventName:  "Notification",
				Message:        "Claude is waiting for your input",
			},
			wantToolPermission: "",
			wantTextNarration:  "Claude is waiting for your input",
		},
		{
			name: "Success notification",
			event: &NotificationEvent{
				SessionID:      "test-session-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/home/test",
				HookEventName:  "Notification",
				Message:        "Task completed successfully",
			},
			wantToolPermission: "",
			wantTextNarration:  "Task completed successfully",
		},
		{
			name: "Error notification",
			event: &NotificationEvent{
				SessionID:      "test-session-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/home/test",
				HookEventName:  "Notification",
				Message:        "An error occurred",
			},
			wantToolPermission: "",
			wantTextNarration:  "An error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock narrator
			mockNarrator.LastToolPermission = ""
			mockNarrator.LastText = ""

			// Call formatNotificationEvent
			watcher.formatNotificationEvent(tt.event)

			// Check narrator calls
			if mockNarrator.LastToolPermission != tt.wantToolPermission {
				t.Errorf("formatNotificationEvent() called NarrateToolUsePermission with %q, want %q", 
					mockNarrator.LastToolPermission, tt.wantToolPermission)
			}
			if mockNarrator.LastText != tt.wantTextNarration {
				t.Errorf("formatNotificationEvent() called NarrateText with %q, want %q", 
					mockNarrator.LastText, tt.wantTextNarration)
			}
		})
	}
}

func TestFormatNotificationEventDebugMode(t *testing.T) {
	mockNarrator := &MockNarrator{}
	watcher := &NotificationWatcher{
		narrator:  mockNarrator,
		debugMode: true,
	}

	event := &NotificationEvent{
		SessionID:      "test-session-12345678",
		TranscriptPath: "/tmp/test.jsonl",
		CWD:            "/home/test",
		HookEventName:  "Notification",
		Message:        "Claude needs your permission to use Write",
	}

	// Just ensure it doesn't panic with debug mode enabled
	watcher.formatNotificationEvent(event)

	// Verify the correct tool was called
	if mockNarrator.LastToolPermission != "Write" {
		t.Errorf("Expected tool permission for 'Write', got %q", mockNarrator.LastToolPermission)
	}
}

func TestProcessNotificationLine(t *testing.T) {
	mockNarrator := &MockNarrator{}
	watcher := &NotificationWatcher{
		narrator:  mockNarrator,
		debugMode: false,
	}

	tests := []struct {
		name           string
		line           string
		wantNarration  bool
		wantTool       string
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

func TestParsePermissionMessageEdgeCases(t *testing.T) {
	watcher := &NotificationWatcher{}

	tests := []struct {
		name          string
		message       string
		wantPermission bool
	}{
		{
			name:          "Message with extra spaces",
			message:       "Claude needs your permission to use  Write  ",
			wantPermission: true, // The current implementation accepts this
		},
		{
			name:          "Partial message",
			message:       "Claude needs your permission",
			wantPermission: false,
		},
		{
			name:          "Case sensitivity",
			message:       "claude needs your permission to use Write",
			wantPermission: false,
		},
		{
			name:          "MCP without space before hyphen",
			message:       "Claude needs your permission to use gopls- go_package_api (MCP)",
			wantPermission: true, // The current implementation accepts this
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPermission, _, _, _ := watcher.parsePermissionMessage(tt.message)
			if isPermission != tt.wantPermission {
				t.Errorf("parsePermissionMessage() for edge case %q = %v, want %v", 
					tt.name, isPermission, tt.wantPermission)
			}
		})
	}
}


func TestNotificationEventTime(t *testing.T) {
	// Test that time formatting works correctly
	mockNarrator := &MockNarrator{}
	watcher := &NotificationWatcher{
		narrator:  mockNarrator,
		debugMode: false,
	}

	event := &NotificationEvent{
		SessionID:      "test-123",
		TranscriptPath: "/tmp/test.jsonl",
		CWD:            "/tmp",
		HookEventName:  "Notification",
		Message:        "Test message",
	}

	// Since we can't easily capture fmt.Print output in this test environment,
	// we'll just ensure the function doesn't panic
	watcher.formatNotificationEvent(event)
}

func TestMCPToolNameFormatting(t *testing.T) {

	tests := []struct {
		name         string
		message      string
		wantToolName string
	}{
		{
			name:         "gopls go_package_api",
			message:      "Claude needs your permission to use gopls - go_package_api (MCP)",
			wantToolName: "mcp__gopls__go_package_api",
		},
		{
			name:         "linear-remote create_comment",
			message:      "Claude needs your permission to use linear-remote - create_comment (MCP)",
			wantToolName: "mcp__linear-remote__create_comment",
		},
		{
			name:         "Regular tool",
			message:      "Claude needs your permission to use Write",
			wantToolName: "Write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNarrator := &MockNarrator{}
			watcher := &NotificationWatcher{
				narrator:  mockNarrator,
				debugMode: false,
			}

			event := &NotificationEvent{
				SessionID:      "test-123",
				TranscriptPath: "/tmp/test.jsonl",
				CWD:            "/tmp",
				HookEventName:  "Notification",
				Message:        tt.message,
			}

			watcher.formatNotificationEvent(event)

			if mockNarrator.LastToolPermission != tt.wantToolName {
				t.Errorf("MCP tool name formatting: got %q, want %q", 
					mockNarrator.LastToolPermission, tt.wantToolName)
			}
		})
	}
}