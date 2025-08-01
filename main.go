package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/kazegusuri/claude-companion/event"
	"github.com/kazegusuri/claude-companion/logger"
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
	pflag.StringVar(&notificationLog, "notification-log", "/var/log/claude-notification.log", "Path to notification log file to watch")
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
	hasDirectFileInput := file != ""
	// project/session options now act as filters for watch mode
	hasProjectsInput := watchProjects && !hasDirectFileInput

	// No longer need to check for required flags since watch-projects is default

	// Determine session file path if using direct file input
	var sessionFilePath string
	if hasDirectFileInput {
		// Use direct file path
		sessionFilePath = file
	}

	// Create narrator
	if useAINarrator && openaiAPIKey == "" {
		logger.LogWarning("AI narrator requires OpenAI API key. Using rule-based narrator.")
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
		logger.LogInfo("Starting notification log watcher for: %s", notificationLog)
		if err := notificationWatcher.Start(); err != nil {
			logger.LogError("Error starting notification watcher: %v", err)
			os.Exit(1)
		}
		defer notificationWatcher.Stop()
	}

	// Start session watcher if using direct file input
	if hasDirectFileInput {
		sessionWatcher := event.NewSessionWatcher(sessionFilePath, eventHandler)

		if headMode {
			logger.LogInfo("Reading file: %s", sessionFilePath)
			if err := sessionWatcher.ReadFullFile(); err != nil {
				logger.LogError("Error reading file: %v", err)
				os.Exit(1)
			}
		} else {
			logger.LogInfo("Monitoring file: %s", sessionFilePath)
			if err := sessionWatcher.Start(); err != nil {
				logger.LogError("Error starting session watcher: %v", err)
				os.Exit(1)
			}
			defer sessionWatcher.Stop()

		}
	}

	// Start projects watcher if configured
	if hasProjectsInput {
		projectsWatcher, err := event.NewProjectsWatcher(projectsRoot, eventHandler)
		if err != nil {
			logger.LogError("Error creating projects watcher: %v", err)
			os.Exit(1)
		}

		// Set filters based on project/session options
		if project != "" {
			projectsWatcher.SetProjectFilter(project)
		}
		if session != "" {
			projectsWatcher.SetSessionFilter(session)
		}

		logger.LogInfo("Starting projects watcher for: %s", projectsRoot)
		if project != "" {
			logger.LogInfo("Filtering to project: %s", project)
		}
		if session != "" {
			logger.LogInfo("Filtering to session: %s", session)
		}

		if err := projectsWatcher.Start(); err != nil {
			logger.LogError("Error starting projects watcher: %v", err)
			os.Exit(1)
		}
		defer projectsWatcher.Stop()
	}

	// If we're running watchers (not head mode), wait for interrupt
	if hasNotificationInput || (hasDirectFileInput && !headMode) || hasProjectsInput {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		logger.LogInfo("Shutting down...")
	}
}
