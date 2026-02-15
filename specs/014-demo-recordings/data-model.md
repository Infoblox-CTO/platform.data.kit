# Data Model: End-to-End Demo Recordings

**Feature**: 014-demo-recordings | **Date**: 2026-02-15

## Entities

### DialogFile

A plain-text file (`demo.txt`) containing a sequence of directives that define what a demo shows. The runner reads this file line by line and executes each directive.

| Field | Type | Description |
|-------|------|-------------|
| Path | string | Absolute or relative path to the dialog file (e.g., `demos/quickstart/demo.txt`) |
| Directives | []Directive | Ordered list of parsed directives |
| Requirements | []string | Extracted from `REQUIRE:` lines; checked before any `CMD:` executes |

### Directive

A single instruction within a dialog file. Discriminated by type (line prefix).

| Type | Prefix | Payload | Behavior |
|------|--------|---------|----------|
| Say | `SAY:` | Text string | Print narration text in cyan bold |
| Command | `CMD:` | Shell command string | Print `$ cmd`, execute via `eval`, show output |
| Wait | `WAIT:` | Float seconds (default 1) | Sleep for the specified duration |
| Require | `REQUIRE:` | Prerequisite name | Check: command existence or env var set |
| Comment | `#` | Anything | Ignored by runner |
| Blank | *(empty)* | Nothing | Ignored by runner |

### DemoDirectory

A named subdirectory under `demos/` containing all artifacts for a single demo.

| Field | Type | Description |
|-------|------|-------------|
| Name | string | Directory name (e.g., `quickstart`, `dev-lifecycle`) |
| DialogFile | string | Path to `demo.txt` within the directory |
| ReadmePath | string | Path to `README.md` describing the demo |
| RecordingsDir | string | Path to `recordings/` subdirectory for `.cast` files |
| Assets | []string | Optional additional files (config templates, input data) |
| Category | enum | `cli-only` (no infra) or `infrastructure` (requires k3d/docker) |

### DemoTest

A Go test function in `tests/e2e/` that executes a demo's dialog file through the runner and verifies success.

| Field | Type | Description |
|-------|------|-------------|
| FunctionName | string | Go test function name (e.g., `TestDemo_Quickstart`) |
| DemoDir | string | Path to the demo directory being tested |
| RequiresInfra | bool | Whether the demo needs `DP_E2E_DEV=1` |
| SkipConditions | []string | Environment variables or tools that gate execution |

### Recording

An asciinema `.cast` file capturing the terminal output of a demo run.

| Field | Type | Description |
|-------|------|-------------|
| Path | string | Path to `.cast` file (e.g., `demos/quickstart/recordings/demo.cast`) |
| DemoDir | string | Reference to the parent demo directory |
| Format | string | `asciicast-v2` (NDJSON) |
| TerminalSize | struct | `{Width: 120, Height: 30}` recommended |

### RunnerScript

The shared bash script that interprets dialog files.

| Field | Type | Description |
|-------|------|-------------|
| Path | string | `demos/run_demo.sh` |
| EntryPoint | string | `main()` function |
| ExitBehavior | string | Exit with non-zero code on first command failure |
| ColorSupport | bool | Uses tput for colored narration output |

## Relationships

```
demos/
├── run_demo.sh                    # RunnerScript (shared, 1 instance)
│
├── quickstart/                    # DemoDirectory (category: cli-only)
│   ├── demo.txt                   #   └── DialogFile
│   │   ├── REQUIRE: (none)        #       └── Directive[]: SAY, CMD, WAIT
│   │   ├── SAY: ...               #
│   │   ├── CMD: dp init ...       #
│   │   └── CMD: dp lint ...       #
│   ├── README.md                  #
│   └── recordings/                #
│       └── demo.cast              #   └── Recording
│
└── dev-lifecycle/                 # DemoDirectory (category: infrastructure)
    ├── demo.txt                   #   └── DialogFile
    │   ├── REQUIRE: k3d           #       └── Directive[]: REQUIRE, SAY, CMD, WAIT
    │   ├── REQUIRE: DP_E2E_DEV    #
    │   ├── CMD: dp dev up         #
    │   └── CMD: dp dev down       #
    ├── README.md                  #
    └── recordings/                #
        └── demo.cast              #   └── Recording

tests/e2e/
├── demo_test.go                   # DemoTest functions
│   ├── TestDemo_Quickstart()      #   → runs demos/quickstart/demo.txt
│   └── TestDemo_DevLifecycle()    #   → runs demos/dev-lifecycle/demo.txt (gated)
└── helpers.go                     # Existing helpers + runDemo() addition
```

## State Transitions

### Runner Execution States

```
┌─────────┐     parse REQUIRE:    ┌──────────────┐     all ok    ┌───────────┐
│  START   │ ──────────────────── │  CHECKING    │ ────────────── │ EXECUTING │
└─────────┘                       │  PREREQS     │                └─────┬─────┘
                                  └──────┬───────┘                      │
                                         │ missing                      │ for each directive
                                         ▼                              ▼
                                  ┌──────────────┐              ┌──────────────┐
                                  │  ERROR:      │              │  DIRECTIVE:  │
                                  │  prereq      │              │  SAY/CMD/    │
                                  │  missing     │              │  WAIT        │
                                  └──────────────┘              └──────┬───────┘
                                                                       │
                                                          CMD fail?    │ CMD success?
                                                            │          │
                                                            ▼          ▼
                                                     ┌───────────┐  ┌──────────┐
                                                     │  ERROR:   │  │  DONE    │
                                                     │  step N   │  │  (exit 0)│
                                                     │  failed   │  └──────────┘
                                                     └───────────┘
```

## Validation Rules

1. **Dialog file**: Must contain at least one `CMD:` directive (a demo with only narration is not useful)
2. **REQUIRE: values**: Must be either a command name (checked via `command -v`) or an environment variable name (checked via `[ -n "${!var}" ]`)
3. **WAIT: values**: Must be a positive number (integer or float); invalid values default to 1 second
4. **CMD: values**: Must not be empty after trimming whitespace
5. **Demo directory**: Must contain `demo.txt` and `README.md` at minimum
6. **Test function naming**: Must follow `TestDemo_<DemoName>` convention
