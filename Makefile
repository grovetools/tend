# Makefile for tend

BINARY_NAME=tend
BIN_DIR=bin
VERSION_PKG=github.com/mattsolo1/grove-core/version

# --- Versioning ---
# For dev builds, we construct a version string from git info.
# For release builds, VERSION is passed in by the CI/CD pipeline (e.g., VERSION=v1.2.3)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD)
GIT_DIRTY  ?= $(shell test -n "`git status --porcelain`" && echo "-dirty")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# If VERSION is not set, default to a dev version string
VERSION ?= $(GIT_BRANCH)-$(GIT_COMMIT)$(GIT_DIRTY)

# Go LDFLAGS to inject version info at compile time
LDFLAGS = -ldflags="\
-X '$(VERSION_PKG).Version=$(VERSION)' \
-X '$(VERSION_PKG).Commit=$(GIT_COMMIT)' \
-X '$(VERSION_PKG).Branch=$(GIT_BRANCH)' \
-X '$(VERSION_PKG).BuildDate=$(BUILD_DATE)'"

# --- Mocking ---
# Example target for building mock binaries for tests.
# In a real project, you would list your mock source directories here.
MOCK_SRC_DIR=tests/mocks
MOCK_BIN_DIR=bin
MOCKS ?= $(shell find $(MOCK_SRC_DIR) -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

.PHONY: all build test clean fmt vet lint run check dev build-all help build-mocks generate-docs \
        build-custom-example build-tmux-example build-examples test-tmux test-headless \
        test-tui-verbose test-tui-interactive run-scenarios

all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .

test:
	@echo "Running tests..."
	@go test -v ./...

clean:
	@echo "Cleaning..."
	@go clean
	@rm -rf $(BIN_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out

fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

lint:
	@echo "Running linter..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Run the CLI
run: build
	@$(BIN_DIR)/$(BINARY_NAME) $(ARGS)

# Run all checks
check: fmt vet test

# Development build with race detector
dev:
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY_NAME) version $(VERSION) with race detector..."
	@go build -race $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .

# Run agent-isolation scenario as smoke test
run-scenarios: build
	@echo "Running agent-isolation scenario..."
	@./$(BIN_DIR)/$(BINARY_NAME) run agent-isolation

# Run tmux TUI tests (requires tmux to be installed)
test-tmux: build-tmux-example
	@echo "Running tmux TUI tests..."
	@if command -v tmux > /dev/null; then \
		$(DIST_DIR)/tmux-tui-test run example-tui-tmux; \
		$(DIST_DIR)/tmux-tui-test run example-bubbletea-headless; \
	else \
		echo "Warning: tmux not installed, skipping tmux tests"; \
	fi

# Run headless TUI tests only (doesn't require tmux)
test-headless: build-tmux-example
	@echo "Running headless TUI tests..."
	@$(DIST_DIR)/tmux-tui-test run example-bubbletea-headless

# Run all TUI tests in verbose mode
test-tui-verbose: build-tmux-example
	@echo "Running TUI tests in verbose mode..."
	@$(DIST_DIR)/tmux-tui-test run all --verbose

# Run TUI tests in interactive debug mode
test-tui-interactive: build-tmux-example
	@echo "Running TUI tests in interactive mode (allows attach)..."
	@$(DIST_DIR)/tmux-tui-test run example-tui-interactive-debug --interactive

# Build custom tend example
build-custom-example:
	@echo "Building custom tend example..."
	@mkdir -p $(DIST_DIR)
	@go build $(LDFLAGS) -o $(DIST_DIR)/custom-tend ./examples/custom-tend

# Build tmux TUI test example
build-tmux-example:
	@echo "Building tmux TUI test example..."
	@mkdir -p $(DIST_DIR)
	@go build $(LDFLAGS) -o $(DIST_DIR)/tmux-tui-test ./examples/tmux-tui-test

# Build all examples
build-examples: build-custom-example build-tmux-example
	@echo "All examples built successfully"

# Cross-compilation targets
PLATFORMS ?= darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
DIST_DIR ?= dist

build-all:
	@echo "Building for multiple platforms into $(DIST_DIR)..."
	@mkdir -p $(DIST_DIR)
	@for platform in $(PLATFORMS); do \
		os=$$(echo $$platform | cut -d'/' -f1); \
		arch=$$(echo $$platform | cut -d'/' -f2); \
		output_name="$(BINARY_NAME)-$${os}-$${arch}"; \
		echo "  -> Building $${output_name} version $(VERSION)"; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $(DIST_DIR)/$${output_name} .; \
	done

# Build mock binaries
build-mocks:
	@if [ -d "$(MOCK_SRC_DIR)" ]; then \
		echo "Building mocks: $(MOCKS)"; \
		mkdir -p $(MOCK_BIN_DIR); \
		for mock in $(MOCKS); do \
			echo "  -> Building mock $$mock"; \
			go build -o $(MOCK_BIN_DIR)/mock-$$mock $(MOCK_SRC_DIR)/$$mock; \
		done; \
	else \
		echo "No mock directory found, skipping mock build."; \
	fi

# Generate the tend-docs.json and markdown documentation files
generate-docs:
	@echo "Generating tend documentation..."
	@go run ./cmd/generate-docs/main.go

# Show available targets
help:
	@echo "Available targets:"
	@echo ""
	@echo "Core targets:"
	@echo "  make build       - Build the binary"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make fmt         - Format code"
	@echo "  make vet         - Run go vet"
	@echo "  make lint        - Run linter"
	@echo "  make run ARGS=.. - Run the CLI with arguments"
	@echo "  make check       - Run all checks"
	@echo "  make dev         - Build with race detector"
	@echo ""
	@echo "Example targets:"
	@echo "  make build-custom-example  - Build custom tend example"
	@echo "  make build-tmux-example    - Build tmux TUI test example"
	@echo "  make build-examples        - Build all examples"
	@echo ""
	@echo "TUI testing targets:"
	@echo "  make test-tmux             - Run tmux TUI tests (requires tmux)"
	@echo "  make test-headless         - Run headless TUI tests only"
	@echo "  make test-tui-verbose      - Run all TUI tests in verbose mode"
	@echo "  make test-tui-interactive  - Run TUI tests in interactive debug mode"
	@echo ""
	@echo "Other targets:"
	@echo "  make run-scenarios - Run agent-isolation scenario"
	@echo "  make build-all     - Build for multiple platforms"
	@echo "  make build-mocks   - Build mock binaries for testing"
	@echo "  make generate-docs - Generate tend-docs.json and markdown documentation"
