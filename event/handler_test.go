package event

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kazegusuri/claude-companion/handler"

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

func (m *mockNarrator) NarrateText(text string, isThinking bool, meta *narrator.EventMeta) (string, bool) {
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
	// Create handler with mock narrator and session manager
	sessionManager := handler.NewSessionManager()
	handler := NewHandler(&mockNarrator{}, sessionManager, false)
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
	// Create handler with mock narrator and session manager
	sessionManager := handler.NewSessionManager()
	handler := NewHandler(&mockNarrator{}, sessionManager, false)
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
	// Create handler with mock narrator and session manager
	sessionManager := handler.NewSessionManager()
	handler := NewHandler(&mockNarrator{}, sessionManager, false)
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

// ===== Buffering Tests =====

// mockFormatterWithRecording records processed events for testing
type mockFormatterWithRecording struct {
	processedEvents []Event
	mu              sync.Mutex
}

func (m *mockFormatterWithRecording) Format(event Event) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedEvents = append(m.processedEvents, event)
	return fmt.Sprintf("Processed: %T\n", event), nil
}

func (m *mockFormatterWithRecording) SetDebugMode(debug bool) {}

func (m *mockFormatterWithRecording) getProcessedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.processedEvents)
}

func (m *mockFormatterWithRecording) clearProcessed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedEvents = nil
}

// Helper function to create test events
func createTestUserMessage(sessionName string, parentUUID *string) *UserMessage {
	return &UserMessage{
		BaseEvent: BaseEvent{
			IsSidechain: false,
			TypeString:  EventTypeUser,
			UUID:        fmt.Sprintf("user-%d", time.Now().UnixNano()),
			Timestamp:   time.Now(),
			ParentUUID:  parentUUID,
			Session: &SessionFile{
				Path:    "/test/path.jsonl",
				Project: "test-project",
				Session: sessionName,
			},
		},
		Message: UserMessageContent{
			Role:    "user",
			Content: "Test message",
		},
	}
}

func createTestHookEvent(sessionName string, hookEventType string) *HookEvent {
	event := &HookEvent{
		BaseEvent: BaseEvent{
			IsSidechain: false,
			TypeString:  EventTypeSystem,
			UUID:        fmt.Sprintf("hook-%d", time.Now().UnixNano()),
			Timestamp:   time.Now(),
			Session: &SessionFile{
				Path:    "/test/path.jsonl",
				Project: "test-project",
				Session: sessionName,
			},
		},
		Content:       fmt.Sprintf("%s [/test/script.sh] completed successfully", hookEventType),
		ToolUseID:     "tool-123",
		Level:         "info",
		HookEventType: hookEventType,
		HookCommand:   "/test/script.sh",
		HookStatus:    "completed successfully",
	}
	return event
}

// Test basic buffering behavior with ParentUUID==nil
func TestHandler_BufferingWithParentUUIDNil(t *testing.T) {
	// Create handler with mock formatter
	mockFormatter := &mockFormatterWithRecording{}
	handler := &Handler{
		narrator:    &mockNarrator{},
		formatter:   mockFormatter,
		debugMode:   true,
		eventChan:   make(chan Event, 100),
		done:        make(chan struct{}),
		taskTracker: NewTaskTracker(),
		buffers:     make(map[string]*BufferInfo),
	}
	handler.Start()
	defer handler.Stop()

	sessionName := "test-session"

	// Send event with ParentUUID==nil
	event1 := createTestUserMessage(sessionName, nil)
	handler.SendEvent(event1)

	// Wait a bit to ensure processing
	time.Sleep(100 * time.Millisecond)

	// Check that event was buffered (not processed)
	if mockFormatter.getProcessedCount() != 0 {
		t.Errorf("Event with ParentUUID==nil should be buffered, but %d events were processed",
			mockFormatter.getProcessedCount())
	}

	// Check buffer exists
	handler.bufferMutex.Lock()
	buffer, exists := handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if !exists {
		t.Error("Buffer should exist for session")
	}
	if buffer != nil && len(buffer.events) != 1 {
		t.Errorf("Buffer should contain 1 event, got %d", len(buffer.events))
	}

	// Send another event with ParentUUID set (should also be buffered)
	parentUUID := "test-parent"
	event2 := createTestUserMessage(sessionName, &parentUUID)
	handler.SendEvent(event2)

	time.Sleep(100 * time.Millisecond)

	// Still no events should be processed
	if mockFormatter.getProcessedCount() != 0 {
		t.Errorf("Subsequent events should also be buffered, but %d events were processed",
			mockFormatter.getProcessedCount())
	}

	handler.bufferMutex.Lock()
	buffer, exists = handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if !exists {
		t.Error("Buffer should still exist")
	}
	if buffer != nil && len(buffer.events) != 2 {
		t.Errorf("Buffer should contain 2 events, got %d", len(buffer.events))
	}
}

