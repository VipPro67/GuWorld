# Nghiên cứu Kiến trúc Go WebSocket Game Server cho 2D MMORPG Web (Spatial AOI, Tick Loop, Packet Protocol)

- **Mã Ticket:** #243 (Part of Epic Map #241)
- **Tác giả:** GuWorld Architecture Research Team
- **Mục tiêu:** Thiết lập kiến trúc server-authoritative cho game 2D Top-Down Pixel MMORPG Web (Phaser 3/4 Client + Next.js HUD Overlay + Go Game Server), phục vụ 100–500 CCU trên cùng một zone với độ trễ thấp và độ tin cậy cao.

---

## 1. Bối cảnh & Yêu cầu Kỹ thuật

Trong mô hình 2D MMORPG/ARPG thời gian thực (tương tự phong cách *Cổ Tiên Lộ*):
1. **Viewport & Phân giải:** Màn hình Web chuẩn 1280x720 hoặc 1920x1080 (Full HD). Nhân vật di chuyển liên tục trên tilemap 2D (kích thước tile 32x32 px hoặc 16x16 px).
2. **Authoritative Server:** Client chỉ gửi input (hướng di chuyển vector/joystick, lệnh dùng skill định hướng skillshot, lướt né dodge/dash). Server tính toán vật lý, kiểm tra va chạm địa hình, trừ mana/khí hải, kiểm tra hồi chiêu (cooldown), tính toán sát thương hitbox và đồng bộ snapshot về client.
3. **Hiệu năng & Băng thông:** Với 100–500 CCU trên một bản đồ, phát tán broadcast toàn cục (O(N²)) sẽ làm sập server và nghẽn băng thông client. Cần cơ chế **Area of Interest (AOI)** để giới hạn broadcast trong phạm vi màn hình/tầm nhìn (O(K), K ≪ N).
4. **Độ mượt mà:** Tick loop server chạy ở 20–30 Hz (33.33ms – 50ms/tick). Client chạy ở 60 FPS và sử dụng Linear Interpolation (LERP) để khử giật lag.

---

## 2. So sánh Thư viện WebSocket: `coder/websocket` vs `gorilla/websocket`

### 2.1 Bảng so sánh chi tiết

| Tiêu chí | `github.com/gorilla/websocket` | `github.com/coder/websocket` (nhooyr.io) | Phân tích cho Game MMO |
| :--- | :--- | :--- | :--- |
| **Trạng thái dự án** | Từng bị archive, hiện cộng đồng duy trì | Tích cực bảo trì bởi Coder, hiện đại | `coder` dùng chuẩn Go hiện đại hơn |
| **Tích hợp `context.Context`** | **Không** (chỉ có `SetReadDeadline`, `SetWriteDeadline`) | **Hỗ trợ First-Class** (`Read(ctx)`, `Write(ctx)`) | Cực kỳ quan trọng: `coder` cho phép hủy connection ngay lập tức khi context của player session/server shutdown bị cancel |
| **Quản lý Bộ nhớ & GC** | Cấp phát buffer truyền thống, tốn allocations | Tích hợp sẵn `sync.Pool` và buffer pooling | `coder` giảm 30–40% áp lực Garbage Collection khi hàng trăm client gửi/nhận packet liên tục |
| **Mô hình Đồng thời (Concurrency)** | Cho phép 1 reader + 1 writer song song. Write không thread-safe nếu không có lock | Tương tự, nhưng tích hợp context-aware writer | Cả 2 đều cần kênh hàng đợi (message queue/channel) cho writer pump |
| **Hỗ trợ Chuẩn Web** | Cần cấu hình handshake thủ công | Đơn giản, zero-dependency, tuân thủ RFC 6455 | `coder` tinh gọn, dễ tích hợp với Gin/Chi/Standard `http.Handler` |
| **WASM Client** | Không tối ưu cho browser WASM | Hỗ trợ biên dịch cả target WASM (nếu cần) | Linh hoạt nếu sau này dùng Go trên client |

