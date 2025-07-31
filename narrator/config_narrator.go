package narrator

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
				// Use basename for the path
				path = fmt.Sprintf("「%s」", filepath.Base(path))
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

			for _, todo := range todos {
				if todoMap, ok := todo.(map[string]interface{}); ok {
					if status, ok := todoMap["status"].(string); ok {
						switch status {
						case "completed":
							completed++
						case "in_progress":
							inProgress++
						case "pending":
							pending++
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

	// Handle MCP tools with dynamic parameters
	if strings.HasPrefix(toolName, "mcp__") {
		// Handle memory-related tools
		if toolName == "mcp__serena__read_memory" {
			if memoryFile, ok := input["memory_file_name"].(string); ok {
				return strings.ReplaceAll(rules.Default, "{memory_file_name}", memoryFile)
			}
		}

		// Handle file search tools
		if toolName == "mcp__serena__find_file" {
			if fileMask, ok := input["file_mask"].(string); ok {
				// Check patterns
				for _, pattern := range rules.Patterns {
					if strings.Contains(fileMask, pattern.Contains) {
						return pattern.Message
					}
				}
				return strings.ReplaceAll(rules.Default, "{file_mask}", fileMask)
			}
		}

		// Handle pattern search tools
		if toolName == "mcp__serena__search_for_pattern" {
			if pattern, ok := input["substring_pattern"].(string); ok {
				// Check patterns
				for _, rule := range rules.Patterns {
					if strings.Contains(pattern, rule.Contains) {
						return rule.Message
					}
				}
				return strings.ReplaceAll(rules.Default, "{substring_pattern}", pattern)
			}
		}

		// Handle symbol search tools
		if toolName == "mcp__serena__find_symbol" {
			if namePath, ok := input["name_path"].(string); ok {
				// Check patterns
				for _, pattern := range rules.Patterns {
					if strings.Contains(namePath, pattern.Contains) {
						return strings.ReplaceAll(pattern.Message, "{name_path}", namePath)
					}
				}
				return strings.ReplaceAll(rules.Default, "{name_path}", namePath)
			}
		}

		// Handle MCP resource tools
		if toolName == "ReadMcpResourceTool" {
			if uri, ok := input["uri"].(string); ok {
				return strings.ReplaceAll(rules.Default, "{uri}", uri)
			}
		}

		// Handle analyze tool
		if toolName == "mcp__serena__analyze" {
			if task, ok := input["task"].(string); ok {
				// Check patterns
				for _, pattern := range rules.Patterns {
					if strings.Contains(task, pattern.Contains) {
						return pattern.Message
					}
				}
				return strings.ReplaceAll(rules.Default, "{task}", task)
			}
		}

		// Handle activate project tool
		if toolName == "mcp__serena__activate_project" {
			if projectName, ok := input["project_name"].(string); ok {
				return strings.ReplaceAll(rules.Default, "{project_name}", projectName)
			}
		}

		// Handle write memory tool
		if toolName == "mcp__serena__write_memory" {
			if memoryFile, ok := input["memory_file_name"].(string); ok {
				return strings.ReplaceAll(rules.Default, "{memory_file_name}", memoryFile)
			}
		}

		// Handle delete memory tool
		if toolName == "mcp__serena__delete_memory" {
			if memoryFile, ok := input["memory_file_name"].(string); ok {
				return strings.ReplaceAll(rules.Default, "{memory_file_name}", memoryFile)
			}
		}

		// Handle create text file tool
		if toolName == "mcp__serena__create_text_file" {
			if filename, ok := input["filename"].(string); ok {
				// Check patterns
				for _, pattern := range rules.Patterns {
					if strings.Contains(filename, pattern.Contains) {
						return strings.ReplaceAll(pattern.Message, "{filename}", filename)
					}
				}
				return strings.ReplaceAll(rules.Default, "{filename}", filename)
			}
		}

		// Handle delete lines tool
		if toolName == "mcp__serena__delete_lines" {
			filePath, _ := input["file_path"].(string)
			startLine, _ := input["start_line"].(float64)
			endLine, _ := input["end_line"].(float64)
			msg := strings.ReplaceAll(rules.Default, "{file_path}", filePath)
			msg = strings.ReplaceAll(msg, "{start_line}", fmt.Sprintf("%.0f", startLine))
			msg = strings.ReplaceAll(msg, "{end_line}", fmt.Sprintf("%.0f", endLine))
			return msg
		}

		// Handle replace lines tool
		if toolName == "mcp__serena__replace_lines" {
			filePath, _ := input["file_path"].(string)
			startLine, _ := input["start_line"].(float64)
			endLine, _ := input["end_line"].(float64)
			msg := strings.ReplaceAll(rules.Default, "{file_path}", filePath)
			msg = strings.ReplaceAll(msg, "{start_line}", fmt.Sprintf("%.0f", startLine))
			msg = strings.ReplaceAll(msg, "{end_line}", fmt.Sprintf("%.0f", endLine))
			return msg
		}

		// Handle insert at line tool
		if toolName == "mcp__serena__insert_at_line" {
			filePath, _ := input["file_path"].(string)
			line, _ := input["line"].(float64)
			msg := strings.ReplaceAll(rules.Default, "{file_path}", filePath)
			msg = strings.ReplaceAll(msg, "{line}", fmt.Sprintf("%.0f", line))
			return msg
		}

		// Handle symbol-related tools
		if toolName == "mcp__serena__find_referencing_code_snippets" ||
			toolName == "mcp__serena__find_referencing_symbols" ||
			toolName == "mcp__serena__insert_after_symbol" ||
			toolName == "mcp__serena__insert_before_symbol" ||
			toolName == "mcp__serena__replace_symbol_body" {
			if symbolName, ok := input["symbol_name"].(string); ok {
				return strings.ReplaceAll(rules.Default, "{symbol_name}", symbolName)
			}
		}

		// Handle execute shell command tool
		if toolName == "mcp__serena__execute_shell_command" {
			if cmd, ok := input["command"].(string); ok {
				// Check patterns
				for _, pattern := range rules.Patterns {
					if strings.Contains(cmd, pattern.Contains) {
						return pattern.Message
					}
				}
			}
			return rules.Default
		}

		// Handle read file tool
		if toolName == "mcp__serena__read_file" {
			if filePath, ok := input["file_path"].(string); ok {
				ext := filepath.Ext(filePath)
				// Check extensions
				if message, ok := rules.Extensions[ext]; ok {
					return strings.ReplaceAll(message, "{file_path}", filePath)
				}
				return strings.ReplaceAll(rules.Default, "{file_path}", filePath)
			}
		}

		// Handle switch modes tool
		if toolName == "mcp__serena__switch_modes" {
			if modes, ok := input["modes"].([]interface{}); ok {
				modeList := make([]string, len(modes))
				for i, mode := range modes {
					if m, ok := mode.(string); ok {
						modeList[i] = m
					}
				}
				return strings.ReplaceAll(rules.Default, "{modes}", strings.Join(modeList, ", "))
			}
		}
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

// NarrateToolUsePermission narrates a tool permission request using config rules
func (cn *ConfigBasedNarrator) NarrateToolUsePermission(toolName string) string {
	// Check if there's a specific permission message for this tool
	if rules, ok := cn.config.Rules[toolName]; ok {
		if rules.PermissionMessage != "" {
			return rules.PermissionMessage
		}
	}

	// Check default config
	if rules, ok := cn.defaultConfig.Rules[toolName]; ok {
		if rules.PermissionMessage != "" {
			return rules.PermissionMessage
		}
	}

	// Use generic permission message
	template := cn.getStringOrDefault(cn.config.Messages.GenericToolPermission, cn.defaultConfig.Messages.GenericToolPermission)
	if template != "" {
		return strings.ReplaceAll(template, "{tool}", toolName)
	}

	// Final fallback
	return fmt.Sprintf("%sの使用許可を求めています", toolName)
}

// NarrateText returns the text as-is
func (cn *ConfigBasedNarrator) NarrateText(text string) string {
	return text
}

// NarrateNotification narrates notification events
func (cn *ConfigBasedNarrator) NarrateNotification(notificationType NotificationType) string {
	// Return messages based on notification type
	switch notificationType {
	case NotificationTypeCompact:
		return "コンテキストを圧縮しています"
	case NotificationTypeSessionStartStartup:
		return "こんにちは！何かお手伝いできることはありますか？"
	case NotificationTypeSessionStartClear:
		return "何かお手伝いできることはありますか？"
	case NotificationTypeSessionStartResume:
		return "前回の作業を続けましょう。どこから再開しますか？"
	default:
		return ""
	}
}
