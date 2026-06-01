# Requirements Document

## Introduction

Hệ thống Tu Luyện & Thăng Cấp (Cultivation & Leveling System) cho game text-based RPG GuWorld, lấy cảm hứng từ tiểu thuyết "Cổ Chân Nhân" (Reverend Insanity). Hệ thống này thay thế cơ chế tích lũy EXP truyền thống bằng một quy trình tiêu hao tài nguyên và sinh tồn khắc nghiệt, chia làm hai giai đoạn: Tu Luyện Phàm Nhân (Rank 1–5) và Độ Kiếp Cổ Tiên (Rank 6–9).

Codebase hiện tại đã có skeleton cơ bản: các cột DB (`wall_progress`, `human_qi`, `next_tribulation_time`, `tribulation_count`) đã tồn tại, endpoint `POST /api/cultivate/break-wall` và `POST /api/cultivate/use-relic-gu` đã được triển khai một phần. Feature này sẽ hoàn thiện toàn bộ logic còn thiếu, bổ sung endpoint Thăng Tiên và Độ Kiếp, đồng thời nâng cấp toàn diện giao diện frontend.

## Glossary

- **Cultivation_System**: Hệ thống tu luyện tổng thể bao gồm cả phàm nhân và cổ tiên.
- **Wall_Breaking_Engine**: Module backend xử lý logic Trùng Kích Vách Khiếu (phá vách khiếu).
- **Ascension_Engine**: Module backend xử lý logic Thăng Tiên (Rank 5 Đỉnh Phong → Rank 6).
- **Tribulation_Engine**: Module backend xử lý logic Độ Kiếp (tai kiếp định kỳ cho Rank 6–9).
- **Aperture_UI**: Component frontend hiển thị trạng thái không khiếu và điều khiển tu luyện phàm nhân.
- **Blessed_Land_UI**: Component frontend hiển thị quản lý tiên khiếu và đếm ngược tai kiếp cho cổ tiên.
- **Player**: Người chơi đã đăng nhập, có đầy đủ thông tin rank, sub_rank, mana, stones, v.v.
- **Rank**: Đại cảnh giới tu luyện, từ 1 (Nhất Chuyển) đến 9 (Cửu Chuyển).
- **SubRank**: Tiểu cảnh giới trong mỗi Rank: 1 (Sơ Giai), 2 (Trung Giai), 3 (Cao Giai), 4 (Đỉnh Phong).
- **WallProgress**: Chỉ số tiến độ phá vách khiếu, từ 0.0 đến 1.0 (100%).
- **HumanQi**: Chỉ số Nhân Khí tích lũy từ chiến đấu và thành tựu, dùng cho Thăng Tiên.
- **AptitudeType**: Tư chất của người chơi: Giáp (A), Ất (B), Bính (C), Đinh (D).
- **PrimevalStones**: Nguyên Thạch — tiền tệ phàm nhân dùng để hỗ trợ tu luyện.
- **ImmortalStones**: Tiên Nguyên Thạch — tiền tệ cổ tiên.
- **DaoMarks**: Đạo Ngân — JSONB stat boost nhận được sau khi vượt qua tai kiếp.
- **Tribulation**: Tai kiếp — sự kiện PvE cường độ cao xảy ra định kỳ với cổ tiên.
- **TribulationCount**: Số lần tai kiếp đã vượt qua trong chu kỳ hiện tại.
- **NextTribulationTime**: Timestamp thời điểm tai kiếp tiếp theo sẽ xảy ra.
- **Relic_Gu**: Cổ Xá Lợi — vật phẩm tiêu hao đặc biệt giúp tăng ngay 1 tiểu cảnh giới.
- **Liquor_Worm_Gu**: Tửu Trùng — Cổ trùng hỗ trợ tinh luyện chân nguyên, tăng hiệu quả phá vách.
- **Mortal_Cultivator**: Cổ sư Phàm Nhân, Rank 1–5.
- **Immortal_Cultivator**: Cổ Tiên, Rank 6–9.

---

## Requirements

### Requirement 1: Mortal Wall-Breaking Cultivation (Trùng Kích Vách Khiếu)

**User Story:** As a Mortal_Cultivator (Rank 1–5), I want to spend mana and primeval stones to accumulate wall progress and break through to the next sub-rank, so that I can advance my cultivation level.

#### Acceptance Criteria

