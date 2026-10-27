# WF-REF-02 Research: Luồng Luyện Cổ hiện tại & Pattern phiên chạy nền

> Khảo sát phục vụ map [Phiên Luyện Cổ Theo Thời Gian & Minigame Nhiều Bước](../refine-timed-session-map.md). Thời điểm khảo sát: 2026-10-27.

## 1. Luồng refine hiện tại (backend)

### 1.1 Route & rate limit
- `POST /api/refine` → `handlers.RefineGu`, `AuthRequired + RequireOnboarded + RateLimitByPlayer(20/min)` — `backend/internal/server/server.go:690-698`.
- `POST /api/refine/deduce` → `handlers.DeduceRecipe` — cùng nhóm route.
- `GET /api/recipes` → `handlers.GetRecipes`, rate limit theo IP 60/phút, không bắt buộc auth — `server.go:490`.

### 1.2 Request / response
- `RefineRequest{recipe_id, selected_gu_instance_ids}` — `handlers/refine_core.go:101-104`.
- Response dùng `respondActionWithInventory` — `handlers/respond.go:57-104` — gồm inventory, stones, mana, HP, soul, rank, `state_revision`; extra `status`, `message`, `new_gu`, `rate_breakdown` — `refine_gu.go:673-686`.
- `RefineRateBreakdown` có 17 trường, gồm `combo_accuracy` (hiện chỉ là `!guRequirementsNotMet`) — `refine_core.go:33-49`.
- Lỗi nghiệp vụ vẫn trả HTTP 200 với `status: "failed"`.

### 1.3 Transaction & thứ tự trừ tài nguyên
- Một `db.DB.Begin()` cho toàn bộ request, `defer tx.Rollback()`, commit cuối — `refine_gu.go:38-43`, `636-639`.
- Validate mana (`:104-118`), đá (`:120-129`), nguyên liệu (`:132-142`), chọn Cổ (`:144-206`).
- Trừ mana + đá **trước khi roll** (`:209-213`); roll tại `:346`; nguyên liệu trừ **sau roll** theo outcome — FAIL_SAFE mất 50% (`:378-396`), còn lại mất 100% (`:397-413`).
- Cổ nguyên liệu: success → tạo instance mới (`:420-454`); failure → Cổ thường bị xóa, Cổ Bản Mệnh còn sống nhưng `DYING` (`:497-541`); backfire trừ 50% HP và 30% Soul (`:524-536`).
- Trạng thái `refine_fatigue` (10 phút) / `refine_backfire` (20 phút) và pity `insight_<rank>` ghi vào `players.statuses` JSONB — `:553-593`.
- Publish sau commit: `ClearPendingPlayerQueues` + `PublishPlayerSnapshot(...)` — `:641-642`.
- Side effect ngoài transaction: audit log (`:651-664`), notification (`:666-670`).

### 1.4 RNG
- `rand.Float64()` toàn cục, không seed, không lưu roll, không commit/reveal — `refine_gu.go:346`.
- Worker refine khác truyền roll từ ngoài vào (`worker/variant_refinement.go:85`).

### 1.5 Publish realtime
- `PublishPlayerSnapshot` — `handlers/player_events.go:112-129` — invalidate cache, reload snapshot, publish `player:<uuid>` qua Redis.
- Transport là **WebSocket** `/api/ws` (`server.go:459-475`), không phải SSE.

---

## 2. Pattern phiên/timer có thể tái dùng

### (a) State machine nhiều bước — `player_instance_sessions` (phù hợp nhất)
- DDL: `db/schema_player_items.go:173-192` — `current_stage`, `status`, `temp_inventory JSONB`, partial unique index `player_active_session_idx … WHERE status='ACTIVE'`.
- Kiểu stage: `InstanceStage{stage_index, description, options}`, `InstanceOption{text, next_stage, requirements}` — `handlers/instance.go:143-153`.
- `EnterInstance` tạo stage 0 (`:336-527`); `ActionInstance` chỉ nhận `option_index`, validate, trừ chi phí, update stage hoặc COMPLETED/FAILED (`:529-935`).
- Hạn chế: **không có timer từng bước, không auto-expire**; session ACTIVE bỏ dở sẽ chặn re-entry.

