#!/usr/bin/env bash
# FF-15: schema-json-canonical
# Enforces: REQ-SJ-05 — the generated empty schema.json is byte-identical to
# the canonical form: {"inputs": {}}\n (two-space indent, trailing newline).
#
# How:
#   1. Run the schema_test.go golden assertion inline via `go test -run` to verify
#      MarshalEmpty produces the exact canonical bytes.
#   2. Additionally, check that testdata/golden/schematic_empty/schema.json
#      matches the canonical form byte-for-byte.
#
# Severity: HARD (blocks merge).
# Cross-ref: FF-15 / FF-builder-new-02
#
# Per REQ-SJ-05 (V1 SIGNED): NO full byte-stability contract for user edits —
# only the GENERATOR canonical form is asserted (what MarshalEmpty produces).
#
# Usage:
#   bash scripts/fitness/schema-json-canonical.sh
set -euo pipefail

GOLDEN="internal/feature/new/testdata/golden/schematic_empty/schema.json"
CANONICAL='{\n  "inputs": {}\n}\n'

# Step 1: Run the schema unit test to verify MarshalEmpty canonical bytes.
echo "FF-15: running Test_MarshalEmpty_CanonicalBytes..."
if ! go test ./internal/feature/new/... -run "^Test_MarshalEmpty_CanonicalBytes$" -count=1 -timeout 30s 2>&1; then
  echo "FF-15 FAIL: Test_MarshalEmpty_CanonicalBytes failed — MarshalEmpty does not produce canonical bytes" >&2
  exit 1
fi
echo "FF-15: Test_MarshalEmpty_CanonicalBytes PASS"

# Step 2: Verify the committed golden schema.json matches canonical form.
if [ ! -f "$GOLDEN" ]; then
  echo "FF-15 FAIL: golden file $GOLDEN not found" >&2
  exit 1
fi

# Build expected bytes (printf interprets \n as newline).
expected=$(printf '%b' '{\n  "inputs": {}\n}\n')
actual=$(cat "$GOLDEN")

if [ "$actual" != "$expected" ]; then
  echo "FF-15 FAIL: $GOLDEN does not match canonical form." >&2
  echo "--- expected ---" >&2
  printf '%b' '{\n  "inputs": {}\n}\n' | cat -A >&2
  echo "--- actual ---" >&2
  cat -A "$GOLDEN" >&2
  exit 1
fi

echo "FF-15 PASS: MarshalEmpty canonical bytes verified + golden schema.json matches."
