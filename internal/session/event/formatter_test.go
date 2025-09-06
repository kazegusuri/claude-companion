package event

import (
	"strings"
	"testing"

	"github.com/kazegusuri/claude-companion/internal/narrator"
)

func TestFormatAssistantText(t *testing.T) {
	formatter := NewFormatter(narrator.NewNoOpNarrator())

	tests := []struct {
		name           string
		text           string
		isThinking     bool
		wantContain    []string
		wantNotContain []string
		description    string
	}{
		{
			name:       "single_line_text",
			text:       "Hello world",
			isThinking: false,
			wantContain: []string{
				"💬 Hello world",
			},
			wantNotContain: []string{
				"📝 Hello world", // Single line should not have 📝
			},
			description: "Single line text should only show narration",
		},
		{
			name:       "multi_line_text",
			text:       "Line 1\nLine 2\nLine 3",
			isThinking: false,
			wantContain: []string{
				"💬 Line 1", // Narration shows first line
				"📝 Line 1", // First line with emoji
				"Line 2",   // Second line without emoji
				"Line 3",   // Third line without emoji
			},
			description: "Multi-line text should show narration and all lines with 📝 on first",
		},
		{
			name:       "text_with_code_block",
			text:       "Here is some code:\n```python\ndef hello():\n    print('Hello')\n```\nEnd of code",
			isThinking: false,
			wantContain: []string{
				"💬 Here is some code:",     // Narration (with placeholder in actual text)
				"📝 Here is some code:",     // First line with emoji
				"End of code",              // Text after code block
				"📝 Code Block 1 (python):", // Code block header
				"def hello():",             // Code content
				"print('Hello')",           // Code content
			},
			wantNotContain: []string{
				"📝 [CODE BLOCK", // Placeholder line should not have 📝 emoji
			},
			description: "Text with code block should show text and code separately",
		},
		{
			name:       "single_line_with_code_block",
			text:       "```go\nfmt.Println(\"test\")\n```",
			isThinking: false,
			wantContain: []string{
				"💬 [CODE BLOCK 1: go]",  // Narration with placeholder
				"📝 Code Block 1 (go):",  // Code block header
				"fmt.Println(\"test\")", // Code content
			},
			wantNotContain: []string{
				"📝 [CODE BLOCK", // Placeholder line should not have 📝
			},
			description: "Single line with only code block",
		},
		{
			name:       "multiple_code_blocks",
			text:       "First block:\n```js\nconsole.log('1');\n```\nSecond block:\n```py\nprint('2')\n```",
			isThinking: false,
			wantContain: []string{
				"💬 First block:",       // Narration
				"📝 First block:",       // First line with emoji
				"Second block:",        // Text between blocks
				"📝 Code Block 1 (js):", // First code block
				"console.log('1');",    // JS code
				"📝 Code Block 2 (py):", // Second code block
				"print('2')",           // Python code
			},
			description: "Multiple code blocks should be shown separately",
		},
		{
			name:       "thinking_mode",
			text:       "Analyzing the problem\nConsidering options",
			isThinking: true,
			wantContain: []string{
				"💬 Analyzing the problem", // Narration
				"📝 Analyzing the problem", // First line with emoji
				"Considering options",     // Second line
			},
			description: "Thinking mode should work the same way",
		},
		{
			name:       "code_block_no_language",
			text:       "Output:\n```\nplain text output\n```",
			isThinking: false,
			wantContain: []string{
				"💬 Output:",              // Narration (NoOpNarrator returns with placeholder)
				"📝 Code Block 1 (text):", // Default language
				"plain text output",      // Code content
			},
			wantNotContain: []string{
				"📝 Output:", // Single line after filtering placeholders
			},
			description: "Code block without language should default to 'text'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatAssistantText(tt.text, tt.isThinking, nil)

			// Check that all expected strings are contained
			for _, want := range tt.wantContain {
				if !strings.Contains(result, want) {
					t.Errorf("FormatAssistantText() result does not contain %q\nGot:\n%s", want, result)
				}
			}

			// Check that unwanted strings are not contained
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(result, notWant) {
					t.Errorf("FormatAssistantText() result should not contain %q\nGot:\n%s", notWant, result)
				}
			}
		})
	}
}

func TestExtractCodeBlocks(t *testing.T) {
	formatter := NewFormatter(narrator.NewNoOpNarrator())

	tests := []struct {
		name         string
		text         string
		wantBlocks   int
		wantLanguage []string
		wantContent  []string
		description  string
	}{
		{
			name:         "single_code_block",
			text:         "```python\ndef hello():\n    pass\n```",
			wantBlocks:   1,
			wantLanguage: []string{"python"},
			wantContent:  []string{"def hello():\n    pass\n"},
			description:  "Should extract single code block",
		},
		{
			name:         "multiple_code_blocks",
			text:         "```js\nconst x = 1;\n```\nSome text\n```go\nfmt.Println()\n```",
			wantBlocks:   2,
			wantLanguage: []string{"js", "go"},
			wantContent:  []string{"const x = 1;\n", "fmt.Println()\n"},
			description:  "Should extract multiple code blocks",
		},
		{
			name:         "code_block_no_language",
			text:         "```\nplain text\n```",
			wantBlocks:   1,
			wantLanguage: []string{"text"},
			wantContent:  []string{"plain text\n"},
			description:  "Should handle code block without language",
		},
		{
			name:         "no_code_blocks",
			text:         "Just regular text without code",
			wantBlocks:   0,
			wantLanguage: []string{},
			wantContent:  []string{},
			description:  "Should return empty for text without code blocks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocks := formatter.ExtractCodeBlocks(tt.text)

			if len(blocks) != tt.wantBlocks {
				t.Errorf("ExtractCodeBlocks() got %d blocks, want %d", len(blocks), tt.wantBlocks)
			}

			for i, block := range blocks {
				if i < len(tt.wantLanguage) {
					if block.Language != tt.wantLanguage[i] {
						t.Errorf("Block %d: got language %q, want %q", i, block.Language, tt.wantLanguage[i])
					}
				}
				if i < len(tt.wantContent) {
					if block.Content != tt.wantContent[i] {
						t.Errorf("Block %d: got content %q, want %q", i, block.Content, tt.wantContent[i])
					}
				}
			}
		})
	}
}
