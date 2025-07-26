package main

import (
	"fmt"
	"regexp"
	"strings"
)

// CompanionFormatter provides enhanced formatting for a more companion-like experience
type CompanionFormatter struct {
	fileOperations []string
	currentTool    string
	// Track statistics for the session
	totalFiles int
	totalTools int
	// Narrator for natural language descriptions
	narrator Narrator
}

// NewCompanionFormatter creates a new CompanionFormatter instance
func NewCompanionFormatter() *CompanionFormatter {
	return &CompanionFormatter{
		fileOperations: make([]string, 0),
	}
}

// SetNarrator sets the narrator for natural language descriptions
func (f *CompanionFormatter) SetNarrator(n Narrator) {
	f.narrator = n
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

	// Use narrator if available
	if f.narrator != nil {
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
			return output.String()
		}
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

		// Show the main text with placeholders
		lines := strings.Split(strings.TrimSpace(processedText), "\n")
		for i, line := range lines {
			if i < 3 || strings.Contains(line, "[CODE BLOCK") {
				output.WriteString(fmt.Sprintf("\n  %s", line))
			} else if i == 3 && len(lines) > 4 {
				output.WriteString("\n  ... (text continues)")
				break
			}
		}

		// Show code blocks separately
		for i, block := range codeBlocks {
			output.WriteString(fmt.Sprintf("\n  \n  📝 Code Block %d (%s):", i+1, block.Language))
			// Show first few lines of code
			codeLines := strings.Split(strings.TrimSpace(block.Content), "\n")
			for j, line := range codeLines {
				if j < 5 {
					output.WriteString(fmt.Sprintf("\n    %s", line))
				} else if j == 5 && len(codeLines) > 6 {
					output.WriteString(fmt.Sprintf("\n    ... (%d more lines)", len(codeLines)-5))
					break
				}
			}
		}
	} else {
		// No code blocks, show text normally but truncated
		lines := strings.Split(strings.TrimSpace(text), "\n")
		for i, line := range lines {
			if i < 5 {
				output.WriteString(fmt.Sprintf("\n  %s", line))
			} else if i == 5 && len(lines) > 6 {
				output.WriteString(fmt.Sprintf("\n  ... (%d more lines)", len(lines)-5))
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
	output.WriteString("\n\n📁 File Operations Summary:")
	for _, op := range f.fileOperations {
		output.WriteString(fmt.Sprintf("\n  - %s", op))
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
