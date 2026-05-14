#!/usr/bin/env bash
# FF-10: no-direct-os-io  (spec cross-ref: FF-init-02)
# Enforces: builder-init-end-to-end REQ-FW-01 (FSWriter port discipline)
#
# All filesystem mutation in internal/feature/init/ MUST route through the
# FSWriter port. Direct calls to os.WriteFile|MkdirAll|Create|Remove|RemoveAll
# are forbidden in production code EXCEPT fswriter.go (which implements the
# port itself). Test files are exempt.
#
# In real mode: scans internal/feature/init/ for direct os.* mutation calls
#   outside fswriter.go.
# In fixture mode (single positional arg): scans the given file path.
# Exits non-zero on any violation.
#
# Usage:
#   bash scripts/fitness/no-direct-os-io.sh                  # real codebase
#   bash scripts/fitness/no-direct-os-io.sh <fixture.go.txt> # fixture mode
set -euo pipefail

PATTERN='\bos\.(WriteFile|MkdirAll|Create|Remove|RemoveAll)\b'
fail=0

if [ $# -ge 1 ]; then
  # Fixture mode: scan the given file path; any match fails.
  fixture="$1"
  if grep -nE "$PATTERN" "$fixture" 2>/dev/null; then
    echo "FF-10 no-direct-os-io: fixture $fixture contains a direct os.* mutation call" >&2
    exit 1
  fi
  exit 0
fi

# Real mode: scan internal/feature/init/ excluding fswriter.go and _test.go files.
# The fswriter.go file is the port implementation — it is allowed and expected
# to call os.* directly.
mapfile -t files < <(
  find internal/feature/init -maxdepth 2 -name '*.go' \
    -not -name 'fswriter.go' \
    -not -name '*_test.go' \
    2>/dev/null
)

for f in "${files[@]}"; do
  [ -f "$f" ] || continue
  if matches=$(grep -nE "$PATTERN" "$f" 2>/dev/null); then
    while IFS= read -r line; do
      echo "FF-10 no-direct-os-io: $f — direct os.* call: $line" >&2
    done <<< "$matches"
    fail=1
  fi
done

exit "$fail"
