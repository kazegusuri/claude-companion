package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kazegusuri/claude-companion/narrator"
)

const (
	// MaxMainTextLines is the maximum number of lines to show for main text with placeholders
	MaxMainTextLines = 10
	// MaxCodePreviewLines is the maximum number of lines to show in code block preview
	MaxCodePreviewLines = 5
	// MaxNormalTextLines is the maximum number of lines to show for normal text without code blocks
	MaxNormalTextLines = 5
)

// CompanionFormatter provides enhanced formatting for a more companion-like experience
type CompanionFormatter struct {
	fileOperations []string
	currentTool    string
	// Track statistics for the session
	totalFiles int
	totalTools int
	// Narrator for natural language descriptions
	narrator narrator.Narrator
}

// NewCompanionFormatter creates a new CompanionFormatter instance
func NewCompanionFormatter(narrator narrator.Narrator) *CompanionFormatter {
	return &CompanionFormatter{
		fileOperations: make([]string, 0),
		narrator:       narrator,
	}
}

// ExtractCodeBlocks extracts code blocks from text content
func (f *CompanionFormatter) ExtractCodeBlocks(text string) []CodeBlock {
	blocks := []CodeBlock{}

	// Match fenced code blocks with optional language
	codeBlockRegex := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		language := match[1]
		if language == "" {
			language = "text"
		}
		blocks = append(blocks, CodeBlock{
			Language: language,
			Content:  match[2],
		})
	}

	return blocks
}

// FormatToolUse formats tool usage for companion display
func (f *CompanionFormatter) FormatToolUse(toolName, toolID string, input map[string]interface{}) string {
	f.currentTool = toolName

	var output strings.Builder

	// Use narrator
	narration := f.narrator.NarrateToolUse(toolName, input)
	if narration != "" {
		output.WriteString(fmt.Sprintf("\n  💬 %s", narration))
		// Track file operations for summary
		if toolName == "Read" || toolName == "Write" || toolName == "Edit" || toolName == "MultiEdit" {
			if path, ok := input["file_path"].(string); ok {
				f.fileOperations = append(f.fileOperations, fmt.Sprintf("%s: %s", toolName, path))
				f.totalFiles++
			}
		}
		f.totalTools++

		// Special handling for TodoWrite - show details even when narrator is used
		if toolName == "TodoWrite" {
			if todos, ok := input["todos"].([]interface{}); ok {
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
								emoji = "✅"
							case "in_progress":
								emoji = "🔄"
							case "pending":
								emoji = "⏳"
							}
							output.WriteString(fmt.Sprintf("\n    %d. %s %s", i+1, emoji, content))
						}
					}
				}
			}
		}

		return output.String()
	}

	// Fallback to emoji-based formatting if narrator is not available
	// Use emojis and formatting based on tool type
	switch toolName {
	case "Read", "mcp__ide__read":
		if filePath, ok := input["file_path"].(string); ok {
			f.fileOperations = append(f.fileOperations, fmt.Sprintf("Read: %s", filePath))
			output.WriteString(fmt.Sprintf("\n  📄 Reading file: %s", filePath))
		}
	case "Write":
		if filePath, ok := input["file_path"].(string); ok {
			f.fileOperations = append(f.fileOperations, fmt.Sprintf("Write: %s", filePath))
			output.WriteString(fmt.Sprintf("\n  ✏️  Writing file: %s", filePath))
		}
	case "Edit", "MultiEdit":
		if filePath, ok := input["file_path"].(string); ok {
			f.fileOperations = append(f.fileOperations, fmt.Sprintf("Edit: %s", filePath))
			output.WriteString(fmt.Sprintf("\n  ✂️  Editing file: %s", filePath))
		}
	case "Bash":
		if command, ok := input["command"].(string); ok {
			output.WriteString(fmt.Sprintf("\n  🖥️  Running command: %s", command))
		}
	case "Grep":
		if pattern, ok := input["pattern"].(string); ok {
			path, _ := input["path"].(string)
			if path == "" {
				path = "current directory"
			}
			output.WriteString(fmt.Sprintf("\n  🔍 Searching for '%s' in %s", pattern, path))
		}
	case "WebFetch":
		if url, ok := input["url"].(string); ok {
			output.WriteString(fmt.Sprintf("\n  🌐 Fetching: %s", url))
		}
	case "Task":
		if desc, ok := input["description"].(string); ok {
			output.WriteString(fmt.Sprintf("\n  🤖 Launching agent: %s", desc))
		}
	case "TodoWrite":
		output.WriteString("\n  ✅ Updating todo list")
		// Display todo list details
		if todos, ok := input["todos"].([]interface{}); ok {
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
							emoji = "✅"
						case "in_progress":
							emoji = "🔄"
						case "pending":
							emoji = "⏳"
						}
						output.WriteString(fmt.Sprintf("\n    %d. %s %s", i+1, emoji, content))
					}
				}
			}
		}
	default:
		if strings.HasPrefix(toolName, "mcp__") {
			// MCP tools
			output.WriteString(fmt.Sprintf("\n  🔧 MCP Tool: %s", toolName))
		} else {
			output.WriteString(fmt.Sprintf("\n  🔧 Tool: %s", toolName))
		}
	}

	// Show detailed input for debugging (optional)
	if len(input) > 0 && toolName != "TodoWrite" {
		output.WriteString(fmt.Sprintf(" (id: %s)", toolID))
	}

	return output.String()
}

