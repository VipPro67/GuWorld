# [SPECIFICATION] Đại Hội Võ Thuật — Giải Đấu 1v1 Theo Cảnh Giới (Martial Arts Tournament)

**Type:** `wayfinder:spec`
**Status:** Implemented
**Tác giả:** (agent)
**Ngày:** 2026-10-11

---

## 1. Mục tiêu & phạm vi

Mở định kỳ một "Đại Hội Võ Thuật": snapshot **người chơi đang online**, gom theo **cảnh giới tu vi** (`players.rank`), mỗi rank lấy **tối đa 128 người gần cảnh giới nhất** (tối thiểu 8 mới mở giải), đấu **1v1 loại trực tiếp** tới khi còn 1 người = vô địch của rank đó.

- **Rank dùng để gom:** `players.rank` (cảnh giới tu vi, 0 → 10, xem `models.GetRankName`). Không dùng `arena_rankings.rank`.
- **Mỗi rank = 1 bracket riêng.** Rank có **< 8 người online → bỏ qua** (không tổ chức rank đó). **Không gộp rank.**
- **Seed theo `sub_rank` gần nhất** (không shuffle) → người gần cảnh giới nhất gặp nhau ở vòng 1.
- **Không có giai đoạn đăng ký** (khác Death Arena): người online tại thời điểm khai mạc được tự động đưa vào.

---

## 2. Nguyên tắc thiết kế

**Clone Death Arena, đổi "team 6v6" → "người chơi 1v1" và "đăng ký tay" → "auto-gom online theo rank".** Toàn bộ máy móc bracket/worker/resolve/reward/SSE/recovery đã battle-tested, giữ nguyên cấu trúc.

| Death Arena (hiện có) | Đại Hội Võ Thuật (mới) |
|---|---|
| `death_arena_tournaments` + `death_arena_teams` + `death_arena_team_members` + `death_arena_matches` | `martial_tournaments` + `martial_combatants` + `martial_matches` |
| Đăng ký tay (REGISTRATION 16:00–17:00 Thứ 7) | Không đăng ký; snapshot online lúc khai mạc |
| `GenerateBracket` từ registered teams (shuffle ngẫu nhiên) | `GenerateMartialBracket` từ online players (giữ thứ tự `sub_rank`, KHÔNG shuffle) |
| `resolveTeamMatch` (N vs N) | `resolveMartialMatch` (1 vs 1) |
| `StartDeathArenaWorker` + recovery | `StartMartialArenaWorker` + recovery (clone) |
| Route `/death-arena/*` | Route `/martial-arena/*` |
| SSE channel `death_arena:events` | SSE channel `martial_arena:events` |

---

## 3. Mô hình dữ liệu (schema)

File mới `backend/internal/db/schema_martial_arena.go`, đăng ký trong `db.go` (`InitSchema`) — đặt **sau** `players` (vì có FK tới `players`) và ngang hàng `death_arena`.

```sql
-- Giải đấu (1 dòng = 1 bracket của 1 rank)
CREATE TABLE IF NOT EXISTS martial_tournaments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status VARCHAR(20) NOT NULL DEFAULT 'LOCKED'
        CHECK (status IN ('LOCKED','IN_PROGRESS','FINISHED')),
    rank INT NOT NULL,                          -- players.rank (cảnh giới 0..10)
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    total_players INT NOT NULL DEFAULT 0,
    current_round INT NOT NULL DEFAULT 0,
    total_rounds INT NOT NULL DEFAULT 0,
    champion_player_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_martial_tourn_status ON martial_tournaments(status);
CREATE INDEX IF NOT EXISTS idx_martial_tourn_sched ON martial_tournaments(scheduled_at);

-- Người tham gia (1 dòng = 1 người trong 1 bracket rank)
CREATE TABLE IF NOT EXISTS martial_combatants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES martial_tournaments(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    seed_number INT NOT NULL,
    rank INT NOT NULL,
    sub_rank INT NOT NULL,
    eliminated BOOLEAN NOT NULL DEFAULT FALSE,
    eliminated_at_round INT,
    final_placement INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tournament_id, player_id)
);
CREATE INDEX IF NOT EXISTS idx_martial_comb_tourn ON martial_combatants(tournament_id);

-- Trận đấu (1 dòng = 1 cặp 1v1)
CREATE TABLE IF NOT EXISTS martial_matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES martial_tournaments(id) ON DELETE CASCADE,
    round_number INT NOT NULL,
    bracket_position INT NOT NULL,
    combatant_a_id UUID NOT NULL REFERENCES martial_combatants(id),
    combatant_b_id UUID REFERENCES martial_combatants(id),
    winner_combatant_id UUID REFERENCES martial_combatants(id),
    combat_log JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING','BYE','IN_PROGRESS','FINISHED')),
    reward_stones INT NOT NULL DEFAULT 0,
    reward_items JSONB DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_martial_matches_round ON martial_matches(tournament_id, round_number);
CREATE INDEX IF NOT EXISTS idx_martial_matches_status ON martial_matches(tournament_id, status, round_number, bracket_position);
```

