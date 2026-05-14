#!/usr/bin/env bash
# FF-12: init-errcode-additive  (spec cross-ref: FF-init-03)
# Enforces: builder-init-end-to-end REQ-EC-01 (additive error codes)
#
# The 5 baseline ErrCode constants (pre-builder-init-end-to-end) MUST remain
# present in internal/shared/errors/codes.go. Removing or renaming any of
# them breaks downstream consumers and violates the additive-only contract.
# The 6 new init_* codes MUST also be present.
#
# Strategy: grep for each required constant declaration; fail if any is
# missing. New constants beyond this list are allowed (genuinely additive).
#
# In real mode: scans internal/shared/errors/codes.go.
# In fixture mode (single positional arg): scans the given file.
#
# Usage:
#   bash scripts/fitness/init-errcode-additive.sh
#   bash scripts/fitness/init-errcode-additive.sh <fixture.go.txt>
set -euo pipefail

# Baseline (pre-builder-init) constants that MUST remain.
BASELINE=(
  ErrCodeNotImplemented
  ErrCodeCancelled
  ErrCodeInvalidInput
  ErrCodeEngineNotFound
  ErrCodeExecutionFailed
)

# New init_* constants introduced by builder-init-end-to-end S-000.
INIT_NEW=(
  ErrCodeInitDirNotEmpty
  ErrCodeInitConfigExists
  ErrCodeInitAgentFileAmbiguous
  ErrCodeInitPackageManagerNotFound
  ErrCodeInitSkillExists
  ErrCodeInitNotImplemented
)

fail=0

if [ $# -ge 1 ]; then
  target="$1"
else
  target="internal/shared/errors/codes.go"
fi

if [ ! -f "$target" ]; then
  echo "FF-12 init-errcode-additive: $target not found" >&2
  exit 1
fi

for const in "${BASELINE[@]}"; do
  if ! grep -qE "\\b${const}\\b" "$target"; then
    echo "FF-12 init-errcode-additive: baseline constant ${const} missing from $target" >&2
    fail=1
  fi
done

# Init constants are only required when we're scanning the real codes.go,
# not arbitrary fixtures.
if [ $# -lt 1 ]; then
  for const in "${INIT_NEW[@]}"; do
    if ! grep -qE "\\b${const}\\b" "$target"; then
      echo "FF-12 init-errcode-additive: init constant ${const} missing from $target" >&2
      fail=1
    fi
  done
fi

exit "$fail"
