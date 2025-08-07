SHELL := /bin/bash
# mcp adapters Makefile

# Variables
BINARY_NAME=mcp-adapters
RTM_BINARY_NAME=rtm-server
SPEKTRIX_BINARY_NAME=spektrix-server
DEBUG_PROXY_NAME=mcp-debug-proxy
GO=go
GOTEST=$(GO) test
GOVET=$(GO) vet
GOFMT=gofmt
GOLINT=golangci-lint

# Build variables
BUILD_DIR=./cmd/core
RTM_BUILD_DIR=./cmd/rtm
SPEKTRIX_BUILD_DIR=./cmd/spektrix
DEBUG_PROXY_DIR=./cmd/mcp_debug_proxy
MAIN_FILE=$(BUILD_DIR)/main.go
RTM_MAIN_FILE=$(RTM_BUILD_DIR)/main.go
SPEKTRIX_MAIN_FILE=$(SPEKTRIX_BUILD_DIR)/main.go
DEBUG_PROXY_FILE=$(DEBUG_PROXY_DIR)/main.go
OUTPUT_DIR=./bin
DOCS_DIR=./docs

# Test variables
UNIT_TEST_DIRS=./cmd/... ./internal/...
COVERAGE_FILE=coverage.out
GOTESTSUM=$(shell which gotestsum 2>/dev/null || echo "")

.PHONY: all build build-rtm build-debug debug-proxy run-debug-proxy test unit-test integration-test clean fmt vet lint coverage run help test-verbose docs-tree

# Clean, format, lint, build all servers, and run ALL tests
all: clean fmt vet lint build-all test

# Run all tests (unit + integration with TestMain)
test: unit-test integration-test

# Generate project documentation tree
docs-tree:
	@echo "Generating project tree documentation..."
	@mkdir -p $(DOCS_DIR)
	@if ! which tree > /dev/null; then echo "⚠️  tree command not found. Install with: brew install tree (macOS) or apt-get install tree (Ubuntu)"; exit 1; fi
	@echo "# Project Structure" > $(DOCS_DIR)/project_structure.md
	@echo "" >> $(DOCS_DIR)/project_structure.md
	@echo "Generated on: $(date '+%Y-%m-%d %H:%M:%S')" >> $(DOCS_DIR)/project_structure.md
	@echo "" >> $(DOCS_DIR)/project_structure.md
	@echo '```' >> $(DOCS_DIR)/project_structure.md
	@tree --gitignore -F --dirsfirst -a -I 'bin|*.log|*.db|coverage.*|test-results.*|.DS_Store' >> $(DOCS_DIR)/project_structure.md
	@echo '```' >> $(DOCS_DIR)/project_structure.md
	@echo "" >> $(DOCS_DIR)/project_structure.md
	@echo "## Key Directories" >> $(DOCS_DIR)/project_structure.md
	@echo "" >> $(DOCS_DIR)/project_structure.md
	@echo '```' >> $(DOCS_DIR)/project_structure.md
	@tree --gitignore -d -L 3 -I 'bin|.git' >> $(DOCS_DIR)/project_structure.md
	@echo '```' >> $(DOCS_DIR)/project_structure.md
	@echo "📁 Project structure updated in $(DOCS_DIR)/project_structure.md"

# Build all servers
build-all: test docs-tree
	@echo "Building all servers..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(SPEKTRIX_BINARY_NAME) $(SPEKTRIX_BUILD_DIR)

# Build the everything server (testing target)
build: test docs-tree
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)

# Build the debug proxy
build-debug:
	@echo "Building $(DEBUG_PROXY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(DEBUG_PROXY_NAME) $(DEBUG_PROXY_DIR)

# Build both main application and debug proxy
debug-proxy: build build-debug
	@echo "Both binaries built successfully"

# Run the debug proxy with default settings
run-debug-proxy: build-debug
	@echo "Starting MCP Debug Proxy..."
	MCP_DEBUG_ENABLED=true MCP_DEBUG_LEVEL=INFO $(OUTPUT_DIR)/$(DEBUG_PROXY_NAME) \
		--target $(OUTPUT_DIR)/$(BINARY_NAME) \
		--port 8080 \
		--target-port 8081

