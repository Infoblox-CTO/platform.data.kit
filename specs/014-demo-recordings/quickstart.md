# Quickstart: End-to-End Demo Recordings

**Feature**: 014-demo-recordings

## Running an Existing Demo

### Prerequisites

```bash
# Build the dp CLI
make build

# Verify it's available
./bin/dp version
```

### Run a demo (no recording)

```bash
# Run the quickstart demo (CLI-only, no infrastructure needed)
./demos/run_demo.sh demos/quickstart/demo.txt

# Output:
#
# Welcome to the Data Platform quickstart demo
#
# First, let's create a new data package
# $ dp init quickstart-demo
# ✓ Created data package: quickstart-demo
#
# Now let's look at the generated structure
# $ ls quickstart-demo/
# dp.yaml
#
# Let's validate the package configuration
# $ cd quickstart-demo && dp lint
# ✓ Linting passed
#
# That's it! You've created, validated, and inspected a data package.
```

### Run an infrastructure demo

```bash
# Infrastructure demos require a running k3d cluster
export DP_E2E_DEV=1

# Run the dev lifecycle demo
./demos/run_demo.sh demos/dev-lifecycle/demo.txt
```

## Recording a Demo

### Install asciinema (one-time)

```bash
brew install asciinema
```

### Record

```bash
# Set terminal dimensions for consistency
stty rows 30 cols 120

# Record the quickstart demo
asciinema rec demos/quickstart/recordings/demo.cast \
  -c "./demos/run_demo.sh demos/quickstart/demo.txt" \
  --overwrite -i 2

# Play it back to verify
asciinema play demos/quickstart/recordings/demo.cast -i 1 -s 1.5
```

## Testing Demos

### Run all demo tests

```bash
# CLI-only demos (no infrastructure needed)
go test ./tests/e2e/... -run TestDemo -v

# Including infrastructure demos (requires k3d cluster)
DP_E2E_DEV=1 go test ./tests/e2e/... -run TestDemo -v
```

### Verify a specific demo

```bash
go test ./tests/e2e/... -run TestDemo_Quickstart -v
```

## Authoring a New Demo

### 1. Create the demo directory

```bash
mkdir -p demos/my-demo/recordings
```

### 2. Write the dialog file

Create `demos/my-demo/demo.txt`:

```
# Demo: My Feature Demo
# Shows: what this demo demonstrates

SAY: Welcome to the My Feature demo

SAY: Step 1 — Create a project
CMD: dp init my-project

WAIT: 1

SAY: Step 2 — Validate the project
CMD: cd my-project && dp lint

SAY: Done!
```

### 3. Write the README

Create `demos/my-demo/README.md`:

```markdown
# My Feature Demo

Demonstrates the my-feature workflow.

## Prerequisites
- `dp` binary (`make build`)

## Run
./demos/run_demo.sh demos/my-demo/demo.txt
```

### 4. Add a test

Add a test function to `tests/e2e/demo_test.go`:

```go
func TestDemo_MyFeature(t *testing.T) {
    skipIfShort(t)
    runDemo(t, "my-demo")
}
```

### 5. Verify

```bash
# Run the demo manually
./demos/run_demo.sh demos/my-demo/demo.txt

# Run the test
go test ./tests/e2e/... -run TestDemo_MyFeature -v
```

## Dialog File Reference

| Directive | Syntax | Purpose |
|-----------|--------|---------|
| `SAY:` | `SAY: text` | Print narration (cyan bold) |
| `CMD:` | `CMD: command` | Execute and display a command |
| `WAIT:` | `WAIT: 1.5` | Pause for N seconds |
| `REQUIRE:` | `REQUIRE: k3d` | Check prerequisite at startup |
| `#` | `# comment` | Comment (ignored) |

## Tips for Stable Demos

- Filter non-deterministic output: `CMD: dp show --json | jq '.name'`
- Limit long output: `CMD: kubectl get pods | head -5`
- Use environment variables for parameterization: `CMD: dp init $PROJECT_NAME`
- Keep demos short (under 2 minutes playback with `-i 1`)
- Test commands individually before adding to dialog files
