# dk Makefile
# Multi-module Go workspace build system

# Install location for binaries
GOPATH := $(HOME)/go

.PHONY: all build test lint clean help
.PHONY: build-contracts build-sdk build-cli build-controller
.PHONY: test-contracts test-sdk test-cli test-controller
.PHONY: lint-contracts lint-sdk lint-cli lint-controller
.PHONY: tidy install run-local
.PHONY: helm-deps

# Default target
all: lint test build

##@ Build

build: build-contracts build-sdk build-cli build-controller ## Build all modules
	@echo "✓ All modules built successfully"

build-contracts:
	@echo "Building contracts..."
	@cd contracts && go build ./...

build-sdk:
	@echo "Building sdk..."
	@cd sdk && go build ./...

build-cli: ## Build CLI binary
	@echo "Building cli..."
	@cd cli && go build -o ../bin/dk .

build-controller: ## Build controller binary
	@echo "Building controller..."
	@cd platform/controller && go build -o ../../bin/controller ./cmd/

##@ Test

test: test-contracts test-sdk test-cli test-controller ## Run all tests
	@echo "✓ All tests passed"

test-unit: test-contracts test-sdk test-cli test-controller ## Run unit tests only
	@echo "✓ All unit tests passed"

test-e2e: ## Run E2E tests
	@echo "Running E2E tests..."
	@cd tests/e2e && go test -v ./...
	@echo "✓ E2E tests passed"

test-short: ## Run short tests (skip E2E)
	@echo "Running short tests (skipping E2E)..."
	@cd contracts && go test -short ./...
	@cd sdk && go test -short ./...
	@cd cli && go test -short ./...
	@cd platform/controller && go test -short ./...
	@echo "✓ Short tests passed"

test-race: ## Run tests with race detector
	@echo "Running tests with race detector..."
	@cd contracts && go test -race ./...
	@cd sdk && go test -race ./...
	@cd cli && go test -race ./...
	@cd platform/controller && go test -race ./...
	@echo "✓ Race detection tests passed"

test-contracts:
	@echo "Testing contracts..."
	@cd contracts && go test -race ./...

test-sdk:
	@echo "Testing sdk..."
	@cd sdk && go test -race ./...

test-cli:
	@echo "Testing cli..."
	@cd cli && go test -race ./...

test-controller:
	@echo "Testing controller..."
	@cd platform/controller && go test -race ./...

coverage: ## Generate coverage reports
	@mkdir -p coverage
	@cd contracts && go test -coverprofile=../coverage/contracts.out ./...
	@cd sdk && go test -coverprofile=../coverage/sdk.out ./...
	@cd cli && go test -coverprofile=../coverage/cli.out ./...
	@cd platform/controller && go test -coverprofile=../../coverage/controller.out ./...
	@echo "Coverage reports generated in coverage/"

test-coverage: coverage ## Display coverage summary
	@echo "Coverage summary:"
	@go tool cover -func=coverage/contracts.out | grep total || true
	@go tool cover -func=coverage/sdk.out | grep total || true
	@go tool cover -func=coverage/cli.out | grep total || true
	@go tool cover -func=coverage/controller.out | grep total || true

##@ Lint

lint: lint-contracts lint-sdk lint-cli lint-controller ## Run linting on all modules
	@echo "✓ All linting passed"

lint-contracts:
	@echo "Linting contracts..."
	@cd contracts && go vet ./... && go fmt ./...

lint-sdk:
	@echo "Linting sdk..."
	@cd sdk && go vet ./... && go fmt ./...

lint-cli:
	@echo "Linting cli..."
	@cd cli && go vet ./... && go fmt ./...

lint-controller:
	@echo "Linting controller..."
	@cd platform/controller && go vet ./... && go fmt ./...

##@ Development

tidy: ## Tidy all go.mod files
	@echo "Tidying modules..."
	@cd contracts && go mod tidy
	@cd sdk && go mod tidy
	@cd cli && go mod tidy
	@cd platform/controller && go mod tidy

install: ## Install dk to GOPATH/bin
	@echo "Installing dk to $(GOPATH)/bin..."
	@cd cli && go build -o $(GOPATH)/bin/dk .
	@echo "✓ Installed dk to $(GOPATH)/bin/dk"

run-local: ## Start local dev stack
	@echo "Starting local development stack..."
	@cd cli && go run . dev up
	@echo "✓ Local stack running"

stop-local: ## Stop local dev stack
	@echo "Stopping local development stack..."
	@cd cli && go run . dev down
	@echo "✓ Local stack stopped"

##@ Helm

helm-deps: ## Build Helm chart dependencies
	@echo "Building Helm chart dependencies..."
	@for chart in sdk/localdev/charts/redpanda sdk/localdev/charts/postgres; do \
		if [ -f "$$chart/Chart.yaml" ] && grep -q "dependencies:" "$$chart/Chart.yaml"; then \
			echo "  Building deps for $$(basename $$chart)..."; \
			helm dependency build "$$chart"; \
		fi; \
	done
	@echo "✓ Helm chart dependencies built"

##@ Code Generation

generate: ## Run go generate
	@echo "Generating code..."
	@cd platform/controller && go generate ./...

manifests: ## Generate CRD manifests
	@echo "Generating CRD manifests..."
	@cd platform/controller && controller-gen crd paths="./..." output:crd:artifacts:config=config/crd

##@ Release

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

release-cli: ## Build release CLI binaries
	@echo "Building release CLI $(VERSION)..."
	@cd cli && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o ../bin/dk-linux-amd64 .
	@cd cli && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o ../bin/dk-linux-arm64 .
	@cd cli && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o ../bin/dk-darwin-amd64 .
	@cd cli && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o ../bin/dk-darwin-arm64 .
	@echo "✓ Release binaries in bin/"

release-controller: ## Build controller Docker image
	@echo "Building controller image..."
	@docker build -t cdpp-controller:$(VERSION) -f platform/controller/Dockerfile .
	@echo "✓ Controller image built"

##@ Cleanup

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -rf coverage/
	@rm -rf dist/
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@echo "✓ Clean complete"

##@ Help

# ANSI color codes
# Disabled automatically in CI (GitHub Actions, Jenkins, GitLab, Travis, CircleCI)
# or when NO_COLOR=1 is set (see https://no-color.org)
CI_DETECTED := $(or $(NO_COLOR),$(CI),$(GITHUB_ACTIONS),$(JENKINS_URL),$(BUILD_NUMBER),$(GITLAB_CI),$(TRAVIS),$(CIRCLECI))
ifdef CI_DETECTED
  C_RESET  :=
  C_BOLD   :=
  C_CYAN   :=
  C_GREEN  :=
  C_YELLOW :=
  C_DIM    :=
else
  C_RESET  := \033[0m
  C_BOLD   := \033[1m
  C_CYAN   := \033[36m
  C_GREEN  := \033[32m
  C_YELLOW := \033[33m
  C_DIM    := \033[2m
endif

help: ## Show this help
	@printf "$(C_BOLD)Data Kit Build System$(C_RESET)\n\n"
	@printf "$(C_DIM)Usage:$(C_RESET) make $(C_CYAN)<target>$(C_RESET)\n"
	@awk 'BEGIN {FS = ":.*## "} \
		/^##@/ { printf "\n$(C_YELLOW)%s$(C_RESET)\n", substr($$0, 5); next } \
		/^[a-zA-Z_0-9-]+:.*## / { printf "  $(C_GREEN)%-20s$(C_RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
