# Research: Unit and End-to-End Tests

**Feature**: 002-unit-e2e-tests  
**Date**: 2026-01-22  
**Status**: Complete

## Research Tasks

### 1. Go Testing Best Practices

**Question**: What are the best practices for writing unit tests in Go?

**Decision**: Use Go stdlib `testing` package with table-driven tests.

**Rationale**:
- Table-driven tests are idiomatic Go and recommended in the Go blog
- Provides clear input/output documentation
- Easy to add new test cases
- Works well with `go test -v` for readable output

**Alternatives Considered**:
- BDD frameworks (Ginkgo/Gomega): Rejected - adds complexity, not idiomatic
- External test runners: Rejected - stdlib is sufficient

**Best Practices to Apply**:
```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"invalid input", invalidInput, nil, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("Function() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("Function() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

### 2. Mocking External Dependencies in Go

**Question**: How to mock HTTP clients, Docker, and file systems for isolated unit tests?

**Decision**: Use interface-based mocking with hand-written mock implementations.

**Rationale**:
- Go interfaces enable dependency injection naturally
- Hand-written mocks are simple and explicit
- No external mocking framework required
- Existing codebase already uses interfaces (e.g., `registry.Client`)

**Alternatives Considered**:
- `gomock`: Rejected - generates complex mock code, maintenance overhead
- `testify/mock`: Acceptable alternative, but hand-written mocks are cleaner for our scope
- `httptest`: Use for HTTP testing (stdlib, no rejection)

**Implementation Pattern**:
```go
// In production code: define interface
type RegistryClient interface {
    Push(ctx context.Context, artifact Artifact) error
    Pull(ctx context.Context, ref string) (Artifact, error)
}

// In test file: create mock
type mockRegistryClient struct {
    pushFunc func(ctx context.Context, artifact Artifact) error
    pullFunc func(ctx context.Context, ref string) (Artifact, error)
}

func (m *mockRegistryClient) Push(ctx context.Context, artifact Artifact) error {
    return m.pushFunc(ctx, artifact)
}
```

---

### 3. Test Fixtures and testdata Directory

**Question**: How to organize test data (sample manifests, configurations)?

**Decision**: Use `testdata/` directories in each package with representative fixtures.

**Rationale**:
- `testdata/` is a Go convention (ignored by build tools)
- Co-located with tests for easy maintenance
- Can include both valid and invalid samples
- Supports golden file testing patterns

**Directory Structure**:
```text
sdk/validate/testdata/
├── valid/
│   ├── pipeline-full.yaml
│   ├── pipeline-minimal.yaml
│   └── datapackage-basic.yaml
├── invalid/
│   ├── missing-name.yaml
│   ├── invalid-binding.yaml
│   └── pii-violation.yaml
└── golden/
    └── expected-output.json
```

---

### 4. CLI Testing Patterns

**Question**: How to test Cobra CLI commands effectively?

**Decision**: Test command initialization and execution using Cobra's testing utilities.

**Rationale**:
- Cobra provides `cmd.Execute()` for testing full command flow
- Can capture stdout/stderr for assertion
- Can test flag parsing and argument validation
- Avoids testing internal implementation details

**Implementation Pattern**:
```go
func TestLintCommand(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantCode int
        wantOut  string
    }{
        {"valid package", []string{"lint", "testdata/valid"}, 0, "✓ Validation passed"},
        {"invalid package", []string{"lint", "testdata/invalid"}, 1, "ERROR"},
        {"strict mode", []string{"lint", "--strict", "testdata/valid"}, 0, ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewRootCmd()
            buf := new(bytes.Buffer)
            cmd.SetOut(buf)
            cmd.SetArgs(tt.args)
            
            code := 0
            if err := cmd.Execute(); err != nil {
                code = 1
            }
            
            if code != tt.wantCode {
                t.Errorf("exit code = %d, want %d", code, tt.wantCode)
            }
        })
    }
}
```

---

### 5. E2E Testing Strategy

**Question**: How to implement end-to-end tests that validate the complete workflow?

**Decision**: Use subprocess execution of the `dp` binary in a temporary directory.

**Rationale**:
- Tests the actual CLI binary (matches real user experience)
- Isolated in temp directories (no side effects)
- Can verify file outputs and exit codes
- Supports `-short` flag for CI optimization

**Implementation Pattern**:
```go
func TestWorkflowE2E(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping E2E test in short mode")
    }
    
    // Create temp directory
    tmpDir := t.TempDir()
    
    // Build CLI binary (or use pre-built)
    binPath := buildCLI(t)
    
    // Test: init → lint → build workflow
    runCmd(t, binPath, tmpDir, "init", "test-pkg", "--type", "pipeline")
    runCmd(t, binPath, tmpDir, "lint")
    runCmd(t, binPath, tmpDir, "build")
    
    // Verify outputs exist
    assertFileExists(t, filepath.Join(tmpDir, "dp.yaml"))
    assertFileExists(t, filepath.Join(tmpDir, ".dp", "artifact.tar"))
}

func runCmd(t *testing.T, bin, dir string, args ...string) {
    t.Helper()
    cmd := exec.Command(bin, args...)
    cmd.Dir = dir
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("command failed: %s\noutput: %s", err, out)
    }
}
```

---

### 6. Test Coverage Tooling

**Question**: How to measure and report test coverage effectively?

**Decision**: Use `go test -coverprofile` with coverage merging for multi-module workspace.

**Rationale**:
- Native Go coverage is accurate and integrated
- Can generate HTML reports for visualization
- CI can enforce coverage thresholds
- Existing Makefile already has coverage target

**Commands**:
```bash
# Per-module coverage
go test -coverprofile=coverage.out ./...

# HTML report
go tool cover -html=coverage.out -o coverage.html

# Coverage percentage
go tool cover -func=coverage.out | grep total
```

**CI Integration**:
```yaml
- name: Test with coverage
  run: |
    go test -race -coverprofile=coverage.out ./...
    COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo "Coverage: $COVERAGE"
```

---

### 7. Race Detection and Test Flags

**Question**: What test flags should be standard for development and CI?

**Decision**: Use `-race` in CI, `-short` for quick local iteration.

**Rationale**:
- Race detection catches concurrency bugs early
- `-short` allows skipping slow E2E tests during development
- `-v` provides verbose output for debugging

**Standard Invocations**:
```bash
# Development (fast)
go test ./...

# Development with verbose
go test -v ./...

# CI (thorough)
go test -race -cover -coverprofile=coverage.out ./...

# Skip E2E
go test -short ./...
```

---

## Summary of Decisions

| Topic | Decision | Framework/Tool |
|-------|----------|---------------|
| Test Framework | Go stdlib | `testing` package |
| Test Pattern | Table-driven tests | Idiomatic Go |
| Mocking | Interface-based mocks | Hand-written |
| HTTP Testing | httptest server | stdlib |
| Fixtures | testdata directories | Go convention |
| CLI Testing | Cobra Execute() | Cobra testing utils |
| E2E Tests | Subprocess execution | exec.Command |
| Coverage | go test -coverprofile | Native tooling |
| CI Flags | -race -cover | Standard flags |

## Dependencies to Add

| Dependency | Purpose | Required |
|------------|---------|----------|
| `testing` | Core test framework | Yes (stdlib) |
| `net/http/httptest` | HTTP server mocking | Yes (stdlib) |
| `os/exec` | E2E subprocess tests | Yes (stdlib) |
| `github.com/stretchr/testify/assert` | Cleaner assertions | Optional |

No new external dependencies required - stdlib is sufficient.
