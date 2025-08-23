.PHONY: build test clean run fmt help

# Go source files (recursive)
GO_FILES := $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")

# Binary output directory
BIN_DIR := bin

# Default target
help:
	@echo "Available targets:"
	@echo "  build   - Build all binaries to bin/ directory"
	@echo "  test    - Run tests"
	@echo "  fmt     - Run gofmt on all Go files"
	@echo "  clean   - Remove all built binaries"
	@echo "  run     - Run with --voice --ai"
	@echo "  server  - Run with --voice --ai --server"
	@echo "  help    - Show this help message"

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

build: $(BIN_DIR)
	@echo "Building claude-companion..."
	@go build -o $(BIN_DIR)/claude-companion .
	@echo "Building claude-code-send..."
	@go build -o $(BIN_DIR)/claude-code-send ./cmd/claude-code-send
	@echo "Building openai-narrator-cli..."
	@go build -o $(BIN_DIR)/openai-narrator-cli ./cmd/openai-narrator-cli
	@echo "Building status-line..."
	@go build -o $(BIN_DIR)/status-line ./cmd/status-line
	@echo "Building test-ws-server..."
	@go build -o $(BIN_DIR)/test-ws-server ./cmd/test-ws-server
	@echo "Building voicevox-cli..."
	@go build -o $(BIN_DIR)/voicevox-cli ./cmd/voicevox-cli
	@echo "All binaries built successfully in $(BIN_DIR)/"

test:
	go test ./...

fmt:
	gofmt -w $(GO_FILES)

clean:
	@echo "Cleaning built binaries..."
	@rm -rf $(BIN_DIR)
	@echo "Clean complete"

run: build
	$(BIN_DIR)/claude-companion --voice --ai

server: build
	$(BIN_DIR)/claude-companion --voice --ai --server
