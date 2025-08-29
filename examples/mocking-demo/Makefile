# Makefile for mocking-demo example

BINARY_NAME=mocking-demo
BIN_DIR=bin
MOCK_SRC_DIR=tests/mocks
MOCK_BIN_DIR=bin

# Find all mock directories
MOCKS=$(shell find $(MOCK_SRC_DIR) -mindepth 1 -maxdepth 1 -type d -exec basename {} \;)

.PHONY: all build build-mocks clean run help

all: build-mocks build

# Build the test binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BIN_DIR)/$(BINARY_NAME) .

# Build all mock binaries
build-mocks:
	@echo "Building mocks: $(MOCKS)"
	@mkdir -p $(MOCK_BIN_DIR)
	@for mock in $(MOCKS); do \
		echo "  -> Building mock-$$mock"; \
		go build -o $(MOCK_BIN_DIR)/mock-$$mock ./$(MOCK_SRC_DIR)/$$mock; \
	done

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@go clean

# Run scenarios
run: build
	@$(BIN_DIR)/$(BINARY_NAME) run $(ARGS)

# Run specific scenarios with examples
run-git: build
	@echo "Running git workflow scenario with mocks..."
	@$(BIN_DIR)/$(BINARY_NAME) run git-workflow -v

run-docker: build
	@echo "Running docker operations scenario with mocks..."
	@$(BIN_DIR)/$(BINARY_NAME) run docker-operations -v

run-llm: build
	@echo "Running LLM integration scenario..."
	@$(BIN_DIR)/$(BINARY_NAME) run llm-integration -v

run-mixed: build
	@echo "Running mixed dependencies scenario..."
	@$(BIN_DIR)/$(BINARY_NAME) run mixed-dependencies -v

# Run with real dependencies examples
run-real-git: build
	@echo "Running git workflow with REAL git binary..."
	@$(BIN_DIR)/$(BINARY_NAME) run git-workflow --use-real-deps=git -v

run-real-all: build
	@echo "Running all scenarios with REAL binaries..."
	@$(BIN_DIR)/$(BINARY_NAME) run --use-real-deps=all -v

# List available scenarios
list: build
	@$(BIN_DIR)/$(BINARY_NAME) list

# Run all scenarios
test: build-mocks build
	@echo "Running all scenarios..."
	@$(BIN_DIR)/$(BINARY_NAME) run

# Alias for test (matches grove convention)
test-e2e: test

# Interactive mode
interactive: build
	@echo "Running in interactive mode..."
	@$(BIN_DIR)/$(BINARY_NAME) run -i $(ARGS)

# Debug mode
debug: build
	@echo "Running in debug mode..."
	@$(BIN_DIR)/$(BINARY_NAME) run -d $(ARGS)

# Help
help:
	@echo "Mocking Demo - Example of grove-tend's mocking capabilities"
	@echo ""
	@echo "Available targets:"
	@echo "  make build         - Build the test binary and mocks"
	@echo "  make build-mocks   - Build only the mock binaries"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make run           - Run scenarios (use ARGS for options)"
	@echo "  make list          - List available scenarios"
	@echo "  make test          - Run all scenarios"
	@echo "  make test-e2e      - Run all scenarios (alias for test)"
	@echo ""
	@echo "Scenario shortcuts:"
	@echo "  make run-git       - Run git workflow scenario"
	@echo "  make run-docker    - Run docker operations scenario"
	@echo "  make run-llm       - Run LLM integration scenario"
	@echo "  make run-mixed     - Run mixed dependencies scenario"
	@echo ""
	@echo "Real dependency examples:"
	@echo "  make run-real-git  - Run with real git binary"
	@echo "  make run-real-all  - Run with all real binaries"
	@echo ""
	@echo "Interactive modes:"
	@echo "  make interactive   - Run in interactive mode"
	@echo "  make debug         - Run in debug mode"
	@echo ""
	@echo "Examples:"
	@echo "  make run ARGS='git-workflow --use-real-deps=git'"
	@echo "  make run ARGS='--tags=docker -v'"
	@echo "  make interactive ARGS='llm-integration'"