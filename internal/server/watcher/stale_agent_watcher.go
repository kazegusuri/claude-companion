package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/kazegusuri/claude-companion/internal/server/db"
)

type StaleAgentWatcher struct {
	db       *db.DB
	interval time.Duration
	logger   *slog.Logger
}

func NewStaleAgentWatcher(database *db.DB, interval time.Duration, logger *slog.Logger) *StaleAgentWatcher {
	return &StaleAgentWatcher{
		db:       database,
		interval: interval,
		logger:   logger,
	}
}

func (w *StaleAgentWatcher) Start(ctx context.Context) {
	w.logger.Info("Starting agent watcher", "interval", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// 初回はすぐに実行
	w.checkAgents()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Stopping agent watcher")
			return
		case <-ticker.C:
			w.checkAgents()
		}
	}
}

func (w *StaleAgentWatcher) checkAgents() {
	agents, err := w.db.ListClaudeAgents()
	if err != nil {
		w.logger.Error("Failed to list claude agents", "error", err)
		return
	}

	for _, agent := range agents {
		if !isProcessRunning(agent.PID) {
			w.logger.Info("Removing stale agent", "pid", agent.PID, "project_dir", agent.ProjectDir, "session_id", agent.SessionID)
			if err := w.db.DeleteClaudeAgent(agent.PID); err != nil {
				w.logger.Error("Failed to delete stale agent", "pid", agent.PID, "error", err)
			}
		}
	}
}

// isProcessRunning checks if a process with the given PID exists
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if the process exists
	// This doesn't actually send a signal, just checks if we can
	err = process.Signal(syscall.Signal(0))
	if err != nil {
		// Process doesn't exist or we don't have permission
		if err == os.ErrProcessDone || err.Error() == "os: process already finished" {
			return false
		}
		// On Unix systems, ESRCH means the process doesn't exist
		if err == syscall.ESRCH {
			return false
		}
		// For "no such process" errors
		if err.Error() == fmt.Sprintf("os: process already released") {
			return false
		}
		return false
	}

	return true
}
