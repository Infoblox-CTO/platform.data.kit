# Quickstart: Running DP Tests

**Feature**: 002-unit-e2e-tests  
**Date**: 2026-01-22

## Prerequisites

- Go 1.25 or later
- Docker (for E2E tests only)
- golangci-lint (for linting)

## Quick Commands

### Run All Tests

```bash
# From repository root
make test
```

### Run Tests for Specific Module

```bash
# Contracts
cd contracts && go test ./...

# SDK
cd sdk && go test ./...

# CLI
cd cli && go test ./...

# Controller
cd platform/controller && go test ./...
```

### Run with Verbose Output

```bash
go test -v ./...
```

### Run with Coverage

```bash
# Generate coverage report
make coverage

# View HTML report
go tool cover -html=coverage/sdk.out
```

### Skip E2E Tests (Fast Mode)

```bash
go test -short ./...
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make test` | Run all unit tests |
| `make test-unit` | Run unit tests only (skip E2E) |
| `make test-e2e` | Run E2E tests only |
| `make coverage` | Generate coverage reports |
| `make test-race` | Run tests with race detection |

## Writing New Tests

### 1. Create Test File

Place `*_test.go` in the same directory as the source file:

```text
sdk/validate/
├── validator.go       # Source
└── validator_test.go  # Test
```

### 2. Use Table-Driven Pattern

```go
func TestValidateDataPackage(t *testing.T) {
    tests := []struct {
        name    string
        input   *contracts.DataPackage
        wantErr bool
    }{
        {"valid package", validPackage(), false},
        {"missing name", &contracts.DataPackage{}, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateDataPackage(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 3. Add Test Fixtures

Create `testdata/` directory for test data:

```text
sdk/validate/testdata/
├── valid/
│   └── pipeline.yaml
└── invalid/
    └── missing-name.yaml
```

Load fixtures in tests:

```go
func loadFixture(t *testing.T, path string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("testdata", path))
    if err != nil {
        t.Fatalf("failed to load fixture: %v", err)
    }
    return data
}
```

### 4. Mock External Dependencies

Use interface-based mocks:

```go
// Test with mock registry
client := &MockRegistryClient{
    PushFunc: func(ctx context.Context, a Artifact) error {
        return nil // Simulate success
    },
}

err := service.PublishWithClient(ctx, pkg, client)
```

## E2E Tests

### Running E2E Tests

```bash
# Requires Docker
cd tests/e2e
go test -v ./...
```

### E2E Test Structure

```go
func TestInitLintBuildWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test")
    }
    
    ctx := setupE2E(t)
    defer ctx.Cleanup()
    
    // Test workflow
    runDP(t, ctx, "init", "test-pkg", "--type", "pipeline")
    runDP(t, ctx, "lint")
    runDP(t, ctx, "build")
    
    // Verify outputs
    assertFileExists(t, ctx.TempDir, "dp.yaml")
}
```

## CI Integration

Tests run automatically on every PR:

```yaml
# .github/workflows/ci.yaml
- name: Test
  run: |
    cd contracts && go test -race -coverprofile=coverage.out ./...
    cd ../sdk && go test -race -coverprofile=coverage.out ./...
    cd ../cli && go test -race -coverprofile=coverage.out ./...
```

## Troubleshooting

### Tests Fail with "package not found"

Ensure you're in the correct directory (tests run per-module):

```bash
cd sdk && go test ./...
```

### Coverage Below Threshold

Check which functions are untested:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -v "100.0%"
```

### E2E Tests Require Docker

E2E tests that use the runner require Docker:

```bash
docker info  # Verify Docker is running
```

Skip E2E tests if Docker is unavailable:

```bash
go test -short ./...
```
