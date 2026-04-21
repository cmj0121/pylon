#!/usr/bin/env bash
# Cross-implementation diagnostic-set parity gate.
#
# For each fixture under scripts/diagnostics-corpus/, runs both the Go
# and JS diagnostic dumpers and diffs their canonical JSON outputs.
# Any mismatch signals that the two implementations emit different
# error text (or different sets) for the same source — a regression
# the byte-compare ASCII parity gate cannot catch.
#
# Coverage skips: bar.shape and bar.empty are unreachable via the
# shared YAML subset (scalar / empty series values are rejected by
# parseFrontmatter upstream). Those codes are exercised via direct
# Meta.Data injection in pkg/pylon/validate_test.go.
#
# Requires: dist/pylon-dump-diagnostics built (make -C src/go build)
# and dist/pylon.min.js built (make -C src/js build).

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
go_bin="$repo_root/dist/pylon-dump-diagnostics"
js_shim="$repo_root/scripts/pylon-dump-diagnostics.mjs"
corpus="$repo_root/scripts/diagnostics-corpus"

test -x "$go_bin" || { echo "missing $go_bin; run make -C src/go build"; exit 2; }
test -f "$repo_root/dist/pylon.min.js" || { echo "missing dist/pylon.min.js; run make -C src/js build"; exit 2; }
test -d "$corpus" || { echo "missing corpus $corpus"; exit 2; }

fail=0
total=0
while IFS= read -r -d '' src; do
  total=$((total + 1))
  name="$(basename "$src" .pylon)"
  go_out=$("$go_bin" "$src")
  js_out=$(node "$js_shim" "$src")
  if [[ "$go_out" != "$js_out" ]]; then
    echo "FAIL: $name"
    diff <(printf '%s\n' "$go_out") <(printf '%s\n' "$js_out") || true
    fail=$((fail + 1))
  fi
done < <(find "$corpus" -name '*.pylon' -print0 | sort -z)

if [[ $fail -ne 0 ]]; then
  echo ""
  echo "$fail of $total fixtures diverged"
  exit 1
fi

echo "parity-diagnostics: $total fixtures OK"
