---
title: Testing
description: Testing strategy and best practices for the Data Platform
---

# Testing Guide

This document describes the testing strategy and best practices for the Data Platform (DP) codebase.

## Quick Start

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests for a specific module
cd sdk && go test -v ./...

# Run tests in short mode (skip E2E)
go test -short ./...
```

## Test Organization

Tests are organized following Go conventions:

```
contracts/
├── datapackage.go
├── datapackage_test.go     # Unit tests for datapackage.go
└── testdata/               # Test fixtures

sdk/
├── validate/
│   ├── validator.go
│   ├── validator_test.go   # Unit tests
│   └── testdata/           # Valid/invalid fixtures
└── manifest/
    ├── parser.go
    ├── parser_test.go
    └── testdata/

cli/
├── cmd/
│   ├── lint.go
│   └── lint_test.go
└── internal/
    └── testutil/           # Shared test utilities

tests/
└── e2e/                    # End-to-end tests
    ├── workflow_test.go
    └── testdata/
```

## Test Patterns

### Table-Driven Tests

We use table-driven tests for functions with multiple input scenarios:

```go
func TestValidatePackage(t *testing.T) {
    tests := []struct {
        name    string
        input   *contracts.DataPackage
        wantErr bool
    }{
        {"valid package", validPackage(), false},
        {"missing name", &contracts.DataPackage{}, true},
        {"empty version", &contracts.DataPackage{Name: "test"}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidatePackage(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Test Fixtures

Test data is stored in `testdata/` directories:

```
testdata/
├── valid/
│   ├── pipeline-full.yaml
│   └── datapackage-basic.yaml
├── invalid/
│   ├── missing-name.yaml
│   └── invalid-binding.yaml
└── golden/
    └── expected-output.json
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

### Mocking

We use interface-based mocks for external dependencies:

```go
// Mock implementation
type mockRegistryClient struct {
    pushFunc func(ctx context.Context, artifact Artifact) error
}

func (m *mockRegistryClient) Push(ctx context.Context, a Artifact) error {
    if m.pushFunc != nil {
        return m.pushFunc(ctx, a)
    }
    return nil
}

// Usage in test
client := &mockRegistryClient{
    pushFunc: func(ctx context.Context, a Artifact) error {
        return nil // Simulate success
    },
}
```

## Running Tests

### Unit Tests

```bash
# All unit tests
make test-unit

# Specific package
cd sdk && go test -v ./validate/...

# With race detection
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View HTML report
```

### End-to-End Tests

E2E tests require Docker and test the full workflow:

```bash
# Run E2E tests
make test-e2e

# Skip E2E in short mode
go test -short ./...
```

E2E tests:
- Use the real `dp` CLI binary
- Run in isolated temp directories
- Clean up after themselves
- Skip automatically if Docker is unavailable

### Coverage Reports

```bash
# Generate coverage report
make coverage

# View coverage by function
go tool cover -func=coverage/combined.out

# View HTML report
go tool cover -html=coverage/sdk.out
```

## Coverage Targets

| Package | Target | Rationale |
|---------|--------|-----------|
| contracts/ | 90% | Core types, high stability |
| sdk/validate/ | 80% | Business logic |
| sdk/manifest/ | 80% | Parsing logic |
| sdk/registry/ | 70% | External integration |
| cli/cmd/ | 60% | UI layer |
| Overall | 70% | CI threshold |

## Writing New Tests

### Checklist

1. ✅ Test file named `*_test.go` in same package
2. ✅ Use table-driven pattern for multiple scenarios
3. ✅ Mock external dependencies
4. ✅ Include edge cases and error conditions
5. ✅ Add fixtures to `testdata/` if needed
6. ✅ Run `go test -race` to check for races

### Common Patterns

**Testing errors:**
```go
if (err != nil) != tt.wantErr {
    t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
}
if tt.wantErrMsg != "" && !strings.Contains(err.Error(), tt.wantErrMsg) {
    t.Errorf("error message = %v, want containing %v", err, tt.wantErrMsg)
}
```

**Testing CLI commands:**
```go
cmd := NewRootCmd()
buf := new(bytes.Buffer)
cmd.SetOut(buf)
cmd.SetArgs([]string{"lint", "--strict"})
err := cmd.Execute()
```

**Testing file operations:**
```go
tmpDir := t.TempDir()  // Automatically cleaned up
path := filepath.Join(tmpDir, "test.yaml")
err := os.WriteFile(path, data, 0644)
```

## CI Integration

Tests run automatically on every PR:

- **Lint**: golangci-lint on all modules
- **Test**: go test with race detection and coverage
- **Build**: Verify all modules build successfully

Coverage reports are uploaded as artifacts and the coverage percentage is displayed in the job summary.

## Troubleshooting

### Tests fail with "package not found"

Ensure you're in the correct module directory:

```bash
cd sdk && go test ./...  # Not from repo root
```

### E2E tests require Docker

Check Docker is running:

```bash
docker info
```

Skip E2E tests if Docker is unavailable:

```bash
go test -short ./...
```

### Coverage is below threshold

Identify untested code:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep -v "100.0%"
```
