# Dev Lifecycle Demo

A scripted walkthrough of the local development lifecycle: **dev up → status → down**.

## What It Shows

1. Starting the local development stack with `dk dev up`
2. Checking service status with `dk dev status`
3. Tearing down the stack with `dk dev down`

## Prerequisites

- **dk binary**: Build with `make build` from the repository root
- **k3d cluster**: A running k3d cluster named `dk-local`
- **DK_E2E_DEV=1**: Set this environment variable to enable infrastructure demos

## How to Run

```bash
# From the repository root
make build
export DK_E2E_DEV=1
./demos/run_demo.sh demos/dev-lifecycle/demo.txt
```

## Expected Output

The demo starts the local development stack (Helm charts for Redpanda, LocalStack, Postgres, Marquez), checks status, and tears it down. You should see:

- Narration text in **cyan** explaining each step
- Commands prefixed with a green `$` prompt
- Helm chart deployment output
- Exit code 0 on success

## Recording with asciinema

To record this demo as a `.cast` file for sharing or embedding:

```bash
# Set your terminal to 120×30 for consistent recordings
# macOS: Terminal → Window → Columns: 120, Rows: 30

# Record the demo (requires running k3d cluster)
export DK_E2E_DEV=1
asciinema rec demos/dev-lifecycle/recordings/dev-lifecycle.cast \
  -c "./demos/run_demo.sh demos/dev-lifecycle/demo.txt" \
  --overwrite \
  -i 2

# Play back the recording
asciinema play demos/dev-lifecycle/recordings/dev-lifecycle.cast -i 1 -s 1.5
```

**Prerequisites for recording**: Install asciinema with `brew install asciinema` (macOS) or see [asciinema.org](https://asciinema.org).
