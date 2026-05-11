#!/usr/bin/env bash
# _meta_invert.sh — fitness meta-test helper
#
# Inverts the exit code of the wrapped script: succeeds IFF the wrapped
# script FAILS on the given fixture. Used by `just fitness-meta` to validate
# that each fitness function correctly catches its own bad-pattern fixture.
#
# Usage:
#   scripts/fitness/_meta_invert.sh <script> <fixture> [extra-args...]
#
# Exit codes:
#   0 — the wrapped script failed on the fixture (rule caught violation) ✓
#   1 — the wrapped script PASSED on the fixture (rule missed violation) ✗
set -euo pipefail

if [ $# -lt 2 ]; then
  echo "usage: _meta_invert.sh <script> <fixture> [extra-args...]" >&2
  exit 2
fi

script="$1"
fixture="$2"
shift 2

if "$script" "$fixture" "$@" >/dev/null 2>&1; then
  echo "META FAIL: $script did NOT catch violation in $fixture" >&2
  exit 1
fi

echo "META OK: $script correctly caught violation in $fixture"
exit 0
