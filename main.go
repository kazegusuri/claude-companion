package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kazegusuri/claude-companion/event"
	"github.com/kazegusuri/claude-companion/narrator"
)

func main() {
	var project, session, file string
	var fullRead, debugMode bool
	var useAINarrator bool
	var openaiAPIKey string
	var narratorConfigPath string
	var enableVoice bool
	var voicevoxURL string
	var voiceSpeakerID int
	var notificationLog string

	flag.StringVar(&project, "project", "", "Project name")
	flag.StringVar(&session, "session", "", "Session name")
	flag.StringVar(&file, "file", "", "Direct path to session file")
	flag.StringVar(&notificationLog, "notification-log", "", "Path to notification log file to watch")
	flag.BoolVar(&fullRead, "full", false, "Read entire file from beginning to end instead of tailing")
	flag.BoolVar(&debugMode, "debug", false, "Enable debug mode with detailed information")
	flag.BoolVar(&useAINarrator, "ai", false, "Use AI narrator (requires OpenAI API key)")
	flag.StringVar(&openaiAPIKey, "openai-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (can also use OPENAI_API_KEY env var)")
	flag.StringVar(&narratorConfigPath, "narrator-config", "", "Path to narrator configuration file (JSON)")
	flag.BoolVar(&enableVoice, "voice", false, "Enable voice output using VOICEVOX")
	flag.StringVar(&voicevoxURL, "voicevox-url", "http://localhost:50021", "VOICEVOX server URL")
	flag.IntVar(&voiceSpeakerID, "voice-speaker", 1, "VOICEVOX speaker ID (default: 1)")
	flag.Parse()

	// Determine input sources
	hasNotificationInput := notificationLog != ""
	hasSessionInput := file != "" || (project != "" && session != "")

	if !hasNotificationInput && !hasSessionInput {
		flag.Usage()
		log.Fatal("Either -file, -notification-log, or both -project and -session flags are required")
	}

	// Determine session file path if applicable
	var sessionFilePath string
	if hasSessionInput {
		if file != "" {
			// Use direct file path
			sessionFilePath = file
		} else {
			// Use project/session path
			homeDir, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("Failed to get home directory: %v", err)
			}
			sessionFilePath = filepath.Join(homeDir, ".claude", "projects", project, session+".jsonl")
		}
	}

	// Create narrator
	if useAINarrator && openaiAPIKey == "" {
		log.Printf("Warning: AI narrator requires OpenAI API key. Using rule-based narrator.")
		useAINarrator = false
	}

	var n narrator.Narrator
	if narratorConfigPath != "" {
		n = narrator.NewHybridNarratorWithConfig(openaiAPIKey, useAINarrator, &narratorConfigPath)
	} else {
		n = narrator.NewHybridNarrator(openaiAPIKey, useAINarrator)
	}

	// Wrap with voice narrator if enabled
	var voiceNarrator *narrator.VoiceNarrator
	if enableVoice {
		voiceClient := narrator.NewVoiceVoxClient(voicevoxURL, voiceSpeakerID)
		voiceNarrator = narrator.NewVoiceNarratorWithTranslator(n, voiceClient, true, openaiAPIKey, useAINarrator)
		n = voiceNarrator
		defer voiceNarrator.Close()
	}

	// Create event handler
	eventHandler := event.NewHandler(n, debugMode)
	eventHandler.Start()
	defer eventHandler.Stop()

	// Start notification watcher if configured
	if hasNotificationInput {
		notificationWatcher := event.NewNotificationWatcher(notificationLog, eventHandler)
		log.Printf("Starting notification log watcher for: %s", notificationLog)
		if err := notificationWatcher.Start(); err != nil {
			log.Fatalf("Error starting notification watcher: %v", err)
		}
		defer notificationWatcher.Stop()
	}

	// Start session watcher if configured
	if hasSessionInput {
		sessionWatcher := event.NewSessionWatcher(sessionFilePath, eventHandler)

		if fullRead {
			log.Printf("Reading file: %s", sessionFilePath)
			if err := sessionWatcher.ReadFullFile(); err != nil {
				log.Fatalf("Error reading file: %v", err)
			}
		} else {
			log.Printf("Monitoring file: %s", sessionFilePath)
			if err := sessionWatcher.Start(); err != nil {
				log.Fatalf("Error starting session watcher: %v", err)
			}
			defer sessionWatcher.Stop()

		}
	}

	// If we're running watchers (not full read mode), wait for interrupt
	if hasNotificationInput || (hasSessionInput && !fullRead) {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("\nShutting down...")
	}
}
