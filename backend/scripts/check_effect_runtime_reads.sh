#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
pattern='Effects\.(Active|Passive|Growth|Other)'
check() {
  local domain=$1; shift
  if rg -n "$pattern" "$@"; then
    echo "legacy effect DTO read remains in ${domain}" >&2
    exit 1
  fi
}
check derived internal/gamefx/fx.go internal/gamefx/energy.go internal/handlers/player_stats.go internal/db/player_gu.go
check scouting internal/gamefx/scouting_engine.go internal/handlers/scouting.go internal/handlers/profile.go internal/handlers/refine_core.go
check growth internal/handlers/player_equip.go internal/worker/growth.go
if rg -n 'item\.Effects' internal/handlers/item_effects.go internal/handlers/player_use.go; then
  echo 'raw item effect read remains in item-use runtime' >&2
  exit 1
fi
