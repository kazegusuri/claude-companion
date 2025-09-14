package event

import (
	"testing"

	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

func TestIntegration_ToolTracking_AcceptFlow(t *testing.T) {
	// Test the accept flow: tool creation → approval → execution → completion
	parser := &Parser{}
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()

	sessionID := "3aca3565-b4b2-4b06-8096-e05e75e7eecc"
	sessionFile := &SessionFile{
		SessionID:      sessionID,
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}

	h := NewHandler(nil, sessionManager, centralHandler, sessionFile)

	// Line 1: SessionStart hook
	line1 := `{"parentUuid":null,"isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mSessionStart:startup\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T00:00:00.000Z","uuid":"b6fbcb5d-2ba1-4b48-82ca-23e7999c8ebb","toolUseID":"ce4e6802-c2c1-40f5-ad68-07e3c0ebfa65","level":"info"}`

	event1, err := parser.Parse(line1)
	if err != nil {
		t.Fatalf("Failed to parse line 1: %v", err)
	}
	// Parse as HookEvent since it contains hook information
	if hookEvent, ok := event1.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event1.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Line 2: User message requesting action
	line2 := `{"parentUuid":"b6fbcb5d-2ba1-4b48-82ca-23e7999c8ebb","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":"Create a test file"},"uuid":"2e2d8a81-139d-4850-ad35-c95039a2659c","timestamp":"2025-01-01T00:00:10.000Z"}`

	event2, err := parser.Parse(line2)
	if err != nil {
		t.Fatalf("Failed to parse line 2: %v", err)
	}
	event2.(*UserMessage).Session = sessionFile
	h.processEvent(event2)

	// Line 3: Assistant proposes tool use
	line3 := `{"parentUuid":"2e2d8a81-139d-4850-ad35-c95039a2659c","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_001","type":"message","role":"assistant","model":"claude-3","content":[{"type":"text","text":"I'll create a test file for you."}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_001","type":"assistant","uuid":"5f1bf9a2-02d5-4af6-8e59-06ef9c1e99d4","timestamp":"2025-01-01T00:00:15.000Z"}`

	event3, err := parser.Parse(line3)
	if err != nil {
		t.Fatalf("Failed to parse line 3: %v", err)
	}
	event3.(*AssistantMessage).Session = sessionFile
	h.processEvent(event3)

	// Line 4: Assistant with tool_use
	line4 := `{"parentUuid":"5f1bf9a2-02d5-4af6-8e59-06ef9c1e99d4","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_001","type":"message","role":"assistant","model":"claude-3","content":[{"type":"tool_use","id":"toolu_01TEPTEYgcdsUB6NirmKdh1i","name":"Bash","input":{"command":"touch test.txt","description":"Create test file"}}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_001","type":"assistant","uuid":"a992a97e-f53d-4773-87a3-2ff2a698eca0","timestamp":"2025-01-01T00:00:16.000Z"}`

	event4, err := parser.Parse(line4)
	if err != nil {
		t.Fatalf("Failed to parse line 4: %v", err)
	}
	event4.(*AssistantMessage).Session = sessionFile
	h.processEvent(event4)

	// Check tool status after creation
	activeTool, exists := h.toolTracker.GetActiveTool()
	if !exists {
		t.Fatal("Tool should be tracked after AssistantMessage with tool_use")
	}
	if activeTool.Status != ToolStatusCreated {
		t.Errorf("Tool status should be 'created', got '%s'", activeTool.Status)
	}
	if activeTool.ToolName != "Bash" {
		t.Errorf("Tool name should be 'Bash', got '%s'", activeTool.ToolName)
	}

	// Line 5: PreToolUse hook completed
	line5 := `{"parentUuid":"a992a97e-f53d-4773-87a3-2ff2a698eca0","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mPreToolUse:Bash\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T00:00:17.000Z","uuid":"6288e8fa-55ca-4a1b-948b-7f967651e62a","toolUseID":"toolu_01TEPTEYgcdsUB6NirmKdh1i","level":"info"}`

	event5, err := parser.Parse(line5)
	if err != nil {
		t.Fatalf("Failed to parse line 5: %v", err)
	}

	// Handle as HookEvent if parsed as such
	if hookEvent, ok := event5.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event5.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Check tool status after PreToolUse
	activeTool, _ = h.toolTracker.GetActiveTool()
	if activeTool.Status != ToolStatusWaitingApproval {
		t.Errorf("Tool status should be 'waiting_approval' after PreToolUse, got '%s'", activeTool.Status)
	}

	// Line 6: Tool result (approved)
	line6 := `{"parentUuid":"6288e8fa-55ca-4a1b-948b-7f967651e62a","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_01TEPTEYgcdsUB6NirmKdh1i","type":"tool_result","content":"File created successfully","is_error":false}]},"uuid":"1ded08fb-824f-4f2f-bb28-1298ad131233","timestamp":"2025-01-01T00:00:20.000Z","toolUseResult":{"stdout":"test.txt created","stderr":"","interrupted":false,"isImage":false}}`

	event6, err := parser.Parse(line6)
	if err != nil {
		t.Fatalf("Failed to parse line 6: %v", err)
	}
	event6.(*UserMessage).Session = sessionFile
	h.processEvent(event6)

	// Check tool status after approval
	activeTool, _ = h.toolTracker.GetActiveTool()
	if activeTool.Status != ToolStatusRunning {
		t.Errorf("Tool status should be 'running' after tool result, got '%s'", activeTool.Status)
	}

	// Line 7: PostToolUse Running
	line7 := `{"parentUuid":"1ded08fb-824f-4f2f-bb28-1298ad131233","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"Running \u001b[1mPostToolUse:Bash\u001b[22m...","isMeta":false,"timestamp":"2025-01-01T00:00:21.000Z","uuid":"d76ed931-7442-434a-a5e8-ee3260652e5a","toolUseID":"toolu_01TEPTEYgcdsUB6NirmKdh1i","level":"info"}`

	event7, err := parser.Parse(line7)
	if err != nil {
		t.Fatalf("Failed to parse line 7: %v", err)
	}

	// Handle as HookEvent if parsed as such
	if hookEvent, ok := event7.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event7.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Status should still be running
	activeTool, _ = h.toolTracker.GetActiveTool()
	if activeTool.Status != ToolStatusRunning {
		t.Errorf("Tool status should still be 'running', got '%s'", activeTool.Status)
	}

	// Line 8: PostToolUse Completed
	line8 := `{"parentUuid":"d76ed931-7442-434a-a5e8-ee3260652e5a","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"3aca3565-b4b2-4b06-8096-e05e75e7eecc","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mPostToolUse:Bash\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T00:00:22.000Z","uuid":"358e8b13-365c-4d5a-b592-6f73f51353c8","toolUseID":"toolu_01TEPTEYgcdsUB6NirmKdh1i","level":"info"}`

	event8, err := parser.Parse(line8)
	if err != nil {
		t.Fatalf("Failed to parse line 8: %v", err)
	}

	// Handle as HookEvent if parsed as such
	if hookEvent, ok := event8.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event8.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Check tool is finished
	tool, exists := h.toolTracker.GetTool("toolu_01TEPTEYgcdsUB6NirmKdh1i")
	if !exists {
		t.Fatal("Tool should still exist")
	}
	if tool.Status != ToolStatusFinished {
		t.Errorf("Tool status should be 'finished', got '%s'", tool.Status)
	}
	if tool.IsError {
		t.Error("Tool should not have error")
	}
	if tool.IsRejected {
		t.Error("Tool should not be rejected")
	}

	// No active tool after completion
	_, exists = h.toolTracker.GetActiveTool()
	if exists {
		t.Error("Should not have active tool after completion")
	}

	// Session should have no active tool
	if sessionFile.ActiveToolUse != nil {
		t.Error("Session should have no active tool after completion")
	}
}

func TestIntegration_ToolTracking_DenyFlow(t *testing.T) {
	// Test the deny flow: tool creation → rejection
	parser := &Parser{}
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()

	sessionID := "c21a6a9f-e970-4fa9-9460-7194bda2204e"
	sessionFile := &SessionFile{
		SessionID:      sessionID,
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}

	h := NewHandler(nil, sessionManager, centralHandler, sessionFile)

	// Line 1: SessionStart hook
	line1 := `{"parentUuid":null,"isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mSessionStart:startup\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T00:00:00.000Z","uuid":"03840b25-0d32-4916-832a-5b264133b504","toolUseID":"1d1e5411-c13d-4edb-9644-fa6581b2ff98","level":"info"}`

	event1, err := parser.Parse(line1)
	if err != nil {
		t.Fatalf("Failed to parse line 1: %v", err)
	}
	// Parse as HookEvent since it contains hook information
	if hookEvent, ok := event1.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event1.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Line 2: User message
	line2 := `{"parentUuid":"03840b25-0d32-4916-832a-5b264133b504","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":"Delete a sensitive file"},"uuid":"723820e6-2641-49a5-b3ea-a62339823f8a","timestamp":"2025-01-01T00:00:10.000Z"}`

	event2, err := parser.Parse(line2)
	if err != nil {
		t.Fatalf("Failed to parse line 2: %v", err)
	}
	event2.(*UserMessage).Session = sessionFile
	h.processEvent(event2)

	// Line 3: Assistant text response
	line3 := `{"parentUuid":"723820e6-2641-49a5-b3ea-a62339823f8a","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_002","type":"message","role":"assistant","model":"claude-3","content":[{"type":"text","text":"I'll help you delete that file."}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_002","type":"assistant","uuid":"a359c429-27f9-4c30-a08b-a89ea97f7055","timestamp":"2025-01-01T00:00:15.000Z"}`

	event3, err := parser.Parse(line3)
	if err != nil {
		t.Fatalf("Failed to parse line 3: %v", err)
	}
	event3.(*AssistantMessage).Session = sessionFile
	h.processEvent(event3)

	// Line 4: Assistant with tool_use
	line4 := `{"parentUuid":"a359c429-27f9-4c30-a08b-a89ea97f7055","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_002","type":"message","role":"assistant","model":"claude-3","content":[{"type":"tool_use","id":"toolu_01Ke4AsnaxRZDsA9KKf8DrCp","name":"Bash","input":{"command":"rm -f sensitive.txt","description":"Delete sensitive file"}}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_002","type":"assistant","uuid":"6766557b-2eb1-48cb-af6d-bb732d4dd661","timestamp":"2025-01-01T00:00:16.000Z"}`

	event4, err := parser.Parse(line4)
	if err != nil {
		t.Fatalf("Failed to parse line 4: %v", err)
	}
	event4.(*AssistantMessage).Session = sessionFile
	h.processEvent(event4)

	// Check tool status after creation
	activeTool, exists := h.toolTracker.GetActiveTool()
	if !exists {
		t.Fatal("Tool should be tracked after AssistantMessage with tool_use")
	}
	if activeTool.Status != ToolStatusCreated {
		t.Errorf("Tool status should be 'created', got '%s'", activeTool.Status)
	}

	// Line 5: PreToolUse hook completed
	line5 := `{"parentUuid":"6766557b-2eb1-48cb-af6d-bb732d4dd661","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mPreToolUse:Bash\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T00:00:17.000Z","uuid":"4f4f0d91-153b-4ddc-bfa2-31d5916328cc","toolUseID":"toolu_01Ke4AsnaxRZDsA9KKf8DrCp","level":"info"}`

	event5, err := parser.Parse(line5)
	if err != nil {
		t.Fatalf("Failed to parse line 5: %v", err)
	}

	// Handle as HookEvent if parsed as such
	if hookEvent, ok := event5.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event5.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Check tool status after PreToolUse
	activeTool, _ = h.toolTracker.GetActiveTool()
	if activeTool.Status != ToolStatusWaitingApproval {
		t.Errorf("Tool status should be 'waiting_approval' after PreToolUse, got '%s'", activeTool.Status)
	}

	// Line 6: Tool result (rejected)
	line6 := `{"parentUuid":"4f4f0d91-153b-4ddc-bfa2-31d5916328cc","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed.","is_error":true,"tool_use_id":"toolu_01Ke4AsnaxRZDsA9KKf8DrCp"}]},"uuid":"df3122cf-2784-4596-82f5-44abed6ec9cc","timestamp":"2025-01-01T00:00:20.000Z","toolUseResult":"Error: The user doesn't want to proceed with this tool use. The tool use was rejected (eg. if it was a file edit, the new_string was NOT written to the file). STOP what you are doing and wait for the user to tell you how to proceed."}`

	event6, err := parser.Parse(line6)
	if err != nil {
		t.Fatalf("Failed to parse line 6: %v", err)
	}
	event6.(*UserMessage).Session = sessionFile
	h.processEvent(event6)

	// Check tool is finished with rejection
	tool, exists := h.toolTracker.GetTool("toolu_01Ke4AsnaxRZDsA9KKf8DrCp")
	if !exists {
		t.Fatal("Tool should still exist")
	}
	if tool.Status != ToolStatusFinished {
		t.Errorf("Tool status should be 'finished' after rejection, got '%s'", tool.Status)
	}
	if !tool.IsError {
		t.Error("Tool should have error flag")
	}
	if !tool.IsRejected {
		t.Error("Tool should be marked as rejected")
	}

	// No active tool after rejection
	_, exists = h.toolTracker.GetActiveTool()
	if exists {
		t.Error("Should not have active tool after rejection")
	}

	// Session should have no active tool
	if sessionFile.ActiveToolUse != nil {
		t.Error("Session should have no active tool after rejection")
	}

	// Line 7: User interruption message (for completeness)
	line7 := `{"parentUuid":"df3122cf-2784-4596-82f5-44abed6ec9cc","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"c21a6a9f-e970-4fa9-9460-7194bda2204e","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]},"uuid":"92f1befa-5047-458e-a9da-baacbd94ac14","timestamp":"2025-01-01T00:00:21.000Z"}`

	event7, err := parser.Parse(line7)
	if err != nil {
		t.Fatalf("Failed to parse line 7: %v", err)
	}
	event7.(*UserMessage).Session = sessionFile
	h.processEvent(event7)

	// Tool should still be finished
	tool, _ = h.toolTracker.GetTool("toolu_01Ke4AsnaxRZDsA9KKf8DrCp")
	if tool.Status != ToolStatusFinished {
		t.Errorf("Tool status should remain 'finished', got '%s'", tool.Status)
	}
}

func TestIntegration_ToolTracking_MultipleTools(t *testing.T) {
	// Test tracking multiple tools in sequence
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()

	sessionFile := &SessionFile{
		SessionID:      "test-session",
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}

	h := NewHandler(nil, sessionManager, centralHandler, sessionFile)

	// First tool
	assistant1 := &AssistantMessage{
		BaseEvent: BaseEvent{
			UUID:      "assistant-1",
			SessionID: "test-session",
			Session:   sessionFile,
		},
		Message: AssistantMessageContent{
			Content: []AssistantContent{
				{
					Type: "tool_use",
					ID:   "tool-1",
					Name: "Edit",
				},
			},
		},
	}
	h.processEvent(assistant1)

	// Second tool
	assistant2 := &AssistantMessage{
		BaseEvent: BaseEvent{
			UUID:      "assistant-2",
			SessionID: "test-session",
			Session:   sessionFile,
		},
		Message: AssistantMessageContent{
			Content: []AssistantContent{
				{
					Type: "tool_use",
					ID:   "tool-2",
					Name: "Write",
				},
			},
		},
	}
	h.processEvent(assistant2)

	// Both tools should be tracked
	tool1, exists1 := h.toolTracker.GetTool("tool-1")
	tool2, exists2 := h.toolTracker.GetTool("tool-2")

	if !exists1 || !exists2 {
		t.Fatal("Both tools should be tracked")
	}

	if tool1.ToolName != "Edit" {
		t.Errorf("Tool 1 should be 'Edit', got '%s'", tool1.ToolName)
	}
	if tool2.ToolName != "Write" {
		t.Errorf("Tool 2 should be 'Write', got '%s'", tool2.ToolName)
	}

	// Complete first tool
	hook1 := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:      "hook-1",
			SessionID: "test-session",
			Session:   sessionFile,
		},
		HookEventType: "PostToolUse:Edit",
		HookStatus:    "finished",
		ToolUseID:     "tool-1",
	}
	h.processEvent(hook1)

	// First tool should be finished, second still active
	tool1, _ = h.toolTracker.GetTool("tool-1")
	if tool1.Status != ToolStatusFinished {
		t.Errorf("Tool 1 should be finished, got '%s'", tool1.Status)
	}

	activeTool, exists := h.toolTracker.GetActiveTool()
	if !exists {
		t.Fatal("Should have active tool (tool-2)")
	}
	if activeTool.ToolUseID != "tool-2" {
		t.Errorf("Active tool should be tool-2, got '%s'", activeTool.ToolUseID)
	}
}

