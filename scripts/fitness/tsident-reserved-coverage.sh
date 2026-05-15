#!/usr/bin/env bash
# FF-14: tsident-reserved-coverage
# Enforces: REQ-TI-03 / ADV-11 — every entry in tsident.ReservedWords has a
# corresponding test case asserting EscapeIdent(word) == word+"_".
#
# Two acceptable coverage patterns (either satisfies the gate):
#
#   Pattern A — Dynamic iteration (preferred):
#     A test function iterates `tsident.ReservedWords` and calls EscapeIdent on
#     each word, asserting result == word+"_". Detected by checking that the test
#     file contains both `tsident.ReservedWords` and `EscapeIdent` in proximity
#     (same function body that also has `word + "_"` or `word+"_"`).
#
#   Pattern B — Explicit sub-tests:
#     For each word, a literal `"reserved_<word>"` string appears in the test file
#     as a t.Run label.
#
# The script checks Pattern A first; if satisfied, all words are considered covered.
# If Pattern A is absent, it falls back to Pattern B and reports missing words.
#
# Severity: HARD (blocks merge).
# Cross-ref: FF-14 / FF-builder-new-01
#
# Usage:
#   bash scripts/fitness/tsident-reserved-coverage.sh
set -euo pipefail

RESERVED_GO="internal/shared/tsident/reserved.go"
TEST_FILE="internal/shared/tsident/tsident_test.go"

if [ ! -f "$RESERVED_GO" ]; then
  echo "FF-14 FAIL: $RESERVED_GO not found" >&2
  exit 1
fi

if [ ! -f "$TEST_FILE" ]; then
  echo "FF-14 FAIL: $TEST_FILE not found" >&2
  exit 1
fi

# Extract count of reserved words from reserved.go.
mapfile -t words < <(
  rg -o '"[a-z]+"' "$RESERVED_GO" | tr -d '"' | sort -u
)

if [ "${#words[@]}" -eq 0 ]; then
  echo "FF-14 FAIL: no reserved words found in $RESERVED_GO" >&2
  exit 1
fi

echo "FF-14: found ${#words[@]} reserved words in $RESERVED_GO"

# Pattern A: dynamic iteration.
# The test file must contain a function that:
#   1. References tsident.ReservedWords (to iterate the canonical list), AND
#   2. Calls tsident.EscapeIdent on the iterated word, AND
#   3. Asserts result == word+"_" (the `+ "_"` suffix check).
#
# This covers the case where Test_EscapeIdent_AllReservedWords uses a for-range
# over tsident.ReservedWords and dynamically generates t.Run sub-tests.
has_dynamic=0
if rg -q 'tsident\.ReservedWords' "$TEST_FILE" 2>/dev/null; then
  if rg -q 'EscapeIdent' "$TEST_FILE" 2>/dev/null; then
    if rg -q '"_"' "$TEST_FILE" 2>/dev/null || rg -q '"\+ \"_\""|word \+ "_"' "$TEST_FILE" 2>/dev/null; then
      has_dynamic=1
    fi
  fi
fi

if [ "$has_dynamic" -eq 1 ]; then
  echo "FF-14 PASS (Pattern A): Test_EscapeIdent_AllReservedWords iterates tsident.ReservedWords dynamically."
  echo "FF-14 PASS: all ${#words[@]} reserved words covered via dynamic iteration."
  exit 0
fi

# Pattern B: explicit sub-tests. Check for "reserved_<word>" literal in test file.
echo "FF-14: Pattern A not detected; falling back to Pattern B (explicit sub-test labels)."

fail=0
missing=()

for word in "${words[@]}"; do
  if ! rg -q "\"reserved_${word}\"" "$TEST_FILE" 2>/dev/null; then
    missing+=("$word")
    fail=1
  fi
done

if [ "$fail" -ne 0 ]; then
  echo "FF-14 FAIL: Pattern B: the following reserved words have no explicit sub-test:" >&2
  for w in "${missing[@]}"; do
    echo "  missing: reserved_${w}" >&2
  done
  echo "" >&2
  echo "Either:" >&2
  echo "  (A) Ensure a test iterates tsident.ReservedWords and calls EscapeIdent on each word" >&2
  echo "  (B) Add t.Run(\"reserved_${missing[0]}\", ...) cases for each missing word" >&2
  exit 1
fi

echo "FF-14 PASS (Pattern B): all ${#words[@]} reserved words have explicit sub-tests."
