#!/usr/bin/env bash
# FF-17: projectconfig-followup-marker (SOFT)
# Enforces: ADR-027 — F-01 followup commitment before builder-add lands.
#
# Asserts that internal/feature/new/projectconfig.go contains the comment
# marker "FOLLOWUP F-01" indicating the intent to promote to
# internal/shared/projectconfig/ before builder-add ships.
#
# SOFT gate: exits 0 with a WARN message on stderr if the marker is absent.
# Does NOT block CI — the marker is a human reminder, not a hard invariant.
#
# When F-01 is implemented (promotion to shared/projectconfig/), both this
# script and the marker comment are deleted as part of that change.
#
# Usage:
#   bash scripts/fitness/projectconfig-followup-marker.sh
set -euo pipefail

TARGET="internal/feature/new/projectconfig.go"

if [ ! -f "$TARGET" ]; then
	echo "FF-17 SKIP: $TARGET not found (S-001 not yet landed)" >&2
	exit 0
fi

if grep -q "FOLLOWUP F-01" "$TARGET"; then
	echo "FF-17 projectconfig-followup-marker OK: marker present in $TARGET"
	exit 0
fi

echo "FF-17 WARN: 'FOLLOWUP F-01' marker missing from $TARGET" >&2
echo "FF-17 WARN: Add '// FOLLOWUP F-01: promote to internal/shared/projectconfig/' before builder-add." >&2
# SOFT: exit 0 to allow CI to continue
exit 0
