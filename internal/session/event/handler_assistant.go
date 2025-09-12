package event

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

// convertAssistantMessage converts session AssistantMessage to central AssistantMessage
func (h *Handler) convertAssistantMessage(msg *AssistantMessage) *internalevent.AssistantMessage {
	// Convert SessionMessageBase
	base := internalevent.SessionMessageBase{
		UUID:        msg.UUID,
		Type:        internalevent.MessageTypeAssistant,
		IsSidechain: msg.IsSidechain,
		CWD:         msg.CWD,
		Timestamp:   msg.Timestamp,
		IsMeta:      false, // AssistantMessage doesn't have IsMeta in session event
	}

	// Convert content
	var content internalevent.AssistantMessageContent

	// Check if this is an API error message
	if msg.IsApiErrorMessage {
		// Parse API error from content
		content = h.parseAPIErrorContent(msg.Message.Content)
	} else if len(msg.Message.Content) == 1 {
		// Single content item
		content = h.convertSingleAssistantContent(msg.Message.Content[0])
	} else if len(msg.Message.Content) > 1 {
		// Multiple content items - create a list
		items := make([]internalevent.AssistantMessageContentItem, 0, len(msg.Message.Content))
		for _, c := range msg.Message.Content {
			if item := h.convertAssistantContentItem(c); item != nil {
				items = append(items, item)
			}
		}
		content = &internalevent.AssistantMessageContentList{
			Items: items,
		}
	}

	// Convert token usage
	var usage *internalevent.TokenUsage
	if msg.Message.Usage.InputTokens > 0 || msg.Message.Usage.OutputTokens > 0 {
		usage = &internalevent.TokenUsage{
			InputTokens:              msg.Message.Usage.InputTokens,
			OutputTokens:             msg.Message.Usage.OutputTokens,
			CacheCreationInputTokens: msg.Message.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     msg.Message.Usage.CacheReadInputTokens,
			ServiceTier:              msg.Message.Usage.ServiceTier,
		}
	}

	// Get transcript path from session if available
	transcriptPath := ""
	if msg.Session != nil {
		transcriptPath = msg.Session.Path
	}

	return &internalevent.AssistantMessage{
		SessionMessageBase: base,
		Session: internalevent.Session{
			SessionID:      msg.SessionID,
			TranscriptPath: transcriptPath,
		},
		RequestID: msg.RequestID,
		Message: internalevent.AssistantMessageData{
			ID:         msg.Message.ID,
			Type:       msg.Message.Type,
			Role:       msg.Message.Role,
			Model:      msg.Message.Model,
			Content:    content,
			StopReason: msg.Message.StopReason,
			StopSeq:    msg.Message.StopSequence,
			Usage:      usage,
		},
		IsApiErrorMessage: msg.IsApiErrorMessage,
	}
}

// convertSingleAssistantContent converts a single assistant content item
func (h *Handler) convertSingleAssistantContent(content AssistantContent) internalevent.AssistantMessageContent {
	switch content.Type {
	case "text", "thinking":
		// Get the text content (from Text field for "text" type, or Thinking field for "thinking" type)
		textContent := content.Text
		if content.Type == "thinking" {
			textContent = content.Thinking
		}

		// Extract code blocks from text
		codeBlocks := extractCodeBlocks(textContent)
		processedText := textContent

		// Replace code blocks with placeholders
		if len(codeBlocks) > 0 {
			processedText = strings.TrimSpace(textContent)
			for i, block := range codeBlocks {
				placeholder := fmt.Sprintf("[CODE BLOCK %d: %s]", i+1, block.Language)
				// Find and replace the original code block
				original := fmt.Sprintf("```%s\n%s```", block.Language, block.Content)
				if block.Language == "text" || block.Language == "" {
					original = fmt.Sprintf("```\n%s```", block.Content)
				}
				processedText = strings.Replace(processedText, original, placeholder, 1)
			}
		}

		return &internalevent.AssistantMessageContentText{
			Type:          content.Type,
			Text:          textContent,
			IsThinking:    content.Type == "thinking",
			ProcessedText: processedText,
			CodeBlocks:    codeBlocks,
		}
	default:
		// For other single content types, wrap in a list with one item
		if item := h.convertAssistantContentItem(content); item != nil {
			return &internalevent.AssistantMessageContentList{
				Items: []internalevent.AssistantMessageContentItem{item},
			}
		}
		return nil
	}
}

