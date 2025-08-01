package main

import (
	"fmt"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	// Load the actual configuration
	config := narrator.GetDefaultNarratorConfig()
	cn := narrator.NewConfigBasedNarrator(config)

	// Test Read tool with various extensions
	fmt.Println("=== Read tool with parseFileType ===")
	testFiles := []string{
		"main.go",
		"app.js",
		"config.yaml",
		"README.md",
		"styles.css",
		"unknown.xyz", // Unknown extension
	}

	for _, filename := range testFiles {
		input := map[string]interface{}{
			"file_path": filename,
		}
		result := cn.NarrateToolUse("Read", input)
		fmt.Printf("%s -> %s\n", filename, result)
	}

	// Test mcp__serena__read_file tool
	fmt.Println("\n=== mcp__serena__read_file tool with parseFileType ===")
	testPaths := []string{
		"/project/src/main.go",
		"/project/lib/utils.js",
		"/config/settings.yaml",
		"/docs/README.md",
		"test.xyz", // Unknown extension
	}

	for _, filepath := range testPaths {
		input := map[string]interface{}{
			"file_path": filepath,
		}
		result := cn.NarrateToolUse("mcp__serena__read_file", input)
		fmt.Printf("%s -> %s\n", filepath, result)
	}

	// Test Write tool
	fmt.Println("\n=== Write tool with parseFileType ===")
	writeTests := []string{
		"test.py",
		"app.ts",
		"styles.scss", // Unknown extension
	}

	for _, filename := range writeTests {
		input := map[string]interface{}{
			"file_path": filename,
		}
		result := cn.NarrateToolUse("Write", input)
		fmt.Printf("%s -> %s\n", filename, result)
	}

	// Test Edit tool
	fmt.Println("\n=== Edit tool with parseFileType ===")
	editInput := map[string]interface{}{
		"file_path":  "components/Header.tsx",
		"old_string": "old content",
		"new_string": "new content",
	}
	result := cn.NarrateToolUse("Edit", editInput)
	fmt.Printf("Header.tsx -> %s\n", result)

	// Test NotebookRead
	fmt.Println("\n=== NotebookRead with parseFileType ===")
	notebookInput := map[string]interface{}{
		"notebook_path": "analysis.ipynb",
	}
	result = cn.NarrateToolUse("NotebookRead", notebookInput)
	fmt.Printf("analysis.ipynb -> %s\n", result)
}
