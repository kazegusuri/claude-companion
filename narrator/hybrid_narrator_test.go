package narrator

import (
	"testing"
)

// mockAINarrator is a mock implementation of AI narrator for testing
type mockAINarrator struct {
	fallback bool
}

func (m *mockAINarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	if !m.fallback {
		return "AIが処理中: " + toolName, false
	}
	return "", true
}

func (m *mockAINarrator) NarrateToolUsePermission(toolName string) (string, bool) {
	return "", true
}

func (m *mockAINarrator) NarrateText(text string, isThinking bool) (string, bool) {
	return text, false
}

func (m *mockAINarrator) NarrateNotification(notificationType NotificationType) (string, bool) {
	return "", false
}

func (m *mockAINarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	return "", false
}

func TestHybridNarrator_NarrateToolUse(t *testing.T) {
	// Define test cases that will be tested under different AI configurations
	testCases := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		// Expected results for different AI configurations
		expectedWithAI         string // AI responds with custom message
		expectedWithAIFallback string // AI returns fallback=true
		expectedWithoutAI      string // No AI at all
	}{
		// Known tools in config - should always use config rule
		{
			name:                   "Bash with git commit",
			toolName:               "Bash",
			input:                  map[string]interface{}{"command": "git commit -m 'test'"},
			expectedWithAI:         "変更をGitにコミットします",
			expectedWithAIFallback: "変更をGitにコミットします",
			expectedWithoutAI:      "変更をGitにコミットします",
		},
		{
			name:                   "Read Go file",
			toolName:               "Read",
			input:                  map[string]interface{}{"file_path": "main.go"},
			expectedWithAI:         "Goファイル「main.go」を読み込みます",
			expectedWithAIFallback: "Goファイル「main.go」を読み込みます",
			expectedWithoutAI:      "Goファイル「main.go」を読み込みます",
		},
		{
			name:                   "Write Python file",
			toolName:               "Write",
			input:                  map[string]interface{}{"file_path": "script.py"},
			expectedWithAI:         "Pythonファイル「script.py」を作成します",
			expectedWithAIFallback: "Pythonファイル「script.py」を作成します",
			expectedWithoutAI:      "Pythonファイル「script.py」を作成します",
		},
		{
			name:                   "WebSearch",
			toolName:               "WebSearch",
			input:                  map[string]interface{}{"query": "Go言語 エラーハンドリング"},
			expectedWithAI:         "「Go言語 エラーハンドリング」についてWeb検索します",
			expectedWithAIFallback: "「Go言語 エラーハンドリング」についてWeb検索します",
			expectedWithoutAI:      "「Go言語 エラーハンドリング」についてWeb検索します",
		},
		{
			name:     "TodoWrite with status counts",
			toolName: "TodoWrite",
			input: map[string]interface{}{
				"todos": []interface{}{
					map[string]interface{}{"status": "completed"},
					map[string]interface{}{"status": "completed"},
					map[string]interface{}{"status": "in_progress"},
					map[string]interface{}{"status": "pending"},
				},
			},
			expectedWithAI:         "TODOリストを更新します（完了: 2, 進行中: 1）",
			expectedWithAIFallback: "TODOリストを更新します（完了: 2, 進行中: 1）",
			expectedWithoutAI:      "TODOリストを更新します（完了: 2, 進行中: 1）",
		},
		// Unknown tool handled by AI when available (config returns generic message)
		{
			name:                   "AIHandledTool",
			toolName:               "AIHandledTool",
			input:                  map[string]interface{}{},
			expectedWithAI:         "ツール「AIHandledTool」を実行します", // Config returns generic message
			expectedWithAIFallback: "ツール「AIHandledTool」を実行します", // Config returns generic message
			expectedWithoutAI:      "ツール「AIHandledTool」を実行します", // Config returns generic message
		},
		// Unknown tools - config returns generic message
		{
			name:                   "completely unknown tool",
			toolName:               "CompletelyUnknownTool",
			input:                  map[string]interface{}{"param": "value"},
			expectedWithAI:         "ツール「CompletelyUnknownTool」を実行します", // Config returns generic message
			expectedWithAIFallback: "ツール「CompletelyUnknownTool」を実行します", // Config returns generic message
			expectedWithoutAI:      "ツール「CompletelyUnknownTool」を実行します", // Config returns generic message
		},
		{
			name:                   "unknown MCP tool",
			toolName:               "mcp__unknown__operation",
			input:                  map[string]interface{}{},
			expectedWithAI:         "AIが処理中: mcp__unknown__operation", // AI handles MCP tools too
			expectedWithAIFallback: "mcp__unknown__operationを実行中...",
			expectedWithoutAI:      "mcp__unknown__operationを実行中...",
		},
		// Known MCP tool in config - should use config rule
		{
			name:                   "mcp__ide__getDiagnostics",
			toolName:               "mcp__ide__getDiagnostics",
			input:                  map[string]interface{}{},
			expectedWithAI:         "コードの診断情報を取得します",
			expectedWithAIFallback: "コードの診断情報を取得します",
			expectedWithoutAI:      "コードの診断情報を取得します",
		},
	}

	// Test configuration 1: With AI that responds to all tools
	t.Run("WithAI", func(t *testing.T) {
		hn := NewHybridNarrator("dummy-api-key", true)
		mockAI := &mockAINarrator{
			fallback: false,
		}
		// Replace the AI narrator in the narrators slice
		if len(hn.narrators) > 1 {
			hn.narrators[1] = mockAI
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, shouldFallback := hn.NarrateToolUse(tc.toolName, tc.input)
				if result != tc.expectedWithAI {
					t.Errorf("NarrateToolUse(%s, %v) = %q, want %q",
						tc.toolName, tc.input, result, tc.expectedWithAI)
				}
				if shouldFallback {
					t.Errorf("NarrateToolUse(%s, %v) returned shouldFallback=true, but expected false",
						tc.toolName, tc.input)
				}
			})
		}
	})

	// Test configuration 2: With AI that always returns fallback=true
	t.Run("WithAIFallback", func(t *testing.T) {
		hn := NewHybridNarrator("dummy-api-key", true)
		mockAI := &mockAINarrator{
			fallback: true,
		}
		// Replace the AI narrator in the narrators slice
		if len(hn.narrators) > 1 {
			hn.narrators[1] = mockAI
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, shouldFallback := hn.NarrateToolUse(tc.toolName, tc.input)
				if result != tc.expectedWithAIFallback {
					t.Errorf("NarrateToolUse(%s, %v) = %q, want %q",
						tc.toolName, tc.input, result, tc.expectedWithAIFallback)
				}
				if shouldFallback {
					t.Errorf("NarrateToolUse(%s, %v) returned shouldFallback=true, but expected false",
						tc.toolName, tc.input)
				}
			})
		}
	})

	// Test configuration 3: Without AI
	t.Run("WithoutAI", func(t *testing.T) {
		hn := NewHybridNarrator("", false)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result, shouldFallback := hn.NarrateToolUse(tc.toolName, tc.input)
				if result != tc.expectedWithoutAI {
					t.Errorf("NarrateToolUse(%s, %v) = %q, want %q",
						tc.toolName, tc.input, result, tc.expectedWithoutAI)
				}
				if shouldFallback {
					t.Errorf("NarrateToolUse(%s, %v) returned shouldFallback=true, but expected false",
						tc.toolName, tc.input)
				}
			})
		}
	})
}
