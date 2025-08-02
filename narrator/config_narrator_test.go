package narrator

import (
	"testing"
)

func TestConfigBasedNarrator_NarrateToolUse(t *testing.T) {
	// Load default configuration
	config := GetDefaultNarratorConfig()
	cn := NewConfigBasedNarrator(config)

	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		expected string
	}{
		// Bash tool tests
		{
			name:     "Bash with git commit",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "git commit -m 'test'"},
			expected: "変更をGitにコミットします",
		},
		{
			name:     "Bash with make test",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "make test"},
			expected: "テストを実行します",
		},
		{
			name:     "Bash with unknown command",
			toolName: "Bash",
			input:    map[string]interface{}{"command": "unknown-cmd"},
			expected: "コマンド「unknown-cmd」を実行します",
		},

		// Read tool tests with file extensions
		{
			name:     "Read Go file",
			toolName: "Read",
			input:    map[string]interface{}{"file_path": "main.go"},
			expected: "Goファイル「main.go」を読み込みます",
		},
		{
			name:     "Read JavaScript file",
			toolName: "Read",
			input:    map[string]interface{}{"file_path": "app.js"},
			expected: "JavaScriptファイル「app.js」を読み込みます",
		},
		{
			name:     "Read unknown extension",
			toolName: "Read",
			input:    map[string]interface{}{"file_path": "data.xyz"},
			expected: "ファイル「data.xyz」を読み込みます",
		},

		// Write tool tests
		{
			name:     "Write Python file",
			toolName: "Write",
			input:    map[string]interface{}{"file_path": "script.py"},
			expected: "Pythonファイル「script.py」を作成します",
		},
		{
			name:     "Write test file",
			toolName: "Write",
			input:    map[string]interface{}{"file_path": "test_something.go"},
			expected: "Goファイル「test_something.go」を作成します", // captures take precedence over patterns
		},

		// Edit tool tests
		{
			name:     "Edit TypeScript file",
			toolName: "Edit",
			input: map[string]interface{}{
				"file_path":  "component.ts",
				"old_string": "old",
				"new_string": "new",
			},
			expected: "TypeScriptファイル「component.ts」を編集します",
		},
		{
			name:     "Edit Go file",
			toolName: "Edit",
			input: map[string]interface{}{
				"file_path":  "main.go",
				"old_string": "func oldFunction",
				"new_string": "func newFunction",
			},
			expected: "Goファイル「main.go」を編集します",
		},

		// MultiEdit tool test
		{
			name:     "MultiEdit with count",
			toolName: "MultiEdit",
			input: map[string]interface{}{
				"file_path": "config.json",
				"edits":     []interface{}{1, 2, 3},
			},
			expected: "ファイル「config.json」に3箇所の変更を加えます",
		},

		// Grep tool tests
		{
			name:     "Grep with pattern",
			toolName: "Grep",
			input: map[string]interface{}{
				"pattern": "TODO",
				"path":    "/src",
			},
			expected: "/srcからTODOコメントを検索します",
		},
		{
			name:     "Grep with func pattern",
			toolName: "Grep",
			input: map[string]interface{}{
				"pattern": "func handleRequest",
				"path":    "/project",
			},
			expected: "/projectから関数定義を検索します",
		},
		{
			name:     "Grep with glob pattern (no path)",
			toolName: "Grep",
			input: map[string]interface{}{
				"pattern": "TODO",
				"glob":    "**/*.go",
			},
			expected: "プロジェクト全体からTODOコメントを検索します",
		},
		{
			name:     "Grep with glob pattern for specific file type",
			toolName: "Grep",
			input: map[string]interface{}{
				"pattern": "import",
				"glob":    "**/*.py",
			},
			expected: "プロジェクト全体から「import」を検索します",
		},

		// WebSearch tool test
		{
			name:     "WebSearch",
			toolName: "WebSearch",
			input:    map[string]interface{}{"query": "Go言語 エラーハンドリング"},
			expected: "「Go言語 エラーハンドリング」についてWeb検索します",
		},

		// Task tool tests
		{
			name:     "Task with description only",
			toolName: "Task",
			input:    map[string]interface{}{"description": "バグ修正"},
			expected: "タスク「バグ修正」を実行します",
		},
		{
			name:     "Task with subagent_type and description",
			toolName: "Task",
			input: map[string]interface{}{
				"description":   "データベース設計レビュー",
				"prompt":        "新しいユーザー管理システムのテーブル構造を確認してください",
				"subagent_type": "database-architect",
			},
			expected: "database-architect agentでタスク「データベース設計レビュー」を実行します",
		},
		{
			name:     "Task with empty subagent_type",
			toolName: "Task",
			input: map[string]interface{}{
				"description":   "ファイル検索",
				"subagent_type": "",
			},
			expected: "タスク「ファイル検索」を実行します",
		},

		// TodoWrite tool test
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
			expected: "TODOリストを更新します（完了: 2, 進行中: 1）",
		},

		// NotebookRead tool test
		{
			name:     "NotebookRead ipynb file",
			toolName: "NotebookRead",
			input:    map[string]interface{}{"notebook_path": "analysis.ipynb"},
			expected: "Jupyterノートブック「analysis.ipynb」を読み込みます",
		},

		// LS tool tests
		{
			name:     "LS with regular directory",
			toolName: "LS",
			input:    map[string]interface{}{"path": "/home/user/documents"},
			expected: "ディレクトリ「documents」の内容を確認します",
		},
		{
			name:     "LS with current directory",
			toolName: "LS",
			input:    map[string]interface{}{"path": "."},
			expected: "現在のディレクトリの内容を確認します",
		},

		// WebFetch tool tests
		{
			name:     "WebFetch from GitHub",
			toolName: "WebFetch",
			input:    map[string]interface{}{"url": "https://github.com/example/repo"},
			expected: "GitHubから情報を取得します",
		},
		{
			name:     "WebFetch from docs site",
			toolName: "WebFetch",
			input:    map[string]interface{}{"url": "https://docs.example.com/guide"},
			expected: "ドキュメントを参照します",
		},
		{
			name:     "WebFetch from unknown domain",
			toolName: "WebFetch",
			input:    map[string]interface{}{"url": "https://example.com/page"},
			expected: "「example.com」から情報を取得します",
		},

		// Glob tool test
		{
			name:     "Glob with test pattern",
			toolName: "Glob",
			input:    map[string]interface{}{"pattern": "*test*"},
			expected: "テストファイルを探します",
		},
		{
			name:     "Glob with Go files",
			toolName: "Glob",
			input:    map[string]interface{}{"pattern": "*.go"},
			expected: "Goファイルを探します",
		},
		{
			name:     "Glob with generic pattern",
			toolName: "Glob",
			input:    map[string]interface{}{"pattern": "src/**/*.txt"},
			expected: "パターン「src/**/*.txt」に一致するファイルを探します",
		},

		// MCP tools with new MCPRules structure
		{
			name:     "mcp__serena__read_memory with MCPRules",
			toolName: "mcp__serena__read_memory",
			input:    map[string]interface{}{"memory_file_name": "notes.md"},
			expected: "メモリファイル「notes.md」を読み込みます",
		},
		{
			name:     "mcp__serena__activate_project with MCPRules",
			toolName: "mcp__serena__activate_project",
			input:    map[string]interface{}{"project_name": "TestProject"},
			expected: "プロジェクト「TestProject」をアクティブ化します",
		},
		{
			name:     "mcp__serena__delete_lines with MCPRules",
			toolName: "mcp__serena__delete_lines",
			input: map[string]interface{}{
				"file_path":  "test.go",
				"start_line": 5.0,
				"end_line":   10.0,
			},
			expected: "「test.go」の5行目から10行目を削除します",
		},
		{
			name:     "mcp__serena__switch_modes with MCPRules",
			toolName: "mcp__serena__switch_modes",
			input: map[string]interface{}{
				"modes": []interface{}{"read", "write", "execute"},
			},
			expected: "モード「read, write, execute」に切り替えます",
		},
		{
			name:     "mcp__ide__getDiagnostics with MCPRules",
			toolName: "mcp__ide__getDiagnostics",
			input:    map[string]interface{}{},
			expected: "コードの診断情報を取得します",
		},
		{
			name:     "mcp__serena__unknown_operation (fallback to default)",
			toolName: "mcp__serena__unknown_operation",
			input:    map[string]interface{}{},
			expected: "Serenaツール「unknown_operation」を実行します",
		},
		// Unknown MCP tool tests
		{
			name:     "unknown MCP tool without server rules",
			toolName: "mcp__unknown_server__some_operation",
			input:    map[string]interface{}{},
			expected: "ツール「mcp__unknown_server__some_operation」を実行します",
		},
		{
			name:     "completely unknown tool",
			toolName: "UnknownTool",
			input:    map[string]interface{}{"param": "value"},
			expected: "ツール「UnknownTool」を実行します",
		},
		{
			name:     "unknown MCP tool with parameters",
			toolName: "mcp__newservice__process_data",
			input:    map[string]interface{}{"data": "test", "mode": "fast"},
			expected: "ツール「mcp__newservice__process_data」を実行します",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cn.NarrateToolUse(tt.toolName, tt.input)
			if result != tt.expected {
				t.Errorf("NarrateToolUse(%s, %v) = %q, want %q", tt.toolName, tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigBasedNarrator_UnknownToolsWithoutGenericMessage(t *testing.T) {
	// Create a config without genericToolExecution message
	// Also create an empty default config to override the global default
	config := &NarratorConfig{
		Rules:    make(map[string]ToolRules),
		Messages: MessageTemplates{}, // Empty messages
	}
	emptyDefaultConfig := &NarratorConfig{
		Rules:    make(map[string]ToolRules),
		Messages: MessageTemplates{}, // Empty messages
	}

	// Create ConfigBasedNarrator with custom configs
	cn := &ConfigBasedNarrator{
		config:        config,
		defaultConfig: emptyDefaultConfig,
	}

	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "unknown tool without generic message",
			toolName: "CompletelyNewTool",
			input:    map[string]interface{}{},
			expected: "CompletelyNewToolを実行中...",
		},
		{
			name:     "unknown MCP tool without generic message",
			toolName: "mcp__newserver__action",
			input:    map[string]interface{}{},
			expected: "mcp__newserver__actionを実行中...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cn.NarrateToolUse(tt.toolName, tt.input)
			if result != tt.expected {
				t.Errorf("NarrateToolUse(%s, %v) = %q, want %q", tt.toolName, tt.input, result, tt.expected)
			}
		})
	}
}

