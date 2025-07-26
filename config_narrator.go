package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ConfigBasedNarrator uses configuration file for narrative rules
type ConfigBasedNarrator struct {
	config        *NarratorConfig
	defaultConfig *NarratorConfig
}

// NewConfigBasedNarrator creates a new config-based narrator
func NewConfigBasedNarrator(config *NarratorConfig) *ConfigBasedNarrator {
	return &ConfigBasedNarrator{
		config:        config,
		defaultConfig: GetDefaultNarratorConfig(),
	}
}

// getStringOrDefault returns the value from config if not empty, otherwise from defaultConfig
func (cn *ConfigBasedNarrator) getStringOrDefault(configValue, defaultValue string) string {
	if configValue != "" {
		return configValue
	}
	return defaultValue
}

// NarrateToolUse converts tool usage to natural Japanese using config rules
func (cn *ConfigBasedNarrator) NarrateToolUse(toolName string, input map[string]interface{}) string {
	rules, ok := cn.config.Rules[toolName]
	if !ok {
		// Try default config
		if defaultRules, ok := cn.defaultConfig.Rules[toolName]; ok {
			rules = defaultRules
		} else {
			// No rules for this tool in both configs
			template := cn.getStringOrDefault(cn.config.Messages.GenericToolExecution, cn.defaultConfig.Messages.GenericToolExecution)
			if template != "" {
				return strings.ReplaceAll(template, "{tool}", toolName)
			}
			panic(fmt.Sprintf("No narration config found for tool: %s", toolName))
		}
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
				msg := cn.getStringOrDefault(cn.config.Messages.CurrentDirectory, cn.defaultConfig.Messages.CurrentDirectory)
				if msg != "" {
					return msg
				}
				panic("No currentDirectory message in config")
			}
			return strings.ReplaceAll(rules.Default, "{dirname}", dirName)
		}
		msg := cn.getStringOrDefault(cn.config.Messages.DirectoryContents, cn.defaultConfig.Messages.DirectoryContents)
		if msg != "" {
			return msg
		}
		panic("No directoryContents message in config")
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
				template := cn.getStringOrDefault(cn.config.Messages.GenericCommandExecution, cn.defaultConfig.Messages.GenericCommandExecution)
				if template != "" {
					return strings.ReplaceAll(template, "{command}", cmd)
				}
				panic(fmt.Sprintf("No genericCommandExecution message in config for command: %s", cmd))
			}
		}
		msg := cn.getStringOrDefault(cn.config.Messages.ComplexTask, cn.defaultConfig.Messages.ComplexTask)
		if msg != "" {
			return msg
		}
		panic("No complexTask message in config")
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
		msg := cn.getStringOrDefault(cn.config.Messages.TodoListUpdate, cn.defaultConfig.Messages.TodoListUpdate)
		if msg != "" {
			return msg
		}
		panic("No todoListUpdate message in config")
	}

	// Handle tools with simple default messages
	if rules.Default != "" {
		return rules.Default
	}

	// Generic fallback
	template := cn.getStringOrDefault(cn.config.Messages.GenericToolExecution, cn.defaultConfig.Messages.GenericToolExecution)
	if template != "" {
		return strings.ReplaceAll(template, "{tool}", toolName)
	}
	panic(fmt.Sprintf("No narration config found for tool: %s", toolName))
}

// NarrateCodeBlock describes a code block
func (cn *ConfigBasedNarrator) NarrateCodeBlock(language, content string) string {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	lineCount := len(lines)

	// Check if we have language-specific rules in custom config
	langRules, hasCustom := cn.config.CodeBlockInfo.Languages[language]
	defaultLangRules, hasDefault := cn.defaultConfig.CodeBlockInfo.Languages[language]

	if hasCustom || hasDefault {
		// Check patterns from custom config first
		if hasCustom {
			for _, pattern := range langRules.Patterns {
				if strings.Contains(content, pattern.Contains) {
					return pattern.Message
				}
			}
		}

		// Check patterns from default config
		if hasDefault {
			for _, pattern := range defaultLangRules.Patterns {
				if strings.Contains(content, pattern.Contains) {
					return pattern.Message
				}
			}
		}

		// Use language default
		var langDefault string
		if hasCustom && langRules.Default != "" {
			langDefault = langRules.Default
		} else if hasDefault && defaultLangRules.Default != "" {
			langDefault = defaultLangRules.Default
		}

		if langDefault != "" {
			return strings.ReplaceAll(langDefault, "{lines}", fmt.Sprintf("%d", lineCount))
		}
	}

	// Also check alternative language names
	alternatives := map[string]string{
		"py":  "python",
		"js":  "javascript",
		"ts":  "typescript",
		"sh":  "bash",
		"yml": "yaml",
		"md":  "markdown",
	}

	if alt, ok := alternatives[language]; ok {
		if langRules, ok := cn.config.CodeBlockInfo.Languages[alt]; ok {
			// Check patterns
			for _, pattern := range langRules.Patterns {
				if strings.Contains(content, pattern.Contains) {
					return pattern.Message
				}
			}
			// Use language default
			if langRules.Default != "" {
				return strings.ReplaceAll(langRules.Default, "{lines}", fmt.Sprintf("%d", lineCount))
			}
		}
	}

	// Use default
	if lineCount == 1 {
		defaultMsg := cn.getStringOrDefault(cn.config.CodeBlockInfo.Default.SingleLine, cn.defaultConfig.CodeBlockInfo.Default.SingleLine)
		if defaultMsg != "" {
			return defaultMsg
		}
		panic("No singleLine code block message in config")
	}

	defaultMsg := cn.getStringOrDefault(cn.config.CodeBlockInfo.Default.MultipleLines, cn.defaultConfig.CodeBlockInfo.Default.MultipleLines)
	if defaultMsg != "" {
		return strings.ReplaceAll(defaultMsg, "{lines}", fmt.Sprintf("%d", lineCount))
	}
	panic(fmt.Sprintf("No multipleLines code block message in config for %d lines", lineCount))
}

// NarrateFileOperation describes file operations
func (cn *ConfigBasedNarrator) NarrateFileOperation(operation, filePath string) string {
	fileName := filepath.Base(filePath)

	// Normalize operation to use "default" for unknown operations
	lookupKey := operation
	switch operation {
	case "Read", "Write", "Edit", "Delete":
		// Keep as-is for known operations
	default:
		// Use "default" for unknown operations
		lookupKey = "default"
	}

	// Check if Messages and FileOperations are configured
	if cn.config.Messages.FileOperations != nil {
		if template, ok := cn.config.Messages.FileOperations[lookupKey]; ok {
			msg := strings.ReplaceAll(template, "{filename}", fileName)
			if lookupKey == "default" {
				msg = strings.ReplaceAll(msg, "{operation}", operation)
			}
			return msg
		}
	}

	// Check default config
	if cn.defaultConfig.Messages.FileOperations != nil {
		if template, ok := cn.defaultConfig.Messages.FileOperations[lookupKey]; ok {
			msg := strings.ReplaceAll(template, "{filename}", fileName)
			if lookupKey == "default" {
				msg = strings.ReplaceAll(msg, "{operation}", operation)
			}
			return msg
		}
	}

	// Panic if no message found
	if lookupKey == "default" {
		panic(fmt.Sprintf("No default file operation message in config for operation: %s on file: %s", operation, fileName))
	} else {
		panic(fmt.Sprintf("No %s file operation message in config for file: %s", operation, fileName))
	}
}
