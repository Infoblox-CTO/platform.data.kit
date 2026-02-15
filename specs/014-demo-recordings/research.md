# Research: End-to-End Demo Recordings

**Feature**: 014-demo-recordings | **Date**: 2026-02-15

## 1. Dialog File Format Design

### Existing Conventions Evaluated

| Tool | Format | Approach | Typing Simulation |
|------|--------|----------|-------------------|
| demo-magic (1.9k ★) | Bash script (sourced lib) | Function calls (`pe`, `pei`, `wait`) | Yes |
| asciinema-scenario (61 ★) | `.scenario` text file | Line prefixes (`$`, `#`, `--`) | Yes (char-by-char) |
| asciinema-rec_script (31 ★) | `.asc` bash script | Comments as narration, code lines | No |
| terminalizer (16.1k ★) | `.yml` config | YAML keys | No (records live) |

**Decision**: Use a custom prefix-based plain-text format inspired by `asciinema-scenario`.

**Rationale**: A simple line-prefix format is the most readable, learnable, and grep-friendly approach. It separates content (what to show) from execution (the runner). Unlike demo-magic (which requires bash scripting knowledge) or terminalizer (which records live sessions), a prefix-based dialog file can be authored and reviewed by anyone.

**Alternatives considered**:
- **Bash script with sourced library (demo-magic style)**: Rejected — requires bash knowledge, mixes content with code, harder to validate
- **YAML-based format**: Rejected — verbose for sequential commands, harder to read at a glance
- **Markdown-based format**: Rejected — ambiguous parsing of fenced code blocks, heavier parser needed

### Chosen Directive Set

| Directive | Syntax | Behavior |
|-----------|--------|----------|
| `SAY:` | `SAY: text to display` | Print narration text in color (cyan bold) |
| `CMD:` | `CMD: dp init my-project` | Print `$ command`, execute, show output |
| `WAIT:` | `WAIT: 1.5` | Pause for N seconds (default 1) |
| `REQUIRE:` | `REQUIRE: k3d` or `REQUIRE: DP_E2E_DEV` | Check prerequisite before any commands |
| `#` | `# comment text` | Ignored by runner (file comments) |
| *(blank)* | *(empty line)* | Ignored by runner |

**Design decisions**:
- Directives use uppercase prefix + colon for unambiguous parsing
- No `INCLUDE:` or `EXPECT:` directives in v1 — keep minimal, extend later
- Environment variables in `CMD:` are expanded by the shell (`eval`)
- No typing simulation — real execution is more authentic than fake typing

## 2. Runner Implementation Language

### Options Evaluated

| Language | Pros | Cons |
|----------|------|------|
| Bash | Universal, no dependencies, natural shell integration | Weaker error handling, limited text processing |
| Python | Better error handling, richer text formatting | Extra dependency, overkill for line-by-line processing |
| Go | Same language as project, testable | Heavyweight for a simple script, harder to debug interactively |

**Decision**: Bash script (`run_demo.sh`).

**Rationale**: All target platforms (macOS, Linux, CI) have bash. The runner's job is simple: read lines, parse prefixes, execute commands. Bash does this natively without any dependencies. The spec (A-001) assumes bash. Error handling is straightforward with manual `$?` checking.

**Alternatives considered**:
- **Python**: Rejected — adds a runtime dependency for a task bash handles natively. The runner has no complex text processing that would benefit from Python.
- **Go binary**: Rejected — over-engineered. A shell script is debuggable with `-x`, editable without compilation, and fits the Unix philosophy of composable tools.

## 3. asciinema Integration

### Key Findings

- **Installation**: `brew install asciinema` (macOS), package managers on Linux
- **Recording**: `asciinema rec output.cast -c "./run_demo.sh demos/quickstart/demo.txt"`
- **Playback flags**: `-i <secs>` (idle clamp), `-s <factor>` (speed multiplier)
- **File format**: asciicast v2 — NDJSON (newline-delimited JSON), first line is metadata header, remaining lines are `[time, type, data]` event tuples
- **Optional dependency**: The runner works without asciinema; recording is a separate manual step (A-002)

