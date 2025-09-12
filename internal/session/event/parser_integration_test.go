package event

import (
	"encoding/json"
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
					// Create parser
					parser := NewParser()

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
					h := &Handler{
						sessionManager: sessionManager,
						centralHandler: mockCentral,
						buffers:        make(map[string]*BufferInfo),
						taskTracker:    NewTaskTracker(),
					}

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

// parseJSON is a helper function to parse JSON into a map
func parseJSON(input string) (map[string]json.RawMessage, error) {
	var result map[string]json.RawMessage
	err := json.Unmarshal([]byte(input), &result)
	return result, err
}

// TestIntegration_Buffering tests session buffering and unbuffering
func TestIntegration_Buffering(t *testing.T) {
	tests := []struct {
		name       string
		events     []string // JSON events to process in order
		wantEvents int      // Expected number of events sent to central handler
	}{
		{
			name: "buffer_until_user_message",
			events: []string{
				// System message should be buffered
				`{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"sys1","sessionID":"test-session","cwd":"/test/dir","content":"System message","isMeta":false}`,
				// User message should trigger unbuffering
				`{"type":"user","timestamp":"2025-01-26T15:30:46Z","uuid":"user1","sessionID":"test-session","cwd":"/test/dir","content":"User input","isMeta":false}`,
			},
			wantEvents: 2, // Both should be sent
		},
		{
			name: "no_buffering_after_user_message",
			events: []string{
				// User message first
				`{"type":"user","timestamp":"2025-01-26T15:30:45Z","uuid":"user1","sessionID":"test-session","cwd":"/test/dir","content":"User input","isMeta":false}`,
				// System message should not be buffered
				`{"type":"system","timestamp":"2025-01-26T15:30:46Z","uuid":"sys1","sessionID":"test-session","cwd":"/test/dir","content":"System message","isMeta":false}`,
			},
			wantEvents: 2, // Both should be sent immediately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock central handler
			mockCentral := &mockCentralHandler{}

			// Create session manager
			sessionManager := handler.NewSessionManager()

			// Create handler with mock central handler
			h := &Handler{
				sessionManager: sessionManager,
				centralHandler: mockCentral,
				buffers:        make(map[string]*BufferInfo),
				taskTracker:    NewTaskTracker(),
			}

			// Create parser
			parser := NewParser()

			// Process each event
			for _, eventJSON := range tt.events {
				event, err := parser.Parse(eventJSON)
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				h.processEvent(event)
			}

			// Check number of events sent
			if len(mockCentral.events) != tt.wantEvents {
				t.Errorf("Expected %d events sent to central handler, got %d", tt.wantEvents, len(mockCentral.events))
			}
		})
	}
}

// TestIntegration_AssistantMessageFormatting tests formatting of assistant messages
func TestIntegration_AssistantMessageFormatting(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantEvent interface{}
	}{
		{
			name:  "assistant_with_content",
			input: `{"type":"assistant","timestamp":"2025-01-26T15:30:45Z","uuid":"asst1","sessionID":"test-session","cwd":"/test/dir","content":"Here is my response","isMeta":false,"toolUsesMetadata":[],"role":"assistant"}`,
			wantEvent: &internalevent.AssistantMessage{
				SessionMessageBase: internalevent.SessionMessageBase{
					UUID:        "asst1",
					Type:        internalevent.MessageTypeAssistant,
					IsSidechain: false,
					CWD:         "/test/dir",
					Timestamp:   mustParseTime("2025-01-26T15:30:45Z"),
					IsMeta:      false,
				},
				Session: internalevent.Session{
					SessionID:      "test-session",
					TranscriptPath: "",
				},
				Message: internalevent.AssistantMessageData{
					Content: &internalevent.AssistantMessageContentList{
						Items: []internalevent.AssistantMessageContentItem{
							&internalevent.AssistantMessageContentText{
								Type: "text",
								Text: "Here is my response",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create parser
			parser := NewParser()

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
			h := &Handler{
				sessionManager: sessionManager,
				centralHandler: mockCentral,
				buffers:        make(map[string]*BufferInfo),
				taskTracker:    NewTaskTracker(),
			}

			// Process the event
			h.processEvent(event)

			// Check event sent
			if len(mockCentral.events) != 1 {
				t.Errorf("Expected 1 event sent to central handler, got %d", len(mockCentral.events))
				return
			}

			gotEvent := mockCentral.events[0]

			// Compare using cmp.Diff
			opts := []cmp.Option{
				cmpopts.IgnoreFields(internalevent.AssistantMessage{}, "Narration"),
			}

			if diff := cmp.Diff(tt.wantEvent, gotEvent, opts...); diff != "" {
				t.Errorf("AssistantMessage mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestIntegration_JSON tests JSON parsing
func TestIntegration_JSON(t *testing.T) {
	// Test that the parser can handle various JSON formats
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{
			name:      "valid_json",
			input:     `{"type":"system","timestamp":"2025-01-26T15:30:45Z","uuid":"123","sessionID":"test","cwd":"/test","content":"test","isMeta":false}`,
			wantError: false,
		},
		{
			name:      "invalid_json",
			input:     `{"type":}`,
			wantError: true,
		},
		{
			name:      "empty_string",
			input:     ``,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			_, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantError {
				t.Errorf("Parse() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// TestParseJSON tests the parseJSON helper function
func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[string]json.RawMessage
		wantErr bool
	}{
		{
			name:  "valid_json",
			input: `{"type":"test","value":123}`,
			want: map[string]json.RawMessage{
				"type":  json.RawMessage(`"test"`),
				"value": json.RawMessage(`123`),
			},
			wantErr: false,
		},
		{
			name:    "invalid_json",
			input:   `{"type":}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Errorf("parseJSON() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}