func TestHybridNarrator_UnknownTools(t *testing.T) {
	// Create HybridNarrator without AI
	hn := NewHybridNarrator("", false)

	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "unknown MCP tool",
			toolName: "mcp__unknown__operation",
			input:    map[string]interface{}{},
			expected: "mcp__unknown__operationを実行中...",
		},
		{
			name:     "completely unknown tool",
			toolName: "NewUnknownTool",
			input:    map[string]interface{}{"param": "value"},
			expected: "NewUnknownToolを実行中...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hn.NarrateToolUse(tt.toolName, tt.input)
			if result != tt.expected {
				t.Errorf("NarrateToolUse(%s, %v) = %q, want %q", tt.toolName, tt.input, result, tt.expected)
			}
		})
	}
}

func TestConfigBasedNarrator_FileTypeMapping(t *testing.T) {
	config := GetDefaultNarratorConfig()
	cn := NewConfigBasedNarrator(config)

	// Test file type name mapping
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "Goファイル"},
		{".js", "JavaScriptファイル"},
		{".ts", "TypeScriptファイル"},
		{".py", "Pythonファイル"},
		{".rb", "Rubyファイル"},
		{".java", "Javaファイル"},
		{".cpp", "C++ファイル"},
		{".rs", "Rustファイル"},
		{".md", "ドキュメント"},
		{".json", "JSON設定ファイル"},
		{".yaml", "YAML設定ファイル"},
		{".ipynb", "Jupyterノートブック"},
		{".unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			result := cn.getFileTypeName(tt.ext)
			if result != tt.expected {
				t.Errorf("getFileTypeName(%s) = %q, want %q", tt.ext, result, tt.expected)
			}
		})
	}
}

