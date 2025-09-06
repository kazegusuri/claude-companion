package event

import (
	"testing"

	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// mockFormatterWithTracking tracks which events were formatted
type mockFormatterWithTracking struct {
	formattedEvents []Event
}

func (m *mockFormatterWithTracking) Format(event Event) (string, error) {
	m.formattedEvents = append(m.formattedEvents, event)
	return "", nil
}

func (m *mockFormatterWithTracking) SetDebugMode(debug bool) {}

// getEventUUID extracts UUID from any event type
func getEventUUID(event Event) string {
	switch e := event.(type) {
	case *HookEvent:
		return e.UUID
	case *UserMessage:
		return e.UUID
	case *AssistantMessage:
		return e.UUID
	case *NotificationEvent:
		return "" // NotificationEvent doesn't have UUID
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
	formatter := &mockFormatterWithTracking{}
	h := &Handler{
		sessionManager: sessionManager,
		formatter:      formatter,
		buffers:        make(map[string]*BufferInfo),
		debugMode:      false,
		taskTracker:    NewTaskTracker(),
	}

	sessionID := "c98f318a-c396-4ce6-a6e9-56699a3b4266"
	sessionFile := &SessionFile{
		Session: sessionID,
		Path:    "/test/transcript.jsonl",
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
				{Type: "text", Text: "I need more context to help you."},
			},
		},
	}

	// Process events
	h.processEvent(hookEvent)
	h.processEvent(userEvent)
	h.processEvent(assistantEvent)

	// Expected UUIDs that should be formatted
	expectedUUIDs := []string{
		"f1f4d2a9-9163-4531-989c-e519a2797cbe", // hookEvent
		"09d4a6f0-3f25-4b66-b101-faa8e9138848", // userEvent
		"3b9f2a92-b18e-458d-8ac9-00d69b0e1de6", // assistantEvent
	}

	// Check that all events were formatted
	if len(formatter.formattedEvents) != len(expectedUUIDs) {
		t.Errorf("Expected %d events to be formatted, got %d", len(expectedUUIDs), len(formatter.formattedEvents))
	}

	// Verify UUIDs of formatted events
	for i, expectedUUID := range expectedUUIDs {
		if i >= len(formatter.formattedEvents) {
			t.Errorf("Missing formatted event at index %d (expected UUID: %s)", i, expectedUUID)
			continue
		}
		actualUUID := getEventUUID(formatter.formattedEvents[i])
		if actualUUID != expectedUUID {
			t.Errorf("Event %d: expected UUID %s, got %s", i, expectedUUID, actualUUID)
		}
	}
}

func TestBufferingWithResume(t *testing.T) {
	// Test case for startup3.jsonl pattern:
	// Events are processed and some are buffered, then released on resume

	sessionManager := handler.NewSessionManager()
	formatter := &mockFormatterWithTracking{}
	h := &Handler{
		sessionManager: sessionManager,
		formatter:      formatter,
		buffers:        make(map[string]*BufferInfo),
		debugMode:      false,
		taskTracker:    NewTaskTracker(),
	}

	sessionID1 := "15498a1f-4f0e-475b-a044-5a9907541d33"
	sessionID2 := "14e690ef-d42d-40d7-ba02-7ea7bfa3f652"
	sessionID3 := "e51a8b11-d429-4f7d-a971-10d6c32c393f"

	// All events use the same SessionFile (session1)
	sessionFile1 := &SessionFile{
		Session: sessionID1,
		Path:    "/test/transcript1.jsonl",
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
	h.processEvent(hookEvent1) // Should be formatted (UUID matches)
	h.processEvent(userEvent1) // Should be formatted
	h.processEvent(hookEvent2) // Should be buffered (UUID mismatch - resume scenario)
	h.processEvent(userEvent2) // Should be buffered (follows buffered event for session1)
	h.processEvent(hookEvent3) // Should be buffered (SessionStart:resume but sessionName != SessionID)
	h.processEvent(userEvent3) // Should be buffered (follows buffered event for session3)
	h.processEvent(hookEvent4) // Should be formatted and release buffer for session1
	h.processEvent(userEvent4) // Should be formatted (after buffer release)

	// Expected UUIDs that should be formatted (buffered events are discarded)
	expectedUUIDs := []string{
		hookEvent1UUID, // hookEvent1
		userEvent1UUID, // userEvent1
		hookEvent4UUID, // hookEvent4
		userEvent4UUID, // userEvent4 (after buffer release)
		// Note: hookEvent2, userEvent2, hookEvent3, userEvent3 are buffered and discarded
	}

	// Check formatted events count
	if len(formatter.formattedEvents) != len(expectedUUIDs) {
		t.Errorf("Expected %d events to be formatted, got %d", len(expectedUUIDs), len(formatter.formattedEvents))
		for i, e := range formatter.formattedEvents {
			t.Logf("Formatted event %d: UUID=%s", i, getEventUUID(e))
		}
	}

	// Verify UUIDs of formatted events
	for i, expectedUUID := range expectedUUIDs {
		if i >= len(formatter.formattedEvents) {
			t.Errorf("Missing formatted event at index %d (expected UUID: %s)", i, expectedUUID)
			continue
		}
		actualUUID := getEventUUID(formatter.formattedEvents[i])
		if actualUUID != expectedUUID {
			t.Errorf("Event %d: expected UUID %s, got %s", i, expectedUUID, actualUUID)
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
