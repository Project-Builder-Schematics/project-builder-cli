#!/usr/bin/env bash
# FF-20: release-anti-loop-guard
# Enforces: cli-versioning-automation REQ-CVA-022, REQ-CVA-024
#
# Asserts that .github/workflows/release.yml:
#   1. Has a job-level `if:` condition on the `bump` job that contains BOTH:
#      - The `!=` operator
#      - The string `github-actions[bot]`
#   2. Does NOT have any step-level `if:` that uses `github.event.head_commit.message`
#      as the sole or primary skip condition (REQ-CVA-024)
#
# The job-level check ensures the authoritative anti-loop guard is in place.
# Checking for `!=` (not just the bot name) ensures it's a "not equal" guard,
# not an accidental equality check that would invert the logic.
#
# Note: yq is NOT used — grep-based for runner portability.
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/release-anti-loop-guard.sh
set -euo pipefail

RELEASE_YML=".github/workflows/release.yml"

fail() {
  echo "FF-20 release-anti-loop-guard: $*" >&2
  exit 1
}

[ -f "$RELEASE_YML" ] || fail "$RELEASE_YML not found"

# 1. The bump job must exist
grep -q 'bump:' "$RELEASE_YML" \
  || fail "no 'bump:' job found in $RELEASE_YML"

# 2. There must be a job-level `if:` that contains `!=` AND `github-actions[bot]`
# A job-level `if:` is indented at 4 or 6 spaces (under jobs: > bump:),
# NOT indented deeper (step-level if: is indented at 8+ spaces under steps:)
#
# Strategy: find the `if:` line(s) that contain both `!=` and `github-actions[bot]`
# at job-level indentation (2-6 leading spaces, NOT 8+)

found_job_level_guard=0
while IFS= read -r line; do
  # Must contain both `!=` and `github-actions[bot]`
  if printf '%s' "$line" | grep -qF 'github-actions[bot]'; then
    if printf '%s' "$line" | grep -qF '!='; then
      # Check it's an `if:` line at job level (not deeply nested steps)
      if printf '%s' "$line" | grep -qE '^[[:space:]]{2,6}if:'; then
        found_job_level_guard=1
        break
      fi
    fi
  fi
done < "$RELEASE_YML"

if [ "$found_job_level_guard" -eq 0 ]; then
  fail "no job-level 'if: ... != ... github-actions[bot]' guard found in $RELEASE_YML — required for anti-loop (REQ-CVA-022). Must use != operator at job level, not step level."
fi

# 3. No step-level if: using head_commit.message as sole guard (REQ-CVA-024)
if grep -qE '^\s{8,}if:.*head_commit\.message' "$RELEASE_YML"; then
  fail "step-level 'if:' using 'head_commit.message' found in $RELEASE_YML — [skip-bump] is decorative only; authoritative guard must be job-level github.actor check (REQ-CVA-024)"
fi

echo "FF-20 release-anti-loop-guard: OK"
