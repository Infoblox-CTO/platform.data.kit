#!/usr/bin/env bash
# run_demo.sh — Execute a scripted demo dialog file
#
# Usage: ./demos/run_demo.sh <path-to-dialog-file>
#
# Dialog file directives:
#   SAY: <text>           — Print narration in cyan bold
#   CMD: <command>        — Execute a command, show output, stop on failure
#   WAIT: <seconds>       — Pause (supports decimals, default 1)
#   REQUIRE: <prereq>     — Declare a prerequisite (command or env var)
#   # <comment>           — Ignored
#   (blank lines)         — Ignored
#
# Exit codes:
#   0 — All commands succeeded
#   1 — A CMD: directive failed
#   2 — A REQUIRE: prerequisite was not met

set -euo pipefail

# ---------------------------------------------------------------------------
# Usage
# ---------------------------------------------------------------------------
if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <path-to-dialog-file>"
  echo ""
  echo "Execute a scripted demo from a plain-text dialog file."
  echo ""
  echo "Directives:"
  echo "  SAY: <text>         Print narration text"
  echo "  CMD: <command>      Execute a shell command"
  echo "  WAIT: <seconds>     Pause execution"
  echo "  REQUIRE: <prereq>   Declare a prerequisite"
  echo "  # <comment>         Comment (ignored)"
  exit 1
fi

DIALOG_FILE="$1"

if [[ ! -f "$DIALOG_FILE" ]]; then
  echo "ERROR: Dialog file not found: $DIALOG_FILE"
  exit 1
fi

# ---------------------------------------------------------------------------
# Terminal color helpers (degrade gracefully when tput unavailable)
# ---------------------------------------------------------------------------
if command -v tput >/dev/null 2>&1 && tput colors >/dev/null 2>&1; then
  CYAN=$(tput setaf 6)
  GREEN=$(tput setaf 2)
  RED=$(tput setaf 1)
  BOLD=$(tput bold)
  RESET=$(tput sgr0)
else
  CYAN=""
  GREEN=""
  RED=""
  BOLD=""
  RESET=""
fi

# ---------------------------------------------------------------------------
# REQUIRE: — First pass: collect and check prerequisites
# ---------------------------------------------------------------------------
check_prerequisites() {
  local dialog_file="$1"
  local missing=()

  while IFS= read -r line || [[ -n "$line" ]]; do
    # Strip leading/trailing whitespace
    local trimmed
    trimmed="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

    if [[ "$trimmed" =~ ^REQUIRE:[[:space:]]+(.*) ]]; then
      local prereq="${BASH_REMATCH[1]}"
      # Strip trailing whitespace from prereq
      prereq="$(echo "$prereq" | sed 's/[[:space:]]*$//')"

      if [[ "$prereq" =~ ^[A-Z_][A-Z0-9_]*$ ]]; then
        # Environment variable check
        if [[ -z "${!prereq:-}" ]]; then
          missing+=("environment variable $prereq")
        fi
      else
        # Command check
        if ! command -v "$prereq" >/dev/null 2>&1; then
          missing+=("command '$prereq'")
        fi
      fi
    fi
  done < "$dialog_file"

  if [[ ${#missing[@]} -gt 0 ]]; then
    echo "${RED}${BOLD}ERROR: Missing prerequisites:${RESET}"
    for m in "${missing[@]}"; do
      echo "  - $m"
    done
    exit 2
  fi
}

# ---------------------------------------------------------------------------
# SAY: — Print narration text in cyan bold
# ---------------------------------------------------------------------------
handle_say() {
  local text="$1"
  echo ""
  echo "${CYAN}${BOLD}${text}${RESET}"
}

# ---------------------------------------------------------------------------
# CMD: — Execute a command, display output, stop on failure
# ---------------------------------------------------------------------------
STEP=0

handle_cmd() {
  local command="$1"
  STEP=$((STEP + 1))

  echo "${GREEN}\$ ${RESET}${command}"

  set +e
  eval "$command"
  local exit_code=$?
  set -e

  if [[ $exit_code -ne 0 ]]; then
    echo ""
    echo "${RED}${BOLD}ERROR: Step ${STEP} failed (exit code ${exit_code}): ${command}${RESET}"
    exit 1
  fi
}

# ---------------------------------------------------------------------------
# WAIT: — Pause for specified seconds (default 1, supports decimals)
# ---------------------------------------------------------------------------
handle_wait() {
  local seconds="$1"

  # Default to 1 if empty or invalid
  if [[ -z "$seconds" ]] || ! [[ "$seconds" =~ ^[0-9]*\.?[0-9]+$ ]]; then
    seconds=1
  fi

  sleep "$seconds"
}

# ---------------------------------------------------------------------------
# Main loop — second pass: execute directives
# ---------------------------------------------------------------------------
check_prerequisites "$DIALOG_FILE"

while IFS= read -r line || [[ -n "$line" ]]; do
  # Strip leading/trailing whitespace
  trimmed="$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"

  # Skip blank lines
  [[ -z "$trimmed" ]] && continue

  # Skip comments
  [[ "$trimmed" =~ ^# ]] && continue

  # Skip REQUIRE: lines (already processed)
  [[ "$trimmed" =~ ^REQUIRE: ]] && continue

  # Dispatch directives
  if [[ "$trimmed" =~ ^SAY:[[:space:]]+(.*) ]]; then
    handle_say "${BASH_REMATCH[1]}"
  elif [[ "$trimmed" =~ ^CMD:[[:space:]]+(.*) ]]; then
    handle_cmd "${BASH_REMATCH[1]}"
  elif [[ "$trimmed" =~ ^WAIT:[[:space:]]*(.*) ]]; then
    handle_wait "${BASH_REMATCH[1]}"
  elif [[ "$trimmed" =~ ^WAIT:$ ]]; then
    handle_wait ""
  else
    echo "${RED}WARNING: Unknown directive: ${trimmed}${RESET}"
  fi
done < "$DIALOG_FILE"