### 2.2 Đánh giá & Khuyến nghị
- **Khuyến nghị lựa chọn:** **`github.com/coder/websocket`** là lựa chọn vượt trội cho Game Server Go hiện đại:
  - Hỗ trợ `context.Context` xuyên suốt từ HTTP handshake đến từng frame Read/Write, giúp dọn dẹp kết nối mồ côi (ghost connection) triệt để.
  - Khả năng quản lý bộ nhớ thông qua buffer pool giúp giảm thiểu GC pauses – yếu tố sống còn tránh giật tick rate trong game server.
  - API tinh gọn, ít boilerplate code hơn so với `gorilla/websocket`.

---

## 3. Thuật toán Phân Vùng Không Gian (Spatial Hash Grid AOI)

### 3.1 Phân tích Kích Thước Cell (Cell Size)
- Viewport màn hình Full HD: `1920 x 1080` px. Camera zoom thông thường của game 2D Pixel bao quát khoảng `960 x 540` đến `1280 x 720` px.
- Nếu chọn **Cell 320 x 320 px**:
  - Vùng lân cận 3x3 cells (9 cells) có kích thước: `960 x 960` px.
  - Ưu điểm: Phân vùng nhỏ, rất ít lãng phí băng thông gửi cho entity ngoài màn hình.
  - Nhược điểm: Tần suất entity chuyển đổi giữa các cell (cell crossing) diễn ra rất nhanh khi nhân vật di chuyển (ví dụ tốc độ chạy 200px/s chỉ mất 1.5s là đổi cell), làm tăng tải tính toán diff Enter/Leave.
- Nếu chọn **Cell 640 x 640 px**:
  - Vùng lân cận 3x3 cells (9 cells) có kích thước: `1920 x 1920` px.
  - Bao phủ trọn vẹn màn hình 1080p ngay cả khi camera rung lắc hoặc zoom rộng.
  - Tần suất chuyển đổi cell giảm đi một nửa so với 320px.
- **Quyết định đề xuất:** Sử dụng **Cell Size = 640 px** (hoặc 480 px tùy theo kích thước tilemap). Đây là tỷ lệ vàng giữa tần suất chuyển cell và phạm vi hiển thị tối ưu cho game MMO 2D.

### 3.2 Thuật toán 9-Cell Neighborhood & Biên Chuyển Đổi (Cell Crossing)

```
[TL] [ T ] [TR]
[ L ] [ C ] [ R ]   <-- 9-Cell Area of Interest (AOI) của entity tại Center [C]
[BL] [ B ] [BR]
```

Mỗi cell được định danh bởi cặp tọa độ nguyên:
$$cx = \lfloor \frac{x}{\text{cellSize}} \rfloor, \quad cy = \lfloor \frac{y}{\text{cellSize}} \rfloor$$

Khi Entity $E$ di chuyển từ cell $C_{old}$ sang cell $C_{new}$:
1. Tập 9 cell cũ: $S_{old} = \text{GetNeighborhood}(C_{old})$
2. Tập 9 cell mới: $S_{new} = \text{GetNeighborhood}(C_{new})$
3. **Tập Cell Rời Khỏi (Leave):** $S_{leave} = S_{old} \setminus S_{new}$
   - Tất cả người chơi trong $S_{leave}$ nhận gói tin: `ENTITY_LEAVE` (Despawn $E$).
   - Nếu $E$ là Player, $E$ nhận `ENTITY_LEAVE` cho tất cả các entity nằm trong $S_{leave}$.
4. **Tập Cell Gia Nhập (Enter):** $S_{enter} = S_{new} \setminus S_{old}$
   - Tất cả người chơi trong $S_{enter}$ nhận gói tin: `ENTITY_ENTER` (Spawn $E$ kèm full profile/vị trí/HP/hướng quay).
   - Nếu $E$ là Player, $E$ nhận `ENTITY_ENTER` cho tất cả các entity nằm trong $S_{enter}$.
