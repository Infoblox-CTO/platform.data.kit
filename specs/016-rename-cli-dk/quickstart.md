# Quickstart: Verifying the `dk` CLI Rename

**Feature**: 016-rename-cli-dk | **Date**: 2026-03-01

## Prerequisites

- Go (latest stable)
- Docker
- k3d (for local dev environment)
- Make

## Step 1: Build the CLI

```bash
make build-cli
```

**Verify**: Binary exists at `bin/dk`:

```bash
ls -la bin/dk
bin/dk version
# Expected: dk version dev
```

## Step 2: Verify Help Output

```bash
bin/dk --help
```

**Verify**: Output shows "DK (DataKit)" branding, all example commands use `dk`, and no references to `dp` appear.

## Step 3: Test Interactive Banner

```bash
# Interactive mode — should show banner
bin/dk init

# Non-interactive mode — should NOT show banner
bin/dk init my-pipeline --runtime cloudquery
```

**Verify**: In interactive mode, an ASCII art "DataKit" banner appears before the first prompt. In non-interactive mode, no banner is shown.

## Step 4: Verify TTY Suppression

```bash
# Piped output — should NOT show banner
echo "" | bin/dk init

# Redirected output — banner should not appear in file
bin/dk init 2>&1 | grep -c "DataKit"
# Expected: 0 (banner goes to stderr or is suppressed)
```

## Step 5: End-to-End Workflow

```bash
# Create a new project
bin/dk init test-project --runtime cloudquery

# Validate
cd test-project && ../bin/dk lint

# Show manifest
../bin/dk show

# Clean up
cd .. && rm -rf test-project
```

**Verify**: All commands complete successfully using `dk`.

## Step 6: Verify Config Paths

```bash
bin/dk config list
```

**Verify**: Config paths reference `.dk/config.yaml` and `~/.config/dk/config.yaml`.

## Step 7: Run Tests

```bash
# Run all unit tests
cd cli && go test ./... -count=1
cd ../sdk && go test ./... -count=1
cd ../contracts && go test ./... -count=1
cd ../platform/controller && go test ./... -count=1
```

**Verify**: All tests pass.

## Step 8: Verify Manifest Filename

```bash
bin/dk init verify-manifest --runtime cloudquery
ls verify-manifest/dk.yaml
# Expected: file exists
rm -rf verify-manifest
```

## Step 9: Verify No `dp` Remnants

```bash
# Search for lingering dp references in user-facing text
grep -rn '"dp ' cli/cmd/ | grep -v '_test.go' | grep -v 'dp.yaml'
# Expected: no matches

bin/dk --help 2>&1 | grep -i '\bdp\b'
# Expected: no matches
```

## Step 10: Install Globally

```bash
make install
dk version
# Expected: dk version dev
```
