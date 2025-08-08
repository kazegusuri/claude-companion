package event

import (
	"encoding/json"
	"testing"
)

func ptrString(s string) *string {
	return &s
}

func TestHookEvent_ParseJSON(t *testing.T) {
	tests := []struct {
		name      string
		jsonData  string
		wantEvent *HookEvent
		wantErr   bool
	}{
		{
			name:     "Stop hook event",
			jsonData: `{"parentUuid":"c55f08ec-93cc-4e4e-9bfe-3be0035464f3","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"78f17a9d-d4da-4d94-ba71-18a48aac42a3","version":"1.0.64","gitBranch":"main","type":"system","content":"\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-07-31T15:42:02.113Z","uuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","toolUseID":"5a59f1ad-02af-4ddf-b129-3af63d9d0049","level":"info"}`,
			wantEvent: &HookEvent{
				BaseEvent: BaseEvent{
					ParentUUID:  ptrString("c55f08ec-93cc-4e4e-9bfe-3be0035464f3"),
					IsSidechain: false,
					UserType:    "external",
					CWD:         "/tmp/test/project",
					SessionID:   "78f17a9d-d4da-4d94-ba71-18a48aac42a3",
					Version:     "1.0.64",
					GitBranch:   "main",
					TypeString:  "system",
					UUID:        "ef16ec60-d3f6-4d59-bd99-d903bcddd8da",
				},
				Content:   "\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
				IsMeta:    false,
				ToolUseID: "5a59f1ad-02af-4ddf-b129-3af63d9d0049",
				Level:     "info",
			},
			wantErr: false,
		},
		{
			name:     "SessionStart:resume hook event",
			jsonData: `{"parentUuid":"ef16ec60-d3f6-4d59-bd99-d903bcddd8da","isSidechain":false,"userType":"external","cwd":"/tmp/test/project","sessionId":"d99240fe-3539-438d-85c6-c51f5eb51902","version":"1.0.67","gitBranch":"feature/test","type":"system","content":"\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully","isMeta":false,"timestamp":"2025-08-03T13:09:46.461Z","uuid":"aa1fc221-60fc-4756-a892-93ffecbd47b9","toolUseID":"e51379a0-afd9-4434-bb3b-40cd178a0dc6","level":"info"}`,
			wantEvent: &HookEvent{
				BaseEvent: BaseEvent{
					ParentUUID:  ptrString("ef16ec60-d3f6-4d59-bd99-d903bcddd8da"),
					IsSidechain: false,
					UserType:    "external",
					CWD:         "/tmp/test/project",
					SessionID:   "d99240fe-3539-438d-85c6-c51f5eb51902",
					Version:     "1.0.67",
					GitBranch:   "feature/test",
					TypeString:  "system",
					UUID:        "aa1fc221-60fc-4756-a892-93ffecbd47b9",
				},
				Content:   "\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
				IsMeta:    false,
				ToolUseID: "e51379a0-afd9-4434-bb3b-40cd178a0dc6",
				Level:     "info",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event HookEvent
			err := json.Unmarshal([]byte(tt.jsonData), &event)

			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check all fields except timestamp
				if event.BaseEvent.ParentUUID == nil && tt.wantEvent.BaseEvent.ParentUUID != nil ||
					event.BaseEvent.ParentUUID != nil && tt.wantEvent.BaseEvent.ParentUUID == nil ||
					(event.BaseEvent.ParentUUID != nil && tt.wantEvent.BaseEvent.ParentUUID != nil && *event.BaseEvent.ParentUUID != *tt.wantEvent.BaseEvent.ParentUUID) {
					t.Errorf("ParentUUID = %v, want %v", event.BaseEvent.ParentUUID, tt.wantEvent.BaseEvent.ParentUUID)
				}
				if event.BaseEvent.IsSidechain != tt.wantEvent.BaseEvent.IsSidechain {
					t.Errorf("IsSidechain = %v, want %v", event.BaseEvent.IsSidechain, tt.wantEvent.BaseEvent.IsSidechain)
				}
				if event.BaseEvent.UserType != tt.wantEvent.BaseEvent.UserType {
					t.Errorf("UserType = %v, want %v", event.BaseEvent.UserType, tt.wantEvent.BaseEvent.UserType)
				}
				if event.BaseEvent.CWD != tt.wantEvent.BaseEvent.CWD {
					t.Errorf("CWD = %v, want %v", event.BaseEvent.CWD, tt.wantEvent.BaseEvent.CWD)
				}
				if event.BaseEvent.SessionID != tt.wantEvent.BaseEvent.SessionID {
					t.Errorf("SessionID = %v, want %v", event.BaseEvent.SessionID, tt.wantEvent.BaseEvent.SessionID)
				}
				if event.BaseEvent.Version != tt.wantEvent.BaseEvent.Version {
					t.Errorf("Version = %v, want %v", event.BaseEvent.Version, tt.wantEvent.BaseEvent.Version)
				}
				if event.BaseEvent.GitBranch != tt.wantEvent.BaseEvent.GitBranch {
					t.Errorf("GitBranch = %v, want %v", event.BaseEvent.GitBranch, tt.wantEvent.BaseEvent.GitBranch)
				}
				if event.BaseEvent.TypeString != tt.wantEvent.BaseEvent.TypeString {
					t.Errorf("TypeString = %v, want %v", event.BaseEvent.TypeString, tt.wantEvent.BaseEvent.TypeString)
				}
				if event.Content != tt.wantEvent.Content {
					t.Errorf("Content = %v, want %v", event.Content, tt.wantEvent.Content)
				}
				if event.IsMeta != tt.wantEvent.IsMeta {
					t.Errorf("IsMeta = %v, want %v", event.IsMeta, tt.wantEvent.IsMeta)
				}
				if event.BaseEvent.UUID != tt.wantEvent.BaseEvent.UUID {
					t.Errorf("UUID = %v, want %v", event.BaseEvent.UUID, tt.wantEvent.BaseEvent.UUID)
				}
				if event.ToolUseID != tt.wantEvent.ToolUseID {
					t.Errorf("ToolUseID = %v, want %v", event.ToolUseID, tt.wantEvent.ToolUseID)
				}
				if event.Level != tt.wantEvent.Level {
					t.Errorf("Level = %v, want %v", event.Level, tt.wantEvent.Level)
				}

				// Check timestamp is parsed correctly
				if event.BaseEvent.Timestamp.IsZero() {
					t.Error("Timestamp should not be zero")
				}
			}
		})
	}
}

