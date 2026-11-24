# Đặc tả: Hành trình tự động theo ô trong một Đại Vực

Trạng thái: **Đặc tả đã chốt để chuẩn bị triển khai**. Đây là thiết kế, chưa phải mô tả tính năng đang phát hành.

## Mục tiêu

Người chơi chọn một ô đích, xem tuyến và thời gian dự kiến, rồi để server di chuyển nhân vật từng ô. Hành trình tiếp tục khi người chơi offline, xử lý luật và sự kiện của mỗi ô, và đồng bộ kết quả giữa các phiên đăng nhập.

## Phạm vi

- Một hành trình hoạt động cho mỗi player.
- Điểm đầu và đích thuộc cùng Đại Vực (`NGU_VUC`) và cùng tầng thế giới. Các khu vực con có `world_map_id` riêng vẫn có thể nối với nhau bằng đường đi bộ qua mép map.
- Có thể đi vào `KHU_VUC`, `SAFE_ZONE`, `BOSS_ZONE`, `PVP_ZONE` nếu từng bước và điều kiện vào vùng cho phép.
- Không tự qua tầng, dùng portal/teleport, vào Bí Cảnh, Tiên Khiếu hoặc map instance.
- Không tự nhặt ground drop, điểm tài nguyên riêng, hay tương tác NPC/merchant/portal. Chỉ tự nhận thứ mà bước đi hiện có tự nhận (Resource/WildGu theo luật và điều kiện hiện hữu).

## Luồng người chơi

1. Trong `ExplorationMap`, bấm ô để đặt đích.
2. Server kiểm tra đích và trả tuyến dự kiến, khoảng cách/ETA cùng thông tin vùng cần thiết.
3. UI vẽ tuyến và marker đích; người chơi bấm **Bắt đầu** để xác nhận.
4. UI hiển thị vị trí, đích, ETA cập nhật, trạng thái, sự kiện gần nhất và lý do dừng. Người chơi có thể **Dừng**, đổi đích hoặc **Tiếp tục** khi trạng thái cho phép.
5. Khi mở lại hoặc reconnect, client tải snapshot hành trình. Player event/SSE cập nhật các revision mới; khi gap hoặc dữ liệu nghi ngờ, tải snapshot lại.

## Mô hình hành trình

PostgreSQL là nguồn chuẩn cho trạng thái hành trình; vị trí thực chỉ nằm ở `players.current_region`, `players.x/y`, `map_instance_id` theo `docs/architecture/PLAYER_DOMAIN.md`.

Một bản ghi hành trình cần mang tối thiểu: player id, journey id, revision/generation, đích (region/map scope và tọa độ), status, `next_step_at`, thời điểm bước đã lập lịch, trạng thái/clock cần để tính ETA, kết quả gần nhất và reason code. Có thể chuẩn hóa tên/cột trong thiết kế schema lúc triển khai. Sự kiện đáng chú ý cần lưu đủ để người chơi xem sau offline; không cần lưu từng bước đi thường thành transcript.

Trạng thái tối thiểu:

- `running`: chờ hoặc thực thi bước kế tiếp.
- `paused_manual_combat`: PvE đã vào combat thường; chờ người chơi thắng rồi bấm Tiếp tục.
- `paused_condition`: tạm thời không thể đi, ví dụ `movement <= 0`, bị trói hoặc trọng thương; tự tiếp tục khi điều kiện hết.
- `completed`: tới đích.
- `cancelled`: người chơi dừng hoặc hành trình bị thay thế bởi mutation di chuyển khác.
- `failed`: đích/đường không còn hợp lệ, player chết/thua, PvP thật, mất quyền trưởng nhóm hoặc lỗi gameplay kết thúc hành trình.

`completed`, `cancelled`, `failed` là trạng thái cuối. Một hành trình mới bắt đầu từ vị trí thực hiện tại.

## Nhịp bước và ETA

Với `movement` hiệu lực dương:

```text
stepDuration = clamp(baseStepDuration * stepLength / effectiveMovement,
                     minStepDuration,
                     maxStepDuration)
```

`stepLength = 1` cho bước ngang/dọc và `sqrt(2)` cho bước chéo. Chỉ effect tác động `movement` thay đổi tốc độ đi; `speed` dùng trong combat không ảnh hưởng. Buff/debuff/tải trang bị làm đổi `movement` áp dụng từ bước kế tiếp; thời hạn bước đang chờ được giữ nguyên. Bước đầu chờ đủ thời gian, không miễn phí.

Giá trị giây của `baseStepDuration`, `minStepDuration`, `maxStepDuration` là tham số cân bằng cần chốt trong implementation/config; Wayfinder chưa khóa chúng. Nếu `movement <= 0`, không chia và chuyển sang `paused_condition`.