1. WHEN a Mortal_Cultivator submits a wall-breaking attempt with a valid `mana_spend` amount, THE Wall_Breaking_Engine SHALL deduct exactly `mana_spend` from `current_mana` and add the corresponding `wall_progress` gain before returning the updated Player.
2. WHEN a Mortal_Cultivator submits a wall-breaking attempt with a `stones_spend` amount greater than 0, THE Wall_Breaking_Engine SHALL deduct exactly `stones_spend` from `primeval_stones` and apply a proportional bonus to `wall_progress` gain.
3. WHEN `wall_progress` reaches or exceeds 1.0 after an attempt, THE Wall_Breaking_Engine SHALL automatically trigger a breakthrough: increment `sub_rank` by 1, reset `wall_progress` to 0.0, increase `max_mana` by 10%, and commit all changes in a single transaction.
4. WHEN a breakthrough causes `sub_rank` to exceed 4, THE Wall_Breaking_Engine SHALL set `sub_rank` to 1 and increment `rank` by 1.
5. WHEN a Mortal_Cultivator with `AptitudeType` "Giáp" performs a wall-breaking attempt, THE Wall_Breaking_Engine SHALL set the aptitude multiplier variable to 1.4 for use in the `wall_progress` gain calculation.
6. WHEN a Mortal_Cultivator with `AptitudeType` "Ất" performs a wall-breaking attempt, THE Wall_Breaking_Engine SHALL set the aptitude multiplier variable to 1.1 and apply it in the formula: `progress_gain = (mana_spend / max_mana) * (0.2 + 0.3 * aptitude_multiplier) + stones_spend * 0.05`.
7. WHEN a Mortal_Cultivator with `AptitudeType` "Bính" performs a wall-breaking attempt, THE Wall_Breaking_Engine SHALL set the aptitude multiplier variable to 0.85 for use in the `wall_progress` gain calculation.
8. WHEN a Mortal_Cultivator with `AptitudeType` "Đinh" performs a wall-breaking attempt, THE Wall_Breaking_Engine SHALL set the aptitude multiplier variable to 0.6 and calculate `wall_progress` gain as `(mana_spend / max_mana) * (0.2 + 0.3 * 0.6) + stones_spend * 0.05`.
9. WHEN a wall-breaking attempt is submitted with `mana_spend` less than or equal to 0 or greater than `current_mana`, THE Wall_Breaking_Engine SHALL reject the request with HTTP 400 and an error message without modifying the Player.
10. WHEN a wall-breaking attempt is submitted with `stones_spend` less than 0 or greater than `primeval_stones`, THE Wall_Breaking_Engine SHALL reject the request with HTTP 400 and an error message without modifying the Player.
11. WHEN a player with `rank` >= 6 attempts to call the wall-breaking endpoint, THE Wall_Breaking_Engine SHALL reject the request with HTTP 400 indicating only Mortal_Cultivators can perform this action.
12. WHEN a wall-breaking attempt does not trigger an automatic breakthrough (progress < 1.0), THE Wall_Breaking_Engine SHALL evaluate a probabilistic success chance based on `mana_spend / max_mana`, `stones_spend`, and `AptitudeType`; if the random roll succeeds, THE Wall_Breaking_Engine SHALL trigger a breakthrough.
13. WHEN a Mortal_Cultivator has an equipped Liquor_Worm_Gu with status "ACTIVE", THE Wall_Breaking_Engine SHALL set the `gu_multiplier` variable to 1.25 and multiply the computed `wall_progress` gain by this value.
14. WHEN a wall-breaking attempt is processed, THE Wall_Breaking_Engine SHALL persist all changes (`current_mana`, `primeval_stones`, `wall_progress`, `max_mana`, `sub_rank`, `rank`) within a single database transaction, rolling back all changes if any write fails.

---

### Requirement 2: Relic Gu Instant Advancement (Cổ Xá Lợi)

**User Story:** As a Mortal_Cultivator, I want to consume a Relic_Gu item to instantly advance one sub-rank without spending mana, so that I can bypass the wall-breaking process when I have the item.

#### Acceptance Criteria

1. WHEN a Mortal_Cultivator submits a relic-use request with a valid `relic_key` present in `materials_inventory` with quantity >= 1, THE Wall_Breaking_Engine SHALL decrement the relic quantity by 1, increment `sub_rank` by 1, and reset `wall_progress` to 0.0.
2. WHEN using a Relic_Gu causes `sub_rank` to exceed 4, THE Wall_Breaking_Engine SHALL set `sub_rank` to 1 and increment `rank` by 1.
3. WHEN a relic-use request is submitted with a `relic_key` not present in `materials_inventory` or with quantity 0, THE Wall_Breaking_Engine SHALL reject the request with HTTP 400 and an error message without modifying the Player.
4. WHEN a Relic_Gu is consumed, THE Wall_Breaking_Engine SHALL NOT deduct any `current_mana` or `primeval_stones`.
5. WHEN a Relic_Gu is consumed, THE Wall_Breaking_Engine SHALL persist all changes within a single database transaction.