// convertAssistantContentItem converts an assistant content item for use in a list
func (h *Handler) convertAssistantContentItem(content AssistantContent) internalevent.AssistantMessageContentItem {
	switch content.Type {
	case "text", "thinking":
		// Get the text content (from Text field for "text" type, or Thinking field for "thinking" type)
		textContent := content.Text
		if content.Type == "thinking" {
			textContent = content.Thinking
		}

		// Extract code blocks from text
		codeBlocks := extractCodeBlocks(textContent)
		processedText := textContent

		// Replace code blocks with placeholders
		if len(codeBlocks) > 0 {
			processedText = strings.TrimSpace(textContent)
			for i, block := range codeBlocks {
				placeholder := fmt.Sprintf("[CODE BLOCK %d: %s]", i+1, block.Language)
				// Find and replace the original code block
				original := fmt.Sprintf("```%s\n%s```", block.Language, block.Content)
				if block.Language == "text" || block.Language == "" {
					original = fmt.Sprintf("```\n%s```", block.Content)
				}
				processedText = strings.Replace(processedText, original, placeholder, 1)
			}
		}

		return &internalevent.AssistantMessageContentText{
			Type:          content.Type,
			Text:          textContent,
			IsThinking:    content.Type == "thinking",
			ProcessedText: processedText,
			CodeBlocks:    codeBlocks,
		}
	case "tool_use":
		return &internalevent.AssistantMessageContentToolUse{
			Type:  content.Type,
			ID:    content.ID,
			Name:  content.Name,
			Input: h.convertToolInput(content.Name, content.Input),
		}
	default:
		// Unknown content type
		return nil
	}
}

// parseAPIErrorContent parses API error from assistant message content
func (h *Handler) parseAPIErrorContent(contents []AssistantContent) internalevent.AssistantMessageContent {
	// Look for text content containing API error
	for _, content := range contents {
		if content.Type == "text" && content.Text != "" {
			// Try to parse API error from text
			statusCode := 0
			errorType := ""
			errorMessage := ""
			rawText := content.Text

			// Extract status code and JSON from the error message
			// Format: "API Error: 500 {\"type\":\"error\",\"error\":{...}}"
			if strings.HasPrefix(content.Text, "API Error: ") {
				parts := strings.SplitN(content.Text, " ", 4)
				if len(parts) >= 3 {
					if code, err := strconv.Atoi(parts[2]); err == nil {
						statusCode = code
					}
				}
			}

			// Try to parse JSON
			if jsonStartIdx := strings.Index(content.Text, "{"); jsonStartIdx != -1 {
				jsonStr := content.Text[jsonStartIdx:]
				var apiError APIError
				if err := json.Unmarshal([]byte(jsonStr), &apiError); err == nil {
					errorType = apiError.Error.Type
					errorMessage = apiError.Error.Message
				}
			}

			return &internalevent.APIErrorMessageContent{
				StatusCode: statusCode,
				ErrorType:  errorType,
				Error: internalevent.APIErrorDetail{
					Type:    errorType,
					Message: errorMessage,
				},
				RawText: rawText,
			}
		}
	}

	// If no error content found, return nil
	return nil
}

