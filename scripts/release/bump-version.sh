#!/usr/bin/env sh
# scripts/release/bump-version.sh
# Computes the next semver version from (current_version, bump_type).
# Enforces: cli-versioning-automation REQ-CVA-034..REQ-CVA-039
#
# Usage:
#   ./bump-version.sh <current_version> <bump_type>
#
# Arguments:
#   current_version  The current version string, e.g. v0.2.3
#                    Must match: ^v[0-9]+\.[0-9]+\.[0-9]+$
#   bump_type        Either "minor" or "patch"
#
# Output:
#   On success (exit 0): the new version string on stdout, e.g. v0.2.4
#   On failure (exit 2): diagnostic to stderr; malformed current_version
#   On failure (exit 3): diagnostic to stderr; invalid bump_type
#
# POSIX-compatible: no bash-isms (no [[, no arrays, no pipefail, no set -o).
# Uses: set -eu only. Arithmetic via $(( )). Splitting via IFS + read.

set -eu

current="${1:-}"
bump="${2:-}"

# --- Validate current_version (exit 2 on malformed) ---

# POSIX case-glob primary validation: must start with v followed by digit groups.
# This is a first-pass check using shell globbing.
case "$current" in
  v[0-9]*.[0-9]*.[0-9]*)
    : # candidate — passes pattern; per-component check below
    ;;
  *)
    printf 'error: current_version must match ^v[0-9]+\\.[0-9]+\\.[0-9]+$ (got: %s)\n' "$current" >&2
    exit 2
    ;;
esac

# Strip the leading 'v' and split into components.
stripped="${current#v}"
major="${stripped%%.*}"
rest="${stripped#*.}"
minor_part="${rest%%.*}"
patch_part="${rest#*.}"

# Per-component all-digit validation (catch: v0.foo.1, v0..1, etc.)
for component in "$major" "$minor_part" "$patch_part"; do
  case "$component" in
    "" | *[!0-9]*)
      printf 'error: version component "%s" is not a non-empty digit string (input: %s)\n' \
        "$component" "$current" >&2
      exit 2
      ;;
  esac
done

# --- Validate bump_type (exit 3 on invalid) ---

case "$bump" in
  minor)
    minor_part=$((minor_part + 1))
    patch_part=0
    ;;
  patch)
    patch_part=$((patch_part + 1))
    ;;
  *)
    printf "error: bump_type must be 'minor' or 'patch' (got: %s)\n" "$bump" >&2
    exit 3
    ;;
esac

# --- Output new version (stdout only) ---
printf 'v%s.%s.%s\n' "$major" "$minor_part" "$patch_part"
