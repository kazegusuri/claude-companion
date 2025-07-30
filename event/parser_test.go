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
			name:        "system_message",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution completed","isMeta":false}`,
			wantType:    "SystemMessage",
			description: "Parse system message",
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
			wantOutput:  "\n[15:30:45] 👤 USER:\n  💬 Hello Claude",
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
			wantOutput:  "\n[15:30:45] 👤 USER:\n  💬 Hello world",
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
			wantOutput:  "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_123",
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
			wantOutput:  "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_456",
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
			wantOutput:  "\n[15:30:45] 👤 USER:\n  💬 Running tool...\n  ✅ Tool Result: toolu_456",
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
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hello! How can I help?\n  💰 Tokens: input=10, output=20, cache_read=100, cache_creation=50",
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
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: WebSearch (id: toolu_789)\n  💰 Tokens: input=5, output=15, cache_read=0, cache_creation=0",
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
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Let me search for that.\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=1, output=2, cache_read=3, cache_creation=4",
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
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hi",
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
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=5, output=10, cache_read=0, cache_creation=0",
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
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Finished.\n  💰 Tokens: input=1, output=1, cache_read=0, cache_creation=0",
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
			wantOutput:  "\n[15:30:45] ℹ️ SYSTEM: Tool execution completed",
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
			wantOutput:  "\n[15:30:45] ⚠️ SYSTEM [warning]: Rate limit warning",
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
			wantOutput:  "\n[15:30:45] ℹ️ SYSTEM: Tool execution started",
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
			wantOutput:  "\n📋 [SUMMARY] Summary text",
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
			wantOutput:  "\n[15:30:45] unknown event",
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
			expectedOutput: "\n[15:30:45] 👤 USER:\n  💬 Hello Claude",
			description:    "Parse and format simple user message",
		},
		{
			name:           "user_message_with_tool_result",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Success"}]}}`,
			expectedOutput: "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_123",
			description:    "Parse and format user message with tool result",
		},
		{
			name:           "user_message_with_tool_error",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":"Error occurred","is_error":true}]}}`,
			expectedOutput: "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_456 ❌ (error)",
			description:    "Parse and format user message with tool error",
		},
		{
			name:           "user_message_mixed_content",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"text","text":"Running tool..."},{"type":"tool_result","tool_use_id":"toolu_789","content":"Done"}]}}`,
			expectedOutput: "\n[15:30:45] 👤 USER:\n  💬 Running tool...\n  ✅ Tool Result: toolu_789",
			description:    "Parse and format user message with mixed content",
		},
		{
			name:           "user_message_with_tool_result_array_content",
			input:          `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":[{"type":"text","text":"File has diagnostics:\n- Error on line 10"}]}]}}`,
			expectedOutput: "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_456",
			description:    "Parse and format user message with tool result containing array content",
		},
		// Assistant Message Tests
		{
			name:           "assistant_message_simple",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hello! How can I help?"}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":50}}}`,
			expectedOutput: "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hello! How can I help?\n  💰 Tokens: input=10, output=20, cache_read=100, cache_creation=50",
			description:    "Parse and format assistant message with tokens",
		},
		{
			name:           "assistant_message_with_tool_use",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_789","name":"WebSearch","input":{"query":"weather today"}}],"usage":{"input_tokens":5,"output_tokens":15,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: WebSearch (id: toolu_789)\n  💰 Tokens: input=5, output=15, cache_read=0, cache_creation=0",
			description:    "Parse and format assistant message with tool use",
		},
		{
			name:           "assistant_message_mixed_content",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Let me search for that."},{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}}`,
			expectedOutput: "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Let me search for that.\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=1, output=2, cache_read=3, cache_creation=4",
			description:    "Parse and format assistant message with mixed content",
		},
		{
			name:           "assistant_message_no_tokens",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hi",
			description:    "Parse and format assistant message without token display (all zeros)",
		},
		{
			name:           "assistant_message_stop_reason_tool_use",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"stop_reason":"tool_use","usage":{"input_tokens":5,"output_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=5, output=10, cache_read=0, cache_creation=0",
			description:    "Parse and format assistant message with stop_reason tool_use",
		},
		{
			name:           "assistant_message_stop_reason_end_turn",
			input:          `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Finished."}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			expectedOutput: "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Finished.\n  💰 Tokens: input=1, output=1, cache_read=0, cache_creation=0",
			description:    "Parse and format assistant message with stop_reason end_turn",
		},
		// System Message Tests
		{
			name:           "system_message_simple",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution completed","isMeta":false}`,
			expectedOutput: "\n[15:30:45] ℹ️ SYSTEM: Tool execution completed",
			description:    "Parse and format simple system message",
		},
		{
			name:           "system_message_with_warning",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
			expectedOutput: "\n[15:30:45] ⚠️ SYSTEM [warning]: Rate limit warning",
			description:    "Parse and format system message with warning level",
		},
		{
			name:           "system_message_with_error",
			input:          `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"API error occurred","isMeta":false,"level":"error"}`,
			expectedOutput: "\n[15:30:45] ❌ SYSTEM [error]: API error occurred",
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
			expectedOutput: "\n[15:30:45] ℹ️ SYSTEM: Tool execution started",
			description:    "Parse and format system message with tool use ID",
		},
		// Summary Event Tests
		{
			name:           "summary_event",
			input:          `{"type":"summary","timestamp":"2025-01-26T15:30:45Z","uuid":"123","summary":"Summary text","leafUuid":"leaf_123"}`,
			expectedOutput: "\n📋 [SUMMARY] Summary text",
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
