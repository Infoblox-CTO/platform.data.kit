# Data Model: Unit and End-to-End Tests

**Feature**: 002-unit-e2e-tests  
**Date**: 2026-01-22  
**Status**: Complete

## Overview

This document defines the test-related entities and structures for the DP testing infrastructure. Unlike typical data models, test entities are primarily code structures (not persisted data) that enable consistent test patterns across the codebase.

## Entities

### TestCase

Represents a single test scenario in a table-driven test.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Human-readable test case name |
| input | varies | Yes | Input data for the function under test |
| want | varies | Yes | Expected output |
| wantErr | bool | No | Whether an error is expected (default: false) |
| wantErrMsg | string | No | Expected error message substring |

**Usage Pattern**:
```go
tests := []struct {
    name    string
    input   string
    want    *DataPackage
    wantErr bool
}{
    {"valid manifest", "testdata/valid.yaml", &DataPackage{Name: "test"}, false},
    {"missing name", "testdata/missing-name.yaml", nil, true},
}
```

---

### TestFixture

Reusable test data stored in `testdata/` directories.

| Category | Location | Purpose |
|----------|----------|---------|
| Valid Manifests | `testdata/valid/` | Known-good inputs for happy path tests |
| Invalid Manifests | `testdata/invalid/` | Error condition inputs |
| Golden Files | `testdata/golden/` | Expected outputs for comparison |
| Mock Responses | `testdata/responses/` | Simulated API/service responses |

**Directory Structure**:
```text
sdk/validate/testdata/
├── valid/
│   ├── pipeline-full.yaml      # Complete pipeline manifest
│   ├── pipeline-minimal.yaml   # Minimal required fields only
│   └── datapackage-basic.yaml  # Basic data package
├── invalid/
│   ├── missing-name.yaml       # Missing required field
│   ├── invalid-binding.yaml    # Malformed binding reference
│   └── pii-no-classification.yaml  # PII field without classification
└── golden/
    └── validation-errors.json  # Expected error output format
```

---

### MockClient

Interface implementations for isolated testing.

| Mock Type | Mocks | Package |
|-----------|-------|---------|
| MockRegistryClient | registry.Client | sdk/registry/mocks/ |
| MockDockerRunner | runner.Docker | sdk/runner/mocks/ |
| MockHTTPTransport | http.RoundTripper | Inline in tests |
| MockLineageEmitter | lineage.Emitter | sdk/lineage/mocks/ |

**MockRegistryClient Definition**:
```go
type MockRegistryClient struct {
    PushFunc func(ctx context.Context, artifact Artifact) error
    PullFunc func(ctx context.Context, ref string) (Artifact, error)
    TagsFunc func(ctx context.Context, repo string) ([]string, error)
}

func (m *MockRegistryClient) Push(ctx context.Context, a Artifact) error {
    if m.PushFunc != nil {
        return m.PushFunc(ctx, a)
    }
    return nil
}
```

---

### E2ETestContext

Context struct for end-to-end test execution.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| TempDir | string | Yes | Isolated temporary directory for test |
| BinPath | string | Yes | Path to compiled `dp` CLI binary |
| Env | map[string]string | No | Environment variables for subprocess |
| Cleanup | func() | Yes | Function to clean up resources |

**Usage**:
```go
func setupE2E(t *testing.T) *E2ETestContext {
    t.Helper()
    tmpDir := t.TempDir()
    binPath := buildOrFindCLI(t)
    
    return &E2ETestContext{
        TempDir: tmpDir,
        BinPath: binPath,
        Env: map[string]string{
            "DP_REGISTRY": "localhost:5000",
        },
        Cleanup: func() {
            // Cleanup handled by t.TempDir()
        },
    }
}
```

---

## Test File Mapping

| Package | Source Files | Test File | Priority |
|---------|-------------|-----------|----------|
| contracts/ | datapackage.go | datapackage_test.go | P1 |
| contracts/ | pipeline.go | pipeline_test.go | P1 |
| contracts/ | errors.go | errors_test.go | P1 |
| contracts/ | binding.go | binding_test.go | P1 |
| sdk/validate/ | validator.go | validator_test.go | P1 |
| sdk/validate/ | datapackage.go | datapackage_test.go | P1 |
| sdk/validate/ | pipeline.go | pipeline_test.go | P1 |
| sdk/validate/ | pii.go | pii_test.go | P1 |
| sdk/manifest/ | parser.go | parser_test.go | P1 |
| sdk/manifest/ | datapackage.go | datapackage_test.go | P1 |
| sdk/manifest/ | pipeline.go | pipeline_test.go | P1 |
| sdk/registry/ | client.go | client_test.go | P2 |
| sdk/registry/ | bundler.go | bundler_test.go | P2 |
| sdk/runner/ | runner.go | runner_test.go | P2 |
| sdk/lineage/ | events.go | events_test.go | P2 |
| sdk/lineage/ | emitter.go | emitter_test.go | P2 |
| cli/cmd/ | lint.go | lint_test.go | P2 |
| cli/cmd/ | init.go | init_test.go | P2 |
| cli/cmd/ | build.go | build_test.go | P2 |
| cli/cmd/ | run.go | run_test.go | P2 |
| cli/cmd/ | publish.go | publish_test.go | P2 |
| platform/controller/ | reconciler.go | reconciler_test.go | P2 |
| tests/e2e/ | N/A | workflow_test.go | P2 |

---

## Coverage Targets

| Package | Target Coverage | Rationale |
|---------|-----------------|-----------|
| contracts/ | 90% | Core types, high stability requirement |
| sdk/validate/ | 80% | Business logic, error conditions |
| sdk/manifest/ | 80% | Parsing logic, edge cases |
| sdk/registry/ | 70% | External dependency interactions |
| sdk/runner/ | 70% | External dependency interactions |
| sdk/lineage/ | 80% | Event creation logic |
| cli/cmd/ | 60% | UI layer, less critical |
| platform/controller/ | 70% | Reconciliation logic |

**Overall Target**: 70% combined coverage (per SC-004)
