# Claude Companion

A real-time parser and viewer for Claude's JSONL log files. This tool helps you monitor and analyze Claude sessions by parsing structured log events and displaying them in a human-readable format.

## Features

- **Real-time monitoring**: Tail Claude's JSONL log files and display new events as they appear
- **Full file reading**: Process entire session files from beginning to end
- **Structured parsing**: Parse and format different event types with type-safe Go structs
- **Human-readable output**: Display events with timestamps and formatted content

## Installation

```bash
# Clone the repository
git clone https://github.com/kazegusuri/claude-companion.git
cd claude-companion

# Build the binary
make build
```

## Usage

### Basic Usage

```bash
# Monitor a session in real-time (tail mode)
./claude-companion -project PROJECT_NAME -session SESSION_ID

# Read entire file from beginning to end
./claude-companion -project PROJECT_NAME -session SESSION_ID -full
```

### Using Make

```bash
# Tail mode
make run PROJECT=project_name SESSION=session_id

# Full read mode
make run PROJECT=project_name SESSION=session_id FULL=1
```

### Command Line Options

- `-project`: Project name (required)
- `-session`: Session ID without .jsonl extension (required)
- `-full`: Read entire file instead of tailing (optional)

## Modes

### Tail Mode (Default)

In tail mode, the tool:
- Opens the log file and seeks to the end
- Continuously monitors for new lines
- Displays events in real-time as they are written
- Runs indefinitely until interrupted (Ctrl+C)

This mode is useful for monitoring active Claude sessions.

### Full Read Mode

In full read mode (`-full` flag), the tool:
- Reads the file from the beginning
- Processes all events sequentially
- Exits after reaching the end of file
- Shows total line count when finished

This mode is useful for analyzing completed sessions or generating reports.

## Event Types

The tool recognizes and formats the following event types:

### 1. User Events (`user`)

User input messages, displayed with timestamp and content.

```
[15:04:05] USER: Hello, Claude!
```

For complex inputs (tool results), displays structured content:

```
[15:04:05] USER:
  Text: Here is my question...
  Tool Result: toolu_123456
```

### 2. Assistant Events (`assistant`)

Claude's responses, showing model name, content, and token usage.

```
[15:04:06] ASSISTANT (claude-opus-4-20250514):
  Text: Hello! How can I help you today?
  Tool Use: WebSearch (id: toolu_789)
    Input: {
      "query": "latest news"
    }
  Tokens: input=10, output=20, cache_read=100, cache_creation=50
```

### 3. System Events (`system`)

System messages with optional severity levels.

```
[15:04:07] SYSTEM [info]: Tool execution completed successfully
[15:04:08] SYSTEM [warning]: Rate limit approaching
```

### 4. Summary Events (`summary`)

Session summaries that provide high-level descriptions.

```
[SUMMARY] Code Review: Python Web Application Security Analysis
```

### 5. Other Events

Any unrecognized event types are displayed with their raw JSON content for debugging.

```
[15:04:09] unknown event
  Raw: {
    "type": "unknown",
    "data": "..."
  }
```

## Log File Location

Claude stores session logs in:
```
~/.claude/projects/PROJECT_NAME/SESSION_ID.jsonl
```

Each line in the JSONL file represents a single event with structured JSON data.

## Development

### Prerequisites

- Go 1.19 or higher
- Make

### Building

```bash
# Build the binary
make build

# Run tests
make test

# Format code
make fmt

# Clean build artifacts
make clean
```

### Project Structure

```
.
├── main.go         # Main application logic and CLI
├── types.go        # Event type definitions
├── Makefile        # Build automation
├── CLAUDE.md       # Instructions for Claude
└── README.md       # This file
```

## Contributing

1. Follow the coding standards in CLAUDE.md
2. Run `make fmt` before committing
3. Ensure all tests pass with `make test`
4. Write clear commit messages

## License

MIT License