package event

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// NotificationWatcher watches the notification log file for new events
type NotificationWatcher struct {
	filePath    string
	eventSender EventSender
	done        chan struct{}
}

// NewNotificationWatcher creates a new notification watcher
func NewNotificationWatcher(filePath string, eventSender EventSender) *NotificationWatcher {
	return &NotificationWatcher{
		filePath:    filePath,
		eventSender: eventSender,
		done:        make(chan struct{}),
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
}

// watch monitors the notification file
func (w *NotificationWatcher) watch() {
	if err := w.watchFile(); err != nil {
		log.Printf("Error watching notification file: %v", err)
	}
}

// watchFile starts watching the notification log file
func (w *NotificationWatcher) watchFile() error {
	file, err := os.Open(w.filePath)
	if err != nil {
		// If file doesn't exist, wait for it to be created
		if os.IsNotExist(err) {
			log.Printf("Notification log file %s does not exist, waiting for it to be created...", w.filePath)
			return w.waitForFileAndWatch()
		}
		return fmt.Errorf("failed to open notification log: %w", err)
	}
	defer file.Close()

	// Move to end of file to only watch new events
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end: %w", err)
	}

	log.Printf("Watching notification log: %s", w.filePath)
	return w.tailFile(file)
}

// waitForFileAndWatch waits for the file to be created then starts watching
func (w *NotificationWatcher) waitForFileAndWatch() error {
	for {
		select {
		case <-w.done:
			return nil
		default:
			if _, err := os.Stat(w.filePath); err == nil {
				log.Printf("Notification log file created: %s", w.filePath)
				return w.watchFile()
			}
			time.Sleep(1 * time.Second)
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
	w.eventSender.SendEvent(&NotificationLogEvent{Event: &notificationEvent})
}
