#!/usr/bin/env bash
set -euo pipefail
TAGS="dev,plugins"
MODE="${1:-smoke}"
OUT="/tmp/flash_gotest_${MODE//\//_}.txt"

case "$MODE" in
   smoke)
      go test -tags="$TAGS" -v $(go list -tags="$TAGS" ./... | grep -v test/integration) 2>&1 | tee "$OUT" || true
      ;;
   race)
      go test -tags="$TAGS" -race -v $(go list -tags="$TAGS" ./... | grep -v test/integration) 2>&1 | tee "$OUT" || true
      ;;
   integration)
      (cd test/integration && go test -tags="$TAGS" -v ./...) 2>&1 | tee "$OUT" || true
      ;;
   all)
      go test -tags="$TAGS" -v ./... 2>&1 | tee "$OUT" || true
      ;;
   *)
      go test -tags="$TAGS" -v "$@" 2>&1 | tee "$OUT" || true
      ;;
esac

awk '
  /^--- PASS:/  { pass++ }
  /^--- FAIL:/  { fail++ }
  /^--- SKIP:/  { skip++ }
  /^ok /        { pkgpass++ }
  /^FAIL[ \t]/  { pkgfail++ }
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
' "$OUT"
