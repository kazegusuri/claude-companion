package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Narrator interface for converting tool actions to natural language
type Narrator interface {
	NarrateToolUse(toolName string, input map[string]interface{}) string
	NarrateCodeBlock(language, content string) string
	NarrateFileOperation(operation, filePath string) string
}

// HybridNarrator uses rules first, then falls back to AI
type HybridNarrator struct {
	rules     map[string]func(map[string]interface{}) string
	ai        *OpenAINarrator
	useAI     bool
	cache     map[string]string
	cacheMu   sync.RWMutex
	cacheTime map[string]time.Time
	cacheTTL  time.Duration
}

// NewHybridNarrator creates a new hybrid narrator
func NewHybridNarrator(apiKey string, useAI bool) *HybridNarrator {
	hn := &HybridNarrator{
		useAI:     useAI,
		cache:     make(map[string]string),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  30 * time.Minute,
	}

	if useAI && apiKey != "" {
		hn.ai = NewOpenAINarrator(apiKey)
	}

	// Initialize rule-based narrations
	hn.rules = map[string]func(map[string]interface{}) string{
		"Bash": func(input map[string]interface{}) string {
			if cmd, ok := input["command"].(string); ok {
				// Git commands
				if strings.HasPrefix(cmd, "git commit") {
					return "変更をGitにコミットします"
				}
				if strings.HasPrefix(cmd, "git push") {
					return "変更をリモートリポジトリにプッシュします"
				}
				if strings.HasPrefix(cmd, "git add") {
					return "ファイルをGitのステージングエリアに追加します"
				}
				if strings.HasPrefix(cmd, "git status") {
					return "Gitリポジトリの状態を確認します"
				}
				if strings.HasPrefix(cmd, "git diff") {
					return "変更内容の差分を確認します"
				}
				if strings.HasPrefix(cmd, "git log") {
					return "コミット履歴を確認します"
				}

				// Make commands
				if strings.HasPrefix(cmd, "make test") {
					return "テストを実行します"
				}
				if strings.HasPrefix(cmd, "make build") {
					return "プロジェクトをビルドします"
				}
				if strings.HasPrefix(cmd, "make fmt") {
					return "コードをフォーマットします"
				}
				if strings.HasPrefix(cmd, "make") {
					target := strings.TrimPrefix(cmd, "make ")
					return fmt.Sprintf("「%s」タスクを実行します", target)
				}

				// Go commands
				if strings.HasPrefix(cmd, "go test") {
					return "Goのテストを実行します"
				}
				if strings.HasPrefix(cmd, "go build") {
					return "Goプログラムをビルドします"
				}
				if strings.HasPrefix(cmd, "go run") {
					return "Goプログラムを実行します"
				}
				if strings.HasPrefix(cmd, "gofmt") || strings.HasPrefix(cmd, "go fmt") {
					return "Goコードをフォーマットします"
				}

				// npm/yarn commands
				if strings.HasPrefix(cmd, "npm install") || strings.HasPrefix(cmd, "yarn install") {
					return "依存パッケージをインストールします"
				}
				if strings.HasPrefix(cmd, "npm run") || strings.HasPrefix(cmd, "yarn run") {
					return "スクリプトを実行します"
				}
				if strings.HasPrefix(cmd, "npm test") || strings.HasPrefix(cmd, "yarn test") {
					return "テストを実行します"
				}

				// Other common commands
				if strings.HasPrefix(cmd, "mkdir") {
					return "ディレクトリを作成します"
				}
				if strings.HasPrefix(cmd, "rm") {
					return "ファイルまたはディレクトリを削除します"
				}
				if strings.HasPrefix(cmd, "cp") {
					return "ファイルをコピーします"
				}
				if strings.HasPrefix(cmd, "mv") {
					return "ファイルを移動します"
				}
				if strings.HasPrefix(cmd, "ls") {
					return "ディレクトリの内容を確認します"
				}
				if strings.HasPrefix(cmd, "cat") {
					return "ファイルの内容を表示します"
				}
				if strings.HasPrefix(cmd, "grep") || strings.HasPrefix(cmd, "rg") {
					return "ファイル内を検索します"
				}
				if strings.HasPrefix(cmd, "find") {
					return "ファイルを検索します"
				}

				// Generic command execution
				cmdParts := strings.Fields(cmd)
				if len(cmdParts) > 0 {
					return fmt.Sprintf("コマンド「%s」を実行します", cmdParts[0])
				}
			}
			return ""
		},

		"Read": func(input map[string]interface{}) string {
			if path, ok := input["file_path"].(string); ok {
				fileName := filepath.Base(path)
				ext := filepath.Ext(fileName)

				// File type specific messages
				switch ext {
				case ".go":
					return fmt.Sprintf("Goファイル「%s」を読み込みます", fileName)
				case ".js", ".ts", ".jsx", ".tsx":
					return fmt.Sprintf("JavaScriptファイル「%s」を読み込みます", fileName)
				case ".py":
					return fmt.Sprintf("Pythonファイル「%s」を読み込みます", fileName)
				case ".md":
					return fmt.Sprintf("ドキュメント「%s」を読み込みます", fileName)
				case ".json":
					return fmt.Sprintf("JSON設定ファイル「%s」を読み込みます", fileName)
				case ".yaml", ".yml":
					return fmt.Sprintf("YAML設定ファイル「%s」を読み込みます", fileName)
				case ".txt":
					return fmt.Sprintf("テキストファイル「%s」を読み込みます", fileName)
				case ".log":
					return fmt.Sprintf("ログファイル「%s」を読み込みます", fileName)
				default:
					return fmt.Sprintf("ファイル「%s」を読み込みます", fileName)
				}
			}
			return ""
		},

		"Write": func(input map[string]interface{}) string {
			if path, ok := input["file_path"].(string); ok {
				fileName := filepath.Base(path)
				if strings.Contains(path, "test") {
					return fmt.Sprintf("テストファイル「%s」を作成します", fileName)
				}
				return fmt.Sprintf("ファイル「%s」を作成します", fileName)
			}
			return ""
		},

		"Edit": func(input map[string]interface{}) string {
			if path, ok := input["file_path"].(string); ok {
				fileName := filepath.Base(path)
				// Try to determine the type of edit from old_string/new_string
				if oldStr, ok := input["old_string"].(string); ok {
					if newStr, ok := input["new_string"].(string); ok {
						if strings.Contains(oldStr, "func") || strings.Contains(newStr, "func") {
							return fmt.Sprintf("関数を%sで修正します", fileName)
						}
						if strings.Contains(oldStr, "import") || strings.Contains(newStr, "import") {
							return fmt.Sprintf("インポート文を%sで更新します", fileName)
						}
						if strings.Contains(oldStr, "TODO") || strings.Contains(newStr, "TODO") {
							return fmt.Sprintf("TODOコメントを%sで更新します", fileName)
						}
					}
				}
				return fmt.Sprintf("ファイル「%s」を編集します", fileName)
			}
			return ""
		},

		"MultiEdit": func(input map[string]interface{}) string {
			if path, ok := input["file_path"].(string); ok {
				fileName := filepath.Base(path)
				if edits, ok := input["edits"].([]interface{}); ok {
					count := len(edits)
					return fmt.Sprintf("ファイル「%s」に%d箇所の変更を加えます", fileName, count)
				}
				return fmt.Sprintf("ファイル「%s」を複数箇所編集します", fileName)
			}
			return ""
		},

		"Grep": func(input map[string]interface{}) string {
			if pattern, ok := input["pattern"].(string); ok {
				path, _ := input["path"].(string)
				if path == "" {
					path = "プロジェクト全体"
				} else {
					path = fmt.Sprintf("「%s」", path)
				}

				// Determine what we're searching for
				if strings.Contains(pattern, "func") {
					return fmt.Sprintf("%sから関数定義を検索します", path)
				}
				if strings.Contains(pattern, "class") {
					return fmt.Sprintf("%sからクラス定義を検索します", path)
				}
				if strings.Contains(pattern, "TODO") {
					return fmt.Sprintf("%sからTODOコメントを検索します", path)
				}
				if strings.Contains(pattern, "error") || strings.Contains(pattern, "Error") {
					return fmt.Sprintf("%sからエラー処理を検索します", path)
				}

				return fmt.Sprintf("%sから「%s」を検索します", path, pattern)
			}
			return ""
		},

		"Glob": func(input map[string]interface{}) string {
			if pattern, ok := input["pattern"].(string); ok {
				if strings.Contains(pattern, "*test*") {
					return "テストファイルを探します"
				}
				if strings.Contains(pattern, "*.go") {
					return "Goファイルを探します"
				}
				if strings.Contains(pattern, "*.js") || strings.Contains(pattern, "*.ts") {
					return "JavaScriptファイルを探します"
				}
				if strings.Contains(pattern, "*.md") {
					return "ドキュメントファイルを探します"
				}
				return fmt.Sprintf("パターン「%s」に一致するファイルを探します", pattern)
			}
			return ""
		},

		"LS": func(input map[string]interface{}) string {
			if path, ok := input["path"].(string); ok {
				dirName := filepath.Base(path)
				if dirName == "." || dirName == "/" {
					return "現在のディレクトリの内容を確認します"
				}
				return fmt.Sprintf("ディレクトリ「%s」の内容を確認します", dirName)
			}
			return "ディレクトリの内容を確認します"
		},

		"WebFetch": func(input map[string]interface{}) string {
			if url, ok := input["url"].(string); ok {
				if strings.Contains(url, "github.com") {
					return "GitHubから情報を取得します"
				}
				if strings.Contains(url, "docs") {
					return "ドキュメントを参照します"
				}
				if strings.Contains(url, "api") {
					return "APIから情報を取得します"
				}
				domain := extractDomain(url)
				return fmt.Sprintf("「%s」から情報を取得します", domain)
			}
			return ""
		},

		"WebSearch": func(input map[string]interface{}) string {
			if query, ok := input["query"].(string); ok {
				return fmt.Sprintf("「%s」についてWeb検索します", query)
			}
			return ""
		},

		"Task": func(input map[string]interface{}) string {
			if desc, ok := input["description"].(string); ok {
				return fmt.Sprintf("タスク「%s」を実行します", desc)
			}
			if prompt, ok := input["prompt"].(string); ok {
				if strings.HasPrefix(prompt, "/") {
					// Slash command
					cmd := strings.Fields(prompt)[0]
					return fmt.Sprintf("コマンド「%s」を実行します", cmd)
				}
			}
			return "複雑なタスクを処理します"
		},

		"TodoWrite": func(input map[string]interface{}) string {
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
				if completed > 0 || inProgress > 0 {
					return fmt.Sprintf("TODOリストを更新します（完了: %d, 進行中: %d）", completed, inProgress)
				}
				return fmt.Sprintf("TODOリストを%d項目で更新します", len(todos))
			}
			return "TODOリストを更新します"
		},

		"NotebookRead": func(input map[string]interface{}) string {
			if path, ok := input["notebook_path"].(string); ok {
				fileName := filepath.Base(path)
				return fmt.Sprintf("Jupyterノートブック「%s」を読み込みます", fileName)
			}
			return ""
		},

		"NotebookEdit": func(input map[string]interface{}) string {
			if path, ok := input["notebook_path"].(string); ok {
				fileName := filepath.Base(path)
				if mode, ok := input["edit_mode"].(string); ok {
					switch mode {
					case "insert":
						return fmt.Sprintf("ノートブック「%s」に新しいセルを追加します", fileName)
					case "delete":
						return fmt.Sprintf("ノートブック「%s」からセルを削除します", fileName)
					default:
						return fmt.Sprintf("ノートブック「%s」のセルを編集します", fileName)
					}
				}
				return fmt.Sprintf("ノートブック「%s」を編集します", fileName)
			}
			return ""
		},

		"ExitPlanMode": func(input map[string]interface{}) string {
			return "実装計画を完了し、コーディングを開始します"
		},
	}

	return hn
}

