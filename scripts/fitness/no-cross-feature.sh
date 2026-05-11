#!/usr/bin/env bash
# FF-02: no-cross-feature
# Enforces: feature-folder-layout.REQ-02.1, fitness-functions-ci.REQ-02.1
#
# Ensures no feature package imports a SIBLING top-level feature package.
# Parent→child imports within the same top-level feature are allowed
# (e.g. internal/feature/skill importing internal/feature/skill/update is OK).
#
# In real mode: uses `go list` over ./internal/feature/...
# In fixture mode (single positional arg): greps the given file for cross-feature
#   import paths (lines matching internal/feature/<name> where <name> is a known
#   top-level feature name different from the owning package).
#
# Usage:
#   bash scripts/fitness/no-cross-feature.sh                  # real codebase
#   bash scripts/fitness/no-cross-feature.sh <fixture.go.txt> # fixture mode
set -euo pipefail

MODULE="github.com/Project-Builder-Schematics/project-builder-cli"
FEATURE_PREFIX="${MODULE}/internal/feature"
fail=0

if [ $# -ge 1 ]; then
  # Fixture mode: grep the file for any internal/feature/<pkg> import path.
  # The fixture should contain a cross-feature import to trigger the violation.
  fixture="$1"
  if grep -qE "\"${FEATURE_PREFIX}/[a-z]+" "$fixture" 2>/dev/null; then
    echo "FF-02 no-cross-feature: fixture $fixture contains a cross-feature import" >&2
    exit 1
  fi
  exit 0
fi

# Real mode: use go list to enumerate each feature package and its imports.
# For each package, check if it imports a SIBLING top-level feature package.
while IFS= read -r line; do
  # Line format: "ImportPath: [dep1 dep2 ...]"
  pkg="${line%%:*}"
  imports="${line#*: }"
  # Strip brackets
  imports="${imports#[}"
  imports="${imports%]}"

  # Determine the top-level feature name for this package.
  # e.g. ".../internal/feature/skill/update" → top-level = "skill"
  suffix="${pkg#${FEATURE_PREFIX}/}"
  top_level="${suffix%%/*}"

  # Check each import
  for dep in $imports; do
    # Only interested in internal/feature/* imports
    if [[ "$dep" == ${FEATURE_PREFIX}/* ]]; then
      dep_suffix="${dep#${FEATURE_PREFIX}/}"
      dep_top="${dep_suffix%%/*}"
      # Allow parent→child: same top-level feature prefix
      if [ "$dep_top" != "$top_level" ]; then
        echo "FF-02 no-cross-feature: $pkg imports sibling feature $dep" >&2
        fail=1
      fi
    fi
  done
done < <(go list -f '{{ .ImportPath }}: {{ .Imports }}' ./internal/feature/... 2>/dev/null)

exit "$fail"
