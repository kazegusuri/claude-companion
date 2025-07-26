.PHONY: build test clean run help

# Default target
help:
	@echo "Available targets:"
	@echo "  build   - Build the claude-companion binary"
	@echo "  test    - Run tests"
	@echo "  clean   - Remove built binary"
	@echo "  run     - Run with example arguments (requires PROJECT and SESSION env vars)"
	@echo "  help    - Show this help message"

build:
	go build -o claude-companion main.go

test:
	go test ./...

clean:
	rm -f claude-companion

run: build
	@if [ -z "$(PROJECT)" ] || [ -z "$(SESSION)" ]; then \
		echo "Usage: make run PROJECT=project_name SESSION=session_name"; \
		exit 1; \
	fi
	./claude-companion -project $(PROJECT) -session $(SESSION)