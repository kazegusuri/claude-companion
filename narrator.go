package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Narrator interface for converting tool actions to natural language
type Narrator interface {
	NarrateToolUse(toolName string, input map[string]interface{}) string
	NarrateCodeBlock(language, content string) string
	NarrateFileOperation(operation, filePath string) string
	NarrateText(text string) string
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

	// Generic fallback
	panic(fmt.Sprintf("No narration config found for tool: %s", toolName))
}

// NarrateCodeBlock describes a code block
func (hn *HybridNarrator) NarrateCodeBlock(language, content string) string {
	// Delegate to config-based narrator
	return hn.configNarrator.NarrateCodeBlock(language, content)
}

// NarrateFileOperation describes file operations
func (hn *HybridNarrator) NarrateFileOperation(operation, filePath string) string {
	// Delegate to config-based narrator
	return hn.configNarrator.NarrateFileOperation(operation, filePath)
}

// NarrateText returns the text as-is
func (hn *HybridNarrator) NarrateText(text string) string {
	return text
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

// NarrateText returns the text as-is
func (ai *OpenAINarrator) NarrateText(text string) string {
	return text
}
