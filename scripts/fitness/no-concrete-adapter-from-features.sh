#!/usr/bin/env bash
# FF-03: no-concrete-adapter-from-features
# Enforces: composition-root.REQ-02.1, fitness-functions-ci.REQ-03.1
#
# Asserts that feature packages only import from the allowed set:
#   - stdlib (no module prefix)
#   - github.com/spf13/cobra
#   - github.com/spf13/viper
#   - internal/feature/<same-top-level>/* (parent→child within same feature)
#   - internal/shared/* (ports and shared types)
#
# Any other external import (concrete adapters, infrastructure packages) fails.
#
# Usage:
#   bash scripts/fitness/no-concrete-adapter-from-features.sh  # real codebase only
set -euo pipefail

MODULE="github.com/Project-Builder-Schematics/project-builder-cli"
FEATURE_PREFIX="${MODULE}/internal/feature"
SHARED_PREFIX="${MODULE}/internal/shared"
fail=0

while IFS= read -r line; do
  pkg="${line%%:*}"
  imports="${line#*: }"
  imports="${imports#[}"
  imports="${imports%]}"

  suffix="${pkg#${FEATURE_PREFIX}/}"
  top_level="${suffix%%/*}"

  for dep in $imports; do
    case "$dep" in
      # Stdlib packages have no dot in the first path segment
      *"."*)
        # External module — check allowlist
        case "$dep" in
          "github.com/spf13/cobra"*)   ;;  # allowed
          "github.com/spf13/viper"*)   ;;  # allowed
          "github.com/charmbracelet/"*) ;;  # allowed (log)
          "${SHARED_PREFIX}/"*)         ;;  # allowed — ports + shared types
          "${FEATURE_PREFIX}/${top_level}/"*) ;;  # allowed — parent→child same feature
          "${FEATURE_PREFIX}/${top_level}") ;;    # allowed — self
          *)
            echo "FF-03 no-concrete-adapter: feature $pkg imports disallowed dep $dep" >&2
            fail=1
            ;;
        esac
        ;;
      # Stdlib — no dot in first segment: always allowed
      *) ;;
    esac
  done
done < <(go list -f '{{ .ImportPath }}: {{ .Imports }}' ./internal/feature/... 2>/dev/null)

exit "$fail"
