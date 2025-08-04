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
SCENARIO_TEST_DIR=./tests/scenarios
COVERAGE_FILE=coverage.out
GOTESTSUM=$(shell which gotestsum 2>/dev/null || echo "")

# TESTING STRATEGY:
# - 'make' and 'make test' = COMPREHENSIVE (local + deployed verification)
# - Auto-deploy if server not responding (no manual deployment dance)
# - Clear messaging about what's happening and why
# 
# Philosophy: Better to wait 2 minutes 100 times than debug something dumb
# Speed is not the priority - comprehensive validation is

.PHONY: all build build-rtm build-debug debug-proxy run-debug-proxy test unit-test integration-test integration-test-local scenario-test scenario-test-local scenario-test-prod scenario-test-raw test-ci clean fmt vet lint coverage run help test-verbose docs-tree

# Clean, format, lint, build all servers, and run COMPREHENSIVE tests
all: clean fmt vet lint build-all test-all deploy-verification

# Run fast tests (unit only)
test-fast: unit-test

# Run standard tests (unit + integration with TestMain)
test: unit-test integration-test

# Run ALL tests (unit + integration + scenario/protocol)
test-all: unit-test integration-test scenario-test

# Verify deployed servers are working
deploy-verification:
	@echo "Verifying deployed servers..."
	@echo "Testing core server..."
	@curl -f -s https://core-tmp.fly.dev/health > /dev/null || echo "⚠️ Core server not responding"
	@echo "Testing RTM server..."
	@curl -f -s https://rtm.fly.dev/health > /dev/null || echo "⚠️ RTM server not responding"
	@bash scripts/test/project-health-check.sh

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

# Build the RTM server (production target)
build-rtm: test docs-tree
	@echo "Building $(RTM_BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)

# Build the Spektrix server
build-spektrix: test docs-tree
	@echo "Building $(SPEKTRIX_BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(SPEKTRIX_BINARY_NAME) $(SPEKTRIX_BUILD_DIR)

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
	@echo "Main server will be available at: http://localhost:8080"
	@echo "Debug endpoints:"
	@echo "  - Health: http://localhost:8080/debug/health"
	@echo "  - Stats: http://localhost:8080/debug/stats"
	@echo "  - Sessions: http://localhost:8080/debug/sessions"
	@echo ""
	MCP_DEBUG_ENABLED=true MCP_DEBUG_LEVEL=INFO $(OUTPUT_DIR)/$(DEBUG_PROXY_NAME) \
		--target $(OUTPUT_DIR)/$(BINARY_NAME) \
		--port 8080 \
		--target-port 8081

# Run unit tests (fast, no build tags)
unit-test:
	@echo "Running unit tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testdox -- -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	else \
		$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	fi

# Run integration tests using TestMain (no shell scripts)
integration-test:
	@echo "Running integration tests with Go TestMain..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testdox -- -race -tags=integration -timeout 60s ./tests/integration/...; \
	else \
		$(GOTEST) -v -race -tags=integration -timeout 60s ./tests/integration/...; \
	fi

# Run integration tests locally (same as integration-test now)
integration-test-local: integration-test

# Run scenario tests (protocol conformance)
scenario-test:
	@echo "Running protocol conformance tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testdox -- -v -tags=scenario -timeout 30s $(SCENARIO_TEST_DIR)/...; \
	else \
		$(GOTEST) -v -tags=scenario -timeout 30s $(SCENARIO_TEST_DIR)/...; \
	fi

