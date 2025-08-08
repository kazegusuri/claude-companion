package event

import (
	"strings"
	"testing"
	"time"

	"github.com/kazegusuri/claude-companion/narrator"
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
			input:       `{"type":"summary","timestamp":"2025-01-26T15:30:45Z","uuid":"123","summary":"Summary text","leafUuid":"leaf_123"}`,
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
		// User Message Tests
		{
			name: "user_message_simple_string",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeUser,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Message: UserMessageContent{
					Role:    "user",
					Content: "Hello Claude",
				},
			},
			wantOutput:  "[15:30:45] 👤 USER:\n  💬 Hello Claude\n",
			description: "Simple user message with string content",
		},
		{
			name: "user_message_with_text_array",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeUser,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Hello world",
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 👤 USER:\n  💬 Hello world\n",
			description: "User message with text in array format",
		},
		{
			name: "user_message_with_tool_result",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeUser,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "toolu_123",
							"content":     "Success",
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_123\n",
			description: "User message with tool result",
		},
		{
			name: "user_message_with_tool_result_array_content",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeUser,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "toolu_456",
							"content": []interface{}{
								map[string]interface{}{
									"type": "text",
									"text": "File has diagnostics:\n- Error on line 10",
								},
							},
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_456\n",
			description: "User message with tool result containing array content",
		},
		{
			name: "user_message_mixed_content",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeUser,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": "Running tool...",
						},
						map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": "toolu_456",
							"content":     "Done",
						},
					},
				},
			},
			wantOutput:  "[15:30:45] 👤 USER:\n  💬 Running tool...\n  ✅ Tool Result: toolu_456\n",
			description: "User message with mixed content types",
		},
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
		// System Message Tests
		{
			name: "system_message_simple",
			event: &SystemMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeSystem,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Content: "Tool execution completed",
				IsMeta:  false,
			},
			wantOutput:  "[15:30:45] 📣 SYSTEM:\n  Tool execution completed\n",
			description: "Simple system message",
		},
		{
			name: "system_message_with_level",
			event: &SystemMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeSystem,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Content: "Rate limit warning",
				IsMeta:  false,
				Level:   "warning",
			},
			wantOutput:  "[15:30:45] 📣 SYSTEM [warning]:\n  ⚠️ Rate limit warning\n",
			description: "System message with warning level",
		},
		{
			name: "system_message_with_tooluse",
			event: &SystemMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeSystem,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "123",
				},
				Content:   "Tool execution started",
				IsMeta:    false,
				ToolUseID: "toolu_123",
			},
			wantOutput:  "[15:30:45] 📣 SYSTEM:\n  Tool execution started\n",
			description: "System message with tool use ID",
		},
		// Summary Event Tests
		{
			name: "summary_event",
			event: &SummaryEvent{
				EventType: EventTypeSummary,
				Summary:   "Summary text",
				LeafUUID:  "leaf_123",
			},
			wantOutput:  "📋 [SUMMARY] Summary text\n",
			description: "Summary event",
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

	tests := []struct {
		name        string
		event       Event
		wantContain string
		description string
	}{
		{
			name: "user_message_debug",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeUser,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "test-uuid-123",
				},
				Message: UserMessageContent{
					Role:    "user",
					Content: "Hello",
				},
			},
			wantContain: "[UUID: test-uuid-123]",
			description: "User message should show UUID in debug mode",
		},
		{
			name: "system_message_meta_debug",
			event: &SystemMessage{
				BaseEvent: BaseEvent{
					TypeString: EventTypeSystem,
					Timestamp:  mustParseTime("2025-01-26T15:30:45Z"),
					UUID:       "sys-uuid-456",
				},
				Content: "Meta message",
				IsMeta:  true,
			},
			wantContain: "META",
			description: "System message should show META flag in debug mode",
		},
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
			wantContain: "[ID: msg-debug-456, ReqID: req-debug-123]",
			description: "Assistant message should show message ID and request ID in debug mode",
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
				t.Errorf("Format() output = %v, should contain %v", output, tt.wantContain)
			}
		})
	}
}

func TestIntegration_ParserAndFormatter(t *testing.T) {
	parser := NewParser()
	formatter := NewFormatter(narrator.NewNoOpNarrator())

	tests := []struct {
		name           string
		input          string
		expectedOutput string
		description    string
	}{
		// User Message Tests
		{
			name:           "user_message_simple",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"Hello Claude"}}`,
			expectedOutput: "[15:30:45] 👤 USER:\n  💬 Hello Claude\n",
			description:    "Parse and format simple user message",
		},
		{
			name:           "user_message_with_tool_result",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Success"}]}}`,
			expectedOutput: "[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_123\n",
			description:    "Parse and format user message with tool result",
		},
		{
			name:           "user_message_with_tool_error",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":"Error occurred","is_error":true}]}}`,
			expectedOutput: "[15:30:45] 👤 USER:\n  ❌ Tool Result: toolu_456\n",
			description:    "Parse and format user message with tool error",
		},
		{
			name:           "user_message_mixed_content",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"text","text":"Running tool..."},{"type":"tool_result","tool_use_id":"toolu_789","content":"Done"}]}}`,
			expectedOutput: "[15:30:45] 👤 USER:\n  💬 Running tool...\n  ✅ Tool Result: toolu_789\n",
			description:    "Parse and format user message with mixed content",
		},
		{
			name:           "user_message_with_tool_result_array_content",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":[{"type":"text","text":"File has diagnostics:\n- Error on line 10"}]}]}}`,
			expectedOutput: "[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_456\n",
			description:    "Parse and format user message with tool result containing array content",
		},
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
		// System Message Tests
		{
			name:           "system_message_simple",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution completed","isMeta":false}`,
			expectedOutput: "[15:30:45] 📣 SYSTEM:\n  Tool execution completed\n",
			description:    "Parse and format simple system message",
		},
		{
			name:           "system_message_with_warning",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
			expectedOutput: "[15:30:45] 📣 SYSTEM [warning]:\n  ⚠️ Rate limit warning\n",
			description:    "Parse and format system message with warning level",
		},
		{
			name:           "system_message_with_error",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"API error occurred","isMeta":false,"level":"error"}`,
			expectedOutput: "[15:30:45] 📣 SYSTEM [error]:\n  ❌ API error occurred\n",
			description:    "Parse and format system message with error level",
		},
		{
			name:           "system_message_meta_hidden",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Internal metadata","isMeta":true}`,
			expectedOutput: "", // Meta messages are hidden in normal mode
			description:    "Parse and format meta system message (should be hidden)",
		},
		{
			name:           "system_message_with_tooluse",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution started","isMeta":false,"toolUseID":"toolu_123"}`,
			expectedOutput: "[15:30:45] 📣 SYSTEM:\n  Tool execution started\n",
			description:    "Parse and format system message with tool use ID",
		},
		// Summary Event Tests
		{
			name:           "summary_event",
			input:          `{"type":"summary","timestamp":"2025-01-26T15:30:45Z","uuid":"123","summary":"Summary text","leafUuid":"leaf_123"}`,
			expectedOutput: "📋 [SUMMARY] Summary text\n",
			description:    "Parse and format summary event",
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

// Helper function to parse time
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
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
