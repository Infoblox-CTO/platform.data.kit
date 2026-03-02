# Feature Specification: Rename CLI from `dp` to `dk` (DataKit) & Add Interactive Banner

**Feature Branch**: `016-rename-cli-dk`  
**Created**: 2026-03-01  
**Status**: Draft  
**Input**: User description: "Rename the dp CLI to dk. The dk name will be more intuitive since the name means datakit. Also add a SLICK banner when doing interactive prompting."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - CLI Rename from `dp` to `dk` (Priority: P1)

A developer who previously used the `dp` command now types `dk` to access all the same functionality. The command name `dk` stands for "DataKit," which is more intuitive and better reflects the product identity. Every subcommand, flag, and workflow that previously worked under `dp` works identically under `dk`.

**Why this priority**: This is the core deliverable. Without the rename, no other changes in this feature are meaningful. All existing users need `dk` to be a seamless drop-in replacement for `dp`.

**Independent Test**: Can be fully tested by building the CLI, running `dk version`, and verifying all subcommands (`dk init`, `dk dev`, `dk run`, `dk lint`, `dk build`, `dk publish`, `dk promote`, `dk show`, `dk logs`, `dk config`, `dk asset`, `dk pipeline`, `dk cell`) work identically to their `dp` counterparts.

**Acceptance Scenarios**:

1. **Given** a developer has `dk` installed, **When** they run `dk version`, **Then** the output shows "dk version X.Y.Z" with the correct version string.
2. **Given** a developer has `dk` installed, **When** they run `dk init my-pipeline --runtime cloudquery`, **Then** a new data package is scaffolded in the `my-pipeline` directory, identical to the previous `dp init` behavior.
3. **Given** a developer has `dk` installed, **When** they run `dk --help`, **Then** the help output references "dk" (not "dp") throughout, including the command name, description, and examples.
4. **Given** a developer runs any existing `dp` workflow end-to-end (`dk init` → `dk dev up` → `dk lint` → `dk run` → `dk build` → `dk publish` → `dk promote`), **When** they complete the workflow, **Then** every step succeeds without errors.

---

### User Story 2 - SLICK ASCII Banner in Interactive Prompts (Priority: P2)

When a developer begins an interactive session (e.g., `dk init` without all required flags, triggering interactive prompts), a visually appealing ASCII art banner is displayed at the top of the terminal. The banner reinforces the "DataKit" brand identity and gives the CLI a polished, professional feel. The banner is only shown during interactive prompting sessions, not during non-interactive or scripted usage.

**Why this priority**: The banner adds a polished, memorable user experience but does not affect core functionality. It depends on the rename (P1) being completed first since the banner will display the "dk" / "DataKit" branding.

**Independent Test**: Can be tested by running `dk init` without arguments in an interactive terminal and verifying that the ASCII banner appears before the first interactive prompt. Then run `dk init my-project --runtime cloudquery` (non-interactive) and verify no banner appears.

**Acceptance Scenarios**:

1. **Given** a developer runs `dk init` without providing all required arguments in an interactive terminal, **When** the CLI enters interactive prompting mode, **Then** a styled ASCII art banner displaying "DataKit" branding is shown before the first prompt.
2. **Given** a developer runs a command non-interactively (all arguments provided or piped input), **When** the command executes, **Then** no banner is displayed.
3. **Given** a developer runs a command with output redirected to a file or pipe, **When** the command executes, **Then** no banner is displayed (the banner detects non-TTY output).

---

### User Story 3 - Update Documentation and Build Artifacts (Priority: P3)

All project documentation, build scripts, demos, and references are updated to reflect the `dk` name. Developers reading docs, READMEs, or demo scripts see `dk` commands instead of `dp`. Build outputs produce `dk` binaries instead of `dp` binaries.

**Why this priority**: Documentation consistency is important for onboarding and trust, but the CLI works correctly even if docs lag slightly behind. This story ensures completeness of the rename across the project.

