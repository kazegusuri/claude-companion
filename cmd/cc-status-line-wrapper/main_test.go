package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kazegusuri/claude-companion/internal/server/db"
)

// TestBuildBinary builds the binary for testing
func TestBuildBinary(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", "cc-status-line-wrapper", ".")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}
	t.Cleanup(func() {
		os.Remove("cc-status-line-wrapper")
	})
}

// TestWrapper tests execution where:
// - grandparent is a Go binary named 'claude'
// - parent is a shell script
// - child is the cc-status-line-wrapper
func TestWrapper(t *testing.T) {
	// Build the wrapper binary
	TestBuildBinary(t)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-claude-grandparent")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "claude-test.log")

	// Get absolute path to the wrapper binary
	absBinary, err := filepath.Abs("./cc-status-line-wrapper")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Build the claude binary (grandparent) from testdata
	claudeBinary := filepath.Join(tmpDir, "claude")
	buildCmd := exec.Command("go", "build", "-o", claudeBinary, "./testdata/claude_main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build claude binary: %v", err)
	}

	// Create a shell script (parent) that calls the wrapper
	shellScript := filepath.Join(tmpDir, "run-wrapper.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# This is the parent shell script
echo '%s' | %s --log-file %s
`,
		`{"model":{"display_name":"Claude 3.5"},"workspace":{"current_dir":"/test/dir","project_dir":"/test/project"},"session_id":"claude-grandparent-test"}`,
		absBinary,
		logFile,
	)
	if err := os.WriteFile(shellScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create shell script: %v", err)
	}

	// Execute: claude (Go binary) -> shell script -> cc-status-line-wrapper
	// Now the grandparent of cc-status-line-wrapper should be 'claude'
	cmd := exec.Command(claudeBinary, shellScript)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "Not called from claude") {
			t.Errorf("Should have been detected as claude but wasn't: %s", stderrStr)
		}
	}

	// Check if log file was created - SHOULD be created when grandparent is 'claude'
	if _, err := os.Stat(logFile); err == nil {
		content, _ := os.ReadFile(logFile)

		// Verify the content
		if !strings.Contains(string(content), "claude-grandparent-test") {
			t.Error("Log file doesn't contain expected session ID")
		}
		if !strings.Contains(string(content), "/test/project") {
			t.Error("Log file doesn't contain expected project directory")
		}
	} else {
		t.Error("Log file should have been created when grandparent is 'claude'")
	}
}

// TestWrapperWithExec tests that the wrapper can execute a command after --
func TestWrapperWithExec(t *testing.T) {
	// Build the wrapper binary
	TestBuildBinary(t)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-claude-exec")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "claude-test.log")
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Get absolute path to the wrapper binary
	absBinary, err := filepath.Abs("./cc-status-line-wrapper")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Build the claude binary (grandparent) from testdata
	claudeBinary := filepath.Join(tmpDir, "claude")
	buildCmd := exec.Command("go", "build", "-o", claudeBinary, "./testdata/claude_main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build claude binary: %v", err)
	}

	// Create a simple shell script to execute after --
	testScript := filepath.Join(tmpDir, "test.sh")
	testScriptContent := fmt.Sprintf(`#!/bin/bash
echo "Executed successfully" > %s
`, outputFile)
	if err := os.WriteFile(testScript, []byte(testScriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Create a shell script (parent) that calls the wrapper with -- argument
	shellScript := filepath.Join(tmpDir, "run-wrapper.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# This is the parent shell script
echo '%s' | "%s" --log-file "%s" -- "%s"
`,
		`{"model":{"display_name":"Claude 3.5"},"workspace":{"current_dir":"/test/dir","project_dir":"/test/project"},"session_id":"claude-exec-test"}`,
		absBinary,
		logFile,
		testScript,
	)
	if err := os.WriteFile(shellScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create shell script: %v", err)
	}

	// Execute: claude (Go binary) -> shell script -> cc-status-line-wrapper -> test.sh
	cmd := exec.Command(claudeBinary, shellScript)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v, stderr: %s, stdout: %s", err, stderr.String(), stdout.String())
	}

	// Check if the test script was executed
	if _, err := os.Stat(outputFile); err != nil {
		t.Error("Test script was not executed - output file not created")
	} else {
		content, _ := os.ReadFile(outputFile)
		if !strings.Contains(string(content), "Executed successfully") {
			t.Errorf("Output file doesn't contain expected content: %s", string(content))
		}
	}

	// Check if log file was still created
	if _, err := os.Stat(logFile); err == nil {
		content, _ := os.ReadFile(logFile)
		if !strings.Contains(string(content), "claude-exec-test") {
			t.Error("Log file doesn't contain expected session ID")
		}
	} else {
		t.Error("Log file should have been created")
	}
}

