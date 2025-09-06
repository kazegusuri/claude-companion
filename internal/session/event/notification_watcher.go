package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/kazegusuri/claude-companion/internal/logger"
)

// NotificationWatcher watches the notification log file for new events
type NotificationWatcher struct {
	filePath      string
	eventSender   EventSender
	done          chan struct{}
	dirWatcher    *fsnotify.Watcher
	fileWatcher   *fsnotify.Watcher
	watchingFile  bool
	retryInterval time.Duration
}

// NewNotificationWatcher creates a new notification watcher
func NewNotificationWatcher(filePath string, eventSender EventSender) *NotificationWatcher {
	return &NotificationWatcher{
		filePath:      filePath,
		eventSender:   eventSender,
		done:          make(chan struct{}),
		retryInterval: 5 * time.Second,
	}
}

// Start starts watching the notification log file
func (w *NotificationWatcher) Start() error {
	go w.watch()
	return nil
}

// Stop stops the watcher
func (w *NotificationWatcher) Stop() {
	close(w.done)
	if w.dirWatcher != nil {
		w.dirWatcher.Close()
	}
	if w.fileWatcher != nil {
		w.fileWatcher.Close()
	}
}

// watch monitors the notification file
func (w *NotificationWatcher) watch() {
	if err := w.watchFile(); err != nil {
		logger.LogError("Error watching notification file: %v", err)
	}
}

// watchFile starts watching the notification log file
func (w *NotificationWatcher) watchFile() error {
	file, err := os.Open(w.filePath)
	if err != nil {
		// If file doesn't exist, wait for it to be created
		if os.IsNotExist(err) {
			logger.LogInfo("Notification log file %s does not exist, waiting for it to be created...", w.filePath)
			return w.waitForFileAndWatch()
		}
		// If permission denied, wait for permissions to change
		if os.IsPermission(err) {
			logger.LogWarning("Permission denied for %s, waiting for permissions to change...", w.filePath)
			return w.waitForPermissionAndWatch()
		}
		return fmt.Errorf("failed to open notification log: %w", err)
	}
	defer file.Close()

	// Move to end of file to only watch new events
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	logger.LogInfo("Watching notification log: %s", w.filePath)
	w.watchingFile = true
	return w.tailFile(file)
}

// waitForFileAndWatch waits for the file to be created using fsnotify
func (w *NotificationWatcher) waitForFileAndWatch() error {
	// Create watcher for parent directory
	dir := filepath.Dir(w.filePath)
	fileName := filepath.Base(w.filePath)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create directory watcher: %w", err)
	}
	w.dirWatcher = watcher

	err = watcher.Add(dir)
	if err != nil {
		watcher.Close()
		return fmt.Errorf("failed to watch directory %s: %w", dir, err)
	}

	logger.LogInfo("Waiting for notification log file to be created: %s", w.filePath)

	for {
		select {
		case <-w.done:
			watcher.Close()
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("directory watcher closed")
			}
			// Check if the created file is our target file
			if event.Op&fsnotify.Create == fsnotify.Create && filepath.Base(event.Name) == fileName {
				logger.LogInfo("Notification log file created: %s", w.filePath)
				watcher.Close()
				w.dirWatcher = nil
				// Give the file a moment to be fully created
				time.Sleep(100 * time.Millisecond)
				return w.watchFile()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("directory watcher error channel closed")
			}
			logger.LogError("Directory watcher error: %v", err)
		}
	}
}

// waitForPermissionAndWatch waits for the file permissions to change
func (w *NotificationWatcher) waitForPermissionAndWatch() error {
	// Create a file watcher to detect permission changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	w.fileWatcher = watcher

	// Watch the file for changes (including permission changes)
	err = watcher.Add(w.filePath)
	if err != nil {
		watcher.Close()
		// If we can't even watch the file, fall back to periodic retry
		return w.retryWithInterval()
	}

	logger.LogInfo("Waiting for permission changes on: %s", w.filePath)

	for {
		select {
		case <-w.done:
			watcher.Close()
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("file watcher closed")
			}
			// On any change event (including chmod), try to open the file
			if event.Op&(fsnotify.Write|fsnotify.Chmod) != 0 {
				file, err := os.Open(w.filePath)
				if err == nil {
					file.Close()
					logger.LogInfo("File permissions changed, now accessible: %s", w.filePath)
					watcher.Close()
					w.fileWatcher = nil
					return w.watchFile()
				}
				// Still no permission, continue waiting
				if os.IsPermission(err) {
					logger.LogWarning("Still no permission to read file: %s", w.filePath)
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("file watcher error channel closed")
			}
			logger.LogError("File watcher error: %v", err)
		case <-time.After(w.retryInterval):
			// Periodic retry in case fsnotify misses the change
			file, err := os.Open(w.filePath)
			if err == nil {
				file.Close()
				logger.LogInfo("File now accessible (periodic check): %s", w.filePath)
				watcher.Close()
				w.fileWatcher = nil
				return w.watchFile()
			}
		}
	}
}

// retryWithInterval retries opening the file at regular intervals
func (w *NotificationWatcher) retryWithInterval() error {
	logger.LogInfo("Falling back to periodic retry for: %s", w.filePath)

	ticker := time.NewTicker(w.retryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return nil
		case <-ticker.C:
			file, err := os.Open(w.filePath)
			if err == nil {
				file.Close()
				logger.LogInfo("File now accessible: %s", w.filePath)
				return w.watchFile()
			}
			if !os.IsPermission(err) && !os.IsNotExist(err) {
				return fmt.Errorf("unexpected error opening file: %w", err)
			}
		}
	}
}

// tailFile continuously reads new lines from the file
func (w *NotificationWatcher) tailFile(file *os.File) error {
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
				w.processNotificationLine(line)
			}
		}
	}
}

// processNotificationLine processes a single line from the notification log
func (w *NotificationWatcher) processNotificationLine(line string) {
	// Parse JSON notification event
	var notificationEvent NotificationEvent
	if err := json.Unmarshal([]byte(line), &notificationEvent); err != nil {
		// Send error through event handler if in debug mode
		return
	}

	// Send event to handler
	w.eventSender.SendEvent(&notificationEvent)
}
