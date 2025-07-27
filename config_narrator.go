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
			pending := 0
			var todoList strings.Builder

			for i, todo := range todos {
				if todoMap, ok := todo.(map[string]interface{}); ok {
					content := ""
					if c, ok := todoMap["content"].(string); ok {
						content = c
					}
					if status, ok := todoMap["status"].(string); ok {
						emoji := ""
						switch status {
						case "completed":
							completed++
							emoji = "✅"
						case "in_progress":
							inProgress++
							emoji = "🔄"
						case "pending":
							pending++
							emoji = "⏳"
						}
						todoList.WriteString(fmt.Sprintf("\n    %d. %s %s", i+1, emoji, content))
					}
				}
			}
			msg := strings.ReplaceAll(rules.Default, "{completed}", fmt.Sprintf("%d", completed))
			msg = strings.ReplaceAll(msg, "{in_progress}", fmt.Sprintf("%d", inProgress))
			return msg + todoList.String()
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

// NarrateText returns the text as-is
func (cn *ConfigBasedNarrator) NarrateText(text string) string {
	return text
}
