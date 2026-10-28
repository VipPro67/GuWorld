# Hệ thống Tông Môn – Tổng quan và hướng dẫn phát triển

Tài liệu này mô tả toàn bộ hệ thống tông môn hiện tại trong GuWorld, tập trung vào cấu trúc dữ liệu, luồng nghiệp vụ, API, worker, các ràng buộc, và checklist khi sửa lỗi hoặc bổ sung tính năng.

Mục tiêu:
- Làm rõ trách nhiệm của từng file và bảng dữ liệu.
- Dễ dàng tra cứu khi cần sửa lỗi, thêm tính năng hoặc refactor.
- Giữ đồng bộ giữa backend, schema, worker, API và dữ liệu game.

---

## 1. Tổng quan

Hệ thống tông môn gồm 5 nhóm chức năng chính:

1. Thành lập và quản lý tông môn
2. Gia nhập, duyệt thành viên, thăng cấp chức vụ
3. Nhiệm vụ tông môn và tiến độ cá nhân
4. Kho tàng tông môn và đóng góp tài nguyên
5. Khu mỏ/chiến tranh lãnh thổ và phòng thủ

Cấu trúc chính:
- Backend handlers: `backend/internal/handlers/sect.go`
- Backend handlers phụ: `backend/internal/handlers/sect_salary.go`, `backend/internal/handlers/sect_warfare.go`
- Schema: `backend/internal/db/schema_sects.go`
- Worker: `backend/internal/worker/sect_maintenance.go`, `backend/internal/worker/sect_mission.go`, `backend/internal/worker/sect_territory.go`
- Router: `backend/internal/server/server.go`

---

## 2. Các file quan trọng

### 2.1 Backend logic

- `backend/internal/handlers/sect.go`
  - Luồng chính: tạo tông môn, gia nhập, rời tông, duyệt đơn, thăng chức, đóng góp, kho tàng, nhiệm vụ, nâng cấp tông.
- `backend/internal/handlers/sect_salary.go`
  - API phát lương cho tông môn.
- `backend/internal/handlers/sect_warfare.go`
  - Quản lý lãnh thổ tông môn, phòng thủ và tấn công.
- `backend/internal/handlers/admin_sects.go`
  - Các API admin giám sát/điều chỉnh dữ liệu tông môn.

### 2.2 Schema & dữ liệu

- `backend/internal/db/schema_sects.go`
  - Khai báo toàn bộ bảng liên quan đến tông môn.
- `backend/internal/db/db.go`
  - Chạy `InitSchema()` và các seeding hiện có liên quan đến tông môn.

### 2.3 Worker / tick jobs

- `backend/internal/worker/sect_maintenance.go`
  - Phí duy trì tông môn, giảm cấp nếu không đủ quỹ; giải tán nếu nợ 3 ngày liên tiếp ở Cấp 1.
- `backend/internal/worker/sect_mission.go`
  - Hết hạn nhiệm vụ tông môn.
- `backend/internal/worker/sect_territory.go`
  - Tích lũy quỹ, linh liệu, điểm cống hiến từ lãnh thổ/mỏ chiếm đóng.

### 2.4 Router

- `backend/internal/server/server.go`
  - Đăng ký toàn bộ endpoint `/api/sects` và `/api/admin/sects`.

---

## 3. Sơ đồ dữ liệu chính

### 3.1 Bảng `sects`

```sql
CREATE TABLE IF NOT EXISTS sects (
  id VARCHAR(50) PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description TEXT NOT NULL,
  region VARCHAR(50) NOT NULL,
  vault_gu JSONB NOT NULL DEFAULT '[]',
  vault_materials JSONB NOT NULL DEFAULT '{}',
  vault_recipes JSONB NOT NULL DEFAULT '[]',
  funds BIGINT NOT NULL DEFAULT 0,
  level INTEGER NOT NULL DEFAULT 1,
  require_approval BOOLEAN NOT NULL DEFAULT FALSE,
  require_promotion_approval BOOLEAN NOT NULL DEFAULT FALSE,
  last_maintenance_at TIMESTAMP WITH TIME ZONE,
  last_salary_paid_at TIMESTAMP WITH TIME ZONE
);
```

