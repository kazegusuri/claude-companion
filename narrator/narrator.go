package narrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/logger"
)

// NotificationType represents different types of notifications
type NotificationType string

const (
	NotificationTypeCompact             NotificationType = "compact"
	NotificationTypeSessionStartStartup NotificationType = "session_start_startup"
	NotificationTypeSessionStartClear   NotificationType = "session_start_clear"
	NotificationTypeSessionStartResume  NotificationType = "session_start_resume"
	NotificationTypeSessionStartCompact NotificationType = "session_start_compact"
)

// Narrator interface for converting tool actions to natural language
type Narrator interface {
	NarrateToolUse(toolName string, input map[string]interface{}) string
	NarrateToolUsePermission(toolName string) string
	NarrateText(text string) string
	NarrateNotification(notificationType NotificationType) string
	NarrateTaskCompletion(description string, subagentType string) string
}

// HybridNarrator uses rules first, then falls back to AI
type HybridNarrator struct {
	rules          map[string]func(map[string]interface{}) string
	ai             *OpenAINarrator
	useAI          bool
	cache          map[string]string
	cacheMu        sync.RWMutex
	cacheTime      map[string]time.Time
	cacheTTL       time.Duration
	configNarrator *ConfigBasedNarrator
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
	hn.configNarrator = configNarrator

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

	// Generic fallback - return a simple message
	return fmt.Sprintf("%sを実行中...", toolName)
}

// NarrateToolUsePermission narrates a tool permission request
func (hn *HybridNarrator) NarrateToolUsePermission(toolName string) string {
	// Check if config narrator has permission rules
	if hn.configNarrator != nil {
		return hn.configNarrator.NarrateToolUsePermission(toolName)
	}

	// Default fallback
	return fmt.Sprintf("%sの使用許可を求めています", toolName)
}

// NarrateText returns the text as-is
func (hn *HybridNarrator) NarrateText(text string) string {
	return text
}

// NarrateNotification narrates notification events
func (hn *HybridNarrator) NarrateNotification(notificationType NotificationType) string {
	// Check if config narrator has notification rules
	if hn.configNarrator != nil {
		return hn.configNarrator.NarrateNotification(notificationType)
	}

	// Default fallback
	switch notificationType {
	case NotificationTypeCompact:
		return "コンテキストを圧縮しています"
	case NotificationTypeSessionStartStartup:
		return "こんにちは！何かお手伝いできることはありますか？"
	case NotificationTypeSessionStartClear:
		return "何かお手伝いできることはありますか？"
	case NotificationTypeSessionStartResume:
		return "前回の作業を続けましょう。どこから再開しますか？"
	case NotificationTypeSessionStartCompact:
		return "セッションを再開しました"
	default:
		return ""
	}
}

