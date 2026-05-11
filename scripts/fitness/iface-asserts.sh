#!/usr/bin/env bash
# FF-05: iface-asserts
# Enforces: fitness-functions-ci.REQ-05.1
#
# Verifies that compile-time interface satisfaction assertions exist for
# both FakeEngine (Engine interface) and NoopRenderer (Renderer interface).
# These assertions catch accidental interface breakage at compile time.
#
# Looks for:
#   var _ engine.Engine = ...  OR  var _ Engine = ...
#   var _ render.Renderer = ... OR  var _ Renderer = ...
#
# Usage:
#   bash scripts/fitness/iface-asserts.sh  # real codebase only
set -euo pipefail

fail=0

# Check for Engine interface assertion in engine package test or source files
if ! grep -rql 'var _ \(engine\.\)\?Engine = ' internal/shared/engine/ 2>/dev/null; then
  echo "FF-05 iface-asserts: missing 'var _ Engine = ...' assertion in internal/shared/engine/" >&2
  fail=1
fi

# Check for Renderer interface assertion in render package test or source files
if ! grep -rql 'var _ \(render\.\)\?Renderer = ' internal/shared/render/ 2>/dev/null; then
  echo "FF-05 iface-asserts: missing 'var _ Renderer = ...' assertion in internal/shared/render/" >&2
  fail=1
fi

exit "$fail"
