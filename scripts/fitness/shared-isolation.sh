#!/usr/bin/env bash
# FF-04: shared-isolation
# Enforces: fitness-functions-ci.REQ-04.1, fitness-functions-ci.REQ-04.2
#
# Asserts three sub-rules:
#
#   (a) internal/shared/engine and internal/shared/render MUST NOT import any
#       internal/feature/* package and MUST NOT import concrete adapters.
#       They may import: stdlib, internal/shared/events, internal/shared/errors,
#       and — for the three sanctioned packages only — external lipgloss deps:
#         render/pretty        — charmbracelet/* (ADR-01, original lipgloss home)
#         render/theme         — charmbracelet/* + muesli/termenv (ADR-01 extension,
#                                color-palette-theming: TerminalColor + termenv.Profile)
#         render/output/themed — charmbracelet/* (unified-output-port S-005,
#                                production Output adapter uses lipgloss styles)
#
#   (b) internal/shared/events and internal/shared/errors MUST only import stdlib
#       (no external modules, no internal packages).
#
#   (c) internal/shared/render/json MUST NOT import any charmbracelet/* package.
#       Lipgloss must remain isolated to render/pretty only (ADR-01).
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
        # Rule (a): must not import feature/* or non-shared internal packages.
        # External modules: only charmbracelet/* is allowed, and ONLY for render/pretty.
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
            # External module — only allowed for the three sanctioned lipgloss importers:
            #   render/pretty  — ADR-01 (original lipgloss home)
            #   render/theme   — ADR-01 extension (color-palette-theming: lipgloss.TerminalColor +
            #                    muesli/termenv for termenv.Profile in resolver.go)
            #   render/output/themed — unified-output-port (S-005): production Output adapter
            #                          uses lipgloss styles derived from theme tokens
            # All other render/* and engine packages must remain stdlib + internal/shared only.
            if [[ "$pkg" == "${SHARED_PREFIX}/render/pretty"* && "$dep" == "github.com/charmbracelet/"* ]]; then
              : # sanctioned: render/pretty may import charmbracelet/* (ADR-01)
            elif [[ "$pkg" == "${SHARED_PREFIX}/render/pretty"* && "$dep" == "github.com/lucasb-eyer/"* ]]; then
              : # sanctioned: go-colorful is a transitive dep pulled by lipgloss/termenv
            elif [[ "$pkg" == "${SHARED_PREFIX}/render/theme"* && "$dep" == "github.com/charmbracelet/"* ]]; then
              : # sanctioned: render/theme may import charmbracelet/lipgloss for TerminalColor (ADR-01 extension, color-palette-theming)
            elif [[ "$pkg" == "${SHARED_PREFIX}/render/theme"* && "$dep" == "github.com/muesli/termenv"* ]]; then
              : # sanctioned: render/theme may import muesli/termenv for termenv.Profile (ADR-01 extension, color-palette-theming)
            elif [[ "$pkg" == "${SHARED_PREFIX}/render/output/themed"* && "$dep" == "github.com/charmbracelet/"* ]]; then
              : # sanctioned: render/output/themed may import charmbracelet/lipgloss for styled Output adapter (unified-output-port S-005)
            else
              echo "FF-04 shared-isolation: $pkg imports external module $dep" >&2
              fail=1
            fi
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

# Rule (c): render/json must NOT import any charmbracelet/* package (ADR-01).
# Lipgloss is isolated to render/pretty only.
json_charmbracelet=$(go list -json "${MODULE}/internal/shared/render/json" 2>/dev/null \
  | python3 -c "import sys,json; d=json.load(sys.stdin); print('\n'.join(i for i in d.get('Imports',[]) if 'charmbracelet' in i))" 2>/dev/null || true)
if [[ -n "$json_charmbracelet" ]]; then
  echo "FF-04 shared-isolation: render/json imports charmbracelet package(s): $json_charmbracelet" >&2
  fail=1
fi

exit "$fail"
