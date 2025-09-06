package narrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
)

// OpenAITranslator uses OpenAI API for English to Japanese translation
type OpenAITranslator struct {
	apiKey     string
	model      string
	httpClient *http.Client
	cache      map[string]string
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
	cacheTime  map[string]time.Time
}

// NewOpenAITranslator creates a new OpenAI translator
func NewOpenAITranslator(apiKey string) *OpenAITranslator {
	return &OpenAITranslator{
		apiKey: apiKey,
		model:  "gpt-4o-mini", // Fast and cost-effective
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:     make(map[string]string),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  1 * time.Hour,
	}
}

// Translate translates English text to Japanese using OpenAI
func (t *OpenAITranslator) Translate(ctx context.Context, text string) (string, error) {
	// Check if text is already mostly Japanese
	if t.isMostlyJapanese(text) {
		return text, nil
	}

	// Check cache
	t.cacheMu.RLock()
	if cached, ok := t.cache[text]; ok {
		if cacheTime, ok := t.cacheTime[text]; ok {
			if time.Since(cacheTime) < t.cacheTTL {
				t.cacheMu.RUnlock()
				return cached, nil
			}
		}
	}
	t.cacheMu.RUnlock()

	// Call OpenAI API
	translated, err := t.callOpenAI(ctx, text)
	if err != nil {
		return text, err // Return original text on error
	}

	// Cache the result
	t.cacheMu.Lock()
	t.cache[text] = translated
	t.cacheTime[text] = time.Now()
	t.cacheMu.Unlock()

	return translated, nil
}

// callOpenAI makes the actual API call to OpenAI
func (t *OpenAITranslator) callOpenAI(ctx context.Context, text string) (string, error) {
	request := openAIRequest{
		Model: t.model,
		Messages: []openAIMessage{
			{
				Role: "system",
				Content: `You are a translator specializing in technical documentation. 
Translate English text to natural Japanese suitable for text-to-speech.
Rules:
1. Keep technical terms that are commonly used in Japanese as-is (API, URL, JSON, etc.)
2. Translate programming-related phrases naturally
3. For file operations, use appropriate Japanese (読み込み, 書き込み, etc.)
4. Return ONLY the translated text, no explanations
5. If the text is already in Japanese, return it unchanged`,
			},
			{
				Role:    "user",
				Content: text,
			},
		},
		Temperature: 0.3,
		MaxTokens:   200,
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
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	resp, err := t.httpClient.Do(req)
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

// isMostlyJapanese checks if text contains Japanese characters
func (t *OpenAITranslator) isMostlyJapanese(text string) bool {
	// Count Japanese characters (Hiragana, Katakana, Kanji)
	japaneseCount := 0
	totalCount := 0

	for _, r := range text {
		if r >= 'A' && r <= 'z' {
			totalCount++
		} else if (r >= 0x3040 && r <= 0x309F) || // Hiragana
			(r >= 0x30A0 && r <= 0x30FF) || // Katakana
			(r >= 0x4E00 && r <= 0x9FAF) { // Kanji
			japaneseCount++
			totalCount++
		}
	}

	if totalCount == 0 {
		return true // No alphabetic characters, assume it's OK
	}

	// If more than 50% is Japanese, consider it mostly Japanese
	return float64(japaneseCount)/float64(totalCount) > 0.5
}

// TranslatorInterface defines the interface for translators
type TranslatorInterface interface {
	Translate(ctx context.Context, text string) (string, error)
}

// CombinedTranslator tries rule-based first, then falls back to OpenAI
type CombinedTranslator struct {
	simpleTranslator *SimpleTranslator
	openAITranslator *OpenAITranslator
	useOpenAI        bool
}

// NewCombinedTranslator creates a translator that combines rule-based and OpenAI
func NewCombinedTranslator(apiKey string, useOpenAI bool) *CombinedTranslator {
	ct := &CombinedTranslator{
		simpleTranslator: NewSimpleTranslator(),
		useOpenAI:        useOpenAI,
	}

	if useOpenAI && apiKey != "" {
		ct.openAITranslator = NewOpenAITranslator(apiKey)
	}

	return ct
}

// Translate attempts translation using available methods
func (ct *CombinedTranslator) Translate(ctx context.Context, text string) (string, error) {
	// Always try simple translation first
	translated := ct.simpleTranslator.Translate(text)

	// If simple translation didn't change much and OpenAI is available, use it
	if ct.useOpenAI && ct.openAITranslator != nil {
		// Check if simple translation still contains significant English
		if ct.containsSignificantEnglish(translated) {
			openAITranslated, err := ct.openAITranslator.Translate(ctx, text)
			if err == nil {
				return openAITranslated, nil
			}
			// Fall back to simple translation on error
			logger.LogError("Failed to translate with OpenAI, falling back to simple translation: %v", err)
		}
	}

	return translated, nil
}

// containsSignificantEnglish checks if text contains significant English words
func (ct *CombinedTranslator) containsSignificantEnglish(text string) bool {
	words := strings.Fields(text)
	englishWords := 0

	for _, word := range words {
		// Skip short words and numbers
		if len(word) <= 2 {
			continue
		}

		// Check if word is mostly ASCII letters
		asciiCount := 0
		for _, r := range word {
			if r >= 'A' && r <= 'z' {
				asciiCount++
			}
		}

		if float64(asciiCount)/float64(len(word)) > 0.8 {
			englishWords++
		}
	}

	// If more than 30% of words are English, consider it significant
	return float64(englishWords)/float64(len(words)) > 0.3
}
