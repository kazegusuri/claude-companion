package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Request struct {
	CWD       string `json:"cwd"`
	SessionID string `json:"sessionId"`
	Command   string `json:"command"`
	Message   string `json:"message"`
}

type Response struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	Error     string `json:"error,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

func main() {
	// Read JSON from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		outputError("Failed to read input", err)
	}

	// Parse JSON
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		outputError("Failed to parse JSON", err)
	}

	// Process command
	switch req.Command {
	case "proceed":
		handleProceed(req)
	case "stop":
		handleStop(req)
	case "send":
		handleSend(req)
	default:
		outputErrorMessage("Unknown command: " + req.Command)
	}
}

func outputError(message string, err error) {
	resp := Response{
		Success: false,
		Error:   message + ": " + err.Error(),
	}
	outputResponse(resp)
	os.Exit(1)
}

func outputErrorMessage(message string) {
	resp := Response{
		Success: false,
		Error:   message,
	}
	outputResponse(resp)
	os.Exit(1)
}

func outputResponse(resp Response) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(resp)
}

func execEmacsCommand(command string) error {
	// Create a temporary file for the Emacs command
	tmpFile, err := os.CreateTemp("", "emacs-cmd-*.el")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file
	defer tmpFile.Close()

	// Write the command to the temp file
	if _, err := tmpFile.WriteString(command); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Execute emacsclient with the temp file
	absPath, err := filepath.Abs(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cmd := exec.Command("emacsclient", "--eval", fmt.Sprintf(`(load-file "%s")`, absPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("emacsclient error: %w, output: %s", err, strings.TrimSpace(string(output)))
	}

	// Optionally log the output for debugging
	_ = output

	return nil
}

func handleProceed(req Request) {
	// Get base directory name from CWD
	basedir := filepath.Base(req.CWD)

	// Execute Emacs command for proceed (send Enter key)
	emacsCmd := fmt.Sprintf(`(with-current-buffer "*claude-code[%s]*" (vterm-send-return))`, basedir)
	if err := execEmacsCommand(emacsCmd); err != nil {
		outputError("Failed to execute Emacs command", err)
		return
	}

	resp := Response{
		Success:   true,
		Message:   "Proceed command executed successfully",
		SessionID: req.SessionID,
	}
	outputResponse(resp)
}

func handleStop(req Request) {
	// Get base directory name from CWD
	basedir := filepath.Base(req.CWD)

	// Execute Emacs command for stop (send Escape key)
	emacsCmd := fmt.Sprintf(`(with-current-buffer "*claude-code[%s]*" (vterm-send-key "<escape>"))`, basedir)
	if err := execEmacsCommand(emacsCmd); err != nil {
		outputError("Failed to execute Emacs command", err)
		return
	}

	resp := Response{
		Success:   true,
		Message:   "Stop command executed successfully",
		SessionID: req.SessionID,
	}
	outputResponse(resp)
}

func handleSend(req Request) {
	// Get base directory name from CWD
	basedir := filepath.Base(req.CWD)

	// Execute Emacs command for send (send the message string and press Enter)
	emacsCmd := fmt.Sprintf(`(with-current-buffer "*claude-code[%s]*" (vterm-send-string %q) (vterm-send-return))`, basedir, req.Message)
	if err := execEmacsCommand(emacsCmd); err != nil {
		outputError("Failed to execute Emacs command", err)
		return
	}

	resp := Response{
		Success:   true,
		Message:   fmt.Sprintf("Message sent: %s", req.Message),
		SessionID: req.SessionID,
	}
	outputResponse(resp)
}
