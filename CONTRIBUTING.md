# Contributing to DP (Data Platform)

Thank you for your interest in contributing to the Data Platform! This document provides guidelines for contributing.

## 🏗️ Development Setup

### Prerequisites

- **Go 1.22+**: [Installation guide](https://go.dev/doc/install)
- **Docker Desktop**: [Installation guide](https://docs.docker.com/desktop/)
- **Make**: Usually pre-installed on macOS/Linux
- **golangci-lint**: `brew install golangci-lint` or [other methods](https://golangci-lint.run/usage/install/)

### Clone and Build

```bash
# Clone repository
git clone https://github.com/Infoblox-CTO/data-platform.git
cd data-platform

# Install dependencies
make deps

# Build all modules
make build

# Run tests
make test

# Run linter
make lint
```

## 📁 Project Structure

The project is organized as a Go monorepo with independent modules:

```
data-platform/
├── contracts/     # Shared types, schemas, validation errors
├── sdk/           # Core functionality (validate, lineage, registry, runner)
├── cli/           # DP CLI implementation
├── platform/
│   └── controller/  # Kubernetes PackageDeployment controller
├── specs/         # Feature specifications (speckit workflow)
├── examples/      # Reference data packages
├── hack/          # Development utilities
└── dashboards/    # Grafana dashboards
```

### Module Dependencies

```
contracts ← sdk ← cli
contracts ← platform/controller
```

## 🔧 Development Workflow

### 1. Pick or Create an Issue

- Check [existing issues](https://github.com/Infoblox-CTO/data-platform/issues)
- For new features, create a discussion first

### 2. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-number-description
```

### 3. Make Changes

Follow these guidelines:

- **Code Style**: Follow Go conventions and run `make lint`
- **Testing**: Add tests for new functionality
- **Documentation**: Update relevant docs

### 4. Test Locally

```bash
# Run all tests
make test

# Run specific package tests
go test ./sdk/validate/...

# Run with coverage
go test -cover ./...

# Build and try CLI
make build
./bin/dp lint examples/kafka-s3-pipeline
```

### 5. Submit PR

```bash
# Push branch
git push origin feature/your-feature-name

# Create PR via GitHub
```

## 📝 Code Guidelines

### Go Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` (automatic via golangci-lint)
- Exported functions must have doc comments
- Error messages should be lowercase, no punctuation

```go
// Good
func (v *Validator) Validate(ctx context.Context) error {
    if pkg == nil {
        return fmt.Errorf("package cannot be nil")
    }
    // ...
}

// Bad
func (v *Validator) validate(ctx context.Context) error {
    if pkg == nil {
        return fmt.Errorf("Package cannot be nil.") // wrong
    }
}
```

### Testing

- Use table-driven tests where appropriate
- Mock external dependencies
- Test error conditions, not just happy paths

```go
func TestValidator_Validate(t *testing.T) {
    tests := []struct {
        name    string
        input   *DataPackage
        wantErr bool
    }{
        {
            name:    "valid package",
            input:   validPackage(),
            wantErr: false,
        },
        {
            name:    "nil package",
            input:   nil,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            v := NewValidator(tt.input)
            err := v.Validate(context.Background())
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Commit Messages

Follow conventional commits:

```
feat: add PII validation to lint command
fix: resolve binding reference validation
docs: update CLI reference documentation
test: add integration tests for runner
refactor: simplify validation context
```

## 🔀 Pull Request Process

1. **Title**: Use conventional commit format
2. **Description**: Explain what and why
3. **Tests**: Ensure all tests pass
4. **Lint**: No linting errors
5. **Docs**: Update if needed

### PR Checklist

- [ ] Tests added/updated
- [ ] Documentation updated
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Commit messages follow convention

## 🐛 Reporting Bugs

Include:

1. **Version**: `dp version`
2. **Environment**: OS, Go version, Docker version
3. **Steps to reproduce**
4. **Expected behavior**
5. **Actual behavior**
6. **Logs/error messages**

## 💡 Feature Requests

For new features:

1. Check if it aligns with the [Constitution](.specify/memory/constitution.md)
2. Create a discussion first
3. Wait for team feedback before implementation

## 🏛️ Constitution

All contributions must align with the [DP Constitution](.specify/memory/constitution.md), particularly:

- **Article I**: Developer Experience first
- **Article II**: Maintain stable contracts
- **Article V**: Security by default
- **Article VII**: Quality gates enforced

## 📞 Getting Help

- **Discussions**: GitHub Discussions for questions
- **Issues**: GitHub Issues for bugs
- **Slack**: #data-platform channel (internal)

## 📄 License

By contributing, you agree that your contributions will be licensed under the Apache 2.0 License.

---

Thank you for contributing! 🎉
