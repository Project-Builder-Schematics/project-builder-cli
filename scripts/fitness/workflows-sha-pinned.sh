#!/usr/bin/env bash
# FF-19: workflows-sha-pinned (release.yml only — two-tier policy)
# Enforces: cli-versioning-automation REQ-CVA-027
#
# Asserts that every `uses:` line in .github/workflows/release.yml references
# a 40-character hex commit SHA (not a mutable tag like @v4 or @main).
#
# SCOPE: release.yml ONLY. ci.yml and other workflow files are EXEMPT from this
# check per the two-tier SHA-pinning policy (ADR-030):
#   - Tier 1 (mandatory): release.yml — highest blast radius (auto-commits to main)
#   - Tier 2 (optional):  other workflow files — accept major-tag pins
#
# Exit 0 on success. Exit 1 with diagnostic naming the offending line(s).
#
# Usage:
#   bash scripts/fitness/workflows-sha-pinned.sh
set -euo pipefail

RELEASE_YML=".github/workflows/release.yml"

fail() {
  echo "FF-19 workflows-sha-pinned: $*" >&2
  exit 1
}

[ -f "$RELEASE_YML" ] || fail "$RELEASE_YML not found"

found_bad=0
line_num=0

while IFS= read -r line; do
  line_num=$((line_num + 1))
  # Check if the line contains a `uses:` entry
  if printf '%s' "$line" | grep -qE '^\s+uses:\s+\S'; then
    # Extract the reference part after `uses: `
    ref=$(printf '%s' "$line" | sed -E 's/^\s+uses:\s+//' | sed 's/#.*//' | tr -d '[:space:]')
    # A valid SHA-pinned reference must end with @<40-hex-chars>
    # Extract what comes after the last @
    after_at=$(printf '%s' "$ref" | grep -oE '@[^@]*$' | sed 's/@//')
    if [ -z "$after_at" ]; then
      echo "FF-19 workflows-sha-pinned: $RELEASE_YML line $line_num: 'uses:' has no @ reference: $line" >&2
      found_bad=1
      continue
    fi
    # Must be exactly 40 lowercase hex characters
    if ! printf '%s' "$after_at" | grep -qE '^[0-9a-f]{40}$'; then
      echo "FF-19 workflows-sha-pinned: $RELEASE_YML line $line_num: 'uses:' is not SHA-pinned (got @$after_at): $line" >&2
      found_bad=1
    fi
  fi
done < "$RELEASE_YML"

if [ "$found_bad" -eq 1 ]; then
  exit 1
fi

echo "FF-19 workflows-sha-pinned: OK (release.yml only — ci.yml exempt per two-tier policy)"
