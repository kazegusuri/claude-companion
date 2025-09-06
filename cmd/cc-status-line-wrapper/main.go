package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/kazegusuri/claude-companion/internal/server/db"
)

// StatusInput represents the JSON input structure
type StatusInput struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	SessionID string `json:"session_id"`
}

func main() {
	// Parse command line flags
	var logFile string
	var dbFile string
	flag.StringVar(&logFile, "log-file", "", "Path to log file (optional)")
	flag.StringVar(&dbFile, "db-file", "", "Path to SQLite database file (optional)")
	flag.Parse()

	// Read JSON input from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read stdin: %v", err)
	}

	// Find -- separator in os.Args to get command to execute
	var execArgs []string
	for i, arg := range os.Args {
		if arg == "--" {
			if i+1 < len(os.Args) {
				execArgs = os.Args[i+1:]
			} else {
				log.Fatalf("Error: No command provided after --")
			}
			break
		}
	}

	// Handle logging if called from claude
	if logFile != "" || dbFile != "" {
		logFromClaude(input, logFile, dbFile)
	}

	// Execute command if provided
	if len(execArgs) > 0 {
		executeCommand(input, execArgs)
	}
}

// logFromClaude handles logging when called from claude process
func logFromClaude(input []byte, logFile string, dbFile string) {
	// Parse JSON
	var statusInput StatusInput
	if err := json.Unmarshal(input, &statusInput); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for logging: %v\n", err)
		return
	}

	// Get parent process ID (equivalent to $PPID in bash)
	ppid := os.Getppid()

	// Get grandparent process ID
	grandparentPID := getGrandparentPID(ppid)

	// Get grandparent process command name
	grandparentCmd := getProcessCommand(grandparentPID)

	// Check if grandparent command contains "claude"
	if !strings.Contains(grandparentCmd, "claude") {
		fmt.Fprintf(os.Stderr, "Warning: Not called from claude process (grandparent: %s), skipping log\n", grandparentCmd)
		return
	}

	// Write to log file if specified
	if logFile != "" {
		writeLogEntry(logFile, grandparentPID, statusInput.SessionID, statusInput.Workspace.ProjectDir)
	}

	// Write to database if specified
	if dbFile != "" {
		if err := writeDBEntry(dbFile, grandparentPID, statusInput.SessionID, statusInput.Workspace.ProjectDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write to database: %v\n", err)
		}
	}
}

// executeCommand executes the command with the given arguments
func executeCommand(input []byte, execArgs []string) {
	cmd := exec.Command(execArgs[0], execArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass the original JSON input to the command's stdin
	cmd.Stdin = bytes.NewReader(input)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("Failed to execute command: %v", err)
	}
}

// getGrandparentPID retrieves the parent PID of the given process
func getGrandparentPID(ppid int) int {
	// Read the status file of the parent process
	statusFile := fmt.Sprintf("/proc/%d/status", ppid)
	data, err := os.ReadFile(statusFile)
	if err != nil {
		return 0
	}

	// Parse the PPid field from the status file
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PPid:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if pid, err := strconv.Atoi(fields[1]); err == nil {
					return pid
				}
			}
		}
	}
	return 0
}

// getProcessCommand retrieves the command name of a process
func getProcessCommand(pid int) string {
	if pid == 0 {
		return "unknown"
	}

	// Try reading from /proc/[pid]/comm
	commFile := fmt.Sprintf("/proc/%d/comm", pid)
	data, err := os.ReadFile(commFile)
	if err != nil {
		// Fallback to reading cmdline if comm fails
		cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", pid)
		data, err = os.ReadFile(cmdlineFile)
		if err != nil {
			return "unknown"
		}
		// cmdline is null-separated, get the first part
		parts := strings.Split(string(data), "\x00")
		if len(parts) > 0 && parts[0] != "" {
			return filepath.Base(parts[0])
		}
		return "unknown"
	}

	// comm file contains the command name with a newline
	return strings.TrimSpace(string(data))
}

// writeLogEntry writes a log entry to the specified file
// Errors are logged to stderr but don't stop execution
func writeLogEntry(logFile string, grandparentPID int, sessionID string, projectDir string) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Log error to stderr but continue
		fmt.Fprintf(os.Stderr, "Error: Failed to open log file %s: %v\n", logFile, err)
		return
	}
	defer file.Close()

	logEntry := fmt.Sprintf("%d,%s,%s\n",
		grandparentPID,
		sessionID,
		projectDir)

	if _, err := file.WriteString(logEntry); err != nil {
		// Log write error to stderr
		fmt.Fprintf(os.Stderr, "Error: Failed to write to log file %s: %v\n", logFile, err)
	}
}

// Alternative implementation using syscall for getting PPID (more portable)
func getParentPID() int {
	return syscall.Getppid()
}

// writeDBEntry writes an entry to the SQLite database
func writeDBEntry(dbFile string, pid int, sessionID string, projectDir string) error {
	// Open database connection
	database, err := db.Open(dbFile)
	if err != nil {
		return err
	}
	defer database.Close()

	// Insert or update the record
	if err := database.UpsertClaudeAgent(pid, sessionID, projectDir); err != nil {
		return err
	}

	return nil
}
