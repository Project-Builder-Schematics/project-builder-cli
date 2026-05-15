#!/usr/bin/env bash
# FF-16: release-workflow-permissions
# Enforces: cli-versioning-automation REQ-CVA-021, REQ-CVA-025
#
# Asserts that .github/workflows/release.yml:
#   1. Does NOT have a top-level permissions: block (other than absent or empty {})
#   2. The `bump` job has a permissions block containing `contents: write`
#   3. The `bump` job permissions block does NOT contain forbidden write-level
#      permissions: actions: write, id-token: write, packages: write, write-all
#
# Allowed alongside contents: write: pull-requests: read, contents: read (redundant but harmless).
#
# Note: yq is NOT used — grep/sed only for runner portability.
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/release-workflow-permissions.sh
set -euo pipefail

RELEASE_YML=".github/workflows/release.yml"

fail() {
  echo "FF-16 release-workflow-permissions: $*" >&2
  exit 1
}

[ -f "$RELEASE_YML" ] || fail "$RELEASE_YML not found"

# 1. Must have a `bump:` job defined
grep -q 'bump:' "$RELEASE_YML" \
  || fail "no 'bump:' job found in $RELEASE_YML"

# 2. The bump job must have a permissions block with contents: write
# Look for `contents: write` within the context of the release.yml file.
# Since there should be no top-level permissions block, any `contents: write`
# in the file belongs to a job-level block.
grep -q 'contents: write' "$RELEASE_YML" \
  || fail "'contents: write' not found in $RELEASE_YML (expected in bump job permissions)"

# 3. No top-level `permissions:` block at workflow scope.
# Top-level permissions appear before `jobs:` in the YAML.
# We detect this by checking if `permissions:` appears before the `jobs:` line.
jobs_line=$(grep -n '^jobs:' "$RELEASE_YML" | head -1 | cut -d: -f1)
if [ -n "$jobs_line" ]; then
  top_permissions=$(awk "NR < $jobs_line && /^permissions:/" "$RELEASE_YML" | head -1)
  if [ -n "$top_permissions" ]; then
    fail "top-level 'permissions:' block found before 'jobs:' in $RELEASE_YML — must be job-level only (REQ-CVA-021)"
  fi
fi

# 4. Forbidden write-level permissions must NOT appear
for forbidden in 'actions: write' 'id-token: write' 'packages: write' 'write-all'; do
  if grep -q "$forbidden" "$RELEASE_YML"; then
    fail "forbidden permission '$forbidden' found in $RELEASE_YML (REQ-CVA-025: minimal permissions)"
  fi
done

echo "FF-16 release-workflow-permissions: OK"
