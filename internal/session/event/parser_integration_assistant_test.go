package event

import (
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

// assistantMessageTestCases contains test cases for AssistantMessage parsing and central handler integration
var assistantMessageTestCases = []parserIntegrationTestCase{
	{
		name:            "assistant_message_with_text_content",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-123","sessionID":"test-session","cwd":"/test/dir","requestId":"req-123","message":{"id":"msg-123","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Here is a simple function:\n` + "```python\\ndef hello():\\n    print('Hello')\\n```" + `\nThis prints Hello."}],"stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":50}}}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-123",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-123",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-123",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.AssistantMessageContentText{
					Type:          "text",
					Text:          "Here is a simple function:\n```python\ndef hello():\n    print('Hello')\n```\nThis prints Hello.",
					ProcessedText: "Here is a simple function:\n[CODE BLOCK 1: python]\nThis prints Hello.",
					CodeBlocks: []internalevent.CodeBlock{
						{
							Language: "python",
							Content:  "def hello():\n    print('Hello')\n",
						},
					},
				},
				StopReason: func(s string) *string { return &s }("end_turn"),
				Usage: &internalevent.TokenUsage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
			IsApiErrorMessage: false,
		},
	},
	{
		name:            "assistant_message_with_multiple_contents",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-456","sessionID":"test-session","cwd":"/test/dir","requestId":"req-456","message":{"id":"msg-456","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Let me think about this..."},{"type":"thinking","thinking":"Analyzing the problem..."},{"type":"tool_use","id":"tool-123","name":"calculator","input":{"expression":"2+2"}}],"stop_reason":"tool_use","usage":{"input_tokens":200,"output_tokens":100}}}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-456",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-456",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-456",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.AssistantMessageContentList{
					Items: []internalevent.AssistantMessageContentItem{
						&internalevent.AssistantMessageContentText{
							Type:          "text",
							Text:          "Let me think about this...",
							ProcessedText: "Let me think about this...",
							CodeBlocks:    []internalevent.CodeBlock{},
						},
						&internalevent.AssistantMessageContentText{
							Type:          "thinking",
							Text:          "Analyzing the problem...",
							IsThinking:    true,
							ProcessedText: "Analyzing the problem...",
							CodeBlocks:    []internalevent.CodeBlock{},
						},
						&internalevent.AssistantMessageContentToolUse{
							Type:  "tool_use",
							ID:    "tool-123",
							Name:  "calculator",
							Input: map[string]interface{}{"expression": "2+2"},
						},
					},
				},
				StopReason: func(s string) *string { return &s }("tool_use"),
				Usage: &internalevent.TokenUsage{
					InputTokens:  200,
					OutputTokens: 100,
				},
			},
			IsApiErrorMessage: false,
		},
	},
	{
		name:            "assistant_message_api_error",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-err","sessionID":"test-session","cwd":"/test/dir","requestId":"req-err","message":{"id":"msg-err","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"API Error: 429 {\"type\":\"error\",\"error\":{\"type\":\"rate_limit_error\",\"message\":\"Rate limit exceeded\"}}"}]},"isApiErrorMessage":true}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-err",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-err",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-err",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.APIErrorMessageContent{
					StatusCode: 429,
					ErrorType:  "rate_limit_error",
					Error: internalevent.APIErrorDetail{
						Type:    "rate_limit_error",
						Message: "Rate limit exceeded",
					},
					RawText: `API Error: 429 {"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`,
				},
			},
			IsApiErrorMessage: true,
		},
	},
	{
		name:            "assistant_message_sidechain_ignored",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-side","sessionID":"test-session","cwd":"/test/dir","isSidechain":true,"message":{"id":"msg-side","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Sidechain message"}]}}`,
		expectEventSent: false, // Sidechain events are ignored
	},
	{
		name:            "assistant_message_multiple_code_blocks",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-multi-code","sessionID":"test-session","cwd":"/test/dir","requestId":"req-multi","message":{"id":"msg-multi","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Here are examples in different languages:\n\n` + "```javascript\\nconst x = 1;\\nconsole.log(x);\\n```\\n\\nAnd in Go:\\n\\n```go\\npackage main\\n\\nfunc main() {\\n    fmt.Println(\\\"Hello\\\")\\n}\\n```\\n\\nBoth examples show basic syntax." + `"}],"stop_reason":"end_turn","usage":{"input_tokens":100,"output_tokens":50}}}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-multi-code",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-multi",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-multi",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.AssistantMessageContentText{
					Type:          "text",
					Text:          "Here are examples in different languages:\n\n```javascript\nconst x = 1;\nconsole.log(x);\n```\n\nAnd in Go:\n\n```go\npackage main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\n\nBoth examples show basic syntax.",
					ProcessedText: "Here are examples in different languages:\n\n[CODE BLOCK 1: javascript]\n\nAnd in Go:\n\n[CODE BLOCK 2: go]\n\nBoth examples show basic syntax.",
					CodeBlocks: []internalevent.CodeBlock{
						{
							Language: "javascript",
							Content:  "const x = 1;\nconsole.log(x);\n",
						},
						{
							Language: "go",
							Content:  "package main\n\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n",
						},
					},
				},
				StopReason: func(s string) *string { return &s }("end_turn"),
				Usage: &internalevent.TokenUsage{
					InputTokens:  100,
					OutputTokens: 50,
				},
			},
			IsApiErrorMessage: false,
		},
	},
	{
		name:            "assistant_message_code_block_no_language",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-no-lang","sessionID":"test-session","cwd":"/test/dir","requestId":"req-no-lang","message":{"id":"msg-no-lang","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"Output:\n` + "```\\nplain text output\\nwithout language specifier\\n```" + `"}],"stop_reason":"end_turn","usage":{"input_tokens":50,"output_tokens":25}}}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-no-lang",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-no-lang",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-no-lang",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.AssistantMessageContentText{
					Type:          "text",
					Text:          "Output:\n```\nplain text output\nwithout language specifier\n```",
					ProcessedText: "Output:\n[CODE BLOCK 1: text]",
					CodeBlocks: []internalevent.CodeBlock{
						{
							Language: "text",
							Content:  "plain text output\nwithout language specifier\n",
						},
					},
				},
				StopReason: func(s string) *string { return &s }("end_turn"),
				Usage: &internalevent.TokenUsage{
					InputTokens:  50,
					OutputTokens: 25,
				},
			},
			IsApiErrorMessage: false,
		},
	},
	{
		name:            "assistant_message_no_code_blocks",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-no-code","sessionID":"test-session","cwd":"/test/dir","requestId":"req-no-code","message":{"id":"msg-no-code","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"This is just regular text without any code blocks. It contains normal explanation."}],"stop_reason":"end_turn","usage":{"input_tokens":30,"output_tokens":15}}}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-no-code",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-no-code",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-no-code",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.AssistantMessageContentText{
					Type:          "text",
					Text:          "This is just regular text without any code blocks. It contains normal explanation.",
					ProcessedText: "This is just regular text without any code blocks. It contains normal explanation.",
					CodeBlocks:    []internalevent.CodeBlock{},
				},
				StopReason: func(s string) *string { return &s }("end_turn"),
				Usage: &internalevent.TokenUsage{
					InputTokens:  30,
					OutputTokens: 15,
				},
			},
			IsApiErrorMessage: false,
		},
	},
	{
		name:            "assistant_message_with_text_and_code_in_array",
		input:           `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"assist-array-code","sessionID":"test-session","cwd":"/test/dir","requestId":"req-array","message":{"id":"msg-array","type":"message","role":"assistant","model":"claude-3-opus","content":[{"type":"text","text":"First, let me explain:"},{"type":"text","text":"Here's the code:\n` + "```python\\nprint('test')\\n```" + `"}],"stop_reason":"end_turn","usage":{"input_tokens":40,"output_tokens":20}}}`,
		expectEventSent: true,
		wantEvent: &internalevent.AssistantMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "assist-array-code",
				Type:        internalevent.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "",
			},
			RequestID: "req-array",
			Message: internalevent.AssistantMessageData{
				ID:    "msg-array",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &internalevent.AssistantMessageContentList{
					Items: []internalevent.AssistantMessageContentItem{
						&internalevent.AssistantMessageContentText{
							Type:          "text",
							Text:          "First, let me explain:",
							ProcessedText: "First, let me explain:",
							CodeBlocks:    []internalevent.CodeBlock{},
						},
						&internalevent.AssistantMessageContentText{
							Type:          "text",
							Text:          "Here's the code:\n```python\nprint('test')\n```",
							ProcessedText: "Here's the code:\n[CODE BLOCK 1: python]",
							CodeBlocks: []internalevent.CodeBlock{
								{
									Language: "python",
									Content:  "print('test')\n",
								},
							},
						},
					},
				},
				StopReason: func(s string) *string { return &s }("end_turn"),
				Usage: &internalevent.TokenUsage{
					InputTokens:  40,
					OutputTokens: 20,
				},
			},
			IsApiErrorMessage: false,
		},
	},
}