Nghĩa vụ quan trọng:
- `funds`: quỹ tông môn.
- `vault_gu`, `vault_materials`, `vault_recipes`: kho riêng của tông môn.
- `level`: cấp tông môn, ảnh hưởng đến lương và hệ số lợi ích.
- `require_approval`: nếu true thì gia nhập cần duyệt.
- `require_promotion_approval`: nếu true thì thăng chức cần được phê duyệt.

### 3.2 Bảng `sect_members`

```sql
CREATE TABLE IF NOT EXISTS sect_members (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
  player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE UNIQUE,
  member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
  contribution_points BIGINT NOT NULL DEFAULT 0,
  joined_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
  is_defending BOOLEAN NOT NULL DEFAULT FALSE
);
```

Ràng buộc quan trọng:
- Mỗi người chỉ thuộc tối đa 1 tông môn.
- `member_rank` có các giá trị như: `NGOAI_MON`, `NOI_MON`, `TINH_ANH`, `CHAN_TRUYEN`, `TRUONG_LAO`, `HO_PHAP`, `CHUONG_MON`.
- `contribution_points`: điểm cống hiến dùng để tính quyền lợi, điều kiện thăng chức, đóng góp kho.
- `is_defending`: thành viên đang trú thủ tại căn cứ tông môn (dùng cho phòng thủ khi bị tấn công).

### 3.3 Bảng `sect_join_requests`

```sql
CREATE TABLE IF NOT EXISTS sect_join_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
  player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
  created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
  UNIQUE (sect_id, player_id)
);
```

- Dùng khi tông môn cần duyệt thành viên mới.
- Khi được chấp thuận, cần tạo hoặc cập nhật `sect_members`.

### 3.4 Bảng `sect_promotion_requests`

```sql
CREATE TABLE IF NOT EXISTS sect_promotion_requests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
  player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  current_rank VARCHAR(30) NOT NULL,
  target_rank VARCHAR(30) NOT NULL,
  status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
  created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
  UNIQUE (sect_id, player_id)
);
```

- Dùng cho flow thăng cấp cần xác nhận.
- Có thể bị chặn nếu `require_promotion_approval` = true.

### 3.5 Bảng `sect_missions`

```sql
CREATE TABLE IF NOT EXISTS sect_missions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sect_id VARCHAR(50) NOT NULL REFERENCES sects(id) ON DELETE CASCADE,
  title VARCHAR(100) NOT NULL,
  description TEXT NOT NULL,
  mission_type VARCHAR(30) NOT NULL,
  target_id VARCHAR(50) NOT NULL,
  target_amount INTEGER NOT NULL DEFAULT 1,
  reward_contribution INTEGER NOT NULL DEFAULT 0,
  required_member_rank VARCHAR(30) NOT NULL DEFAULT 'NGOAI_MON',
  expires_at TIMESTAMP WITHOUT TIME ZONE,
  UNIQUE (sect_id, title)
);
```

- `mission_type`: GATHER, COMBAT, REFINE, DONATE
- `target_id`: polymorphic, có thể là item/gu id tùy loại.
- `expires_at`: nhiệm vụ board có thời hạn. Worker tự xoá khi quá hạn.

### 3.6 Bảng `player_sect_missions`

```sql
CREATE TABLE IF NOT EXISTS player_sect_missions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
  mission_id UUID NOT NULL REFERENCES sect_missions(id) ON DELETE CASCADE,
  current_progress INTEGER NOT NULL DEFAULT 0,
  status VARCHAR(30) NOT NULL DEFAULT 'IN_PROGRESS',
  assigned_at TIMESTAMP WITHOUT TIME ZONE DEFAULT now(),
  expires_at TIMESTAMP WITHOUT TIME ZONE,
  UNIQUE (player_id, mission_id)
);
```

- Mỗi cá nhân nhận nhiệm vụ sẽ có 1 record riêng.
- `status` của player mission có thể là `IN_PROGRESS`, `COMPLETED`, `EXPIRED`, ... tùy logic handler.