// NarrateToolUse converts tool usage to natural Japanese
func (hn *HybridNarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%v", toolName, input)
	hn.cacheMu.RLock()
	if cached, ok := hn.cache[cacheKey]; ok {
		if cacheTime, ok := hn.cacheTime[cacheKey]; ok {
			if time.Since(cacheTime) < hn.cacheTTL {
				hn.cacheMu.RUnlock()
				return cached
			}
		}
	}
	hn.cacheMu.RUnlock()

	// Try rule-based narration first
	if rule, ok := hn.rules[toolName]; ok {
		if narration := rule(input); narration != "" {
			// Cache the result
			hn.cacheMu.Lock()
			hn.cache[cacheKey] = narration
			hn.cacheTime[cacheKey] = time.Now()
			hn.cacheMu.Unlock()
			return narration
		}
	}

	// Fall back to AI if enabled
	if hn.useAI && hn.ai != nil {
		narration := hn.ai.NarrateToolUse(toolName, input)
		if narration != "" {
			// Cache the AI result
			hn.cacheMu.Lock()
			hn.cache[cacheKey] = narration
			hn.cacheTime[cacheKey] = time.Now()
			hn.cacheMu.Unlock()
			return narration
		}
	}

	// Generic fallback
	return fmt.Sprintf("ツール「%s」を実行します", toolName)
}

