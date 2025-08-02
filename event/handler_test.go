package event

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kazegusuri/claude-companion/narrator"
)

// mockNarrator is a simple test narrator
type mockNarrator struct{}

func (m *mockNarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	return "mock-narrate-" + toolName, false
}

func (m *mockNarrator) NarrateToolUsePermission(toolName string) (string, bool) {
	return "mock-permission-" + toolName, false
}

func (m *mockNarrator) NarrateText(text string) (string, bool) {
	return text, false
}

func (m *mockNarrator) NarrateNotification(notificationType narrator.NotificationType) (string, bool) {
	return "mock-notification", false
}

func (m *mockNarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	if subagentType != "" && description != "" {
		return subagentType + " agentがタスク「" + description + "」を完了しました", false
	} else if description != "" {
		return "タスク「" + description + "」が完了しました", false
	}
	return "タスクが完了しました", false
}

// captureOutput captures printed output during test
func captureOutput(t *testing.T, f func()) string {
	// Create a pipe to capture output
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	// Save current stdout
	old := os.Stdout
	os.Stdout = w

	// Create channel to signal when done reading
	outputChan := make(chan string)

	// Start reading from pipe
	go func() {
		data, _ := io.ReadAll(r)
		outputChan <- string(data)
	}()

	// Execute function
	f()

	// Restore stdout and close writer
	os.Stdout = old
	w.Close()

	// Get output
	output := <-outputChan
	r.Close()

	return output
}

