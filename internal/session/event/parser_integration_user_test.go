package event

import (
	"encoding/json"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

var userMessageTestCases = []parserIntegrationTestCase{
	{
		name:            "simple_string_content",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-123","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Hello, this is a test message"},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-123",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentMessage{
					Text: "Hello, this is a test message",
				},
			},
		},
	},
	{
		name:            "multiline_content",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-456","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Line 1\nLine 2\nLine 3"},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-456",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentMessage{
					Text: "Line 1\nLine 2\nLine 3",
				},
			},
		},
	},
	{
		name:            "meta_message",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-789","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Meta message"},"isMeta":true}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-789",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      true,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentMessage{
					Text: "Meta message",
				},
			},
		},
	},
	{
		name:            "with_sidechain",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-side","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"Sidechain message"},"isSidechain":true,"isMeta":false}`,
		expectEventSent: false, // Sidechain messages are ignored
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-side",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: true,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentMessage{
					Text: "Sidechain message",
				},
			},
		},
	},
	{
		name:            "empty_content",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-empty","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":""},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-empty",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentMessage{
					Text: "",
				},
			},
		},
	},
	{
		name:            "local_command_stdout_empty",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-empty","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<local-command-stdout></local-command-stdout>"},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-local-empty",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentLocalCommand{
					Output: "",
				},
			},
		},
	},
	{
		name:            "local_command_stdout_no_content",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-no-content","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<local-command-stdout>(no content)</local-command-stdout>"},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-local-no-content",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentLocalCommand{
					Output: "",
				},
			},
		},
	},
	{
		name:            "local_command_stdout_with_output",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-output","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<local-command-stdout>Command executed successfully\nOutput line 1\nOutput line 2</local-command-stdout>"},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-local-output",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentLocalCommand{
					Output: "Command executed successfully\nOutput line 1\nOutput line 2",
				},
			},
		},
	},
	{
		name:            "local_command_stdout_in_array",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-local-array","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"<local-command-stdout>Array output</local-command-stdout>"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-local-array",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentLocalCommand{
							Output: "Array output",
						},
					},
				},
			},
		},
	},
	{
		name:            "command_clear",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-cmd-clear","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":"<command-name>/clear</command-name>\n<command-message>clear</command-message>\n<command-args></command-args>"},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-cmd-clear",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentCommand{
					CommandName:    "/clear",
					CommandMessage: "clear",
					CommandArgs:    "",
				},
			},
		},
	},
	{
		name:            "command_with_args",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-cmd-search","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"<command-name>/search</command-name>\n<command-message>search files</command-message>\n<command-args>*.go -type f</command-args>"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-cmd-search",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentCommand{
							CommandName:    "/search",
							CommandMessage: "search files",
							CommandArgs:    "*.go -type f",
						},
					},
				},
			},
		},
	},
	{
		name:            "array_content_multiple_items",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-array-multi","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"Example text 1"},{"type":"text","text":"<command-name>/example</command-name>\n<command-message>example message</command-message>"},{"type":"text","text":"<local-command-stdout>Example output</local-command-stdout>"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-array-multi",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentMessage{
							Text: "Example text 1",
						},
						&internalevent.UserMessageContentCommand{
							CommandName:    "/example",
							CommandMessage: "example message",
							CommandArgs:    "",
						},
						&internalevent.UserMessageContentLocalCommand{
							Output: "Example output",
						},
					},
				},
			},
		},
	},
	{
		name:            "interrupted_for_tool_use",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-interrupted-tool","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-interrupted-tool",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentInterrupted{
							Reason: "[Request interrupted by user for tool use]",
						},
					},
				},
			},
		},
	},
	{
		name:            "tool_result_success",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-tool-success","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_01ABC","content":"Tool executed successfully"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-tool-success",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentToolResult{
							ToolUseID: "toolu_01ABC",
							Content:   "Tool executed successfully",
							IsError:   false,
						},
					},
				},
			},
		},
	},
	{
		name:            "tool_result_error",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-tool-error","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_01DEF","content":"<tool_use_error>File not found</tool_use_error>","is_error":true}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-tool-error",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentToolResult{
							ToolUseID: "toolu_01DEF",
							Content:   "<tool_use_error>File not found</tool_use_error>",
							IsError:   true,
						},
					},
				},
			},
		},
	},
	{
		name:            "tool_result_mixed_with_text",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-tool-mixed","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"text","text":"Running tool..."},{"type":"tool_result","tool_use_id":"toolu_01GHI","content":"Result: 42"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-tool-mixed",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentMessage{
							Text: "Running tool...",
						},
						&internalevent.UserMessageContentToolResult{
							ToolUseID: "toolu_01GHI",
							Content:   "Result: 42",
							IsError:   false,
						},
					},
				},
			},
		},
	},
	{
		name:            "unknown_content_type",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-unknown-type","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"type":"unknown_type","data":"some data"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-unknown-type",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentUnknown{
							Type: "unknown_type",
							Data: json.RawMessage(`{"data":"some data","type":"unknown_type"}`),
						},
					},
				},
			},
		},
	},
	{
		name:            "no_type_field",
		input:           `{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user-no-type","sessionId":"test-session","cwd":"/test/dir","message":{"role":"user","content":[{"custom_field":"custom_value"}]},"isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.UserMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "user-no-type",
				Type:        internalevent.MessageTypeUser,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			Message: internalevent.UserMessageData{
				Role: "user",
				Content: &internalevent.UserMessageContentList{
					Items: []internalevent.UserMessageContentItem{
						&internalevent.UserMessageContentUnknown{
							Type: "",
							Data: json.RawMessage(`{"custom_field":"custom_value"}`),
						},
					},
				},
			},
		},
	},
}
