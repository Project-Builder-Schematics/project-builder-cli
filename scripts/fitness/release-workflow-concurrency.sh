#!/usr/bin/env bash
# FF-17: release-workflow-concurrency
# Enforces: cli-versioning-automation REQ-CVA-026
#
# Asserts that .github/workflows/release.yml has a concurrency: block at
# workflow level with:
#   1. A non-empty `group:` value
#   2. `cancel-in-progress: false` (NOT true — partial tags would result)
#
# Note: yq is NOT used — grep/sed only for runner portability.
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/release-workflow-concurrency.sh
set -euo pipefail

RELEASE_YML=".github/workflows/release.yml"

fail() {
  echo "FF-17 release-workflow-concurrency: $*" >&2
  exit 1
}

[ -f "$RELEASE_YML" ] || fail "$RELEASE_YML not found"

# 1. concurrency: block must exist
grep -q '^concurrency:' "$RELEASE_YML" \
  || fail "no top-level 'concurrency:' block found in $RELEASE_YML (REQ-CVA-026)"

# 2. group: must be present and non-empty
# Matches `  group: <something>` under the concurrency block
grep -qE '^\s+group:\s*\S' "$RELEASE_YML" \
  || fail "'group:' field missing or empty in concurrency block of $RELEASE_YML"

# 3. cancel-in-progress: false MUST be set (not true)
grep -qE '^\s+cancel-in-progress:\s*false' "$RELEASE_YML" \
  || fail "'cancel-in-progress: false' not found in $RELEASE_YML — must be false, not true (partial tags would result)"

# 4. Explicitly reject cancel-in-progress: true
if grep -qE '^\s+cancel-in-progress:\s*true' "$RELEASE_YML"; then
  fail "'cancel-in-progress: true' found in $RELEASE_YML — MUST be false (REQ-CVA-026)"
fi

echo "FF-17 release-workflow-concurrency: OK"