Thưởng gồm 5 loại (nguyên thạch, tiên nguyên thạch, item, cổ trùng, danh hiệu) → **claim-based**, clone bảng `death_arena_champion_rewards`:

```sql
CREATE TABLE IF NOT EXISTS martial_champion_rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tournament_id UUID NOT NULL REFERENCES martial_tournaments(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    placement INT NOT NULL,
    combatant_id UUID NOT NULL REFERENCES martial_combatants(id),
    title VARCHAR(100),
    reward_stones INT NOT NULL DEFAULT 0,
    reward_immortal_stones INT NOT NULL DEFAULT 0,
    reward_gu_id VARCHAR(100),
    reward_gu_rank INT,
    reward_items JSONB DEFAULT '[]',
    claimed BOOLEAN NOT NULL DEFAULT FALSE,
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_martial_rewards_tourn ON martial_champion_rewards(tournament_id);
CREATE INDEX IF NOT EXISTS idx_martial_rewards_player ON martial_champion_rewards(player_id);
```

---

## 4. Vòng đời giải đấu

```
[đến giờ khai mạc] ──> snapshot online theo rank ──> tạo bracket từng rank ──> resolve 1v1 ──> advance ──> vô địch ──> FINISHED
                              (rank < 2: skip)
```

Bỏ hẳn trạng thái `REGISTRATION` (không có đăng ký). Các trạng thái còn lại: `LOCKED` → `IN_PROGRESS` → `FINISHED`.

---

## 5. Matchmaking (gom online theo rank) — logic mới duy nhất

**Định nghĩa "online"**: tái dùng đúng pattern của `leaderboard.go`:

```sql
(last_activity >= NOW() - INTERVAL '5 minutes'
 OR last_mana_update >= NOW() - INTERVAL '5 minutes')
```

(`isPlayerOnline` trong `tribulation_online.go` có thêm tier WS/in-memory, nhưng là check 1 người — với snapshot hàng loạt thì cửa sổ SQL 5 phút là đủ, giống base query của leaderboard.)

**Thuật toán gom** (chạy 1 lần lúc khai mạc):

```go
// 1. Danh sách rank đang có người online
ranks := SELECT DISTINCT rank FROM players
         WHERE (online) AND rank BETWEEN $minRank AND $maxRank
         ORDER BY rank

// 2. Với mỗi rank: snapshot tối đa 128 người gần sub_rank nhất
for _, rank := range ranks {
    players := SELECT id, sub_rank FROM players
               WHERE rank = $1 AND (online)
               ORDER BY sub_rank DESC
               LIMIT 128
    if len(players) < 8 { continue }           // rank < 8 người online → KHÔNG tổ chức
    createTournament(rank, players)            // tạo tournament + combatants + bracket + start
}
```

