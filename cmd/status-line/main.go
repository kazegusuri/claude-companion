// This implementation is based on the article:
// https://zenn.dev/pnd/articles/claude-code-statusline
// Original JavaScript version has been ported to Go

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const COMPACTION_THRESHOLD = 200000 * 0.8

type InputData struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Cwd       string `json:"cwd"`
	SessionID string `json:"session_id"`
}

type TranscriptEntry struct {
	Type    string `json:"type"`
	Message struct {
		Usage *Usage `json:"usage"`
	} `json:"message"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func main() {
	// Read JSON from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println("[Error] 📁 . | 🪙 0 | 0%")
		return
	}

	var data InputData
	if err := json.Unmarshal(input, &data); err != nil {
		fmt.Println("[Error] 📁 . | 🪙 0 | 0%")
		return
	}

	// Extract values
	model := data.Model.DisplayName
	if model == "" {
		model = "Unknown"
	}

	currentDir := data.Workspace.CurrentDir
	if currentDir == "" {
		currentDir = data.Cwd
	}
	if currentDir == "" {
		currentDir = "."
	}
	currentDir = filepath.Base(currentDir)

	sessionID := data.SessionID

	// Calculate token usage for current session
	totalTokens := 0

	if sessionID != "" {
		// Find all transcript files
		projectsDir := filepath.Join(os.Getenv("HOME"), ".claude", "projects")

		if _, err := os.Stat(projectsDir); err == nil {
			// Get all project directories
			entries, err := os.ReadDir(projectsDir)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						projectDir := filepath.Join(projectsDir, entry.Name())
						transcriptFile := filepath.Join(projectDir, sessionID+".jsonl")

						if _, err := os.Stat(transcriptFile); err == nil {
							totalTokens = calculateTokensFromTranscript(transcriptFile)
							break
						}
					}
				}
			}
		}
	}

	// Calculate percentage
	percentage := int(float64(totalTokens)/COMPACTION_THRESHOLD*100 + 0.5) // Round instead of truncate
	if percentage > 100 {
		percentage = 100
	}

	// Format token display
	tokenDisplay := formatTokenCount(totalTokens)

	// Color coding for percentage
	var percentageColor string
	switch {
	case percentage >= 90:
		percentageColor = "\x1b[31m" // Red
	case percentage >= 70:
		percentageColor = "\x1b[33m" // Yellow
	default:
		percentageColor = "\x1b[32m" // Green
	}

	// Build status line
	statusLine := fmt.Sprintf("[%s] 📁 %s | 🪙 %s | %s%d%%\x1b[0m",
		model, currentDir, tokenDisplay, percentageColor, percentage)

	fmt.Println(statusLine)
}

func calculateTokensFromTranscript(filePath string) int {
	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	var lastUsage *Usage
	scanner := bufio.NewScanner(file)
	// Increase buffer size for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line size

	for scanner.Scan() {
		var entry TranscriptEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid JSON lines
		}

		// Check if this is an assistant message with usage data
		if entry.Type == "assistant" && entry.Message.Usage != nil {
			lastUsage = entry.Message.Usage
		}
	}

	if lastUsage != nil {
		// The last usage entry contains cumulative tokens
		totalTokens := lastUsage.InputTokens +
			lastUsage.OutputTokens +
			lastUsage.CacheCreationInputTokens +
			lastUsage.CacheReadInputTokens
		return totalTokens
	}

	return 0
}

func formatTokenCount(tokens int) string {
	switch {
	case tokens >= 1000000:
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	case tokens >= 1000:
		return fmt.Sprintf("%.1fK", float64(tokens)/1000)
	default:
		return fmt.Sprintf("%d", tokens)
	}
}
