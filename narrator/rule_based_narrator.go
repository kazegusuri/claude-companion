package narrator

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RuleBasedNarrator uses configuration file for narrative rules
type RuleBasedNarrator struct {
	config        *NarratorConfig
	defaultConfig *NarratorConfig
}

// NewRuleBasedNarrator creates a new rule-based narrator
func NewRuleBasedNarrator(config *NarratorConfig) *RuleBasedNarrator {
	return &RuleBasedNarrator{
		config:        config,
		defaultConfig: GetDefaultNarratorConfig(),
	}
}

// getFileTypeName returns the file type name for a given extension
func (cn *RuleBasedNarrator) getFileTypeName(ext string) string {
	// First check user config
	if cn.config.FileTypeNames != nil {
		if name, ok := cn.config.FileTypeNames[ext]; ok {
			return name
		}
	}
	// Then check default config
	if cn.defaultConfig.FileTypeNames != nil {
		if name, ok := cn.defaultConfig.FileTypeNames[ext]; ok {
			return name
		}
	}
	// Return empty string if not found
	return ""
}

// parseMCPToolName parses an MCP tool name into server and operation
func parseMCPToolName(toolName string) (server string, operation string, isMCP bool) {
	if !strings.HasPrefix(toolName, "mcp__") {
		return "", "", false
	}
	// Remove mcp__ prefix
	name := strings.TrimPrefix(toolName, "mcp__")
	// Split by first underscore
	parts := strings.SplitN(name, "__", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	// Legacy format: mcp__server_operation
	underscoreIndex := strings.Index(name, "_")
	if underscoreIndex > 0 {
		return name[:underscoreIndex], name[underscoreIndex+1:], true
	}
	return "", "", false
}

// getStringOrDefault returns the value from config if not empty, otherwise from defaultConfig
func (cn *RuleBasedNarrator) getStringOrDefault(configValue, defaultValue string) string {
	if configValue != "" {
		return configValue
	}
	return defaultValue
}

// applyCaptures applies capture rules to a template string using input values
func (cn *RuleBasedNarrator) applyCaptures(template string, captures []CaptureRule, input map[string]interface{}) string {
	result := template
	for _, capture := range captures {
		if value, exists := input[capture.InputKey]; exists {
			// Convert value to string
			var strValue string
			switch v := value.(type) {
			case string:
				strValue = v
			case float64:
				strValue = fmt.Sprintf("%.0f", v)
			case int:
				strValue = fmt.Sprintf("%d", v)
			case []interface{}:
				// For arrays, join with comma
				strList := make([]string, len(v))
				for i, item := range v {
					if s, ok := item.(string); ok {
						strList[i] = s
					} else {
						strList[i] = fmt.Sprintf("%v", item)
					}
				}
				strValue = strings.Join(strList, ", ")
			default:
				strValue = fmt.Sprintf("%v", v)
			}
			// Automatically generate placeholder from inputKey
			placeholder := fmt.Sprintf("{%s}", capture.InputKey)
			result = strings.ReplaceAll(result, placeholder, strValue)

			// If parseFileType is true, also replace {filetype}
			if capture.ParseFileType && strValue != "" {
				ext := filepath.Ext(strValue)
				fileTypeName := cn.getFileTypeName(ext)
				if fileTypeName == "" {
					fileTypeName = "ファイル" // Default for unknown extensions
				}
				result = strings.ReplaceAll(result, "{filetype}", fileTypeName)
			}
		}
	}
	return result
}

// processDefaultWithCaptures processes a default message with capture rules if available
func (cn *RuleBasedNarrator) processDefaultWithCaptures(rules ToolRules, input map[string]interface{}) string {
	if rules.Default == "" {
		return ""
	}

	// If captures are configured, use them
	if len(rules.Captures) > 0 {
		return cn.applyCaptures(rules.Default, rules.Captures, input)
	}

	// Fallback to the original default without replacement
	return rules.Default
}

// handleGenericMCPTool handles MCP tools using configuration-driven approach
func (cn *RuleBasedNarrator) handleGenericMCPTool(toolName string, rules ToolRules, input map[string]interface{}) string {
	// First try patterns if available
	for _, pattern := range rules.Patterns {
		for _, value := range input {
			if strValue, ok := value.(string); ok {
				if strings.Contains(strValue, pattern.Contains) {
					// Apply captures to pattern message
					if len(rules.Captures) > 0 {
						return cn.applyCaptures(pattern.Message, rules.Captures, input)
					}
					return pattern.Message
				}
			}
		}
	}

	// Then try captures with default message
	if len(rules.Captures) > 0 {
		return cn.applyCaptures(rules.Default, rules.Captures, input)
	}

	// Final fallback
	return rules.Default
}

// NarrateToolUse converts tool usage to natural Japanese using config rules
func (cn *RuleBasedNarrator) NarrateToolUse(toolName string, input map[string]interface{}) (string, bool) {
	// Handle MCP tools first with new MCPRules structure
	if server, operation, isMCP := parseMCPToolName(toolName); isMCP {
		// Check if we have MCPRules for this server
		var mcpRules MCPRules
		var found bool

		// First check user config
		if cn.config.MCPRules != nil {
			mcpRules, found = cn.config.MCPRules[server]
		}

		// Then check default config
		if !found && cn.defaultConfig.MCPRules != nil {
			mcpRules, found = cn.defaultConfig.MCPRules[server]
		}

		if found {
			// Check if we have specific rules for this operation
			if operationRules, ok := mcpRules.Rules[operation]; ok {
				if len(operationRules.Captures) > 0 {
					return cn.applyCaptures(operationRules.Default, operationRules.Captures, input), false
				}
				return operationRules.Default, false
			}

			// Use server default
			if mcpRules.Default != "" {
				return strings.ReplaceAll(mcpRules.Default, "{operation}", operation), false
			}
		}

		// MCP tool but no rules found - return empty string for fallback
		return "", true
	}

	rules, ok := cn.config.Rules[toolName]
	if !ok {
		// Try default config
		if defaultRules, ok := cn.defaultConfig.Rules[toolName]; ok {
			rules = defaultRules
		} else {
			// No rules for this tool in both configs
			template := cn.getStringOrDefault(cn.config.Messages.GenericToolExecution, cn.defaultConfig.Messages.GenericToolExecution)
			if template != "" {
				return strings.ReplaceAll(template, "{tool}", toolName), false
			}
			// Return empty string for fallback
			return "", true
		}
	}

	// Handle tool-specific logic
	switch toolName {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			// Check prefixes
			for _, prefix := range rules.Prefixes {
				if strings.HasPrefix(cmd, prefix.Prefix) {
					return prefix.Message, false
				}
			}

			// Use default if no prefix matches
			if rules.Default != "" {
				// Extract first word as command name
				cmdParts := strings.Fields(cmd)
				if len(cmdParts) > 0 {
					return strings.ReplaceAll(rules.Default, "{command}", cmdParts[0]), false
				}
			}
		}
		return "", true

	case "Read", "Write", "Edit", "NotebookRead", "NotebookEdit":
		var filePath string
		if path, ok := input["file_path"].(string); ok {
			filePath = path
		} else if path, ok := input["notebook_path"].(string); ok {
			filePath = path
		}

		if filePath != "" {
			fileName := filepath.Base(filePath)

			// Add filename to input for captures
			inputWithFilename := make(map[string]interface{})
			for k, v := range input {
				inputWithFilename[k] = v
			}
			inputWithFilename["filename"] = fileName

			// Always use applyCaptures
			return cn.applyCaptures(rules.Default, rules.Captures, inputWithFilename), false
		}

	case "MultiEdit":
		if path, ok := input["file_path"].(string); ok {
			fileName := filepath.Base(path)
			if edits, ok := input["edits"].([]interface{}); ok {
				count := len(edits)
				msg := strings.ReplaceAll(rules.Default, "{filename}", fileName)
				msg = strings.ReplaceAll(msg, "{count}", fmt.Sprintf("%d", count))
				return msg, false
			}
			return strings.ReplaceAll(rules.Default, "{filename}", fileName), false
		}

	case "Grep":
		// Always use configuration-driven approach
		// Create a copy of input with default path if not present
		modifiedInput := make(map[string]interface{})
		for k, v := range input {
			modifiedInput[k] = v
		}
		// Set default path if not present
		if _, hasPath := modifiedInput["path"]; !hasPath {
			modifiedInput["path"] = "プロジェクト全体"
		}
		return cn.handleGenericMCPTool(toolName, rules, modifiedInput), false

	case "Glob":
		// Use configuration-driven approach if captures are configured
		if len(rules.Captures) > 0 {
			return cn.handleGenericMCPTool(toolName, rules, input), false
		}

		// Fallback to hardcoded logic
		if pattern, ok := input["pattern"].(string); ok {
			// Check patterns
			for _, rule := range rules.Patterns {
				if strings.Contains(pattern, rule.Contains) {
					return rule.Message, false
				}
			}

			// Use default
			return strings.ReplaceAll(rules.Default, "{pattern}", pattern), false
		}

	case "LS":
		if path, ok := input["path"].(string); ok {
			dirName := filepath.Base(path)
			if dirName == "." || dirName == "/" {
				msg := cn.getStringOrDefault(cn.config.Messages.CurrentDirectory, cn.defaultConfig.Messages.CurrentDirectory)
				if msg != "" {
					return msg, false
				}
				// Return empty string for fallback
				return "", true
			}
			return strings.ReplaceAll(rules.Default, "{dirname}", dirName), false
		}
		msg := cn.getStringOrDefault(cn.config.Messages.DirectoryContents, cn.defaultConfig.Messages.DirectoryContents)
		if msg != "" {
			return msg, false
		}
		// Return empty string for fallback
		return "", true

	case "WebFetch":
		if url, ok := input["url"].(string); ok {
			// Check patterns
			for _, rule := range rules.Patterns {
				if strings.Contains(url, rule.Contains) {
					return rule.Message, false
				}
			}

			// Use default
			domain := extractDomain(url)
			return strings.ReplaceAll(rules.Default, "{domain}", domain), false
		}

	case "WebSearch":
		if query, ok := input["query"].(string); ok {
			return strings.ReplaceAll(rules.Default, "{query}", query), false
		}

	case "Task":
		// Get basic info
		desc, hasDesc := input["description"].(string)
		prompt, hasPrompt := input["prompt"].(string)
		subagentType, hasSubagentType := input["subagent_type"].(string)

		// Handle slash command first
		if hasPrompt && strings.HasPrefix(prompt, "/") {
			cmd := strings.Fields(prompt)[0]
			template := cn.getStringOrDefault(cn.config.Messages.GenericCommandExecution, cn.defaultConfig.Messages.GenericCommandExecution)
			if template != "" {
				return strings.ReplaceAll(template, "{command}", cmd), false
			}
			// Return empty string for fallback
			return "", true
		}

		// Build message based on available info
		if hasSubagentType && subagentType != "" && hasDesc {
			// When subagent_type is not empty, include agent type and description
			return fmt.Sprintf("%s agentでタスク「%s」を実行します", subagentType, desc), false
		} else if hasDesc {
			// Just description
			return strings.ReplaceAll(rules.Default, "{description}", desc), false
		}

		// Default message
		msg := cn.getStringOrDefault(cn.config.Messages.ComplexTask, cn.defaultConfig.Messages.ComplexTask)
		if msg != "" {
			return msg, false
		}
		// Return empty string for fallback
		return "", true

	case "TodoWrite":
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
			return msg, false
		}
		msg := cn.getStringOrDefault(cn.config.Messages.TodoListUpdate, cn.defaultConfig.Messages.TodoListUpdate)
		if msg != "" {
			return msg, false
		}
		// Return empty string for fallback
		return "", true
	}

	// Handle tools with simple default messages
	if rules.Default != "" {
		// Check if captures are configured
		if len(rules.Captures) > 0 {
			return cn.applyCaptures(rules.Default, rules.Captures, input), false
		}
		return rules.Default, false
	}

	// Generic fallback
	template := cn.getStringOrDefault(cn.config.Messages.GenericToolExecution, cn.defaultConfig.Messages.GenericToolExecution)
	if template != "" {
		return strings.ReplaceAll(template, "{tool}", toolName), false
	}
	// Return empty string for fallback
	return "", true
}