- `maxPlayersPerRank = 128` (đủ thì 7 vòng, không bye).
- `minPlayersPerRank = 8` (dưới 8 người thì bỏ qua cả rank).
- **Seed theo `sub_rank DESC`**, KHÔNG shuffle → người gần cảnh giới nhất gặp nhau ở vòng 1 (khớp yêu cầu "lấy player online gần nhất").
- **Không gộp rank**: rank nào đủ ≥ 8 người thì tự mở giải riêng.

---

## 6. Sinh bracket + bye

Clone `GenerateBracket` (`death_arena_generate.go`) thành `GenerateMartialBracket(tournamentID)`:
- Đầu vào: `martial_combatants` (đã seed sẵn theo `sub_rank`) thay vì registered teams.
- Bỏ `rand.Shuffle` (giữ thứ tự seed).
- Giữ nguyên `nextPowerOf2` + logic bye (số người 8–128 không phải lúc nào cũng là luỹ thừa 2 → bye xử lý người lẻ).
- `totalRounds = ceil(log2(n))`.

Clone `StartTournament` → `StartMartialTournament` (LOCKED → IN_PROGRESS, rồi `go processPendingMartialMatches`).

---

## 7. Resolve trận 1v1

Clone `resolveTeamMatch` (`death_arena_process.go`) thành `resolveMartialMatch(tournamentID, matchID, combatantAID, combatantBID *string)`:
- Thay `loadTeamMembersForCombat(teamID)` bằng `loadCombatant(combatantID)` — cùng câu SELECT (player + Gu + dao_marks + path_masteries + combat_settings), chỉ đổi `WHERE p.id = $1` và bỏ join `death_arena_team_members`.
- Tái dùng **nguyên vẹn** `buildTeamCombatant(member *teamMemberCombatant)` (đầu vào struct giữ nguyên) và `simulator.Simulate`:

```go
simState := &simulator.CombatState{Turn: 0, Seed: time.Now().UnixNano(), VirtualMode: true}
simState.Allies  = append(simState.Allies, ally)     // 1 người (Kind=Player)
simState.Enemies = append(simState.Enemies, enemy)   // 1 người (Kind=Enemy)
logs := simulator.Simulate(simState)
```

- Xác định winner: clone logic so `ally.HP` vs `enemy.HP` (giữ nguyên tie-break theo %HP, draw → A thắng).
- Lưu `combat_log` JSONB, update `winner_combatant_id`, mark `eliminated` cho combatant thua. **Không có thưởng theo vòng** (8–128 người, thưởng chỉ phát ở chung kết, xem mục 8).
- Giữ nguyên **fallback winner khi lỗi** (bracket không bao giờ kẹt).

---

## 8. Advance round + vô địch + thưởng

Clone `advanceToNextRound` (`death_arena_process.go`) thành `advanceMartialRound`:
- Lấy winner của vòng hiện tại → tạo matches vòng kế (pair kề nhau), BYE khi lẻ.
- Vòng cuối: cập nhật `champion_player_id`, status `FINISHED`, `finished_at`.
- Gọi `awardMartialChampionRewards(tx, tournamentID)` — clone `awardChampionRewards`, đổi team → combatant/player.

### Thưởng (5 loại, claim-based)

Clone `awardChampionRewards` → `awardMartialChampionRewards`: ghi `martial_champion_rewards` (placement + title + stones + immortal_stones + gu + items) cho **top 3**, và trao Gu ngay qua `db.AddPlayerGu` (y hệt Death Arena). Giá trị mặc định (điều chỉnh được trong code):

| Hạng | Danh hiệu | Nguyên thạch | Tiên nguyên thạch | Cổ trùng (Gu) | Item |
|---|---|---|---|---|---|
| 1 (Vô địch) | "Võ Tôn Đại Hội" | 5000 | 500 | 1 Gu rank cao | 1 item hiếm |
| 2 (Á quân) | "Võ Vương Đại Hội" | 3000 | 300 | 1 Gu rank vừa | 1 item |
| 3 (Quý quân) | "Võ Tướng Đại Hội" | 1500 | 150 | 1 Gu rank thấp | — |

