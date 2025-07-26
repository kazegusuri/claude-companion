# Instructions for Claude

## Project Overview
This is a Go daemon that parses Claude's JSONL log files in real-time.

## Important Guidelines

### Code Formatting
After editing any Go files, you MUST run:
```bash
make fmt
```

This ensures all Go code follows standard formatting conventions.

### Building
Use the Makefile for building:
```bash
make build
```

### Testing
Before committing any changes, run:
```bash
make test
```

### Running the Application
```bash
make run PROJECT=project_name SESSION=session_name
```

## Code Style
- Follow standard Go conventions
- Use meaningful variable names
- Keep functions focused and small
- Add appropriate error handling

## Git Workflow
1. Make changes
2. Run `make fmt` to format code
3. Run `make test` to ensure tests pass
4. Commit with meaningful commit messages