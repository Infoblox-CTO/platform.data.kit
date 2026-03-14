---
name: dk-new-transform
description: Scaffold a new dk transform package inside an existing project
user_invocable: true
---

# dk new-transform — Scaffold a Transform

Create a new Transform package inside an existing dk project.

## Inputs

Ask the user for any values not already provided:

- **name** — DNS-safe transform name (lowercase, alphanumeric, hyphens, 3-63 chars)
- **runtime** — one of: `cloudquery`, `generic-go`, `generic-python`, `dbt`
- **mode** — `batch` (default) or `streaming` (note: dbt does not support streaming)
- **namespace** — team namespace (default: `default`)
- **team** — team label

## Steps

1. **Scaffold the transform** inside the project's `transforms/` directory:
   ```bash
   cd <project>/transforms
   dk init <name> --runtime <runtime> --mode <mode> --namespace <namespace> --team <team>
   ```

2. **Lint the new package** to confirm it's valid out of the box:
   ```bash
   dk lint transforms/<name>
   ```

3. **Show the updated pipeline graph:**
   ```bash
   dk pipeline show --scan-dir .
   ```

4. **Show the user the generated `dk.yaml`** and explain what to customise next:
   - `spec.inputs` / `spec.outputs` — wire up real DataSet names
   - `spec.trigger` — set scheduling (cron) or event-driven triggers
   - `spec.image` — set container image (for generic-go/generic-python/dbt runtimes)
   - Dataset manifests in `dataset/` — update schemas to match real data

## Notes

- For `generic-go` runtimes, `dk init` also runs `go mod tidy` and `go fmt`.
- If a parent `go.work` file exists, the scaffold isolates itself with `GOWORK=off`.
- The scaffold creates sample connectors and stores — these are starting points, not production config.
