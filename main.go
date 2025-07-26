package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var project, session string
	var fullRead bool
	flag.StringVar(&project, "project", "", "Project name")
	flag.StringVar(&session, "session", "", "Session name")
	flag.BoolVar(&fullRead, "full", false, "Read entire file from beginning to end instead of tailing")
	flag.Parse()

	if project == "" || session == "" {
		flag.Usage()
		log.Fatal("Both -project and -session flags are required")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	filePath := filepath.Join(homeDir, ".claude", "projects", project, session+".jsonl")

	if fullRead {
		log.Printf("Reading file: %s", filePath)
		if err := readFullFile(filePath); err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
	} else {
		log.Printf("Monitoring file: %s", filePath)
		if err := tailFile(filePath); err != nil {
			log.Fatalf("Error tailing file: %v", err)
		}
	}
}

func tailFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Move to end of file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	reader := bufio.NewReader(file)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// No new data, wait a bit
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("error reading line: %w", err)
		}

		// Process the line
		if len(line) > 0 {
			processJSONLine(line)
		}
	}
}

func readFullFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) > 0 {
			processJSONLine(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	log.Printf("Finished reading %d lines", lineNum)
	return nil
}

func processJSONLine(line string) {
	// First, parse to get the event type
	var baseEvent BaseEvent
	if err := json.Unmarshal([]byte(line), &baseEvent); err != nil {
		log.Printf("Failed to parse base event: %v", err)
		return
	}

	switch baseEvent.Type {
	case EventTypeUser:
		var event UserMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("Failed to parse user message: %v", err)
			return
		}
		fmt.Printf("\n[%s] USER: %s\n", event.Timestamp.Format("15:04:05"), event.Message.Content)

	case EventTypeAssistant:
		var event AssistantMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("Failed to parse assistant message: %v", err)
			return
		}
		fmt.Printf("\n[%s] ASSISTANT (%s):\n", event.Timestamp.Format("15:04:05"), event.Message.Model)

		for _, content := range event.Message.Content {
			switch content.Type {
			case "text":
				fmt.Printf("  Text: %s\n", content.Text)
			case "tool_use":
				fmt.Printf("  Tool Use: %s (id: %s)\n", content.Name, content.ID)
				if content.Input != nil {
					inputJSON, _ := json.MarshalIndent(content.Input, "    ", "  ")
					fmt.Printf("    Input: %s\n", string(inputJSON))
				}
			}
		}

		if event.Message.Usage.OutputTokens > 0 {
			fmt.Printf("  Tokens: input=%d, output=%d, cache_read=%d, cache_creation=%d\n",
				event.Message.Usage.InputTokens,
				event.Message.Usage.OutputTokens,
				event.Message.Usage.CacheReadInputTokens,
				event.Message.Usage.CacheCreationInputTokens)
		}

	case EventTypeSystem:
		var event SystemMessage
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			log.Printf("Failed to parse system message: %v", err)
			return
		}
		if !event.IsMeta {
			fmt.Printf("\n[%s] SYSTEM: %s\n", event.Timestamp.Format("15:04:05"), event.Content)
		}

	default:
		// For tool results and other types, show basic info
		fmt.Printf("\n[%s] %s event\n", baseEvent.Timestamp.Format("15:04:05"), baseEvent.Type)

		// Also show raw JSON for unknown types
		var data map[string]interface{}
		json.Unmarshal([]byte(line), &data)
		prettyJSON, _ := json.MarshalIndent(data, "  ", "  ")
		fmt.Printf("  Raw: %s\n", string(prettyJSON))
	}
}
