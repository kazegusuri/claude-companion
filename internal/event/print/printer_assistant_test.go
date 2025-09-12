package print

import (
	"time"

	"github.com/kazegusuri/claude-companion/internal/event"
)

// assistantMessageTestCases contains test cases for AssistantMessage printing
var assistantMessageTestCases = []printerTestCase{
	{
		name:      "assistant_message_api_error",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-api-error",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-api-error",
			},
			RequestID: "req-error-123",
			Message: event.AssistantMessageData{
				ID:    "msg-error-123",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.APIErrorMessageContent{
					StatusCode: 429,
					ErrorType:  "rate_limit_error",
					Error: event.APIErrorDetail{
						Type:    "rate_limit_error",
						Message: "Rate limit exceeded. Please wait before trying again.",
					},
				},
			},
			IsApiErrorMessage: true,
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  ❌ API Error 429: rate_limit_error - Rate limit exceeded. Please wait before trying again.\n",
		description: "Assistant message with API error",
	},
	{
		name:      "assistant_message_api_error_with_narration",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-api-error-narration",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-api-error-narration",
			},
			RequestID: "req-error-456",
			Message: event.AssistantMessageData{
				ID:    "msg-error-456",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.APIErrorMessageContent{
					StatusCode: 500,
					ErrorType:  "internal_server_error",
					Error: event.APIErrorDetail{
						Type:    "internal_server_error",
						Message: "An internal server error occurred.",
					},
				},
			},
			IsApiErrorMessage: true,
			Narration: &event.NarrationMessage{
				Text: "API Error occurred: Internal server error (500)",
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  ❌ API Error occurred: Internal server error (500)\n",
		description: "Assistant message with API error and narration",
	},
	{
		name:      "assistant_message_api_error_raw_text_fallback",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-api-error-raw",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-api-error-raw",
			},
			RequestID: "req-error-789",
			Message: event.AssistantMessageData{
				ID:    "msg-error-789",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.APIErrorMessageContent{
					RawText: "API Error: Something went wrong",
				},
			},
			IsApiErrorMessage: true,
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  ❌ API Error: Something went wrong\n",
		description: "Assistant message with API error raw text fallback",
	},
	{
		name:      "assistant_message_api_error_debug_mode",
		debugMode: true,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-api-error-debug",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-api-error-debug",
			},
			RequestID: "req-error-debug",
			Message: event.AssistantMessageData{
				ID:    "msg-error-debug",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.APIErrorMessageContent{
					StatusCode: 403,
					ErrorType:  "permission_error",
					Error: event.APIErrorDetail{
						Type:    "permission_error",
						Message: "You don't have permission to access this resource.",
					},
				},
			},
			IsApiErrorMessage: true,
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus): [ID: msg-error-debug, ReqID: req-error-debug]\n" +
			"  ❌ API Error 403: permission_error - You don't have permission to access this resource.\n",
		description: "Assistant message with API error in debug mode",
	},
	{
		name:      "assistant_message_text_content",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-text",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-text",
			},
			RequestID: "req-text-123",
			Message: event.AssistantMessageData{
				ID:    "msg-text-123",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentText{
					Type:          "text",
					Text:          "Here is a simple function:\n```python\ndef hello():\n    print('Hello')\n```\nThis prints Hello.",
					ProcessedText: "Here is a simple function:\n[CODE BLOCK 1: python]\nThis prints Hello.",
					CodeBlocks: []event.CodeBlock{
						{
							Language: "python",
							Content:  "def hello():\n    print('Hello')\n",
						},
					},
					Narration: &event.NarrationMessage{
						Text: "Generated a simple hello function",
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Generated a simple hello function\n" +
			"  📝 Code Block 1 (python):\n" +
			"    def hello():\n" +
			"        print('Hello')\n",
		description: "Assistant message with text content and code block",
	},
	{
		name:      "assistant_message_content_list",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-list",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-list",
			},
			RequestID: "req-list-123",
			Message: event.AssistantMessageData{
				ID:    "msg-list-123",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentText{
							Type: "text",
							Text: "First text block.",
						},
						&event.AssistantMessageContentText{
							Type:       "thinking",
							Text:       "Thinking about the problem...\nAnalyzing the requirements.\nConsidering solutions.",
							IsThinking: true,
						},
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-123",
							Name: "calculator",
							Input: &event.ToolUseGeneric{
								Data: map[string]interface{}{"expression": "2+2"},
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  📝 First text block.\n" +
			"  🤔 Thinking about the problem...\n" +
			"  Analyzing the requirements.\n" +
			"  Considering solutions.\n" +
			"  🛠️ Tool Use: calculator\n",
		description: "Assistant message with content list",
	},
}
