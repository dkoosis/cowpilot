SHELL := /bin/bash
# mcp adapters Makefile

# ============================================================================
# TESTING PHILOSOPHY - DO NOT CHANGE WITHOUT UNDERSTANDING THE CONSEQUENCES
# ============================================================================
# The default 'make' command runs EVERY test including production validation.
# This is BY DESIGN. We prefer to wait 2 minutes than discover broken 
# registrations days later and spend hours debugging.
#
# NEVER change the default behavior to skip tests.
# NEVER make production tests "optional" by default.
# 
# If you need to work offline or skip production tests:
#   - Use: make quick
#   - Or: SKIP_PRODUCTION_TESTS=true make
#
# But the DEFAULT must ALWAYS test EVERYTHING.
# ============================================================================

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

.PHONY: all build build-rtm build-debug debug-proxy run-debug-proxy test unit-test integration-test clean fmt vet lint coverage run help test-verbose docs-tree quick

# DEFAULT: Clean, format, lint, test EVERYTHING (including production), then build
# DO NOT CHANGE THIS - see TESTING PHILOSOPHY above
all: clean fmt vet lint test-everything build-all

# Quick mode - skip production tests when you need to work offline
# Use: make quick
quick:
	@echo "Quick build mode (skipping production tests)..."
	@SKIP_PRODUCTION_TESTS=true $(MAKE) all

# Run all tests (unit + integration + production validation)
test: unit-test integration-test production-validation

# Run EVERYTHING - all tests including production checks
test-everything: unit-test integration-test claude-test rtm-health-test production-validation

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

# Build all servers (assumes tests already passed)
build-all: docs-tree
	@echo "Building all servers..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(SPEKTRIX_BINARY_NAME) $(SPEKTRIX_BUILD_DIR)

# Build the everything server (assumes tests already passed)
build: docs-tree
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

# Test RTM production health - verifies deployed server is ready
rtm-health-test:
	@echo "Testing RTM production health..."
	@GO_TEST=1 $(GOTEST) -v -timeout 30s ./tests/integration -run TestRTMProductionHealth

# Test all production services
production-test:
	@echo "Running production validation suite..."
	@GO_TEST=1 $(GOTEST) -v -timeout 60s ./tests/integration -run TestProductionSuite

# Production validation - MUST PASS for build to succeed
production-validation:
	@if [ "$$SKIP_PRODUCTION_TESTS" = "true" ]; then \
		echo "Skipping production validation (SKIP_PRODUCTION_TESTS=true)"; \
	else \
		echo "Validating production services..."; \
		echo "Note: This WILL FAIL if production servers are not deployed."; \
		echo "To skip: export SKIP_PRODUCTION_TESTS=true or use 'make quick'"; \
		$(MAKE) rtm-health-test; \
		$(MAKE) production-test; \
	fi

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
deploy-rtm: test-everything
	@echo "Building and deploying RTM server to rtm.fly.dev..."
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)
	fly deploy -a rtm -c fly-rtm.toml
	@echo "Waiting for deployment..."
	@sleep 10
	@echo "Verifying RTM OAuth for Claude.ai..."
	@$(GOTEST) -v ./tests/integration -run TestRTMProductionHealth || (echo "RTM OAuth broken - Claude.ai won't connect!" && exit 1)

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

# RTM Diagnostics & Testing
rtm-test: rtm-test-unit rtm-test-integration

rtm-test-unit:
	@echo "Running RTM unit tests..."
	@$(GOTEST) -v -count=1 ./internal/rtm

rtm-test-integration:
	@echo "Running RTM integration tests..."
	@$(GOTEST) -v -count=1 ./internal/rtm -run Integration

rtm-test-oauth:
	@echo "Testing RTM OAuth flow..."
	@$(GOTEST) -v -count=1 ./internal/rtm -run TestOAuthFlow

rtm-test-e2e:
	@echo "Running RTM E2E test..."
	@go run ./cmd/rtm/e2e_test.go

# RTM Production Commands
rtm-status:
	@echo "Quick RTM production check..."
	@chmod +x scripts/check-rtm-production.sh 2>/dev/null || true
	@./scripts/check-rtm-production.sh 2>/dev/null || \
		(echo "Checking RTM app status..." && \
		 curl -s -f -m 5 https://rtm.fly.dev/health > /dev/null && echo "✅ RTM is online" || echo "❌ RTM is offline")

rtm-logs:
	@echo "Fetching RTM production logs..."
	@fly logs -a rtm | tail -50

rtm-secrets:
	@echo "Checking RTM Fly.io secrets..."
	@fly secrets list -a rtm | grep -E "RTM_|SERVER_URL" || echo "No RTM secrets found"

# OAuth diagnostics commands
diagnose: diagnose-prod

diagnose-local:
	@echo "Running LOCAL RTM diagnostics..."
	@chmod +x scripts/diagnose-rtm.sh 2>/dev/null || true
	@./scripts/diagnose-rtm.sh 2>/dev/null || \
		(echo "Checking local environment..." && \
		 [ -n "$$RTM_API_KEY" ] && echo "✓ RTM_API_KEY set" || echo "✗ RTM_API_KEY missing" && \
		 [ -n "$$RTM_API_SECRET" ] && echo "✓ RTM_API_SECRET set" || echo "✗ RTM_API_SECRET missing")

diagnose-prod:
	@echo "Running PRODUCTION RTM diagnostics..."
	@chmod +x scripts/diagnose-rtm.sh 2>/dev/null || true
	@./scripts/diagnose-rtm.sh --production 2>/dev/null || make rtm-status

monitor-oauth:
	@echo "Monitoring RTM OAuth flow..."
	@fly logs -a rtm | grep -E "OAuth|auth|token|callback|frob"

# Show help
help:
	@echo "MCP Adapters - Make Commands"
	@echo "============================="
	@echo ""
	@echo "🚀 Quick Commands:"
	@echo "  make               - Run EVERYTHING (all tests including production)"
	@echo "  make quick         - Build without production tests (offline mode)"
	@echo "  make rtm-status    - Quick check if RTM production is working"
	@echo "  make diagnose      - Full production diagnostics"
	@echo "  make deploy-rtm    - Deploy RTM to Fly.io"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test          - Run all tests"
	@echo "  make rtm-test      - Run RTM-specific tests"
	@echo "  make rtm-test-oauth - Test OAuth flow specifically"
	@echo "  make claude-test   - Test Claude.ai OAuth compliance"
	@echo "  make rtm-health-test - Test RTM production health"
	@echo "  make production-test - Run full production validation"
	@echo ""
	@echo "🔍 Diagnostics:"
	@echo "  make diagnose-prod  - Check production (Fly.io)"
	@echo "  make diagnose-local - Check local environment"
	@echo "  make monitor-oauth  - Watch OAuth logs in real-time"
	@echo "  make rtm-logs      - View RTM production logs"
	@echo "  make rtm-secrets   - Check Fly.io secrets"
	@echo ""
	@echo "🔨 Build & Deploy:"
	@echo "  make build         - Build all servers"
	@echo "  make deploy-rtm    - Deploy RTM to production"
	@echo "  make run           - Run locally"
	@echo "  make dev           - Run with hot reload"
	@echo ""
	@echo "🧙 Maintenance:"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make fmt           - Format code"
	@echo "  make lint          - Run linter"
	@echo "  make coverage      - Generate coverage report"
	@echo ""
	@echo "For production issues, start with: make rtm-status"
