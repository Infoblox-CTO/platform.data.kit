# Demos

Scripted, reproducible terminal demos for the DataKit CLI. Each demo is a plain-text **dialog file** executed by a shared runner script, producing clean, deterministic output suitable for presentations and recordings.

## Available Demos

| Demo | Description | Prerequisites | Infrastructure |
|------|-------------|---------------|----------------|
| [quickstart](quickstart/) | `dk init` → `dk lint` → `dk show` | `dk` binary | No |
| [dev-lifecycle](dev-lifecycle/) | `dk dev up` → `status` → `down` | `dk` binary, k3d, `DK_E2E_DEV=1` | Yes |

## Running a Demo

```bash
# Build the dk binary first
make build

# Run any demo from the repository root
./demos/run_demo.sh demos/<demo-name>/demo.txt

# Example: run the quickstart demo
./demos/run_demo.sh demos/quickstart/demo.txt
```

## Dialog File Format Reference

Dialog files are plain-text files (`.txt`) with one directive per line. The runner reads each line and dispatches by prefix.

### Directives

#### `SAY: <text>`

Print narration text in cyan bold. A blank line is added before each narration block for readability.

```
SAY: Let's create a new data package
```

#### `CMD: <command>`

Execute a shell command. The command is shown with a green `$ ` prompt, then executed via `eval`. Supports pipes, `&&`, and environment variable expansion. If the command fails, the runner stops and reports the step number and exit code.

```
CMD: dk init my-project
CMD: cd my-project && dk lint
CMD: dk show --json | jq '.name'
```

#### `WAIT: <seconds>`

Pause execution for the specified duration. Supports decimals. Defaults to 1 second if omitted or invalid.

```
WAIT: 2
WAIT: 0.5
```

#### `REQUIRE: <prerequisite>`

Declare a prerequisite. All `REQUIRE:` lines are collected and checked before any commands run.

- **Commands** (lowercase/mixed): checked via `command -v` (e.g., `k3d`, `docker`)
- **Environment variables** (uppercase + underscores + digits): checked via `[ -n "$VAR" ]` (e.g., `DK_E2E_DEV`)

```
REQUIRE: k3d
REQUIRE: DK_E2E_DEV
```

#### `# <comment>`

Comments and blank lines are ignored by the runner.

```
# This is a comment
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All commands succeeded |
| 1 | A `CMD:` directive failed |
| 2 | A `REQUIRE:` prerequisite was not met |

## Creating a New Demo

Follow these steps to add a new demo to the library:

### 1. Create the directory structure

```bash
mkdir -p demos/<demo-name>/recordings
touch demos/<demo-name>/recordings/.gitkeep
```

### 2. Write the dialog file

Create `demos/<demo-name>/demo.txt`:

```
# Demo: <Title>
# Shows: <brief description>
# Prerequisites: <what's needed>

REQUIRE: dk    # if the demo uses dk commands

SAY: Welcome to the <demo-name> demo

SAY: Step 1 description
CMD: <command>

WAIT: 1

SAY: Step 2 description
CMD: <another-command>

SAY: Done!
```

### 3. Write the README

Create `demos/<demo-name>/README.md` with:

- Demo title and description
- What it demonstrates
- Prerequisites
- How to run (`./demos/run_demo.sh demos/<demo-name>/demo.txt`)
- Expected output
- Recording instructions (asciinema)

### 4. Add a test

Add a test function to `tests/e2e/demo_test.go`:

```go
func TestDemo_<DemoName>(t *testing.T) {
    skipIfShort(t)

    // Add environment variable gates if infrastructure is required:
    // if os.Getenv("DK_E2E_DEV") == "" {
    //     t.Skip("set DK_E2E_DEV=1 to enable this test")
    // }

    result := runDemo(t, "<demo-name>")

    if result.ExitCode != 0 {
        t.Fatalf("<demo-name> demo failed with exit code %d\nstdout:\n%s\nstderr:\n%s",
            result.ExitCode, result.Stdout, result.Stderr)
    }
}
```

### 5. Record (optional)

```bash
asciinema rec demos/<demo-name>/recordings/<demo-name>.cast \
  -c "./demos/run_demo.sh demos/<demo-name>/demo.txt" \
  --overwrite \
  -i 2
```

### 6. Update this README

Add the new demo to the **Available Demos** table above.

## Best Practices for Stable Output

Demos are executed in CI tests, so output must be **deterministic and stable**:

- **Use temp directories**: Avoid polluting the workspace with demo artifacts
  ```
  CMD: export DEMO_DIR=$(mktemp -d) && echo "Working in $DEMO_DIR"
  CMD: cd "$DEMO_DIR" && dk init my-project
  ```

- **Filter verbose output**: Use `| head`, `| tail`, or `| grep` to limit output
  ```
  CMD: dk show --json | jq '.metadata.name'
  ```

- **Clean up**: Remove temp files at the end of the demo
  ```
  CMD: rm -rf "$DEMO_DIR"
  ```

- **Avoid non-deterministic output**: Timestamps, random IDs, and varying line counts will cause flaky tests

- **Use absolute paths in CMD**: Since `eval` runs in the runner's shell, `cd` in one `CMD:` persists to the next. Use `$DEMO_DIR` or explicit paths to avoid confusion.

## Recording Workflow

### Prerequisites

Install asciinema: `brew install asciinema` (macOS) or see [asciinema.org](https://asciinema.org)

### Recommended Terminal Size

Set your terminal to **120 columns × 30 rows** for consistent recordings:
- macOS Terminal: Terminal → Window → Columns: 120, Rows: 30
- iTerm2: Session → Edit Session → Columns: 120, Rows: 30

### Record

```bash
asciinema rec demos/<demo-name>/recordings/<demo-name>.cast \
  -c "./demos/run_demo.sh demos/<demo-name>/demo.txt" \
  --overwrite \
  -i 2
```

### Playback

```bash
asciinema play demos/<demo-name>/recordings/<demo-name>.cast -i 1 -s 1.5
```

### Upload (optional)

```bash
asciinema upload demos/<demo-name>/recordings/<demo-name>.cast
```
