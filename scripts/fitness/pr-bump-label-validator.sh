#!/usr/bin/env bash
# FF-21: pr-bump-label-validator
# Enforces: cli-versioning-automation REQ-CVA-006..014
#
# Asserts that .github/workflows/ci.yml contains a job named
# `bump-label-validation` satisfying all of:
#
#   1. The workflow triggers on pull_request events that include at minimum:
#      opened, reopened, synchronize, labeled, unlabeled
#
#   2. The job has a permissions block with pull-requests: read
#
#   3. Label data is accessed via toJson(github.event.pull_request.labels)
#      assigned to an env var — NOT inline ${{ }} shell-expansion in run: blocks
#      (anti-injection per REQ-CVA-014)
#
#   4. The validation logic references both bump:minor AND bump:patch literals
#
#   5. The error message for zero-label case is distinct from the two-label case:
#      - zero:  message includes "0" or "bump:minor" or "bump:patch" (label names)
#      - two:   message includes "mutually exclusive" or "cannot have both" or "both"
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/pr-bump-label-validator.sh
set -euo pipefail

CI_YML=".github/workflows/ci.yml"

fail() {
  echo "FF-21 pr-bump-label-validator: $*" >&2
  exit 1
}

[ -f "$CI_YML" ] || fail "$CI_YML not found"

# 1. Job named bump-label-validation must exist
grep -q 'bump-label-validation:' "$CI_YML" \
  || fail "no job named 'bump-label-validation' found in $CI_YML"

# 2. pull_request event types must include labeled and unlabeled
# The on: block must list labeled and unlabeled under pull_request types
grep -q 'labeled' "$CI_YML" \
  || fail "pull_request event types do not include 'labeled' in $CI_YML"
grep -q 'unlabeled' "$CI_YML" \
  || fail "pull_request event types do not include 'unlabeled' in $CI_YML"
# Must also include synchronize (REQ-CVA-012)
grep -q 'synchronize' "$CI_YML" \
  || fail "pull_request event types do not include 'synchronize' in $CI_YML"

# 3. pull-requests: read permission must appear in ci.yml (for the new job)
grep -q 'pull-requests: read' "$CI_YML" \
  || fail "no 'pull-requests: read' permission found in $CI_YML (required for bump-label-validation job)"

# 4. Labels accessed via toJson env var — NOT inline ${{ github.event.pull_request.labels }} in run:
# The LABELS_JSON env var pattern (toJson) must be present
grep -q 'toJson(github.event.pull_request.labels)' "$CI_YML" \
  || fail "labels must be accessed via toJson(github.event.pull_request.labels) env var, not inline interpolation (REQ-CVA-014)"

# 5. Both bump:minor and bump:patch must be referenced in the validation logic
grep -q 'bump:minor' "$CI_YML" \
  || fail "validation logic does not reference 'bump:minor' label in $CI_YML"
grep -q 'bump:patch' "$CI_YML" \
  || fail "validation logic does not reference 'bump:patch' label in $CI_YML"

# 6. Distinct error messages: zero-label case must reference label names or count
#    Two-label case must indicate mutual exclusivity
# Zero-label message: must contain a reference to the valid labels in an error context
grep -qE '(bump:minor|bump:patch).*(Add|add|label|0)|(Add|add|label|0).*(bump:minor|bump:patch)' "$CI_YML" \
  || fail "zero-label error message must reference valid label names and instruct user to add one (REQ-CVA-009.2)"

# Two-label message: must indicate mutual exclusivity
grep -qE '(mutually exclusive|cannot have both|both.*bump|Choose one|choose one)' "$CI_YML" \
  || fail "two-label error message must indicate mutual exclusivity (e.g. 'mutually exclusive', 'cannot have both') (REQ-CVA-009.3)"

echo "FF-21 pr-bump-label-validator: OK"