---

## 4. Luồng nghiệp vụ chính

### 4.1 Tạo tông môn

Điểm vào: `CreateSect`

Workflow:
1. Kiểm tra người chơi đã thuộc tông chưa.
2. Validate tên tông, region, dữ liệu đầu vào.
3. Tạo `sects` mới.
4. Chèn `sect_members` với tài khoản sáng lập, rank mặc định là `CHUONG_MON` hoặc tương đương nếu logic đã quy định.
5. Khởi tạo kho tàng mặc định (JSONB rỗng / empty object).

Lưu ý:
- Không nên thêm logic tạo tông ở handler khác; nên giữ trong `sect.go`.
- Nếu sửa tên trường hoặc rank bắt buộc, cần cập nhật cả schema và flow UI.

### 4.2 Gia nhập tông môn

Điểm vào: `JoinSect`

Workflow:
1. Kiểm tra player đã có `sect_members` hay chưa.
2. Nếu `require_approval = true`:
   - Tạo `sect_join_requests` pending.
   - Chờ leader / quản lý duyệt.
3. Nếu không cần duyệt:
   - Insert trực tiếp vào `sect_members`.
4. Tránh có duplicate hoặc join trùng khi đã có request đang chờ.

Ràng buộc cần giữ:
- Unique `sect_id + player_id` trên join request.
- Unique `player_id` trên `sect_members`.

### 4.3 Duyệt đơn xin vào và thăng cấp

Điểm vào:
- `ActionJoinRequest`
- `ActionPromotionRequest`

Logic:
- Chỉ người có quyền quản lý tông môn mới xử lý.
- `canManageSect(rank)` quy định:
  - `CHUONG_MON`
  - `TRUONG_LAO`
  - `HO_PHAP`
- Duyệt/huỷ phải đồng bộ trạng thái request và cập nhật `sect_members` hoặc rank tương ứng.

### 4.4 Rời tông môn

Điểm vào: `LeaveSect`

Workflow:
1. Kiểm tra player đang ở sect nào.
2. Xoá bản ghi `sect_members`.
3. Xoá các `player_sect_missions` còn liên quan.
4. Xoá `sect_promotion_requests` của player trong tông đó.
5. Không xóa lịch sử audit log / hệ thống messages / nhóm chat nếu có.

Lưu ý:
- Nếu đang là leader, nên thêm logic chuyển quyền/kiểm tra leader trước khi xóa.
- Tránh làm mất dữ liệu audit hoặc hệ thống message không đồng bộ.

### 4.5 Nhiệm vụ tông môn

Điểm vào:
- `GetSectMissions`
- `CreateCustomMission`
- `AcceptSectMission`
- `CompleteSectMission`
- `DeleteSectMission`
- `AbandonSectMission`

Cơ chế:
- Tông có board mission (`sect_missions`).
- Mỗi thành viên nhận nhiệm vụ riêng (`player_sect_missions`).
- Tiến độ `current_progress` tăng theo thao tác game tương ứng.
- Khi hết hạn, worker tự đổi trạng thái `EXPIRED` hoặc xoá bài board mission.

Lưu ý khi sửa:
- `target_id` là polymorphic, không có FK, nên cần kiểm tra `mission_type` trước khi xử lý.
- `expires_at` phải luôn đồng bộ giữa board mission và player mission.

### 4.6 Kho tàng tông môn

Điểm vào:
- `DonateStones`
- `DonateMaterials`
- `DonateGu`
- `DonateRecipe`
- `StockSectVault`
- `BuyFromSectVault`
- `DestroySectVaultItem`
- `DonateBulk`

Cơ chế:
- `sects.vault_gu`, `vault_materials`, `vault_recipes` là JSONB.
- Khi đóng góp, cần validate item thật sự thuộc sở hữu người chơi.
- Khi mua vật phẩm từ kho, phải trừ item trong kho và cộng cho player, đồng thời cập nhật `sect_members.contribution_points` nếu logic quy định.
- `DestroySectVaultItem`: chỉ `CHUONG_MON` và `HO_PHAP` (helper `canDestroyVaultItem`), xoá cả stack (`gu` theo `gu_id+rank`, `material` theo `item_id`, `recipe` theo `recipe_id`), không hoàn quỹ.

