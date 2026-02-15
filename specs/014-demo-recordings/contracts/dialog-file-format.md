# Dialog File Format Contract

**Feature**: 014-demo-recordings | **Date**: 2026-02-15

## Dialog File Specification

A dialog file is a plain-text file (conventionally named `demo.txt`) that describes a scripted terminal demo as an ordered sequence of directives. The runner script (`run_demo.sh`) reads a dialog file and executes it deterministically.

### File Format

- **Encoding**: UTF-8
- **Line endings**: LF (Unix-style)
- **Extension**: `.txt` (plain text, no special tooling needed to edit)
- **Convention**: One file per demo, named `demo.txt`

### Directives

Each non-blank, non-comment line is a directive. Directives are identified by an uppercase prefix followed by a colon and a space.

#### `SAY: <text>`

Print narration text to the terminal. The text is displayed in cyan bold (using `tput setaf 6; tput bold`) to distinguish it from command output.

```
SAY: Let's create a new data package
SAY: This will scaffold the project structure
```

**Behavior**:
- Print `<text>` to stdout with color formatting
- Add a blank line before narration for readability
- No command execution

#### `CMD: <command>`

Execute a shell command and display its output. The command is shown with a `$ ` prompt prefix before execution.

```
CMD: dp init my-project
CMD: cd my-project && dp lint
CMD: dp show --json | jq '.name'
```

**Behavior**:
1. Print `$ <command>` to stdout (green prompt, white command)
2. Execute `<command>` via `eval` in the current shell (supports pipes, env vars, `&&`)
3. Display stdout and stderr
4. If exit code ≠ 0: print error message with step number, exit runner with same code

**Environment variable expansion**: Variables like `$PROJECT_NAME` are expanded by `eval`. Set them before running the demo or use `export` in a prior `CMD:`.

#### `WAIT: <seconds>`

Pause execution for the specified duration. Used to give viewers time to read narration or observe output.

```
WAIT: 2
WAIT: 0.5
WAIT: 1.5
```

**Behavior**:
- Sleep for `<seconds>` (supports decimals)
- Default: 1 second if no value provided
- Invalid values: treated as 1 second

#### `REQUIRE: <prerequisite>`

Declare a prerequisite that must be satisfied before any commands execute. All `REQUIRE:` directives are collected and checked at startup.

```
REQUIRE: k3d
REQUIRE: docker
REQUIRE: DP_E2E_DEV
```

**Two forms of prerequisites**:
1. **Command**: checked via `command -v <name>` (e.g., `k3d`, `docker`, `asciinema`)
2. **Environment variable**: checked via `[ -n "${<name>}" ]` (e.g., `DP_E2E_DEV`)

**Detection heuristic**: If the name matches `^[A-Z_]+$` (all uppercase with underscores), treat as environment variable. Otherwise treat as command.

**Behavior**:
- Collected from anywhere in the file (typically at the top)
- Checked before any `SAY:` or `CMD:` executes
- If any prerequisite is missing: print all missing prerequisites and exit with code 2

#### `# <comment>`

A comment line. Ignored entirely by the runner. Used for documentation within the dialog file.

```
# This demo shows the basic dp workflow
# Author: team-dp
# Last updated: 2026-02-15
```

#### Blank lines

Blank lines (empty or whitespace-only) are ignored by the runner.

## Runner Script Contract

### Interface

```bash
./demos/run_demo.sh <path-to-dialog-file>
```

**Arguments**:
- `$1`: Path to the dialog file (required)

**Exit codes**:
- `0`: All commands succeeded
- `1`: A `CMD:` directive failed (runner prints step number and failing command)
- `2`: A `REQUIRE:` prerequisite was not met

### Output Format

```
# (blank line)
# SAY text in cyan bold
Let's create a new data package

$ dp init my-project                    # CMD prompt in green
✓ Created data package: my-project      # Command stdout (unmodified)

$ dp lint                               # Next CMD
✓ Linting passed                        # Command stdout

ERROR: Step 5 failed (exit code 1): dp build    # On failure
```

### Environment

- The runner inherits the calling shell's environment
- Commands execute in the runner's working directory (not the dialog file's directory)
- The `dp` binary must be on `$PATH` or referenced by absolute path

## Example Dialog File

```
# Demo: Data Package Quickstart
# Shows: dp init → dp lint → dp show
# Prerequisites: dp binary (make build)

SAY: Welcome to the Data Platform quickstart demo

SAY: First, let's create a new data package
CMD: dp init quickstart-demo

SAY: Now let's look at the generated structure
CMD: ls quickstart-demo/

WAIT: 1

SAY: Let's validate the package configuration
CMD: cd quickstart-demo && dp lint

SAY: Finally, let's inspect the package metadata
CMD: cd quickstart-demo && dp show

SAY: That's it! You've created, validated, and inspected a data package.
```

## Compatibility

- **Version**: 1.0 (initial format)
- **Evolution strategy**: New directives are additive (existing dialog files remain valid)
- **Planned extensions** (future, not in v1): `INCLUDE: <file>` (embed another dialog), `EXPECT: <substring>` (assert output content)
