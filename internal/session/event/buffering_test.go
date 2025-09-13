package event

import (
	"fmt"
	"testing"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// getEventUUID extracts UUID from any event type
func getEventUUID(event interface{}) string {
	switch e := event.(type) {
	case *HookEvent:
		return e.UUID
	case *UserMessage:
		return e.UUID
	case *AssistantMessage:
		return e.UUID
	case *NotificationEvent:
		return "" // NotificationEvent doesn't have UUID
	case *internalevent.UserMessage:
		return e.UUID
	case *internalevent.AssistantMessage:
		return e.UUID
	case *internalevent.SystemMessage:
		return e.UUID
	default:
		return ""
	}
}

func TestBufferingNormalStartup(t *testing.T) {
	// Test case for startup1.jsonl pattern:
	// 1. HookEvent with parentUuid=null (SessionStart:startup)
	// 2. UserMessage with parentUuid=<first-event-uuid>
	// 3. AssistantMessage with parentUuid=<user-message-uuid>
	// All events should be processed normally

	sessionManager := handler.NewSessionManager()

	// Create a mock central handler to process AssistantMessage
	centralHandler := NewMockCentralHandler()

	sessionID := "c98f318a-c396-4ce6-a6e9-56699a3b4266"

	h := &Handler{
		sessionManager: sessionManager,
		centralHandler: centralHandler,
		buffers:        make(map[string]*BufferInfo),
		taskTracker:    NewTaskTracker(),
		session: &SessionFile{
			SessionID: sessionID, // Set expected sessionID for normal session start
		},
	}
	sessionFile := &SessionFile{
		SessionID:      sessionID,
		TranscriptPath: "/test/transcript.jsonl",
	}

	// Pre-register session with HandleWarmupEvent
	// This simulates the session being created during warmup
	parentUUID := "parent-uuid"
	warmupEvent := &BaseEvent{
		UUID:        "f1f4d2a9-9163-4531-989c-e519a2797cbe", // Same UUID as the first event
		SessionID:   sessionID,
		CWD:         "/test/workspace",
		TypeString:  "warmup",
		ParentUUID:  &parentUUID,
		IsSidechain: false,
		Session:     sessionFile,
	}
	h.HandleWarmupEvent(warmupEvent)

	// Event 1: HookEvent with parentUuid=null (SessionStart:startup)
	hookEvent := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:        "f1f4d2a9-9163-4531-989c-e519a2797cbe", // Same UUID as warmup
			SessionID:   sessionID,
			CWD:         "/test/workspace",
			TypeString:  "system",
			ParentUUID:  nil, // null parentUuid
			IsSidechain: false,
			Session:     sessionFile,
		},
		HookEventType: "SessionStart:startup",
	}

	// Event 2: UserMessage with parentUuid pointing to first event
	parentUUID1 := "f1f4d2a9-9163-4531-989c-e519a2797cbe"
	userEvent := &UserMessage{
		BaseEvent: BaseEvent{
			UUID:        "09d4a6f0-3f25-4b66-b101-faa8e9138848",
			SessionID:   sessionID,
			CWD:         "/test/workspace",
			TypeString:  "user",
			ParentUUID:  &parentUUID1,
			IsSidechain: false,
			Session:     sessionFile,
		},
		Message: UserMessageContent{
			Role:    "user",
			Content: "aaa",
		},
	}

	// Event 3: AssistantMessage with parentUuid pointing to user message
	parentUUID2 := "09d4a6f0-3f25-4b66-b101-faa8e9138848"
	assistantEvent := &AssistantMessage{
		BaseEvent: BaseEvent{
			UUID:        "3b9f2a92-b18e-458d-8ac9-00d69b0e1de6",
			SessionID:   sessionID,
			CWD:         "/test/workspace",
			TypeString:  "assistant",
			ParentUUID:  &parentUUID2,
			IsSidechain: false,
			Session:     sessionFile,
		},
		Message: AssistantMessageContent{
			Role: "assistant",
			Content: []AssistantContent{
				{Type: "tool_use", ID: "tool-123", Name: "calculator", Input: map[string]interface{}{"expression": "2+2"}},
			},
		},
	}

	// Process events
	h.processEvent(hookEvent)
	h.processEvent(userEvent)
	h.processEvent(assistantEvent)

	// Expected UUIDs that should be formatted
	// Note: HookEvent is now sent to central handler and not formatted by session handler
	// Note: UserMessage is now handled by central event handler and not formatted by session handler
	// Note: AssistantMessage is now handled by central event handler and not formatted by session handler
	// All events are now handled by central handler

	// Check that events were sent to central handler
	centralEvents := centralHandler.GetEvents()

	// Expected events in order
	expectedEvents := []struct {
		eventType string
		uuid      string
	}{
		{"SystemMessage", "f1f4d2a9-9163-4531-989c-e519a2797cbe"},    // hookEvent (SessionStart:startup)
		{"UserMessage", "09d4a6f0-3f25-4b66-b101-faa8e9138848"},      // userEvent
		{"AssistantMessage", "3b9f2a92-b18e-458d-8ac9-00d69b0e1de6"}, // assistantEvent
	}

	if len(centralEvents) != len(expectedEvents) {
		t.Errorf("Expected %d events in central handler, got %d", len(expectedEvents), len(centralEvents))
		for i, event := range centralEvents {
			t.Logf("Event %d: %T UUID=%s", i, event, getEventUUID(event))
		}
		return
	}

	// Check each event's type and UUID
	for i, expected := range expectedEvents {
		event := centralEvents[i]
		actualUUID := getEventUUID(event)

		// Check event type
		actualType := ""
		switch event.(type) {
		case *internalevent.UserMessage:
			actualType = "UserMessage"
		case *internalevent.AssistantMessage:
			actualType = "AssistantMessage"
		case *internalevent.SystemMessage:
			actualType = "SystemMessage"
		default:
			actualType = fmt.Sprintf("%T", event)
		}

		if actualType != expected.eventType {
			t.Errorf("Event %d: expected type %s, got %s", i, expected.eventType, actualType)
		}

		if actualUUID != expected.uuid {
			t.Errorf("Event %d: expected UUID %s, got %s", i, expected.uuid, actualUUID)
		}
	}
}

