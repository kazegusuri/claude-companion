.PHONY: build test clean run fmt help

# Go source files
GO_FILES := $(wildcard *.go)

# Default target
help:
	@echo "Available targets:"
	@echo "  build   - Build the claude-companion binary"
	@echo "  test    - Run tests"
	@echo "  fmt     - Run gofmt on all Go files"
	@echo "  clean   - Remove built binary"
	@echo "  run     - Run with example arguments (requires PROJECT and SESSION env vars)"
	@echo "  help    - Show this help message"

claude-companion: $(GO_FILES)
	go build -o claude-companion .

build: claude-companion

test:
	go test ./...

fmt:
	gofmt -w *.go

clean:
	rm -f claude-companion

run: build
	@if [ -z "$(PROJECT)" ] || [ -z "$(SESSION)" ]; then \
		echo "Usage: make run PROJECT=project_name SESSION=session_name [FULL=1]"; \
		exit 1; \
	fi
	@if [ -n "$(FULL)" ]; then \
		./claude-companion -project $(PROJECT) -session $(SESSION) -full; \
	else \
		./claude-companion -project $(PROJECT) -session $(SESSION); \
	fi