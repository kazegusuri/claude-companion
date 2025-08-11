# Development

## Prerequisites
- Go 1.19 or higher
- Make

## Building
```bash
make build    # Build the binary
make test     # Run tests
make fmt      # Format code
make clean    # Clean build artifacts
```

## Project Structure
```
.
├── main.go                      # Main application and CLI
├── event/
│   ├── event.go                # Event type definitions
│   ├── parser.go               # Event parsing logic
│   ├── formatter.go            # Event formatting
│   ├── handler.go              # Event handling and routing
│   ├── session_watcher.go      # Individual session file watcher
│   ├── projects_watcher.go     # Projects directory watcher
│   ├── notification_watcher.go # Notification log watcher
│   └── session_file_manager.go # Session watcher lifecycle management
├── narrator/
│   ├── narrator.go             # Narrator interface and base implementation
│   ├── config_narrator.go      # Configuration-based narrator
│   ├── voice_narrator.go       # Voice output implementation
│   ├── priority_queue.go       # Priority queue for voice messages
│   └── voicevox.go            # VOICEVOX client
├── Makefile                    # Build automation
├── CLAUDE.md                   # Instructions for Claude
└── README.md                   # This file
```

## Contributing

1. Follow the coding standards in CLAUDE.md
2. Run `make fmt` before committing
3. Ensure all tests pass with `make test`
4. Write clear commit messages