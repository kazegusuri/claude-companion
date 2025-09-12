package event

import (
	"strings"
	"testing"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/narrator"
)

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

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
			input:       `{"parentUuid":"c55f08ec-93cc-4e4e-9bfe-3be0035464f3","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"78f17a9d-d4da-4d94-ba71-18a48aac42a3","version":"1.0.64","gitBranch":"main","type":"system","content":"\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-07-31T15:42:02.113Z","uuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","toolUseID":"5a59f1ad-02af-4ddf-b129-3af63d9d0049","level":"info"}`,
			wantType:    "HookEvent",
			description: "Parse hook event with Stop",
		},
		{
			name:        "hook_event_session_start",
			input:       `{"parentUuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d99240fe-3539-438d-85c6-c51f5eb51902","version":"1.0.67","gitBranch":"feature/test","type":"system","content":"\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-08-03T13:09:46.461Z","uuid":"aa1fc221-60fc-4756-a892-93ffecbd47b9","toolUseID":"e51379a0-afd9-4434-bb3b-40cd178a0dc6","level":"info"}`,
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

func TestFormatter_Format(t *testing.T) {
	formatter := NewFormatter(narrator.NewNoOpNarrator())

	tests := []struct {
		name        string
		event       Event
		wantOutput  string
		wantErr     bool
		description string
	}{
		// Assistant Message Tests
		{
			name: "assistant_message_simple_text",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID: "req_123",
				Message: AssistantMessageContent{
					ID:    "msg_123",
					Type:  "message",
					Role:  "assistant",
					Model: "claude-3-opus",
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "Hello! How can I help?",
						},
					},
					Usage: Usage{
						InputTokens:              10,
						OutputTokens:             20,
						CacheReadInputTokens:     100,
						CacheCreationInputTokens: 50,
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hello! How can I help?\n  💰 Tokens: input=10, output=20, cache_read=100, cache_creation=50\n",
			description: "Assistant message with text and token usage",
		},
		{
			name: "assistant_api_error_503",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID:         "req_123",
				IsApiErrorMessage: true,
				Message: AssistantMessageContent{
					ID:    "msg_123",
					Type:  "message",
					Role:  "assistant",
					Model: "<synthetic>",
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "API Error: 503 {\"type\":\"error\",\"error\":{\"type\":\"service_unavailable\",\"message\":\"Service temporarily unavailable\"}}",
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (<synthetic>):\n  ❌ API Error 503: service_unavailable - Service temporarily unavailable\n",
			description: "Assistant API error message with 503 status",
		},
		{
			name: "assistant_api_error_429",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "456",
				},
				RequestID:         "req_456",
				IsApiErrorMessage: true,
				Message: AssistantMessageContent{
					ID:    "msg_456",
					Type:  "message",
					Role:  "assistant",
					Model: "<synthetic>",
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "API Error: 429 {\"type\":\"error\",\"error\":{\"type\":\"rate_limit_error\",\"message\":\"Too many requests\"}}",
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (<synthetic>):\n  ❌ API Error 429: rate_limit_error - Too many requests\n",
			description: "Assistant API error message with 429 rate limit",
		},
		{
			name: "assistant_api_error_without_json",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "789",
				},
				RequestID:         "req_789",
				IsApiErrorMessage: true,
				Message: AssistantMessageContent{
					ID:    "msg_789",
					Type:  "message",
					Role:  "assistant",
					Model: "<synthetic>",
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "API Error: Connection timeout",
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (<synthetic>):\n  ❌ API Error: Connection timeout\n",
			description: "Assistant API error message without JSON",
		},
		{
			name: "assistant_message_with_tool_use",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID: "req_123",
				Message: AssistantMessageContent{
					ID:    "msg_123",
					Type:  "message",
					Role:  "assistant",
					Model: "claude-3-opus",
					Content: []AssistantContent{
						{
							Type:  "tool_use",
							ID:    "toolu_789",
							Name:  "WebSearch",
							Input: map[string]interface{}{"query": "weather today"},
						},
					},
					Usage: Usage{
						InputTokens:              5,
						OutputTokens:             15,
						CacheReadInputTokens:     0,
						CacheCreationInputTokens: 0,
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: WebSearch (id: toolu_789)\n  💰 Tokens: input=5, output=15, cache_read=0, cache_creation=0\n",
			description: "Assistant message with tool use",
		},
		{
			name: "assistant_message_mixed_content",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID: "req_123",
				Message: AssistantMessageContent{
					ID:    "msg_123",
					Type:  "message",
					Role:  "assistant",
					Model: "claude-3-opus",
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "Let me search for that.",
						},
						{
							Type:  "tool_use",
							ID:    "toolu_999",
							Name:  "Search",
							Input: map[string]interface{}{"q": "test"},
						},
					},
					Usage: Usage{
						InputTokens:              1,
						OutputTokens:             2,
						CacheReadInputTokens:     3,
						CacheCreationInputTokens: 4,
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Let me search for that.\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=1, output=2, cache_read=3, cache_creation=4\n",
			description: "Assistant message with mixed content",
		},
		{
			name: "assistant_message_no_tokens",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID: "req_123",
				Message: AssistantMessageContent{
					ID:    "msg_123",
					Type:  "message",
					Role:  "assistant",
					Model: "claude-3-opus",
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "Hi",
						},
					},
					Usage: Usage{
						InputTokens:              0,
						OutputTokens:             0,
						CacheReadInputTokens:     0,
						CacheCreationInputTokens: 0,
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hi\n",
			description: "Assistant message without token display (all zeros)",
		},
		{
			name: "assistant_message_stop_reason_tool_use",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID: "req_123",
				Message: AssistantMessageContent{
					ID:         "msg_123",
					Type:       "message",
					Role:       "assistant",
					Model:      "claude-3-opus",
					StopReason: stringPtr("tool_use"),
					Content: []AssistantContent{
						{
							Type:  "tool_use",
							ID:    "toolu_999",
							Name:  "Search",
							Input: map[string]interface{}{"q": "test"},
						},
					},
					Usage: Usage{
						InputTokens:              5,
						OutputTokens:             10,
						CacheReadInputTokens:     0,
						CacheCreationInputTokens: 0,
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=5, output=10, cache_read=0, cache_creation=0\n",
			description: "Assistant message with stop_reason tool_use",
		},
		{
			name: "assistant_message_stop_reason_end_turn",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				RequestID: "req_123",
				Message: AssistantMessageContent{
					ID:         "msg_123",
					Type:       "message",
					Role:       "assistant",
					Model:      "claude-3-opus",
					StopReason: stringPtr("end_turn"),
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "Finished.",
						},
					},
					Usage: Usage{
						InputTokens:              1,
						OutputTokens:             1,
						CacheReadInputTokens:     0,
						CacheCreationInputTokens: 0,
					},
				},
			},
			wantOutput:  "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Finished.\n  💰 Tokens: input=1, output=1, cache_read=0, cache_creation=0\n",
			description: "Assistant message with stop_reason end_turn",
		},
		// Unknown Event Tests
		{
			name: "unknown_event",
			event: &BaseEvent{
				TypeString: "unknown",
				Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
				UUID:       "123",
			},
			wantOutput:  "[15:30:45] unknown event\n",
			description: "Unknown event type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.Format(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Format() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if output != tt.wantOutput {
				t.Errorf("Format() output = %v, want %v", output, tt.wantOutput)
			}
		})
	}
}

func TestFormatter_DebugMode(t *testing.T) {
	formatter := NewFormatter(narrator.NewNoOpNarrator())
	formatter.SetDebugMode(true)

	// Verify debug mode is actually enabled
	if !logger.IsDebugMode() {
		t.Fatal("Debug mode was not enabled")
	}

	tests := []struct {
		name        string
		event       Event
		wantContain string
		description string
	}{
		{
			name: "assistant_message_debug",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeAssistant,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "ast-uuid-789",
				},
				RequestID: "req-debug-123",
				Message: AssistantMessageContent{
					ID:         "msg-debug-456",
					Type:       "message",
					Role:       "assistant",
					Model:      "claude-3-opus",
					StopReason: stringPtr("end_turn"),
					Content: []AssistantContent{
						{
							Type: "text",
							Text: "Debug test",
						},
					},
				},
			},
			wantContain: "ReqID: req-debug-123",
			description: "Assistant message should show request ID in debug mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.Format(tt.event)
			if err != nil {
				t.Errorf("Format() error = %v", err)
				return
			}

			if !strings.Contains(output, tt.wantContain) {
				t.Errorf("Format() output = %q, should contain %q", output, tt.wantContain)
			}
		})
	}

	// Reset debug mode after test
	logger.SetDebugMode(false)
}

