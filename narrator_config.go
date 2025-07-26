package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

//go:embed narrator-rules.json
var defaultNarratorRulesJSON string

// NarratorConfig represents the configuration for narrative rules
type NarratorConfig struct {
	Rules map[string]ToolRules `json:"rules"`
}

// ToolRules represents rules for a specific tool
type ToolRules struct {
	// For simple tools, just use a default message
	Default string `json:"default,omitempty"`

	// For Bash commands, use prefix matching
	Prefixes []PrefixRule `json:"prefixes,omitempty"`

	// For file-based tools, use extension-based rules
	Extensions map[string]string `json:"extensions,omitempty"`

	// For pattern-based matching
	Patterns []PatternRule `json:"patterns,omitempty"`
}

// PrefixRule represents a prefix-based rule (mainly for Bash commands)
type PrefixRule struct {
	Prefix  string `json:"prefix"`
	Message string `json:"message"`
}

// PatternRule represents a pattern-based rule
type PatternRule struct {
	Contains string `json:"contains"`
	Message  string `json:"message"`
}

// LoadNarratorConfig loads narrator configuration from a file
func LoadNarratorConfig(path string) (*NarratorConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config NarratorConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// LoadNarratorConfigWithDefaults loads config or returns default if file doesn't exist
func LoadNarratorConfigWithDefaults(path string) *NarratorConfig {
	config, err := LoadNarratorConfig(path)
	if err == nil {
		return config
	}

	// Return default configuration
	return GetDefaultNarratorConfig()
}

// GetDefaultNarratorConfig returns the default narrator configuration
func GetDefaultNarratorConfig() *NarratorConfig {
	var config NarratorConfig
	if err := json.Unmarshal([]byte(defaultNarratorRulesJSON), &config); err != nil {
		// This should never happen as the embedded JSON is validated at compile time
		panic(fmt.Sprintf("failed to parse embedded narrator rules: %v", err))
	}
	return &config
}