func TestHandler_IgnoreSidechainEvents(t *testing.T) {
	// Create handler with mock narrator
	handler := NewHandler(&mockNarrator{}, false)
	handler.Start()
	defer handler.Stop()

	// Test cases for different event types with isSidechain = true
	tests := []struct {
		name  string
		event Event
	}{
		{
			name: "UserMessage with sidechain",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					IsSidechain: true,
					TypeString:  "user",
					UUID:        "test-user-uuid",
					Timestamp:   time.Now(),
				},
				Message: UserMessageContent{
					Role:    "user",
					Content: "test message",
				},
			},
		},
		{
			name: "AssistantMessage with sidechain",
			event: &AssistantMessage{
				BaseEvent: BaseEvent{
					IsSidechain: true,
					TypeString:  "assistant",
					UUID:        "test-assistant-uuid",
					Timestamp:   time.Now(),
				},
				Message: AssistantMessageContent{
					Model: "test-model",
					Content: []AssistantContent{
						{Type: "text", Text: "test response"},
					},
				},
			},
		},
		{
			name: "SystemMessage with sidechain",
			event: &SystemMessage{
				BaseEvent: BaseEvent{
					IsSidechain: true,
					TypeString:  "system",
					UUID:        "test-system-uuid",
					Timestamp:   time.Now(),
				},
				Content: "system message",
			},
		},
		{
			name: "UserMessage with tool_result and sidechain",
			event: &UserMessage{
				BaseEvent: BaseEvent{
					IsSidechain: true,
					TypeString:  "user",
					UUID:        "test-toolresult-uuid",
					Timestamp:   time.Now(),
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"tool_use_id": "test-tool-id",
							"type":        "tool_result",
							"content":     []interface{}{map[string]interface{}{"type": "text", "text": "result"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(t, func() {
				handler.SendEvent(tt.event)
				// Give some time for event processing
				time.Sleep(50 * time.Millisecond)
			})

			// Should have no output for sidechain events
			if output != "" {
				t.Errorf("Expected no output for sidechain event, got: %s", output)
			}
		})
	}
}

func TestHandler_TaskToolResultNarration(t *testing.T) {
	// Create handler with mock narrator
	handler := NewHandler(&mockNarrator{}, false)
	handler.Start()
	defer handler.Stop()

	tests := []struct {
		name           string
		taskMessage    *AssistantMessage
		resultMessage  *UserMessage
		expectedOutput string
	}{
		{
			name: "Task with subagent_type",
			taskMessage: &AssistantMessage{
				BaseEvent: BaseEvent{
					IsSidechain: false,
					TypeString:  "assistant",
					UUID:        "assistant-uuid",
					Timestamp:   time.Now(),
				},
				Message: AssistantMessageContent{
					Model: "test-model",
					Content: []AssistantContent{
						{
							Type: "tool_use",
							ID:   "task-id-123",
							Name: "Task",
							Input: map[string]interface{}{
								"description":   "データベース最適化",
								"subagent_type": "database-engineer",
								"prompt":        "データベースのパフォーマンスを分析してください",
							},
						},
					},
				},
			},
			resultMessage: &UserMessage{
				BaseEvent: BaseEvent{
					IsSidechain: false,
					TypeString:  "user",
					UUID:        "result-uuid",
					Timestamp:   time.Now(),
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"tool_use_id": "task-id-123",
							"type":        "tool_result",
							"content":     []interface{}{map[string]interface{}{"type": "text", "text": "タスクが完了しました"}},
						},
					},
				},
			},
			expectedOutput: "database-engineer agentがタスク「データベース最適化」を完了しました",
		},
		{
			name: "Task without subagent_type",
			taskMessage: &AssistantMessage{
				BaseEvent: BaseEvent{
					IsSidechain: false,
					TypeString:  "assistant",
					UUID:        "assistant-uuid-2",
					Timestamp:   time.Now(),
				},
				Message: AssistantMessageContent{
					Model: "test-model",
					Content: []AssistantContent{
						{
							Type: "tool_use",
							ID:   "task-id-456",
							Name: "Task",
							Input: map[string]interface{}{
								"description": "コード解析",
								"prompt":      "プロジェクトのコード品質を確認してください",
							},
						},
					},
				},
			},
			resultMessage: &UserMessage{
				BaseEvent: BaseEvent{
					IsSidechain: false,
					TypeString:  "user",
					UUID:        "result-uuid-2",
					Timestamp:   time.Now(),
				},
				Message: UserMessageContent{
					Role: "user",
					Content: []interface{}{
						map[string]interface{}{
							"tool_use_id": "task-id-456",
							"type":        "tool_result",
							"content":     []interface{}{map[string]interface{}{"type": "text", "text": "解析が完了しました"}},
						},
					},
				},
			},
			expectedOutput: "タスク「コード解析」が完了しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send task message first
			handler.SendEvent(tt.taskMessage)
			time.Sleep(50 * time.Millisecond)

			// Capture output when sending result message
			output := captureOutput(t, func() {
				handler.SendEvent(tt.resultMessage)
				time.Sleep(50 * time.Millisecond)
			})

			// Debug: show the full output
			t.Logf("Full output: %q", output)

			// Check if expected narration is in output
			if !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("Expected output to contain '%s', got: %s", tt.expectedOutput, output)
			}
			// Also check that it's a completion message
			if !strings.Contains(output, "完了しました") {
				t.Errorf("Expected completion message, got: %s", output)
			}
		})
	}
}

func TestHandler_NonTaskToolResult(t *testing.T) {
	// Create handler with mock narrator
	handler := NewHandler(&mockNarrator{}, false)
	handler.Start()
	defer handler.Stop()

	// Send a tool result for a non-Task tool
	resultMessage := &UserMessage{
		BaseEvent: BaseEvent{
			IsSidechain: false,
			TypeString:  "user",
			UUID:        "result-uuid",
			Timestamp:   time.Now(),
		},
		Message: UserMessageContent{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{
					"tool_use_id": "other-tool-id",
					"type":        "tool_result",
					"content":     []interface{}{map[string]interface{}{"type": "text", "text": "other tool result"}},
				},
			},
		},
	}

	output := captureOutput(t, func() {
		handler.SendEvent(resultMessage)
		time.Sleep(50 * time.Millisecond)
	})

	// Should have no special narration for non-Task tools
	if strings.Contains(output, "タスク") || strings.Contains(output, "agent") {
		t.Errorf("Non-Task tool result should not have Task narration, got: %s", output)
	}
}
