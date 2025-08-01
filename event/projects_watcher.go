package event

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/kazegusuri/claude-companion/logger"
)

// ProjectsWatcher watches the ~/.claude/projects directory for changes
type ProjectsWatcher struct {
	rootPath       string
	watcher        *fsnotify.Watcher
	sessionManager *SessionFileManager
	debugMode      bool
	done           chan struct{}
	wg             sync.WaitGroup
	projectFilter  string
	sessionFilter  string
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
		debugMode:      handler.debugMode,
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

	if w.debugMode {
		logger.LogInfo("Started watching projects directory: %s", w.rootPath)
	}
	return nil
}

// SetProjectFilter sets the project filter
func (w *ProjectsWatcher) SetProjectFilter(project string) {
	w.projectFilter = project
}

// SetSessionFilter sets the session filter
func (w *ProjectsWatcher) SetSessionFilter(session string) {
	w.sessionFilter = session
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
				logger.LogWarning("Skipping directory due to permission error: %s", path)
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

			// Apply project filter
			if w.projectFilter != "" {
				// Check if this directory is under the projects root
				rel, err := filepath.Rel(w.rootPath, path)
				if err == nil && rel != "." {
					// Get the project name (first component of relative path)
					parts := strings.Split(rel, string(filepath.Separator))
					if len(parts) > 0 && parts[0] != w.projectFilter {
						return filepath.SkipDir
					}
				}
			}

			if err := w.watcher.Add(path); err != nil {
				if w.debugMode {
					logger.LogError("Error adding directory to watcher: %s - %v", path, err)
				}
			} else {
				if w.debugMode {
					logger.LogInfo("Watching directory: %s", path)
				}
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
			logger.LogError("Watcher error: %v", err)

		case <-w.done:
			return
		}
	}
}

// shouldProcessFile checks if a file should be processed based on filters
func (w *ProjectsWatcher) shouldProcessFile(path string) bool {
	// Apply project filter
	if w.projectFilter != "" {
		rel, err := filepath.Rel(w.rootPath, path)
		if err != nil {
			return false
		}
		parts := strings.Split(rel, string(filepath.Separator))
		if len(parts) == 0 || parts[0] != w.projectFilter {
			return false
		}
	}

	// Apply session filter
	if w.sessionFilter != "" {
		base := filepath.Base(path)
		// Remove .jsonl extension
		sessionName := strings.TrimSuffix(base, ".jsonl")
		if sessionName != w.sessionFilter {
			return false
		}
	}

	return true
}

// handleEvent processes file system events
func (w *ProjectsWatcher) handleEvent(event fsnotify.Event) {
	// Check if it's a .jsonl file
	if !strings.HasSuffix(event.Name, ".jsonl") {
		// If it's a new directory, add it to the watcher
		if event.Op&fsnotify.Create == fsnotify.Create {
			if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
				// Check if we should watch this directory based on project filter
				if w.projectFilter != "" {
					rel, err := filepath.Rel(w.rootPath, event.Name)
					if err == nil && rel != "." {
						parts := strings.Split(rel, string(filepath.Separator))
						if len(parts) > 0 && parts[0] != w.projectFilter {
							return
						}
					}
				}
				if err := w.addDirectoryTree(event.Name); err != nil {
					logger.LogError("Error adding new directory: %v", err)
				}
			}
		}
		return
	}

	// Check if we should process this file based on filters
	if !w.shouldProcessFile(event.Name) {
		return
	}

	// Handle .jsonl file events
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		if w.debugMode {
			logger.LogInfo("New session file created: %s", event.Name)
		}
		if err := w.sessionManager.AddOrUpdateWatcher(event.Name); err != nil {
			logger.LogError("Error creating watcher for new file: %v", err)
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		if w.debugMode {
			logger.LogInfo("Session file updated: %s", event.Name)
		}
		if err := w.sessionManager.AddOrUpdateWatcher(event.Name); err != nil {
			logger.LogError("Error updating watcher for file: %v", err)
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		if w.debugMode {
			logger.LogInfo("Session file removed: %s", event.Name)
		}
		// The session manager will clean it up automatically on idle timeout
	}
}

// GetActiveWatcherCount returns the number of active session watchers
func (w *ProjectsWatcher) GetActiveWatcherCount() int {
	return w.sessionManager.GetActiveWatcherCount()
}