# Run unit tests (fast, no build tags)
unit-test:
	@echo "Running unit tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		GO_TEST=1 $(GOTESTSUM) --format testdox -- -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	else \
		GO_TEST=1 $(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	fi

# Run integration tests using TestMain (no shell scripts)
integration-test:
	@echo "Running integration tests with Go TestMain..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		GO_TEST=1 $(GOTESTSUM) --format testdox -- -race -tags=integration -timeout 60s ./tests/integration/...; \
	else \
		GO_TEST=1 $(GOTEST) -v -race -tags=integration -timeout 60s ./tests/integration/...; \
	fi

# Test Claude OAuth compliance - MUST PASS before deployment
claude-test:
	@echo "Testing Claude.ai OAuth compliance..."
	GO_TEST=1 $(GOTEST) -v -race -timeout 30s ./tests/integration -run TestClaudeOAuthCompliance
	GO_TEST=1 $(GOTEST) -v -race -timeout 30s ./tests/integration -run TestRTMOAuthFlow

# Enhanced test output
GOTESTSUM := $(shell which gotestsum 2>/dev/null)
ifdef GOTESTSUM
    GOTEST = gotestsum --format testname --format-hide-empty-pkg --
else
    GOTEST = go test
endif

# Run tests with verbose human-readable output
test-verbose:
	@echo "Running unit tests with verbose output..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		GO_TEST=1 $(GOTESTSUM) --format testdox -- -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	else \
		GO_TEST=1 $(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	fi
	@echo "Running integration tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		GO_TEST=1 $(GOTESTSUM) --format testdox -- -race ./tests/integration/...; \
	else \
		GO_TEST=1 $(GOTEST) -v -race ./tests/integration/...; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f $(COVERAGE_FILE)
	@rm -f debug_conversations.db
	@rm -f *.db
	@rm -f server.log
	@rm -f $(DOCS_DIR)/project_structure.md

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w .

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@if ! which $(GOLINT) > /dev/null; then echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; fi
	@$(GOLINT) run

# Generate test coverage report
coverage: unit-test
	@echo "Generating coverage report..."
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run the application locally
run: build
	@echo "Running $(BINARY_NAME)..."
	$(OUTPUT_DIR)/$(BINARY_NAME)

# Run with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "air not found. Install with: go install github.com/air-verse/air@latest" && exit 1)
	air

# Deploy test/core server (ephemeral)
deploy-core-tmp: build
	@echo "Deploying test server to core-tmp app..."
	fly apps create core-tmp || true
	fly deploy -a core-tmp -c fly-core-tmp.toml

# Deploy RTM server to production
deploy-rtm: test claude-test
	@echo "Building and deploying RTM server to rtm.fly.dev..."
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)
	fly deploy -a rtm -c fly-rtm.toml

# Cleanup ephemeral test server
cleanup-core-tmp:
	@echo "Destroying ephemeral core-tmp app..."
	fly apps destroy core-tmp --yes 2>/dev/null || true
	@echo "✅ Cleanup complete"

# CI-specific test with junit output
test-ci:
	@echo "Running tests for CI..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		GO_TEST=1 $(GOTESTSUM) --junitfile test-results.xml --format testname -- -race -coverprofile=$(COVERAGE_FILE) ./...; \
	else \
		GO_TEST=1 $(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) ./...; \
	fi

# OAuth diagnostics commands
diagnose: diagnose-setup
	@echo "Running OAuth diagnostics..."
	@bash scripts/diagnostics/run_diagnostics.sh

diagnose-local: diagnose-setup
	@echo "Running OAuth diagnostics (local)..."
	@bash scripts/diagnostics/run_diagnostics.sh local

diagnose-setup:
	@echo "Setting up diagnostic tools..."
	@chmod +x scripts/diagnostics/*.sh
	@cd scripts/diagnostics && go build -o oauth_trace oauth_trace.go

monitor-oauth:
	@echo "Starting OAuth real-time monitor..."
	@go run scripts/diagnostics/monitor_realtime.go

diagnose-prod:
	@echo "Checking production OAuth endpoints..."
	@go run scripts/diagnostics/oauth_trace.go
	@echo ""
	@echo "Fetching production logs..."
	@flyctl logs --app rtm | tail -100 | grep -E "\[OAuth|ERROR|401|token|callback" || echo "No OAuth-related logs found"

# Show help
include makefile_help.mk