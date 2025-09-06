package event

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kazegusuri/claude-companion/internal/server/handler"
	"github.com/kazegusuri/claude-companion/narrator"
)

func TestSessionFileManager(t *testing.T) {
	// Create a mock handler with session manager
	sessionManager := handler.NewSessionManager()
	handler := NewHandler(narrator.NewNoOpNarrator(), sessionManager, false)

	manager := NewSessionFileManager(handler)
	manager.idleTimeout = 100 * time.Millisecond // Short timeout for testing
	manager.checkInterval = 50 * time.Millisecond

	// Start the manager
	manager.Start()
	defer manager.Stop()

	// Create a temp file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(testFile, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add a watcher
	if err := manager.AddOrUpdateWatcher(testFile); err != nil {
		t.Errorf("Failed to add watcher: %v", err)
	}

	// Check active count
	if count := manager.GetActiveWatcherCount(); count != 1 {
		t.Errorf("Expected 1 active watcher, got %d", count)
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Check that watcher was cleaned up
	if count := manager.GetActiveWatcherCount(); count != 0 {
		t.Errorf("Expected 0 active watchers after cleanup, got %d", count)
	}
}

func TestProjectsWatcherInitialization(t *testing.T) {
	// Create a temp directory
	tmpDir := t.TempDir()

	// Create a mock handler
	sessionManager := handler.NewSessionManager()
	handler := NewHandler(narrator.NewNoOpNarrator(), sessionManager, false)

	// Create projects watcher
	watcher, err := NewProjectsWatcher(tmpDir, handler)
	if err != nil {
		t.Fatalf("Failed to create projects watcher: %v", err)
	}

	// Check that root path was set correctly
	if watcher.rootPath != tmpDir {
		t.Errorf("Expected root path %s, got %s", tmpDir, watcher.rootPath)
	}

	// Cleanup
	watcher.Stop()
}

func TestProjectsWatcherHomeExpansion(t *testing.T) {
	// Create a mock handler
	sessionManager := handler.NewSessionManager()
	handler := NewHandler(narrator.NewNoOpNarrator(), sessionManager, false)

	// Test home directory expansion
	watcher, err := NewProjectsWatcher("~/test", handler)
	if err != nil {
		t.Fatalf("Failed to create projects watcher: %v", err)
	}
	defer watcher.Stop()

	// Check that ~ was expanded
	if len(watcher.rootPath) == 0 || watcher.rootPath[0] == '~' {
		t.Errorf("Home directory not expanded: %s", watcher.rootPath)
	}
}
