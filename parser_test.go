package main

import (
	"strings"
	"testing"

	"github.com/kazegusuri/claude-companion/narrator"
)

func TestEventParser_ParseAndFormat(t *testing.T) {
	parser := NewEventParser(narrator.NewNoOpNarrator())
	// Tests will check enhanced formatting by default

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
			wantOutput:  "\n[15:30:45] 👤 USER:\n  💬 Hello Claude",
			description: "Simple user message with string content",
		},
		{
			name:        "user_message_with_text_array",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"text","text":"Hello world"}]}}`,
			wantOutput:  "\n[15:30:45] 👤 USER:\n  💬 Hello world",
			description: "User message with text in array format",
		},
		{
			name:        "user_message_with_tool_result",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_123","content":"Success"}]}}`,
			wantOutput:  "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_123",
			description: "User message with tool result",
		},
		{
			name:        "user_message_with_tool_result_array_content",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_456","content":[{"type":"text","text":"File has diagnostics:\n- Error on line 10"}]}]}}`,
			wantOutput:  "\n[15:30:45] 👤 USER:\n  ✅ Tool Result: toolu_456",
			description: "User message with tool result containing array content",
		},
		{
			name:        "user_message_mixed_content",
			input:       `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":[{"type":"text","text":"Running tool..."},{"type":"tool_result","tool_use_id":"toolu_456","content":"Done"}]}}`,
			wantOutput:  "\n[15:30:45] 👤 USER:\n  💬 Running tool...\n  ✅ Tool Result: toolu_456",
			description: "User message with mixed content types",
		},

		// Assistant Message Tests
		{
			name:        "assistant_message_simple_text",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hello! How can I help?"}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":50}}}`,
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hello! How can I help?\n  💰 Tokens: input=10, output=20, cache_read=100, cache_creation=50",
			description: "Assistant message with text and token usage",
		},
		{
			name:        "assistant_message_with_tool_use",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_789","name":"WebSearch","input":{"query":"weather today"}}],"usage":{"input_tokens":5,"output_tokens":15,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: WebSearch (id: toolu_789)\n  💰 Tokens: input=5, output=15, cache_read=0, cache_creation=0",
			description: "Assistant message with tool use",
		},
		{
			name:        "assistant_message_mixed_content",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Let me search for that."},{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}}`,
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Let me search for that.\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=1, output=2, cache_read=3, cache_creation=4",
			description: "Assistant message with mixed content",
		},
		{
			name:        "assistant_message_no_tokens",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":0,"output_tokens":0,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Hi",
			description: "Assistant message without token display (all zeros)",
		},
		{
			name:        "assistant_message_stop_reason_tool_use",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_999","name":"Search","input":{"q":"test"}}],"stop_reason":"tool_use","usage":{"input_tokens":5,"output_tokens":10,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  🔧 Tool: Search (id: toolu_999)\n  💰 Tokens: input=5, output=10, cache_read=0, cache_creation=0",
			description: "Assistant message with stop_reason tool_use",
		},
		{
			name:        "assistant_message_stop_reason_end_turn",
			input:       `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Finished."}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantOutput:  "\n[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n  💬 Finished.\n  💰 Tokens: input=1, output=1, cache_read=0, cache_creation=0",
			description: "Assistant message with stop_reason end_turn",
		},

		// System Message Tests
		{
			name:        "system_message_simple",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Tool execution completed","isMeta":false}`,
			wantOutput:  "\n[15:30:45] ℹ️ SYSTEM: Tool execution completed",
			description: "Simple system message",
		},
		{
			name:        "system_message_with_level",
			input:       `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
			wantOutput:  "\n[15:30:45] ⚠️ SYSTEM [warning]: Rate limit warning",
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
			wantOutput:  "\n[15:30:45] ℹ️ SYSTEM [info]: PreToolUse:Search completed",
			description: "System message with tool use ID",
		},

		// Summary Event Tests
		{
			name:        "summary_event",
			input:       `{"type":"summary","summary":"Code Review: Security Analysis Complete","leafUuid":"leaf_123"}`,
			wantOutput:  "\n📋 [SUMMARY] Code Review: Security Analysis Complete",
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
	parser := NewEventParser(narrator.NewNoOpNarrator())

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

func TestEventParser_DebugMode(t *testing.T) {
	parser := NewEventParser(narrator.NewNoOpNarrator())
	parser.SetDebugMode(true)

	tests := []struct {
		name         string
		input        string
		wantContains []string
		description  string
	}{
		{
			name:         "user_message_with_debug_info",
			input:        `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-123","message":{"role":"user","content":"Hello Claude"}}`,
			wantContains: []string{"👤 USER:", "UUID: user-123", "💬 Hello Claude"},
			description:  "User message should show UUID in debug mode",
		},
		{
			name:         "assistant_message_with_debug_info",
			input:        `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"asst-123","requestId":"req-456","message":{"id":"msg-789","type":"message","role":"assistant","model":"claude-3","content":[{"type":"text","text":"Hi"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantContains: []string{"🤖 ASSISTANT", "ID: msg-789", "ReqID: req-456", "Stop: end_turn"},
			description:  "Assistant message should show IDs and stop reason in debug mode",
		},
		{
			name:         "system_message_meta_shown_in_debug",
			input:        `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"sys-123","content":"Meta info","isMeta":true}`,
			wantContains: []string{"ℹ️ SYSTEM:", "UUID: sys-123", "META"},
			description:  "Meta system messages should be shown in debug mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := parser.ParseAndFormat(tt.input)
			if err != nil {
				t.Fatalf("ParseAndFormat() error: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("ParseAndFormat() output missing expected content\nDescription: %s\nWant substring: %q\nGot: %q",
						tt.description, want, output)
				}
			}
		})
	}
}

func TestEventParser_ComplexToolInput(t *testing.T) {
	parser := NewEventParser(narrator.NewNoOpNarrator())
	// Tests will check enhanced formatting by default

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
	// In enhanced mode, we don't show the raw input JSON anymore
}

func TestEventParser_CompanionMode(t *testing.T) {
	parser := NewEventParser(narrator.NewNoOpNarrator())
	// parser.companionMode is true by default

	tests := []struct {
		name         string
		input        string
		wantContains []string
		description  string
	}{
		{
			name:         "user_message_with_emoji",
			input:        `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"123","message":{"role":"user","content":"Hello Claude"}}`,
			wantContains: []string{"👤 USER:", "Hello Claude"},
			description:  "User message should have emoji in companion mode",
		},
		{
			name:         "assistant_message_with_emoji_and_tokens",
			input:        `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Hello! How can I help?"}],"usage":{"input_tokens":10,"output_tokens":20,"cache_read_input_tokens":100,"cache_creation_input_tokens":50}}}`,
			wantContains: []string{"🤖 ASSISTANT", "💰 Tokens:", "Hello! How can I help?"},
			description:  "Assistant message should have emoji and formatted tokens",
		},
		{
			name:         "assistant_with_code_block",
			input:        `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Here's a function:\n\n` + "```" + `python\ndef hello():\n    print('Hello')\n` + "```" + `\n\nThis prints hello."}],"usage":{"input_tokens":1,"output_tokens":2,"cache_read_input_tokens":3,"cache_creation_input_tokens":4}}}`,
			wantContains: []string{"📝 Code Block 1 (python):", "def hello():", "```"},
			description:  "Should extract and format code blocks",
		},
		{
			name:         "assistant_with_tool_use",
			input:        `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"123","requestId":"req_123","message":{"id":"msg_123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"tool_use","id":"toolu_789","name":"Read","input":{"file_path":"/test/file.txt"}}],"usage":{"input_tokens":5,"output_tokens":15,"cache_read_input_tokens":0,"cache_creation_input_tokens":0}}}`,
			wantContains: []string{"📄 Reading file: /test/file.txt"},
			description:  "Should show file read with emoji",
		},
		{
			name:         "system_message_error",
			input:        `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","level":"error","content":"Connection failed"}`,
			wantContains: []string{"❌ SYSTEM [error]:", "Connection failed"},
			description:  "Error system message should have error emoji",
		},
		{
			name:         "summary_with_emoji",
			input:        `{"type":"summary","summary":"Test Summary"}`,
			wantContains: []string{"📋 [SUMMARY]", "Test Summary"},
			description:  "Summary should have clipboard emoji",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := parser.ParseAndFormat(tt.input)
			if err != nil {
				t.Fatalf("ParseAndFormat() error: %v", err)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("ParseAndFormat() output missing expected content\nDescription: %s\nWant substring: %q\nGot: %q",
						tt.description, want, output)
				}
			}
		})
	}
}