// FormatAssistantText formats assistant text content with code block extraction
func (f *CompanionFormatter) FormatAssistantText(text string) string {
	var output strings.Builder

	// Extract code blocks
	codeBlocks := f.ExtractCodeBlocks(text)

	if len(codeBlocks) > 0 {
		// Replace code blocks with placeholders to show structure
		processedText := text
		for i, block := range codeBlocks {
			placeholder := fmt.Sprintf("[CODE BLOCK %d: %s]", i+1, block.Language)
			// Find and replace the original code block
			original := fmt.Sprintf("```%s\n%s```", block.Language, block.Content)
			if block.Language == "text" || block.Language == "" {
				original = fmt.Sprintf("```\n%s```", block.Content)
			}
			processedText = strings.Replace(processedText, original, placeholder, 1)
		}

		// Show the main text without code block placeholders
		lines := strings.Split(strings.TrimSpace(processedText), "\n")
		shownLines := 0
		for _, line := range lines {
			// Skip lines that are just code block placeholders
			if strings.HasPrefix(strings.TrimSpace(line), "[CODE BLOCK") && strings.HasSuffix(strings.TrimSpace(line), "]") {
				continue
			}
			if shownLines < MaxMainTextLines {
				output.WriteString(fmt.Sprintf("\n  %s", line))
				shownLines++
			} else {
				output.WriteString("\n  ... (text continues)")
				break
			}
		}

		// Show code blocks separately
		for i, block := range codeBlocks {
			output.WriteString(fmt.Sprintf("\n  📝 Code Block %d (%s):", i+1, block.Language))
			output.WriteString("\n    ```")
			// Show first few lines of code
			codeLines := strings.Split(strings.TrimSpace(block.Content), "\n")
			for j, line := range codeLines {
				if j < MaxCodePreviewLines {
					output.WriteString(fmt.Sprintf("\n    %s", line))
				} else if j == MaxCodePreviewLines && len(codeLines) > MaxCodePreviewLines+1 {
					output.WriteString(fmt.Sprintf("\n    ... (%d more lines)", len(codeLines)-MaxCodePreviewLines))
					break
				}
			}
			output.WriteString("\n    ```")
		}
	} else {
		// No code blocks, show text normally but truncated
		lines := strings.Split(strings.TrimSpace(text), "\n")
		for i, line := range lines {
			if i < MaxNormalTextLines {
				if i == 0 {
					// Use narrator to process the first line, then add 💬
					narrated := f.narrator.NarrateText(line)
					output.WriteString(fmt.Sprintf("\n  💬 %s", narrated))
				} else {
					output.WriteString(fmt.Sprintf("\n  %s", line))
				}
			} else if i == MaxNormalTextLines && len(lines) > MaxNormalTextLines+1 {
				output.WriteString(fmt.Sprintf("\n  ... (%d more lines)", len(lines)-MaxNormalTextLines))
				break
			}
		}
	}

	return output.String()
}

// GetFileSummary returns a summary of file operations performed
func (f *CompanionFormatter) GetFileSummary() string {
	if len(f.fileOperations) == 0 {
		return ""
	}

	var output strings.Builder
	output.WriteString("\n  📁 File Operations Summary:")
	for _, op := range f.fileOperations {
		output.WriteString(fmt.Sprintf("\n    - %s", op))
	}

	return output.String()
}

// Reset clears the formatter state
func (f *CompanionFormatter) Reset() {
	f.fileOperations = []string{}
	f.currentTool = ""
	// Keep cumulative statistics - don't reset totalFiles and totalTools
}

// CodeBlock represents a code block extracted from text
type CodeBlock struct {
	Language string
	Content  string
}
