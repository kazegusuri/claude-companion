package event

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/kazegusuri/claude-companion/logger"
)

// SessionWatcher watches session log files
type SessionWatcher struct {
	filePath     string
	eventHandler *Handler
	parser       *Parser
	done         chan struct{}
}

// NewSessionWatcher creates a new session watcher
func NewSessionWatcher(filePath string, eventHandler *Handler) *SessionWatcher {
	return &SessionWatcher{
		filePath:     filePath,
		eventHandler: eventHandler,
		parser:       NewParserWithPath(filePath),
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
	if err := w.tailFile(); err != nil {
		logger.LogError("Error watching session file: %v", err)
	}
}

// tailFile tails the session file
func (w *SessionWatcher) tailFile() error {
	file, err := os.Open(w.filePath)
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
