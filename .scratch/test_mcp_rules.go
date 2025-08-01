package main

import (
	"fmt"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	// Load the default configuration
	config := narrator.GetDefaultNarratorConfig()
	cn := narrator.NewConfigBasedNarrator(config)

	fmt.Println("=== Testing MCP Rules ===")

	// Test mcp__serena__read_memory
	input1 := map[string]interface{}{
		"memory_file_name": "project_notes.md",
	}
	result1 := cn.NarrateToolUse("mcp__serena__read_memory", input1)
	fmt.Printf("mcp__serena__read_memory: %s\n", result1)

	// Test mcp__serena__activate_project
	input2 := map[string]interface{}{
		"project_name": "MyProject",
	}
	result2 := cn.NarrateToolUse("mcp__serena__activate_project", input2)
	fmt.Printf("mcp__serena__activate_project: %s\n", result2)

	// Test mcp__serena__delete_lines
	input3 := map[string]interface{}{
		"file_path":  "main.go",
		"start_line": 10.0,
		"end_line":   20.0,
	}
	result3 := cn.NarrateToolUse("mcp__serena__delete_lines", input3)
	fmt.Printf("mcp__serena__delete_lines: %s\n", result3)

	// Test mcp__serena__read_file
	input4 := map[string]interface{}{
		"file_path": "/src/main.go",
	}
	result4 := cn.NarrateToolUse("mcp__serena__read_file", input4)
	fmt.Printf("mcp__serena__read_file: %s\n", result4)

	// Test mcp__serena__switch_modes
	input5 := map[string]interface{}{
		"modes": []interface{}{"read", "write"},
	}
	result5 := cn.NarrateToolUse("mcp__serena__switch_modes", input5)
	fmt.Printf("mcp__serena__switch_modes: %s\n", result5)

	// Test mcp__ide__getDiagnostics
	input6 := map[string]interface{}{}
	result6 := cn.NarrateToolUse("mcp__ide__getDiagnostics", input6)
	fmt.Printf("mcp__ide__getDiagnostics: %s\n", result6)

	// Test unknown MCP operation (should use default)
	result7 := cn.NarrateToolUse("mcp__serena__unknown_operation", map[string]interface{}{})
	fmt.Printf("mcp__serena__unknown_operation: %s\n", result7)
}
