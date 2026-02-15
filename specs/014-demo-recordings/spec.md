# Feature Specification: End-to-End Demo Recordings

**Feature Branch**: `014-demo-recordings`
**Created**: 2026-02-15
**Status**: Draft
**Input**: User description: "End-to-end demo recording. Use scripted dialog files with a runner, asciinema for recording, and test-based verification. Demos should be organizable so multiple different demos can be recorded independently."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Scripted, Reproducible Demo Execution (Priority: P1)

A developer or CI job runs a demo script that deterministically executes a sequence of CLI commands with narration text, producing clean, readable terminal output. The demo is defined in a plain-text dialog file (`demo.txt`) that separates **what to show** (narration + commands) from **how to run it** (the runner engine). This means anyone can author a new demo by writing a simple text file — no code changes to the runner are needed.

The runner reads the dialog file line by line, prints narration text, displays a shell prompt followed by each command, executes it, shows the output, and optionally pauses between steps. If any command fails, the runner stops immediately with a clear error message identifying the failed step.

**Why this priority**: Without a deterministic runner and dialog format, demos are ad-hoc live recordings that drift, break, and can't be maintained. This is the foundational layer everything else builds on.

**Independent Test**: Author a minimal dialog file with 3 SAY/CMD steps using basic shell commands (`echo`, `ls`). Run the demo runner against it. Verify narration prints, commands execute with output shown, and a deliberate failure (e.g., `false`) stops the runner with an error.

**Acceptance Scenarios**:

1. **Given** a dialog file with `SAY:`, `CMD:`, and `WAIT:` directives, **When** the runner executes it, **Then** narration text is printed, commands are executed with their output shown, and pauses are observed between steps.
2. **Given** a dialog file where one command returns a non-zero exit code, **When** the runner executes it, **Then** the runner stops immediately, prints the failing step number and command, and exits with a non-zero code.
3. **Given** a dialog file with a `WAIT: 1.5` directive, **When** the runner executes it, **Then** the runner pauses for approximately 1.5 seconds before continuing to the next step.
4. **Given** a dialog file with comment lines (starting with `#`), **When** the runner executes it, **Then** comment lines are ignored and do not appear in output.
5. **Given** a dialog file references an environment variable placeholder, **When** the runner executes it with that variable set, **Then** the variable is expanded in the command.

---

### User Story 2 - Test-Backed Demo Verification (Priority: P1)

Each demo is backed by a Go test that runs the demo's dialog file through the runner and verifies that every command succeeds. This means demos are validated as part of the normal test suite — a broken demo is caught the same way a broken feature is caught. The test infrastructure reuses the existing E2E test helpers (`runDP`, `CommandResult`, etc.) and follows the same patterns (environment-variable gating, `testing.Short()` skip).

Demos that require infrastructure (like a k3d cluster) are gated by environment variables (e.g., `DP_E2E_DEV=1`), while demos that only exercise CLI help/init/lint commands run without special prerequisites.

**Why this priority**: Demo recordings that aren't tested will silently break as the CLI evolves. Test-backed verification is co-equal with the runner — without it, demos rot.

**Independent Test**: Run `go test ./tests/e2e/... -run TestDemo` and verify that all demo dialog files execute successfully. Break a command in a dialog file and verify the test fails.

**Acceptance Scenarios**:

1. **Given** a demo dialog file exists in the demo directory, **When** `go test` runs the corresponding test function, **Then** every command in the dialog is executed and verified to succeed.
2. **Given** a demo requires a k3d cluster (tagged with a `REQUIRE:` directive), **When** `DP_E2E_DEV` is not set, **Then** the test is skipped with a clear message.
3. **Given** a demo uses only basic CLI commands (no infrastructure), **When** tests run without any special environment variables, **Then** the demo test executes normally and passes.
4. **Given** a developer adds a new demo dialog file to the demos directory, **When** they add a corresponding test entry, **Then** it is automatically picked up by `go test`.

---

### User Story 3 - Asciinema Recording of Scripted Demos (Priority: P2)

A developer records a demo as an asciinema `.cast` file by running the demo runner inside `asciinema rec`. The recording captures the clean, scripted output — not a messy live session. Recordings are stored in the repository alongside their dialog files. Playback uses idle-time clamping (`-i 1`) to compress wait times, and optional speed-up (`-s 2`) for snappier viewing.