**Independent Test**: Can be tested by searching the entire repository for references to `dp` as a CLI command and verifying that all user-facing references have been updated to `dk`. Build artifacts in `bin/` should be named `dk` instead of `dp`.

**Acceptance Scenarios**:

1. **Given** a developer reads any README, demo script, or documentation page, **When** they look for CLI command examples, **Then** all examples reference `dk` (not `dp`).
2. **Given** a developer runs `make build` (or equivalent), **When** the build completes, **Then** the output binary is named `dk` (e.g., `bin/dk`, `bin/dk-linux-amd64`).
3. **Given** a developer runs `make install`, **When** the installation completes, **Then** the CLI is installed as `dk` in the Go bin path.

---

### Edge Cases

- What happens when a user types `dp` after the rename? They get "command not found." No shim or alias is provided. The project is pre-production — clean breaks are expected.
- What happens when the banner is displayed in a very narrow terminal (< 40 columns)? The banner should gracefully degrade — either display a simplified version or skip the banner entirely.
- What happens when the CLI is run inside a CI/CD pipeline (non-TTY)? The banner must not be shown to avoid polluting CI logs.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The CLI binary MUST be renamed from `dp` to `dk` across all build targets (local build, cross-compilation, install).
- **FR-002**: The root command name MUST change from `dp` to `dk` in the Cobra command definition (`Use: "dk"`).
- **FR-003**: All help text, long descriptions, and usage examples within the CLI MUST reference `dk` instead of `dp`.
- **FR-004**: The version command output MUST display "dk version X.Y.Z" instead of "dp version X.Y.Z".
- **FR-005**: The CLI MUST display a styled ASCII art banner when entering interactive prompting mode (when running in a TTY and the command requires user input via prompts).
- **FR-006**: The banner MUST NOT be displayed when the CLI runs non-interactively (all arguments provided, piped input, output redirected, or non-TTY environment).
- **FR-007**: The banner MUST display "DataKit" branding that visually reinforces the product identity.
- **FR-008**: The Go package comment in `main.go` MUST be updated to reference `dk` instead of `dp`.
- **FR-009**: The Makefile build targets MUST produce binaries named `dk` (e.g., `bin/dk`, `bin/dk-linux-amd64`, `bin/dk-darwin-arm64`).
- **FR-010**: All project documentation (READMEs, demo scripts, quickstart guides, docs site content) MUST be updated to reference `dk` commands.
- **FR-011**: The banner MUST gracefully handle narrow terminals (< 40 columns width) by either displaying a simplified version or omitting it.

### Key Entities

- **CLI Binary**: The compiled executable, renamed from `dp` to `dk`. Distributed as cross-platform binaries for Linux (amd64/arm64) and macOS (amd64/arm64).
- **ASCII Banner**: A styled text-art element displayed during interactive CLI sessions. Contains "DataKit" branding. Only rendered to TTY outputs.
- **Interactive Prompting Mode**: A CLI state where the tool solicits input from the user via terminal prompts (e.g., during `dk init` when required arguments are missing).

## Assumptions

- No backward compatibility is provided. The project is pre-production and all effort goes toward consistency with currently proposed concepts.
- The ASCII banner design will use standard ASCII characters (no Unicode box-drawing) to ensure compatibility across all terminal emulators.
- The banner color/style will use ANSI escape codes only when the terminal supports them (detected at runtime).
- Internal Go package paths and module names do not need to change as part of this feature (only the binary name and user-facing text).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of CLI commands that previously worked under `dp` work identically under `dk` with no behavior changes.
- **SC-002**: Zero references to `dp` as an executable name remain in user-facing help text, documentation, or build output after implementation.
- **SC-003**: The ASCII banner is displayed within 100ms of entering interactive mode, adding no perceivable delay to the user experience.
- **SC-004**: The banner is correctly suppressed in 100% of non-interactive scenarios (piped input, redirected output, CI/CD environments).
- **SC-005**: All existing CLI tests pass after the rename with updated command references.
- **SC-006**: New developers can complete the quickstart workflow using `dk` commands from the updated documentation on their first attempt.
