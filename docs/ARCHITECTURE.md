# Architecture

## Runtime overview

```text
Browser / Next.js UI
  -> Zustand store (`frontend/src/store/useGameStore.ts`)
  -> Backend API (`/api/...`) with cookie credentials
  -> Gin handlers
  -> PostgreSQL tables + CSV-seeded dictionaries
  -> Optional Redis cache and background workers
```

## Backend startup

`backend/cmd/server/main.go` thực hiện theo thứ tự:

1. Load `.env` từ `.env`, `backend/.env nếu có.
2. Kết nối PostgreSQL qua `db.Connect()`.
3. Gọi `db.InitSchema()` để tạo/migrate bảng và seed dữ liệu.
4. Kết nối Redis qua `cache.Connect()` nếu cấu hình sẵn; lỗi Redis chỉ warning.
5. Load Gu registry và Item registry từ database vào memory cache.
6. Start worker đói Cổ, worker tai kiếp và worker quái mỏ.
7. Khởi tạo Gin router và listen `PORT` hoặc `8080`.

## Backend folders

| Đường dẫn | Vai trò |
| --- | --- |
| `backend/cmd/server` | Binary entrypoint. |
| `backend/internal/server` | CORS và route table. |
| `backend/internal/db` | PostgreSQL connection, schema, migrations, seed CSV. |
| `backend/internal/models` | Domain structs, JSON scanners và registry cache. |
| `backend/internal/handlers` | HTTP handlers và gameplay services. |
| `backend/internal/middleware` | Auth và rate limiting. |
| `backend/internal/auth` | JWT helpers. |
| `backend/internal/cache` | Redis connector optional. |
| `backend/internal/worker` | Background jobs. |
| `backend/*.csv` | Seed data. |

## Frontend folders

| Đường dẫn | Vai trò |
| --- | --- |
| `frontend/src/app` | Next.js pages/layout: main app, topup, topup success. |
| `frontend/src/components` | UI gameplay views/components. |
| `frontend/src/components/killer-moves` | UI xây dựng/quản lý sát chiêu. |
| `frontend/src/store/useGameStore.ts` | Zustand state, API calls, optimistic/derived mana HP estimation, SSE subscriptions. |
| `frontend/public` | Static assets mặc định. |

## State and networking

- Frontend dùng cookie session (`credentials: "include"`) cho API protected.
- `NEXT_PUBLIC_BACKEND_URL` quyết định backend URL; mặc định là `http://localhost:8080`.
- Store gom state player, dictionary, recipes, marketplace, trade, party, arena, chat, mines, aperture, sect, feedback.
- SSE đang có cho player events, trade events và Bảo Hoàng chat.

## Background workers

- Hunger worker: giảm hunger/trạng thái Cổ theo chu kỳ.
- Tribulation worker: kiểm tra/tạo tai kiếp và xử lý enemy grid.
- Mine monster worker: gán quái vào mỏ bỏ hoang.

## Persistence model

Schema được tạo bằng SQL trong `backend/internal/db/db.go`. Các bảng quan trọng gồm:

- `players`, `referrals`, `wallets`, `topup_orders`, `wallet_transactions`.
- `gu_dictionary`, `item_dictionary`, `refining_recipes`.
- `regions`, `map_cells`, `monster_templates`, `monster_spawns`, `loot_tables`.
- `immortal_apertures`, `dao_marks`, `aperture_resource_nodes`, `tribulation_events`, `tribulation_enemies`, `aperture_variant_humans`.
- `auction`, `trades`, `messages`, `parties`, `arena_rankings`.
- `sects`, `sect_members`, `sect_missions`, `player_sect_missions`, `sect_join_requests`, `sect_promotion_requests`, `sect_mines`.
- `mine_slots`, `instance_templates`, `player_instance_sessions`, `killer_moves`, `feedback`, `feedback_votes`.
