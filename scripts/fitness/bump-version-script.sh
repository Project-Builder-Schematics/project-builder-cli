#!/usr/bin/env bash
# FF-23: bump-version-script
# Enforces: cli-versioning-automation REQ-CVA-034, REQ-CVA-040, REQ-CVA-041
#
# Asserts:
#   1. scripts/release/bump-version.sh exists and is executable
#   2. scripts/release/bump-version.test.sh exists and is executable
#   3. The test driver passes (exits 0) — meaning all ≥8 test cases pass
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/bump-version-script.sh
set -euo pipefail

SCRIPT="scripts/release/bump-version.sh"
TEST_DRIVER="scripts/release/bump-version.test.sh"

fail() {
  echo "FF-23 bump-version-script: $*" >&2
  exit 1
}

# 1. bump-version.sh must exist and be executable
[ -f "$SCRIPT" ] \
  || fail "$SCRIPT not found (REQ-CVA-034)"
[ -x "$SCRIPT" ] \
  || fail "$SCRIPT exists but is not executable (REQ-CVA-034: chmod +x required)"

# 2. bump-version.test.sh must exist and be executable
[ -f "$TEST_DRIVER" ] \
  || fail "$TEST_DRIVER not found (REQ-CVA-040)"
[ -x "$TEST_DRIVER" ] \
  || fail "$TEST_DRIVER exists but is not executable (REQ-CVA-040: chmod +x required)"

# 3. Run the test driver — all cases must pass
echo "FF-23: running $TEST_DRIVER ..."
if ! sh "$TEST_DRIVER"; then
  fail "$TEST_DRIVER exited non-zero — one or more bump-version test cases failed (REQ-CVA-041)"
fi

echo "FF-23 bump-version-script: OK"
