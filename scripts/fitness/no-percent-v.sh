#!/usr/bin/env bash
# FF-09: no-percent-v (V2 NEW)
# Enforces: fitness-functions-ci.REQ-09
#
# Negative-greps for `Message:` assignments that use `%v` or `%+v` in
# Sprintf-style calls within feature handler files. This catches patterns like:
#   Message: fmt.Sprintf("failed: %v", err)
# which leak the wrapped Cause into the user-facing message string.
#
# In real mode: scans internal/feature/*/handler.go files (direct children)
#               and internal/feature/*/*/handler.go (one level deeper, e.g. skill/update)
# In fixture mode (single positional arg): scans the given file
#
# Exits non-zero if any %v or %+v appears in a Message: context.
#
# Usage:
#   bash scripts/fitness/no-percent-v.sh                  # real codebase
#   bash scripts/fitness/no-percent-v.sh <fixture.go.txt> # fixture mode
set -euo pipefail

fail=0

if [ $# -ge 1 ]; then
  mapfile -t files < <(echo "$1")
else
  mapfile -t files < <(find internal/feature -name 'handler.go' 2>/dev/null)
fi

for f in "${files[@]}"; do
  [ -f "$f" ] || continue

  # Match lines with Message: that contain %v or %+v
  while IFS= read -r match; do
    [ -n "$match" ] || continue
    echo "FF-09 no-percent-v: $f — Message uses %%v interpolation: $match" >&2
    fail=1
  done < <(grep -n 'Message:.*%[v+]' "$f" 2>/dev/null || true)
done

exit "$fail"