// NarrateToolUsePermission narrates a tool permission request using config rules
func (cn *RuleBasedNarrator) NarrateToolUsePermission(toolName string) (string, bool) {
	// Check if there's a specific permission message for this tool
	if rules, ok := cn.config.Rules[toolName]; ok {
		if rules.PermissionMessage != "" {
			return rules.PermissionMessage, false
		}
	}

	// Check default config
	if rules, ok := cn.defaultConfig.Rules[toolName]; ok {
		if rules.PermissionMessage != "" {
			return rules.PermissionMessage, false
		}
	}

	// Use generic permission message
	template := cn.getStringOrDefault(cn.config.Messages.GenericToolPermission, cn.defaultConfig.Messages.GenericToolPermission)
	if template != "" {
		return strings.ReplaceAll(template, "{tool}", toolName), false
	}

	// Final fallback
	return fmt.Sprintf("%sの使用許可を求めています", toolName), true
}

// NarrateText returns the text as-is
func (cn *RuleBasedNarrator) NarrateText(text string, isThinking bool) (string, bool) {
	return text, true
}

// NarrateNotification narrates notification events
func (cn *RuleBasedNarrator) NarrateNotification(notificationType NotificationType) (string, bool) {
	// Return messages based on notification type
	switch notificationType {
	case NotificationTypeCompact:
		return "コンテキストを圧縮しています", false
	case NotificationTypeSessionStartStartup:
		return "こんにちは！何かお手伝いできることはありますか？", false
	case NotificationTypeSessionStartClear:
		return "何かお手伝いできることはありますか？", false
	case NotificationTypeSessionStartResume:
		return "前回の作業を続けましょう。どこから再開しますか？", false
	case NotificationTypeSessionStartCompact:
		return "セッションを再開しました", false
	default:
		return "", true
	}
}

// NarrateTaskCompletion narrates task completion events
func (cn *RuleBasedNarrator) NarrateTaskCompletion(description string, subagentType string) (string, bool) {
	// Build message based on available information
	if subagentType != "" && description != "" {
		return fmt.Sprintf("%s agentがタスク「%s」を完了しました", subagentType, description), false
	} else if description != "" {
		return fmt.Sprintf("タスク「%s」が完了しました", description), false
	}
	return "タスクが完了しました", false
}
