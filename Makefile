# Makefile for tend

BINARY_NAME=tend
BIN_DIR=bin
VERSION_PKG=github.com/grovetools/core/version

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

# --- Cross-compile contract (set by `grove build --target`) ---
# GROVE_BUILD_OUT redirects output so cross binaries never clobber native bin/.
# GROVE_TARGET_* are applied only to the final `go build`; codegen prereqs stay native.
ifneq ($(strip $(GROVE_BUILD_OUT)),)
BIN_DIR = $(GROVE_BUILD_OUT)
endif
ifneq ($(strip $(GROVE_TARGET_GOOS)),)
GO_CROSS_ENV = GOOS=$(GROVE_TARGET_GOOS) GOARCH=$(GROVE_TARGET_GOARCH) CGO_ENABLED=0
endif

.PHONY: all build test clean fmt fmt-check vet lint run check dev build-all help generate-docs \
        test-e2e build-e2e-mocks build-e2e-runner build-e2e-fixtures

all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@$(GO_CROSS_ENV) go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .

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
	@gofumpt -w .

fmt-check:
	@unformatted="$$(gofumpt -l . 2>/dev/null)"; \
	if [ -n "$$unformatted" ]; then \
		echo "Unformatted files (run 'make fmt'):"; \
		echo "$$unformatted" | sed 's/^/  /'; \
		exit 1; \
	fi

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
check: fmt-check vet lint test

# Development build with race detector
dev:
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY_NAME) version $(VERSION) with race detector..."
	@go build -race $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .

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

# --- E2E Testing for Tend ---
# For self-testing grove-tend, we need to:
# 1. Build the test runner from tests/e2e/ (includes scenarios)
# 2. Build mocks (since proxy is skipped when testing ourselves)
E2E_RUNNER=tend-e2e
E2E_MOCK_SRC=tests/e2e/tend/mocks/src
E2E_MOCK_BIN=tests/e2e/tend/mocks/bin

build-e2e-mocks:
	@if [ -d "$(E2E_MOCK_SRC)" ]; then \
		mkdir -p $(E2E_MOCK_BIN); \
		for mock in $$(ls $(E2E_MOCK_SRC)); do \
			echo "  Building mock-$$mock..."; \
			go build -o $(E2E_MOCK_BIN)/mock-$$mock ./$(E2E_MOCK_SRC)/$$mock; \
		done; \
	fi

build-e2e-runner:
	@echo "Building E2E test runner..."
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(E2E_RUNNER) ./tests/e2e

build-e2e-fixtures:
	@echo "Building E2E test fixtures..."
	@mkdir -p tests/e2e/fixtures/bin
	@cd tests/e2e/fixtures/list-tui && GOWORK=off go build -o ../bin/list-tui .
	@cd tests/e2e/fixtures/task-manager && GOWORK=off go build -o ../bin/task-manager .
	@cd tests/e2e/fixtures/file-saver && GOWORK=off go build -o ../bin/file-saver .

test-e2e: build-e2e-runner build-e2e-mocks build-e2e-fixtures
	@echo "Running tend E2E test suite..."
	@$(BIN_DIR)/$(E2E_RUNNER) run $(ARGS)

# Generate documentation using the new docgen tool
generate-docs:
	@echo "Generating tend documentation using docgen..."
	@docgen generate

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
	@echo "E2E testing targets:"
	@echo "  make test-e2e          - Run E2E test suite (mocks built automatically)"
	@echo ""
	@echo "Other targets:"
	@echo "  make build-all     - Build for multiple platforms"
	@echo "  make generate-docs - Generate tend-docs.json and markdown documentation"
