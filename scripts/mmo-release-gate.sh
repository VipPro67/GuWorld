#!/usr/bin/env bash
# Repeatable automated subset of the MMO Gate D release checklist.
#
# This script never claims a release gate passed. It runs the checks that can be
# reproduced locally/CI, prints their results, and leaves browser E2E, staging
# load, DB crash/dupe replay, canary and on-call evidence explicitly NOT MEASURED.
#
# Usage:
#   scripts/mmo-release-gate.sh
#   MMO_RELEASE_SKIP_FRONTEND=1 scripts/mmo-release-gate.sh
#   MMO_RELEASE_SKIP_RACE=1 scripts/mmo-release-gate.sh
#   MMO_RELEASE_SOAK_SECONDS=60 scripts/mmo-release-gate.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND="$ROOT/backend"
FRONTEND="$ROOT/frontend"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

SKIP_FRONTEND="${MMO_RELEASE_SKIP_FRONTEND:-0}"
SKIP_RACE="${MMO_RELEASE_SKIP_RACE:-0}"
SOAK_SECONDS="${MMO_RELEASE_SOAK_SECONDS:-10}"

step() {
  printf '\n== %s ==\n' "$1"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'MISSING: %s\n' "$1" >&2
    exit 2
  }
}

require_cmd go
require_cmd node
require_cmd npm

printf 'MMO release gate (automated subset)\n'
printf 'root=%s\n' "$ROOT"
printf 'commit=%s\n' "$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || printf 'unknown')"
printf 'date=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf 'frontend=%s race=%s soak=%ss\n' \
  "$([ "$SKIP_FRONTEND" = 1 ] && printf skipped || printf included)" \
  "$([ "$SKIP_RACE" = 1 ] && printf skipped || printf included)" \
  "$SOAK_SECONDS"

step "backend format (realtime surface)"
UNFORMATTED="$(cd "$BACKEND" && gofmt -l \
  internal/realtime internal/ws internal/handlers/monitor.go internal/server/server.go \
  cmd/server cmd/mmo_slice cmd/mmo_bench cmd/mmo_soak cmd/gudata internal/db/schema_realtime.go)"
if [ -n "$UNFORMATTED" ]; then
  printf '%s\n' "$UNFORMATTED" >&2
  printf 'FAIL: gofmt drift\n' >&2
  exit 1
fi
printf 'PASS: gofmt clean\n'

step "backend vet"
(cd "$BACKEND" && go vet ./...)
printf 'PASS: go vet\n'

step "backend tests"
(cd "$BACKEND" && go test ./...)
printf 'PASS: go test ./...\n'

if [ "$SKIP_RACE" != 1 ]; then
  step "backend race tests (realtime/ws)"
  (cd "$BACKEND" && go test -race ./internal/realtime/... ./internal/ws/...)
  printf 'PASS: race tests\n'
fi

step "Gu coverage + compiled catalog parity"
(cd "$BACKEND" && \
  go run ./cmd/gudata audit-realtime -data data -strict -csv "$TMP_DIR/gu-coverage.csv")
if ! cmp -s "$TMP_DIR/gu-coverage.csv" "$ROOT/docs/mmo/gu-coverage.csv"; then
  printf 'FAIL: docs/mmo/gu-coverage.csv is stale; regenerate it intentionally\n' >&2
  exit 1
fi
printf 'PASS: coverage CSV matches\n'

(cd "$BACKEND" && \
  go run ./cmd/gudata compile-realtime-gu -data data \
    -profiles data/mmo/gu/transpiled_profiles.json \
    -report "$TMP_DIR/gu-compiled.json")
if ! cmp -s "$TMP_DIR/gu-compiled.json" "$ROOT/docs/mmo/gu-compiled.json"; then
  printf 'FAIL: docs/mmo/gu-compiled.json is stale; regenerate it intentionally\n' >&2
  exit 1
fi
printf 'PASS: compiled catalog matches\n'

step "map server/client parity"
(cd "$BACKEND" && go test ./internal/realtime/nav/ -run 'TestServerAndClientMapCopiesStayIdentical|TestAlphaMapsParseAndLink|TestAlphaMapMonsterIDsExistInCSV')
printf 'PASS: map artifacts\n'

step "headless 5-player slice"
(cd "$BACKEND" && go run ./cmd/mmo_slice -players 5 -mobs 10 -ticks 400)
printf 'PASS: slice harness\n'

step "reconnect/ownership soak (${SOAK_SECONDS}s)"
(cd "$BACKEND" && go run ./cmd/mmo_soak -duration "${SOAK_SECONDS}s" -players 5 -mobs 5 -reconnect-every 2s)
printf 'PASS: soak smoke (long soak still NOT MEASURED)\n'

if [ "$SKIP_FRONTEND" != 1 ]; then
  step "frontend dependency check"
  if [ ! -d "$FRONTEND/node_modules" ]; then
    printf 'FAIL: frontend/node_modules missing; run npm ci or set MMO_RELEASE_SKIP_FRONTEND=1\n' >&2
    exit 1
  fi

  step "frontend lint"
  (cd "$FRONTEND" && npm run lint)
  printf 'PASS: lint\n'

  step "frontend typecheck"
  (cd "$FRONTEND" && npm run typecheck)
  printf 'PASS: typecheck\n'

  step "frontend unit tests"
  (cd "$FRONTEND" && npm test)
  printf 'PASS: unit tests\n'

  step "frontend production build"
  (cd "$FRONTEND" && npm run build)
  printf 'PASS: build\n'
fi

cat <<'EOF'

== Automated subset result ==
PASS: format, vet, backend tests, race, Gu coverage/compile parity, map parity,
      slice, reconnect smoke, frontend lint/typecheck/test/build (unless skipped).

NOT MEASURED (Gate D cannot pass without these):
- Browser E2E for movement/aim/combat/loot/reconnect/flag-OFF legacy.
- 5/30/100 real-network client load, RTT/jitter/loss, FPS/VRAM, AOI bytes.
- PostgreSQL/Redis crash injection, loot replay 100x, checkpoint loss bounds.
- Staging canary 5% -> 25% -> 100% and a rollback rehearsal.
- On-call paging/dashboard drill and signed QA/operations approval.

Record evidence in docs/mmo/release-checklist.md and do not mark Gate D passed
from this script alone.
EOF