// NarrateTaskCompletion narrates task completion events
func (hn *HybridNarrator) NarrateTaskCompletion(description string, subagentType string) string {
	// Check if config narrator has task completion rules
	if hn.configNarrator != nil {
		return hn.configNarrator.NarrateTaskCompletion(description, subagentType)
	}

	// Default fallback
	if subagentType != "" && description != "" {
		return fmt.Sprintf("%s agentがタスク「%s」を完了しました", subagentType, description)
	} else if description != "" {
		return fmt.Sprintf("タスク「%s」が完了しました", description)
	}
	return "タスクが完了しました"
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
	apiKey     string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

// NewOpenAINarrator creates a new OpenAI narrator
func NewOpenAINarrator(apiKey string) *OpenAINarrator {
	// Check environment variable for model override
	model := os.Getenv("OPENAI_NARRATOR_MODEL")
	if model == "" {
		// Default to gpt-4.1-nano (newest, most efficient)
		// Other options: gpt-4o-nano, gpt-4o-mini, gpt-3.5-turbo, gpt-4o
		model = "gpt-4.1-nano"
	}

	return &OpenAINarrator{
		apiKey:  apiKey,
		model:   model,
		timeout: 5 * time.Second,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NarrateToolUse uses OpenAI to narrate tool usage
func (ai *OpenAINarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	// Create a prompt for the AI
	prompt := ai.createToolPrompt(toolName, input)

	// Call OpenAI API
	response, err := ai.callOpenAI(ctx, prompt)
	if err != nil {
		// Return empty to fallback to rule-based
		logger.LogError("Failed to call OpenAI for tool narration: %v", err)
		return ""
	}

	return response
}

// NarrateToolUsePermission narrates a tool permission request
func (ai *OpenAINarrator) NarrateToolUsePermission(toolName string) string {
	// For permission requests, we can use a simpler prompt
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	prompt := fmt.Sprintf(`以下のツールの使用許可を求めていることを、簡潔な日本語で説明してください。

ツール: %s

以下の点に注意してください：
- 10-15文字程度の短い文で説明
- 「〜の許可を求めています」の形式
- 自然で分かりやすい日本語を使用

例:
- ファイル書き込みの許可を求めています
- コマンド実行の許可を求めています`, toolName)

	response, err := ai.callOpenAI(ctx, prompt)
	if err != nil {
		// Fallback to simple format
		logger.LogError("Failed to call OpenAI for tool permission narration: %v", err)
		return fmt.Sprintf("%sの使用許可を求めています", toolName)
	}

	return response
}

// NarrateText returns the text as-is
func (ai *OpenAINarrator) NarrateText(text string) string {
	return text
}

// NarrateNotification narrates notification events
func (ai *OpenAINarrator) NarrateNotification(notificationType NotificationType) string {
	// Return default messages
	switch notificationType {
	case NotificationTypeCompact:
		return "コンテキストを圧縮しています"
	case NotificationTypeSessionStartStartup:
		return "こんにちは！何かお手伝いできることはありますか？"
	case NotificationTypeSessionStartClear:
		return "何かお手伝いできることはありますか？"
	case NotificationTypeSessionStartResume:
		return "前回の作業を続けましょう。どこから再開しますか？"
	case NotificationTypeSessionStartCompact:
		return "セッションを再開しました"
	default:
		return ""
	}
}

// NarrateTaskCompletion narrates task completion events
func (ai *OpenAINarrator) NarrateTaskCompletion(description string, subagentType string) string {
	// Return default message
	if subagentType != "" && description != "" {
		return fmt.Sprintf("%s agentがタスク「%s」を完了しました", subagentType, description)
	} else if description != "" {
		return fmt.Sprintf("タスク「%s」が完了しました", description)
	}
	return "タスクが完了しました"
}

// OpenAI API structures
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *openAIError `json:"error,omitempty"`
}

type openAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// createToolPrompt creates a prompt for the AI to narrate tool usage
func (ai *OpenAINarrator) createToolPrompt(toolName string, input map[string]interface{}) string {
	// Convert input to a readable format
	inputJSON, _ := json.MarshalIndent(input, "", "  ")

	prompt := fmt.Sprintf(`あなたはAIアシスタントの行動を簡潔に説明するロボットです。以下のツール実行を、まるでロボットが喋っているかのように短い日本語で説明してください。

ツール: %s
入力パラメータ:
%s

以下の点に注意してください：
- 10-20文字程度の短い文で説明
- 「〜します」の形式で終わる
- 技術的な詳細は省略
- 自然で分かりやすい日本語を使用
- ファイル名やパスは「」で囲む

例:
- ファイル「main.go」を読み込みます
- テストを実行します
- 変更をコミットします`, toolName, string(inputJSON))

	return prompt
}

// callOpenAI makes the actual API call to OpenAI
func (ai *OpenAINarrator) callOpenAI(ctx context.Context, prompt string) (string, error) {
	request := openAIRequest{
		Model: ai.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: "あなたはAIアシスタントの行動を簡潔に説明するロボットです。短く、分かりやすい日本語で応答してください。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   50,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.apiKey)

	resp, err := ai.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response openAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if response.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s", response.Error.Message)
	}

	if len(response.Choices) > 0 {
		return strings.TrimSpace(response.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no response from OpenAI")
}

// NoOpNarrator is a narrator that returns empty strings for all operations
type NoOpNarrator struct{}

// NewNoOpNarrator creates a new no-op narrator
func NewNoOpNarrator() *NoOpNarrator {
	return &NoOpNarrator{}
}

// NarrateToolUse returns empty string
func (n *NoOpNarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	return ""
}

// NarrateToolUsePermission returns empty string
func (n *NoOpNarrator) NarrateToolUsePermission(toolName string) string {
	return ""
}

// NarrateText returns the text as-is
func (n *NoOpNarrator) NarrateText(text string) string {
	return text
}

// NarrateNotification returns empty string
func (n *NoOpNarrator) NarrateNotification(notificationType NotificationType) string {
	return ""
}

// NarrateTaskCompletion returns empty string
func (n *NoOpNarrator) NarrateTaskCompletion(description string, subagentType string) string {
	return ""
}
