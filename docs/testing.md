---
title: Testing
description: Testing strategy and best practices for the Data Platform
---

# Testing Guide

This document describes the testing strategy and best practices for the DataKit codebase.

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
├── transform.go
├── transform_test.go       # Unit tests for transform.go
├── asset.go
├── asset_test.go           # Unit tests for asset.go
├── connector.go
├── store.go
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
func TestValidateTransform(t *testing.T) {
    tests := []struct {
        name    string
        input   *contracts.Transform
        wantErr bool
    }{
        {"valid transform", validTransform(), false},
        {"missing name", &contracts.Transform{}, true},
        {"empty version", &contracts.Transform{Metadata: contracts.TransformMetadata{Name: "test"}}, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateTransform(tt.input)
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
│   ├── transform-full.yaml
│   └── asset-basic.yaml
├── invalid/
│   ├── missing-name.yaml
│   └── invalid-store-ref.yaml
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
- Use the real `dk` CLI binary
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

- **Test**: go test with race detection and coverage
- **Build**: Verify all modules build successfully

Coverage reports are uploaded as artifacts and the coverage percentage is displayed in the job summary.

## Using Seed Profiles for Test Data

Assets can declare multiple **seed profiles** in their `dev.seed` section.
This lets you set up different data scenarios for tests without writing SQL
or managing fixture files separately.

### Defining profiles

```yaml title="asset/source.yaml"
spec:
  dev:
    seed:
      inline:                          # default profile
        - { id: 1, name: "alice" }
        - { id: 2, name: "bob" }
      profiles:
        large:
          file: testdata/1000-rows.csv
        edge-cases:
          inline:
            - { id: -1, name: "" }
            - { id: 999, name: "O'Reilly" }
        empty: {}
```

### Loading profiles in tests

Switch between profiles to run the same pipeline against different data:

```bash
# Reset to default data
dk dev seed

# Run with edge-case data
dk dev seed --profile edge-cases
dk run

# Run with a large data set
dk dev seed --profile large
dk run

# Test with an empty table
dk dev seed --profile empty --clean
dk run
```

### Idempotency

Seed runs are **idempotent by default**. A SHA-256 checksum of the resolved
data is stored in a `_dp_seed_meta` table in PostgreSQL. On subsequent runs,
if the data hasn't changed, the seed is skipped entirely — no duplicate-key
errors, no unnecessary writes.

Use `--force` to re-seed even when the checksum matches, or `--clean` to
`DROP` and recreate the table from scratch.

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
