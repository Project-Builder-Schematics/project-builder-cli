#!/usr/bin/env bash
# FF-25: no-direct-stdout-in-features
# Enforces: output-discipline/REQ-01.1, output-discipline/REQ-02.1, output-discipline/REQ-03.1
#
# All user-facing emission in internal/feature/* MUST route through the
# output.Output port. Direct writes to stdout/stderr are forbidden in
# production code (test files are exempt).
#
# Blocked patterns:
#   fmt.Print / fmt.Println / fmt.Printf               — direct format/print
#   fmt.Fprint( / fmt.Fprintln( / fmt.Fprintf(         — write to os.Stdout or os.Stderr
#   cmd.Println / cmd.Printf / cmd.Print               — cobra.Command passthrough
#   io.WriteString(<os.Stdout|os.Stderr>)              — raw string write to stdout/stderr
#   os.Stdout.Write(                                   — raw byte write to stdout
#
# Exits 0 on a clean tree with a confirmation line.
# Exits 1 on any match, printing file:line:match to stderr.
#
# Usage:
#   bash scripts/fitness/no-direct-stdout-in-features.sh   # real codebase
set -euo pipefail

fail=0
matches=""

# Pattern 1: fmt.Print / fmt.Println / fmt.Printf (bare — not Fprint*)
# Matches any of fmt.Print( fmt.Println( fmt.Printf(
bare_print=$(rg -n \
  'fmt\.(Print|Println|Printf)\(' \
  --glob 'internal/feature/**/*.go' \
  --glob '!internal/feature/**/*_test.go' \
  2>/dev/null || true)

if [[ -n "$bare_print" ]]; then
  echo "FF-25 no-direct-stdout-in-features: fmt.Print* found in feature production code:" >&2
  echo "$bare_print" >&2
  fail=1
fi

# Pattern 2: fmt.Fprint* when writing to os.Stdout or os.Stderr
# e.g. fmt.Fprintln(os.Stdout, ...) or fmt.Fprintf(os.Stderr, ...)
fprint_stdout=$(rg -n \
  'fmt\.Fprint(ln|f)?\(os\.(Stdout|Stderr)' \
  --glob 'internal/feature/**/*.go' \
  --glob '!internal/feature/**/*_test.go' \
  2>/dev/null || true)

if [[ -n "$fprint_stdout" ]]; then
  echo "FF-25 no-direct-stdout-in-features: fmt.Fprint* to os.Stdout/os.Stderr found in feature production code:" >&2
  echo "$fprint_stdout" >&2
  fail=1
fi

# Pattern 3: cobra.Command passthrough methods (cmd.Println / cmd.Printf / cmd.Print)
cobra_print=$(rg -n \
  '\bcmd\.(Println|Printf|Print)\(' \
  --glob 'internal/feature/**/*.go' \
  --glob '!internal/feature/**/*_test.go' \
  2>/dev/null || true)

if [[ -n "$cobra_print" ]]; then
  echo "FF-25 no-direct-stdout-in-features: cmd.Print* found in feature production code:" >&2
  echo "$cobra_print" >&2
  fail=1
fi

# Pattern 4: io.WriteString targeting os.Stdout or os.Stderr
io_writestring=$(rg -n \
  'io\.WriteString\(os\.(Stdout|Stderr)' \
  --glob 'internal/feature/**/*.go' \
  --glob '!internal/feature/**/*_test.go' \
  2>/dev/null || true)

if [[ -n "$io_writestring" ]]; then
  echo "FF-25 no-direct-stdout-in-features: io.WriteString(os.Stdout/Stderr) found in feature production code:" >&2
  echo "$io_writestring" >&2
  fail=1
fi

# Pattern 5: os.Stdout.Write( — raw byte write to stdout
stdout_write=$(rg -n \
  'os\.Stdout\.Write\(' \
  --glob 'internal/feature/**/*.go' \
  --glob '!internal/feature/**/*_test.go' \
  2>/dev/null || true)

if [[ -n "$stdout_write" ]]; then
  echo "FF-25 no-direct-stdout-in-features: os.Stdout.Write( found in feature production code:" >&2
  echo "$stdout_write" >&2
  fail=1
fi

if [[ "$fail" -eq 0 ]]; then
  echo "FF-25 no-direct-stdout-in-features: OK"
fi

exit "$fail"