---

### Requirement 3: Immortal Ascension (Thăng Tiên — Rank 5 → Rank 6)

**User Story:** As a Mortal_Cultivator at Rank 5 Đỉnh Phong (rank=5, sub_rank=4), I want to attempt the Immortal Ascension ritual, so that I can break through to become an Immortal_Cultivator at Rank 6.

#### Acceptance Criteria

1. WHEN a player with `rank` != 5 or `sub_rank` != 4 calls the ascension endpoint, THE Ascension_Engine SHALL reject the request with HTTP 400 indicating the prerequisite rank is not met.
2. WHEN a player with insufficient `primeval_stones` (below the rank-5 ascension cost threshold) calls the ascension endpoint, THE Ascension_Engine SHALL reject the request with HTTP 400 indicating insufficient resources.
3. WHEN a player with `human_qi` below the minimum threshold calls the ascension endpoint, THE Ascension_Engine SHALL reject the request with HTTP 400 indicating insufficient Nhân Khí.
4. WHEN a valid ascension attempt succeeds (sufficient `human_qi` and `primeval_stones`), THE Ascension_Engine SHALL set `rank` to 6, `sub_rank` to 1, reset `wall_progress` to 0.0, and convert `current_mana` to an equivalent `immortal_stones` amount.
5. WHEN a valid ascension attempt succeeds, THE Ascension_Engine SHALL set `next_tribulation_time` to exactly 10 minutes after the current server time and set `tribulation_count` to 0.
6. WHEN a valid ascension attempt succeeds, THE Ascension_Engine SHALL deduct the required `primeval_stones` cost AND reset `human_qi` to 0.0 within the same transaction.
7. WHEN a valid ascension attempt fails due to `human_qi` imbalance, THE Ascension_Engine SHALL reduce the player's `hp` to 1.0 (near-death penalty) without changing `rank` or `sub_rank`.
8. WHEN an ascension attempt is processed, THE Ascension_Engine SHALL persist all changes within a single database transaction.
9. WHEN a player with `rank` >= 6 calls the ascension endpoint, THE Ascension_Engine SHALL reject the request with HTTP 400.

---

### Requirement 4: Immortal Tribulation System (Độ Kiếp — Rank 6–9)

**User Story:** As an Immortal_Cultivator (Rank 6–9), I want to face periodic tribulation events when the timer expires, so that I can earn Dao Marks and eventually advance to the next rank.

#### Acceptance Criteria

1. WHEN a player with `rank` < 6 calls the tribulation endpoint, THE Tribulation_Engine SHALL reject the request with HTTP 400.
2. WHEN a player calls the tribulation endpoint before `next_tribulation_time` has been reached, THE Tribulation_Engine SHALL reject the request with HTTP 400 and return the remaining time in seconds.
3. WHEN a player calls the tribulation endpoint at or after `next_tribulation_time`, THE Tribulation_Engine SHALL initiate a tribulation combat event appropriate to the player's current `rank` and `tribulation_count`.
4. WHEN a Rank 6 player faces a tribulation and `tribulation_count` = 0 (first occurrence), THE Tribulation_Engine SHALL classify it as "Địa Tai" (Earth Calamity); for every 10th occurrence within the cycle, THE Tribulation_Engine SHALL classify it as "Thiên Kiếp" (Heavenly Tribulation), with `next_tribulation_time` set to 10 minutes after resolution.
5. WHEN a Rank 7 player faces a tribulation, THE Tribulation_Engine SHALL classify it as "Địa Tai", "Thiên Kiếp", or "Hạo Kiếp" (Grand Tribulation) based on `tribulation_count`, with `next_tribulation_time` set to 10 minutes after resolution.
6. WHEN a Rank 8 player faces a tribulation, THE Tribulation_Engine SHALL classify it as "Thiên Kiếp", "Hạo Kiếp", or "Vạn Kiếp" (Supreme Tribulation) based on `tribulation_count`, with `next_tribulation_time` set to 10 minutes after resolution.
7. WHEN a player survives a tribulation, THE Tribulation_Engine SHALL increment `tribulation_count` by 1 and grant `dao_marks` bonuses scaled to the tribulation type.
8. WHEN a player's `tribulation_count` reaches the rank-advancement threshold (3 major tribulations for the current rank), THE Tribulation_Engine SHALL increment `rank` by 1, set `sub_rank` to 1, and reset `tribulation_count` to 0.
9. WHEN a player dies during a tribulation (hp reaches 0), THE Tribulation_Engine SHALL apply a heavy penalty: reduce `immortal_stones` by 50% and reset `tribulation_count` to 0, without advancing rank.
10. WHEN a tribulation is resolved (win or loss), THE Tribulation_Engine SHALL persist all changes within a single database transaction.
11. WHEN a player survives a tribulation, THE Tribulation_Engine SHALL return a detailed log message describing the tribulation type, dao_marks gained, and next tribulation time.

