# Claude Companion

A real-time parser and viewer for Claude's JSONL log files. This tool helps you monitor and analyze Claude sessions by parsing structured log events and displaying them in a human-readable format.

## Features

- **Real-time monitoring**: Tail Claude's JSONL log files and display new events as they appear
- **Full file reading**: Process entire session files from beginning to end
- **Structured parsing**: Parse and format different event types with type-safe Go structs
- **Human-readable output**: Display events with timestamps and formatted content
- **Companion mode**: Enhanced formatting with emojis, code block extraction, and file operation tracking
- **Large file support**: Handle JSONL lines up to 1MB in size

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

# Use a specific file path directly
./claude-companion -file /path/to/session.jsonl

# Disable companion mode for simpler output
./claude-companion -project PROJECT_NAME -session SESSION_ID -companion=false
```

### Using Make

```bash
# Tail mode
make run PROJECT=project_name SESSION=session_id

# Full read mode
make run PROJECT=project_name SESSION=session_id FULL=1
```

### Command Line Options

- `-project`: Project name (required when not using -file)
- `-session`: Session ID without .jsonl extension (required when not using -file)
- `-file`: Direct path to a session file (alternative to -project/-session)
- `-full`: Read entire file instead of tailing (optional)
- `-companion`: Enable companion mode with enhanced formatting (default: true)
- `-narrator`: Narrator mode for tool actions: rule, ai, or off (default: rule)
- `-openai-key`: OpenAI API key for AI narrator mode (optional, can use OPENAI_API_KEY env var)

## Operating Modes

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

### Companion Mode

Companion mode (enabled by default) provides enhanced formatting for better readability:

- **Natural language narration**: Describes tool actions in Japanese (💬)
- **Emoji indicators**: Visual cues for different event types (🤖 for assistant, 👤 for user, etc.)
- **Code block extraction**: Automatically detects and formats code blocks from assistant responses
- **Tool visualization**: Shows tool executions with descriptive icons (📄 for file reads, ✏️ for writes, etc.)
- **Smart truncation**: Long messages are intelligently truncated to maintain readability
- **File operation tracking**: Summarizes all file operations at the end of each assistant message

To disable companion mode and use simple formatting, use `-companion=false`.

#### Narrator Feature

The companion mode includes a narrator that describes tool actions in natural language:

```bash
# Use rule-based narrator (default)
./claude-companion -project myproject -session mysession -narrator=rule

# Use AI-powered narrator (requires OpenAI API key)
export OPENAI_API_KEY=your-api-key
./claude-companion -project myproject -session mysession -narrator=ai

# Disable narrator
./claude-companion -project myproject -session mysession -narrator=off
```

Example output with narrator:
```
[15:30:45] 🤖 ASSISTANT (claude-3-opus):
  💬 ファイル「main.go」を読み込みます
  💬 テストを実行します
  💬 変更をGitにコミットします
```

The narrator supports common development tools and commands:
- File operations (Read, Write, Edit)
- Git commands (commit, push, pull, etc.)
- Build tools (make, go build, npm, etc.)
- Search operations (grep, find, etc.)
- Web operations (fetch, search)

## Event Types

The tool recognizes and formats the following event types:

### 1. User Events (`user`)

User input messages, displayed with timestamp and content.

**Standard mode:**
```
[15:04:05] USER: Hello, Claude!
```

**Companion mode:**
```
[15:04:05] 👤 USER:
  Hello, Claude!
```

For complex inputs (tool results), companion mode provides clearer indicators:
```
[15:04:05] 👤 USER:
  🎯 Command execution
[15:04:06] 👤 USER:
  ✅ Tool Result: toolu_123456
```

### 2. Assistant Events (`assistant`)

Claude's responses, showing model name, content, and token usage.

**Standard mode:**
```
[15:04:06] ASSISTANT (claude-opus-4-20250514):
  Text: Hello! How can I help you today?
  Tool Use: WebSearch (id: toolu_789)
    Input: {
      "query": "latest news"
    }
  Tokens: input=10, output=20, cache_read=100, cache_creation=50
```

**Companion mode:**
```
[15:04:06] 🤖 ASSISTANT (claude-opus-4-20250514):
  Hello! How can I help you today?
  
  📝 Code Block 1 (python):
    def hello_world():
        print("Hello, World!")
    ... (10 more lines)
  
  🌐 Fetching: https://example.com
  📄 Reading file: /path/to/file.txt
  💰 Tokens: in=10, out=20, cache=100
  
📁 File Operations Summary:
  - Read: /path/to/file.txt
  - Write: /path/to/output.txt
```

### 3. System Events (`system`)

System messages with optional severity levels.

**Standard mode:**
```
[15:04:07] SYSTEM [info]: Tool execution completed successfully
[15:04:08] SYSTEM [warning]: Rate limit approaching
```

**Companion mode:**
```
[15:04:07] ℹ️ SYSTEM [info]: Tool execution completed successfully
[15:04:08] ⚠️ SYSTEM [warning]: Rate limit approaching
[15:04:09] ❌ SYSTEM [error]: Connection failed
```

### 4. Summary Events (`summary`)

Session summaries that provide high-level descriptions.

**Standard mode:**
```
[SUMMARY] Code Review: Python Web Application Security Analysis
```

**Companion mode:**
```
📋 [SUMMARY] Code Review: Python Web Application Security Analysis
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
├── main.go                # Main application logic and CLI
├── types.go              # Event type definitions
├── parser.go             # Event parsing and formatting logic
├── parser_test.go        # Unit tests for parser
├── companion_formatter.go # Companion mode formatting utilities
├── narrator.go           # Natural language narrator for tool actions
├── Makefile              # Build automation
├── CLAUDE.md             # Instructions for Claude
└── README.md             # This file
```

## Contributing

1. Follow the coding standards in CLAUDE.md
2. Run `make fmt` before committing
3. Ensure all tests pass with `make test`
4. Write clear commit messages

## License

MIT License