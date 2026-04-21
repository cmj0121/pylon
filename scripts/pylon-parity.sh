#!/usr/bin/env bash
# Cross-implementation parity gate.
#
# For each .pylon fixture under src/go/pylon/testdata/, render via
# both the Go binary (dist/pylon) and the JS library (src/js/pylon.js
# via a headless node shim) and byte-compare the ASCII outputs.
#
# Requires: dist/pylon built (run `make -C src/go build` first), and
# dist/pylon.min.js built (run `make -C src/js build` first).

set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
go_bin="$repo_root/dist/pylon"
js_bundle="$repo_root/dist/pylon.min.js"
fixtures="$repo_root/src/go/pylon/testdata"

test -x "$go_bin" || { echo "missing $go_bin; run make -C src/go build"; exit 2; }
test -f "$js_bundle" || { echo "missing $js_bundle; run make -C src/js build"; exit 2; }

fail=0
total=0
while IFS= read -r -d '' src; do
  total=$((total + 1))
  name="$(basename "$src" .pylon)"
  go_out=$("$go_bin" -f ascii "$src")
  js_out=$(node "$repo_root/scripts/pylon-render-js.mjs" "$src")
  if [[ "$go_out" != "$js_out" ]]; then
    echo "FAIL: $name"
    diff <(printf '%s' "$go_out") <(printf '%s' "$js_out") || true
    fail=$((fail + 1))
  fi
done < <(find "$fixtures" -name '*.pylon' -print0)

if [[ $fail -ne 0 ]]; then
  echo ""
  echo "$fail of $total fixtures diverged"
  exit 1
fi

echo "parity: $total fixtures OK"
