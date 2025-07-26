package main

import (
	"strings"
	"testing"
)

func TestEventParser_ParseAndFormat(t *testing.T) {
	parser := NewEventParser()

	tests := []struct {
		name        string
		input       string
		wantOutput  string
		wantErr     bool
		description string
	}{
		// User Message Tests
		{
			name:        "user_message_simple_string",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"Hello Claude"}}`,
			wantOutput:  "\n[15:30:45] USER: Hello Claude",
			description: "Simple user message with string content",
		},
		{
			name:        "user_message_with_text_array",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"text","text":"Hello world"}]}}`,
			wantOutput:  "\n[15:30:45] USER:\n  Text: Hello world",
			description: "User message with text in array format",
		},
		{
			name:        "user_message_with_tool_result",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Success"}]}}`,
			wantOutput:  "\n[15:30:45] USER:\n  Tool Result: toolu_123",
			description: "User message with tool result",
		},
		{
			name:        "user_message_with_tool_result_array_content",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":[{"type":"text","text":"File has diagnostics:\n- Error on line 10"}]}]}}`,
			wantOutput:  "\n[15:30:45] USER:\n  Tool Result: toolu_456",
			description: "User message with tool result containing array content",
		},
		{
			name:        "user_message_mixed_content",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"text","text":"Running tool..."},{"type":"tool_result","tool_use_id":"toolu_456","content":"Done"}]}}`,
			wantOutput:  "\n[15:30:45] USER:\n  Text: Running tool...\n  Tool Result: toolu_456",
			description: "User message with mixed content types",
		},

		// Assistant Message Tests
		{
			name:        "assistant_message_simple_text",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hello! How can I help?"}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":50}}}`,
			wantOutput:  "\n[15:30:45] ASSISTANT (claude-3-opus):\n  Text: Hello! How can I help?\n  Tokens: input=10, output=20, cache_read=100, cache_creation=50",
			description: "Assistant message with text and token usage",
		},
		{
			name:        "assistant_message_with_tool_use",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_789","name":"WebSearch","input":{"query":"weather today"}}],"usage":{"input_tokens":5,"output_tokens":15,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] ASSISTANT (claude-3-opus):\n  Tool Use: WebSearch (id: toolu_789)\n    Input: {\n      \"query\": \"weather today\"\n    }\n  Tokens: input=5, output=15, cache_read=0, cache_creation=0",
			description: "Assistant message with tool use",
		},
		{
			name:        "assistant_message_mixed_content",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Let me search for that."},{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}}`,
			wantOutput:  "\n[15:30:45] ASSISTANT (claude-3-opus):\n  Text: Let me search for that.\n  Tool Use: Search (id: toolu_999)\n    Input: {\n      \"q\": \"test\"\n    }\n  Tokens: input=1, output=2, cache_read=3, cache_creation=4",
			description: "Assistant message with mixed content",
		},
		{
			name:        "assistant_message_no_tokens",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] ASSISTANT (claude-3-opus):\n  Text: Hi",
			description: "Assistant message without token display (all zeros)",
		},
		{
			name:        "assistant_message_stop_reason_tool_use",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"stop_reason":"tool_use","usage":{"input_tokens":5,"output_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] ASSISTANT (claude-3-opus):\n  Tool Use: Search (id: toolu_999)\n    Input: {\n      \"q\": \"test\"\n    }\n  Tokens: input=5, output=10, cache_read=0, cache_creation=0",
			description: "Assistant message with stop_reason tool_use",
		},
		{
			name:        "assistant_message_stop_reason_end_turn",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Finished."}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] ASSISTANT (claude-3-opus):\n  Text: Finished.\n  Tokens: input=1, output=1, cache_read=0, cache_creation=0",
			description: "Assistant message with stop_reason end_turn",
		},

		// System Message Tests
		{
			name:        "system_message_simple",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution completed","isMeta":false}`,
			wantOutput:  "\n[15:30:45] SYSTEM: Tool execution completed",
			description: "Simple system message",
		},
		{
			name:        "system_message_with_level",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
			wantOutput:  "\n[15:30:45] SYSTEM [warning]: Rate limit warning",
			description: "System message with warning level",
		},
		{
			name:        "system_message_meta_hidden",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Internal meta info","isMeta":true}`,
			wantOutput:  "",
			description: "Meta system message should be hidden",
		},
		{
			name:        "system_message_with_tooluse",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"PreToolUse:Search completed","isMeta":false,"toolUseID":"toolu_123","level":"info"}`,
			wantOutput:  "\n[15:30:45] SYSTEM [info]: PreToolUse:Search completed",
			description: "System message with tool use ID",
		},

		// Summary Event Tests
		{
			name:        "summary_event",
			input:       `{"type":"summary","summary":"Code Review: Security Analysis Complete","leafUuid":"leaf_123"}`,
			wantOutput:  "\n[SUMMARY] Code Review: Security Analysis Complete",
			description: "Summary event",
		},

		// Unknown Event Tests
		{
			name:        "unknown_event_type",
			input:       `{"type":"custom","timestamp":"2025-01-26T15:30:45Z","uuid":"123","data":"some custom data"}`,
			wantOutput:  "\n[15:30:45] custom event\n  Raw: {\n    \"data\": \"some custom data\",\n    \"timestamp\": \"2025-01-26T15:30:45Z\",\n    \"type\": \"custom\",\n    \"uuid\": \"123\"\n  }",
			description: "Unknown event type with raw display",
		},

		// Error Cases
		{
			name:        "invalid_json",
			input:       `{invalid json}`,
			wantErr:     true,
			description: "Invalid JSON should return error",
		},
		{
			name:        "empty_json",
			input:       `{}`,
			wantOutput:  "\n[00:00:00]  event\n  Raw: {}",
			description: "Empty JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.ParseAndFormat(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAndFormat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got != tt.wantOutput {
				t.Errorf("ParseAndFormat() output mismatch\nDescription: %s\nGot:\n%q\nWant:\n%q",
					tt.description, got, tt.wantOutput)
			}
		})
	}
}

func TestEventParser_EdgeCases(t *testing.T) {
	parser := NewEventParser()

	tests := []struct {
		name       string
		input      string
		shouldPass bool
	}{
		{
			name:       "user_content_null",
			input:      `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":null}}`,
			shouldPass: true,
		},
		{
			name:       "assistant_empty_content_array",
			input:      `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3","content":[],"usage":{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			shouldPass: true,
		},
		{
			name:       "system_with_special_chars",
			input:      `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Error: \"Failed\" with \n newline","isMeta":false}`,
			shouldPass: true,
		},
		{
			name:       "very_long_content",
			input:      `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"` + strings.Repeat("a", 1000) + `"}}`,
			shouldPass: true,
		},
		{
			name:       "tool_result_with_is_error_true",
			input:      `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_error","content":"API Error: Rate limit exceeded","is_error":true}]}}`,
			shouldPass: true,
		},
		{
			name:       "unicode_in_content",
			input:      `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"Hello 世界 🌍"}}`,
			shouldPass: true,
		},
		{
			name:       "very_long_tool_use_id",
			input:      `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_` + strings.Repeat("x", 100) + `","content":"Done"}]}}`,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.ParseAndFormat(tt.input)
			if tt.shouldPass && err != nil {
				t.Errorf("ParseAndFormat() unexpected error: %v", err)
			}
		})
	}
}

func TestEventParser_ComplexToolInput(t *testing.T) {
	parser := NewEventParser()

	// Test complex nested tool input
	input := `{
		"type": "assistant",
		"timestamp": "2025-01-26T15:30:45Z",
		"uuid": "123",
		"requestId": "req_123",
		"message": {
			"id": "msg_123",
			"type": "message",
			"role": "assistant",
			"model": "claude-3",
			"content": [{
				"type": "tool_use",
				"id": "toolu_123",
				"name": "ComplexTool",
				"input": {
					"nested": {
						"array": [1, 2, 3],
						"object": {"key": "value"}
					},
					"string": "test",
					"number": 42,
					"boolean": true
				}
			}],
			"usage": {"input_tokens": 1, "output_tokens": 1, "cache_read_input_tokens": 0, "cache_creation_input_tokens": 0}
		}
	}`

	output, err := parser.ParseAndFormat(input)
	if err != nil {
		t.Fatalf("ParseAndFormat() error: %v", err)
	}

	// Check that complex input is properly formatted
	if !strings.Contains(output, "ComplexTool") {
		t.Error("Output should contain tool name")
	}
	if !strings.Contains(output, `"nested"`) {
		t.Error("Output should contain nested structure")
	}
}