### (b) Deadline + worker auto-resolve — tribulation
- DDL `tribulation_events` (`started_at`, `ended_at`, `status`) — `db/schema_aperture.go:284-303`.
- Worker `ProcessActiveTribulationGrid` — `handlers/tribulation_worker.go:17-45` — đọc ngoài tx, xử lý từng event trong tx ngắn với `statement_timeout`.
- Deadline sweep 24h — `tribulation_trigger.go:749-773`; auto-resolve — `:1127-1205`.
- Mọi so sánh dùng `NOW()` / `time.Since(started_at)` phía server.

### (c) Queue + lock chéo — aperture refinement
- Bảng `aperture_refinement_queue` — `db/schema_aperture.go:216-237`.
- Worker `variant_refinement.go`: advisory lock theo aperture (`:45-49`), `SELECT … FOR UPDATE OF q SKIP LOCKED` (`:124-128`), cập nhật tiến độ theo lô (`:253`).
- Lưu ý: `processApertureRefinement` hiện không có caller — đây là pattern dữ liệu/UI, chưa phải vòng chạy thật.

### (d) Lazy elapsed-time
- `primeval_veins.accumulated_since` (`schema_world.go:213`, tính khi đọc `primeval_vein.go:55, 182-372`).
- Aperture nodes qua `last_collected_at` (`aperture_collect.go:156-208`), cleanse cooldown (`aperture_cleanse.go:22, 67-77`).

### (e) Hạ tầng worker
- `worker.RunEvery` / `SafeRun` — `backend/internal/worker/runner.go:14-44`.
- Ví dụ đăng ký: `cmd/server/main.go:468`; status expiry `worker/status_expiry.go:15-97`; sect mission `worker/sect_mission.go`.

**Khuyến nghị:** (a) cho state machine bước, (b) cho deadline/auto-resolve, (c) cho lock session khi nhiều nguồn cùng chạm.

---

## 3. Minigame / staged interaction hiện có

- **Không có minigame, quicktime, puzzle hay timing game nào.**
- Grep `minigame|mini-game|quicktime|puzzle|accuracy` chỉ trúng stat `Accuracy` của combat, validator catalog, và `combo_accuracy` (boolean cosmetic) của refine.
- Staged interaction duy nhất là instance/dream-realm (choice-based, không timer từng bước).

---

## 4. Data model công thức

- DDL `refining_recipes` — `db/schema_dictionaries.go:111-125` — PK `(target_gu_id, target_rank, variant)`; cột `required_gus`, `required_materials` JSONB, `success_rate`, `primeval_stones_cost`, `immortal_essence_stones_cost`; FK `gu_dictionary(id, rank)`.
- **Không có** duration, stage count, stage config.
- Go model `models.RefiningRecipe` — `models/recipe.go:29-40`; `RequiredGuItem{Rank, Qty}` (`:18-26`); cache `models.GetRecipeByTarget`.
- CSV `backend/data/Recipe.csv` (~4.483 dòng), header: `target_gu_id,target_rank,variant,required_gus,required_materials,success_rate,primeval_stones_cost,immortal_essence_stones_cost`.
- Import Admin: `recipe → Recipe.csv → refining_recipes`, key 3 cột — `db/import_workflow.go:74`.
- Cần bảng/cột mới cho nhịp độ (WF-REF-06). Precedent JSON stage list: `instance_templates.map_data JSONB` — `db/schema_world.go:236-251`.

---

## 5. Offline / login

- **Không có resolve session khi login.** Login chỉ evaluate mana/HP/lifespan (`player_auth_login.go:40-85`, `:292-324`).
- Mọi deadline hiện tại do worker toàn cục: tribulation 24h, spatial realm expiry (`spatial_realms.go:585-604`), sect mission, status expiry.
- Hệ quả: phiên refine bỏ dở phải có **worker sweep** và/hoặc **lazy-resolve** ở endpoint GET session.

---

## 6. Lock & rủi ro phiên dài

- `RefineGu` gọi `evaluateManaTx` → `PlayerSvc.EvaluateAndPersistMana` (`mana.go:15-17` → `services/player_service.go:462-509`).
- Hàm này `setFastFailTimeouts` (lock 2s, statement 8s — `player_service.go:372`) rồi `playerstate.Load(..., ForUpdate: true)` = `SELECT … FOR UPDATE` trên `players` (`:47-49`, `:64-66`), update `version = version + 1` (`:487-500`).
- Vì vậy row `players` bị khóa suốt handler. **Không được kéo dài qua thời gian chờ** — sẽ chặn login/combat/movement/feed của chính người chơi (`55P03`/`57014`).
- Pattern an toàn: lock row session (`FOR UPDATE`), chỉ lock player trong transaction ngắn của từng bước/finalize; worker finalize qua đúng `PlayerSvc` trong tx riêng.

