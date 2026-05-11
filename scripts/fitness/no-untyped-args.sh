#!/usr/bin/env bash
# FF-08: no-untyped-args (V2 NEW)
# Enforces: fitness-functions-ci.REQ-08
#
# Negative-greps for bare `[]string` in internal/shared/engine/*.go files.
# Lines marked with `// fitness:allow-untyped-args env-allowlist` are excluded
# (these are intentional, specifically typed fields like EnvAllowlist).
#
# In real mode: scans internal/shared/engine/*.go
# In fixture mode (single positional arg): scans the given file
#
# Exits non-zero if any unmarked []string occurrence is found.
#
# Usage:
#   bash scripts/fitness/no-untyped-args.sh                  # real codebase
#   bash scripts/fitness/no-untyped-args.sh <fixture.go.txt> # fixture mode
set -euo pipefail

fail=0

if [ $# -ge 1 ]; then
  mapfile -t files < <(echo "$1")
else
  # Exclude test files: the rule targets the production API surface only.
  # Test functions may legitimately use []string for test data/assertions.
  mapfile -t files < <(find internal/shared/engine -name '*.go' ! -name '*_test.go' 2>/dev/null)
fi

for f in "${files[@]}"; do
  [ -f "$f" ] || continue

  # Find lines matching \[\]string that do NOT contain the exemption marker
  while IFS= read -r match; do
    [ -n "$match" ] || continue
    echo "FF-08 no-untyped-args: $f — untyped []string: $match" >&2
    fail=1
  done < <(grep -n '\[\]string' "$f" 2>/dev/null | grep -v '// fitness:allow-untyped-args env-allowlist' || true)
done

exit "$fail"
