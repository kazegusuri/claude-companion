package event

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// TestIntegration_Subagent tests parsing subagent JSONL files and verifying TaskCompletionEvent
func TestIntegration_Subagent(t *testing.T) {
	tests := []struct {
		name                    string
		filename                string
		handlerSessionID        string
		expectedEventTypes      []internalevent.Type
		expectedTaskCompletions []*internalevent.TaskCompletionMessage
		description             string
	}{
		{
			name:             "single_subagent",
			filename:         "testdata/subagent_single.jsonl",
			handlerSessionID: "a1e8bcd9-266a-4793-895d-d648f674981e",
			expectedEventTypes: []internalevent.Type{
				internalevent.EventTypeSummary,   // Line 1: Summary event
				internalevent.EventTypeSystem,    // Line 2: SessionStart:startup
				internalevent.EventTypeUser,      // Line 3: @agent-hello ドイツ語
				internalevent.EventTypeAssistant, // Line 4: First assistant message
				internalevent.EventTypeAssistant, // Line 5: Tool use (Task)
				internalevent.EventTypeSystem,    // Line 6: PreToolUse:Task
				// Line 7-8: Subagent events (IsSidechain=true) are buffered, not sent to central
				internalevent.EventTypeTaskComplete, // Generated from tool result
				internalevent.EventTypeUser,         // Line 9: Tool result
				internalevent.EventTypeAssistant,    // Line 10: Final assistant message
				internalevent.EventTypeSystem,       // Line 11: Stop notification
			},
			expectedTaskCompletions: []*internalevent.TaskCompletionMessage{
				{
					TaskInfo: internalevent.TaskInfo{
						ToolUseID:    "toolu_01Gk1dVWnqfbPW24DbPgeNNg",
						Description:  "Say hello in German",
						SubagentType: "hello",
					},
					SubagentTask: &internalevent.SubagentTask{
						UUID:       "64e2488a-6afa-4708-8942-17212a00081a", // First subagent event UUID
						EventCount: 2,                                      // 2 subagent events
					},
				},
			},
			description: "Single subagent execution with Task completion",
		},
		{
			name:             "multi_subagent",
			filename:         "testdata/subagent_multi.jsonl",
			handlerSessionID: "5e43694d-ba5d-490c-ad85-fdd5bd0f57b4",
			expectedEventTypes: []internalevent.Type{
				internalevent.EventTypeSystem,    // Line 1: SessionStart:startup
				internalevent.EventTypeUser,      // Line 2: @agent-hello イタリア語 @agent-hello 中国語
				internalevent.EventTypeAssistant, // Line 3: First assistant message
				internalevent.EventTypeAssistant, // Line 4: Tool use (Task) - Italian
				internalevent.EventTypeAssistant, // Line 5: Tool use (Task) - Chinese
				internalevent.EventTypeSystem,    // Line 6: PreToolUse:Task - Chinese
				internalevent.EventTypeSystem,    // Line 7: PreToolUse:Task - Italian
				// Lines 8-9, 11-12: Subagent events (IsSidechain=true) are buffered
				internalevent.EventTypeTaskComplete, // Generated from Italian task result
				internalevent.EventTypeUser,         // Line 10: Tool result for Italian
				internalevent.EventTypeTaskComplete, // Generated from Chinese task result
				internalevent.EventTypeUser,         // Line 13: Tool result for Chinese
				internalevent.EventTypeAssistant,    // Line 14: Final assistant message
				internalevent.EventTypeSystem,       // Line 15: Stop notification
			},
			expectedTaskCompletions: []*internalevent.TaskCompletionMessage{
				{
					TaskInfo: internalevent.TaskInfo{
						ToolUseID:    "toolu_013gkdLtsBuRyQhsV7x8qdbg",
						Description:  "Say hello in Italian",
						SubagentType: "hello",
					},
					SubagentTask: &internalevent.SubagentTask{
						UUID:       "9ae8348d-68f8-4e15-8f9f-d34510bda186",
						EventCount: 2,
					},
				},
				{
					TaskInfo: internalevent.TaskInfo{
						ToolUseID:    "toolu_01DFQeQpbdNmzS9LZNaEoW6j",
						Description:  "Say hello in Chinese",
						SubagentType: "hello",
					},
					SubagentTask: &internalevent.SubagentTask{
						UUID:       "f25546af-08c3-45cc-aba3-cd77f1203e95",
						EventCount: 2,
					},
				},
			},
			description: "Multiple parallel subagent executions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test components
			sessionManager := handler.NewSessionManager()
			centralHandler := &mockCentralHandler{}
			narratorInstance := &mockNarrator{}
			session := &SessionFile{
				SessionID:      tt.handlerSessionID,
				TranscriptPath: tt.filename,
			}

			// Create handler
			h := NewHandler(narratorInstance, sessionManager, centralHandler, session)

			// Create parser
			parser := NewParserWithPath(tt.filename)

			// Open test file
			file, err := os.Open(tt.filename)
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer file.Close()

			// Process each line
			scanner := bufio.NewScanner(file)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				if strings.TrimSpace(line) == "" {
					continue
				}

				event, err := parser.Parse(line)
				if err != nil {
					t.Errorf("Line %d: Failed to parse: %v", lineNum, err)
					continue
				}

				if event != nil {
					h.processEvent(event)
				}
			}

			if err := scanner.Err(); err != nil {
				t.Fatalf("Scanner error: %v", err)
			}

			// Verify event types
			actualEventTypes := []internalevent.Type{}
			for _, event := range centralHandler.events {
				actualEventTypes = append(actualEventTypes, event.Type())
			}

			if diff := cmp.Diff(tt.expectedEventTypes, actualEventTypes); diff != "" {
				t.Errorf("Event types mismatch (-want +got):\n%s", diff)
			}

			// Collect all TaskCompletionMessages for verification
			var actualTaskCompletions []*internalevent.TaskCompletionMessage
			for _, event := range centralHandler.events {
				if tc, ok := event.(*internalevent.TaskCompletionMessage); ok {
					actualTaskCompletions = append(actualTaskCompletions, tc)
				}
			}

			// Compare TaskCompletionMessages using cmp.Diff
			if tt.expectedTaskCompletions != nil {
				opts := []cmp.Option{
					cmpopts.IgnoreFields(internalevent.TaskCompletionMessage{}, "Session", "Timestamp", "Narration"),
				}
				if diff := cmp.Diff(tt.expectedTaskCompletions, actualTaskCompletions, opts...); diff != "" {
					t.Errorf("TaskCompletionMessages mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

// TestIntegration_SubagentNoMatch tests the case where subagent result doesn't match
func TestIntegration_SubagentNoMatch(t *testing.T) {
	// Create a minimal test case where the tool result doesn't match any subagent
	jsonlContent := `{"type":"user","message":{"role":"user","content":"test"},"uuid":"1","timestamp":"2024-01-01T00:00:00Z","parentUuid":null,"isSidechain":false,"sessionId":"test-session","cwd":"/test","version":"1.0.0"}
{"type":"assistant","message":{"id":"msg1","type":"message","role":"assistant","model":"claude","content":[{"type":"tool_use","id":"tool1","name":"Task","input":{"description":"Test task","subagent_type":"test"}}]},"uuid":"2","timestamp":"2024-01-01T00:00:01Z","parentUuid":"1","isSidechain":false,"sessionId":"test-session","cwd":"/test","version":"1.0.0"}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"tool1","content":"Different result"}]},"uuid":"3","timestamp":"2024-01-01T00:00:02Z","parentUuid":"2","isSidechain":false,"sessionId":"test-session","cwd":"/test","version":"1.0.0","toolUseResult":{"content":[{"type":"text","text":"Different result"}]}}`

	// Write test data to a temporary file
	tmpfile, err := os.CreateTemp("", "test_subagent_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(jsonlContent); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Create test components
	sessionManager := handler.NewSessionManager()
	centralHandler := &mockCentralHandler{}
	narratorInstance := &mockNarrator{}
	session := &SessionFile{
		SessionID:      "test-session",
		TranscriptPath: tmpfile.Name(),
	}

	// Create handler
	h := NewHandler(narratorInstance, sessionManager, centralHandler, session)

	// Create parser
	parser := NewParserWithPath(tmpfile.Name())

	// Open test file
	file, err := os.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer file.Close()

	// Process each line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		event, err := parser.Parse(line)
		if err != nil {
			continue
		}

		if event != nil {
			h.processEvent(event)
		}
	}

	// Find the TaskCompletionMessage
	var taskCompletion *internalevent.TaskCompletionMessage
	for _, event := range centralHandler.events {
		if tc, ok := event.(*internalevent.TaskCompletionMessage); ok {
			taskCompletion = tc
			break
		}
	}

	if taskCompletion == nil {
		t.Error("Expected TaskCompletionMessage not found")
	} else {
		// Should have no SubagentTask since there's no matching subagent
		if taskCompletion.SubagentTask != nil {
			t.Errorf("Expected SubagentTask to be nil, got %v", taskCompletion.SubagentTask)
		}
	}
}