func TestIntegration_ToolTracking_StateTransitionOrder(t *testing.T) {
	// Test that state transitions happen in correct order even if events arrive out of order
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()

	sessionFile := &SessionFile{
		SessionID:      "test-session",
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}

	h := NewHandler(nil, sessionManager, centralHandler, sessionFile)

	// Create tool
	assistant := &AssistantMessage{
		BaseEvent: BaseEvent{
			UUID:      "assistant-1",
			SessionID: "test-session",
			Session:   sessionFile,
		},
		Message: AssistantMessageContent{
			Content: []AssistantContent{
				{
					Type: "tool_use",
					ID:   "tool-1",
					Name: "Bash",
				},
			},
		},
	}
	h.processEvent(assistant)

	// Try to finish before running (should fail)
	if h.toolTracker.FinishTool("tool-1", false, false) {
		// This is actually valid - can go from Created to Finished
		tool, _ := h.toolTracker.GetTool("tool-1")
		if tool.Status != ToolStatusFinished {
			t.Errorf("Tool should be finished, got '%s'", tool.Status)
		}
	}

	// Create another tool for proper flow test
	assistant2 := &AssistantMessage{
		BaseEvent: BaseEvent{
			UUID:      "assistant-2",
			SessionID: "test-session",
			Session:   sessionFile,
		},
		Message: AssistantMessageContent{
			Content: []AssistantContent{
				{
					Type: "tool_use",
					ID:   "tool-2",
					Name: "Edit",
				},
			},
		},
	}
	h.processEvent(assistant2)

	// Move to WaitingApproval
	hook1 := &HookEvent{
		BaseEvent: BaseEvent{
			UUID:      "hook-1",
			SessionID: "test-session",
			Session:   sessionFile,
		},
		HookEventType: "PreToolUse:Edit",
		HookStatus:    "finished",
		ToolUseID:     "tool-2",
	}
	h.processEvent(hook1)

	tool, _ := h.toolTracker.GetTool("tool-2")
	if tool.Status != ToolStatusWaitingApproval {
		t.Errorf("Tool should be waiting_approval, got '%s'", tool.Status)
	}

	// Try to go back to Created (should fail)
	if h.toolTracker.UpdateToolStatus("tool-2", ToolStatusCreated) {
		t.Error("Should not allow going back to Created from WaitingApproval")
	}

	// Tool status should remain unchanged
	tool, _ = h.toolTracker.GetTool("tool-2")
	if tool.Status != ToolStatusWaitingApproval {
		t.Errorf("Tool should still be waiting_approval, got '%s'", tool.Status)
	}
}

