package print

import (
	"time"

	"github.com/kazegusuri/claude-companion/internal/event"
)

// assistantMessageToolUseTestCases contains test cases for AssistantMessage with ToolUse content
var assistantMessageToolUseTestCases = []printerTestCase{
	// TodoWrite tool
	{
		name:      "tool_use_todowrite",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-todo",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-todo",
			},
			RequestID: "req-tool-todo",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-todo",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-todo-123",
							Name: "TodoWrite",
							Input: &event.ToolUseTodoWrite{
								Todos: []event.TodoItem{
									{Content: "Implement feature A", Status: "completed"},
									{Content: "Fix bug B", Status: "in_progress", ActiveForm: "Fixing bug B"},
									{Content: "Review PR C", Status: "pending"},
								},
							},
							Narration: &event.NarrationMessage{
								Text: "Updating todo list with 3 items",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Updating todo list with 3 items\n" +
			"    1. ✅ Implement feature A\n" +
			"    2. 🔄 Fix bug B\n" +
			"    3. ⏳ Review PR C\n",
		description: "Assistant message with TodoWrite tool use",
	},
	{
		name:      "tool_use_todowrite_no_narration",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-todo-no-narr",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-todo-no-narr",
			},
			RequestID: "req-tool-todo-no-narr",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-todo-no-narr",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-todo-456",
							Name: "TodoWrite",
							Input: &event.ToolUseTodoWrite{
								Todos: []event.TodoItem{
									{Content: "Task 1", Status: "pending"},
									{Content: "Task 2", Status: "completed"},
								},
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: TodoWrite\n" +
			"    1. ⏳ Task 1\n" +
			"    2. ✅ Task 2\n",
		description: "TodoWrite tool without narration",
	},

	// Bash tool
	{
		name:      "tool_use_bash",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-bash",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-bash",
			},
			RequestID: "req-tool-bash",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-bash",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-bash-123",
							Name: "Bash",
							Input: &event.ToolUseBash{
								Command:     "ls -la",
								Description: "List files in current directory",
							},
							Narration: &event.NarrationMessage{
								Text: "Listing files in the current directory",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Listing files in the current directory\n" +
			"  $ ls -la\n",
		description: "Assistant message with Bash tool use",
	},
	{
		name:      "tool_use_bash_with_background",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-bash-bg",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-bash-bg",
			},
			RequestID: "req-tool-bash-bg",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-bash-bg",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-bash-bg-123",
							Name: "Bash",
							Input: &event.ToolUseBash{
								Command:         "npm run dev",
								Description:     "Start development server",
								RunInBackground: true,
								Timeout:         60000,
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: Bash\n" +
			"    Command: npm run dev\n" +
			"    Description: Start development server\n",
		description: "Bash tool with background execution",
	},

	// Read tool
	{
		name:      "tool_use_read",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-read",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-read",
			},
			RequestID: "req-tool-read",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-read",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-read-123",
							Name: "Read",
							Input: &event.ToolUseRead{
								FilePath: "/test/file.txt",
								Limit:    100,
								Offset:   50,
							},
							Narration: &event.NarrationMessage{
								Text: "Reading file.txt from line 50",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Reading file.txt from line 50\n",
		description: "Assistant message with Read tool use",
	},

	// Write tool
	{
		name:      "tool_use_write",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-write",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-write",
			},
			RequestID: "req-tool-write",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-write",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-write-123",
							Name: "Write",
							Input: &event.ToolUseWrite{
								FilePath: "/test/output.txt",
								Content:  "Line 1\nLine 2\nLine 3\n",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: Write\n" +
			"    File: /test/output.txt\n" +
			"    Content: 4 lines\n",
		description: "Write tool without narration",
	},

	// Edit tool
	{
		name:      "tool_use_edit",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-edit",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-edit",
			},
			RequestID: "req-tool-edit",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-edit",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-edit-123",
							Name: "Edit",
							Input: &event.ToolUseEdit{
								FilePath:   "/test/file.go",
								OldString:  "fmt.Println(\"Hello\")",
								NewString:  "fmt.Println(\"Hello, World!\")",
								ReplaceAll: true,
							},
							Narration: &event.NarrationMessage{
								Text: "Updating greeting message in file.go",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Updating greeting message in file.go\n",
		description: "Assistant message with Edit tool use",
	},

	// MultiEdit tool
	{
		name:      "tool_use_multiedit",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-multiedit",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-multiedit",
			},
			RequestID: "req-tool-multiedit",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-multiedit",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-multiedit-123",
							Name: "MultiEdit",
							Input: &event.ToolUseMultiEdit{
								FilePath: "/test/file.go",
								Edits: []event.EditItem{
									{OldString: "old1", NewString: "new1"},
									{OldString: "old2", NewString: "new2", ReplaceAll: true},
									{OldString: "old3", NewString: "new3"},
								},
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: MultiEdit\n" +
			"    File: /test/file.go\n" +
			"    Edits: 3 changes\n",
		description: "MultiEdit tool without narration",
	},

	// Grep tool
	{
		name:      "tool_use_grep",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-grep",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-grep",
			},
			RequestID: "req-tool-grep",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-grep",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-grep-123",
							Name: "Grep",
							Input: &event.ToolUseGrep{
								Pattern:         "TODO",
								Path:            "/src",
								Glob:            "*.go",
								CaseInsensitive: true,
								ShowLineNumbers: true,
							},
							Narration: &event.NarrationMessage{
								Text: "Searching for TODO comments in Go files",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Searching for TODO comments in Go files\n",
		description: "Assistant message with Grep tool use",
	},

	// Glob tool
	{
		name:      "tool_use_glob",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-glob",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-glob",
			},
			RequestID: "req-tool-glob",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-glob",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-glob-123",
							Name: "Glob",
							Input: &event.ToolUseGlob{
								Pattern: "**/*.test.js",
								Path:    "/test",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: Glob\n" +
			"    Pattern: **/*.test.js\n" +
			"    Path: /test\n",
		description: "Glob tool without narration",
	},

	// Task tool
	{
		name:      "tool_use_task",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-task",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-task",
			},
			RequestID: "req-tool-task",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-task",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-task-123",
							Name: "Task",
							Input: &event.ToolUseTask{
								Description:  "Analyze codebase",
								Prompt:       "Find all TODO comments and create a summary",
								SubagentType: "general-purpose",
							},
							Narration: &event.NarrationMessage{
								Text: "Starting code analysis task",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Starting code analysis task\n",
		description: "Assistant message with Task tool use",
	},

	// WebFetch tool
	{
		name:      "tool_use_webfetch",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-webfetch",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-webfetch",
			},
			RequestID: "req-tool-webfetch",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-webfetch",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-webfetch-123",
							Name: "WebFetch",
							Input: &event.ToolUseWebFetch{
								URL:    "https://example.com/api/docs",
								Prompt: "Extract API endpoints",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: WebFetch\n" +
			"    URL: https://example.com/api/docs\n",
		description: "WebFetch tool without narration",
	},

	// WebSearch tool
	{
		name:      "tool_use_websearch",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-websearch",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-websearch",
			},
			RequestID: "req-tool-websearch",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-websearch",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-websearch-123",
							Name: "WebSearch",
							Input: &event.ToolUseWebSearch{
								Query:          "golang error handling best practices",
								AllowedDomains: []string{"go.dev", "golang.org"},
							},
							Narration: &event.NarrationMessage{
								Text: "Searching for Go error handling best practices",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Searching for Go error handling best practices\n",
		description: "Assistant message with WebSearch tool use",
	},

	// MCP tool
	{
		name:      "tool_use_mcp",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-mcp",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-mcp",
			},
			RequestID: "req-tool-mcp",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-mcp",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-mcp-123",
							Name: "mcp__github__create_issue",
							Input: &event.ToolUseMCP{
								Server: "github",
								Tool:   "create_issue",
								Data: map[string]interface{}{
									"repo":  "user/project",
									"title": "Bug report",
									"body":  "Description of the bug",
								},
							},
							Narration: &event.NarrationMessage{
								Text: "Creating GitHub issue",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  💬 Creating GitHub issue\n",
		description: "Assistant message with MCP tool use",
	},

	// Generic tool (unknown tool)
	{
		name:      "tool_use_generic",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-generic",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-generic",
			},
			RequestID: "req-tool-generic",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-generic",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-generic-123",
							Name: "UnknownTool",
							Input: &event.ToolUseGeneric{
								Data: map[string]interface{}{
									"param1": "value1",
									"param2": 42,
								},
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  🛠️ Tool Use: UnknownTool\n",
		description: "Generic tool without narration",
	},

	// Content list with multiple tool uses
	{
		name:      "tool_use_multiple_in_list",
		debugMode: false,
		event: &event.AssistantMessage{
			SessionMessageBase: event.SessionMessageBase{
				UUID:        "uuid-tool-multi",
				Type:        event.MessageTypeAssistant,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   time.Date(2024, 1, 1, 15, 30, 45, 0, time.UTC),
				IsMeta:      false,
			},
			Session: event.Session{
				SessionID: "session-tool-multi",
			},
			RequestID: "req-tool-multi",
			Message: event.AssistantMessageData{
				ID:    "msg-tool-multi",
				Type:  "message",
				Role:  "assistant",
				Model: "claude-3-opus",
				Content: &event.AssistantMessageContentList{
					Items: []event.AssistantMessageContentItem{
						&event.AssistantMessageContentText{
							Type: "text",
							Text: "Let me help you with that.",
						},
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-read-1",
							Name: "Read",
							Input: &event.ToolUseRead{
								FilePath: "/test/input.txt",
							},
							Narration: &event.NarrationMessage{
								Text: "Reading input file",
							},
						},
						&event.AssistantMessageContentText{
							Type:       "thinking",
							Text:       "Processing the file content...",
							IsThinking: true,
						},
						&event.AssistantMessageContentToolUse{
							Type: "tool_use",
							ID:   "tool-write-1",
							Name: "Write",
							Input: &event.ToolUseWrite{
								FilePath: "/test/output.txt",
								Content:  "Processed content",
							},
							Narration: &event.NarrationMessage{
								Text: "Writing processed output",
							},
						},
					},
				},
			},
		},
		wantOutput: "[15:30:45] 🤖 ASSISTANT (claude-3-opus):\n" +
			"  📝 Let me help you with that.\n" +
			"  💬 Reading input file\n" +
			"  🤔 Processing the file content...\n" +
			"  💬 Writing processed output\n",
		description: "Multiple tool uses in content list",
	},
}
