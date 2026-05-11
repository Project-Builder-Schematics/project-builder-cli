#!/usr/bin/env bash
# FF-01: handler-loc
# Enforces: inert-stub-contract.REQ-02.1, fitness-functions-ci.REQ-01.1
#
# Counts non-blank, non-comment SLOC in handler.go files.
# In real mode: scans internal/feature/*/handler.go (direct children only).
# In fixture mode (single positional arg): scans the given file path.
# Exits non-zero if any file exceeds 100 SLOC.
#
# Usage:
#   bash scripts/fitness/handler-loc.sh                  # real codebase
#   bash scripts/fitness/handler-loc.sh <fixture.go.txt> # fixture mode
set -euo pipefail

MAX_LOC=100
fail=0

if [ $# -ge 1 ]; then
  # Fixture mode: single file path provided
  files=("$1")
else
  # Real mode: glob direct-child handler.go files under internal/feature/*/
  # Using find to avoid glob expansion issues when no files match
  mapfile -t files < <(find internal/feature -maxdepth 2 -name 'handler.go' 2>/dev/null)
fi

for f in "${files[@]}"; do
  [ -f "$f" ] || continue
  loc=$(awk '
    /^[[:space:]]*$/ { next }
    /^[[:space:]]*\/\// { next }
    { n++ }
    END { print n+0 }
  ' "$f")
  if [ "$loc" -gt "$MAX_LOC" ]; then
    echo "FF-01 handler-loc: $f has $loc SLOC (max $MAX_LOC)" >&2
    fail=1
  fi
done

exit "$fail"