**Decision**: asciinema is an optional tool for producing `.cast` recordings. The runner does not depend on it.

**Rationale**: Keeping asciinema optional means:
1. Demo tests run without it (FR-008 compliance)
2. CI doesn't need asciinema installed
3. Developers can run demos without recording
4. The `.cast` format is an output artifact, not a dependency

**Recording workflow**:
```bash
# Terminal size for consistency
stty rows 30 cols 120

# Record
asciinema rec demos/quickstart/recordings/demo.cast \
  -c "./demos/run_demo.sh demos/quickstart/demo.txt" \
  --overwrite -i 2

# Playback (verify)
asciinema play demos/quickstart/recordings/demo.cast -i 1 -s 1.5
```

## 4. Go Test Integration for Shell Script Execution

### Approach

Use `os/exec.Command("bash", scriptPath, dialogFilePath)` from Go tests, following the existing E2E test patterns in `tests/e2e/helpers.go`.

**Decision**: Add a `runDemo(t, demoDir)` helper to `tests/e2e/` that executes the runner script against a demo's dialog file and captures output.

**Rationale**: This mirrors the existing `runDP()` / `runDPInDir()` pattern. The Go test:
1. Locates the runner script relative to repo root
2. Locates the demo directory
3. Executes `bash run_demo.sh <demo-dir>/demo.txt`
4. Asserts exit code 0
5. Optionally checks stdout for expected narration text

**Key patterns from existing E2E tests**:
- `dpBinaryPath(t)` pattern for locating files relative to repo root → reuse for `demoRunnerPath(t)`
- `CommandResult{Stdout, Stderr, ExitCode}` → reuse directly
- `skipIfShort(t)` → apply to all demo tests
- Environment gating (`DP_E2E_DEV`) → apply to infrastructure demos via `REQUIRE:` directive mapping

### Test Discovery

**Decision**: Explicit test functions per demo (not auto-discovery).

**Rationale**: Go's test framework doesn't support directory scanning for test generation at compile time. Each demo gets a named test function (`TestDemo_Quickstart`, `TestDemo_DevLifecycle`) that maps to its dialog file. This is explicit, grep-friendly, and matches the existing E2E test style.

**Alternative considered**: Table-driven test with `os.ReadDir("demos/")` — rejected because it makes test failures harder to isolate and doesn't align with the existing per-feature test functions in `tests/e2e/`.

## 5. Demo Directory Structure

### Options Evaluated

| Layout | Description |
|--------|-------------|
| `demos/` at repo root | Top-level, visible, matches `docs/`, `tests/`, `examples/` |
| `tests/e2e/demos/` | Nested under tests, emphasizes test-backing |
| `docs/demos/` | Under documentation, emphasizes content over testing |

**Decision**: `demos/` at repository root.

**Rationale**: Demos are first-class artifacts (Article I — DX is product). They are not just tests, and not just documentation — they are executable content that bridges both. A top-level `demos/` directory:
1. Is immediately discoverable by new contributors
2. Parallels existing top-level directories (`docs/`, `tests/`, `examples/`)
3. Can be referenced from documentation without deep relative paths
4. Keeps the runner script (`run_demo.sh`) accessible

### Per-Demo Layout

```
demos/
├── run_demo.sh               # Shared runner script
├── README.md                  # Demo index + authoring guide (FR-014)
├── quickstart/                # CLI-only demo (FR-011)
│   ├── demo.txt               # Dialog file
│   ├── README.md              # What this demo shows, prerequisites
│   └── recordings/            # .cast files (gitignored or committed)
└── dev-lifecycle/             # Infrastructure demo (FR-012)
    ├── demo.txt
    ├── README.md
    └── recordings/
```

**Design decisions**:
- Runner script is shared (one `run_demo.sh` for all demos)
- Each demo is self-contained in its own directory
- `recordings/` subdirectory keeps `.cast` files organized
- No shared state between demos — each is independently runnable
