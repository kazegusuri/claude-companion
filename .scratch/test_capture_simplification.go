package main

import (
	"fmt"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	// Test capture functionality with simplified structure
	config := &narrator.NarratorConfig{
		Rules: map[string]narrator.ToolRules{
			"mcp__test__tool": {
				Default: "Processing {project_name} in {memory_name}",
				Captures: []narrator.CaptureRule{
					{InputKey: "project_name"},
					{InputKey: "memory_name"},
				},
			},
		},
	}

	cn := narrator.NewConfigBasedNarrator(config)

	// Test with input values
	input := map[string]interface{}{
		"project_name": "MyProject",
		"memory_name":  "cache.md",
	}

	result := cn.NarrateToolUse("mcp__test__tool", input)
	fmt.Printf("Result: %s\n", result)
	// Expected: "Processing MyProject in cache.md"

	// Test with missing values
	input2 := map[string]interface{}{
		"project_name": "AnotherProject",
	}

	result2 := cn.NarrateToolUse("mcp__test__tool", input2)
	fmt.Printf("Result with missing value: %s\n", result2)
	// Expected: "Processing AnotherProject in {memory_name}"
}
