# Cowpilot Makefile

# Variables
BINARY_NAME=cowpilot
RTM_BINARY_NAME=rtm-server
SPEKTRIX_BINARY_NAME=spektrix-server
DEBUG_PROXY_NAME=mcp-debug-proxy
GO=go
GOTEST=$(GO) test
GOVET=$(GO) vet
GOFMT=gofmt
GOLINT=golangci-lint

# Build variables
BUILD_DIR=./cmd/everything
RTM_BUILD_DIR=./cmd/rtm-server
SPEKTRIX_BUILD_DIR=./cmd/spektrix-server
DEBUG_PROXY_DIR=./cmd/mcp-debug-proxy
MAIN_FILE=$(BUILD_DIR)/main.go
RTM_MAIN_FILE=$(RTM_BUILD_DIR)/main.go
SPEKTRIX_MAIN_FILE=$(SPEKTRIX_BUILD_DIR)/main.go
DEBUG_PROXY_FILE=$(DEBUG_PROXY_DIR)/main.go
OUTPUT_DIR=./bin

# Test variables
UNIT_TEST_DIRS=./cmd/... ./internal/...
SCENARIO_TEST_DIR=./tests/scenarios
COVERAGE_FILE=coverage.out
GOTESTSUM=$(shell which gotestsum 2>/dev/null || echo "")

.PHONY: all build build-rtm build-debug debug-proxy run-debug-proxy test unit-test integration-test scenario-test scenario-test-local scenario-test-prod scenario-test-raw test-ci clean fmt vet lint coverage run help test-verbose

# Clean, format, lint, and run ALL tests including scenarios
all: clean fmt vet lint test scenario-test-local

# Run ALL tests (unit, integration, and scenario)
test: unit-test integration-test scenario-test-local

# Build the RTM server (production target)
build-rtm: test
	@echo "Building $(RTM_BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)

# Build the Spektrix server
build-spektrix: test
	@echo "Building $(SPEKTRIX_BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(SPEKTRIX_BINARY_NAME) $(SPEKTRIX_BUILD_DIR)

# Build all servers
build-all: test
	@echo "Building all servers..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(RTM_BINARY_NAME) $(RTM_BUILD_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(SPEKTRIX_BINARY_NAME) $(SPEKTRIX_BUILD_DIR)

# Build the everything server (testing target)
build: test
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

# Run unit tests
unit-test:
	@echo "Running unit tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testdox -- -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	else \
		$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	fi

# Run integration tests
integration-test:
	@echo "Running integration tests against deployed instance..."
	@export MCP_SERVER_URL="https://cowpilot.fly.dev/mcp" && \
	if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testdox -- -race ./tests/...; \
	else \
		$(GOTEST) -v -race ./tests/...; \
	fi

# Run integration tests locally (development only)
integration-test-local:
	@echo "Running integration tests against local server..."
	@LOCAL_TEST=true ./scripts/test/test-mcp-integration.sh

# Run scenario tests (for CI/staging)
scenario-test:
	@echo "Running scenario tests..."
	@if [ -z "$(MCP_SERVER_URL)" ]; then \
		echo "MCP_SERVER_URL not set. Using production server..."; \
		export MCP_SERVER_URL="https://cowpilot.fly.dev/"; \
	fi
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format dots-v2 -- -v $(SCENARIO_TEST_DIR)/...; \
	else \
		$(GOTEST) -v $(SCENARIO_TEST_DIR)/...; \
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
	FLY_APP_NAME=local-test ./bin/cowpilot --disable-auth > server.log 2>&1 & \
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
		export MCP_SERVER_URL="https://cowpilot.fly.dev/" && $(GOTESTSUM) --format testdox -- $(SCENARIO_TEST_DIR)/...; \
	else \
		export MCP_SERVER_URL="https://cowpilot.fly.dev/" && $(GOTEST) -v $(SCENARIO_TEST_DIR)/...; \
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

# Deploy RTM server to Fly.io (runs tests first)
deploy: build-rtm
	@echo "All tests passed. Deploying RTM server to Fly.io..."
	fly deploy

# CI-specific test with junit output
test-ci:
	@echo "Running tests for CI..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --junitfile test-results.xml --format testname -- -race -coverprofile=$(COVERAGE_FILE) ./...; \
	else \
		$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) ./...; \
	fi

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Clean, format, vet, lint, test, and run scenarios"
	@echo "  build            - Run all tests then build the binary"
	@echo "  build-debug      - Build the debug proxy binary"
	@echo "  debug-proxy      - Build both main application and debug proxy"
	@echo "  run-debug-proxy  - Build and run the debug proxy with default settings"
	@echo "  test             - Run all tests (unit, integration, and scenarios)"
	@echo "  unit-test        - Run unit tests with coverage"
	@echo "  integration-test - Run integration tests"
	@echo "  scenario-test    - Run scenario tests (Go + shell scripts)"
	@echo "  scenario-test-local - Run scenario tests against local server (localhost:8080)"
	@echo "  scenario-test-prod - Run scenario tests against production (cowpilot.fly.dev)"
	@echo "  scenario-test-raw - Run raw SSE/JSON-RPC tests using curl and jq"
	@echo "  clean            - Remove build artifacts"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run linter (requires golangci-lint)"
	@echo "  coverage         - Generate test coverage report"
	@echo "  run              - Build and run locally"
	@echo "  dev              - Run with hot reload (requires air)"
	@echo "  deploy           - Test, build, and deploy to Fly.io"
	@echo "  test-verbose     - Run tests with human-readable output"