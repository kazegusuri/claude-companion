# cc-status-line-wrapper

A wrapper command for Claude status line that logs session information and optionally executes commands.

## Features

- Receives Claude session information via JSON from stdin
- Logs session data to file or SQLite database
- Executes commands with the original JSON input passed to their stdin
- Only logs when called from a Claude process (checks grandparent process)

## Installation

```bash
go build -o cc-status-line-wrapper ./cmd/cc-status-line-wrapper
```

## Usage

### Claude Configuration

Add to your Claude settings file `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "/usr/local/bin/cc-status-line-wrapper --log-file /var/log/claude-status-line.log --db-file /run/claude-companion/db.sqlite -- /usr/local/bin/claude-status-line"
  }
}
```

This configuration:
- Logs session information to `/var/log/claude-status-line.log`
- Stores data in SQLite database at `/run/claude-companion/db.sqlite`
- Executes the original `claude-status-line` command with the JSON input

**Note:** When using system directories, you need to prepare them first:

For the database directory:
```bash
sudo mkdir /run/claude-companion
sudo chmod 777 /run/claude-companion
```

For the log file:
```bash
sudo touch /var/log/claude-status-line.log
sudo chmod 666 /var/log/claude-status-line.log
```

## Options

| Flag | Description |
|------|-------------|
| `--log-file <path>` | Path to log file for CSV-style logging |
| `--db-file <path>` | Path to SQLite database file |
| `--` | Separator for command execution (everything after this is executed as a command) |

## Input Format

The wrapper expects JSON input via stdin with the following structure:

```json
{
  "model": {
    "display_name": "Model Name"
  },
  "workspace": {
    "current_dir": "/current/directory",
    "project_dir": "/project/directory"
  },
  "session_id": "unique-session-id"
}
```

## Log File Format

When using `--log-file`, data is written in CSV format:

```
<grandparent_pid>,<session_id>,<project_dir>
```

## Database Schema

When using `--db-file`, data is stored in SQLite with the following schema:

### Table: `claude_agents`

| Column | Type | Description |
|--------|------|-------------|
| `pid` | INTEGER | Process ID (Primary Key) |
| `session_id` | TEXT | Claude session identifier |
| `project_dir` | TEXT | Project directory path |
| `created_at` | TIMESTAMP | Record creation time |
| `updated_at` | TIMESTAMP | Last update time |

## Process Hierarchy Check

The wrapper checks if it's being called from a Claude process by examining the grandparent process name. If the grandparent process doesn't contain "claude" in its name, logging is skipped (but command execution still occurs if specified).