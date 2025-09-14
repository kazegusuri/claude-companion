package event

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kazegusuri/claude-companion/internal/logger"
)

// SessionWatcher watches session log files
type SessionWatcher struct {
	filePath     string
	eventHandler *Handler
	parser       *Parser
	done         chan struct{}
	lastPosition int64 // Track the last read position
}

// NewSessionWatcher creates a new session watcher using a builder
func NewSessionWatcher(filePath string, builder *HandlerBuilder) *SessionWatcher {
	// Build handler and parser from the file path
	handler, parser := builder.BuildFromPath(filePath)

	return &SessionWatcher{
		filePath:     filePath,
		eventHandler: handler,
		parser:       parser,
		done:         make(chan struct{}),
	}
}

// Start starts watching the session file
func (w *SessionWatcher) Start() error {
	go w.watch()
	return nil
}

// Stop stops the watcher
func (w *SessionWatcher) Stop() {
	close(w.done)
}

// watch monitors the session file
func (w *SessionWatcher) watch() {
	// Tail the file (includes warmup processing)
	if err := w.tailFile(); err != nil {
		logger.LogError("Error watching session file: %v", err)
	}
}

// tailFile processes existing lines then tails the session file
func (w *SessionWatcher) tailFile() error {
	file, err := os.Open(w.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// First, process all existing lines (warmup)
	if err := w.processExistingLines(file); err != nil {
		return fmt.Errorf("failed to process existing lines: %w", err)
	}

	// Now continue tailing from the current position
	reader := bufio.NewReader(file)

	for {
		select {
		case <-w.done:
			return nil
		default:
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
				// Parse the line into an event
				event, err := w.parser.Parse(line)
				if err != nil {
					logger.LogError("Error parsing line: %v", err)
					continue
				}
				w.eventHandler.SendEvent(event)
			}
		}
	}
}

// ReadFullFile reads the entire session file
func (w *SessionWatcher) ReadFullFile() error {
	file, err := os.Open(w.filePath)
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
			// Parse the line into an event
			event, err := w.parser.Parse(line)
			if err != nil {
				logger.LogError("Error parsing line %d: %v", lineNum, err)
				continue
			}
			w.eventHandler.SendEvent(event)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	logger.LogInfo("Finished reading %d lines", lineNum)
	return nil
}

// processExistingLines reads all existing lines in the file to initialize the session
func (w *SessionWatcher) processExistingLines(file *os.File) error {
	// Set warmup mode
	w.eventHandler.SetWarmupMode(true)
	defer w.eventHandler.SetWarmupMode(false)

	// Start from the beginning of the file
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("failed to seek to start: %w", err)
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer size to handle very long JSON lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	lineCount := 0

	// Read all lines from the beginning to the current end
	for scanner.Scan() {
		lineCount++
		line := scanner.Text()
		if len(line) > 0 {
			// Parse the line into a full event
			event, err := w.parser.Parse(line)
			if err != nil {
				logger.LogError("Error parsing warmup line %d: %v", lineCount, err)
				continue
			}
			// Send to warmup handler
			w.eventHandler.HandleWarmupEvent(event)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file during warmup: %w", err)
	}

	// Record the current position after processing existing lines
	w.lastPosition, err = file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}

	if logger.IsDebugMode() {
		logger.LogInfo("Warmup completed: processed %d lines from %s, position: %d", lineCount, w.filePath, w.lastPosition)
	}

	return nil
}
