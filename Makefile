# MCP Manager Makefile

.PHONY: all build proto clean test help

# Build variables
BINARY_NAME=mcp-manager
DAEMON_NAME=mcp-daemon
BUILD_DIR=./bin
DAEMON_PATH=./cmd/mcp-daemon/main.go
CLI_PATH=./cmd/mcp-manager/main.go

# Proto variables
PROTO_DIR=proto
PROTO_OUT=internal/grpc/pb
PROTO_FILES=$(wildcard $(PROTO_DIR)/*.proto)

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Default target
all: proto build

# Generate protobuf code
proto:
	@echo "🔧 Generating protobuf code..."
	@mkdir -p $(PROTO_OUT)
	@if [ -z "$(PROTO_FILES)" ]; then \
		echo "⚠️  No .proto files found in $(PROTO_DIR)"; \
	else \
		protoc \
			--go_out=$(PROTO_OUT) \
			--go_opt=paths=source_relative \
			--go-grpc_out=$(PROTO_OUT) \
			--go-grpc_opt=paths=source_relative \
			-I $(PROTO_DIR) \
			$(PROTO_FILES) && \
		echo "✅ Protobuf generation complete"; \
	fi

# Build all binaries
build: build-daemon build-manager

# Build daemon
build-daemon:
	@echo "🔨 Building $(DAEMON_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(DAEMON_NAME) $(DAEMON_PATH)
	@echo "✅ Built: $(BUILD_DIR)/$(DAEMON_NAME)"

# Build manager (client)
build-manager:
	@echo "🔨 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CLI_PATH)
	@echo "✅ Built: $(BUILD_DIR)/$(BINARY_NAME)"

# Aliases for convenience
daemon: build-daemon
manager: build-manager

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	@$(GOMOD) tidy
	@$(GOMOD) download
	@echo "✅ Dependencies installed"

# Run tests
test:
	@echo "🧪 Running tests..."
	@$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

# Run gRPC tests specifically
test-grpc:
	@echo "🧪 Running gRPC tests..."
	@$(GOTEST) -v ./internal/grpc/...

# Format code
fmt:
	@echo "🎨 Formatting code..."
	@$(GOFMT) -w .
	@echo "✅ Code formatted"

# Check formatting
fmt-check:
	@echo "🔍 Checking code formatting..."
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "❌ Code is not formatted. Files needing formatting:"; \
		$(GOFMT) -l .; \
		echo "Run 'make fmt' to fix"; \
		exit 1; \
	else \
		echo "✅ Code is properly formatted"; \
	fi

# Run go vet
vet:
	@echo "🔍 Running go vet..."
	@$(GOCMD) vet ./...
	@echo "✅ No issues found"

# Run all checks
check: fmt-check vet test

# Clean build artifacts
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(PROTO_OUT)
	@rm -f coverage.out coverage.html
	@echo "✅ Clean complete"

# Install binaries to $GOPATH/bin
install: build
	@echo "📦 Installing binaries..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $$($(GOCMD) env GOPATH)/bin/
	@cp $(BUILD_DIR)/$(DAEMON_NAME) $$($(GOCMD) env GOPATH)/bin/
	@echo "✅ Installed to $$($(GOCMD) env GOPATH)/bin/"

# Quick development workflow
dev: fmt proto test
	@echo "✅ Development checks complete"

# Show help
help:
	@echo "MCP Manager - Available targets:"
	@echo ""
	@echo "🔨 Building:"
	@echo "  make proto          - Generate protobuf code"
	@echo "  make build          - Build both daemon and manager"
	@echo "  make daemon         - Build daemon only"
	@echo "  make manager        - Build manager only"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test           - Run all tests"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make test-grpc      - Run gRPC tests only"
	@echo ""
	@echo "🛠️  Development:"
	@echo "  make dev            - Format, generate proto, and test"
	@echo "  make fmt            - Format code"
	@echo "  make fmt-check      - Check if code is formatted"
	@echo "  make vet            - Run go vet"
	@echo "  make check          - Run all checks (fmt, vet, test)"
	@echo ""
	@echo "📦 Other:"
	@echo "  make deps           - Install dependencies"
	@echo "  make install        - Install binaries to GOPATH/bin"
	@echo "  make clean          - Remove build artifacts"
	@echo ""
	@echo "💡 For development, use go run directly:"
	@echo "  go run ./cmd/mcp-manager -standalone    # Run TUI standalone"
	@echo "  go run ./cmd/mcp-daemon run             # Run daemon"

# Default help
.DEFAULT_GOAL := help