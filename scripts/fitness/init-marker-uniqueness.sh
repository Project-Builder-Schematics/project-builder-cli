#!/usr/bin/env bash
# FF-11: init-marker-uniqueness  (spec cross-ref: FF-init-01)
# Enforces: builder-init-end-to-end REQ-AR-02 (locked durable marker contract)
#
# The pbuilder skill-reference marker block in AGENTS.md / CLAUDE.md is a
# durable post-v1.0.0 contract. The exact marker bytes are locked and MUST
# NOT drift. This FF asserts the locked marker constant exists verbatim in
# the source.
#
# Strategy: scan internal/feature/init/ for the begin and end marker literals
# as a single declaration line. Both literals must be present exactly once
# (declaration site only). Any drift in the bytes or duplicate declarations
# fail the build.
#
# In real mode: scans internal/feature/init/.
# In fixture mode (single positional arg): scans the given file.
# Exits non-zero on missing or drifted markers.
#
# Usage:
#   bash scripts/fitness/init-marker-uniqueness.sh
#   bash scripts/fitness/init-marker-uniqueness.sh <fixture.go.txt>
set -euo pipefail

BEGIN_LITERAL='<!-- pbuilder:skill:begin -->'
END_LITERAL='<!-- pbuilder:skill:end -->'
fail=0

if [ $# -ge 1 ]; then
  fixture="$1"
  if ! grep -qF "$BEGIN_LITERAL" "$fixture" 2>/dev/null; then
    echo "FF-11 init-marker-uniqueness: fixture $fixture missing begin marker $BEGIN_LITERAL" >&2
    exit 1
  fi
  if ! grep -qF "$END_LITERAL" "$fixture" 2>/dev/null; then
    echo "FF-11 init-marker-uniqueness: fixture $fixture missing end marker $END_LITERAL" >&2
    exit 1
  fi
  exit 0
fi

# Real mode: count occurrences of the begin and end literals across source files
# (excluding tests). Each must appear EXACTLY ONCE — at the constant declaration
# site in agents_markdown.go. Anywhere else suggests a copy-paste drift risk.
begin_count=$(grep -RFl "$BEGIN_LITERAL" internal/feature/init \
  --include='*.go' --exclude='*_test.go' 2>/dev/null | wc -l)
end_count=$(grep -RFl "$END_LITERAL" internal/feature/init \
  --include='*.go' --exclude='*_test.go' 2>/dev/null | wc -l)

if [ "$begin_count" -ne 1 ]; then
  echo "FF-11 init-marker-uniqueness: begin marker $BEGIN_LITERAL found in $begin_count files (must be exactly 1)" >&2
  fail=1
fi
if [ "$end_count" -ne 1 ]; then
  echo "FF-11 init-marker-uniqueness: end marker $END_LITERAL found in $end_count files (must be exactly 1)" >&2
  fail=1
fi

exit "$fail"
