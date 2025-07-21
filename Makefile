# Cowpilot Makefile

# Variables
BINARY_NAME=cowpilot
GO=go
GOTEST=$(GO) test
GOVET=$(GO) vet
GOFMT=gofmt
GOLINT=golangci-lint

# Build variables
BUILD_DIR=./cmd/cowpilot
MAIN_FILE=$(BUILD_DIR)/main.go
OUTPUT_DIR=./bin

# Test variables
UNIT_TEST_DIRS=./internal/...
INTEGRATION_TEST_DIR=./tests/integration
E2E_TEST_DIR=./tests/e2e
COVERAGE_FILE=coverage.out
GOTESTSUM=$(shell which gotestsum 2>/dev/null || echo "")

.PHONY: all build test unit-test integration-test e2e-test e2e-test-local e2e-test-prod e2e-test-raw test-ci clean fmt vet lint coverage run help

# Default target
all: clean fmt vet lint test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO) build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(BUILD_DIR)

# Run all tests
test: unit-test integration-test

# Run unit tests
unit-test:
	@echo "Running unit tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testname -- -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	else \
		$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) $(UNIT_TEST_DIRS); \
	fi

# Run integration tests
integration-test:
	@echo "Running integration tests..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format testname -- -v -race $(INTEGRATION_TEST_DIR)/...; \
	else \
		$(GOTEST) -v -race $(INTEGRATION_TEST_DIR)/...; \
	fi

# Run e2e tests (for CI/staging)
e2e-test:
	@echo "Running e2e tests..."
	@if [ -z "$(MCP_SERVER_URL)" ]; then \
		echo "MCP_SERVER_URL not set. Using production server..."; \
		export MCP_SERVER_URL="https://cowpilot.fly.dev/"; \
	fi
	@if [ -n "$(GOTESTSUM)" ]; then \
		$(GOTESTSUM) --format dots-v2 -- -v $(E2E_TEST_DIR)/...; \
	else \
		$(GOTEST) -v $(E2E_TEST_DIR)/...; \
	fi

# Run e2e tests against local server
e2e-test-local:
	@echo "Running e2e tests against local server..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		export MCP_SERVER_URL="http://localhost:8080/" && $(GOTESTSUM) --format dots-v2 -- -v $(E2E_TEST_DIR)/...; \
	else \
		export MCP_SERVER_URL="http://localhost:8080/" && $(GOTEST) -v $(E2E_TEST_DIR)/...; \
	fi

# Run e2e tests against production
e2e-test-prod:
	@echo "Running e2e tests against production..."
	@if [ -n "$(GOTESTSUM)" ]; then \
		export MCP_SERVER_URL="https://cowpilot.fly.dev/" && $(GOTESTSUM) --format dots-v2 -- -v $(E2E_TEST_DIR)/...; \
	else \
		export MCP_SERVER_URL="https://cowpilot.fly.dev/" && $(GOTEST) -v $(E2E_TEST_DIR)/...; \
	fi

# Run raw SSE/JSON-RPC tests
e2e-test-raw:
	@echo "Running raw SSE/JSON-RPC tests..."
	@bash $(E2E_TEST_DIR)/raw_sse_test.sh

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f $(COVERAGE_FILE)

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
	@which $(GOLINT) > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	$(GOLINT) run

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

# Deploy to Fly.io
deploy: test build
	@echo "Deploying to Fly.io..."
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
	@echo "  all              - Clean, format, vet, lint, test, and build"
	@echo "  build            - Build the binary"
	@echo "  test             - Run all tests"
	@echo "  unit-test        - Run unit tests with coverage"
	@echo "  integration-test - Run integration tests"
	@echo "  e2e-test         - Run end-to-end tests (uses MCP_SERVER_URL or production)"
	@echo "  e2e-test-local   - Run e2e tests against local server (localhost:8080)"
	@echo "  e2e-test-prod    - Run e2e tests against production (cowpilot.fly.dev)"
	@echo "  e2e-test-raw     - Run raw SSE/JSON-RPC tests using curl and jq"
	@echo "  clean            - Remove build artifacts"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run linter (requires golangci-lint)"
	@echo "  coverage         - Generate test coverage report"
	@echo "  run              - Build and run locally"
	@echo "  dev              - Run with hot reload (requires air)"
	@echo "  deploy           - Test, build, and deploy to Fly.io"
