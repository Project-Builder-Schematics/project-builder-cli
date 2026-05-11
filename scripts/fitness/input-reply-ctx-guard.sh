#!/usr/bin/env bash
# FF-07: input-reply-ctx-guard
# Enforces: fitness-functions-ci.REQ-07.1
#
# For every Reply channel send site (lines matching '\.Reply\s*<-' or
# 'Reply\s*<-'), the surrounding 5 lines (before and after) must contain
# both 'select' and 'ctx.Done()'. This is a proxy check — an AST-based
# version is planned as a followup (see design §12).
#
# In real mode: scans all .go files under internal/
# In fixture mode (single positional arg): scans the given file
#
# Exits non-zero if any unguarded Reply send is found.
#
# Usage:
#   bash scripts/fitness/input-reply-ctx-guard.sh                  # real codebase
#   bash scripts/fitness/input-reply-ctx-guard.sh <fixture.go.txt> # fixture mode
set -euo pipefail

fail=0

if [ $# -ge 1 ]; then
  mapfile -t files < <(echo "$1")
else
  mapfile -t files < <(find internal -name '*.go' 2>/dev/null)
fi

for f in "${files[@]}"; do
  [ -f "$f" ] || continue

  # Read file into array of lines
  mapfile -t lines < "$f"
  total="${#lines[@]}"

  for (( i=0; i<total; i++ )); do
    line="${lines[$i]}"
    # Match Reply send patterns: `.Reply <-` or `Reply <-` (with optional spaces)
    if echo "$line" | grep -qE '\.?Reply[[:space:]]*<-'; then
      # Determine window: 5 lines before and after (clamped to file bounds)
      start=$(( i > 5 ? i - 5 : 0 ))
      end=$(( i + 5 < total - 1 ? i + 5 : total - 1 ))

      window=""
      for (( j=start; j<=end; j++ )); do
        window+="${lines[$j]}"$'\n'
      done

      has_select=$(echo "$window" | grep -c 'select' || true)
      has_ctx=$(echo "$window" | grep -c 'ctx\.Done()' || true)

      if [ "$has_select" -eq 0 ] || [ "$has_ctx" -eq 0 ]; then
        lineno=$(( i + 1 ))
        echo "FF-07 input-reply-ctx-guard: $f:$lineno — Reply send not ctx-guarded (missing select+ctx.Done() within 5 lines)" >&2
        fail=1
      fi
    fi
  done
done

exit "$fail"
