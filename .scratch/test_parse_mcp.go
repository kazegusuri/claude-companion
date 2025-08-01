package main

import (
	"fmt"
	"strings"
)

func parseMCPToolName(toolName string) (server string, operation string, isMCP bool) {
	if !strings.HasPrefix(toolName, "mcp__") {
		return "", "", false
	}
	// Remove mcp__ prefix
	name := strings.TrimPrefix(toolName, "mcp__")
	// Split by first underscore
	parts := strings.SplitN(name, "__", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], true
	}
	// Legacy format: mcp__server_operation
	underscoreIndex := strings.Index(name, "_")
	if underscoreIndex > 0 {
		return name[:underscoreIndex], name[underscoreIndex+1:], true
	}
	return "", "", false
}

func main() {
	testCases := []string{
		"mcp__serena__read_memory",
		"mcp__serena__activate_project",
		"mcp__ide__getDiagnostics",
		"mcp__serena_read_file", // legacy format
		"ReadMcpResourceTool",
		"ListMcpResourcesTool",
	}

	for _, tc := range testCases {
		server, operation, isMCP := parseMCPToolName(tc)
		fmt.Printf("%s -> server: %q, operation: %q, isMCP: %v\n", tc, server, operation, isMCP)
	}
}