Điểm nguy hiểm:
- JSONB luồng dữ liệu là map / array, dễ bị type mismatch.
- Khi sửa `vault_gu`, cần tính đến `hunger` của Gu và logic `normalizeVaultGuHunger`, `mergeVaultGuHunger`.

### 4.7 Phát lương tông môn

Điểm vào: `PaySectSalary`

Công thức cơ bản:
- `calculateSalary(sectLevel, memberRank)`
- `baseSalary = sectLevel * 100`
- `bonus` theo rank:
  - `NGOAI_MON: 0`
  - `NOI_MON: 50`
  - `TINH_ANH: 150`
  - `CHAN_TRUYEN: 400`
  - `TRUONG_LAO: 1000`
  - `HO_PHAP: 2000`
  - `CHUONG_MON: 5000`

Ràng buộc:
- Chỉ leader/management mới có quyền phát lương.
- Mỗi tông chỉ được phát lương tối thiểu 24h / lần.
- Tổng lương phải <= `sects.funds`.

Lưu ý sửa lỗi:
- Có bug/logic dễ gây lặp audit log nếu vòng lặp `for _, m := range members` chưa gán `pid` rõ ràng.
- Khi sửa, cần giữ nhất quán giữa `packages`, `auditlog`, và `tx.Commit()`.

### 4.8 Cấp độ / bảo trì tông môn

Worker: `sect_maintenance.go`

Workflow:
1. Duyệt toàn bộ `sects`.
2. Nếu quá 24h kể từ `last_maintenance_at`, xét phí duy trì.
3. Nếu quỹ đủ -> trừ phí, cập nhật `last_maintenance_at`, reset `unpaid_maintenance_count = 0`.
4. Nếu quỹ thiếu -> giảm cấp tông môn xuống `max(level-1, 1)`, **đồng thời kẹp `buildings`**: `main_hall` và mọi kiến trúc đang cao hơn đều bị đưa về cấp mới (hàm `clampBuildings`), cập nhật `last_maintenance_at`, reset `unpaid_maintenance_count = 0`.
5. Gửi tin nhắn cảnh báo cho leader.

Công thức phí (`computeVaultFees`):
- `Phí = Cấp × 2.000 + Số thành viên × 200 + Phí Kho Tàng`.
- Linh tài: `floor(tổng số lượng / 200)` Quỹ — đồng hạng mọi rank, **không** chiết khấu Bảo Khố.
- Cổ phàm `Σ(Rank × 10 × SL)`, Tiên Cổ `Σ(Rank × 200 × SL)`, Bí phương `Σ(Rank × 50)` — ba mục này được `DiscountVaultFee` giảm 50%-90% theo cấp Bảo Khố.

Điểm cần lưu ý:
- Tối thiểu là Cấp 1. Nếu đã ở Cấp 1 mà vẫn thiếu quỹ thì **không giáng nữa**: tăng `sects.unpaid_maintenance_count`, chốt `last_maintenance_at` và gửi tin kênh tông môn (không gửi thư "giáng cấp"). Vẫn cho phép quyên góp/nhập kho trong thời gian ân hạn.
- **Giải tán khi phá sản**: đủ `sectDisbandGraceDays = 3` ngày liên tiếp không trả được phí ở Cấp 1 (`sectShouldDisband`) → `disbandSect` xoá `sects` (FK CASCADE xoá members/vault/missions/defenders/requests), gửi thư `SECT_DISBANDED` cho từng thành viên. Không dùng kênh `sect` để thông báo vì `messages.sect_id` cascade sẽ xoá tin theo.
- Nếu UPDATE trừ phí thất bại, worker đọc lại `funds` + `last_maintenance_at` để phân biệt "đã trừ rồi" với "thiếu quỹ", tránh giáng cấp oan.
- Nếu cần thay đổi logic cấp tối thiểu, cập nhật ở worker + doc.

## 5. Rank và quyền quản lý

