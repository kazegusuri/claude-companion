.PHONY: build test clean run fmt help all

# Go source files (recursive)
GO_FILES := $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*")

# Binary output directory
BIN_DIR := bin

# Main binary
MAIN_BINARY := claude-companion

# Command binaries from cmd/ directory
CMD_BINARIES := claude-code-send \
	openai-narrator-cli \
	status-line \
	cc-status-line-wrapper \
	test-ws-server \
	voicevox-cli

# All binaries
ALL_BINARIES := $(MAIN_BINARY) $(CMD_BINARIES)

# Default target
all: build

help:
	@echo "Available targets:"
	@echo "  build   - Build all binaries to bin/ directory"
	@echo "  test    - Run tests"
	@echo "  fmt     - Run gofmt on all Go files"
	@echo "  clean   - Remove all built binaries"
	@echo "  run     - Run with --voice --ai"
	@echo "  server  - Run with --voice --ai --server"
	@echo "  help    - Show this help message"
	@echo ""
	@echo "Build specific binary:"
	@echo "  build-<binary-name> - Build a specific binary (e.g., make build-status-line)"
	@echo ""
	@echo "Available binaries:"
	@echo "  $(ALL_BINARIES)"

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

# Build all binaries
build: $(BIN_DIR)
	@echo "Building all binaries..."
	@echo "  Building $(MAIN_BINARY)..."
	@go build -o $(BIN_DIR)/$(MAIN_BINARY) .
	@for binary in $(CMD_BINARIES); do \
		echo "  Building $$binary..."; \
		go build -o $(BIN_DIR)/$$binary ./cmd/$$binary || exit 1; \
	done
	@echo "✅ All binaries built successfully in $(BIN_DIR)/"
	@echo "Built binaries: $(ALL_BINARIES)"

# Pattern rule for building individual binaries
build-%: $(BIN_DIR)
	@if [ "$*" = "$(MAIN_BINARY)" ]; then \
		echo "Building $*..."; \
		go build -o $(BIN_DIR)/$* .; \
	else \
		echo "Building $*..."; \
		go build -o $(BIN_DIR)/$* ./cmd/$*; \
	fi
	@echo "✅ $* built successfully in $(BIN_DIR)/"

test:
	go test ./...

fmt:
	gofmt -w $(GO_FILES)

clean:
	@echo "Cleaning built binaries..."
	@rm -rf $(BIN_DIR)
	@echo "✅ Clean complete"

run: build
	$(BIN_DIR)/$(MAIN_BINARY) --voice --ai

server: build
	$(BIN_DIR)/$(MAIN_BINARY) --voice --ai --server

# Install binaries to system
install: build
	@echo "Installing binaries to /usr/local/bin..."
	@for binary in $(ALL_BINARIES); do \
		echo "  Installing $$binary..."; \
		sudo cp $(BIN_DIR)/$$binary /usr/local/bin/ || exit 1; \
	done
	@echo "✅ Installation complete"

# Uninstall binaries from system
uninstall:
	@echo "Removing binaries from /usr/local/bin..."
	@for binary in $(ALL_BINARIES); do \
		echo "  Removing $$binary..."; \
		sudo rm -f /usr/local/bin/$$binary; \
	done
	@echo "✅ Uninstall complete"
