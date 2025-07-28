package event

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// ProjectsWatcher watches the ~/.claude/projects directory for changes
type ProjectsWatcher struct {
	rootPath       string
	watcher        *fsnotify.Watcher
	sessionManager *SessionFileManager
	done           chan struct{}
	wg             sync.WaitGroup
}

// NewProjectsWatcher creates a new projects watcher
func NewProjectsWatcher(rootPath string, handler *Handler) (*ProjectsWatcher, error) {
	// Expand home directory if needed
	if strings.HasPrefix(rootPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		rootPath = filepath.Join(home, rootPath[2:])
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	sessionManager := NewSessionFileManager(handler)

	return &ProjectsWatcher{
		rootPath:       rootPath,
		watcher:        watcher,
		sessionManager: sessionManager,
		done:           make(chan struct{}),
	}, nil
}

// Start begins watching the projects directory
func (w *ProjectsWatcher) Start() error {
	// Start session manager
	w.sessionManager.Start()

	// Add root directory and all subdirectories
	if err := w.addDirectoryTree(w.rootPath); err != nil {
		return err
	}

	// Start watching
	w.wg.Add(1)
	go w.watch()

	log.Printf("Started watching projects directory: %s", w.rootPath)
	return nil
}

// Stop stops the watcher
func (w *ProjectsWatcher) Stop() {
	close(w.done)
	w.watcher.Close()
	w.wg.Wait()
	w.sessionManager.Stop()
}

// addDirectoryTree recursively adds directories to the watcher
func (w *ProjectsWatcher) addDirectoryTree(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip directories we can't access
			if os.IsPermission(err) {
				log.Printf("Skipping directory due to permission error: %s", path)
				return nil
			}
			return err
		}

		// Only watch directories
		if info.IsDir() {
			// Skip hidden directories (except .claude itself)
			if strings.HasPrefix(info.Name(), ".") && info.Name() != ".claude" && path != root {
				return filepath.SkipDir
			}

			if err := w.watcher.Add(path); err != nil {
				log.Printf("Error adding directory to watcher: %s - %v", path, err)
			} else {
				log.Printf("Watching directory: %s", path)
			}
		}

		return nil
	})
}

// watch handles file system events
func (w *ProjectsWatcher) watch() {
	defer w.wg.Done()

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)

		case <-w.done:
			return
		}
	}
}

// handleEvent processes file system events
func (w *ProjectsWatcher) handleEvent(event fsnotify.Event) {
	// Check if it's a .jsonl file
	if !strings.HasSuffix(event.Name, ".jsonl") {
		// If it's a new directory, add it to the watcher
		if event.Op&fsnotify.Create == fsnotify.Create {
			if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
				if err := w.addDirectoryTree(event.Name); err != nil {
					log.Printf("Error adding new directory: %v", err)
				}
			}
		}
		return
	}

	// Handle .jsonl file events
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		log.Printf("New session file created: %s", event.Name)
		if err := w.sessionManager.AddOrUpdateWatcher(event.Name); err != nil {
			log.Printf("Error creating watcher for new file: %v", err)
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		log.Printf("Session file updated: %s", event.Name)
		if err := w.sessionManager.AddOrUpdateWatcher(event.Name); err != nil {
			log.Printf("Error updating watcher for file: %v", err)
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		log.Printf("Session file removed: %s", event.Name)
		// The session manager will clean it up automatically on idle timeout
	}
}

// GetActiveWatcherCount returns the number of active session watchers
func (w *ProjectsWatcher) GetActiveWatcherCount() int {
	return w.sessionManager.GetActiveWatcherCount()
}
