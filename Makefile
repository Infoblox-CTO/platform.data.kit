# CDPP Makefile
# Multi-module Go workspace build system

.PHONY: all build test lint clean help
.PHONY: build-contracts build-sdk build-cli build-controller
.PHONY: test-contracts test-sdk test-cli test-controller
.PHONY: lint-contracts lint-sdk lint-cli lint-controller
.PHONY: tidy install run-local

# Default target
all: lint test build

# ============================================================================
# Build targets
# ============================================================================

build: build-contracts build-sdk build-cli build-controller
	@echo "✓ All modules built successfully"

build-contracts:
	@echo "Building contracts..."
	@cd contracts && go build ./...

build-sdk:
	@echo "Building sdk..."
	@cd sdk && go build ./...

build-cli:
	@echo "Building cli..."
	@cd cli && go build -o ../bin/cdpp .

build-controller:
	@echo "Building controller..."
	@cd platform/controller && go build -o ../../bin/controller ./cmd/

# ============================================================================
# Test targets
# ============================================================================

test: test-contracts test-sdk test-cli test-controller
	@echo "✓ All tests passed"

test-contracts:
	@echo "Testing contracts..."
	@cd contracts && go test -race -cover ./...

test-sdk:
	@echo "Testing sdk..."
	@cd sdk && go test -race -cover ./...

test-cli:
	@echo "Testing cli..."
	@cd cli && go test -race -cover ./...

test-controller:
	@echo "Testing controller..."
	@cd platform/controller && go test -race -cover ./...

# Coverage report
coverage:
	@mkdir -p coverage
	@cd contracts && go test -coverprofile=../coverage/contracts.out ./...
	@cd sdk && go test -coverprofile=../coverage/sdk.out ./...
	@cd cli && go test -coverprofile=../coverage/cli.out ./...
	@cd platform/controller && go test -coverprofile=../../coverage/controller.out ./...
	@echo "Coverage reports generated in coverage/"

# ============================================================================
# Lint targets
# ============================================================================

lint: lint-contracts lint-sdk lint-cli lint-controller
	@echo "✓ All linting passed"

lint-contracts:
	@echo "Linting contracts..."
	@cd contracts && golangci-lint run ./...

lint-sdk:
	@echo "Linting sdk..."
	@cd sdk && golangci-lint run ./...

lint-cli:
	@echo "Linting cli..."
	@cd cli && golangci-lint run ./...

lint-controller:
	@echo "Linting controller..."
	@cd platform/controller && golangci-lint run ./...

lint-fix:
	@echo "Fixing lint issues..."
	@cd contracts && golangci-lint run --fix ./...
	@cd sdk && golangci-lint run --fix ./...
	@cd cli && golangci-lint run --fix ./...
	@cd platform/controller && golangci-lint run --fix ./...

# ============================================================================
# Development targets
# ============================================================================

tidy:
	@echo "Tidying modules..."
	@cd contracts && go mod tidy
	@cd sdk && go mod tidy
	@cd cli && go mod tidy
	@cd platform/controller && go mod tidy

install: build-cli
	@echo "Installing cdpp to GOPATH/bin..."
	@cp bin/cdpp $(GOPATH)/bin/cdpp
	@echo "✓ Installed cdpp"

run-local:
	@echo "Starting local development stack..."
	@docker compose -f hack/compose/docker-compose.yaml up -d
	@echo "✓ Local stack running"

stop-local:
	@echo "Stopping local development stack..."
	@docker compose -f hack/compose/docker-compose.yaml down
	@echo "✓ Local stack stopped"

# ============================================================================
# Code generation targets
# ============================================================================

generate:
	@echo "Generating code..."
	@cd platform/controller && go generate ./...

manifests:
	@echo "Generating CRD manifests..."
	@cd platform/controller && controller-gen crd paths="./..." output:crd:artifacts:config=config/crd

# ============================================================================
# Release targets
# ============================================================================

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

release-cli:
	@echo "Building release CLI $(VERSION)..."
	@cd cli && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../bin/cdpp-linux-amd64 .
	@cd cli && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o ../bin/cdpp-linux-arm64 .
	@cd cli && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ../bin/cdpp-darwin-amd64 .
	@cd cli && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ../bin/cdpp-darwin-arm64 .
	@echo "✓ Release binaries in bin/"

release-controller:
	@echo "Building controller image..."
	@docker build -t cdpp-controller:$(VERSION) -f platform/controller/Dockerfile .
	@echo "✓ Controller image built"

# ============================================================================
# Cleanup targets
# ============================================================================

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf coverage/
	@rm -rf dist/
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@echo "✓ Clean complete"

# ============================================================================
# Help
# ============================================================================

help:
	@echo "CDPP Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build           Build all modules"
	@echo "  build-cli       Build CLI binary"
	@echo "  build-controller Build controller binary"
	@echo ""
	@echo "Test targets:"
	@echo "  test            Run all tests"
	@echo "  coverage        Generate coverage reports"
	@echo ""
	@echo "Lint targets:"
	@echo "  lint            Run linting on all modules"
	@echo "  lint-fix        Auto-fix lint issues"
	@echo ""
	@echo "Development targets:"
	@echo "  tidy            Tidy all go.mod files"
	@echo "  install         Install cdpp to GOPATH/bin"
	@echo "  run-local       Start local dev stack"
	@echo "  stop-local      Stop local dev stack"
	@echo ""
	@echo "Code generation:"
	@echo "  generate        Run go generate"
	@echo "  manifests       Generate CRD manifests"
	@echo ""
	@echo "Release targets:"
	@echo "  release-cli     Build release CLI binaries"
	@echo "  release-controller Build controller Docker image"
	@echo ""
	@echo "Cleanup:"
	@echo "  clean           Remove build artifacts"