**Why this priority**: The recording step depends on the runner (US1) being solid. It adds distribution value — shareable demo artifacts — but the runner and test verification are independently useful without recordings.

**Independent Test**: Install asciinema, run a demo through `asciinema rec`, then play back the `.cast` file and verify it contains the expected narration and command output.

**Acceptance Scenarios**:

1. **Given** a demo dialog file and the runner, **When** a developer runs `asciinema rec demo.cast -c "./run_demo.sh demo.txt"`, **Then** a `.cast` file is produced containing the scripted terminal output.
2. **Given** a recorded `.cast` file, **When** played back with `asciinema play demo.cast -i 1`, **Then** idle gaps are compressed to at most 1 second while narration and output remain readable.
3. **Given** the recording directory structure, **When** a new demo is added, **Then** it follows the same layout convention (`demos/<name>/demo.txt`, `demos/<name>/recordings/`).

---

### User Story 4 - Organized Demo Library (Priority: P2)

Demos are organized in a consistent directory structure where each demo is self-contained: a dialog file, optional assets (config files, input data), a README explaining what the demo shows and its prerequisites, and a directory for recordings. New demos are added by creating a new directory — no changes to existing demos or infrastructure required.

Demos are categorized by what they exercise: quick CLI-only demos (no infrastructure), dev environment demos (require k3d), and workflow demos (init → run → build → publish). This makes it easy to find, maintain, and selectively run demos.

**Why this priority**: Organization enables scaling to many demos. Without it, a flat list of dialog files becomes unmaintainable. However, the core runner and testing work without a directory convention.

**Independent Test**: Create two demos in separate directories. Verify each can be run independently. Verify the test runner discovers and validates both.

**Acceptance Scenarios**:

1. **Given** the demos directory structure, **When** a developer lists the contents, **Then** each demo is in its own named subdirectory with a `demo.txt` and `README.md`.
2. **Given** a demo has a `REQUIRE: k3d` directive, **When** the demo index or README is consulted, **Then** the prerequisite is clearly documented.
3. **Given** multiple demos exist, **When** a developer wants to record only one, **Then** they can target it by name (e.g., `./run_demo.sh demos/quickstart/demo.txt`).

---

### User Story 5 - Demo Authoring Guide (Priority: P3)

A contributor guide documents how to author new demos: the dialog file format, available directives, best practices for stable output (filtering with `| head`, `--json | jq`, suppressing progress bars), and how to add test coverage. This enables any team member or AI agent to create new demos without reverse-engineering the system.

**Why this priority**: Documentation enables scaling the demo library beyond the initial author. It's valuable but not blocking — the first demos can be created by the person who builds the system.

**Independent Test**: Follow the guide to create a new demo from scratch. Verify the demo runs, tests pass, and can be recorded.

**Acceptance Scenarios**:

1. **Given** the authoring guide exists, **When** a new contributor reads it, **Then** they can create a working demo (dialog file + test) without additional help.
2. **Given** the guide documents all directives, **When** a developer looks up a directive, **Then** the format, behavior, and example are clearly described.

---

### Edge Cases

