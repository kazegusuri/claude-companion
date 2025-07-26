package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ConfigBasedNarrator uses configuration file for narrative rules
type ConfigBasedNarrator struct {
	config *NarratorConfig
}

// NewConfigBasedNarrator creates a new config-based narrator
func NewConfigBasedNarrator(config *NarratorConfig) *ConfigBasedNarrator {
	return &ConfigBasedNarrator{
		config: config,
	}
}

// NarrateToolUse converts tool usage to natural Japanese using config rules
func (cn *ConfigBasedNarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	rules, ok := cn.config.Rules[toolName]
	if !ok {
		// No rules for this tool
		return fmt.Sprintf("ツール「%s」を実行します", toolName)
	}

	// Special handling for Bash commands
	if toolName == "Bash" {
		if cmd, ok := input["command"].(string); ok {
			// Check prefixes
			for _, prefix := range rules.Prefixes {
				if strings.HasPrefix(cmd, prefix.Prefix) {
					return prefix.Message
				}
			}

			// Use default if no prefix matches
			if rules.Default != "" {
				// Extract first word as command name
				cmdParts := strings.Fields(cmd)
				if len(cmdParts) > 0 {
					return strings.ReplaceAll(rules.Default, "{command}", cmdParts[0])
				}
			}
		}
		return ""
	}

	// Handle file-based tools
	if toolName == "Read" || toolName == "Write" || toolName == "Edit" ||
		toolName == "NotebookRead" || toolName == "NotebookEdit" {
		var filePath string
		if path, ok := input["file_path"].(string); ok {
			filePath = path
		} else if path, ok := input["notebook_path"].(string); ok {
			filePath = path
		}

		if filePath != "" {
			fileName := filepath.Base(filePath)
			ext := filepath.Ext(fileName)

			// Check extensions
			if message, ok := rules.Extensions[ext]; ok {
				return strings.ReplaceAll(message, "{filename}", fileName)
			}

			// Check patterns
			for _, pattern := range rules.Patterns {
				if toolName == "Edit" || toolName == "NotebookEdit" {
					// For edit tools, check patterns in the input content
					if oldStr, ok := input["old_string"].(string); ok {
						if strings.Contains(oldStr, pattern.Contains) {
							msg := strings.ReplaceAll(pattern.Message, "{filename}", fileName)
							return msg
						}
					}
					if newStr, ok := input["new_string"].(string); ok {
						if strings.Contains(newStr, pattern.Contains) {
							msg := strings.ReplaceAll(pattern.Message, "{filename}", fileName)
							return msg
						}
					}
					if mode, ok := input["edit_mode"].(string); ok {
						if strings.Contains(mode, pattern.Contains) {
							msg := strings.ReplaceAll(pattern.Message, "{filename}", fileName)
							return msg
						}
					}
				} else if toolName == "Write" {
					// For write, check patterns in the file path
					if strings.Contains(filePath, pattern.Contains) {
						msg := strings.ReplaceAll(pattern.Message, "{filename}", fileName)
						return msg
					}
				}
			}

			// Use default
			if rules.Default != "" {
				return strings.ReplaceAll(rules.Default, "{filename}", fileName)
			}
		}
	}

	// Handle MultiEdit
	if toolName == "MultiEdit" {
		if path, ok := input["file_path"].(string); ok {
			fileName := filepath.Base(path)
			if edits, ok := input["edits"].([]interface{}); ok {
				count := len(edits)
				msg := strings.ReplaceAll(rules.Default, "{filename}", fileName)
				msg = strings.ReplaceAll(msg, "{count}", fmt.Sprintf("%d", count))
				return msg
			}
			return strings.ReplaceAll(rules.Default, "{filename}", fileName)
		}
	}

	// Handle Grep
	if toolName == "Grep" {
		if pattern, ok := input["pattern"].(string); ok {
			path, _ := input["path"].(string)
			if path == "" {
				path = "プロジェクト全体"
			} else {
				path = fmt.Sprintf("「%s」", path)
			}

			// Check patterns
			for _, rule := range rules.Patterns {
				if strings.Contains(pattern, rule.Contains) {
					msg := strings.ReplaceAll(rule.Message, "{path}", path)
					msg = strings.ReplaceAll(msg, "{pattern}", pattern)
					return msg
				}
			}

			// Use default
			msg := strings.ReplaceAll(rules.Default, "{path}", path)
			msg = strings.ReplaceAll(msg, "{pattern}", pattern)
			return msg
		}
	}

	// Handle Glob
	if toolName == "Glob" {
		if pattern, ok := input["pattern"].(string); ok {
			// Check patterns
			for _, rule := range rules.Patterns {
				if strings.Contains(pattern, rule.Contains) {
					return rule.Message
				}
			}

			// Use default
			return strings.ReplaceAll(rules.Default, "{pattern}", pattern)
		}
	}

	// Handle LS
	if toolName == "LS" {
		if path, ok := input["path"].(string); ok {
			dirName := filepath.Base(path)
			if dirName == "." || dirName == "/" {
				return "現在のディレクトリの内容を確認します"
			}
			return strings.ReplaceAll(rules.Default, "{dirname}", dirName)
		}
		return "ディレクトリの内容を確認します"
	}

	// Handle WebFetch
	if toolName == "WebFetch" {
		if url, ok := input["url"].(string); ok {
			// Check patterns
			for _, rule := range rules.Patterns {
				if strings.Contains(url, rule.Contains) {
					return rule.Message
				}
			}

			// Use default
			domain := extractDomain(url)
			return strings.ReplaceAll(rules.Default, "{domain}", domain)
		}
	}

	// Handle WebSearch
	if toolName == "WebSearch" {
		if query, ok := input["query"].(string); ok {
			return strings.ReplaceAll(rules.Default, "{query}", query)
		}
	}

	// Handle Task
	if toolName == "Task" {
		if desc, ok := input["description"].(string); ok {
			return strings.ReplaceAll(rules.Default, "{description}", desc)
		}
		if prompt, ok := input["prompt"].(string); ok {
			if strings.HasPrefix(prompt, "/") {
				// Slash command
				cmd := strings.Fields(prompt)[0]
				return fmt.Sprintf("コマンド「%s」を実行します", cmd)
			}
		}
		return "複雑なタスクを処理します"
	}

	// Handle TodoWrite
	if toolName == "TodoWrite" {
		if todos, ok := input["todos"].([]interface{}); ok {
			completed := 0
			inProgress := 0
			for _, todo := range todos {
				if todoMap, ok := todo.(map[string]interface{}); ok {
					if status, ok := todoMap["status"].(string); ok {
						switch status {
						case "completed":
							completed++
						case "in_progress":
							inProgress++
						}
					}
				}
			}
			msg := strings.ReplaceAll(rules.Default, "{completed}", fmt.Sprintf("%d", completed))
			msg = strings.ReplaceAll(msg, "{in_progress}", fmt.Sprintf("%d", inProgress))
			return msg
		}
		return "TODOリストを更新します"
	}

	// Handle tools with simple default messages
	if rules.Default != "" {
		return rules.Default
	}

	// Generic fallback
	return fmt.Sprintf("ツール「%s」を実行します", toolName)
}