Hằng số rank và trật tự:

```go
switch rank {
case "NGOAI_MON": return 1
case "NOI_MON": return 2
case "TINH_ANH": return 3
case "CHAN_TRUYEN": return 4
case "TRUONG_LAO": return 5
case "HO_PHAP": return 6
case "CHUONG_MON": return 7
}
```

`canManageSect`:

```go
return rank == "CHUONG_MON" || rank == "TRUONG_LAO" || rank == "HO_PHAP"
```

Điều này ảnh hưởng trực tiếp tới:
- Duyệt đơn xin vào
- Duyệt thăng chức
- Phát lương
- Bố trí phòng thủ mỏ
- Tạo/sửa/xoá nhiệm vụ tông môn
- Đóng góp / stock vault

---

## 6. Endpoint tông môn

### 6.1 Public / cần auth

Base route: `/api/sects`

#### Danh sách API hiện tại

- `GET /api/sects` — lấy danh sách tông môn
- `POST /api/sects/:id/join` — xin vào tông môn
- `POST /api/sects/join-request/cancel` — huỷ đơn xin
- `POST /api/sects/leave` — rời tông
- `GET /api/sects/me` — lấy trạng thái tông của người chơi
- `POST /api/sects/promote` — đề xuất thăng rank
- `POST /api/sects/donate/stones` — đóng góp linh thạch
- `POST /api/sects/donate/materials` — đóng góp nguyên liệu
- `POST /api/sects/donate/gu` — đóng góp Gu
- `POST /api/sects/donate/recipe` — đóng góp công thức
- `POST /api/sects/donate/bulk` — đóng góp hàng loạt
- `POST /api/sects/vault/buy` — mua từ kho tông môn
- `GET /api/sects/missions` — lấy nhiệm vụ board
- `POST /api/sects/missions/:missionId/accept` — nhận nhiệm vụ
- `POST /api/sects/missions/:missionId/complete` — hoàn thành nhiệm vụ
- `DELETE /api/sects/missions/:missionId` — xoá nhiệm vụ (nếu có quyền)
- `DELETE /api/sects/missions/:missionId/abandon` — bỏ nhiệm vụ
- `POST /api/sects/create` — tạo tông môn
- `GET /api/sects/me/members` — thành viên tông của mình
- `POST /api/sects/members/:memberId/rank` — đổi rank
- `POST /api/sects/members/:memberId/kick` — đuổi thành viên
- `POST /api/sects/missions/create` — tạo mission tùy chỉnh
- `POST /api/sects/vault/stock` — bổ sung vật phẩm vào kho
- `POST /api/sects/vault/stock/bulk` — bổ sung nhiều item
- `POST /api/sects/upgrade` — nâng cấp tông
- `POST /api/sects/toggle-approval` — bật/tắt duyệt tự động
- `POST /api/sects/toggle-promotion-approval` — bật/tắt duyệt thăng cấp
- `POST /api/sects/pay-salary` — phát lương
- `GET /api/sects/join-requests` — xem đơn xin vào
- `POST /api/sects/join-requests/:requestId/action` — duyệt/huỷ đơn xin vào
- `GET /api/sects/promotion-requests` — xem thăng cấp
- `POST /api/sects/promotion-requests/:requestId/action` — duyệt/huỷ thăng cấp
- `GET /api/sects/mines` — xem mỏ của tông / hệ thống
- `POST /api/sects/mines/defend` — bố trí phòng thủ
- `POST /api/sects/mines/undefend` — rút phòng thủ
- `POST /api/sects/mines/attack` — tấn công mỏ của tông khác

### 6.2 Admin endpoints

Base route: `/api/admin/sects`

- `GET /api/admin/sects`
- `GET /api/admin/sects/:id`
- `PUT /api/admin/sects/:id`
- `POST /api/admin/sects/:id/members/:memberId/leader`
- `DELETE /api/admin/sects/:id/members/:memberId`
- `POST /api/admin/sects/:id/members`
- `DELETE /api/admin/sects/:id`
- `GET /api/admin/sects/:id/missions`
- `POST /api/admin/sects/:id/missions`
- `DELETE /api/admin/sects/:id/missions/:missionId`
- `POST /api/admin/sects/:id/vault/stock`
- `DELETE /api/admin/sects/:id/vault/stock`

