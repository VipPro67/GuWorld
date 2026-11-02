# WT-313 — Wire Protocol v2 (Binary, Datagram, Versioning)

> Ticket: [Spec: wire protocol v2 (binary, datagram, versioning)](https://github.com/VipPro67/GuWorld/issues/313) · Map: [Map: WebTransport thay thế hoàn toàn WebSocket (bỏ Caddy, TLS/QUIC trong Go)](https://github.com/VipPro67/GuWorld/issues/303)
> Ngày chốt: 2026-11-02 · Phụ thuộc: [Chốt mô hình session & kênh realtime](https://github.com/VipPro67/GuWorld/issues/311)
> Phạm vi: định dạng byte trên đường truyền. Tần số snapshot cuối cùng do [Spec: zone sim phát snapshot + tần số](https://github.com/VipPro67/GuWorld/issues/314) quyết; spec này cho bảng byte/s ở 20/30/60 Hz để ticket đó chọn.

## 0. Nguyên tắc chung

| Hạng mục | Quy ước |
|---|---|
| Endianness | **Little-endian** cho mọi số cố định (`binary.LittleEndian` / `DataView(..., true)`) |
| Varint | **Unsigned LEB128** cho `seq`, `tick`, `event_id`, `hp/mana`…; giá trị âm không dùng (dùng zigzag riêng nếu cần) |
| Chuỗi | `u8` hoặc `u16` độ dài **tính bằng byte UTF-8**, theo sau là bytes; không NUL terminator |
| Cờ (flags) | Bitfield `u8`/`u16`; bit không định nghĩa **phải bằng 0**, decoder bỏ qua nhưng không lỗi |
| Magic | `"GWT"` = `0x47 0x57 0x54` |
| Version | `2` (`u8`) |
| Float trên wire | Chỉ dùng cho tọa độ đã lượng tử (số nguyên) và `movement_speed` (`u16` × 0.01). Không có `float64` trong hot path |
| Encode | Tái sử dụng `[]byte`/`ArrayBuffer` scratch; không cấp phát per-entity trong tick loop |

**Không** còn JSON trên đường truyền cho `world.*`. Ngoại lệ có chủ đích: `payload` của `world.events` và blob `skills` trong `world.welcome` vẫn là JSON bytes (xem §5.6, §5.2) — hai chỗ tần suất thấp và schema payload do gamefx tự do định nghĩa.

---

## 1. Khung Stream `0x01` (`world`, reliable)

Khi client mở Bidirectional Stream và ghi byte đầu tiên `0x01` (`StreamType = world`), handshake tiếp theo:

```
[0x01]                          StreamType (do #311 quy định, nằm ngoài frame)
"GWT"                           magic 3 byte
u8   version = 2
```

Sau handshake, mọi frame có dạng:

```
u32  frame_len                  // độ dài phần opcode+payload, KHÔNG gồm chính nó
u8   opcode
...  payload
```

Giới hạn: `frame_len ≤ 1 MiB` (bằng `maxMessageSize` WS cũ). Frame lỗi → đóng stream + session `4002`.

### 1.1 Bảng opcode Stream `0x01`

| Opcode | Tên | Hướng | Payload |
|---|---|---|---|
| `0x01` | `JOIN` | C→S | §5.1 |
| `0x02` | `WELCOME` | S→C | §5.2 |
| `0x03` | `LEAVE` | C→S | rỗng |
| `0x04` | `TRANSFER` | S→C | §5.10 |
| `0x05` | `REJECT` | S→C | §5.11 |
| `0x06` | `EVENTS` | S→C | §5.6 |
| `0x07` | `SNAPSHOT` | S→C | Baseline/full hoặc delta tràn khung (§5.7) |
| `0x08` | `SPAWN` | S→C | §5.4 (entity vào AOI giữa baseline) |
| `0x09` | `MOVE_TARGET` | C→S | §5.5 |
| `0x0A` | `CAST` | C→S | §5.5 |
| `0x0B` | `ATTACK` | C→S | §5.5 |
| `0x0C` | `CAPTURE` | C→S | §5.5 |
| `0x0D` | `COMPANION_STANCE` | C→S | §5.5 |
| `0x0E` | `RESYNC` | C→S | `u8 reason_len, reason` (client xin baseline mới) |
| `0x0F` | `PONG`/`KEEPALIVE` | S→C | `u64 server_ms` (dùng cho RTT đo chủ động; tuỳ chọn) |

---

## 2. Khung Datagram

Mỗi QUIC Datagram bắt đầu bằng 1 byte header:

```
bits 7..6 = version (2)
bits 5..0 = opcode
```

| Opcode | Tên | Hướng | Payload |
|---|---|---|---|
| `0x01` | `INPUT` | C→S | §5.3 |
| `0x02` | `SNAPSHOT_DELTA` | S→C | §5.7 |
| `0x03` | `PING` | C→S | `u64 client_ms` (RTT đo chủ động) |

**Quy tắc kích thước:** trước khi ghi, cả hai phía **bắt buộc** kiểm tra `encoded_len <= datagrams.maxDatagramSize` (server: `quic.Config` negotiated size; client: `transport.datagrams.maxDatagramSize`, Safari hard-code **1024**). Vượt ngưỡng:
- Client `INPUT`: không xảy ra (gói cố định ≤ 12 B).
- Server `SNAPSHOT_DELTA`: **tự động tràn sang Stream `0x01`** dưới dạng frame `SNAPSHOT` (§5.7). Không bao giờ `write` datagram quá cỡ (browser bỏ im lặng).

---

## 3. `net_id` — định danh entity trên wire

- Server cấp `net_id: uint16` **trong phạm vi một zone**, tuần tự tăng, **bỏ qua** id đang sống khi wrap qua `65535`; `0` dành riêng cho "không có mục tiêu".
- Mọi tham chiếu entity trên wire (`snapshot`, `spawn`, `events`, input action) dùng `net_id`, **không** dùng `entity_id` string (UUID 36 B).
- **Client xoá toàn bộ bảng ánh xạ** `net_id ↔ entity_id` khi:
  1. nhận `WELCOME` (baseline đầu tiên),
  2. nhận frame `SNAPSHOT` có `is_baseline = 1`,
  3. nhận `TRANSFER` (đổi zone).
- Bảng ánh xạ metadata (`entity_id`, `name`, `template_id`, `kind`, `party_id`, `pvp`, `radius`) đến từ:
  - baseline `SNAPSHOT` (§5.7) — toàn bộ entity đang thấy, hoặc
  - `SPAWN` (§5.4) — entity mới vào AOI giữa hai baseline.
- Entity rời AOI: xuất hiện trong danh sách `removed` của delta/baseline (§5.7), **lặp 3 delta liên tiếp** để chống rớt gói.

---

## 4. Bản ghi Entity (dùng chung cho SPAWN / SNAPSHOT)

### 4.1 `SPAWN` record (metadata đầy đủ, gửi 1 lần)

```
u16   net_id
u16   entity_id_len + bytes        // UUID 36 B
u16   name_len + bytes
u16   template_id_len + bytes
u8    kind                          // 0=player, 1=monster, 2=companion
u16   party_id_len + bytes          // 0 = không party
u8    flags                         // bit0 pvp, bit1 stealthed, bit2 flying, bit3 form_active
u8    radius_x10                    // bán kính × 10, 0..25.5
uvarint max_hp
uvarint max_mana
uvarint max_thought
u16   owner_net_id                  // 0 nếu không companion
u8    companion_kind_len + bytes    // 0 nếu không companion
u8    stance                        // 0=DEFEND, 1=FOCUS, 2=FOLLOW (companion)
```

Ước tính: **~95–130 B**/entity (UUID + tên tiếng Việt + template).

### 4.2 Delta record (mỗi tick)

```
u16   net_id
u8    mask                          // bitfield bên dưới
[u8   ext_mask]                     // chỉ khi mask.bit7 = 1
...   các trường có bit tương ứng, theo thứ tự cột dưới
```

| Mask bit | Trường | Encoding |
|---|---|---|
| 0 | position | `u16 x, u16 y` — lượng tử `1/16` unit, phạm vi `0..4095.94`, sai số ±0.031 |
| 1 | vitals | `u8 sub_mask` (bit0 hp, bit1 shield, bit2 mana, bit3 thought) + `uvarint` cho từng cái |
| 2 | revision | `uvarint entity_revision` |
| 3 | statuses | `u8 count` + mỗi status: `u8 status_id` (bảng enum intern ở baseline) |
| 4 | cast | `u8 skill_slot` (0 = không cast) + `u32 cast_end_tick` + `u16 cast_windup_ticks` |
| 5 | flags | `u8` (bit0 stealthed, bit1 flying, bit2 form_active, bit3 pvp) |
| 6 | transformation | `u8 form_kind` + `u16 form_net_id` (intern) + `u16 model_scale_x100` + `u16 collision_radius_x10` + `uvarint current_form_hp` + `u8 flags` |
| 7 | extension | `u8 ext_mask`, xem bảng dưới |

`ext_mask`:

| Bit | Trường | Encoding |
|---|---|---|
| 0 | ai_state | `u8 enum` |
| 1 | aggro | `u16 net_id` |
| 2 | owner | `u16 net_id` |
| 3 | companion_kind | `u8 len + bytes` |
| 4 | stance | `u8` |
| 5 | radius | `u8 ×10` |
| 6 | max_hp | `uvarint` |
| 7 | max_mana | `uvarint` |

**Kích thước điển hình:**
- Chỉ di chuyển: `2 + 1 + 4 = 7 B`.
- Di chuyển + hp đổi: `7 + 1 + 3 = 11 B`.
- Chỉ đổi status: `2 + 1 + 1 + 1 = 5 B`.
- Không đổi gì: **không xuất hiện trong delta** (server chỉ gửi entity khác baseline hoặc đã đổi trong baseline window).

---

## 5. Định dạng từng message

### 5.1 `JOIN` (C→S, stream `0x01`, opcode `0x01`)

```
u8   zone_len   + bytes
u8   mapver_len + bytes
u8   content_ver_len + bytes
u32  request_id
```

### 5.2 `WELCOME` (S→C, opcode `0x02`)

```
u16  session_id_len + bytes
u8   zone_len + bytes
u8   map_ver_len + bytes
u8   content_ver_len + bytes
u16  tick_rate                  // 20/30/60
u16  movement_speed_x100
u32  server_tick
u16  entity_id_len + bytes
uvarint entity_revision
u16  baseline_id
u8   map_hash_len + bytes       // 0 nếu không có
u16  skills_json_len + bytes    // JSON array giữ nguyên schema WorldSkillInfo (một lần/join)
u8   loadout_rev_len + bytes
u32  snapshot_len + snapshot    // baseline snapshot theo §5.7 (is_baseline=1)
```

`skills` giữ JSON vì: gửi đúng 1 lần/join, schema lồng nhau nhiều (visuals, shape, mode), và client đã có parser `WorldSkillInfo`. Không ảnh hưởng ngân sách tick.

### 5.3 `INPUT` (C→S, datagram `0x01`)

```
u8   header   (version|opcode)
uvarint seq
i8   move_x_x127               // vector đã normalize, -127..127
i8   move_y_x127
```

Tổng **5–7 B**. Tần suất 10 Hz (client hiện gửi 10 Hz) → ~60 B/s. Không ACK riêng: `ack_seq` nằm trong snapshot delta; input mất gói không sao vì gói sau là vector mới nhất.

### 5.4 `SPAWN` (S→C, opcode `0x08`)

```
u8   count
count × SPAWN record (§4.1)
```
Gửi ngay khi entity vào AOI, trên stream reliable. Không chờ baseline.

### 5.5 Action commands (C→S, stream `0x01`)

```
MOVE_TARGET (0x09):  uvarint seq, u16 x, u16 y          // 1/16 unit
CAST        (0x0A):  uvarint seq, u8 slot, u8 mode,
                      u16 target_net_id, i16 aim_x_x16, i16 aim_y_x16,
                      u16 x, u16 y                        // aim tối đa ±2047
ATTACK      (0x0B):  uvarint seq, u16 target_net_id, i16 aim_x_x16, i16 aim_y_x16
CAPTURE     (0x0C):  uvarint seq, u16 target_net_id, u16 gu_id_len + bytes
STANCE      (0x0D):  uvarint seq, u8 stance, u16 focus_net_id
```

Tất cả đi stream reliable: tần suất thấp, mất 1 lệnh = lỗi gameplay rõ ràng, và tránh phải tự viết ACK/retry trên UDP. (`MOVE_TARGET` chuyển từ datagram sang stream — tinh chỉnh ánh xạ của [Chốt mô hình session & kênh realtime](https://github.com/VipPro67/GuWorld/issues/311).)

### 5.6 `EVENTS` (S→C, opcode `0x06`)

```
uvarint from_event_id
u8   count
count × {
  uvarint event_id
  uvarint tick
  u8   type_id                  // bảng enum cố định, sinh từ danh sách event type hiện có
  u16  source_net_id
  u16  target_net_id
  u16  payload_len
  bytes payload_json            // JSON của gamefx, giữ nguyên
}
```
Gộp batch mỗi tick như `world.events` hiện tại. Event type mới chưa có `type_id` → gán `0` (`UNKNOWN`) và client fallback đọc `payload_json` như cũ, kèm log debug; không chặn trận đấu.

### 5.7 `SNAPSHOT` (baseline/full: stream `0x07`; delta: datagram `0x02`)

**Baseline / full / delta-tràn-khung** (stream `0x07`):

```
u32  tick
u16  baseline_id
u32  base_tick
uvarint ack_seq
u8   is_baseline               // 1 = baseline
u16  entity_count
entity_count × SPAWN record (§4.1)     // baseline/full gửi metadata đầy đủ
u16  removed_count
removed_count × u16 net_id
u8   projectile_count
projectile_count × {
  u16 proj_id
  u16 source_net_id
  u16 skill_id_len + bytes
  u16 vfx_len + bytes
  u16 x, u16 y
  u8  radius_x10
}
```

**Delta** (datagram `0x02`):

```
u8   header (version|opcode)
u32  tick
u16  baseline_id
uvarint ack_seq
u8   flags                     // bit0 has_projectiles, bit1 is_baseline_hint (luôn 0 trên datagram)
u8   entity_count
entity_count × delta record (§4.2)
u8   removed_count
removed_count × u16 net_id
[projectiles giống trên nếu flags.bit0 = 1]
```

**Quy tắc:**
- Baseline phát lúc `WELCOME` và định kỳ mỗi **40 tick (2 s @20 Hz)**, giữ nguyên `SnapshotBaselineTicks` hiện tại. Baseline luôn đi stream để không bao giờ mất.
- Delta gửi mỗi tick; nếu `encode_len > maxDatagramSize` → đóng gói lại thành frame `SNAPSHOT` trên stream (cùng layout baseline nhưng `is_baseline = 0`).
- **Rớt delta không cần resync**: delta là baseline-relative (kế thừa `BuildDelta` hiện tại), nên gói kế tiếp vẫn chứa mọi entity khác baseline. Chỉ cần baseline tới được.
- `removed` lặp trong **3 delta liên tiếp** để tránh bóng ma.
- Nếu client phát hiện `baseline_id` lạ (ví dụ bỏ lỡ baseline trên stream — không xảy ra với reliable, nhưng phòng lỗi), gửi `RESYNC`; server phát baseline mới ngay.

### 5.8 `TRANSFER` (S→C, opcode `0x04`)

```
u16 from_zone_len + bytes
u16 to_zone_len + bytes
u16 transfer_id_len + bytes
u16 target_x, u16 target_y        // 1/16 unit, map-local
u32 legacy_x_x100, u32 legacy_y_x100
uvarint hp
```

### 5.9 `REJECT` (S→C, opcode `0x05`)

```
u16 code_len + bytes              // ví dụ "realtime_character_already_owned"
u16 msg_len + bytes               // có thể rỗng
u8  flags                          // bit0 retryable
```

### 5.10 `RESYNC` (C→S, opcode `0x0E`)

```
u8 reason_len + bytes
```

---

## 6. Versioning & tương thích

- Stream `0x01`: magic `GWT` + `version = 2` ngay sau byte `StreamType`. Lệch magic/version → server gửi `REJECT{code:"protocol_version_mismatch", retryable:false}` rồi đóng session bằng `CloseWithError(4002, "protocol_version_mismatch")`; client hiện màn hình "Bản game đã cũ, tải lại trang".
- Datagram: `version` 2 bit. Gói version khác **bỏ qua im lặng** (có thể là gói cũ đang bay trên mạng).
- Không fallback JSON, không query `?codec=`. Một wire protocol duy nhất.
- Sửa layout phá vỡ tương thích → tăng `version` lên 3 và triển khai cùng lúc backend + client (big-bang theo [Chốt kế hoạch rollout & rollback khi xoá WS](https://github.com/VipPro67/GuWorld/issues/309)).
- Thay đổi **không phá vỡ** (thêm `type_id` event, thêm bit mask còn trống) → giữ version 2, decoder cũ bỏ qua bit lạ.

---

## 7. Ước tính byte/s (đối chiếu JSON hiện tại)

Giả định mô hình (sẽ được đo lại bởi [Đo baseline realtime trước khi đổi transport](https://github.com/VipPro67/GuWorld/issues/308) và [Prototype: spike WebTransport end-to-end Go ↔ browser](https://github.com/VipPro67/GuWorld/issues/310)):

| Zone | Entity trong AOI (V) | Entity di chuyển mỗi tick (K) |
|---|---|---|
| 5 người | 8 | 4 |
| 20 người | 15 | 8 |
| 50 người | 25 | 14 |

Thành phần mỗi giây (một client):
- Delta: `Hz × (16 + 7K + 2×projectiles)`.
- Baseline: `V × 110 B / 2 s`.
- Spawn/metadata: ~100 B/s.
- Events: ~300 B/s (5 event/s × 60 B).
- Input (upstream): ~60 B/s.

### 7.1 Downstream (KB/s, mỗi client)

| Zone | v2 @20 Hz | v2 @30 Hz | v2 @60 Hz | JSON hiện tại @20 Hz | Giảm |
|---|---|---|---|---|---|
| 5 người | **~1.9** | ~2.4 | ~3.0 | ~35 | ~18× |
| 20 người | **~2.8** | ~3.6 | ~4.5 | ~69 | ~25× |
| 50 người | **~4.1** | ~5.4 | ~6.7 | ~119 | ~29× |

### 7.2 Upstream (client → server)

| Loại | v2 | JSON hiện tại |
|---|---|---|
| Input 10 Hz | ~0.06 KB/s | ~2.5 KB/s |
| Lệnh combat/click | ~0.02 KB/s | ~0.1 KB/s |
| Tổng | **~0.08 KB/s** | **~2.6 KB/s** |

### 7.3 Tải server mỗi zone (fan-out downstream)

| Zone | v2 @20 Hz | v2 @60 Hz |
|---|---|---|
| 5 người | ~9.5 KB/s (76 kbps) | ~15 KB/s |
| 20 người | ~56 KB/s (450 kbps) | ~90 KB/s |
| 50 người | ~205 KB/s (1.6 Mbps) | ~335 KB/s |

Kết luận cho [Spec: zone sim phát snapshot + tần số](https://github.com/VipPro67/GuWorld/issues/314): cả 60 Hz vẫn nằm trong ngân sách băng thông của một Lightsail box; quyết định tần số nên dựa trên **CPU encode + p99 tick**, không phải băng thông.

Kiểm tra trần datagram: delta 1024 B chứa được `(1024 − 16) / 7 ≈ 144` entity di chuyển; AOI thực tế tối đa ~30–40 → dư địa lớn, tràn stream gần như chỉ xảy ra với baseline.

---

## 8. Nơi đặt code & chiến lược test

| Phần | Vị trí đề xuất |
|---|---|
| Go codec | `backend/internal/realtime/protocol/v2/` (encoder, decoder, enum `type_id`, `net_id` allocator) |
| TS codec | `frontend/src/game/network/v2/` (`Codec.ts`, `Datagram.ts`, `NetIdTable.ts`) |
| Golden vectors | `backend/internal/realtime/protocol/v2/testdata/*.hex` — cùng file được test Go lẫn test TS (vitest) để chống lệch hai implementation |
| Fuzz | Fuzz decoder Go (không panic, không cấp phát vô hạn) như `protocol/fuzz_test.go` hiện có |
| Đo kích thước | Benchmark encode `Snapshot` 5/20/50 entity, so với `json.Marshal` để xác nhận bảng §7 |

**Điều kiện hoàn thành khi thi công:** golden vector Go ↔ TS khớp 100%; fuzz 60 s không panic; benchmark xác nhận sai số bảng §7 < 30%; `MaxDatagramSize` guard có unit test cho trường hợp 1024 và 1211.

---

## 9. Việc chuyển giao cho ticket khác

- **Tần số snapshot & redundancy** → [Spec: zone sim phát snapshot + tần số](https://github.com/VipPro67/GuWorld/issues/314).
- **Client decode, input replay, interpolation** → [Spec: client rewrite sang WebTransport](https://github.com/VipPro67/GuWorld/issues/316).
- **UX khi lệch version / không hỗ trợ WT** → [Chốt UX cho client không hỗ trợ WebTransport](https://github.com/VipPro67/GuWorld/issues/312).
- **Resume token sau mất mạng** → [Spec: fast reconnect & session resume khi mạng đổi](https://github.com/VipPro67/GuWorld/issues/318).
- **Binary hóa payload event (thay JSON)** → để sau khi v2 chạy ổn; ghi vào fog của map.