// convertToolInput converts tool input to appropriate typed structure
func (h *Handler) convertToolInput(toolName string, input interface{}) internalevent.AssistantMessageContentToolUseInput {
	// Convert input to map if possible
	inputMap, ok := input.(map[string]interface{})
	if !ok {
		// If not a map, return generic
		return &internalevent.ToolUseGeneric{
			Data: map[string]interface{}{"input": input},
		}
	}

	// Handle MCP tools
	if strings.HasPrefix(toolName, "mcp__") {
		parts := strings.SplitN(toolName, "__", 3)
		server := ""
		tool := toolName
		if len(parts) >= 3 {
			server = parts[1]
			tool = parts[2]
		}
		return &internalevent.ToolUseMCP{
			Server: server,
			Tool:   tool,
			Data:   inputMap,
		}
	}

	// Convert based on tool name
	switch toolName {
	case "TodoWrite":
		result := &internalevent.ToolUseTodoWrite{}
		if todos, ok := inputMap["todos"].([]interface{}); ok {
			for _, todo := range todos {
				if todoMap, ok := todo.(map[string]interface{}); ok {
					item := internalevent.TodoItem{}
					if content, ok := todoMap["content"].(string); ok {
						item.Content = content
					}
					if status, ok := todoMap["status"].(string); ok {
						item.Status = status
					}
					if activeForm, ok := todoMap["activeForm"].(string); ok {
						item.ActiveForm = activeForm
					}
					result.Todos = append(result.Todos, item)
				}
			}
		}
		return result

	case "Bash":
		result := &internalevent.ToolUseBash{}
		if cmd, ok := inputMap["command"].(string); ok {
			result.Command = cmd
		}
		if desc, ok := inputMap["description"].(string); ok {
			result.Description = desc
		}
		if bg, ok := inputMap["run_in_background"].(bool); ok {
			result.RunInBackground = bg
		}
		if timeout, ok := inputMap["timeout"].(float64); ok {
			result.Timeout = int(timeout)
		}
		return result

	case "Read":
		result := &internalevent.ToolUseRead{}
		if path, ok := inputMap["file_path"].(string); ok {
			result.FilePath = path
		}
		if limit, ok := inputMap["limit"].(float64); ok {
			result.Limit = int(limit)
		}
		if offset, ok := inputMap["offset"].(float64); ok {
			result.Offset = int(offset)
		}
		return result

	case "Write":
		result := &internalevent.ToolUseWrite{}
		if path, ok := inputMap["file_path"].(string); ok {
			result.FilePath = path
		}
		if content, ok := inputMap["content"].(string); ok {
			result.Content = content
		}
		return result

	case "Edit":
		result := &internalevent.ToolUseEdit{}
		if path, ok := inputMap["file_path"].(string); ok {
			result.FilePath = path
		}
		if old, ok := inputMap["old_string"].(string); ok {
			result.OldString = old
		}
		if new, ok := inputMap["new_string"].(string); ok {
			result.NewString = new
		}
		if replaceAll, ok := inputMap["replace_all"].(bool); ok {
			result.ReplaceAll = replaceAll
		}
		return result

	case "MultiEdit":
		result := &internalevent.ToolUseMultiEdit{}
		if path, ok := inputMap["file_path"].(string); ok {
			result.FilePath = path
		}
		if edits, ok := inputMap["edits"].([]interface{}); ok {
			for _, edit := range edits {
				if editMap, ok := edit.(map[string]interface{}); ok {
					item := internalevent.EditItem{}
					if old, ok := editMap["old_string"].(string); ok {
						item.OldString = old
					}
					if new, ok := editMap["new_string"].(string); ok {
						item.NewString = new
					}
					if replaceAll, ok := editMap["replace_all"].(bool); ok {
						item.ReplaceAll = replaceAll
					}
					result.Edits = append(result.Edits, item)
				}
			}
		}
		return result

	case "Grep":
		result := &internalevent.ToolUseGrep{}
		if pattern, ok := inputMap["pattern"].(string); ok {
			result.Pattern = pattern
		}
		if path, ok := inputMap["path"].(string); ok {
			result.Path = path
		}
		if glob, ok := inputMap["glob"].(string); ok {
			result.Glob = glob
		}
		if typeStr, ok := inputMap["type"].(string); ok {
			result.Type = typeStr
		}
		if mode, ok := inputMap["output_mode"].(string); ok {
			result.OutputMode = mode
		}
		if after, ok := inputMap["-A"].(float64); ok {
			result.ContextAfter = int(after)
		}
		if before, ok := inputMap["-B"].(float64); ok {
			result.ContextBefore = int(before)
		}
		if context, ok := inputMap["-C"].(float64); ok {
			result.Context = int(context)
		}
		if caseInsensitive, ok := inputMap["-i"].(bool); ok {
			result.CaseInsensitive = caseInsensitive
		}
		if lineNumbers, ok := inputMap["-n"].(bool); ok {
			result.ShowLineNumbers = lineNumbers
		}
		if limit, ok := inputMap["head_limit"].(float64); ok {
			result.HeadLimit = int(limit)
		}
		if multiline, ok := inputMap["multiline"].(bool); ok {
			result.Multiline = multiline
		}
		return result

	case "Glob":
		result := &internalevent.ToolUseGlob{}
		if pattern, ok := inputMap["pattern"].(string); ok {
			result.Pattern = pattern
		}
		if path, ok := inputMap["path"].(string); ok {
			result.Path = path
		}
		return result

	case "Task":
		result := &internalevent.ToolUseTask{}
		if desc, ok := inputMap["description"].(string); ok {
			result.Description = desc
		}
		if prompt, ok := inputMap["prompt"].(string); ok {
			result.Prompt = prompt
		}
		if subagent, ok := inputMap["subagent_type"].(string); ok {
			result.SubagentType = subagent
		}
		return result

	case "WebFetch":
		result := &internalevent.ToolUseWebFetch{}
		if url, ok := inputMap["url"].(string); ok {
			result.URL = url
		}
		if prompt, ok := inputMap["prompt"].(string); ok {
			result.Prompt = prompt
		}
		return result

	case "WebSearch":
		result := &internalevent.ToolUseWebSearch{}
		if query, ok := inputMap["query"].(string); ok {
			result.Query = query
		}
		if allowed, ok := inputMap["allowed_domains"].([]interface{}); ok {
			for _, domain := range allowed {
				if d, ok := domain.(string); ok {
					result.AllowedDomains = append(result.AllowedDomains, d)
				}
			}
		}
		if blocked, ok := inputMap["blocked_domains"].([]interface{}); ok {
			for _, domain := range blocked {
				if d, ok := domain.(string); ok {
					result.BlockedDomains = append(result.BlockedDomains, d)
				}
			}
		}
		return result

	case "NotebookEdit":
		result := &internalevent.ToolUseNotebookEdit{}
		if path, ok := inputMap["notebook_path"].(string); ok {
			result.NotebookPath = path
		}
		if cellID, ok := inputMap["cell_id"].(string); ok {
			result.CellID = cellID
		}
		if cellType, ok := inputMap["cell_type"].(string); ok {
			result.CellType = cellType
		}
		if editMode, ok := inputMap["edit_mode"].(string); ok {
			result.EditMode = editMode
		}
		if source, ok := inputMap["new_source"].(string); ok {
			result.NewSource = source
		}
		return result

	case "ExitPlanMode":
		result := &internalevent.ToolUseExitPlanMode{}
		if plan, ok := inputMap["plan"].(string); ok {
			result.Plan = plan
		}
		return result

	case "BashOutput":
		result := &internalevent.ToolUseBashOutput{}
		if bashID, ok := inputMap["bash_id"].(string); ok {
			result.BashID = bashID
		}
		if filter, ok := inputMap["filter"].(string); ok {
			result.Filter = filter
		}
		return result

	case "KillBash":
		result := &internalevent.ToolUseKillBash{}
		if shellID, ok := inputMap["shell_id"].(string); ok {
			result.ShellID = shellID
		}
		return result

	default:
		// Unknown tool, use generic
		return &internalevent.ToolUseGeneric{
			Data: inputMap,
		}
	}
}

// extractCodeBlocks extracts code blocks from text
func extractCodeBlocks(text string) []internalevent.CodeBlock {
	blocks := []internalevent.CodeBlock{}

	// Match fenced code blocks with optional language
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		language := match[1]
		if language == "" {
			language = "text"
		}
		blocks = append(blocks, internalevent.CodeBlock{
			Language: language,
			Content:  match[2],
		})
	}

	return blocks
}