5. **Tập Cell Giữ Nguyên (Stay):** $S_{stay} = S_{old} \cap S_{new}$
   - Nhận gói tin cập nhật vị trí nhẹ `ENTITY_MOVE` trong tick loop định kỳ.

Độ phức tạp:
- Tìm kiếm lân cận: $O(1)$ thông qua Hash Map key `(cx, cy)`.
- Broadcast: $O(K)$ với $K$ là số người chơi trong 9 cell lân cận, loại bỏ hoàn toàn bài toán $O(N^2)$ của toàn server.

---

## 4. Thiết Kế Game Loop Ticker (Simulation Engine)

### 4.1 Tần số Tick & Chu trình 6 Pha (6-Phase Tick Pipeline)
Chọn **20 Hz (50ms/tick)** làm tần số chuẩn (hoặc 30 Hz / 33.33ms nếu đòi hỏi phản xạ combat eSports). Chu kỳ 50ms đủ phản xạ nhanh cho Action RPG trên nền Web mà không gây bão packet lên mạng di động/WiFi.

Trong mỗi chu kỳ Tick, Game Loop chạy tuần tự 6 pha server-authoritative:

```
┌─────────────────────────────────────────────────────────────┐
│ 1. INGEST INPUTS (Lấy lệnh di chuyển, skill từ queue)       │
├─────────────────────────────────────────────────────────────┤
│ 2. PLAYER SIMULATION (Tính vector vận tốc, va chạm static)  │
├─────────────────────────────────────────────────────────────┤
│ 3. MONSTER AI (Aggro detection, di chuyển quái, đánh cơ bản)│
├─────────────────────────────────────────────────────────────┤
│ 4. PROJECTILES & COMBAT (Bay đạn, va chạm hitbox, sát thương)│
├─────────────────────────────────────────────────────────────┤
│ 5. TIMERS & COOLDOWNS (Giảm CD kỹ năng, hiệu ứng buff/dot)   │
├─────────────────────────────────────────────────────────────┤
│ 6. SPATIAL FLUSH (Tạo snapshot delta theo 9-cell & gửi WS)  │
└─────────────────────────────────────────────────────────────┘
```

### 4.2 Xử lý Tick Overrun, Drift Compensation & Panic Safety
- **Drift Compensation:** Sử dụng `time.NewTicker` kết hợp đo thời gian thực tế `time.Since(tickStart)`. Nếu thời gian xử lý vượt quá 50ms, ghi nhận warning `Tick Overrun` và điều chỉnh bù trừ.
- **Panic Safety:** Toàn bộ code trong tick loop phải được bọc trong hàm bẫy `defer recover()` (tương tự pattern `worker.SafeRun` trong `backend/internal/worker/runner.go`), đảm bảo một bug logic của 1 monster hoặc 1 skillshot không bao giờ làm crash tiến trình Go server.

---

## 5. Thiết Kế Packet Protocol & Đồng Bộ Trạng Thái

### 5.1 So sánh JSON Rút Gọn vs Binary Protocol

| Tiêu chuẩn | JSON Rút Gọn (Compact Array/Struct) | Protocol Buffers (proto3) | Custom BytePack (ArrayBuffer) |
| :--- | :--- | :--- | :--- |
| **Kích thước gói tin** | Trung bình (~60–90 bytes/move) | Rất nhỏ (~12–18 bytes/move) | Siêu nhỏ (~8–14 bytes/move) |
| **Tốc độ mã hóa / giải mã** | Vừa phải (Go `json.Marshal` / JS `JSON.parse`) | Rất nhanh (Native code / WASM) | Cực nhanh (Direct typed memory view) |
| **Áp lực GC trên Server** | Tạo nhiều string/interface allocation | Thấp (Tái sử dụng struct buffer) | Rất thấp (Zero-allocation byte pool) |
| **Công thái học khi Debug** | **Tuyệt vời**: Đọc trực tiếp trên Chrome DevTools WS tab | Khó: Dữ liệu nhị phân cần decode | Khó: Raw hex bytes |
| **Khả năng thay đổi schema** | Dễ dàng thêm/bớt trường | Rất chặt chẽ qua file `.proto` | Thủ công, dễ lỗi lệch offset byte |

