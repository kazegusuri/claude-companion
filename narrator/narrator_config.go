package narrator

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
	Rules         map[string]ToolRules `json:"rules"`
	Messages      MessageTemplates     `json:"messages"`
	FileTypeNames map[string]string    `json:"fileTypeNames"` // Extension to file type name mapping
	MCPRules      map[string]MCPRules  `json:"mcpRules"`      // MCP-specific rules by server name
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

	// For permission requests
	PermissionMessage string `json:"permissionMessage,omitempty"`

	// For configurable input value captures and replacements
	Captures []CaptureRule `json:"captures,omitempty"`
}

// PrefixRule represents a prefix-based rule (mainly for Bash commands)
type PrefixRule struct {
	Prefix  string `json:"prefix"`
	Message string `json:"message"`
}

// PatternRule represents a pattern-based rule
type PatternRule struct {
	Contains        string `json:"contains"`
	Message         string `json:"message"`
	AppendToDefault bool   `json:"appendToDefault"` // If true, append pattern info to default message
}

// CaptureRule represents a configurable input capture and replacement rule
type CaptureRule struct {
	InputKey      string `json:"inputKey"`      // The key in the input map to capture from (placeholder will be {inputKey})
	ParseFileType bool   `json:"parseFileType"` // If true, parse the value as a file path and add {filetype} replacement
	Type          string `json:"type,omitempty"` // Optional type: "file" to use filepath.Base on the value
}

// MCPRules represents rules for a specific MCP server
type MCPRules struct {
	Default string               `json:"default"` // Default message for unknown operations
	Rules   map[string]ToolRules `json:"rules"`   // Operation-specific rules
}

// MessageTemplates contains general message templates
type MessageTemplates struct {
	GenericToolExecution    string `json:"genericToolExecution"`    // For unmatched tools
	GenericCommandExecution string `json:"genericCommandExecution"` // For unmatched commands
	ComplexTask             string `json:"complexTask"`             // For complex tasks
	CurrentDirectory        string `json:"currentDirectory"`        // For current directory
	DirectoryContents       string `json:"directoryContents"`       // For directory listing
	TodoListUpdate          string `json:"todoListUpdate"`          // For todo list updates
	GenericToolPermission   string `json:"genericToolPermission"`   // For tool permission requests
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
