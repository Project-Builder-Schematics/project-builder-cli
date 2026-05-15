#!/usr/bin/env bash
# FF-22: codeowners-workflows
# Enforces: cli-versioning-automation REQ-CVA-030
#
# Asserts:
#   1. .github/CODEOWNERS exists.
#   2. It contains a line whose pattern starts with /.github/workflows/
#      followed by at least one owner token (@user or @org/team).
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/codeowners-workflows.sh
set -euo pipefail

CODEOWNERS=".github/CODEOWNERS"

if [ ! -f "$CODEOWNERS" ]; then
  echo "FF-22 codeowners-workflows: $CODEOWNERS not found" >&2
  exit 1
fi

# Pattern: line starts with /.github/workflows/ and has at least one @owner
if ! grep -qE '^/\.github/workflows/[[:space:]]+@[A-Za-z0-9_/-]+' "$CODEOWNERS"; then
  echo "FF-22 codeowners-workflows: $CODEOWNERS has no rule for /.github/workflows/ with an @owner" >&2
  echo "  Expected a line like: /.github/workflows/  @username-or-team" >&2
  exit 1
fi

echo "FF-22 codeowners-workflows: OK"