// TestWrapperWithExtraArgs tests that extra arguments before -- are ignored
func TestWrapperWithExtraArgs(t *testing.T) {
	// Build the wrapper binary
	TestBuildBinary(t)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-claude-extra-args")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "output.txt")

	// Get absolute path to the wrapper binary
	absBinary, err := filepath.Abs("./cc-status-line-wrapper")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Build the claude binary (grandparent) from testdata
	claudeBinary := filepath.Join(tmpDir, "claude")
	buildCmd := exec.Command("go", "build", "-o", claudeBinary, "./testdata/claude_main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build claude binary: %v", err)
	}

	// Create a simple shell script to execute after --
	testScript := filepath.Join(tmpDir, "test.sh")
	testScriptContent := fmt.Sprintf(`#!/bin/bash
echo "Extra args ignored" > %s
`, outputFile)
	if err := os.WriteFile(testScript, []byte(testScriptContent), 0755); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	// Create a shell script (parent) that calls the wrapper with extra args before --
	shellScript := filepath.Join(tmpDir, "run-wrapper.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# This is the parent shell script with extra args before --
echo '%s' | "%s" extra arg1 arg2 -- "%s"
`,
		`{"model":{"display_name":"Claude 3.5"},"workspace":{"current_dir":"/test/dir","project_dir":"/test/project"},"session_id":"claude-extra-args-test"}`,
		absBinary,
		testScript,
	)
	if err := os.WriteFile(shellScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create shell script: %v", err)
	}

	// Execute: claude (Go binary) -> shell script -> cc-status-line-wrapper -> test.sh
	cmd := exec.Command(claudeBinary, shellScript)

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check if the test script was executed (extra args should be ignored)
	if _, err := os.Stat(outputFile); err != nil {
		t.Error("Test script was not executed - extra args before -- should be ignored")
	} else {
		content, _ := os.ReadFile(outputFile)
		if !strings.Contains(string(content), "Extra args ignored") {
			t.Errorf("Output file doesn't contain expected content: %s", string(content))
		}
	}
}

// TestWrapperPassesStdin tests that the wrapper passes JSON input to the executed command
func TestWrapperPassesStdin(t *testing.T) {
	// Build the wrapper binary
	TestBuildBinary(t)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-claude-stdin")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "stdin-output.txt")

	// Get absolute path to the wrapper binary
	absBinary, err := filepath.Abs("./cc-status-line-wrapper")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Build the claude binary (grandparent) from testdata
	claudeBinary := filepath.Join(tmpDir, "claude")
	buildCmd := exec.Command("go", "build", "-o", claudeBinary, "./testdata/claude_main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build claude binary: %v", err)
	}

	// Create a shell script (parent) that calls the wrapper and pipes stdin to a file
	shellScript := filepath.Join(tmpDir, "run-wrapper.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# This is the parent shell script
echo '%s' | "%s" -- cat > "%s"
`,
		`{"model":{"display_name":"Claude 3.5"},"workspace":{"current_dir":"/test/dir","project_dir":"/test/project"},"session_id":"stdin-test"}`,
		absBinary,
		outputFile,
	)
	if err := os.WriteFile(shellScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create shell script: %v", err)
	}

	// Execute: claude (Go binary) -> shell script -> cc-status-line-wrapper -> cat
	cmd := exec.Command(claudeBinary, shellScript)

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check if stdin was passed to the command
	if _, err := os.Stat(outputFile); err != nil {
		t.Error("Output file not created - stdin was not passed to command")
	} else {
		content, _ := os.ReadFile(outputFile)
		// Verify the JSON content was passed through
		if !strings.Contains(string(content), "stdin-test") {
			t.Errorf("Output doesn't contain expected session ID from stdin: %s", string(content))
		}
		if !strings.Contains(string(content), "Claude 3.5") {
			t.Errorf("Output doesn't contain expected model name from stdin: %s", string(content))
		}
	}
}