func TestBufferingWithResume(t *testing.T) {
	// Test case for startup3.jsonl pattern:
	// Events are processed and some are buffered, then released on resume

	sessionManager := handler.NewSessionManager()

	// Create a mock central handler to process AssistantMessage
	centralHandler := NewMockCentralHandler()

	sessionID1 := "15498a1f-4f0e-475b-a044-5a9907541d33"
	sessionID2 := "14e690ef-d42d-40d7-ba02-7ea7bfa3f652"
	sessionID3 := "e51a8b11-d429-4f7d-a971-10d6c32c393f"

	h := &Handler{
		sessionManager: sessionManager,
		centralHandler: centralHandler,
		buffers:        make(map[string]*BufferInfo),
		taskTracker:    NewTaskTracker(),
		session: &SessionFile{
			SessionID: sessionID1, // Handler expects sessionID1 for this session file
		},
	}

	// All events use the same SessionFile (session1)
	sessionFile1 := &SessionFile{
		SessionID:      sessionID1,
		TranscriptPath: "/test/transcript1.jsonl",
	}

	// Pre-register sessions with HandleWarmupEvent
	// Session1: register with the first event's UUID
	parentUUID := "parent-uuid"
	warmupEvent1 := &BaseEvent{
		UUID:        "fbe8aea6-88ec-4f4d-a14e-9adca5fb7759", // Same as hookEvent1
		SessionID:   sessionID1,
		CWD:         "/test/workspace",
		TypeString:  "warmup",
		ParentUUID:  &parentUUID,
		IsSidechain: false,
		Session:     sessionFile1,
	}
	h.HandleWarmupEvent(warmupEvent1)

	// Session2: register with old UUID (will be resumed)
	warmupEvent2 := &BaseEvent{
		UUID:        "old-uuid-for-session2", // Different from hookEvent2
		SessionID:   sessionID2,
		CWD:         "/test/workspace",
		TypeString:  "warmup",
		ParentUUID:  &parentUUID,
		IsSidechain: false,
		Session:     sessionFile1, // Same SessionFile
	}
	h.HandleWarmupEvent(warmupEvent2)

	// Session3: register normally
	warmupEvent3 := &BaseEvent{
		UUID:        "old-uuid-for-session3",
		SessionID:   sessionID3,
		CWD:         "/test/workspace",
		TypeString:  "warmup",
		ParentUUID:  &parentUUID,
		IsSidechain: false,
		Session:     sessionFile1, // Same SessionFile
	}
	h.HandleWarmupEvent(warmupEvent3)

	// Event 1: HookEvent with parentUuid=null for session1 (normal start - UUID matches)
	hookEvent1UUID := "fbe8aea6-88ec-4f4d-a14e-9adca5fb7759" // Same as warmup
	hookEvent1 := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:        hookEvent1UUID,
			SessionID:   sessionID1,
			CWD:         "/test/workspace",
			TypeString:  "system",
			ParentUUID:  nil,
			IsSidechain: false,
			Session:     sessionFile1,
		},
		HookEventType: "SessionStart:startup",
	}

	// Event 2: UserMessage for session1
	userEvent1UUID := "c2900a6e-117d-4b2c-9253-36cae51f610e"
	userEvent1 := &UserMessage{
		BaseEvent: BaseEvent{
			UUID:        userEvent1UUID,
			SessionID:   sessionID1,
			CWD:         "/test/workspace",
			TypeString:  "user",
			ParentUUID:  &hookEvent1UUID,
			IsSidechain: false,
			Session:     sessionFile1,
		},
		Message: UserMessageContent{
			Role:    "user",
			Content: "hi",
		},
	}

	// Event 3: HookEvent with parentUuid=null for session2 (resume - UUID different)
	hookEvent2UUID := "70e2ed6e-47d9-4a0e-bb17-b50d890620ff" // Different from warmup
	hookEvent2 := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:        hookEvent2UUID,
			SessionID:   sessionID2,
			CWD:         "/test/workspace",
			TypeString:  "system",
			ParentUUID:  nil,
			IsSidechain: false,
			Session:     sessionFile1, // Same SessionFile
		},
		HookEventType: "SessionStart:startup",
	}

	// Event 4: UserMessage from old session but with current sessionId (should be buffered)
	userEvent2UUID := "8e2fcffa-6721-4b8d-b9fa-d469cf9abf45"
	userEvent2 := &UserMessage{
		BaseEvent: BaseEvent{
			UUID:        userEvent2UUID,
			SessionID:   sessionID1, // Note: sessionId changed to session1
			CWD:         "/test/workspace",
			TypeString:  "user",
			ParentUUID:  &hookEvent2UUID,
			IsSidechain: false,
			Session:     sessionFile1,
		},
		Message: UserMessageContent{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hello"},
			},
		},
	}

	// Event 5: HookEvent with SessionStart:resume for different session (session3)
	hookEvent3ParentUUID := "2efd38d0-179e-4f74-aff8-0894053a2f30"
	hookEvent3UUID := "4d9936b7-bb1f-49bd-8a63-40aa1e0150e6"
	hookEvent3 := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:        hookEvent3UUID,
			SessionID:   sessionID3,
			CWD:         "/test/workspace",
			TypeString:  "system",
			ParentUUID:  &hookEvent3ParentUUID,
			IsSidechain: false,
			Session:     sessionFile1, // Same SessionFile
		},
		HookEventType: "SessionStart:resume",
	}

	// Event 6: UserMessage for session3
	userEvent3UUID := "09ee04d5-ad46-4a9a-a5f4-4164e2893a4d"
	userEvent3 := &UserMessage{
		BaseEvent: BaseEvent{
			UUID:        userEvent3UUID,
			SessionID:   sessionID3,
			CWD:         "/test/workspace",
			TypeString:  "user",
			ParentUUID:  &hookEvent3UUID,
			IsSidechain: false,
			Session:     sessionFile1, // Same SessionFile
		},
		Message: UserMessageContent{
			Role: "user",
			Content: []interface{}{
				map[string]interface{}{"type": "text", "text": "hi"},
			},
		},
	}

	// Event 7: HookEvent with SessionStart:resume for session1 (should release buffer)
	hookEvent4ParentUUID := "e8fce59a-5f08-471c-a09c-d7a2040b151d"
	hookEvent4UUID := "3911a506-3578-4cd4-a98b-52a74c8dd4e5"
	hookEvent4 := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:        hookEvent4UUID,
			SessionID:   sessionID1,
			CWD:         "/test/workspace",
			TypeString:  "system",
			ParentUUID:  &hookEvent4ParentUUID,
			IsSidechain: false,
			Session:     sessionFile1,
		},
		HookEventType: "SessionStart:resume",
	}

	// Event 8: UserMessage after resume (should be formatted normally)
	userEvent4UUID := "new-user-event-after-resume"
	userEvent4 := &UserMessage{
		BaseEvent: BaseEvent{
			UUID:        userEvent4UUID,
			SessionID:   sessionID1,
			CWD:         "/test/workspace",
			TypeString:  "user",
			ParentUUID:  &hookEvent4UUID,
			IsSidechain: false,
			Session:     sessionFile1,
		},
		Message: UserMessageContent{
			Role:    "user",
			Content: "message after resume",
		},
	}

	// Process events
	h.processEvent(hookEvent1) // Normal start: parentUUID=null, sessionID matches handler's sessionID
	h.processEvent(userEvent1) // Processed normally
	h.processEvent(hookEvent2) // Resume start: parentUUID=null, sessionID2 != handler's sessionID1, starts buffering
	h.processEvent(userEvent2) // Buffered: part of resumed session's history
	h.processEvent(hookEvent3) // Buffered: SessionStart:resume but sessionID3 != handler's sessionID1
	h.processEvent(userEvent3) // Buffered: follows buffered event
	h.processEvent(hookEvent4) // Resume end: SessionStart:resume with sessionID1 == handler's sessionID1, releases buffer
	h.processEvent(userEvent4) // Processed normally after buffer release

	// Check that events were sent to central handler (non-buffered ones)
	centralEvents := centralHandler.GetEvents()

	// Expected events: Only events that were NOT buffered should be in central handler
	// - hookEvent1: Normal start, sent to central as SystemMessage
	// - userEvent1: Processed normally, sent to central
	// - hookEvent2-3, userEvent2-3: Buffered and discarded (resume detection)
	// - ResumeEvent: Sent when buffer is released
	// - hookEvent4: SessionStart:resume, releases buffer, sent to central as SystemMessage
	// - userEvent4: Processed normally after buffer release, sent to central
	expectedEvents := []struct {
		eventType   string
		uuid        string
		description string
	}{
		{"SystemMessage", "fbe8aea6-88ec-4f4d-a14e-9adca5fb7759", "hookEvent1 - normal start"},
		{"UserMessage", "c2900a6e-117d-4b2c-9253-36cae51f610e", "userEvent1 - normal processing"},
		{"ResumeEvent", "", "Resume event sent when buffer is released"},
		{"SystemMessage", "3911a506-3578-4cd4-a98b-52a74c8dd4e5", "hookEvent4 - SessionStart:resume"},
		{"UserMessage", "new-user-event-after-resume", "userEvent4 - after buffer release"},
	}

	if len(centralEvents) != len(expectedEvents) {
		t.Errorf("Expected %d events in central handler, got %d", len(expectedEvents), len(centralEvents))
		for i, event := range centralEvents {
			t.Logf("Event %d: %T UUID=%s", i, event, getEventUUID(event))
		}
		return
	}

	// Check each event's type and UUID (skip UUID check for ResumeEvent)
	for i, expected := range expectedEvents {
		event := centralEvents[i]

		// Check event type
		actualType := ""
		switch ev := event.(type) {
		case *internalevent.UserMessage:
			actualType = "UserMessage"
		case *internalevent.AssistantMessage:
			actualType = "AssistantMessage"
		case *internalevent.SystemMessage:
			actualType = "SystemMessage"
		case *internalevent.ResumeEvent:
			actualType = "ResumeEvent"
			// Check ResumeEvent content
			if ev.BufferedCount <= 0 {
				t.Errorf("ResumeEvent should have BufferedCount > 0, got %d", ev.BufferedCount)
			}
		default:
			actualType = fmt.Sprintf("%T", event)
		}

		if actualType != expected.eventType {
			t.Errorf("Event %d (%s): expected type %s, got %s",
				i, expected.description, expected.eventType, actualType)
		}

		// Check UUID for non-ResumeEvent
		if expected.uuid != "" {
			actualUUID := getEventUUID(event)
			if actualUUID != expected.uuid {
				t.Errorf("Event %d (%s): expected UUID %s, got %s",
					i, expected.description, expected.uuid, actualUUID)
			}
		}
	}

	// Verify that all buffers are released after hookEvent4 (SessionStart:resume for session1)
	// All events use the same SessionFile (session1), so the buffer was released
	if len(h.buffers) != 0 {
		t.Errorf("Expected all buffers to be released, got %d buffers", len(h.buffers))
		for sessionName := range h.buffers {
			t.Logf("Buffer still exists for session: %s", sessionName)
		}
	}
}
