#!/usr/bin/env bash
set -euo pipefail

go test -tags="dev,plugins" -v $(go list -tags="dev,plugins" ./... | grep -v test/integration) 2>&1 | tee /tmp/smoke_output.txt || true

awk '
  /^--- PASS:/  { pass++ }
  /^--- FAIL:/  { fail++ }
  /^--- SKIP:/  { skip++ }
  /^ok /        { pkgpass++ }
  /^FAIL /      { pkgfail++ }
  END {
    total = pass + fail + skip
    printf "\n────────────────────────────────────\n"
    printf "  Tests     : %d total\n", total
    printf "  Passed    : %d\n", pass
    printf "  Failed    : %d\n", fail
    printf "  Skipped   : %d\n", skip
    printf "  Packages  : %d ok, %d failed\n", pkgpass, pkgfail
    printf "────────────────────────────────────\n"
    if (fail > 0 || pkgfail > 0) exit 1
  }
' /tmp/smoke_output.txt