# Run scenario tests against local server
scenario-test-local:
	@echo "Starting local server and running scenario tests..."
	@# Kill any existing process on port 8080
	@lsof -ti:8080 | xargs kill -9 2>/dev/null || true
	@sleep 1
	@# Build the binary first
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	@echo "Starting server, logging to server.log..."
	@rm -f server.log
	@if [ -f .test-times.log ]; then \
		LAST_TIME=$$(tail -1 .test-times.log | cut -d' ' -f2); \
		echo "Last run took $$LAST_TIME seconds"; \
	fi
	@START_TIME=$$(date +%s); \
	FLY_APP_NAME=local-test ./bin/mcp-adapters --disable-auth > server.log 2>&1 & \
	SERVER_PID=$$!; \
	trap 'echo " ▶ Trapped signal, stopping server..."; kill $$SERVER_PID 2>/dev/null || true; if [ $$TEST_EXIT -ne 0 ]; then echo "--- Server Log ---"; cat server.log; fi' EXIT INT TERM; \
	sleep 3; \
	echo "Server started with PID $$SERVER_PID"; \
	echo "Running Go scenario tests..."; \
	if [ -n "$(GOTESTSUM)" ]; then \
		export MCP_SERVER_URL="http://localhost:8080/mcp" && $(GOTESTSUM) --format testdox --jsonfile test-results.json -- -timeout 120s $(SCENARIO_TEST_DIR)/...; \
	else \
		export MCP_SERVER_URL="http://localhost:8080/mcp" && $(GOTEST) -v -timeout 120s -json $(SCENARIO_TEST_DIR)/... | tee test-results.json; \
	fi; \
	TEST_EXIT=$$?; \
	echo "Running shell script scenario tests..."; \
	cd scripts/test && ./run-tests.sh quick; \
	SHELL_EXIT=$$?; \
	echo " ▶ Stopping server with PID $$SERVER_PID"; \
	kill $$SERVER_PID 2>/dev/null || true; \
	wait $$SERVER_PID 2>/dev/null || true; \
	END_TIME=$$(date +%s); \
	DURATION=$$((END_TIME - START_TIME)); \
	echo "$$(date '+%Y-%m-%d %H:%M:%S') $$DURATION" >> .test-times.log; \
	echo " ▶ Total test time: $$DURATION seconds"; \
	if [ -f test-results.json ]; then \
		bash scripts/utils/track-performance.sh; \
	fi; \
	if [ $$TEST_EXIT -ne 0 ] || [ $$SHELL_EXIT -ne 0 ]; then \
		echo "--- Server Log (from Makefile) ---" && cat server.log; \
		exit 1; \
	fi

# Run scenario tests against production
scenario-test-prod:
	@echo "Running scenario tests against production..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		export MCP_SERVER_URL="https://core-tmp.fly.dev/" && $(GOTESTSUM) --format testdox -- $(SCENARIO_TEST_DIR)/...; \
	else \
		export MCP_SERVER_URL="https://core-tmp.fly.dev/" && $(GOTEST) -v $(SCENARIO_TEST_DIR)/...; \
	fi

# Run raw SSE/JSON-RPC tests
scenario-test-raw:
	@echo "Running raw SSE/JSON-RPC tests..."
	@# Build the binary first
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	@# Start server in background
	FLY_APP_NAME=local-test $(OUTPUT_DIR)/$(BINARY_NAME) & \
	SERVER_PID=$$!; \
	sleep 3; \
	echo "Server started with PID $$SERVER_PID"; \
	bash $(SCENARIO_TEST_DIR)/raw_sse_test.sh http://localhost:8080; \
	TEST_EXIT=$$?; \
	echo "Stopping server with PID $$SERVER_PID"; \
	kill $$SERVER_PID 2>/dev/null || true; \
	wait $$SERVER_PID 2>/dev/null || true; \
	exit $$TEST_EXIT

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
		$(GOTESTSUM) --format testdox -- -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	else \
		$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	fi
	@echo "Running integration tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testdox -- -race ./tests/...; \
	else \
		$(GOTEST) -v -race ./tests/...; \
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

# Cleanup ephemeral test server
cleanup-core-tmp:
	@echo "Destroying ephemeral core-tmp app..."
	fly apps destroy core-tmp --yes 2>/dev/null || true
	@echo "✅ Cleanup complete"

# Deploy RTM server (production target)
deploy-rtm: build-rtm
	@echo "Deploying RTM server to rtm app..."
	fly deploy -a rtm -c fly-rtm.toml

# Deploy Spektrix server
deploy-spektrix: build-spektrix
	@echo "Deploying Spektrix server..."
	fly apps create spektrix || true
	fly deploy -a spektrix --build-arg SERVER_TYPE=spektrix
	@echo ""
	@echo "⚠️  Set Spektrix secrets:"
	@echo "fly secrets set -a spektrix SPEKTRIX_CLIENT_NAME=your_client SPEKTRIX_API_USER=your_user SPEKTRIX_API_KEY=your_key"
	@echo ""

# CI-specific test with junit output
test-ci:
	@echo "Running tests for CI..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --junitfile test-results.xml --format testname -- -race -coverprofile=$(COVERAGE_FILE) ./...; \
	else \
		$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) ./...; \
	fi

# Show help
include makefile_help.mk
