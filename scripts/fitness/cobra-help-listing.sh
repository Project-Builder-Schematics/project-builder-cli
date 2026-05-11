#!/usr/bin/env bash
# FF-06: cobra-help-listing
# Enforces: cobra-command-tree.REQ-01.2, fitness-functions-ci.REQ-06.1
#
# Runs `go run ./cmd/builder --help` and verifies that all 8 feature command
# names appear in the output. Fails if any name is missing.
#
# Usage:
#   bash scripts/fitness/cobra-help-listing.sh  # real codebase only
set -euo pipefail

COMMANDS=(init execute add info sync validate remove skill)
fail=0

help_output=$(go run ./cmd/builder --help 2>&1)

for cmd in "${COMMANDS[@]}"; do
  if ! echo "$help_output" | grep -q "\b${cmd}\b"; then
    echo "FF-06 cobra-help-listing: --help output missing command: ${cmd}" >&2
    fail=1
  fi
done

exit "$fail"
