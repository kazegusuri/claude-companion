package main

import (
	"fmt"

	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	// Load default configuration
	config := narrator.GetDefaultNarratorConfig()
	cn := narrator.NewConfigBasedNarrator(config)

	// Test cases for MCP tools with capture functionality
	testCases := []struct {
		toolName string
		input    map[string]interface{}
		expected string
	}{
		{
			toolName: "mcp__serena__read_memory",
			input: map[string]interface{}{
				"memory_file_name": "test_memory.txt",
			},
			expected: "メモリファイル「test_memory.txt」を読み込みます",
		},
		{
			toolName: "mcp__serena__activate_project",
			input: map[string]interface{}{
				"project_name": "my-awesome-project",
			},
			expected: "プロジェクト「my-awesome-project」をアクティブ化します",
		},
		{
			toolName: "mcp__serena__find_file",
			input: map[string]interface{}{
				"file_mask": "*.go",
			},
			expected: "パターン「*.go」に一致するファイルを検索します",
		},
		{
			toolName: "ReadMcpResourceTool",
			input: map[string]interface{}{
				"uri": "file://path/to/resource",
			},
			expected: "MCPリソース「file://path/to/resource」を読み込みます",
		},
	}

	fmt.Println("Testing new capture-driven MCP tool processing...")
	fmt.Println("============================================================")

	for i, tc := range testCases {
		result := cn.NarrateToolUse(tc.toolName, tc.input)
		fmt.Printf("Test %d: %s\n", i+1, tc.toolName)
		fmt.Printf("  Input: %v\n", tc.input)
		fmt.Printf("  Result: %s\n", result)
		fmt.Printf("  Expected: %s\n", tc.expected)
		if result == tc.expected {
			fmt.Printf("  Status: ✅ PASS\n")
		} else {
			fmt.Printf("  Status: ❌ FAIL\n")
		}
		fmt.Println()
	}

	// Test fallback for tools without capture config
	fmt.Println("Testing fallback for tools without capture config...")
	fmt.Println("============================================================")

	result := cn.NarrateToolUse("mcp__serena__list_memories", map[string]interface{}{})
	fmt.Printf("Tool: mcp__serena__list_memories\n")
	fmt.Printf("Result: %s\n", result)
	fmt.Printf("Expected: 利用可能なメモリファイルを一覧表示します\n")
	if result == "利用可能なメモリファイルを一覧表示します" {
		fmt.Printf("Status: ✅ PASS\n")
	} else {
		fmt.Printf("Status: ❌ FAIL\n")
	}
}