---

## 7. Worker & scheduled jobs

### 7.1 `StartSectMaintenanceWorker`

- Interval được truyền khi khởi động service.
- Tác dụng: đánh thuế duy trì tông và xử lý giảm cấp nếu thiếu quỹ.
- Xài trực tiếp `db.QueryContext` + `db.ExecContext`.

### 7.2 `StartSectMissionWorker`

- Mỗi tick nhìn vào `player_sect_missions` và `sect_missions`.
- Hết hạn -> chuyển `status = 'EXPIRED'` và xoá mission board hết hạn.

### 7.3 `StartSectTerritoryWorker`

- Duyệt các tông môn có `last_territory_tick_at` cũ hơn 1 phút.
- Cộng Quỹ = `(Cấp_TM×2000 + Số_TV×200) × coverage(Cấp Linh Mạch)` mỗi ngày, nhân hệ số thủ hộ ($+10\%$ mỗi thủ hộ), rồi cập nhật `last_territory_tick_at`.
  - `coverage(L) = min(0.60 + 0.05×L, 1.05)`: cấp 1 = 65%, cấp 8 = 100% (hòa vốn với phí cấp + thành viên), cấp 9+ = 105%.
  - Phí Kho Tàng **không** được bù.
- Không còn cộng điểm cống hiến thụ động cho thành viên.

---

## 8. Rủi ro và bug thường gặp

### 8.1 Duplicate & unique constraint

Các bảng dễ lỗi duplicate:
- `sect_members.player_id` unique
- `sect_join_requests` unique `(sect_id, player_id)`
- `sect_promotion_requests` unique `(sect_id, player_id)`

Khi sửa các flow insert/update, luôn kiểm tra:
- `player_id` đã thuộc sect chưa.
- `pending request` đã tồn tại chưa.
- action delete/insert phải xoá phụ thuộc của player mission nếu cần.

### 8.2 JSONB null / type mismatch

Với `vault_gu`, `vault_materials`, `vault_recipes`, lưu ý:
- `[]` / `{}` không phải nil.
- `json.Unmarshal` từ `[]byte` cần kiểm tra `len(raw) > 0`.
- Tránh assert kiểu dữ liệu không đúng, ví dụ map string->int hoặc []struct.

### 8.3 Cập nhật điểm cống hiến và quỹ không đồng bộ

Luồng đóng góp hoặc thu hoạch lãnh thổ có nhiều query trong 1 transaction:
- Cập nhật `sects.funds`
- Cập nhật `sect_members.contribution_points`
- Cập nhật vault JSONB
- Xoá / thêm item player inventory

Nếu thiếu `tx.Commit()` hoặc lỗi xử lý trước khi `Commit`, dữ liệu sẽ không nhất quán.

### 8.4 Mission hết hạn nhưng player mission chưa đồng bộ

Có khoảng thời gian giữa:
- `sect_missions.expires_at` chấm dứt
- `player_sect_missions.expires_at` còn tồn tại

Nên kiểm tra cả 2 nơi khi tạo/hoàn thành nhiệm vụ.

### 8.5 Bảo trì tông / cấp tông không có threshold rõ ràng

`sect_maintenance.go` đang dùng logic khá đơn giản:
- tính phí theo `level`
- nếu thiếu quỹ thì giảm cấp theo 1 bước
- không có cơ chế khóa tông hoặc hủy quyền quản lý.

Nếu cần mở rộng ( ví dụ level max, giảm cấp nhiều hơn 1 bước, log phức tạp ), phải cập nhật cả worker và docs.

---

## 9. Checklist sửa lỗi / bổ sung tính năng

### Khi sửa một bug trong tông môn:

1. Xác định file liên quan đúng:
   - `sect.go` cho flow core
   - `sect_salary.go` cho phát lương
   - `sect_warfare.go` cho chiến tranh tông môn
   - `schema_sects.go` cho schema
   - `sect_*.go` trong worker cho scheduled logic
