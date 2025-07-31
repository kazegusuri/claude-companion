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

		// WebSearch tool test
		{
			name:     "WebSearch",
			toolName: "WebSearch",
			input:    map[string]interface{}{"query": "Go言語 エラーハンドリング"},
			expected: "「Go言語 エラーハンドリング」についてWeb検索します",
		},

		// Task tool test
		{
			name:     "Task",
			toolName: "Task",
			input:    map[string]interface{}{"description": "バグ修正"},
			expected: "タスク「バグ修正」を実行します",
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

		// MCP tool tests with captures
		{
			name:     "mcp__serena__read_memory",
			toolName: "mcp__serena__read_memory",
			input:    map[string]interface{}{"memory_file_name": "project_notes.md"},
			expected: "メモリファイル「project_notes.md」を読み込みます",
		},
		{
			name:     "mcp__serena__activate_project",
			toolName: "mcp__serena__activate_project",
			input:    map[string]interface{}{"project_name": "MyProject"},
			expected: "プロジェクト「MyProject」をアクティブ化します",
		},
		{
			name:     "mcp__serena__find_symbol",
			toolName: "mcp__serena__find_symbol",
			input:    map[string]interface{}{"name_path": "handleRequest"},
			expected: "シンボル「handleRequest」を検索します",
		},
		{
			name:     "mcp__serena__delete_lines",
			toolName: "mcp__serena__delete_lines",
			input: map[string]interface{}{
				"file_path":  "main.go",
				"start_line": 10.0,
				"end_line":   20.0,
			},
			expected: "「main.go」の10行目から20行目を削除します",
		},
		{
			name:     "mcp__serena__read_file with Go file",
			toolName: "mcp__serena__read_file",
			input:    map[string]interface{}{"file_path": "/src/main.go"},
			expected: "Goファイル「/src/main.go」を読み込みます",
		},
		{
			name:     "mcp__serena__read_file with unknown extension",
			toolName: "mcp__serena__read_file",
			input:    map[string]interface{}{"file_path": "data.xyz"},
			expected: "ファイル「data.xyz」を読み込みます",
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

		// MCP tools with hardcoded logic removed
		{
			name:     "mcp__serena__execute_shell_command with test command",
			toolName: "mcp__serena__execute_shell_command",
			input:    map[string]interface{}{"command": "test -f file.txt"},
			expected: "シェルコマンドを実行します",
		},
		{
			name:     "mcp__serena__execute_shell_command with build command",
			toolName: "mcp__serena__execute_shell_command",
			input:    map[string]interface{}{"command": "go build"},
			expected: "シェルコマンドを実行します",
		},
		{
			name:     "mcp__serena__switch_modes with multiple modes",
			toolName: "mcp__serena__switch_modes",
			input: map[string]interface{}{
				"modes": []interface{}{"read", "write", "execute"},
			},
			expected: "モード「read, write, execute」に切り替えます",
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
