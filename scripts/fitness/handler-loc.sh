#!/usr/bin/env bash
# FF-01/FF-16: handler-loc
# Enforces: inert-stub-contract.REQ-02.1, fitness-functions-ci.REQ-01.1
#
# Counts non-blank, non-comment SLOC in handler.go and handler_*.go files.
# In real mode: scans all handler files under internal/feature/ (depth 2).
# In fixture mode (single positional arg): scans the given file path.
# Exits non-zero if any file exceeds 100 SLOC (HARD gate).
# Emits a WARN line (but exits 0) if any file exceeds 80 SLOC (SOFT gate).
#
# FF-16 (S-000b): extended to cover feature/new/handler_schematic.go and
# feature/new/handler_collection.go in addition to feature/init/handler.go.
#
# Usage:
#   bash scripts/fitness/handler-loc.sh                  # real codebase
#   bash scripts/fitness/handler-loc.sh <fixture.go.txt> # fixture mode
set -euo pipefail

HARD_MAX=100
SOFT_MAX=80
fail=0

if [ $# -ge 1 ]; then
  # Fixture mode: single file path provided
  files=("$1")
else
  # Real mode: scan handler.go and handler_*.go files under internal/feature/
  # Depth 2 catches both feature/init/handler.go and feature/new/handler_*.go
  mapfile -t files < <(find internal/feature -maxdepth 2 \( -name 'handler.go' -o -name 'handler_*.go' \) 2>/dev/null | grep -v '_test\.go')
fi

for f in "${files[@]}"; do
  [ -f "$f" ] || continue
  loc=$(awk '
    /^[[:space:]]*$/ { next }
    /^[[:space:]]*\/\// { next }
    { n++ }
    END { print n+0 }
  ' "$f")
  if [ "$loc" -gt "$HARD_MAX" ]; then
    echo "FF-16 handler-loc HARD FAIL: $f has $loc SLOC (max $HARD_MAX)" >&2
    fail=1
  elif [ "$loc" -gt "$SOFT_MAX" ]; then
    echo "FF-16 handler-loc SOFT WARN: $f has $loc SLOC (soft limit $SOFT_MAX)" >&2
  fi
done

exit "$fail"