---

### Requirement 5: HumanQi Accumulation

**User Story:** As a player, I want my HumanQi to increase automatically when I win combat encounters and complete exploration events, so that I can accumulate the resource needed for Immortal Ascension.

#### Acceptance Criteria

1. WHEN a player wins a PvE combat encounter, THE Cultivation_System SHALL increment `human_qi` by a base reward of 2.0 plus an amount proportional to the enemy's difficulty (enemy rank × 5.0).
2. WHEN a player successfully explores a map node, THE Cultivation_System SHALL increment `human_qi` by 1.0.
3. WHEN `human_qi` would exceed 1000.0, THE Cultivation_System SHALL cap it at 1000.0.
4. WHEN `human_qi` is updated, THE Cultivation_System SHALL persist the change within the same database transaction as the triggering action.

---

### Requirement 6: Mortal Cultivation UI (Aperture_UI — Ôn Dưỡng Không Khiếu)

**User Story:** As a Mortal_Cultivator, I want a clear and interactive cultivation panel in the dashboard, so that I can easily manage mana spending, view wall progress, and use Relic Gu items.

#### Acceptance Criteria

1. WHEN a Mortal_Cultivator views the dashboard, THE Aperture_UI SHALL display the current `wall_progress` as a percentage progress bar (0–100%) with a label "Tiến Độ Vách Khiếu".
2. WHEN a Mortal_Cultivator views the cultivation panel and `current_mana` > 0, THE Aperture_UI SHALL display a slider or numeric input allowing the player to choose a mana amount between 1 and `current_mana` (floored to integer); WHEN `current_mana` = 0, THE Aperture_UI SHALL hide the mana input slider.
3. WHEN a Mortal_Cultivator adjusts the mana input, THE Aperture_UI SHALL display a cost preview showing the estimated `wall_progress` gain and success probability before the player confirms.
4. WHEN a Mortal_Cultivator clicks the "Trùng Kích" button, THE Aperture_UI SHALL call the wall-breaking API, disable the button during the request, and display the result message in the LogConsole.
5. WHEN a Mortal_Cultivator has one or more Relic_Gu items in `materials_inventory`, THE Aperture_UI SHALL display a quick-use slot for each Relic_Gu with the item name and quantity.
6. WHEN a Mortal_Cultivator clicks a Relic_Gu quick-use button, THE Aperture_UI SHALL call the relic-use API and display the result in the LogConsole.
7. WHEN a breakthrough occurs (sub_rank or rank advances), THE Aperture_UI SHALL update all rank/sub_rank labels immediately; THE Aperture_UI SHALL also attempt to display a visual celebration effect, but label updates SHALL proceed independently even if the celebration effect fails to render.
8. WHEN the wall-breaking or relic-use API call is in progress, THE Aperture_UI SHALL set `actionLoading` to true and disable all cultivation action buttons to prevent duplicate submissions.

---

### Requirement 7: Immortal Cultivation UI (Blessed_Land_UI — Tiên Khiếu Quản Lý)

**User Story:** As an Immortal_Cultivator (Rank 6+), I want a dedicated Blessed Land Management panel that replaces the mortal cultivation UI, so that I can monitor tribulation timers, view my Dao Marks, and face tribulations when ready.

#### Acceptance Criteria

1. WHEN an Immortal_Cultivator views the dashboard, THE Blessed_Land_UI SHALL replace the Aperture_UI wall-breaking controls with the Blessed Land Management panel.
2. WHEN an Immortal_Cultivator views the Blessed Land panel, THE Blessed_Land_UI SHALL display a countdown timer showing time remaining until `next_tribulation_time` in DD:HH:MM:SS format.
3. WHEN `next_tribulation_time` has been reached (countdown reaches 00:00:00:00), THE Blessed_Land_UI SHALL enable a "Đối Mặt Tai Kiếp" (Face Tribulation) button and display a visual alert, regardless of whether the Blessed Land panel is currently visible to the user.
4. WHEN an Immortal_Cultivator clicks "Đối Mặt Tai Kiếp", THE Blessed_Land_UI SHALL call the tribulation API, disable the button during the request, and display the result in the LogConsole.
5. WHEN an Immortal_Cultivator views the Blessed Land panel, THE Blessed_Land_UI SHALL display all current `dao_marks` as a formatted list with mark name and value.
6. WHEN an Immortal_Cultivator views the Blessed Land panel, THE Blessed_Land_UI SHALL display the current `tribulation_count` and the number of major tribulations required to advance to the next rank.
7. WHEN the tribulation API call is in progress, THE Blessed_Land_UI SHALL set `actionLoading` to true and disable the tribulation button to prevent duplicate submissions.
8. WHEN an Immortal_Cultivator views the Blessed Land panel, THE Blessed_Land_UI SHALL display the current `immortal_stones` balance and the `rank`/`sub_rank` in the appropriate immortal terminology.

