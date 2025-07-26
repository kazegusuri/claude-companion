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
	flag.StringVar(&project, "project", "", "Project name")
	flag.StringVar(&session, "session", "", "Session name")
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

	log.Printf("Monitoring file: %s", filePath)

	if err := tailFile(filePath); err != nil {
		log.Fatalf("Error tailing file: %v", err)
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

func processJSONLine(line string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		log.Printf("Failed to parse JSON: %v", err)
		return
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}

	fmt.Printf("New event:\n%s\n---\n", string(prettyJSON))
}
