package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var project, session, file string
	var fullRead, companionMode bool
	var narratorMode string
	var openaiAPIKey string

	flag.StringVar(&project, "project", "", "Project name")
	flag.StringVar(&session, "session", "", "Session name")
	flag.StringVar(&file, "file", "", "Direct path to session file")
	flag.BoolVar(&fullRead, "full", false, "Read entire file from beginning to end instead of tailing")
	flag.BoolVar(&companionMode, "companion", true, "Enable companion mode with enhanced formatting")
	flag.StringVar(&narratorMode, "narrator", "rule", "Narrator mode: rule, ai, or off")
	flag.StringVar(&openaiAPIKey, "openai-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (can also use OPENAI_API_KEY env var)")
	flag.Parse()

	var filePath string
	if file != "" {
		// Use direct file path
		filePath = file
	} else {
		// Use project/session path
		if project == "" || session == "" {
			flag.Usage()
			log.Fatal("Either -file or both -project and -session flags are required")
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get home directory: %v", err)
		}

		filePath = filepath.Join(homeDir, ".claude", "projects", project, session+".jsonl")
	}

	// Create narrator based on mode
	var narrator Narrator
	if companionMode && narratorMode != "off" {
		useAI := narratorMode == "ai"
		if useAI && openaiAPIKey == "" {
			log.Printf("Warning: AI narrator mode requires OpenAI API key. Falling back to rule-based mode.")
			useAI = false
		}
		narrator = NewHybridNarrator(openaiAPIKey, useAI)
	}

	if fullRead {
		log.Printf("Reading file: %s", filePath)
		if err := readFullFile(filePath, companionMode, narrator); err != nil {
			log.Fatalf("Error reading file: %v", err)
		}
	} else {
		log.Printf("Monitoring file: %s", filePath)
		if err := tailFile(filePath, companionMode, narrator); err != nil {
			log.Fatalf("Error tailing file: %v", err)
		}
	}
}

func tailFile(filePath string, companionMode bool, narrator Narrator) error {
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
			processJSONLine(line, companionMode, narrator)
		}
	}
}

func readFullFile(filePath string, companionMode bool, narrator Narrator) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle very long JSON lines (default is 64KB)
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) > 0 {
			processJSONLine(line, companionMode, narrator)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	log.Printf("Finished reading %d lines", lineNum)
	return nil
}

func processJSONLine(line string, companionMode bool, narrator Narrator) {
	parser := NewEventParser()
	parser.SetCompanionMode(companionMode)
	if narrator != nil {
		parser.SetNarrator(narrator)
	}
	output, err := parser.ParseAndFormat(line)
	if err != nil {
		log.Printf("Failed to parse event: %v", err)
		return
	}
	if output != "" {
		fmt.Print(output)
	}
}