// NarrateCodeBlock describes a code block
func (hn *HybridNarrator) NarrateCodeBlock(language, content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	lineCount := len(lines)

	switch language {
	case "go":
		// Analyze Go code
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
func (hn *HybridNarrator) NarrateFileOperation(operation, filePath string) string {
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

// Helper function to extract domain from URL
func extractDomain(url string) string {
	// Simple domain extraction
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}

	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		domain := parts[0]
		// Remove port if present
		if colonIdx := strings.Index(domain, ":"); colonIdx != -1 {
			domain = domain[:colonIdx]
		}
		return domain
	}
	return url
}

// OpenAINarrator uses OpenAI API for narration
type OpenAINarrator struct {
	apiKey  string
	model   string
	timeout time.Duration
}

// NewOpenAINarrator creates a new OpenAI narrator
func NewOpenAINarrator(apiKey string) *OpenAINarrator {
	return &OpenAINarrator{
		apiKey:  apiKey,
		model:   "gpt-3.5-turbo", // Use faster, cheaper model for narration
		timeout: 5 * time.Second,
	}
}

// NarrateToolUse uses OpenAI to narrate tool usage
func (ai *OpenAINarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	// For now, just return empty to use rule-based fallback
	// OpenAI integration can be implemented later
	return ""
}

// NarrateCodeBlock uses OpenAI to describe code blocks
func (ai *OpenAINarrator) NarrateCodeBlock(language, content string) string {
	// For now, just return empty to use rule-based fallback
	return ""
}

// NarrateFileOperation uses OpenAI to describe file operations
func (ai *OpenAINarrator) NarrateFileOperation(operation, filePath string) string {
	// For now, just return empty to use rule-based fallback
	return ""
}
