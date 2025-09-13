package event

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// mustParseTime is a helper function to parse time in tests
func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

// parserIntegrationTestCase defines a test case for parser integration tests
type parserIntegrationTestCase struct {
	name            string
	input           string
	wantEvent       interface{} // Expected event sent to central handler
	expectEventSent bool        // Whether event should be sent to central handler
}

// TestIntegration_ParseAndSendToCentral tests parsing JSON input and sending to central handler
func TestIntegration_ParseAndSendToCentral(t *testing.T) {
	testGroups := map[string][]parserIntegrationTestCase{
		"SystemMessage":    systemMessageTestCases,
		"SummaryEvent":     summaryEventTestCases,
		"UserMessage":      userMessageTestCases,
		"AssistantMessage": assistantMessageTestCases,
	}

	for groupName, tests := range testGroups {
		t.Run(groupName, func(t *testing.T) {
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// Create parser with session info
					parser := NewParserWithPath("/test/session.jsonl")

					// Parse the event
					event, err := parser.Parse(tt.input)
					if err != nil {
						t.Fatalf("Parse() error = %v", err)
					}

					// Create mock central handler
					mockCentral := &mockCentralHandler{}

					// Create session manager
					sessionManager := handler.NewSessionManager()

					// Create handler with mock central handler
					sessionFile := &SessionFile{
						SessionID:      "test-session",
						TranscriptPath: "/test/session.jsonl",
						Project:        "test-project",
					}
					h := NewHandler(&mockNarrator{}, sessionManager, mockCentral, sessionFile)

					// Process the event
					h.processEvent(event)

					// Check events sent to central handler based on expectEventSent flag
					if tt.expectEventSent {
						// This event should be sent to central handler
						if len(mockCentral.events) != 1 {
							t.Errorf("Expected 1 event sent to central handler, got %d", len(mockCentral.events))
							return
						}

						gotEvent := mockCentral.events[0]

						// Compare using cmp.Diff based on expected type
						opts := []cmp.Option{
							cmpopts.IgnoreFields(internalevent.SystemMessage{}, "Narration"),
							cmpopts.IgnoreFields(internalevent.SummaryEvent{}, "Narration"),
							cmpopts.IgnoreFields(internalevent.NotificationEvent{}, "Narration"),
							cmpopts.IgnoreFields(internalevent.UserMessage{}, "Narration"),
						}

						if diff := cmp.Diff(tt.wantEvent, gotEvent, opts...); diff != "" {
							t.Errorf("%s mismatch (-want +got):\n%s", groupName, diff)
						}
					} else {
						// This event should not be sent to central handler
						if len(mockCentral.events) != 0 {
							t.Errorf("Expected no events sent to central handler, got %d", len(mockCentral.events))
						}
					}
				})
			}
		})
	}
}

// TestIntegration_Buffering tests parsing various JSONL files and verifying central events
func TestIntegration_Buffering(t *testing.T) {
	tests := []struct {
		name               string
		filename           string
		handlerSessionID   string
		expectedEventTypes []string
		description        string
	}{
		{
			name:             "double_resume",
			filename:         "testdata/double_resume.jsonl",
			handlerSessionID: "8b22a4cc-23bc-4910-adfc-f8f5c43ff5d3", // Session B
			expectedEventTypes: []string{
				"ResumeEvent",   // Line 10: SessionStart:resume (resume end)
				"SystemMessage", // Additional SystemMessage event
				"UserMessage",   // Line 11: Meta message (isMeta=true) - actually sent
				"UserMessage",   // Line 12: Regular user message
				"UserMessage",   // Line 13: Regular user message
				"UserMessage",   // Line 14: Regular user message
				"UserMessage",   // Line 15: Regular user message
				"ResumeEvent",   // Line 27: SessionStart:resume (second resume end)
				"SystemMessage", // Additional SystemMessage event
			},
			description: "Session with double resume scenarios",
		},
		{
			name:             "buffer_until_sessionstart_resume",
			filename:         "testdata/buffer_until_sessionstart_resume.jsonl",
			handlerSessionID: "handler-session",
			expectedEventTypes: []string{
				"ResumeEvent",   // SessionStart:resume hook event (resume end)
				"SystemMessage", // System message sent to central handler
			},
			description: "Events buffered until SessionStart:resume (buffered events are discarded)",
		},
		{
			name:             "normal_session_start_no_buffering",
			filename:         "testdata/normal_session_start_no_buffering.jsonl",
			handlerSessionID: "handler-session",
			expectedEventTypes: []string{
				"SystemMessage", // System message with matching sessionID
				"UserMessage",   // User message should be sent normally
			},
			description: "Normal session start without buffering (both events sent immediately)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock central handler
			mockCentral := &mockCentralHandler{}

			// Create session manager
			sessionManager := handler.NewSessionManager()

			// Create handler with mock central handler
			sessionFile := &SessionFile{
				SessionID:      tt.handlerSessionID,
				TranscriptPath: tt.filename,
				Project:        "test-project",
			}
			h := NewHandler(&mockNarrator{}, sessionManager, mockCentral, sessionFile)

			// Create parser with testdata file path
			parser := NewParserWithPath(tt.filename)

			// Read and parse the file line by line
			file, err := os.Open(tt.filename)
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", tt.filename, err)
			}
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := scanner.Text()
				if strings.TrimSpace(line) == "" {
					continue // Skip empty lines
				}

				event, err := parser.Parse(line)
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				h.processEvent(event)
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Error reading file %s: %v", tt.filename, err)
			}

			// Verify the number of events
			if len(mockCentral.events) != len(tt.expectedEventTypes) {
				t.Errorf("Expected %d events, got %d", len(tt.expectedEventTypes), len(mockCentral.events))
				for i, event := range mockCentral.events {
					t.Logf("Event %d: %T", i+1, event)
				}
				return
			}

			// Verify event types
			for i, expectedType := range tt.expectedEventTypes {
				gotEvent := mockCentral.events[i]
				gotType := getEventType(gotEvent)

				if gotType != expectedType {
					t.Errorf("Event %d: expected type %s, got %s", i+1, expectedType, gotType)
				}
			}
		})
	}
}

// getEventType returns the string representation of the event type
func getEventType(event interface{}) string {
	switch event.(type) {
	case *internalevent.ResumeEvent:
		return "ResumeEvent"
	case *internalevent.UserMessage:
		return "UserMessage"
	case *internalevent.AssistantMessage:
		return "AssistantMessage"
	case *internalevent.SystemMessage:
		return "SystemMessage"
	case *internalevent.SummaryEvent:
		return "SummaryEvent"
	case *internalevent.NotificationEvent:
		return "NotificationEvent"
	default:
		return "Unknown"
	}
}
