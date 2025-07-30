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

### Setting up Claude Hooks

Claude Companion requires configuring Claude to send notification events. This is done by setting up hooks in Claude's settings:

1. **Install the notification script**:
   ```bash
   # Copy the notification script to /usr/local/bin
   sudo cp script/claude-notification.sh /usr/local/bin/
   sudo chmod +x /usr/local/bin/claude-notification.sh
   
   # Create log file with appropriate permissions
   sudo touch /var/log/claude-notification.log
   sudo chmod 666 /var/log/claude-notification.log
   ```

2. **Configure Claude hooks** in `~/.claude/settings.json`:
   ```json
   {
     "hooks": {
       "SessionStart": [
         {
           "matcher": "*",
           "hooks": [
             {
               "type": "command",
               "command": "/usr/local/bin/claude-notification.sh"
             }
           ]
         }
       ],
       "PreCompact": [
         {
           "matcher": "*",
           "hooks": [
             {
               "type": "command",
               "command": "/usr/local/bin/claude-notification.sh"
             }
           ]
         }
       ],
       "Notification": [
         {
           "matcher": "",
           "hooks": [
             {
               "type": "command",
               "command": "/usr/local/bin/claude-notification.sh"
             }
           ]
         }
       ]
     }
   }
   ```

3. **Start Claude Companion** to monitor notification events:
   ```bash
   # In one terminal, start monitoring notifications
   ./claude-companion -file /var/log/claude-notification.log -full
   
   # In another terminal, monitor a specific Claude session
   ./claude-companion -project PROJECT_NAME -session SESSION_ID
   ```

The notification script will capture events from Claude and write them to `/var/log/claude-notification.log`, which Claude Companion can then parse and display in real-time.

## Usage

### Basic Usage

```bash
# Monitor a session in real-time (tail mode)
./claude-companion -project PROJECT_NAME -session SESSION_ID

# Read entire file from beginning to end
./claude-companion -project PROJECT_NAME -session SESSION_ID -full

# Use a specific file path directly
./claude-companion -file /path/to/session.jsonl

# Enable debug mode for detailed information
./claude-companion -project PROJECT_NAME -session SESSION_ID -debug
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
- `-debug`: Enable debug mode with detailed information (default: false)
- `-ai`: Use AI narrator instead of rule-based narrator (default: false)
- `-openai-key`: OpenAI API key for AI narrator (optional, can use OPENAI_API_KEY env var)
- `-narrator-config`: Path to custom narrator configuration file (optional)
- `-voice`: Enable voice output using VOICEVOX (default: false)
- `-voicevox-url`: VOICEVOX server URL (default: http://localhost:50021)
- `-voice-speaker`: VOICEVOX speaker ID (default: 1)

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

### Enhanced Formatting

The tool provides enhanced formatting for better readability:

- **Natural language narration**: Describes tool actions in Japanese (💬)
- **Emoji indicators**: Visual cues for different event types (🤖 for assistant, 👤 for user, etc.)
- **Code block extraction**: Automatically detects and formats code blocks from assistant responses
- **Tool visualization**: Shows tool executions with descriptive icons (📄 for file reads, ✏️ for writes, etc.)
- **Smart truncation**: Long messages are intelligently truncated to maintain readability
- **File operation tracking**: Summarizes all file operations at the end of each assistant message

### Debug Mode

Enable debug mode with `-debug` to see additional information:

- Event UUIDs and IDs
- Request IDs for assistant messages
- Stop reasons for assistant responses
- Meta system messages (normally hidden)
- Full content size information for truncated messages
- Tool use IDs for system messages

#### Narrator Feature

The tool includes a narrator that describes tool actions in natural language:

```bash
# Use rule-based narrator (default)
./claude-companion -project myproject -session mysession

# Use AI-powered narrator (requires OpenAI API key)
export OPENAI_API_KEY=your-api-key
./claude-companion -project myproject -session mysession -ai

# Use custom narrator configuration
./claude-companion -project myproject -session mysession -narrator-config=/path/to/config.json
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

#### Voice Output Feature

The tool can speak the narrator's descriptions using VOICEVOX text-to-speech engine:

```bash
# Prerequisites: Start VOICEVOX engine
# Download from: https://github.com/VOICEVOX/voicevox_engine
# Run: ./run (or run.exe on Windows)

# Enable voice output
./claude-companion -project myproject -session mysession -voice

# Use custom VOICEVOX server URL
./claude-companion -project myproject -session mysession -voice -voicevox-url http://localhost:50021

# Change speaker voice (see VOICEVOX for available speakers)
./claude-companion -project myproject -session mysession -voice -voice-speaker 3
```

Voice output requirements:
- VOICEVOX engine running (default port: 50021)
- Audio playback support:
  - macOS: `afplay` (built-in)
  - Linux: `aplay` or `paplay`
  - Windows: PowerShell (built-in)

The voice narrator will:
- Speak tool action descriptions in Japanese
- Queue multiple narrations to avoid overlapping
- Gracefully handle errors without interrupting the main output

## Event Types

The tool recognizes and formats the following event types:

### 1. User Events (`user`)

User input messages, displayed with timestamp and content.

User input messages are displayed with timestamps and emojis:

```
[15:04:05] 👤 USER:
  💬 Hello, Claude!
```

For complex inputs (tool results), the tool provides clear indicators:
```
[15:04:05] 👤 USER:
  🎯 Command execution
[15:04:06] 👤 USER:
  ✅ Tool Result: toolu_123456
```

### 2. Assistant Events (`assistant`)

Claude's responses, showing model name, content, and token usage.

Claude's responses are shown with model information and formatted content:
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

System messages with optional severity levels are shown with appropriate emojis:

```
[15:04:07] ℹ️ SYSTEM [info]: Tool execution completed successfully
[15:04:08] ⚠️ SYSTEM [warning]: Rate limit approaching
[15:04:09] ❌ SYSTEM [error]: Connection failed
```

In debug mode, additional information like UUID and tool use IDs are displayed.

### 4. Summary Events (`summary`)

Session summaries that provide high-level descriptions:

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