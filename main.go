package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kazegusuri/claude-companion/event"
	"github.com/kazegusuri/claude-companion/narrator"
	"github.com/spf13/pflag"
)

func main() {
	var project, session, file string
	var headMode, debugMode bool
	var useAINarrator bool
	var openaiAPIKey string
	var narratorConfigPath string
	var enableVoice bool
	var voicevoxURL string
	var voiceSpeakerID int
	var notificationLog string
	var watchProjects bool
	var projectsRoot string

	pflag.StringVarP(&project, "project", "p", "", "Project name")
	pflag.StringVarP(&session, "session", "s", "", "Session name")
	pflag.StringVarP(&file, "file", "f", "", "Direct path to session file")
	pflag.StringVar(&notificationLog, "notification-log", "", "Path to notification log file to watch")
	pflag.BoolVar(&headMode, "head", false, "Read entire file from beginning to end instead of tailing")
	pflag.BoolVarP(&debugMode, "debug", "d", false, "Enable debug mode with detailed information")
	pflag.BoolVar(&useAINarrator, "ai", false, "Use AI narrator (requires OpenAI API key)")
	pflag.StringVar(&openaiAPIKey, "openai-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (can also use OPENAI_API_KEY env var)")
	pflag.StringVar(&narratorConfigPath, "narrator-config", "", "Path to narrator configuration file (JSON)")
	pflag.BoolVar(&enableVoice, "voice", false, "Enable voice output using VOICEVOX")
	pflag.StringVar(&voicevoxURL, "voicevox-url", "http://localhost:50021", "VOICEVOX server URL")
	pflag.IntVar(&voiceSpeakerID, "voice-speaker", 1, "VOICEVOX speaker ID (default: 1)")
	// watchProjects is now the default behavior
	pflag.StringVar(&projectsRoot, "projects-root", "~/.claude/projects", "Root directory for projects")
	pflag.Parse()

	// Default behavior is to watch projects
	watchProjects = true

	// Determine input sources
	hasNotificationInput := notificationLog != ""
	hasSessionInput := file != "" || (project != "" && session != "")
	hasProjectsInput := watchProjects && !hasSessionInput && !hasNotificationInput

	// No longer need to check for required flags since watch-projects is default

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

		if headMode {
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

	// Start projects watcher if configured
	if hasProjectsInput {
		projectsWatcher, err := event.NewProjectsWatcher(projectsRoot, eventHandler)
		if err != nil {
			log.Fatalf("Error creating projects watcher: %v", err)
		}
		log.Printf("Starting projects watcher for: %s", projectsRoot)
		if err := projectsWatcher.Start(); err != nil {
			log.Fatalf("Error starting projects watcher: %v", err)
		}
		defer projectsWatcher.Stop()
	}

	// If we're running watchers (not head mode), wait for interrupt
	if hasNotificationInput || (hasSessionInput && !headMode) || hasProjectsInput {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Println("\nShutting down...")
	}
}