### 5.2 Chiến lược Đề xuất: Hybrid Evolutionary Protocol
1. **Giai đoạn Prototype (Milestone 0.1):** Sử dụng **Compact Array JSON**:
   - Định dạng: `[Opcode, EntityID, Param1, Param2, ...]`
   - Ví dụ gói tin di chuyển: `[10, "p_101", 1250, 840, 150, 0]` (Opcode 10 = Move, ID, X, Y, Vx, Vy).
   - Dễ debug trên client Phaser, kiểm tra trực quan mà không cần dựng tool build Protobuf.
2. **Giai đoạn Production (Sau khi prototype ổn định):**
   - Chuyển đổi các gói tin hot-path tần suất cao (Move `10`, CastSkill `20`, ProjectileUpdate `25`, HitDamage `30`) sang **Protobuf** hoặc **Binary ArrayBuffer**.
   - Giữ các gói tin cold-path tần suất thấp (Chat, Auth, Quest, Mở hòm đồ) ở định dạng JSON để phát triển nhanh.

### 5.3 Opcodes Chuẩn Cho 2D Action MMO
- `1` : `C2S_HEARTBEAT_PING` / `S2C_HEARTBEAT_PONG`
- `2` : `C2S_AUTH_CONNECT` (JWT session token)
- `10`: `C2S_INPUT_MOVE` (Vector di chuyển góc angle + speed)
- `11`: `S2C_ENTITY_MOVE` (Đồng bộ vị trí X, Y, Vx, Vy, TickID)
- `12`: `S2C_ENTITY_ENTER` (Xuất hiện trong tầm nhìn - full state)
- `13`: `S2C_ENTITY_LEAVE` (Rời khỏi tầm nhìn)
- `20`: `C2S_SKILL_CAST` (ID kỹ năng, Tọa độ ngắm TargetX, TargetY)
- `21`: `S2C_SKILL_CASTED` (Hiệu ứng bắt đầu niệm / bắn đạn)
- `22`: `S2C_PROJECTILE_SPAWN` (Tạo đạn bay)
- `30`: `S2C_DAMAGE_EVENT` (TargetID, DamageAmount, IsCrit, NewHP)
- `40`: `C2S_DODGE_DASH` (Lướt né, kích hoạt iframe bất tử ngắn hạn)

---

## 6. Code Mẫu Thực Tế (Idiomatic Go Implementation)

Xem chi tiết struct `SpatialGrid`, `GameTicker` và `ClientSession` trong báo cáo nghiên cứu.

---

## 7. Kết Luận & Khuyến Nghị Kiến Trúc Cho Map #241

1. **Thư viện WebSocket:** Sử dụng `github.com/coder/websocket` để đảm bảo hỗ trợ `context.Context` trọn vẹn, dọn dẹp connection chuẩn mực và bộ đệm buffer pooling giảm tải GC.
2. **Kích thước Cell:** Đặt mặc định `640x640 px`. Dùng thuật toán tập hợp `ComputeAOIDiff` để chỉ gửi gói tin `ENTER`/`LEAVE` khi entity vượt qua ranh giới cell, loại bỏ ghost entity và giảm 85% lưu lượng mạng so với broadcast diện rộng.
3. **Game Loop:** Chạy độc lập ở tần số **20 Hz (50ms)**, bọc trong pattern `safeTick` với panic recovery và tính toán delta time $dt$ lũy kế. Client Phaser chạy 60 FPS nội suy (LERP) mượt mà.
4. **Packet Protocol:** 
   - **Phase 1 (Prototype 0.1):** Compact JSON dạng mảng số `[Opcode, ID, X, Y, ...]`.
   - **Phase 2 (Production):** Protobuf nhị phân cho các gói tin va chạm, di chuyển và combat thời gian thực.
