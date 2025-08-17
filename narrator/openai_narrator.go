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
	"time"

	"github.com/kazegusuri/claude-companion/logger"
)

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
func (ai *OpenAINarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	// Create a prompt for the AI
	prompt := ai.createToolPrompt(toolName, input)

	// Call OpenAI API
	response, err := ai.callOpenAI(ctx, prompt, 0.3, 50)
	if err != nil {
		// Return empty to fallback to rule-based
		logger.LogError("Failed to call OpenAI for tool narration: %v", err)
		return "", true
	}

	return response, false
}

// NarrateToolUsePermission narrates a tool permission request
func (ai *OpenAINarrator) NarrateToolUsePermission(toolName string) (string, bool) {
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

	response, err := ai.callOpenAI(ctx, prompt, 0.3, 50)
	if err != nil {
		// Fallback to simple format
		logger.LogError("Failed to call OpenAI for tool permission narration: %v", err)
		return fmt.Sprintf("%sの使用許可を求めています", toolName), true
	}

	return response, false
}

// NarrateText returns the text as-is
func (ai *OpenAINarrator) NarrateText(text string, isThinking bool) (string, bool) {
	// If text is a single line without newlines, return as-is
	if !strings.Contains(text, "\n") {
		return text, false
	}

	// For permission requests, we can use a simpler prompt
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	// Determine max sentences based on text size (4KB = 4096 bytes)
	maxSentences := 3
	maxTokens := 150
	if len(text) > 4096 {
		maxSentences = 4
		maxTokens = 200
	}

	prompt := fmt.Sprintf(`以下の点に注意して与えられた文章を要約して簡潔な文章にしてください。

- 最大でも%d文で終わるようにすること
- 復数文になる場合でも改行を含めない
- 読み上げ(Text To Speach)に利用するため、読み上げやすい日本語を使用
- URLやファイルのパスなどは含まない
- 「私は」などの主語は省略すること
- 1行目の文章をもとに提案なのか報告なのか確認なのかを意識すること
- 1行目の文章から行ったことであれば「〜しました」、これから行うことであれば「〜します」という形式にすること
- 文章が長い場合は、特に重要なポイントを中心に要約すること

以下の文章を要約してください:
%s
`, maxSentences, text)

	if isThinking {
		prompt = fmt.Sprintf(`あなたは質問や課題を与えられて思考中です。与えられた文章をこれからあなたが行う行動として簡潔に要約してください。

- 最大でも%d文で終わるようにすること
- 復数文になる場合でも改行を含めない
- 読み上げ(Text To Speach)に利用するため、読み上げやすい日本語を使用
- URLやファイルのパスなどは含まない
- 「私は」などの主語は省略すること
- 「〜します」のような一人称での表現にすること
- これからの行うことをとくに意識して、行動を説明する

以下の文章を要約してください:
%s
`, maxSentences, text)
	}

	// Use higher token limit for text narration
	response, err := ai.callOpenAI(ctx, prompt, 0.8, maxTokens)
	if err != nil {
		// Fallback to simple format
		logger.LogError("Failed to call OpenAI for tool permission narration: %v", err)
		return text, true
	}

	return response, false
}

// NarrateNotification narrates notification events
func (ai *OpenAINarrator) NarrateNotification(notificationType NotificationType) (string, bool) {
	// Always return empty string and false
	return "", false
}

// NarrateTaskCompletion narrates task completion events
func (ai *OpenAINarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	// Always return empty string and false
	return "", false
}

// NarrateAPIError narrates an API error
func (ai *OpenAINarrator) NarrateAPIError(statusCode int, errorType string, message string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), ai.timeout)
	defer cancel()

	prompt := fmt.Sprintf(`APIエラーが発生しました。以下のエラーをユーザーに分かりやすく日本語で説明してください。

エラー情報:
- HTTPステータスコード: %d
- エラータイプ: %s
- エラーメッセージ: %s

以下の点に注意してください：
- 20-30文字程度の短い文で説明
- 技術的な詳細は省略し、ユーザーが理解しやすい表現を使用
- 可能であれば対処法も簡潔に含める
- Claude のサーバーで発生したことを説明してください。

例:
- Claude のサーバーが過負荷状態です。少し待ってから再試行してください。
- Claude のサーバーからリクエストエラーを受け取りました。入力内容を確認してください。`, statusCode, errorType, message)

	response, err := ai.callOpenAI(ctx, prompt, 0.3, 50)
	if err != nil {
		// Fallback to simple format
		logger.LogError("Failed to call OpenAI for API error narration: %v", err)
		return fmt.Sprintf("APIエラー %d: %s", statusCode, message), true
	}

	return response, false
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
	// For Bash tool, use command field if available
	keysStr := ""
	if toolName == "Bash" {
		if command, ok := input["command"].(string); ok {
			keysStr = command
		}
	} else {
		// Extract only the keys from input parameters
		keys := make([]string, 0, len(input))
		for key := range input {
			keys = append(keys, key)
		}
		if len(keys) > 0 {
			keysStr = strings.Join(keys, ", ")
		}
	}

	// Adjust prompt based on tool type
	var promptTemplate string
	if toolName == "Bash" {
		promptTemplate = `あなたはAIアシスタントの行動を簡潔に説明するロボットです。以下のコマンド実行を、まるでロボットが喋っているかのように短い日本語で説明してください。

ツール: %s
コマンド: %s

以下の点に注意してください：
- 10-20文字程度の短い文で説明
- 「〜します」の形式で終わる
- 技術的な詳細は省略
- 自然で分かりやすい日本語を使用
- コマンド名から何をするかを判断する

例:
- テストを実行します
- パッケージをインストールします
- ビルドを実行します
- コードをフォーマットします`
	} else {
		promptTemplate = `あなたはAIアシスタントの行動を簡潔に説明するロボットです。以下のツール実行を、まるでロボットが喋っているかのように短い日本語で説明してください。

ツール: %s
入力パラメータのキー: %s

以下の点に注意してください：
- 10-20文字程度の短い文で説明
- 「〜します」の形式で終わる
- 技術的な詳細は省略
- 自然で分かりやすい日本語を使用
- ファイル名やパスは「」で囲む
- パスはファイル名のみにする
- regexp や 正規表現のパターンなどがあれば 正規表現 という表現を使う

例:
- ファイル「main.go」を読み込みます
- テストを実行します
- 変更をコミットします
- 正規表現を使って検索します`
	}

	prompt := fmt.Sprintf(promptTemplate, toolName, keysStr)
	return prompt
}

// callOpenAI makes the actual API call to OpenAI
func (ai *OpenAINarrator) callOpenAI(ctx context.Context, prompt string, temperature float64, maxTokens int) (string, error) {
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
		Temperature: temperature,
		MaxTokens:   maxTokens,
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