- Claim: clone `ClaimReward` + `GetMyRewards` → cộng stones/immortal_stones, gán title, gắn items khi player bấm nhận.
- Cần thêm cột lưu title lên `players` (clone `death_arena_title` trong `schema_players.go` → thêm `martial_title VARCHAR(100) DEFAULT ''`).

Clone `processPendingMatches` thành `processPendingMartialMatches` (giữ nguyên mutex `isProcessing`, recovery match IN_PROGRESS → PENDING, intermission `RoundDuration`).

---

## 9. Scheduler định kỳ + recovery

Clone `death_arena_worker.go` → `martial_arena_worker.go`:
- `StartMartialArenaWorker(ctx)` — gọi từ `main.go` cùng chỗ `StartDeathArenaWorker`.
- `getNextMartialArenaTarget()` — đổi lịch: **Thứ 3 hàng tuần, 12:00 VN**. **Bỏ** các nhánh liên quan `REGISTRATION`/`OPEN_REGISTRATION` (vì không có đăng ký). Chỉ còn 1 event type `START_TOURNAMENT`.
- `handleScheduledEvent("START_TOURNAMENT")` → chạy snapshot + tạo bracket từng rank (mục 5).
- `recoverMartialArenaOnStartup()` — clone `recoverDeathArenaOnStartup`: resume giải `IN_PROGRESS` bị treo, start giải `LOCKED` bị kẹt. **Quan trọng vì** snapshot lúc khai mạc mà server restart giữa chừng thì bracket đã nằm trong DB, không cần snapshot lại.

> **Khác biệt recovery:** Death Arena có thể tái tạo giải từ `REGISTRATION` đã ghi. Đại Hội không có đăng ký nên nếu crash **trước** khi snapshot + tạo bracket, giải coi như bỏ lỡ (không hồi phục) — chấp nhận được vì khung thời gian snapshot chỉ là 1 khoảnh khắc.

---

## 10. API routes

File `backend/internal/server/server.go`, thêm group (clone block `/death-arena` dòng 845):

```go
martialArena := api.Group("/martial-arena")
martialArena.Use(middleware.AuthRequired())
martialArena.Use(middleware.RequireOnboarded())
martialArena.Use(middleware.RateLimitByPlayer(25, time.Minute))
{
    martialArena.GET("/current", handlers.GetCurrentMartialTournament)   // trạng thái + rank của bản thân
    martialArena.GET("/bracket", handlers.GetMartialBracket)            // trả danh sách bracket theo rank
    martialArena.GET("/matches", handlers.GetMyMartialMatches)
    martialArena.GET("/matches/:id/log", handlers.GetMartialMatchCombatLog)
    martialArena.GET("/history", handlers.GetMartialHistory)
    martialArena.GET("/rewards", handlers.GetMyMartialRewards)
    martialArena.POST("/claim-reward", handlers.ClaimMartialReward)
}
```

- **Bỏ** `/register`, `/unregister` (auto-gom) và `/my-team` (solo).
- `GET /bracket` trả cấu trúc hơi khác Death Arena: **nhiều bracket song song** (mỗi rank 1 cái), key theo `rank`:

```json
{
  "tournaments": [
    { "rank": 6, "tournament": {...}, "matches": [...] },
    { "rank": 5, "tournament": {...}, "matches": [...] }
  ]
}
```

- `GET /current` trả: giải đang diễn ra của **rank mình** (nếu có) + danh sách rank đang đấu.

---

## 11. Realtime (SSE) + cache

- Clone `death_arena_events.go` → `BroadcastMartialArenaEvent(eventType, tournamentID, data)` publish lên kênh `martial_arena:events`.
- Event types giữ nguyên: `tournament_started`, `match_updated`, `round_advanced`, `round_intermission`, `tournament_finished`.
- Cache keys: `martial_arena:bracket:current`, `martial_arena:bracket:<rank>`, `martial_arena:match_log:<id>` (clone pattern từ `GetBracket`/`GetMatchCombatLog`).
- Frontend: SSE subscribe kênh mới trong store (clone wiring hiện có của death arena).

---

## 12. Frontend