func TestIntegration_BackgroundTaskFlow(t *testing.T) {
	// Test the complete background task flow: Bash(run_in_background) -> BashOutput -> KillShell
	parser := &Parser{}
	centralHandler := NewMockCentralHandler()
	sessionManager := handler.NewSessionManager()

	sessionID := "test-session-123"
	sessionFile := &SessionFile{
		SessionID:      sessionID,
		TranscriptPath: "/test/transcript.jsonl",
		Project:        "test-project",
	}

	h := NewHandler(nil, sessionManager, centralHandler, sessionFile)

	// Line 1: Assistant proposes Bash tool with run_in_background: true
	line1 := `{"parentUuid":"parent-uuid-001","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_bg_001","type":"message","role":"assistant","model":"claude-3","content":[{"type":"tool_use","id":"toolu_bg_001","name":"Bash","input":{"command":"npm run dev","description":"Start development server","run_in_background":true}}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_bg_001","type":"assistant","uuid":"assistant-uuid-001","timestamp":"2025-01-01T10:00:00.000Z"}`

	event1, err := parser.Parse(line1)
	if err != nil {
		t.Fatalf("Failed to parse line 1: %v", err)
	}
	event1.(*AssistantMessage).Session = sessionFile
	h.processEvent(event1)

	// Check tool status after creation
	activeTool, exists := h.toolTracker.GetActiveTool()
	if !exists {
		t.Fatal("Tool should be tracked after AssistantMessage with tool_use")
	}
	if activeTool.Status != ToolStatusCreated {
		t.Errorf("Tool status should be 'created', got '%s'", activeTool.Status)
	}
	if activeTool.ToolName != "Bash" {
		t.Errorf("Tool name should be 'Bash', got '%s'", activeTool.ToolName)
	}

	// Line 2: PreToolUse hook completed
	line2 := `{"parentUuid":"assistant-uuid-001","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mPreToolUse:Bash\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T10:00:01.000Z","uuid":"system-uuid-001","toolUseID":"toolu_bg_001","level":"info"}`

	event2, err := parser.Parse(line2)
	if err != nil {
		t.Fatalf("Failed to parse line 2: %v", err)
	}

	// Handle as HookEvent if parsed as such
	if hookEvent, ok := event2.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event2.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Check tool status after PreToolUse
	activeTool, _ = h.toolTracker.GetActiveTool()
	if activeTool.Status != ToolStatusWaitingApproval {
		t.Errorf("Tool status should be 'waiting_approval' after PreToolUse, got '%s'", activeTool.Status)
	}

	// Line 3: Tool result with backgroundTaskId
	line3 := `{"parentUuid":"system-uuid-001","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_bg_001","type":"tool_result","content":"Command running in background with ID: bg-task-123","is_error":false}]},"uuid":"user-uuid-001","timestamp":"2025-01-01T10:00:02.000Z","toolUseResult":{"stdout":"","stderr":"","interrupted":false,"isImage":false,"backgroundTaskId":"bg-task-123"}}`

	event3, err := parser.Parse(line3)
	if err != nil {
		t.Fatalf("Failed to parse line 3: %v", err)
	}
	event3.(*UserMessage).Session = sessionFile

	// Track background task when tool result is received
	userMsg := event3.(*UserMessage)
	if userMsg.ToolUseResult == nil {
		t.Fatal("ToolUseResult should not be nil")
	}

	// Parse ToolUseResult as a map
	resultMap, ok := userMsg.ToolUseResult.(map[string]interface{})
	if !ok {
		t.Fatal("ToolUseResult should be a map")
	}

	bgTaskID, ok := resultMap["backgroundTaskId"].(string)
	if !ok || bgTaskID == "" {
		t.Fatal("backgroundTaskId should exist and not be empty")
	}

	// Find the tool_use_id from the message content
	var toolUseID string
	contents, ok := userMsg.Message.Content.([]interface{})
	if !ok {
		t.Fatal("Message content should be an array")
	}

	for _, content := range contents {
		contentMap, ok := content.(map[string]interface{})
		if !ok {
			continue
		}
		if contentMap["type"] == "tool_result" {
			if id, ok := contentMap["tool_use_id"].(string); ok {
				toolUseID = id
				break
			}
		}
	}

	if toolUseID == "" {
		t.Fatal("tool_use_id not found in message content")
	}

	// Track the background task
	h.toolTracker.TrackBackgroundTask(
		bgTaskID,
		toolUseID,
		"npm run dev",
		"",
	)

	h.processEvent(event3)

	// Check tool status after approval
	activeTool, _ = h.toolTracker.GetActiveTool()
	if activeTool.Status != ToolStatusRunning {
		t.Errorf("Tool status should be 'running' after tool result, got '%s'", activeTool.Status)
	}

	// Check background task was tracked
	backgroundTask, exists := h.toolTracker.GetBackgroundTask("bg-task-123")
	if !exists {
		t.Fatal("Background task should be tracked")
	}
	if backgroundTask.BackgroundTaskID != "bg-task-123" {
		t.Errorf("Background task ID should be 'bg-task-123', got '%s'", backgroundTask.BackgroundTaskID)
	}
	if backgroundTask.IsTerminated {
		t.Error("Background task should not be terminated yet")
	}

	// Check active background tasks
	activeTasks := h.toolTracker.GetActiveBackgroundTasks()
	if len(activeTasks) != 1 {
		t.Errorf("Should have 1 active background task, got %d", len(activeTasks))
	}

	// Line 4: PostToolUse Running
	line4 := `{"parentUuid":"user-uuid-001","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"Running \u001b[1mPostToolUse:Bash\u001b[22m...","isMeta":false,"timestamp":"2025-01-01T10:00:03.000Z","uuid":"system-uuid-002","toolUseID":"toolu_bg_001","level":"info"}`

	event4, err := parser.Parse(line4)
	if err != nil {
		t.Fatalf("Failed to parse line 4: %v", err)
	}

	if hookEvent, ok := event4.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event4.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Line 5: PostToolUse Completed
	line5 := `{"parentUuid":"system-uuid-002","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","type":"system","subtype":"informational","content":"\u001b[1mPostToolUse:Bash\u001b[22m [/test/bin/hook.sh] completed successfully","isMeta":false,"timestamp":"2025-01-01T10:00:04.000Z","uuid":"system-uuid-003","toolUseID":"toolu_bg_001","level":"info"}`

	event5, err := parser.Parse(line5)
	if err != nil {
		t.Fatalf("Failed to parse line 5: %v", err)
	}

	if hookEvent, ok := event5.(*HookEvent); ok {
		hookEvent.Session = sessionFile
		h.processEvent(hookEvent)
	} else if sysMsg, ok := event5.(*SystemMessage); ok {
		sysMsg.Session = sessionFile
		h.processEvent(sysMsg)
	}

	// Check tool is finished
	tool, exists := h.toolTracker.GetTool("toolu_bg_001")
	if !exists {
		t.Fatal("Tool should still exist")
	}
	if tool.Status != ToolStatusFinished {
		t.Errorf("Tool status should be 'finished', got '%s'", tool.Status)
	}

	// Background task should still be active after tool finishes
	backgroundTask, _ = h.toolTracker.GetBackgroundTask("bg-task-123")
	if backgroundTask.IsTerminated {
		t.Error("Background task should still be active after tool finishes")
	}

	// Line 6: BashOutput checking status
	line6 := `{"parentUuid":"system-uuid-003","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_bg_002","type":"message","role":"assistant","model":"claude-3","content":[{"type":"tool_use","id":"toolu_output_001","name":"BashOutput","input":{"bash_id":"bg-task-123"}}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_bg_002","type":"assistant","uuid":"assistant-uuid-002","timestamp":"2025-01-01T10:00:10.000Z"}`

	event6, err := parser.Parse(line6)
	if err != nil {
		t.Fatalf("Failed to parse line 6: %v", err)
	}
	event6.(*AssistantMessage).Session = sessionFile
	h.processEvent(event6)

	// Line 7: BashOutput result
	line7 := `{"parentUuid":"assistant-uuid-002","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_output_001","type":"tool_result","content":"<status>running</status>\n\n<stdout>\n  Server started on port 3000\n  Ready for connections\n</stdout>\n\n<stderr>\n$ npm run dev\n</stderr>\n\n<timestamp>2025-01-01T10:00:11.000Z</timestamp>"}]},"uuid":"user-uuid-002","timestamp":"2025-01-01T10:00:11.000Z","toolUseResult":{"shellId":"bg-task-123","command":"npm run dev","status":"running","exitCode":null,"stdout":"  Server started on port 3000\n  Ready for connections","stderr":"$ npm run dev","stdoutLines":2,"stderrLines":1,"timestamp":"2025-01-01T10:00:11.000Z"}}`

	event7, err := parser.Parse(line7)
	if err != nil {
		t.Fatalf("Failed to parse line 7: %v", err)
	}
	event7.(*UserMessage).Session = sessionFile
	h.processEvent(event7)

	// Background task should still be active after BashOutput
	backgroundTask, _ = h.toolTracker.GetBackgroundTask("bg-task-123")
	if backgroundTask.IsTerminated {
		t.Error("Background task should still be active after BashOutput")
	}

	// Line 8: KillShell to terminate the background task
	line8 := `{"parentUuid":"user-uuid-002","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","message":{"id":"msg_test_bg_003","type":"message","role":"assistant","model":"claude-3","content":[{"type":"tool_use","id":"toolu_kill_001","name":"KillShell","input":{"shell_id":"bg-task-123"}}],"stop_reason":null,"stop_sequence":null},"requestId":"req_test_bg_003","type":"assistant","uuid":"assistant-uuid-003","timestamp":"2025-01-01T10:00:20.000Z"}`

	event8, err := parser.Parse(line8)
	if err != nil {
		t.Fatalf("Failed to parse line 8: %v", err)
	}
	event8.(*AssistantMessage).Session = sessionFile
	h.processEvent(event8)

	// Line 9: KillShell result
	line9 := `{"parentUuid":"assistant-uuid-003","isSidechain":false,"userType":"external","cwd":"/test/workspace","sessionId":"test-session-123","version":"1.0.0","gitBranch":"main","type":"user","message":{"role":"user","content":[{"tool_use_id":"toolu_kill_001","type":"tool_result","content":"{\"message\":\"Successfully killed shell: bg-task-123 (npm run dev)\",\"shell_id\":\"bg-task-123\"}"}]},"uuid":"user-uuid-003","timestamp":"2025-01-01T10:00:21.000Z","toolUseResult":{"message":"Successfully killed shell: bg-task-123 (npm run dev)","shell_id":"bg-task-123"}}`

	event9, err := parser.Parse(line9)
	if err != nil {
		t.Fatalf("Failed to parse line 9: %v", err)
	}
	event9.(*UserMessage).Session = sessionFile

	// Terminate the background task when KillShell result is received
	killMsg := event9.(*UserMessage)
	if killMsg.ToolUseResult == nil {
		t.Fatal("KillShell ToolUseResult should not be nil")
	}

	// Parse the result to get shell_id
	killResultMap, ok := killMsg.ToolUseResult.(map[string]interface{})
	if !ok {
		t.Fatal("KillShell ToolUseResult should be a map")
	}

	shellID, ok := killResultMap["shell_id"].(string)
	if !ok || shellID == "" {
		t.Fatal("shell_id should exist and not be empty in KillShell result")
	}

	// Terminate the background task
	if !h.toolTracker.TerminateBackgroundTask(shellID) {
		t.Fatal("Failed to terminate background task")
	}

	h.processEvent(event9)

	// Check background task is now terminated
	backgroundTask, exists = h.toolTracker.GetBackgroundTask("bg-task-123")
	if !exists {
		t.Fatal("Background task should still exist after termination")
	}
	if !backgroundTask.IsTerminated {
		t.Error("Background task should be terminated after KillShell")
	}
	if backgroundTask.TerminatedAt == nil {
		t.Error("Background task should have TerminatedAt timestamp")
	}

	// Check no active background tasks remain
	activeTasks = h.toolTracker.GetActiveBackgroundTasks()
	if len(activeTasks) != 0 {
		t.Errorf("Should have 0 active background tasks after termination, got %d", len(activeTasks))
	}

	// Check all background tasks (including terminated)
	allTasks := h.toolTracker.GetAllBackgroundTasks()
	if len(allTasks) != 1 {
		t.Errorf("Should have 1 total background task, got %d", len(allTasks))
	}
}