// NarrateCodeBlock describes a code block
func (cn *ConfigBasedNarrator) NarrateCodeBlock(language, content string) string {
	// This is not configurable, so use the same logic as before
	lines := strings.Split(strings.TrimSpace(content), "\n")
	lineCount := len(lines)

	switch language {
	case "go":
		if strings.Contains(content, "func main()") {
			return "メイン関数を定義します"
		}
		if strings.Contains(content, "func Test") {
			return "テスト関数を定義します"
		}
		if strings.Contains(content, "type") && strings.Contains(content, "struct") {
			return "構造体を定義します"
		}
		if strings.Contains(content, "type") && strings.Contains(content, "interface") {
			return "インターフェースを定義します"
		}
		return fmt.Sprintf("Goコード（%d行）を記述します", lineCount)

	case "python", "py":
		if strings.Contains(content, "def ") {
			return "Python関数を定義します"
		}
		if strings.Contains(content, "class ") {
			return "Pythonクラスを定義します"
		}
		return fmt.Sprintf("Pythonコード（%d行）を記述します", lineCount)

	case "javascript", "js", "typescript", "ts":
		if strings.Contains(content, "function") || strings.Contains(content, "const") && strings.Contains(content, "=>") {
			return "JavaScript関数を定義します"
		}
		if strings.Contains(content, "class ") {
			return "JavaScriptクラスを定義します"
		}
		if strings.Contains(content, "import") || strings.Contains(content, "export") {
			return "モジュールの設定を行います"
		}
		return fmt.Sprintf("JavaScriptコード（%d行）を記述します", lineCount)

	case "bash", "sh", "shell":
		return "シェルスクリプトを記述します"

	case "json":
		return "JSON設定を記述します"

	case "yaml", "yml":
		return "YAML設定を記述します"

	case "markdown", "md":
		return "ドキュメントを記述します"

	case "sql":
		if strings.Contains(strings.ToUpper(content), "CREATE TABLE") {
			return "テーブルを定義します"
		}
		if strings.Contains(strings.ToUpper(content), "SELECT") {
			return "データを検索します"
		}
		return "SQLクエリを記述します"

	default:
		if lineCount == 1 {
			return "1行のコードを記述します"
		}
		return fmt.Sprintf("%d行のコードを記述します", lineCount)
	}
}

// NarrateFileOperation describes file operations
func (cn *ConfigBasedNarrator) NarrateFileOperation(operation, filePath string) string {
	fileName := filepath.Base(filePath)

	switch operation {
	case "Read":
		return fmt.Sprintf("「%s」を読み込みました", fileName)
	case "Write":
		return fmt.Sprintf("「%s」を作成しました", fileName)
	case "Edit":
		return fmt.Sprintf("「%s」を編集しました", fileName)
	case "Delete":
		return fmt.Sprintf("「%s」を削除しました", fileName)
	default:
		return fmt.Sprintf("「%s」に対して%s操作を行いました", fileName, operation)
	}
}