Clone bộ `frontend/src/components/DeathArena/*`:
- `MartialArenaView.tsx` — tab: Đang diễn ra / Bảng đấu / Lịch sử.
- `MartialArenaBracket.tsx` — hiển thị **nhiều bracket theo rank** (selector rank → bracket của rank đó).
- `MartialArenaCombatModal.tsx` — replay combat log (clone `DeathArenaCombatModal.tsx`).
- Thêm slice `martialArena` trong `frontend/src/store/slices/` + entry vào `useGameStore.ts` (clone slice `deathArena`).
- Thêm route/view trong `frontend/src/app/page.tsx` (map `martial-arena: "Đại Hội Võ Thuật"`).

---

## 13. File plan

**Tạo mới:**
- `backend/internal/db/schema_martial_arena.go`
- `backend/internal/handlers/martial_arena_generate.go` (snapshot + bracket + start)
- `backend/internal/handlers/martial_arena_process.go` (resolve 1v1 + advance)
- `backend/internal/handlers/martial_arena_worker.go` (scheduler + recovery)
- `backend/internal/handlers/martial_arena_read.go` (GET handlers)
- `backend/internal/handlers/martial_arena_rewards.go` (thưởng + claim)
- `backend/internal/handlers/martial_arena_events.go` (SSE broadcast)
- `frontend/src/components/MartialArena/*.tsx`
- `frontend/src/store/slices/martialArena.ts`

**Sửa:**
- `backend/internal/db/db.go` — gọi `schemaMartialArena` trong `InitSchema`
- `backend/cmd/server/main.go` — `go StartMartialArenaWorker(ctx)`
- `backend/internal/server/server.go` — group `/martial-arena`
- `frontend/src/store/useGameStore.ts` — compose slice
- `frontend/src/app/page.tsx` — route/view
- `docs/API.md` — ghi routes mới (theo AGENTS.md)

---

## 14. Thứ tự triển khai

1. Schema + đăng ký `InitSchema` (không phá vỡ gì, additive).
2. `martial_arena_generate.go`: snapshot + bracket + start (dùng `buildTeamCombatant` sẵn có để build combatant).
3. `martial_arena_process.go`: resolve 1v1 + advance (clone từ death_arena).
4. `martial_arena_worker.go`: scheduler + recovery.
5. Read handlers + SSE.
6. Routes.
7. Frontend (slice + view).
8. Test: `go test ./...`, `npm run lint`, `npm run build`. Manual test bằng cách giả lập online (UPDATE `last_activity`).

---

## 15. Rủi ro / edge cases

- **Crash giữa snapshot và tạo bracket**: giải rank đó mất (không có đăng ký để hồi phục). Chấp nhận; nếu cần thì snapshot vào 1 bảng staging trước rồi mới tạo bracket.
- **Người chơi offline giữa chừng**: vẫn đấu bình thường (auto-battle không cần online lúc resolve). Combatant đã snapshot vào DB.
- **Số người lẻ / không phải luỹ thừa 2**: bye xử lý qua `nextPowerOf2`.
- **Rank 10 (Vĩnh Sinh) ít người**: thường < 8 → bỏ qua (đã chốt: không gộp rank).
- **Một người ở nhiều rank?** `players.rank` là 1 giá trị duy nhất → mỗi người chỉ vào đúng 1 bracket.
- **Giải chạy song song nhiều rank**: `processPendingMartialMatches` dùng mutex `isProcessing` riêng cho martial (không share với death arena) — mỗi rank có thể xử lý tuần tự trong 1 worker, hoặc 1 goroutine/rank (nhẹ).

---

## 16. Quyết định đã chốt

1. **Lịch khai mạc:** Thứ 3 hàng tuần, **12:00 VN**.
2. **Không gộp rank:** mỗi rank tự mở giải riêng.
3. **Thưởng:** claim-based, 5 loại (nguyên thạch, tiên nguyên thạch, item, cổ trùng, danh hiệu) — bảng `martial_champion_rewards`.
4. **Quy mô:** `maxPlayersPerRank = 128`, `minPlayersPerRank = 8` (dưới 8 → bỏ qua rank).