2. Kiểm tra bảng dữ liệu có đúng schema không.
3. Kiểm tra transaction / `tx.Commit()` / `Rollback()`.
4. Kiểm tra unique constraint / duplicated rows.
5. Kiểm tra JSONB và kiểu trả về.
6. Kiểm tra chỉ `sect_members` và `sects` bị ảnh hưởng, không làm lệch các table khác.
7. Cập nhật docs nếu thay đổi API, business rule hoặc worker.

### Khi thêm tính năng mới:

1. Xác định tính năng thuộc loại nào:
   - core sect flow
   - member management
   - mission system
   - vault & reward
   - warfare / mine
2. Thêm/đổi schema nếu cần.
3. Đăng ký route mới trong `backend/internal/server/server.go`.
4. Viết handler trong file phù hợp.
5. Cập nhật worker nếu có tính năng theo thời gian.
6. Nếu cần, cập nhật `docs/API.md` hoặc `docs/GAME_DATA.md`.
7. Kiểm tra đồng bộ frontend nếu API payload thay đổi.

---

## 10. Gợi ý khi refactor tương lai

- Không nên tách logic tông môn vào service layer quá sớm nếu chưa có pattern chung rõ ràng.
- Nên gom logic theo 3 nhóm:
  - `sect core`: join / leave / rank / approval
  - `sect economy`: vault / contribution / salary
  - `sect warfare`: mines / shield / attack
- Với mỗi file nên giữ 1 trách nhiệm rõ ràng, tránh đặt tất cả logic vào `sect.go` nếu quá lớn.
- Nếu mở rộng thêm hệ thống hàng loạt, nên cân nhắc tạo helper chung cho:
  - `getCurrentSectOfPlayer`
  - `requireSectPermission`
  - `validateMemberRankTransition`
  - `applySectContribution`

---

## 11. Tóm tắt nhanh

Hệ thống tông môn hiện tại đang hoạt động theo mô hình:
- Tông là entity chính lưu kho, quỹ, cấp độ, quyền duyệt.
- Thành viên thuộc 1 tông và có rank.
- Nghề nghiệp / chuyển cấp / nhiệm vụ / lãnh thổ / ngân quỹ đều được tính bằng query + transaction và worker.
- Dữ liệu phụ thuộc chặt chẽ vào `sects`, `sect_members`, và `sect_missions`, `player_sect_missions`.
- Khi cần sửa lỗi, luôn kiểm tra đồng bộ giữa schema, handler, route, worker và frontend payload.

---

## 12. Đường dẫn tham khảo nhanh

- `backend/internal/handlers/sect.go`
- `backend/internal/handlers/sect_salary.go`
- `backend/internal/handlers/sect_warfare.go`
- `backend/internal/db/schema_sects.go`
- `backend/internal/worker/sect_maintenance.go`
- `backend/internal/worker/sect_mission.go`
- `backend/internal/worker/sect_territory.go`
- `backend/internal/server/server.go`

---

## 13. Lưu ý cập nhật khi thay đổi logic hệ thống

Khi có thay đổi về API, nghiệp vụ hay bảng dữ liệu của tông môn, cần cập nhật đồng bộ:
- `docs/API.md`
- `docs/GAME_DATA.md`
- `WIKI.md` nếu người chơi thấy trực tiếp
- `README.md` nếu ảnh hưởng startup hoặc dev flow
- Bản tài liệu này (`docs/SECT_SYSTEM.md`) để giữ nguyên lịch sử kiến thức

---

## 14. Mục tiêu phát triển tiếp theo

Các phần có khả năng phát triển tiếp trong tương lai:
- hệ thống thành tích tông môn
- danh sách leaderboards theo cống hiến
- mission tier / daily quest theo cấp tông
- hệ thống guild war / chiến tranh tông lớn hơn
- auto-balance cho phát lương, phí duy trì, mining output
- dashboard admin trực quan cho sect management

Tài liệu này nên được cập nhật mỗi khi sửa hoặc bổ sung rule mới.
