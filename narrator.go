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
	return NewHybridNarratorWithConfig(apiKey, useAI, nil)
}

// NewHybridNarratorWithConfig creates a new hybrid narrator with optional config
func NewHybridNarratorWithConfig(apiKey string, useAI bool, configPath *string) *HybridNarrator {
	hn := &HybridNarrator{
		useAI:     useAI,
		cache:     make(map[string]string),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  30 * time.Minute,
	}

	if useAI && apiKey != "" {
		hn.ai = NewOpenAINarrator(apiKey)
	}

	// Load config if path is provided, otherwise use defaults
	var config *NarratorConfig
	if configPath != nil && *configPath != "" {
		config = LoadNarratorConfigWithDefaults(*configPath)
	} else {
		config = GetDefaultNarratorConfig()
	}

	// Create config-based narrator
	configNarrator := NewConfigBasedNarrator(config)

	// Convert config-based narrator to rule functions
	// Create a wrapper function for each tool that delegates to configNarrator
	hn.rules = map[string]func(map[string]interface{}) string{}

	// Get all tool names from config
	for toolName := range config.Rules {
		// Capture toolName in closure
		tool := toolName
		hn.rules[tool] = func(input map[string]interface{}) string {
			return configNarrator.NarrateToolUse(tool, input)
		}
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
