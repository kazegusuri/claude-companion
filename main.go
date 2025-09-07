package main

import (
	"os"
	"os/signal"
	"syscall"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
	"github.com/kazegusuri/claude-companion/internal/event/print"
	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/narrator"
	"github.com/kazegusuri/claude-companion/internal/server/api"
	"github.com/kazegusuri/claude-companion/internal/server/db"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
	"github.com/kazegusuri/claude-companion/internal/server/watcher"
	"github.com/kazegusuri/claude-companion/internal/server/websocket"
	"github.com/kazegusuri/claude-companion/internal/session/event"
	"github.com/kazegusuri/claude-companion/internal/speech"
	"github.com/labstack/echo/v4"
	"github.com/spf13/pflag"
)

func main() {
	var project, session, file string
	var headMode bool
	var useAINarrator bool
	var openaiAPIKey string
	var narratorConfigPath string
	var enableVoice bool
	var voicevoxURL string
	var voiceSpeakerID int
	var notificationLog string
	var watchProjects bool
	var projectsRoot string
	var enableServer bool
	var serverPort string
	var dbFile string

	pflag.StringVarP(&project, "project", "p", "", "Project name")
	pflag.StringVarP(&session, "session", "s", "", "Session name")
	pflag.StringVarP(&file, "file", "f", "", "Direct path to session file")
	pflag.StringVar(&notificationLog, "notification-log", "/var/log/claude-notification.log", "Path to notification log file to watch")
	pflag.BoolVar(&headMode, "head", false, "Read entire file from beginning to end instead of tailing")
	var debugMode bool
	pflag.BoolVarP(&debugMode, "debug", "d", false, "Enable debug mode with detailed information")
	pflag.BoolVar(&useAINarrator, "ai", false, "Use AI narrator (requires OpenAI API key)")
	pflag.StringVar(&openaiAPIKey, "openai-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (can also use OPENAI_API_KEY env var)")
	pflag.StringVar(&narratorConfigPath, "narrator-config", "", "Path to narrator configuration file (JSON)")
	pflag.BoolVar(&enableVoice, "voice", false, "Enable voice output using VOICEVOX")
	pflag.StringVar(&voicevoxURL, "voicevox-url", "http://localhost:50021", "VOICEVOX server URL")
	pflag.IntVar(&voiceSpeakerID, "voice-speaker", 1, "VOICEVOX speaker ID (default: 1)")
	pflag.BoolVar(&enableServer, "server", false, "Enable WebSocket server for audio streaming")
	pflag.StringVar(&serverPort, "server-port", ":8080", "WebSocket server port (default: :8080)")
	pflag.StringVar(&dbFile, "db-file", "/var/lib/claude-companion/db.sqlite", "Path to SQLite database file (default: /var/lib/claude-companion/db.sqlite)")
	// watchProjects is now the default behavior
	pflag.StringVar(&projectsRoot, "projects-root", "~/.claude/projects", "Root directory for projects")
	pflag.Parse()

	// Default behavior is to watch projects
	watchProjects = true

	// Set global debug mode in logger
	logger.SetDebugMode(debugMode)

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
		logger.LogError("AI narrator requires OpenAI API key. Please set OPENAI_API_KEY environment variable or use --openai-key flag.")
		os.Exit(1)
	}

	// Initialize database if server mode is enabled
	var database *db.DB
	if enableServer {
		logger.LogInfo("Initializing database at: %s", dbFile)
		var err error
		database, err = db.Open(dbFile)
		if err != nil {
			logger.LogError("Failed to open database: %v", err)
			logger.LogError("Make sure the directory exists and has proper permissions:")
			logger.LogError("  sudo mkdir -p $(dirname %s)", dbFile)
			logger.LogError("  sudo chown $USER:$USER $(dirname %s)", dbFile)
			os.Exit(1)
		}
		defer func() {
			if database != nil {
				database.Close()
			}
		}()
		logger.LogInfo("Database initialized successfully")

		// Start watcher manager
		watcherManager := watcher.NewManager(database, logger.NewSlogLogger())

		// Set up callbacks for agent changes
		watcherManager.SetOnAgentAdded(func(pid int, agent db.ClaudeAgent) {
			logger.LogInfo("New Claude agent started: PID=%d, Project=%s, Session=%s",
				pid, agent.ProjectDir, agent.SessionID)
		})
		watcherManager.SetOnAgentRemoved(func(pid int) {
			logger.LogInfo("Claude agent stopped: PID=%d", pid)
		})

		// Start all watchers
		watcherManager.Start()
		defer watcherManager.Stop()
	}

	// Create session manager early as it's needed by multiple components
	sessionManager := handler.NewSessionManager()

	var n narrator.Narrator
	if narratorConfigPath != "" {
		n = narrator.NewHybridNarratorWithConfig(openaiAPIKey, useAINarrator, &narratorConfigPath)
	} else {
		n = narrator.NewHybridNarrator(openaiAPIKey, useAINarrator)
	}

	// Wrap with voice narrator if enabled
	var voiceNarrator *narrator.VoiceNarrator
	var wsServer *websocket.Server

	// Start HTTP server if server mode is enabled (for both WebSocket and API)
	if enableServer {
		// If voice is enabled, create WebSocket server
		if enableVoice {
			// Create WebSocket server with session manager and database
			wsServer = websocket.NewServer(sessionManager, database)
			go wsServer.Run()
		}

		// Create Echo server with API routes and WebSocket if enabled
		var echoServer *echo.Echo
		if wsServer != nil {
			echoServer = api.SetupEchoServerWithWebSocket(database, wsServer)
			logger.LogInfo("WebSocket endpoint: ws://localhost%s/ws/audio", serverPort)
		} else {
			echoServer = api.SetupEchoServer(database)
		}

		// Start HTTP server with Echo handling all routes
		go func() {
			logger.LogInfo("HTTP server listening on %s", serverPort)
			logger.LogInfo("API endpoint: http://localhost%s/api/agents", serverPort)
			if err := echoServer.Start(serverPort); err != nil {
				logger.LogError("Failed to start HTTP server: %v", err)
			}
		}()
	}

	if enableVoice {
		// Create synthesizer
		synthesizer := speech.NewVoiceVox(voicevoxURL, voiceSpeakerID)
		// Check if VOICEVOX is available
		if !synthesizer.IsAvailable() {
			logger.LogError("VOICEVOX server is not available at %s. Please make sure VOICEVOX is running.", voicevoxURL)
			logger.LogError("You can start VOICEVOX with: docker run -d --rm -it -p '127.0.0.1:50021:50021' voicevox/voicevox_engine:cpu-latest")
			os.Exit(1)
		}

		// Create player based on server option
		var player speech.Player
		if enableServer && wsServer != nil {
			// Use WebSocket player
			player = speech.NewWebSocketPlayer(wsServer)
		} else {
			// Use native player
			player = speech.NewNativePlayer()
		}

		voiceNarrator = narrator.NewVoiceNarratorWithTranslator(n, synthesizer, player, true, openaiAPIKey, useAINarrator)
		n = voiceNarrator
		defer voiceNarrator.Close()
	}

	// Create printer for central event handler
	printer := print.NewNotificationPrinter()

	// Create central event handler with emitter
	var emitter handler.MessageEmitter
	if wsServer != nil {
		emitter = wsServer
	}
	centralEventHandler := internalevent.NewHandler(sessionManager, n, printer, emitter)

	// Create session event handler with central handler
	sessionEventHandler := event.NewHandler(n, sessionManager, centralEventHandler)

	// Start both handlers
	centralEventHandler.Start()
	defer centralEventHandler.Stop()
	sessionEventHandler.Start()
	defer sessionEventHandler.Stop()

	// Start notification watcher if configured
	if hasNotificationInput {
		notificationWatcher := event.NewNotificationWatcher(notificationLog, sessionEventHandler)
		logger.LogInfo("Starting notification log watcher for: %s", notificationLog)
		if err := notificationWatcher.Start(); err != nil {
			logger.LogError("Error starting notification watcher: %v", err)
			os.Exit(1)
		}
		defer notificationWatcher.Stop()
	}

	// Start session watcher if using direct file input
	if hasDirectFileInput {
		sessionWatcher := event.NewSessionWatcher(sessionFilePath, sessionEventHandler)

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
		projectsWatcher, err := event.NewProjectsWatcher(projectsRoot, sessionEventHandler)
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
