package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	// Create a temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "test-db")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbFile := filepath.Join(tmpDir, "test.db")

	// Open database
	database, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Test UpsertClaudeAgent
	pid := 12345
	sessionID := "test-session-123"
	projectDir := "/test/project"

	err = database.UpsertClaudeAgent(pid, sessionID, projectDir)
	if err != nil {
		t.Fatalf("Failed to upsert agent: %v", err)
	}

	// Test GetClaudeAgent
	agent, err := database.GetClaudeAgent(pid)
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Agent should not be nil")
	}

	if agent.PID != pid {
		t.Errorf("Wrong PID: got %d, want %d", agent.PID, pid)
	}
	if agent.SessionID != sessionID {
		t.Errorf("Wrong session ID: got %s, want %s", agent.SessionID, sessionID)
	}
	if agent.ProjectDir != projectDir {
		t.Errorf("Wrong project dir: got %s, want %s", agent.ProjectDir, projectDir)
	}

	// Test update (upsert with same PID)
	newSessionID := "updated-session-456"
	err = database.UpsertClaudeAgent(pid, newSessionID, projectDir)
	if err != nil {
		t.Fatalf("Failed to update agent: %v", err)
	}

	agent, err = database.GetClaudeAgent(pid)
	if err != nil {
		t.Fatalf("Failed to get updated agent: %v", err)
	}

	if agent.SessionID != newSessionID {
		t.Errorf("Session ID not updated: got %s, want %s", agent.SessionID, newSessionID)
	}

	// Test ListClaudeAgents
	// Add another agent
	pid2 := 67890
	err = database.UpsertClaudeAgent(pid2, "session-2", "/project/2")
	if err != nil {
		t.Fatalf("Failed to insert second agent: %v", err)
	}

	agents, err := database.ListClaudeAgents()
	if err != nil {
		t.Fatalf("Failed to list agents: %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("Wrong number of agents: got %d, want 2", len(agents))
	}

	// Test GetClaudeAgent with non-existent PID
	nonExistent, err := database.GetClaudeAgent(99999)
	if err != nil {
		t.Fatalf("Unexpected error for non-existent agent: %v", err)
	}
	if nonExistent != nil {
		t.Error("Non-existent agent should be nil")
	}
}

func TestDeleteOldAgents(t *testing.T) {
	// Create a temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "test-db-delete")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbFile := filepath.Join(tmpDir, "test.db")

	// Open database
	database, err := Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Insert agents
	err = database.UpsertClaudeAgent(1001, "session-1", "/project/1")
	if err != nil {
		t.Fatalf("Failed to insert agent 1: %v", err)
	}

	// Sleep briefly to ensure time difference
	time.Sleep(10 * time.Millisecond)

	err = database.UpsertClaudeAgent(1002, "session-2", "/project/2")
	if err != nil {
		t.Fatalf("Failed to insert agent 2: %v", err)
	}

	// Delete agents older than 5 milliseconds
	deleted, err := database.DeleteOldAgents(5 * time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to delete old agents: %v", err)
	}

	// At least the first agent should be deleted
	if deleted < 1 {
		t.Errorf("Expected at least 1 agent to be deleted, got %d", deleted)
	}

	// List remaining agents
	agents, err := database.ListClaudeAgents()
	if err != nil {
		t.Fatalf("Failed to list agents after deletion: %v", err)
	}

	// Should have at most 1 agent remaining
	if len(agents) > 1 {
		t.Errorf("Too many agents remaining: got %d, want <= 1", len(agents))
	}
}
