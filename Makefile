# Makefile for tend

BINARY_NAME=tend
INSTALL_PATH=/usr/local/bin
BIN_DIR=bin

.PHONY: all build install uninstall test clean fmt vet lint run

all: build

build:
	@mkdir -p $(BIN_DIR)
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BIN_DIR)/$(BINARY_NAME) .

install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo cp $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installed successfully!"

uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Uninstalled successfully!"

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
	@echo "Building $(BINARY_NAME) with race detector..."
	@go build -race -o $(BIN_DIR)/$(BINARY_NAME) .

# Run agent-isolation scenario as smoke test
run-scenarios: build
	@echo "Running agent-isolation scenario..."
	@./$(BIN_DIR)/$(BINARY_NAME) run agent-isolation

# Show available targets
help:
	@echo "Available targets:"
	@echo "  make build       - Build the binary"
	@echo "  make install     - Build and install to $(INSTALL_PATH)"
	@echo "  make uninstall   - Remove from $(INSTALL_PATH)"
	@echo "  make test        - Run tests"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make fmt         - Format code"
	@echo "  make vet         - Run go vet"
	@echo "  make lint        - Run linter"
	@echo "  make run ARGS=.. - Run the CLI with arguments"
	@echo "  make check       - Run all checks"
	@echo "  make dev         - Build with race detector"
	@echo "  make run-scenarios - Run agent-isolation scenario"
	@echo "  make help        - Show this help"