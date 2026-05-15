#!/usr/bin/env bash
# FF-14: version-const-regex
# Enforces: cli-versioning-automation REQ-CVA-001
#
# Asserts that cmd/builder/version.go exists and contains a line matching:
#   const Version = "vX.Y.Z"
# where X, Y, Z are non-negative integers. The entire declaration must be on
# a single line with no leading/trailing whitespace inside the quoted value.
#
# Exit 0 on success. Exit 1 with diagnostic on failure.
#
# Usage:
#   bash scripts/fitness/version-const-regex.sh
set -euo pipefail

TARGET="cmd/builder/version.go"

if [ ! -f "$TARGET" ]; then
  echo "FF-14 version-const-regex: $TARGET not found" >&2
  exit 1
fi

# Match exactly: const Version = "vX.Y.Z" (digits only inside, no trailing spaces)
if ! grep -qE '^const Version = "v[0-9]+\.[0-9]+\.[0-9]+"$' "$TARGET"; then
  echo "FF-14 version-const-regex: $TARGET does not contain a valid 'const Version = \"vX.Y.Z\"' declaration" >&2
  echo "  Expected pattern: ^const Version = \"v[0-9]+\\.[0-9]+\\.[0-9]+\"\$" >&2
  exit 1
fi

echo "FF-14 version-const-regex: OK"
