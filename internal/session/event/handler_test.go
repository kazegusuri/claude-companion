package event

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/server/handler"

	"github.com/kazegusuri/claude-companion/internal/narrator"
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

func (m *mockNarrator) NarrateAPIError(statusCode int, errorType string, message string) (string, bool) {
	return fmt.Sprintf("APIエラー %d: %s", statusCode, message), false
}

// MockCentralHandler is a mock implementation of CentralEventHandler for testing
type MockCentralHandler struct {
	mu     sync.Mutex
	events []internalevent.Event
}

// NewMockCentralHandler creates a new mock central handler
func NewMockCentralHandler() *MockCentralHandler {
	return &MockCentralHandler{
		events: make([]internalevent.Event, 0),
	}
}

// SendEvent stores the event for later inspection
func (m *MockCentralHandler) SendEvent(event internalevent.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

// GetEvents returns all stored events
func (m *MockCentralHandler) GetEvents() []internalevent.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]internalevent.Event(nil), m.events...)
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
	mockNarr := &mockNarrator{}
	mockPrint := &mockPrinter{}
	centralHandler := internalevent.NewHandler(sessionManager, mockNarr, mockPrint, nil)
	sessionFile := &SessionFile{
		SessionID:      "test-session",
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}
	handler := NewHandler(mockNarr, sessionManager, centralHandler, sessionFile)
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
	// Create handler with mock central handler to capture events
	sessionManager := handler.NewSessionManager()
	mockNarr := &mockNarrator{}
	mockCentral := &mockCentralHandlerWithLock{}
	sessionFile := &SessionFile{
		SessionID:      "test-session",
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}
	handler := NewHandler(mockNarr, sessionManager, mockCentral, sessionFile)
	handler.Start()
	defer handler.Stop()

	tests := []struct {
		name          string
		taskMessage   *AssistantMessage
		resultMessage *UserMessage
		expectedEvent *internalevent.TaskCompletionMessage
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
					SessionID:   "test-session",
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
			expectedEvent: &internalevent.TaskCompletionMessage{
				Session: internalevent.Session{
					SessionID:      "test-session",
					TranscriptPath: "/test/transcript.jsonl",
				},
				TaskInfo: internalevent.TaskInfo{
					ToolUseID:    "task-id-123",
					Description:  "データベース最適化",
					SubagentType: "database-engineer",
				},
				// Timestamp will be set during test
			},
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
					SessionID:   "test-session",
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
			expectedEvent: &internalevent.TaskCompletionMessage{
				Session: internalevent.Session{
					SessionID:      "test-session",
					TranscriptPath: "/test/transcript.jsonl",
				},
				TaskInfo: internalevent.TaskInfo{
					ToolUseID:    "task-id-456",
					Description:  "コード解析",
					SubagentType: "",
				},
				// Timestamp will be set during test
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear captured events
			mockCentral.capturedEvents = nil

			// Send task message first
			handler.SendEvent(tt.taskMessage)
			time.Sleep(50 * time.Millisecond)

			// Send result message
			handler.SendEvent(tt.resultMessage)
			time.Sleep(100 * time.Millisecond) // Wait for async processing

			// Get captured events
			capturedEvents := mockCentral.getCapturedEvents()

			// Filter for TaskCompletionMessage events
			var taskCompletionEvents []*internalevent.TaskCompletionMessage
			for _, event := range capturedEvents {
				if taskEvent, ok := event.(*internalevent.TaskCompletionMessage); ok {
					taskCompletionEvents = append(taskCompletionEvents, taskEvent)
				}
			}

			// Should have exactly one TaskCompletionMessage
			if len(taskCompletionEvents) != 1 {
				t.Fatalf("Expected 1 TaskCompletionMessage, got %d", len(taskCompletionEvents))
			}

			// Compare the event (ignoring timestamp)
			actual := taskCompletionEvents[0]
			tt.expectedEvent.Timestamp = actual.Timestamp // Use actual timestamp for comparison

			if diff := cmp.Diff(tt.expectedEvent, actual); diff != "" {
				t.Errorf("TaskCompletionMessage mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHandler_NonTaskToolResult(t *testing.T) {
	// Create handler with mock narrator and session manager
	sessionManager := handler.NewSessionManager()
	mockNarr := &mockNarrator{}
	mockPrint := &mockPrinter{}
	centralHandler := internalevent.NewHandler(sessionManager, mockNarr, mockPrint, nil)
	sessionFile := &SessionFile{
		SessionID:      "test-session",
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}
	handler := NewHandler(mockNarr, sessionManager, centralHandler, sessionFile)
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

// mockCentralHandlerWithLock is an extended version with mutex for thread safety
type mockCentralHandlerWithLock struct {
	capturedEvents []internalevent.Event
	mu             sync.Mutex
}

func (m *mockCentralHandlerWithLock) SendEvent(event internalevent.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.capturedEvents = append(m.capturedEvents, event)
}

func (m *mockCentralHandlerWithLock) getCapturedEvents() []internalevent.Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]internalevent.Event{}, m.capturedEvents...)
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
			SessionID:   sessionName,
			Session: &SessionFile{
				TranscriptPath: "/test/path.jsonl",
				Project:        "test-project",
				SessionID:      sessionName,
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
			SessionID:   sessionName,
			Session: &SessionFile{
				TranscriptPath: "/test/path.jsonl",
				Project:        "test-project",
				SessionID:      sessionName,
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
	// Create handler with mock central handler
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()
	handler := &Handler{
		narrator:       &mockNarrator{},
		centralHandler: centralHandler,
		taskTracker:    NewTaskTracker(),
		buffers:        make(map[string]*BufferInfo),
		sessionManager: sessionManager,
		session: &SessionFile{
			SessionID: "test-session", // Set the expected sessionID
		},
	}
	handler.Start()
	defer handler.Stop()

	sessionName := "test-session"

	// Send event with ParentUUID==nil but different SessionID to trigger resume
	event1 := createTestUserMessage("different-session", nil) // Different SessionID
	event1.BaseEvent.Session = &SessionFile{
		SessionID:      sessionName, // But same session name for buffering
		TranscriptPath: "/test/path.jsonl",
	}
	handler.SendEvent(event1)

	// Wait a bit to ensure processing
	time.Sleep(100 * time.Millisecond)

	// Check that event was buffered (not sent to central handler)
	if len(centralHandler.GetEvents()) != 0 {
		t.Errorf("Event with ParentUUID==nil should be buffered, but %d events were sent to central handler",
			len(centralHandler.GetEvents()))
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

	// Still no events should be sent to central handler
	if len(centralHandler.GetEvents()) != 0 {
		t.Errorf("Subsequent events should also be buffered, but %d events were sent to central handler",
			len(centralHandler.GetEvents()))
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
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()
	handler := &Handler{
		narrator:       &mockNarrator{},
		centralHandler: centralHandler,
		taskTracker:    NewTaskTracker(),
		buffers:        make(map[string]*BufferInfo),
		sessionManager: sessionManager,
		session: &SessionFile{
			SessionID: "session-222", // Set the expected sessionID
		},
	}
	handler.Start()
	defer handler.Stop()

	sessionName := "session-222"

	// Send event with ParentUUID==nil but different SessionID to trigger buffering
	event1 := createTestUserMessage("different-session", nil)
	event1.BaseEvent.Session = &SessionFile{
		SessionID:      sessionName,
		TranscriptPath: "/test/path.jsonl",
	}
	handler.SendEvent(event1)

	time.Sleep(100 * time.Millisecond)

	// Verify buffering started
	handler.bufferMutex.Lock()
	_, exists := handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if !exists {
		t.Error("Buffer should exist before SessionStart:resume")
	}

	// Send SessionStart:resume event with correct sessionID
	hookEvent := createTestHookEvent(sessionName, "SessionStart:resume")
	hookEvent.BaseEvent.SessionID = "session-222" // Match handler's sessionID
	handler.SendEvent(hookEvent)

	time.Sleep(100 * time.Millisecond)

	// Buffer should be released
	handler.bufferMutex.Lock()
	_, exists = handler.buffers[sessionName]
	handler.bufferMutex.Unlock()

	if exists {
		t.Error("Buffer should be released after SessionStart:resume")
	}

	// Check that events were sent to central handler
	events := centralHandler.GetEvents()
	if len(events) < 2 {
		t.Errorf("Expected at least 2 events (ResumeEvent and HookEvent), got %d", len(events))
	}

	// Check for ResumeEvent
	var foundResumeEvent bool
	for _, e := range events {
		if resumeEvent, ok := e.(*internalevent.ResumeEvent); ok {
			foundResumeEvent = true
			if resumeEvent.BufferedCount != 1 {
				t.Errorf("Expected BufferedCount to be 1, got %d", resumeEvent.BufferedCount)
			}
			if resumeEvent.ResumedFromID != "different-session" {
				t.Errorf("Expected ResumedFromID to be 'different-session', got %s", resumeEvent.ResumedFromID)
			}
		}
	}

	if !foundResumeEvent {
		t.Error("ResumeEvent should be sent to central handler")
	}
}

// Test buffer release on timeout
func TestHandler_ReleaseBufferOnTimeout(t *testing.T) {
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()
	handler := &Handler{
		narrator:       &mockNarrator{},
		centralHandler: centralHandler,
		taskTracker:    NewTaskTracker(),
		buffers:        make(map[string]*BufferInfo),
		sessionManager: sessionManager,
		session: &SessionFile{
			SessionID: "timeout-test", // Set the expected sessionID
		},
	}
	handler.Start()
	defer handler.Stop()

	sessionName := "timeout-test"

	// Send event with ParentUUID==nil and different sessionID to trigger buffering
	// Use AssistantMessage instead of UserMessage since UserMessage is now handled by central handler
	event1 := &AssistantMessage{
		BaseEvent: BaseEvent{
			IsSidechain: false,
			TypeString:  EventTypeAssistant,
			UUID:        "assistant-1",
			Timestamp:   time.Now(),
			ParentUUID:  nil,
			SessionID:   "different-session", // Different sessionID to trigger buffering
			Session: &SessionFile{
				TranscriptPath: "/test/path.jsonl",
				Project:        "test-project",
				SessionID:      sessionName,
			},
		},
		Message: AssistantMessageContent{
			Model: "test-model",
			Content: []AssistantContent{
				{Type: "text", Text: "Test response"},
			},
		},
	}
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

	// Buffered events are discarded, so nothing should be sent to central handler initially
	// (they were discarded after timeout)

	// Check central handler events before sending new event
	centralEventsBefore := len(centralHandler.GetEvents())

	// New event should be processed normally through central handler
	parentUUID := "new-parent"
	// Use UserMessage which goes to central handler
	event2 := &UserMessage{
		BaseEvent: BaseEvent{
			IsSidechain: false,
			TypeString:  EventTypeUser,
			UUID:        "user-2",
			Timestamp:   time.Now(),
			ParentUUID:  &parentUUID,
			SessionID:   sessionName,
			Session: &SessionFile{
				TranscriptPath: "/test/path.jsonl",
				Project:        "test-project",
				SessionID:      sessionName,
			},
		},
		Message: UserMessageContent{
			Role:    "user",
			Content: "test message after timeout",
		},
	}
	handler.SendEvent(event2)

	time.Sleep(100 * time.Millisecond)

	// Check that the new event was processed through central handler
	centralEventsAfter := len(centralHandler.GetEvents())
	if centralEventsAfter != centralEventsBefore+1 {
		t.Errorf("New event after timeout should be processed through central handler, got %d events (was %d)",
			centralEventsAfter, centralEventsBefore)
	}
}

// Test multiple sessions buffering independently
func TestHandler_MultipleSessionBuffering(t *testing.T) {
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()
	handler := &Handler{
		narrator:       &mockNarrator{},
		centralHandler: centralHandler,
		taskTracker:    NewTaskTracker(),
		buffers:        make(map[string]*BufferInfo),
		sessionManager: sessionManager,
		session: &SessionFile{
			SessionID: "main-session", // Set the expected sessionID
		},
	}
	handler.Start()
	defer handler.Stop()

	session1 := "session-1"
	session2 := "session-2"

	// Send ParentUUID==nil events with different sessionIDs to trigger buffering
	event1 := createTestUserMessage("different-1", nil)
	event1.BaseEvent.Session = &SessionFile{SessionID: session1, TranscriptPath: "/test/path1.jsonl"}

	event2 := createTestUserMessage("different-2", nil)
	event2.BaseEvent.Session = &SessionFile{SessionID: session2, TranscriptPath: "/test/path2.jsonl"}

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
	hookEvent.BaseEvent.SessionID = "main-session" // Match handler's sessionID to trigger release
	hookEvent.BaseEvent.Session = &SessionFile{SessionID: session1, TranscriptPath: "/test/path1.jsonl"}
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

	// Check that events were sent to central handler
	// ResumeEvent and HookEvent for session-1 should be sent (session-2 remains buffered)
	events := centralHandler.GetEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events to be sent to central handler (ResumeEvent and HookEvent for session-1), got %d", len(events))
	}

	// Check for ResumeEvent
	var foundResumeEvent bool
	for _, e := range events {
		if _, ok := e.(*internalevent.ResumeEvent); ok {
			foundResumeEvent = true
		}
	}
	if !foundResumeEvent {
		t.Error("ResumeEvent should be sent when buffer is released")
	}
}