func TestIntegration_ParserAndFormatter(t *testing.T) {
	// Reset debug mode to ensure test isolation
	logger.SetDebugMode(false)

	parser := NewParser()
	formatter := NewFormatter(narrator.NewNoOpNarrator())

	tests := []struct {
		name           string
		input          string
		expectedOutput string
		description    string
	}{
		// Assistant Message Tests
		{
			name:           "assistant_message_simple",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hello! How can I help?"}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":50}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hello! How can I help?\n  💰 Tokens: input=10, output=20, cache_read=100, cache_creation=50\n",
			description:    "Parse and format assistant message with tokens",
		},
		{
			name:           "assistant_message_with_tool_use",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_789","name":"WebSearch","input":{"query":"weather today"}}],"usage":{"input_tokens":5,"output_tokens":15,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: WebSearch (id: toolu_789)\n  💰 Tokens: input=5, output=15, cache_read=0, cache_creation=0\n",
			description:    "Parse and format assistant message with tool use",
		},
		{
			name:           "assistant_message_mixed_content",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Let me search for that."},{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Let me search for that.\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=1, output=2, cache_read=3, cache_creation=4\n",
			description:    "Parse and format assistant message with mixed content",
		},
		{
			name:           "assistant_message_no_tokens",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hi\n",
			description:    "Parse and format assistant message without token display (all zeros)",
		},
		{
			name:           "assistant_message_stop_reason_tool_use",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"stop_reason":"tool_use","usage":{"input_tokens":5,"output_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=5, output=10, cache_read=0, cache_creation=0\n",
			description:    "Parse and format assistant message with stop_reason tool_use",
		},
		{
			name:           "assistant_message_stop_reason_end_turn",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Finished."}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Finished.\n  💰 Tokens: input=1, output=1, cache_read=0, cache_creation=0\n",
			description:    "Parse and format assistant message with stop_reason end_turn",
		},
		{
			name:           "assistant_message_with_thinking",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-opus-4-20250514","content":[{"type":"thinking","thinking":"すべてのタスクが完了しました。結果をまとめてユーザーに報告します。","signature":"xxx"}],"usage":{"input_tokens":11,"output_tokens":14,"cache_read_input_tokens":45769,"cache_creation_input_tokens":772}}}`,
			expectedOutput: "[15:30:45] 🤖 ASSISTANT (claude-opus-4-20250514):\n  💬 すべてのタスクが完了しました。結果をまとめてユーザーに報告します。\n  💰 Tokens: input=11, output=14, cache_read=45769, cache_creation=772\n",
			description:    "Parse and format assistant message with thinking content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the event
			event, err := parser.Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Format the event
			output, err := formatter.Format(event)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			if output != tt.expectedOutput {
				t.Errorf("Integration test output = %v, want %v", output, tt.expectedOutput)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func TestExtractSessionFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantProj string
		wantSess string
	}{
		{
			name:     "standard_claude_path",
			path:     "/home/user/.claude/projects/myproject/session123.jsonl",
			wantProj: "myproject",
			wantSess: "session123",
		},
		{
			name:     "with_tilde",
			path:     "~/.claude/projects/test-project/my-session.jsonl",
			wantProj: "test-project",
			wantSess: "my-session",
		},
		{
			name:     "complex_session_name",
			path:     "/Users/john/.claude/projects/web-app/2025-01-26-feature-branch.jsonl",
			wantProj: "web-app",
			wantSess: "2025-01-26-feature-branch",
		},
		{
			name:     "non_claude_path",
			path:     "/var/log/something.jsonl",
			wantProj: "log",
			wantSess: "something",
		},
		{
			name:     "without_jsonl_extension",
			path:     "/home/user/.claude/projects/myproject/session123.txt",
			wantProj: "myproject",
			wantSess: "session123.txt",
		},
		{
			name:     "simple_path",
			path:     "project/session.jsonl",
			wantProj: "project",
			wantSess: "session",
		},
		{
			name:     "custom_directory",
			path:     "/custom/dir/my-project/my-session.jsonl",
			wantProj: "my-project",
			wantSess: "my-session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := extractSessionFromPath(tt.path)

			if session == nil {
				t.Fatal("extractSessionFromPath() returned nil, want non-nil")
			}
			if session.Project != tt.wantProj {
				t.Errorf("extractSessionFromPath() Project = %v, want %v", session.Project, tt.wantProj)
			}
			if session.Session != tt.wantSess {
				t.Errorf("extractSessionFromPath() Session = %v, want %v", session.Session, tt.wantSess)
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

	if userMsg.Session.Session != "test-session" {
		t.Errorf("Session.Session = %v, want test-session", userMsg.Session.Session)
	}
}
