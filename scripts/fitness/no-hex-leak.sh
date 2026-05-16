#!/usr/bin/env bash
# FF-24: no-hex-leak
# Enforces: theme-tokens/REQ-03.1, render-pretty/REQ-05.1, REQ-05.2
#           (from sdd/color-palette-theming)
#
# Hex color literals matching "#[0-9a-fA-F]{6,8}" MUST NOT appear in Go
# source files outside internal/shared/render/theme/. That package is the
# only authorised home for raw hex values; all other code references tokens
# by name via theme.Resolve(<token>).
#
# Exits 0 on a clean tree; non-zero (with file:line list to stderr) if any
# violation is found.
#
# Usage:
#   bash scripts/fitness/no-hex-leak.sh   # real codebase
set -euo pipefail

if matches=$(rg -n '"#[0-9a-fA-F]{6,8}"' --type go \
    -g '!internal/shared/render/theme/**' 2>/dev/null); then
    if [[ -n "$matches" ]]; then
        echo "FF-24 no-hex-leak: raw hex color literal(s) outside internal/shared/render/theme/" >&2
        echo "$matches" >&2
        exit 1
    fi
fi
echo "FF-24 no-hex-leak: OK"
