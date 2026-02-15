# Quickstart Demo

A scripted walkthrough of the core `dp` CLI workflow: **init → lint → show → dev up → build → run → dev down**.

## What It Shows

1. Creating a new data package with `dp init`
2. Exploring the generated project structure and source code
3. Validating the configuration with `dp lint`
4. Inspecting package metadata with `dp show`
5. Starting the local development environment with `dp dev up`
6. Building the package into an OCI artifact with `dp build`
7. Running the pipeline against the dev environment with `dp run`
8. Tearing down the dev environment with `dp dev down`

## Prerequisites

- **dp binary**: Build with `make build` from the repository root
- **Docker**: Required for `dp build` and `dp run`
- **k3d**: Required for the local development environment (`dp dev up`)

## How to Run

```bash
# From the repository root
make build
./demos/run_demo.sh demos/quickstart/demo.txt
```

## Expected Output

The demo creates a temporary `quickstart-demo` data package, validates it, brings up the local dev environment, builds and runs the pipeline, then tears everything down. You should see:

- Narration text in **cyan** explaining each step
- Commands prefixed with a green `$` prompt
- Dev environment startup output from `dp dev up`
- Validation, build, and run output from `dp` commands
- Pipeline output: `Hello from quickstart-demo pipeline!`
- Dev environment teardown from `dp dev down`
- Exit code 0 on success

## Recording with asciinema

To record this demo as a `.cast` file for sharing or embedding:

```bash
# Set your terminal to 120×30 for consistent recordings
# macOS: Terminal → Window → Columns: 120, Rows: 30

# Record the demo
asciinema rec demos/quickstart/recordings/quickstart.cast \
  -c "./demos/run_demo.sh demos/quickstart/demo.txt" \
  --overwrite \
  -i 2

# Play back the recording
asciinema play demos/quickstart/recordings/quickstart.cast -i 1 -s 1.5
```

**Prerequisites for recording**: Install asciinema with `brew install asciinema` (macOS) or see [asciinema.org](https://asciinema.org).
