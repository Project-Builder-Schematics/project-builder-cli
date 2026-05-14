#!/usr/bin/env bash
# FF-13: init-skill-bytes-stable  (spec cross-ref: FF-init-04)
# Enforces: builder-init-end-to-end REQ-SA-01 (bundled SKILL.md byte-stability)
#
# The SKILL.md placeholder is bundled into the binary via //go:embed and is
# a durable supply-chain artefact. Any change to its bytes MUST be deliberate
# and surface in code review. This FF compares the SHA-256 of the current
# template/SKILL.md against the committed .sha256 fixture. Drift fails CI;
# updating the placeholder requires also updating the fixture in the same PR
# (which forces reviewer attention on the supply-chain change).
#
# In real mode: computes SHA-256 of internal/feature/init/template/SKILL.md.
# In fixture mode (single positional arg): treats the arg as the markdown
#   file to hash; compares against the .sha256 sibling.
# Exits non-zero on hash drift, missing files, or missing fixture.
#
# Usage:
#   bash scripts/fitness/init-skill-bytes-stable.sh
#   bash scripts/fitness/init-skill-bytes-stable.sh <fixture.md>
set -euo pipefail

if [ $# -ge 1 ]; then
  TARGET="$1"
else
  TARGET="internal/feature/init/template/SKILL.md"
fi

FIXTURE="${TARGET}.sha256"

if [ ! -f "$TARGET" ]; then
  echo "FF-13 init-skill-bytes-stable: target $TARGET not found" >&2
  exit 1
fi

if [ ! -f "$FIXTURE" ]; then
  echo "FF-13 init-skill-bytes-stable: fixture $FIXTURE not found — commit the .sha256 alongside the markdown" >&2
  exit 1
fi

expected=$(awk '{print $1}' "$FIXTURE")
actual=$(sha256sum "$TARGET" | awk '{print $1}')

if [ "$expected" != "$actual" ]; then
  echo "FF-13 init-skill-bytes-stable: SHA-256 drift on $TARGET" >&2
  echo "  expected: $expected (from $FIXTURE)" >&2
  echo "  actual:   $actual" >&2
  echo "If this change is deliberate, update $FIXTURE with the new hash and re-run." >&2
  exit 1
fi

exit 0