// TestWrapperPassesStdout tests that the wrapper passes command stdout to its stdout
func TestWrapperPassesStdout(t *testing.T) {
	// Build the wrapper binary
	TestBuildBinary(t)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-claude-stdout")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	outputFile := filepath.Join(tmpDir, "stdout-capture.txt")

	// Get absolute path to the wrapper binary
	absBinary, err := filepath.Abs("./cc-status-line-wrapper")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Build the claude binary (grandparent) from testdata
	claudeBinary := filepath.Join(tmpDir, "claude")
	buildCmd := exec.Command("go", "build", "-o", claudeBinary, "./testdata/claude_main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build claude binary: %v", err)
	}

	// Create a shell script (parent) that calls the wrapper and captures stdout
	shellScript := filepath.Join(tmpDir, "run-wrapper.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# This is the parent shell script
echo '%s' | "%s" -- echo "Test stdout output" > "%s"
`,
		`{"model":{"display_name":"Claude 3.5"},"workspace":{"current_dir":"/test/dir","project_dir":"/test/project"},"session_id":"stdout-test"}`,
		absBinary,
		outputFile,
	)
	if err := os.WriteFile(shellScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create shell script: %v", err)
	}

	// Execute: claude (Go binary) -> shell script -> cc-status-line-wrapper -> echo
	cmd := exec.Command(claudeBinary, shellScript)

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check if stdout was passed through
	if _, err := os.Stat(outputFile); err != nil {
		t.Error("Output file not created - stdout was not passed through")
	} else {
		content, _ := os.ReadFile(outputFile)
		expectedOutput := "Test stdout output"
		if !strings.Contains(string(content), expectedOutput) {
			t.Errorf("Output doesn't contain expected stdout: got %q, want %q", string(content), expectedOutput)
		}
	}
}

// TestWrapperWithDatabase tests that the wrapper saves data to SQLite database
func TestWrapperWithDatabase(t *testing.T) {
	// Build the wrapper binary
	TestBuildBinary(t)

	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "test-claude-database")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbFile := filepath.Join(tmpDir, "test.db")

	// Get absolute path to the wrapper binary
	absBinary, err := filepath.Abs("./cc-status-line-wrapper")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	// Build the claude binary (grandparent) from testdata
	claudeBinary := filepath.Join(tmpDir, "claude")
	buildCmd := exec.Command("go", "build", "-o", claudeBinary, "./testdata/claude_main.go")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build claude binary: %v", err)
	}

	// Create a shell script (parent) that calls the wrapper with --db-file
	shellScript := filepath.Join(tmpDir, "run-wrapper.sh")
	scriptContent := fmt.Sprintf(`#!/bin/bash
# This is the parent shell script
echo '%s' | "%s" --db-file "%s"
`,
		`{"model":{"display_name":"Claude 3.5"},"workspace":{"current_dir":"/test/dir","project_dir":"/test/project"},"session_id":"db-test-session"}`,
		absBinary,
		dbFile,
	)
	if err := os.WriteFile(shellScript, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("Failed to create shell script: %v", err)
	}

	// Execute: claude (Go binary) -> shell script -> cc-status-line-wrapper
	cmd := exec.Command(claudeBinary, shellScript)

	err = cmd.Run()
	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Check if database file was created
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatal("Database file was not created")
	}

	// Connect to database and verify the data
	database, err := db.Open(dbFile)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Query the database - get all agents
	agents, err := database.ListClaudeAgents()
	if err != nil {
		t.Fatalf("Failed to list agents: %v", err)
	}

	// Verify we have at least one agent
	if len(agents) == 0 {
		t.Fatal("No agents found in database")
	}

	// Verify the first agent's data
	agent := agents[0]
	if agent.SessionID != "db-test-session" {
		t.Errorf("Wrong session ID in database: got %s, want db-test-session", agent.SessionID)
	}
	if agent.ProjectDir != "/test/project" {
		t.Errorf("Wrong project dir in database: got %s, want /test/project", agent.ProjectDir)
	}
	if agent.PID == 0 {
		t.Error("PID should not be 0")
	}
}
