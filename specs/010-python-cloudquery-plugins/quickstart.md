# Quickstart: Python CloudQuery Plugin

**Feature**: 010-python-cloudquery-plugins

## Prerequisites

- `dp` CLI installed and on PATH
- Docker running (Rancher Desktop or Docker Desktop)
- k3d dev environment running: `dp dev up`
- `cloudquery` CLI installed
- `kubectl` and `k3d` installed

## Steps

### 1. Scaffold a Python plugin

```bash
dp init -t cloudquery -l python foo
cd foo
```

**Expected**: Directory `foo/` created with `dp.yaml`, `main.py`, `requirements.txt`, `pyproject.toml`, `plugin/`, `tests/`.

### 2. Inspect the project

```bash
cat dp.yaml
```

**Expected**: `spec.type: cloudquery`, `spec.cloudquery.language: python`, `spec.cloudquery.role: source`.

### 3. Build and run the plugin

```bash
dp run
```

**Expected**: Plugin image built with Python 3.11 (matching distroless runtime), imported into k3d, deployed as pod, gRPC port-forwarded, tables discovered:

```
Building CloudQuery plugin image: default/foo:latest (lang=python)
...
Discovered 1 table(s):

  example_resource
    An example resource table. Replace with your actual data source.
    Columns:
      id      utf8
      name    utf8
      value   int64
      active  bool

✓ CloudQuery plugin is working correctly
```

### 4. Sync to local files

```bash
dp run --sync
```

**Expected**: Sync completes, JSON output in `./cq-sync-output/`:

```
✓ Sync completed: foo → file
  Output directory: ./cq-sync-output/
```

### 5. Verify file output

```bash
ls cq-sync-output/
cat cq-sync-output/*.json
```

**Expected**: JSON file with 2 example resource records.

### 6. Sync to PostgreSQL

```bash
dp run --sync --destination postgresql
```

**Expected**: Sync completes with auto-detected PostgreSQL:

```
Preparing sync: foo → postgresql
  Config: registry=ghcr.io/infobloxopen
  ...
✓ Sync completed: foo → postgresql
```

### 7. Verify PostgreSQL output (optional)

```bash
kubectl exec -it deploy/dp-postgres-postgres -n dp-local -- psql -U postgres -c "SELECT * FROM example_resource;"
```

**Expected**: 2 rows with id, name, value, active columns.

### 8. Run unit tests

```bash
dp test
```

**Expected**: pytest runs, all tests pass.

### 9. Cleanup

```bash
cd ..
rm -rf foo
```

## Validation Criteria

- [ ] Step 1: `dp init -t cloudquery -l python foo` creates all expected files
- [ ] Step 3: `dp run` builds, deploys, discovers tables with zero errors
- [ ] Step 4: `dp run --sync` writes JSON to `./cq-sync-output/`
- [ ] Step 6: `dp run --sync --destination postgresql` syncs to PostgreSQL with zero errors
- [ ] Step 8: `dp test` runs pytest with all tests passing
