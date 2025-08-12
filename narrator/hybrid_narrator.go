package narrator

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// HybridNarrator uses multiple narrators in sequence
type HybridNarrator struct {
	narrators []Narrator
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
		cache:     make(map[string]string),
		cacheTime: make(map[string]time.Time),
		cacheTTL:  30 * time.Minute,
		narrators: make([]Narrator, 0),
	}

	// Load config if path is provided, otherwise use defaults
	var config *NarratorConfig
	if configPath != nil && *configPath != "" {
		config = LoadNarratorConfigWithDefaults(*configPath)
	} else {
		config = GetDefaultNarratorConfig()
	}

	// Create rule-based narrator (always first)
	ruleBasedNarrator := NewRuleBasedNarrator(config)
	hn.narrators = append(hn.narrators, ruleBasedNarrator)

	// Add AI narrator if enabled
	if useAI && apiKey != "" {
		aiNarrator := NewOpenAINarrator(apiKey)
		hn.narrators = append(hn.narrators, aiNarrator)
	}

	return hn
}

// NarrateToolUse converts tool usage to natural Japanese
func (hn *HybridNarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	// Create cache key - for Bash tool, use command; otherwise use sorted input keys
	var cacheKey string
	if toolName == "Bash" {
		if command, ok := input["command"].(string); ok {
			cacheKey = fmt.Sprintf("%s:%s", toolName, command)
		} else {
			cacheKey = fmt.Sprintf("%s:no-command", toolName)
		}
	} else {
		// Create cache key using tool name and sorted input keys
		keys := make([]string, 0, len(input))
		for key := range input {
			keys = append(keys, key)
		}
		// Sort keys for consistent cache key
		if len(keys) > 1 {
			sort.Strings(keys)
		}
		cacheKey = fmt.Sprintf("%s:%s", toolName, strings.Join(keys, ","))
	}

	// Check cache first
	hn.cacheMu.RLock()
	if cached, ok := hn.cache[cacheKey]; ok {
		if cacheTime, ok := hn.cacheTime[cacheKey]; ok {
			if time.Since(cacheTime) < hn.cacheTTL {
				hn.cacheMu.RUnlock()
				return cached, false
			}
		}
	}
	hn.cacheMu.RUnlock()

	// Try each narrator in sequence
	for _, narrator := range hn.narrators {
		narration, shouldFallback := narrator.NarrateToolUse(toolName, input)
		if !shouldFallback {
			// Cache the result
			hn.cacheMu.Lock()
			hn.cache[cacheKey] = narration
			hn.cacheTime[cacheKey] = time.Now()
			hn.cacheMu.Unlock()
			return narration, false
		}
	}

	// Generic fallback - return a simple message
	return fmt.Sprintf("%sを実行中...", toolName), false
}

// NarrateToolUsePermission narrates a tool permission request
func (hn *HybridNarrator) NarrateToolUsePermission(toolName string) (string, bool) {
	// Check cache first
	cacheKey := fmt.Sprintf("permission:%s", toolName)
	hn.cacheMu.RLock()
	if cached, ok := hn.cache[cacheKey]; ok {
		if cacheTime, ok := hn.cacheTime[cacheKey]; ok {
			if time.Since(cacheTime) < hn.cacheTTL {
				hn.cacheMu.RUnlock()
				return cached, false
			}
		}
	}
	hn.cacheMu.RUnlock()

	// Try each narrator in sequence
	for _, narrator := range hn.narrators {
		narration, shouldFallback := narrator.NarrateToolUsePermission(toolName)
		if !shouldFallback {
			// Cache the result
			hn.cacheMu.Lock()
			hn.cache[cacheKey] = narration
			hn.cacheTime[cacheKey] = time.Now()
			hn.cacheMu.Unlock()
			return narration, false
		}
	}
	// Fallback
	return fmt.Sprintf("%sの使用許可を求めています", toolName), false
}

// NarrateText returns the text as-is
func (hn *HybridNarrator) NarrateText(text string, isThinking bool) (string, bool) {
	// Try each narrator in sequence with first line only
	for _, narrator := range hn.narrators {
		narration, shouldFallback := narrator.NarrateText(text, isThinking)
		if !shouldFallback {
			return narration, false
		}
	}

	// Extract first line for narration
	firstLine := text
	if idx := strings.IndexByte(text, '\n'); idx != -1 {
		firstLine = text[:idx]
	}

	// Default behavior - return first line as-is
	return firstLine, false
}

// NarrateNotification narrates notification events
func (hn *HybridNarrator) NarrateNotification(notificationType NotificationType) (string, bool) {
	// Try each narrator in sequence
	for _, narrator := range hn.narrators {
		narration, shouldFallback := narrator.NarrateNotification(notificationType)
		if !shouldFallback {
			return narration, false
		}
	}
	// No fallback
	return "", false
}

// NarrateTaskCompletion narrates task completion events
func (hn *HybridNarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	// Try each narrator in sequence
	for _, narrator := range hn.narrators {
		narration, shouldFallback := narrator.NarrateTaskCompletion(description, subagentType)
		if !shouldFallback {
			return narration, false
		}
	}
	// Fallback
	return "タスクが完了しました", false
}
