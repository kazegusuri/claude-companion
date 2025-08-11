# Claude Companion

A real-time parser and viewer for Claude's JSONL log files. This tool helps you monitor and analyze Claude sessions by parsing structured log events and displaying them in a human-readable format with voice narration support.

**Note**: This is a hobby project. The interface and functionality may change without notice.

## Features

- **Real-time monitoring**: Tail Claude's JSONL log files and display new events as they appear
- **Project-wide watching**: Monitor all sessions across projects with smart filtering
- **Notification integration**: Capture and display Claude hook notifications in real-time
- **Voice narration**: Speak tool actions using VOICEVOX text-to-speech engine
- **AI-powered narrator**: Natural language descriptions of tool actions using OpenAI
- **Human-readable output**: Display events with timestamps and formatted content

## Installation

```bash
# Clone the repository
git clone https://github.com/kazegusuri/claude-companion.git
cd claude-companion

# Build the binary
make build
```

### Setting up Claude Hooks

Claude Companion can capture notification events from Claude through hooks:

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

## Usage

### Quick Start

```bash
# Watch all projects with voice and AI narration (recommended)
./claude-companion --voice --ai

# Watch all projects without voice narration
./claude-companion

# Watch specific project
./claude-companion -p myproject

# Read a specific file directly
./claude-companion -f /path/to/session.jsonl
```

### Command Line Options

#### Core Options
- `-p, --project`: Filter to specific project name
- `-s, --session`: Filter to specific session name
- `-f, --file`: Direct path to a session file
- `--head`: Read entire file from beginning to end instead of tailing
- `-d, --debug`: Enable debug mode with detailed information

#### Narrator Options
- `--ai`: Use AI narrator (requires OpenAI API key)
- `--openai-key`: OpenAI API key (can also use OPENAI_API_KEY env var)
- `--narrator-config`: Path to custom narrator configuration file

#### Voice Options
- `--voice`: Enable voice output using VOICEVOX
- `--voicevox-url`: VOICEVOX server URL (default: http://localhost:50021)
- `--voice-speaker`: VOICEVOX speaker ID (default: 1)

#### Other Options
- `--notification-log`: Path to notification log file (default: /var/log/claude-notification.log)
- `--projects-root`: Root directory for projects (default: ~/.claude/projects)

## Operating Modes

### Watch Mode (Default)

By default, Claude Companion watches all projects under `~/.claude/projects`. You can filter what to watch:

```bash
# Watch all projects and sessions
./claude-companion

# Watch only "myproject"
./claude-companion -p myproject

# Watch only sessions named "coding" across all projects
./claude-companion -s coding

# Watch only "coding" sessions in "myproject"
./claude-companion -p myproject -s coding
```

The watcher automatically:
- Detects new projects and sessions
- Handles file creation and deletion
- Manages multiple session watchers efficiently
- Cleans up idle watchers automatically

### Direct File Mode

For monitoring a specific file:

```bash
# Tail a specific file
./claude-companion -f /path/to/session.jsonl

# Read entire file
./claude-companion -f /path/to/session.jsonl --head
```

### Notification Monitoring

The tool automatically monitors `/var/log/claude-notification.log` if it exists:
- Waits for file creation if it doesn't exist
- Handles permission errors gracefully
- Resumes watching when permissions are granted

**Note**: Notification monitoring requires Claude hooks to be configured. See the "Setting up Claude Hooks" section above for instructions on configuring the notification script and Claude's `settings.json`.

## Voice Narration

### Prerequisites

1. **Install VOICEVOX** (choose one):
   - **Docker (quickest method)**:
     ```bash
     docker run -d --rm -it -p '127.0.0.1:50021:50021' voicevox/voicevox_engine:cpu-latest
     ```
   - **Manual installation**:
     - Download from: https://github.com/VOICEVOX/voicevox_engine
     - Run the engine: `./run` (or `run.exe` on Windows)

2. **Audio playback support**:
   - macOS: `afplay` (built-in)
   - Linux: `aplay` or `paplay`
   - Windows: PowerShell (built-in)

3. **AI Narrator (Optional)**:
   - When using the `--ai` option, you must set the `OPENAI_API_KEY` environment variable
   - Without the `--ai` option, English text narration may not work properly
   - With the `--ai` option, text will be translated to Japanese for narration

### Usage

```bash
# Enable voice narration
./claude-companion --voice

# Use specific speaker
./claude-companion --voice --voice-speaker 3

# With AI narrator for more natural descriptions
./claude-companion --voice --ai
```

The voice system features:
- Priority-based queue for important messages
- Non-blocking audio playback
- Graceful error handling
- Support for multiple speakers

## Event Types

### 1. User Events
```
[15:04:05] 👤 USER:
  💬 Hello, Claude!
```

### 2. Assistant Events
```
[15:04:06] 🤖 ASSISTANT (claude-3-sonnet):
  I'll help you with that task.
  
  💬 ファイル「main.go」を読み込みます
  📄 Reading: main.go
  
  💬 テストを実行します
  🏃 Running: make test
```

### 3. System Events
```
[15:04:07] ℹ️ SYSTEM [info]: Tool execution completed
[15:04:08] ⚠️ SYSTEM [warning]: Rate limit approaching
```

### 4. Notification Events
```
[15:04:09] 🔔 NOTIFICATION [SessionStart]:
  Project: myproject
  Session: coding-session
```

## Narrator Configuration

Create a custom narrator configuration file:

```json
{
  "rules": [
    {
      "pattern": "^git commit",
      "template": "{base} Gitにコミットします: {detail}"
    },
    {
      "tool": "Read",
      "template": "ファイル「{path}」を読み込みます"
    }
  ]
}
```

Use it with:
```bash
./claude-companion --narrator-config=/path/to/config.json
```

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development instructions.

## License

MIT License