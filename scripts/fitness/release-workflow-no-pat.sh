#!/usr/bin/env bash
# FF-18: release-workflow-no-pat
# Enforces: cli-versioning-automation REQ-CVA-028
#
# Asserts that .github/workflows/release.yml references NO secrets other than
# GITHUB_TOKEN. Specifically, no ${{ secrets.* }} reference may have a name
# that includes: PAT, PERSONAL, TOKEN (other than GITHUB_TOKEN), or any
# secret name that is not GITHUB_TOKEN.
#
# Note: yq is NOT used — grep-based for runner portability.
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/release-workflow-no-pat.sh
set -euo pipefail

RELEASE_YML=".github/workflows/release.yml"

fail() {
  echo "FF-18 release-workflow-no-pat: $*" >&2
  exit 1
}

[ -f "$RELEASE_YML" ] || fail "$RELEASE_YML not found"

# Extract all secrets references: ${{ secrets.FOO }}
# Allow ONLY secrets.GITHUB_TOKEN
# Reject any other secrets reference

found_bad=0
while IFS= read -r line; do
  # Extract secret name from ${{ secrets.NAME }}
  secret_name=$(printf '%s' "$line" | grep -oE 'secrets\.[A-Za-z0-9_]+' | sed 's/secrets\.//')
  if [ -n "$secret_name" ] && [ "$secret_name" != "GITHUB_TOKEN" ]; then
    echo "FF-18 release-workflow-no-pat: forbidden secret reference 'secrets.$secret_name' found in $RELEASE_YML (REQ-CVA-028: GITHUB_TOKEN only)" >&2
    found_bad=1
  fi
done < <(grep -E '\$\{\{[[:space:]]*secrets\.' "$RELEASE_YML" || true)

if [ "$found_bad" -eq 1 ]; then
  exit 1
fi

echo "FF-18 release-workflow-no-pat: OK"