ETA là ước lượng từ tuyến còn lại và tốc độ đã biết; cập nhật sau mỗi bước hoặc khi tuyến/tốc độ đổi. Không coi ETA là cam kết do encounter, thay đổi địa hình, quyền vào vùng hoặc tốc độ có thể đổi.

Không trừ mana/Chân Nguyên/Thể Lực cho bước đi; chi phí 0 khớp code hiện tại. Chi phí của encounter/sự kiện vẫn theo luật riêng.

## Tìm đường và chuyển vùng

- Server tìm đường đi bộ ngắn nhất trên lưới qua các ô được phép đi, hỗ trợ bước ngang/dọc/chéo. Không cắt góc chéo nếu cả hai ô cạnh tạo góc đều bị chặn.
- Ô chưa khám phá được xem là chưa biết, không tự coi là chướng ngại. Không gửi nội dung ẩn cho client chỉ để vẽ tuyến.
- Khi tới gần phát hiện chướng ngại hoặc quyền vào vùng đổi, tính lại từ vị trí đã commit. Nếu hết đường/đích không hợp lệ, dừng tại vị trí cuối cùng và báo lý do.
- Qua ranh giới map con bằng lối đi bộ hiện có; xử lý mỗi vùng thành một đoạn. Sau edge transition và đổi tọa độ, tính đoạn tiếp theo từ tọa độ/vùng mới. Không tự dùng portal/teleport.
- Mỗi bước vẫn chạy guard/arrival flow hiện có để giữ region entry, trigger, mission progress, party sync và publish semantics. Tuyến server chỉ là đề xuất đường; client không quyết định bước hợp lệ.

## Trigger, combat và vật phẩm

- Thực thi luật bước hiện hành tại từng ô: khám phá, Resource/WildGu, quái/boss, ambush, kiểm tra vùng và các trigger đã có.
- PvE: nếu `combat_settings.skip_battle` bật thì auto-resolve theo flow hiện hữu. Thắng thì giữ thưởng/loot/kết quả combat và tiếp tục từ vị trí thực. Thua/hồi sinh thì `failed`.
- PvE khi `skip_battle` tắt: `paused_manual_combat`; sau khi người chơi thắng, không tự chạy tiếp. Người chơi phải bấm **Tiếp tục**.
- PvP thật: không gửi lệnh tấn công chủ động và không auto-resolve; kết thúc tự đi, chuyển combat thủ công. Khi triển khai phải phân biệt PvP thật trước nhánh auto-resolve tổng quát hiện có.
- Tương tác cần thao tác riêng không tự chạy khi đi ngang hoặc tới đích. Túi đầy, quyền nhặt và ô thay đổi tuân thủ luật hiện hữu.

## Offline, restart và tuần tự hóa

- Offline không dừng hành trình. Worker chạy trên server.
- Sau restart, tính bù từng bước theo thứ tự lịch sử, không teleport đến tọa độ suy ra. Giới hạn số bước xử lý mỗi lượt worker; lượt sau tiếp tục. Dừng catch-up tại combat cần người chơi, trạng thái pause, điều kiện thất bại hoặc không còn đường.
- Bước đã lên lịch giữ thời hạn; tốc độ có hạn tính theo thời điểm dự kiến của từng bước. Buff hết hạn trong thời gian server tắt chỉ ảnh hưởng các bước xảy ra trước hạn.
- Nội dung world/điều kiện được kiểm tra khi xử lý bước; transaction và claim đảm bảo một bước chỉ commit một lần.
- Bắt đầu, đổi đích, dừng, pause/resume và commit bước được tuần tự hóa với player row và journey revision/CAS. Nhiều backend có thể cùng poll nhưng chỉ một worker được claim/commit một bước. Mọi event phát sau commit.
- Dừng hoặc đổi đích hủy bước đang chờ; đích mới tính đủ thời gian bước đầu từ vị trí đã commit. Mọi movement khác thành công (đi tay, portal, teleport, party drag, auto farm) kết thúc hành trình cũ cùng mutation vị trí. Lệnh movement bị từ chối mà vị trí không đổi không hủy hành trình.
- Trưởng nhóm có thể khởi hành và kéo party theo luật hiện có. Mất quyền trưởng nhóm kết thúc hành trình; member bị kéo có journey riêng thì journey đó kết thúc.
- Điều kiện tạm thời làm pause và tự resume khi hết. Chết/thua, PvP thật, mất quyền trưởng nhóm, đích/đường bất hợp lệ là kết thúc có lý do.

## API và event contract

Đường dẫn cụ thể thuộc thiết kế API khi triển khai; cần các thao tác server-authoritative sau:

