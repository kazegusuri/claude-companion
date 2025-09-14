package event

import (
	"testing"
)

func TestParser_Parse(t *testing.T) {
	parser := NewParserWithPath("/test/session.jsonl")

	tests := []struct {
		name        string
		input       string
		wantType    string
		wantErr     bool
		description string
	}{
		{
			name:        "user_message",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"Hello Claude"}}`,
			wantType:    "UserMessage",
			description: "Parse user message",
		},
		{
			name:        "assistant_message",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hello!"}]}}`,
			wantType:    "AssistantMessage",
			description: "Parse assistant message",
		},
		{
			name:        "assistant_message_with_thinking",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-opus-4-20250514","content":[{"type":"thinking","thinking":"すべてのタスクが完了しました。結果をまとめてユーザーに報告します。","signature":"xxx"}]}}`,
			wantType:    "AssistantMessage",
			description: "Parse assistant message with thinking content",
		},
		{
			name:        "assistant_api_error_message",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","isApiErrorMessage":true,"message":{"id":"msg_123","type":"message","role":"assistant","model":"<synthetic>","content":[{"type":"text","text":"API Error: 503 {\"type\":\"error\",\"error\":{\"type\":\"service_unavailable\",\"message\":\"Service temporarily unavailable\"}}"}]}}`,
			wantType:    "AssistantMessage",
			description: "Parse assistant API error message",
		},
		{
			name:        "system_message",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution completed","isMeta":false}`,
			wantType:    "SystemMessage",
			description: "Parse system message",
		},
		{
			name:        "hook_event_stop",
			input:       `{"parentUuid":"c55f08ec-93cc-4e4e-9bfe-3be0035464f3","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"78f17a9d-d4da-4d94-ba71-18a48aac42a3","version":"1.0.64","gitBranch":"main","type":"system","content":"\u001b[1mStop\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-07-31T15:42:02.113Z","uuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","toolUseID":"5a59f1ad-02af-4ddf-b129-3af63d9d0049","level":"info"}`,
			wantType:    "HookEvent",
			description: "Parse hook event with Stop",
		},
		{
			name:        "hook_event_session_start",
			input:       `{"parentUuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d99240fe-3539-438d-85c6-c51f5eb51902","version":"1.0.67","gitBranch":"feature/test","type":"system","content":"\u001b[1mSessionStart:resume\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-08-03T13:09:46.461Z","uuid":"aa1fc221-60fc-4756-a892-93ffecbd47b9","toolUseID":"e51379a0-afd9-4434-bb3b-40cd178a0dc6","level":"info"}`,
			wantType:    "HookEvent",
			description: "Parse hook event with SessionStart:resume",
		},
		{
			name:        "summary_event",
			input:       `{"type":"summary","summary":"Summary text","leafUuid":"leaf_123"}`,
			wantType:    "SummaryEvent",
			description: "Parse summary event",
		},
		{
			name:        "unknown_event",
			input:       `{"type":"unknown","timestamp":"2025-01-26T15:30:45Z","uuid":"123"}`,
			wantType:    "BaseEvent",
			description: "Parse unknown event type",
		},
		{
			name:        "invalid_json",
			input:       `{invalid json}`,
			wantErr:     true,
			description: "Invalid JSON should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Check the type of the returned event
			var gotType string
			switch event.(type) {
			case *UserMessage:
				gotType = "UserMessage"
			case *AssistantMessage:
				gotType = "AssistantMessage"
			case *SystemMessage:
				gotType = "SystemMessage"
			case *HookEvent:
				gotType = "HookEvent"
			case *SummaryEvent:
				gotType = "SummaryEvent"
			case *BaseEvent:
				gotType = "BaseEvent"
			default:
				gotType = "Unknown"
			}

			if gotType != tt.wantType {
				t.Errorf("Parse() returned type = %v, want %v", gotType, tt.wantType)
			}

			// Additional validation for HookEvent
			if hookEvent, ok := event.(*HookEvent); ok && tt.wantType == "HookEvent" {
				// Verify that hook content was parsed
				if hookEvent.HookEventType == "" {
					t.Error("HookEvent.HookEventType should not be empty")
				}
				if hookEvent.HookCommand == "" {
					t.Error("HookEvent.HookCommand should not be empty")
				}
				if hookEvent.HookStatus == "" {
					t.Error("HookEvent.HookStatus should not be empty")
				}
				if hookEvent.HookName == "" {
					t.Error("HookEvent.HookName should not be empty")
				}
			}
		})
	}
}

func TestParserWithPath(t *testing.T) {
	logPath := "/home/user/.claude/projects/test-project/test-session.jsonl"
	parser := NewParserWithPath(logPath)

	// Parse a simple user message
	input := `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"Hello"}}`

	event, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	userMsg, ok := event.(*UserMessage)
	if !ok {
		t.Fatalf("Parse() returned wrong type: %T", event)
	}

	// Check if Session was set correctly
	if userMsg.Session == nil {
		t.Fatal("Session should not be nil")
	}

	if userMsg.Session.Project != "test-project" {
		t.Errorf("Session.Project = %v, want test-project", userMsg.Session.Project)
	}

	if userMsg.Session.SessionID != "test-session" {
		t.Errorf("Session.SessionID = %v, want test-session", userMsg.Session.SessionID)
	}
}
