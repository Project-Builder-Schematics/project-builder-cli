#!/usr/bin/env sh
# bump-version.test.sh
# Test driver for scripts/release/bump-version.sh
# Enforces: cli-versioning-automation REQ-CVA-040, REQ-CVA-041
#
# Runs ≥8 test cases covering the full arithmetic matrix.
# Exit 0 if ALL cases pass. Exit 1 if any case fails.
# Prints PASS/FAIL per case and a summary line at the end.
#
# Usage:
#   sh scripts/release/bump-version.test.sh
#   bash scripts/release/bump-version.test.sh

SCRIPT="$(dirname "$0")/bump-version.sh"

pass_count=0
fail_count=0

run_case() {
  label="$1"
  input_ver="$2"
  input_bump="$3"
  expected_out="$4"
  expected_exit="$5"

  actual_out=$("$SCRIPT" "$input_ver" "$input_bump" 2>/dev/null)
  actual_exit=$?

  ok=1

  if [ "$expected_exit" = "nonzero" ]; then
    if [ "$actual_exit" -eq 0 ]; then
      ok=0
    fi
    # stdout must be empty on error-path cases
    if [ -n "$actual_out" ]; then
      ok=0
    fi
  else
    if [ "$actual_exit" -ne 0 ]; then
      ok=0
    fi
    if [ "$actual_out" != "$expected_out" ]; then
      ok=0
    fi
  fi

  if [ "$ok" -eq 1 ]; then
    echo "PASS  $label"
    pass_count=$((pass_count + 1))
  else
    if [ "$expected_exit" = "nonzero" ]; then
      echo "FAIL  $label | expected: exit nonzero + empty stdout | got: exit=$actual_exit stdout='$actual_out'"
    else
      echo "FAIL  $label | expected: '$expected_out' (exit 0) | got: '$actual_out' (exit $actual_exit)"
    fi
    fail_count=$((fail_count + 1))
  fi
}

# === Case 1: v0.0.0 + patch → v0.0.1 (REQ-CVA-032: first-release bootstrap) ===
run_case "v0.0.0 + patch → v0.0.1" "v0.0.0" "patch" "v0.0.1" "0"

# === Case 2: v0.0.0 + minor → v0.1.0 (REQ-CVA-032: first-release bootstrap) ===
run_case "v0.0.0 + minor → v0.1.0" "v0.0.0" "minor" "v0.1.0" "0"

# === Case 3: v0.0.5 + patch → v0.0.6 (REQ-CVA-035: basic arithmetic) ===
run_case "v0.0.5 + patch → v0.0.6" "v0.0.5" "patch" "v0.0.6" "0"

# === Case 4: v0.0.5 + minor → v0.1.0 (REQ-CVA-036: patch RESETS to 0 — primary mutation target) ===
# A bug that returns v0.1.5 instead of v0.1.0 will be caught HERE.
run_case "v0.0.5 + minor → v0.1.0 (patch reset)" "v0.0.5" "minor" "v0.1.0" "0"

# === Case 5: v0.9.0 + patch → v0.9.1 (REQ-CVA-037: multi-digit) ===
run_case "v0.9.0 + patch → v0.9.1" "v0.9.0" "patch" "v0.9.1" "0"

# === Case 6: v0.9.0 + minor → v0.10.0 (REQ-CVA-037: digit-boundary crossing) ===
run_case "v0.9.0 + minor → v0.10.0 (digit boundary)" "v0.9.0" "minor" "v0.10.0" "0"

# === Case 7: v0.0.99 + patch → v0.0.100 (REQ-CVA-037: three-digit patch) ===
run_case "v0.0.99 + patch → v0.0.100 (three-digit)" "v0.0.99" "patch" "v0.0.100" "0"

# === Case 8: "" (empty) + patch → exit nonzero, empty stdout (REQ-CVA-038: malformed input) ===
run_case "empty string + patch → exit nonzero" "" "patch" "" "nonzero"

# === Case 9: "garbage" + patch → exit nonzero (REQ-CVA-038: malformed version) ===
run_case "garbage + patch → exit nonzero" "garbage" "patch" "" "nonzero"

# === Case 10: v0.0.0 + bogus → exit nonzero (REQ-CVA-039: invalid bump_type) ===
run_case "v0.0.0 + bogus → exit nonzero" "v0.0.0" "bogus" "" "nonzero"

# === Summary ===
total=$((pass_count + fail_count))
echo ""
echo "${pass_count}/${total} passed"

if [ "$fail_count" -gt 0 ]; then
  exit 1
fi

exit 0