func TestHookEvent_ParseHookContent(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		wantHookEventType string
		wantHookCommand   string
		wantHookStatus    string
		wantHookName      string
		wantErr           bool
	}{
		{
			name:              "Stop event with ANSI codes",
			content:           "\u001b[1mStop\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
			wantHookEventType: "Stop",
			wantHookCommand:   "/usr/local/bin/claude-notification.sh",
			wantHookStatus:    "completed successfully",
			wantHookName:      "claude-notification.sh",
			wantErr:           false,
		},
		{
			name:              "SessionStart:resume event with ANSI codes",
			content:           "\u001b[1mSessionStart:resume\u001b[22m [/usr/local/bin/claude-notification.sh] completed successfully",
			wantHookEventType: "SessionStart:resume",
			wantHookCommand:   "/usr/local/bin/claude-notification.sh",
			wantHookStatus:    "completed successfully",
			wantHookName:      "claude-notification.sh",
			wantErr:           false,
		},
		{
			name:              "Simple event without ANSI codes",
			content:           "TestEvent [/path/to/script.sh] failed with error",
			wantHookEventType: "TestEvent",
			wantHookCommand:   "/path/to/script.sh",
			wantHookStatus:    "failed with error",
			wantHookName:      "script.sh",
			wantErr:           false,
		},
		{
			name:              "Event with colon type",
			content:           "PreCompact:auto [/hooks/compact.sh] started",
			wantHookEventType: "PreCompact:auto",
			wantHookCommand:   "/hooks/compact.sh",
			wantHookStatus:    "started",
			wantHookName:      "compact.sh",
			wantErr:           false,
		},
		{
			name:    "Invalid format",
			content: "This is not a valid hook content",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &HookEvent{Content: tt.content}
			err := event.ParseHookContent()

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHookContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if event.HookEventType != tt.wantHookEventType {
					t.Errorf("HookEventType = %v, want %v", event.HookEventType, tt.wantHookEventType)
				}
				if event.HookCommand != tt.wantHookCommand {
					t.Errorf("HookCommand = %v, want %v", event.HookCommand, tt.wantHookCommand)
				}
				if event.HookStatus != tt.wantHookStatus {
					t.Errorf("HookStatus = %v, want %v", event.HookStatus, tt.wantHookStatus)
				}
				if event.HookName != tt.wantHookName {
					t.Errorf("HookName = %v, want %v", event.HookName, tt.wantHookName)
				}
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Simple bold text",
			input: "\u001b[1mBold Text\u001b[22m",
			want:  "Bold Text",
		},
		{
			name:  "Multiple ANSI codes",
			input: "\u001b[1m\u001b[31mRed Bold\u001b[0m Normal",
			want:  "Red Bold Normal",
		},
		{
			name:  "No ANSI codes",
			input: "Plain text without codes",
			want:  "Plain text without codes",
		},
		{
			name:  "Complex ANSI sequence",
			input: "\u001b[38;5;196mExtended color\u001b[0m",
			want:  "Extended color",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI() = %v, want %v", got, tt.want)
			}
		})
	}
}