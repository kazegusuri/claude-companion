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
			Input: content.Input,
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
