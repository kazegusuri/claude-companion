package main

import (
	"fmt"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	// Load the actual configuration
	config := narrator.GetDefaultNarratorConfig()
	cn := narrator.NewConfigBasedNarrator(config)

	// Test mcp__serena__read_memory
	input1 := map[string]interface{}{
		"memory_file_name": "project_notes.md",
	}
	result1 := cn.NarrateToolUse("mcp__serena__read_memory", input1)
	fmt.Printf("mcp__serena__read_memory: %s\n", result1)

	// Test ReadMcpResourceTool
	input2 := map[string]interface{}{
		"uri": "resource://example/data",
	}
	result2 := cn.NarrateToolUse("ReadMcpResourceTool", input2)
	fmt.Printf("ReadMcpResourceTool: %s\n", result2)

	// Test mcp__serena__activate_project
	input3 := map[string]interface{}{
		"project_name": "MyProject",
	}
	result3 := cn.NarrateToolUse("mcp__serena__activate_project", input3)
	fmt.Printf("mcp__serena__activate_project: %s\n", result3)

	// Test mcp__serena__find_referencing_symbols
	input4 := map[string]interface{}{
		"symbol_name": "handleRequest",
	}
	result4 := cn.NarrateToolUse("mcp__serena__find_referencing_symbols", input4)
	fmt.Printf("mcp__serena__find_referencing_symbols: %s\n", result4)

	// Test mcp__serena__delete_lines (multiple parameters)
	input5 := map[string]interface{}{
		"file_path":  "main.go",
		"start_line": 10.0,
		"end_line":   20.0,
	}
	result5 := cn.NarrateToolUse("mcp__serena__delete_lines", input5)
	fmt.Printf("mcp__serena__delete_lines: %s\n", result5)

	// Test mcp__serena__insert_at_line (multiple parameters)
	input6 := map[string]interface{}{
		"file_path": "config.yaml",
		"line":      15.0,
	}
	result6 := cn.NarrateToolUse("mcp__serena__insert_at_line", input6)
	fmt.Printf("mcp__serena__insert_at_line: %s\n", result6)
}
