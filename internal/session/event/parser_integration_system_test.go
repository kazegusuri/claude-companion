package event

import (
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

var systemMessageTestCases = []parserIntegrationTestCase{
	{
		name:            "simple",
		input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test-session","cwd":"/test/dir","content":"Tool execution completed","isMeta":false}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "123",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "Tool execution completed",
			Level:      "",
		},
	},
	{
		name:            "with_level",
		input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test-session","cwd":"/test/dir","content":"Rate limit warning","isMeta":false,"level":"warning"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "123",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "Rate limit warning",
			Level:      "warning",
		},
	},
	{
		name:            "with_error",
		input:           `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"456","sessionID":"test-session","cwd":"/test/dir","content":"API error occurred","isMeta":false,"level":"error"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "456",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/test/dir",
				Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "API error occurred",
			Level:      "error",
		},
	},
	// HookEvent as SystemMessage test cases
	{
		name:            "hook_event_stop",
		input:           `{"parentUuid":"c55f08ec-93cc-4e4e-9bfe-3be0035464f3","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"78f17a9d-d4da-4d94-ba71-18a48aac42a3","version":"1.0.64","gitBranch":"main","type":"system","content":"\u001b[1mStop\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-07-31T15:42:02.113Z","uuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","toolUseID":"5a59f1ad-02af-4ddf-b129-3af63d9d0049","level":"info"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "ef16ec60-d3f6-4d59-bd99-d903bcddd8da",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/tmp/test/project",
				Timestamp:   mustParseTime("2025-07-31T15:42:02.113Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "\u001b[1mStop\u001b[22m [/test/bin/hook.sh] completed successfully",
			Content: &internalevent.HookSystemMessageContent{
				HookName: "Stop",
				Command:  "/test/bin/hook.sh",
				Status:   internalevent.HookStatusFinished,
				Type:     "",
				Message:  "\u001b[1mStop\u001b[22m [/test/bin/hook.sh] completed successfully",
			},
			Level:     "info",
			ToolUseID: "5a59f1ad-02af-4ddf-b129-3af63d9d0049",
		},
	},
	{
		name:            "hook_event_session_start_resume",
		input:           `{"parentUuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d99240fe-3539-438d-85c6-c51f5eb51902","version":"1.0.67","gitBranch":"feature/test","type":"system","content":"\u001b[1mSessionStart:resume\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-08-03T13:09:46.461Z","uuid":"aa1fc221-60fc-4756-a892-93ffecbd47b9","toolUseID":"e51379a0-afd9-4434-bb3b-40cd178a0dc6","level":"info"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "aa1fc221-60fc-4756-a892-93ffecbd47b9",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/tmp/test/project",
				Timestamp:   mustParseTime("2025-08-03T13:09:46.461Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "\u001b[1mSessionStart:resume\u001b[22m [/test/bin/hook.sh] completed successfully",
			Content: &internalevent.HookSystemMessageContent{
				HookName: "SessionStart",
				Command:  "/test/bin/hook.sh",
				Status:   internalevent.HookStatusFinished,
				Type:     "resume",
				Message:  "\u001b[1mSessionStart:resume\u001b[22m [/test/bin/hook.sh] completed successfully",
			},
			Level:     "info",
			ToolUseID: "e51379a0-afd9-4434-bb3b-40cd178a0dc6",
		},
	},
	{
		name:            "hook_event_posttooluse_running",
		input:           `{"parentUuid":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d1e2f3a4-b5c6-7890-abcd-ef1234567890","version":"1.0.99","gitBranch":"test-branch","type":"system","subtype":"informational","content":"Running \u001b[1mPostToolUse:Bash\u001b[22m...","isMeta":false,"timestamp":"2025-09-14T10:15:20.123Z","uuid":"f1e2d3c4-b5a6-9870-dcba-ef4321567890","toolUseID":"toolu_01XyzAbcDefGhiJklMnoPqr","level":"info"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "f1e2d3c4-b5a6-9870-dcba-ef4321567890",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/tmp/test/project",
				Timestamp:   mustParseTime("2025-09-14T10:15:20.123Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "Running \u001b[1mPostToolUse:Bash\u001b[22m...",
			Content: &internalevent.HookSystemMessageContent{
				HookName: "PostToolUse",
				Command:  "",
				Status:   internalevent.HookStatusStarted,
				Type:     "Bash",
				Message:  "Running \u001b[1mPostToolUse:Bash\u001b[22m...",
			},
			Level:     "info",
			ToolUseID: "toolu_01XyzAbcDefGhiJklMnoPqr",
		},
	},
	{
		name:            "hook_event_posttooluse_completed",
		input:           `{"parentUuid":"f1e2d3c4-b5a6-9870-dcba-ef4321567890","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d1e2f3a4-b5c6-7890-abcd-ef1234567890","version":"1.0.99","gitBranch":"test-branch","type":"system","subtype":"informational","content":"\u001b[1mPostToolUse:Bash\u001b[22m [/test/bin/posttool-hook.sh] completed successfully","isMeta":false,"timestamp":"2025-09-14T10:15:20.456Z","uuid":"a9b8c7d6-e5f4-3210-fedc-ba0987654321","toolUseID":"toolu_01XyzAbcDefGhiJklMnoPqr","level":"info"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SystemMessage{
			SessionMessageBase: internalevent.SessionMessageBase{
				UUID:        "a9b8c7d6-e5f4-3210-fedc-ba0987654321",
				Type:        internalevent.MessageTypeSystem,
				IsSidechain: false,
				CWD:         "/tmp/test/project",
				Timestamp:   mustParseTime("2025-09-14T10:15:20.456Z"),
				IsMeta:      false,
			},
			Session: internalevent.Session{
				SessionID:      "test-session",
				TranscriptPath: "/test/session.jsonl",
			},
			RawContent: "\u001b[1mPostToolUse:Bash\u001b[22m [/test/bin/posttool-hook.sh] completed successfully",
			Content: &internalevent.HookSystemMessageContent{
				HookName: "PostToolUse",
				Command:  "/test/bin/posttool-hook.sh",
				Status:   internalevent.HookStatusFinished,
				Type:     "Bash",
				Message:  "\u001b[1mPostToolUse:Bash\u001b[22m [/test/bin/posttool-hook.sh] completed successfully",
			},
			Level:     "info",
			ToolUseID: "toolu_01XyzAbcDefGhiJklMnoPqr",
		},
	},
}