- `preview`: nhận đích; trả validation, tuyến/đoạn tuyến phù hợp, distance/ETA và journey revision hiện hành.
- `start` hoặc `replace`: xác nhận đích, tạo journey hoặc thay journey hiện tại có kiểm soát revision.
- `stop`: kết thúc hành trình và hủy bước chưa commit.
- `resume`: chỉ cho phép với `paused_manual_combat` sau khi combat kết thúc, hoặc trạng thái mà policy resume cho phép.
- `get snapshot`: trả trạng thái chuẩn để load/reconnect/resync.

Snapshot/event cần có journey id/revision, status, scoped destination, vị trí chuẩn hoặc tham chiếu player snapshot, next-step/ETA, route segment nếu hữu ích, last event/result và reason code. Event cũ bị bỏ qua theo revision; reconnect/gap luôn resync snapshot. API mutation phải idempotent hoặc dùng generation/revision để chống double submit và stale worker.

## Frontend

- Mở rộng `ExplorationMap.tsx` để chọn đích, preview tuyến, xác nhận bắt đầu, hiển thị marker/tuyến, ETA/trạng thái, Dừng/Tiếp tục và lỗi cụ thể.
- Mở rộng `worldSlice.ts` hoặc slice phù hợp để gọi API và giữ snapshot/revision. Tích hợp player SSE hiện hữu; không để state local tab làm nguồn vận hành.
- Khi vùng đổi, fetch grid/marker vùng mới và lấy route segment mới; khi reconnect tải snapshot journey trước khi hiện trạng thái điều khiển.
- Thông báo trạng thái nêu việc đã xảy ra, trạng thái journey và hành động cần làm: tiếp tục sau combat, chờ điều kiện, đổi đích hoặc bắt đầu lại.

## Điểm nối code hiện tại

- Movement guard/write/publish: `backend/internal/handlers/movement.go`.
- Từng ô, combat, loot/Resource/WildGu, vùng và path chunks: `backend/internal/handlers/exploration_move.go`, đặc biệt `ExecuteMovePlayerPath` và `resolveCellArrivalWithInstance`.
- Region/atlas/layer: `backend/internal/handlers/zone_registry.go`, `world.go`.
- Worker loop: `backend/internal/worker/runner.go` (`RunEvery`, `SafeRun`).
- Player snapshot, SSE và frontend store: `backend/internal/playerstate/`, `frontend/src/store/useGameStore.ts`, `frontend/src/store/slices/worldSlice.ts`.
- Map UI: `frontend/src/components/ExplorationMap.tsx`.

## Thứ tự triển khai đề xuất

1. Chốt thông số step duration và kiểm chứng tải/chiến lược pathfinding trên Đại Vực; viết test cho route, corner blocking, unknown cells, edge transitions và retry.
2. Thiết kế schema idempotent và API snapshot/preview/start/stop/resume; thêm revision, CAS/claim, persistence của event gần nhất và compatibility DDL.
3. Tích hợp worker resumable/catch-up, guard và từng bước arrival transaction; làm rõ atomicity với các side effect hiện được lưu Redis so với DB.
4. Hủy journey từ mọi movement mutation thành công, party sync, PvP/manual combat và luồng death/respawn; đảm bảo không tạo race giữa worker với movement tay.
5. Nối player event/SSE/resync và frontend map; xử lý nhiều tab/thiết bị và reconnect.
6. Cập nhật `docs/architecture/PLAYER_DOMAIN.md`, `docs/guides/API.md`, `docs/features/world-and-exploration.md`, `docs/guides/DEVELOPMENT.md` nếu thêm worker, `docs/guides/GAME_DATA.md` nếu effect/data thay đổi, và `WIKI.md`/`FEATURES.md` theo code-doc contract.

## Tiêu chí chấp nhận

- Đích sai Đại Vực/tầng/instance bị từ chối; vùng con cùng Đại Vực chỉ qua nếu luật đi bộ và region entry cho phép.
- Mỗi bước được commit đúng một lần, sau đúng thời hạn dựa trên `movement`; bước chéo, buff hết hạn và movement thay đổi cho kết quả nhất quán.
- Đường không cắt góc, tới ô chưa biết được, vật cản mới khiến route recalc; không còn đường thì dừng ở vị trí cuối và nêu lý do.
- Resource/WildGu/encounter/mission progress chạy như bước hiện có; PvE auto/manual và PvP theo đúng trạng thái nêu trên; không tự nhặt drop/điểm tương tác riêng.
- Offline, restart, nhiều backend, dừng/đổi đích và đi tay đồng thời không nhân đôi bước hoặc kéo vị trí sai; catch-up có giới hạn và bảo toàn thứ tự sự kiện.
- Player snapshot/SSE đưa mọi tab về cùng revision; reconnect resync được journey và lịch sử gần nhất.
