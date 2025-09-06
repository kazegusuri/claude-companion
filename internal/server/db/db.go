package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/google/uuid"
)

// ClaudeAgent represents a claude agent record in the database
type ClaudeAgent struct {
	ID         string // UUID primary key
	PID        int
	SessionID  string
	ProjectDir string
	AgentType  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// DB wraps the SQL database connection
type DB struct {
	conn *sql.DB
}

// Open opens a connection to the SQLite database
func Open(dbFile string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize the database schema
	if err := db.CreateTables(); err != nil {
		conn.Close()
		return nil, err
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// CreateTables creates the necessary tables if they don't exist
func (db *DB) CreateTables() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS claude_agents (
		id TEXT PRIMARY KEY,
		pid INTEGER NOT NULL UNIQUE,
		session_id TEXT NOT NULL,
		project_dir TEXT NOT NULL,
		agent_type TEXT NOT NULL DEFAULT 'Claude Code',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	);
	`

	if _, err := db.conn.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// UpsertClaudeAgent inserts or updates a claude agent record
func (db *DB) UpsertClaudeAgent(pid int, sessionID string, projectDir string) error {
	return db.UpsertClaudeAgentWithType(pid, sessionID, projectDir, "Claude Code")
}

// UpsertClaudeAgentWithType inserts or updates a claude agent record with agent type
func (db *DB) UpsertClaudeAgentWithType(pid int, sessionID string, projectDir string, agentType string) error {
	now := time.Now()
	id := uuid.New().String()

	// Use ON CONFLICT with PID's UNIQUE constraint
	upsertSQL := `
	INSERT INTO claude_agents (id, pid, session_id, project_dir, agent_type, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(pid) DO UPDATE SET
		session_id = excluded.session_id,
		project_dir = excluded.project_dir,
		agent_type = excluded.agent_type,
		updated_at = excluded.updated_at;
	`

	if _, err := db.conn.Exec(upsertSQL, id, pid, sessionID, projectDir, agentType, now, now); err != nil {
		return fmt.Errorf("failed to insert/update record: %w", err)
	}

	return nil
}

// GetClaudeAgent retrieves a claude agent record by PID
func (db *DB) GetClaudeAgent(pid int) (*ClaudeAgent, error) {
	query := `
	SELECT id, pid, session_id, project_dir, agent_type, created_at, updated_at
	FROM claude_agents
	WHERE pid = ?
	`

	var agent ClaudeAgent
	err := db.conn.QueryRow(query, pid).Scan(
		&agent.ID,
		&agent.PID,
		&agent.SessionID,
		&agent.ProjectDir,
		&agent.AgentType,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	return &agent, nil
}

// ListClaudeAgents returns all claude agents
func (db *DB) ListClaudeAgents() ([]ClaudeAgent, error) {
	query := `
	SELECT id, pid, session_id, project_dir, agent_type, created_at, updated_at
	FROM claude_agents
	ORDER BY updated_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	var agents []ClaudeAgent
	for rows.Next() {
		var agent ClaudeAgent
		err := rows.Scan(
			&agent.ID,
			&agent.PID,
			&agent.SessionID,
			&agent.ProjectDir,
			&agent.AgentType,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return agents, nil
}

// DeleteOldAgents deletes agents that haven't been updated for the specified duration
func (db *DB) DeleteOldAgents(olderThan time.Duration) (int64, error) {
	threshold := time.Now().Add(-olderThan)

	deleteSQL := `
	DELETE FROM claude_agents
	WHERE updated_at < ?
	`

	result, err := db.conn.Exec(deleteSQL, threshold)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old agents: %w", err)
	}

	return result.RowsAffected()
}

// GetClaudeAgentByID retrieves a claude agent record by ID
func (db *DB) GetClaudeAgentByID(id string) (*ClaudeAgent, error) {
	query := `
	SELECT id, pid, session_id, project_dir, agent_type, created_at, updated_at
	FROM claude_agents
	WHERE id = ?
	`

	var agent ClaudeAgent
	err := db.conn.QueryRow(query, id).Scan(
		&agent.ID,
		&agent.PID,
		&agent.SessionID,
		&agent.ProjectDir,
		&agent.AgentType,
		&agent.CreatedAt,
		&agent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query agent: %w", err)
	}

	return &agent, nil
}

// DeleteClaudeAgent deletes a claude agent record by PID
func (db *DB) DeleteClaudeAgent(pid int) error {
	deleteSQL := `
	DELETE FROM claude_agents
	WHERE pid = ?
	`

	_, err := db.conn.Exec(deleteSQL, pid)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	return nil
}