func TestConfigBasedNarrator_CaptureRules(t *testing.T) {
	// Create a test configuration with captures
	config := &NarratorConfig{
		Rules: map[string]ToolRules{
			"TestTool": {
				Default: "Processing {input1} and {input2}",
				Captures: []CaptureRule{
					{InputKey: "input1"},
					{InputKey: "input2"},
				},
			},
			"TestFileTypeTool": {
				Default: "{filetype}「{file_path}」を処理します",
				Captures: []CaptureRule{
					{InputKey: "file_path", ParseFileType: true},
				},
			},
		},
		Messages: MessageTemplates{
			GenericToolExecution: "ツール「{tool}」を実行します",
		},
		FileTypeNames: map[string]string{
			".go": "Goファイル",
			".js": "JavaScriptファイル",
		},
	}
	// Create narrator with test config but no default config
	cn := &ConfigBasedNarrator{
		config:        config,
		defaultConfig: &NarratorConfig{}, // Empty default config
	}

	tests := []struct {
		name     string
		toolName string
		input    map[string]interface{}
		expected string
	}{
		{
			name:     "Simple capture replacement",
			toolName: "TestTool",
			input: map[string]interface{}{
				"input1": "value1",
				"input2": "value2",
			},
			expected: "Processing value1 and value2",
		},
		{
			name:     "File type parsing with known extension",
			toolName: "TestFileTypeTool",
			input: map[string]interface{}{
				"file_path": "main.go",
			},
			expected: "Goファイル「main.go」を処理します",
		},
		{
			name:     "File type parsing with unknown extension",
			toolName: "TestFileTypeTool",
			input: map[string]interface{}{
				"file_path": "data.xyz",
			},
			expected: "ファイル「data.xyz」を処理します",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cn.NarrateToolUse(tt.toolName, tt.input)
			if result != tt.expected {
				t.Errorf("NarrateToolUse(%s, %v) = %q, want %q", tt.toolName, tt.input, result, tt.expected)
			}
		})
	}
}