---

### Requirement 8: Ascension UI (Thăng Tiên Panel)

**User Story:** As a Mortal_Cultivator at Rank 5 Đỉnh Phong, I want a dedicated Ascension panel to appear in the cultivation UI, so that I can attempt the Immortal Ascension ritual when I meet the prerequisites.

#### Acceptance Criteria

1. WHEN a player has `rank` = 5 and `sub_rank` = 4, THE Aperture_UI SHALL display an "Thăng Tiên" (Ascension) section below the wall-breaking controls.
2. WHEN the Ascension section is displayed, THE Aperture_UI SHALL show the player's current `human_qi` value, the required minimum threshold, and the required `primeval_stones` cost.
3. WHEN the player meets all ascension prerequisites (`human_qi` >= threshold and `primeval_stones` >= cost), THE Aperture_UI SHALL enable the "Thăng Tiên" button.
4. WHEN the player does not meet ascension prerequisites, THE Aperture_UI SHALL display the "Thăng Tiên" button in a disabled state with a tooltip explaining which prerequisite is unmet.
5. WHEN a player clicks the "Thăng Tiên" button, THE Aperture_UI SHALL display a confirmation dialog warning about the risk of failure (near-death penalty) before calling the ascension API.
6. WHEN the ascension API call is in progress, THE Aperture_UI SHALL set `actionLoading` to true and disable all cultivation buttons.
7. WHEN the ascension API returns a success response, THE Aperture_UI SHALL deactivate the Aperture_UI wall-breaking controls and activate the Blessed_Land_UI, ensuring only one panel is active at a time.
8. WHEN the ascension API returns a failure response, THE Aperture_UI SHALL display the failure message and updated HP in the LogConsole.

---

### Requirement 9: API Security and Data Integrity

**User Story:** As a system operator, I want all cultivation API endpoints to enforce authentication, authorization, and transactional integrity, so that player data remains consistent and cannot be exploited.

#### Acceptance Criteria

1. WHEN any cultivation endpoint (`/api/cultivate/*`) is called without a valid authentication cookie, THE Cultivation_System SHALL reject the request with HTTP 401.
2. WHEN any cultivation endpoint is called by an authenticated player, THE Cultivation_System SHALL only modify the data of the authenticated player (identified by the JWT cookie), not any other player's data.
3. WHEN a cultivation action successfully modifies multiple player fields, THE Cultivation_System SHALL commit all changes atomically; IF any write within the transaction fails, THE Cultivation_System SHALL roll back all changes and leave the player's data unchanged.
4. WHEN a concurrent request attempts to modify the same player's cultivation data simultaneously, THE Cultivation_System SHALL use `SELECT ... FOR UPDATE` row-level locking to prevent race conditions.
5. IF a database transaction fails during any cultivation action, THEN THE Cultivation_System SHALL return HTTP 500 with an error message and leave the player's data unchanged.

---

### Requirement 10: Zustand Store Integration

**User Story:** As a frontend developer, I want the Zustand game store to expose cultivation action functions for ascension and tribulation, so that UI components can call them consistently with the existing pattern.

#### Acceptance Criteria

1. THE Zustand store SHALL expose an `attemptAscension` action that calls `POST /api/cultivate/ascension` and updates the `player` state on success.
2. THE Zustand store SHALL expose a `faceTribulation` action that calls `POST /api/cultivate/tribulation` and updates the `player` state on success.
3. WHEN `attemptAscension` or `faceTribulation` is called, THE Zustand store SHALL set `actionLoading` to true before the request and false after the request completes (success or failure).
4. WHEN `attemptAscension` or `faceTribulation` returns an error, THE Zustand store SHALL call `addLog` with the error message and set the `error` state.
5. THE Zustand store `Player` interface SHALL include `human_qi`, `wall_progress`, `next_tribulation_time`, `tribulation_count`, and `dao_marks` fields (these already exist and SHALL be kept consistent with the backend response shape).
