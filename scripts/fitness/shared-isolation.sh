#!/usr/bin/env bash
# FF-04: shared-isolation
# Enforces: fitness-functions-ci.REQ-04.1, fitness-functions-ci.REQ-04.2
#
# Asserts two sub-rules:
#
#   (a) internal/shared/engine and internal/shared/render MUST NOT import any
#       internal/feature/* package and MUST NOT import concrete adapters.
#       They may import: stdlib, internal/shared/events, internal/shared/errors.
#
#   (b) internal/shared/events and internal/shared/errors MUST only import stdlib
#       (no external modules, no internal packages).
#
# Usage:
#   bash scripts/fitness/shared-isolation.sh  # real codebase only
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

  pkg_suffix="${pkg#${SHARED_PREFIX}/}"
  top_pkg="${pkg_suffix%%/*}"  # engine, render, events, errors, etc.

  for dep in $imports; do
    case "$top_pkg" in
      engine|render)
        # Rule (a): must not import feature/* or concrete adapters
        case "$dep" in
          "${FEATURE_PREFIX}/"*)
            echo "FF-04 shared-isolation: $pkg imports feature package $dep" >&2
            fail=1
            ;;
          "${MODULE}/"*)
            # Internal but not feature: only shared/* is allowed
            if [[ "$dep" != "${SHARED_PREFIX}/"* ]]; then
              echo "FF-04 shared-isolation: $pkg imports non-shared internal package $dep" >&2
              fail=1
            fi
            ;;
          *"."*)
            # External module — not allowed for engine/render at skeleton phase
            # (only stdlib + internal/shared are permitted)
            echo "FF-04 shared-isolation: $pkg imports external module $dep" >&2
            fail=1
            ;;
        esac
        ;;
      events|errors)
        # Rule (b): stdlib-only — no external modules, no internal packages
        case "$dep" in
          *"."*)
            echo "FF-04 shared-isolation: $pkg (events/errors) imports non-stdlib $dep" >&2
            fail=1
            ;;
          "${MODULE}/"*)
            echo "FF-04 shared-isolation: $pkg (events/errors) imports internal package $dep" >&2
            fail=1
            ;;
        esac
        ;;
    esac
  done
done < <(go list -f '{{ .ImportPath }}: {{ .Imports }}' ./internal/shared/... 2>/dev/null)

exit "$fail"