- What happens when a demo command produces non-deterministic output (timestamps, UUIDs)? The dialog format supports piping commands through filters (`| head`, `| jq`, `| sed`) to stabilize output. The authoring guide recommends this practice.
- What happens when a demo's prerequisite is not met (e.g., k3d not installed)? The `REQUIRE:` directive causes the runner to check prerequisites before executing any steps and exit with a clear error if unmet. The corresponding test skips.
- What happens when a demo command takes a long time (e.g., `dp dev up`)? The runner executes it normally. During recording, `asciinema play -i 1` compresses idle time. Authors can add `SAY: This may take a moment...` before long steps for viewer context.
- What happens when the dp CLI changes a command's output format? The test-backed verification catches the breakage. The demo dialog file is updated to match the new output.
- What happens when two demos share setup steps (e.g., both need `dp init`)? Each demo is self-contained. Shared setup is repeated in each dialog file rather than using a shared state that creates coupling. Helper dialog snippets could be `INCLUDE:`-ed in a future extension.
- What happens when a demo is recorded on a terminal with different dimensions? The authoring guide recommends a standard terminal size (120×30). The runner can optionally enforce this via `stty` or document the recommendation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide a dialog file format with at least four directives: `SAY:` (print narration), `CMD:` (execute and display a shell command), `WAIT:` (pause for a specified duration), and `#` (comment, ignored).
- **FR-002**: The system MUST provide a runner script that reads a dialog file and executes it step by step, printing narration, displaying `$ <command>` prompts, running commands, and showing their output.
- **FR-003**: The runner MUST stop execution immediately when any command returns a non-zero exit code, reporting the step number, the command that failed, and the exit code.
- **FR-004**: The runner MUST support a `REQUIRE:` directive that checks for prerequisites (e.g., `k3d`, `asciinema`, `DP_E2E_DEV`) before executing any steps, exiting with a clear error if any prerequisite is missing.
- **FR-005**: The runner MUST expand environment variables in `CMD:` directives so that demos can be parameterized (e.g., `CMD: dp init $PROJECT_NAME`).
- **FR-006**: Each demo MUST have a corresponding Go test in `tests/e2e/` that executes the dialog file through the runner and verifies all commands succeed.
- **FR-007**: Demo tests that require infrastructure (k3d cluster, Docker) MUST be gated by environment variables (e.g., `DP_E2E_DEV=1`) and skipped when the variable is not set.
- **FR-008**: Demo tests that exercise only CLI commands (help, init, lint, show) MUST run without special environment variables or infrastructure.
- **FR-009**: Demos MUST be organized in a directory structure where each demo is a named subdirectory containing at minimum a `demo.txt` dialog file and a `README.md`.
- **FR-010**: The system MUST support recording demos via `asciinema rec` by running the runner as asciinema's command, producing a `.cast` file.
- **FR-011**: The system MUST include at least one demo covering the core developer workflow: `dp init` → `dp lint` → `dp show` (a quick, infrastructure-free demo).
- **FR-012**: The system MUST include at least one demo covering the dev environment lifecycle: `dp dev up` → `dp dev status` → `dp dev down` (requires k3d).
- **FR-013**: The runner MUST produce clean output suitable for recording: no shell prompt decorations, no background job noise, consistent formatting.
- **FR-014**: An authoring guide MUST document the dialog file format, all directives, best practices for stable output, and how to add test coverage for a new demo.

### Key Entities

- **Dialog File**: A plain-text file (`demo.txt`) containing a sequence of directives (`SAY:`, `CMD:`, `WAIT:`, `REQUIRE:`, `#`) that define what a demo shows. Separates content from execution logic.
- **Demo Runner**: A script (shell or Python) that interprets a dialog file and executes it deterministically. Responsible for printing narration, running commands, enforcing error handling, and managing pacing.
- **Demo Directory**: A named subdirectory under `demos/` containing a demo's dialog file, README, optional assets, and a `recordings/` subdirectory for `.cast` files.
- **Demo Test**: A Go test function in `tests/e2e/` that executes a demo dialog file through the runner and verifies all commands succeed, providing CI-backed validation that demos remain functional.
- **Recording**: An asciinema `.cast` file capturing the terminal output of a demo run, suitable for playback and sharing.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new demo can be created (dialog file + test) in under 15 minutes by following the authoring guide.
- **SC-002**: All demo tests pass as part of the standard `go test ./tests/e2e/...` run (non-infrastructure demos run without gating; infrastructure demos skip gracefully).
- **SC-003**: At least two demos exist at launch: one CLI-only quickstart and one dev environment lifecycle demo.
- **SC-004**: Demo recordings (`.cast` files) play back in under 2 minutes each with idle-time clamping at 1 second.
- **SC-005**: When a CLI command's output changes, the corresponding demo test fails, catching the drift before the recording becomes stale.
- **SC-006**: A contributor who has never seen the demo system can author a new demo by reading only the authoring guide and existing examples.

## Assumptions

- **A-001**: The demo runner will be a shell script (bash) for simplicity and portability, since all target platforms (macOS, Linux) have bash. Python is an alternative if error handling or text effects require it.
- **A-002**: asciinema is an optional dependency — demos can be run and tested without it. Recording is a separate, manual step.
- **A-003**: The `dp` CLI binary is built via `make build` before demos run, consistent with the existing E2E test pattern.
- **A-004**: Demo tests reuse the existing `tests/e2e/` package and helpers (`runDP`, `CommandResult`) rather than introducing a new test framework.
- **A-005**: Demo dialog files use only commands available in the repository (the `dp` binary, standard Unix tools). No external service dependencies for CLI-only demos.
- **A-006**: Terminal dimensions of 120×30 are recommended for recording consistency but not enforced by the runner.