---

## 7. Frontend

### 7.1 `RefiningFurnace.tsx` (1.420 dòng)
- Mount từ `app/page.tsx:27` (dynamic, ssr:false), render tại `:452-461` với `onRefine={refineGu}`.
- Slot Cổ theo `required_gus`, click và drag/drop; recipe < 100% bị chặn, phải deduce trước (`:1080-1127`).
- `handleRefine` (`:283-349`): set `isRefining`, chạy label stage giả mỗi 600ms (`:290-305`), `setTimeout` 2.5s rồi gọi **một lần** `onRefine` (`:308-313`). Không có abort, không cleanup interval/timeout khi unmount.
- Kết quả render overlay full-screen trong panel (`:1184-1420`); breakdown tỉ lệ chỉ hiện **sau** khi có kết quả (`:494-580`, `:1285`, `:1393`).
- Hiệu ứng shake/fire/smoke dùng CSS keyframes (`app/globals.css:294-330`).

### 7.2 Store
- `cultivationSlice.ts`: `refineGu` (`:185-225`) — `beginAction("player:cultivation")` chặn toàn cục, POST `/api/refine`, không optimistic update, `applyInventoryFromResponse` + toast + log; `finally endAction`.
- `deduceRecipe` (`:227-273`) tương tự.
- `applyInventoryFromResponse` — `useGameStore.ts:1400-1469+` (merge theo revision).

### 7.3 Realtime
- WebSocket manager `lib/websocket.ts` — `subscribe` (`:208-234`), `request` (`:252`), `command` (`:275`).
- `subscribeEvents` — `useGameStore.ts:1785`; kênh `"player"` (`:1812-1969`) xử lý các event đã biết, rồi **fallback snapshot** ở `:1939-1965`. Event lạ không có `id`/`patch` bị **bỏ qua im lặng**.
- Kênh notification: server `notify` publish `notif:<uuid>` (`pkg/notify/notify.go:182-184`) → client `"notification"` → `notificationsSlice.ts:100-118`.
- Muốn thêm event session phải sửa: hằng số/type `player_events.go:33-43`, hàm publish, có thể thêm channel prefix trong `ws/redis_sub.go:16-23, 91-111` + `ws/client.go:168-181`, và nhánh subscribe trong `useGameStore`.

### 7.4 UI pattern tái dùng
- `components/ui/GameProgressBar.tsx`: `GameProgressBar` (variant `"refinement"`, `:5-15`), `GameCircularProgress` (`:159-211`), `GameStepProgress` (`:215-275`).
- Countdown: `CountdownTimer.tsx:12-50`, `SpatialRealmHUD.tsx:13-58`, `TribulationRealTimeInfo.tsx:85-116`.
- Staged reveal: `GachaPanel.tsx:41-88` (600ms/reward, 2s finish, cooldown).
- Job có tiến độ gần nhất: `ImmortalApertureView.tsx:3146-3247` + `apertureSlice.ts:553-610` (queue UI).
- Modal: `GameModal.tsx`; toast: `useUIStore.addToast`; notification bell: `NotificationBell.tsx`.

### 7.5 Minigame stack
- Phaser 4.2.1 chỉ dùng cho thế giới MMO (`game/bootstrap/createPhaserGame.ts`, scenes `Boot/Preload/World`).
- Không có `<canvas>`/`requestAnimationFrame` nào khác trong components.
- v1 nên dùng DOM/SVG + CSS, không đụng Phaser.

---

## 8. Kết luận cho thiết kế

1. Tạo **bảng phiên riêng** thay vì nhồi vào `players`; một ACTIVE/người chơi.
2. Mỗi request bước là một transaction ngắn, lock row phiên; finalize mới lock `players`.
3. Worker quét phiên hết hạn 30s/lần + lazy-resolve khi client mở UI.
4. Client chỉ gửi input rời rạc; server giữ seed + deadline + điểm.
5. UI dùng lại `GameStepProgress`/`GameProgressBar`, thay đoạn giả lập trong `RefiningFurnace`.