// Test buffer release on SessionStart:resume
func TestHandler_ReleaseBufferOnSessionStartResume(t *testing.T) {
	mockFormatter := &mockFormatterWithRecording{}
	handler := &Handler{
		narrator:    &mockNarrator{},
		formatter:   mockFormatter,
		debugMode:   true,
		eventChan:   make(chan Event, 100),
		done:        make(chan struct{}),
		taskTracker: NewTaskTracker(),
		buffers:     make(map[string]*BufferInfo),
	}
	handler.Start()
	defer handler.Stop()

	sessionName := "resume-test"

	// Send event with ParentUUID==nil to trigger buffering
	event1 := createTestUserMessage(sessionName, nil)
	handler.SendEvent(event1)

	time.Sleep(100 * time.Millisecond)

	// Verify buffering started
	handler.bufferMutex.Lock()
	_, exists := handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if !exists {
		t.Error("Buffer should exist before SessionStart:resume")
	}

	// Send SessionStart:resume event
	hookEvent := createTestHookEvent(sessionName, "SessionStart:resume")
	handler.SendEvent(hookEvent)

	time.Sleep(100 * time.Millisecond)

	// Buffer should be released
	handler.bufferMutex.Lock()
	_, exists = handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if exists {
		t.Error("Buffer should be released after SessionStart:resume")
	}

	// Hook event itself should be processed
	if mockFormatter.getProcessedCount() != 1 {
		t.Errorf("Expected 1 event (hook) to be processed, got %d", mockFormatter.getProcessedCount())
	}
}

// Test buffer release on timeout
func TestHandler_ReleaseBufferOnTimeout(t *testing.T) {
	mockFormatter := &mockFormatterWithRecording{}
	handler := &Handler{
		narrator:    &mockNarrator{},
		formatter:   mockFormatter,
		debugMode:   true,
		eventChan:   make(chan Event, 100),
		done:        make(chan struct{}),
		taskTracker: NewTaskTracker(),
		buffers:     make(map[string]*BufferInfo),
	}
	handler.Start()
	defer handler.Stop()

	sessionName := "timeout-test"

	// Send event with ParentUUID==nil
	event1 := createTestUserMessage(sessionName, nil)
	handler.SendEvent(event1)

	time.Sleep(100 * time.Millisecond)

	// Verify buffer exists
	handler.bufferMutex.Lock()
	_, exists := handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if !exists {
		t.Error("Buffer should exist initially")
	}

	// Wait for timeout (1 second + buffer)
	time.Sleep(1100 * time.Millisecond)

	// Buffer should be released
	handler.bufferMutex.Lock()
	_, exists = handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if exists {
		t.Error("Buffer should be released after timeout")
	}

	// Buffered events are discarded, so nothing should be processed
	if mockFormatter.getProcessedCount() != 0 {
		t.Errorf("Buffered events should be discarded, but %d events were processed",
			mockFormatter.getProcessedCount())
	}

	// New event should be processed normally
	parentUUID := "new-parent"
	event2 := createTestUserMessage(sessionName, &parentUUID)
	handler.SendEvent(event2)

	time.Sleep(100 * time.Millisecond)

	if mockFormatter.getProcessedCount() != 1 {
		t.Errorf("New event after timeout should be processed, got %d processed events",
			mockFormatter.getProcessedCount())
	}
}

// Test multiple sessions buffering independently
func TestHandler_MultipleSessionBuffering(t *testing.T) {
	mockFormatter := &mockFormatterWithRecording{}
	handler := &Handler{
		narrator:    &mockNarrator{},
		formatter:   mockFormatter,
		debugMode:   true,
		eventChan:   make(chan Event, 100),
		done:        make(chan struct{}),
		taskTracker: NewTaskTracker(),
		buffers:     make(map[string]*BufferInfo),
	}
	handler.Start()
	defer handler.Stop()

	session1 := "session-1"
	session2 := "session-2"

	// Send ParentUUID==nil events for both sessions
	event1 := createTestUserMessage(session1, nil)
	event2 := createTestUserMessage(session2, nil)

	handler.SendEvent(event1)
	handler.SendEvent(event2)

	time.Sleep(100 * time.Millisecond)

	// Both sessions should have buffers
	handler.bufferMutex.Lock()
	_, exists1 := handler.buffers[session1]
	_, exists2 := handler.buffers[session2]
	handler.bufferMutex.Unlock()

	if !exists1 || !exists2 {
		t.Error("Both sessions should have buffers")
	}

	// Release buffer for session1 only
	hookEvent := createTestHookEvent(session1, "SessionStart:resume")
	handler.SendEvent(hookEvent)

	time.Sleep(100 * time.Millisecond)

	// Check buffer states
	handler.bufferMutex.Lock()
	_, exists1 = handler.buffers[session1]
	_, exists2 = handler.buffers[session2]
	handler.bufferMutex.Unlock()

	if exists1 {
		t.Error("Session1 buffer should be released")
	}
	if !exists2 {
		t.Error("Session2 buffer should still exist")
	}

	// Only the hook event should be processed
	if mockFormatter.getProcessedCount() != 1 {
		t.Errorf("Only hook event should be processed, got %d", mockFormatter.getProcessedCount())
	}
}
