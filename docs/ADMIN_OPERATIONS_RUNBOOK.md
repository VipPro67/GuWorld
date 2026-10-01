# Runbook vận hành Admin

Tài liệu này dành cho người trực vận hành có quyền truy cập host và PostgreSQL.
Mọi thời gian điều tra dùng UTC. Với thao tác **break-glass**, luôn tạo incident,
ghi `INCIDENT_ID`, chụp backup trước khi sửa, dùng tài khoản DB cá nhân nếu có,
và không xóa `audit_logs` hay tài khoản bị điều tra.

## Quan sát và cảnh báo

- `GET /api/admin/monitor/admin-operations` (permission `monitoring.read`) trả
  counter trong memory kể từ lúc process khởi động hoặc lần reset gần nhất:
  403 theo permission, mutation thất bại theo action, destructive action, lỗi
  import, lỗi ghi audit, lần dùng fallback `is_admin`, và lỗi resolve permission.
- `POST /api/admin/monitor/requests/reset` reset cả request metrics/traces và các
  counter trên. Chỉ reset sau khi đã lưu bằng chứng của incident.
- Tìm các structured event sau trong log backend: `admin_permission_denied`,
  `admin_permission_resolution_error`, `admin_mutation_failed`,
  `admin_destructive_action`, `admin_import_failed`,
  `authorization_legacy_is_admin_fallback`, `audit_write_failed`.
- Metric là counter theo process, không bền vững qua restart. Log và
  `audit_logs` là nguồn để dựng timeline. Endpoint monitor cũng cần DB để xác
  thực, vì vậy không dùng nó làm health check chính khi DB hỏng.

## Tạo admin đầu tiên

1. Khởi động PostgreSQL, Redis và backend. `InitSchema` tạo tài khoản bootstrap
   `admin` nếu chưa tồn tại và chỉ gán `super_admin` cho tài khoản đó khi hệ
   thống chưa có super admin.
2. Đăng nhập bootstrap qua kênh nội bộ. Bản cài đặt cũ có thể còn credential
   bootstrap trong mã nguồn; coi credential đó đã lộ, không gửi lại qua
   chat/ticket và không dùng lâu dài. Sau lần tạo đầu tiên, startup không tự
   bật lại `is_admin` cho tài khoản đã bị hạ quyền.
3. Đổi mật khẩu bootstrap ngay. Tạo hoặc chọn tài khoản cá nhân của operator.
4. Trong Admin, dùng `PUT /api/admin/players/:id/roles` để gán
   `super_admin`. Endpoint yêu cầu reason, confirmation target, mật khẩu xác
   thực lại và `Idempotency-Key`; ghi mã incident/change request vào reason.
5. Đăng nhập bằng tài khoản cá nhân, kiểm tra `/api/admin/me`, rồi hạ quyền hoặc
   khóa tài khoản bootstrap. Luôn duy trì tối thiểu hai super admin cá nhân;
   backend từ chối xóa super admin cuối cùng.
6. Xác minh audit action `admin.update_player_roles`. Không cấp quyền bằng cách
   chỉ sửa `players.is_admin`: đây chỉ là projection tương thích cũ.

## Khôi phục super admin (break-glass)

Chỉ dùng khi không còn super admin dùng được. Ưu tiên restore quyền bằng API nếu
còn một super admin. Các bảng RBAC đang hoạt động dùng `role_key`.

```bash
export TARGET_USERNAME='operator-name'
export INCIDENT_ID='INC-YYYY-NNNN'
docker compose -f docker-compose.local.yml exec -T db \
  psql -X -v ON_ERROR_STOP=1 -U gu_admin -d gu_db \
  -v target_username="$TARGET_USERNAME" -v incident_id="$INCIDENT_ID" <<'SQL'
SELECT id, username FROM players WHERE username = :'target_username';
-- DỪNG nếu kết quả không đúng chính xác một dòng.
BEGIN;
SELECT pg_advisory_xact_lock(782014621);
INSERT INTO player_admin_roles (player_id, role_key, assigned_by)
SELECT id, 'super_admin', NULL FROM players WHERE username = :'target_username'
ON CONFLICT (player_id, role_key) DO NOTHING;
UPDATE players SET is_admin = true WHERE username = :'target_username';
INSERT INTO audit_logs
  (player_id, actor_username, action, effective_roles, effective_permission,
   entity_type, entity_id, change_data, reason, request_summary, ip_address, success)
SELECT id, 'break-glass-operator', 'admin.recover_super_admin',
       '["super_admin"]'::jsonb, 'break_glass', 'player', id::text,
       '{"source":"operator_sql"}'::jsonb, :'incident_id',
       'Emergency super-admin recovery', 'local-console', true
FROM players WHERE username = :'target_username';
COMMIT;
SQL
```

Xác minh đúng một role được cấp, đăng nhập lại, kiểm tra `/api/admin/me` và audit.
Sau đó rotate credential DB/host đã dùng và đóng incident với người phê duyệt.

## Thu hồi tài khoản bị xâm nhập

1. Cô lập: chặn ingress/IP nếu phù hợp, giữ log, ghi mốc UTC đầu tiên, không xóa
   player. Việc xóa làm mất liên kết actor do foreign key `ON DELETE SET NULL`.
2. Nếu còn super admin an toàn, thay roles của nạn nhân bằng `[]` qua API và
   ghi incident trong reason. Sau đó vô hiệu hóa session bằng quy trình quản lý
   tài khoản.
3. Nếu API không an toàn/không dùng được, chạy transaction break-glass sau.

```bash
export TARGET_USERNAME='compromised-name'
export INCIDENT_ID='INC-YYYY-NNNN'
docker compose -f docker-compose.local.yml exec -T db \
  psql -X -v ON_ERROR_STOP=1 -U gu_admin -d gu_db \
  -v target_username="$TARGET_USERNAME" -v incident_id="$INCIDENT_ID" <<'SQL'
SELECT id, username FROM players WHERE username = :'target_username';
-- DỪNG nếu kết quả không đúng chính xác một dòng hoặc đây là super admin an toàn cuối cùng.
BEGIN;
SELECT pg_advisory_xact_lock(782014621);
DELETE FROM player_admin_roles
 WHERE player_id = (SELECT id FROM players WHERE username = :'target_username');
UPDATE players
   SET is_admin = false, active_session_id = gen_random_uuid()::text
 WHERE username = :'target_username';
INSERT INTO audit_logs
  (player_id, actor_username, action, effective_permission, entity_type,
   entity_id, change_data, reason, request_summary, ip_address, success)
SELECT id, 'break-glass-operator', 'admin.revoke_compromised_account',
       'break_glass', 'player', id::text,
       '{"roles":[]}'::jsonb, :'incident_id',
       'Emergency privilege and session revocation', 'local-console', true
FROM players WHERE username = :'target_username';
COMMIT;
SQL
```

4. Reset/rotate password, OAuth identity và mọi secret mà tài khoản có thể thấy.
   Principal được resolve từ DB ở mỗi request; thay `active_session_id` làm JWT
   hiện tại bị từ chối ở request kế tiếp.
5. Điều tra toàn bộ khoảng thời gian có khả năng bị xâm nhập theo mục tiếp theo.

## Điều tra thao tác admin

1. Ghi `from`/`to` UTC, actor, player ID, IP, action/entity và request ID biết được.
2. Dùng Admin → Audit hoặc `GET /api/admin/audit-logs` với `action`,
   `entity_type`, `player_id`, `success`, `search`, pagination. Bắt đầu với
   `success=false`, sau đó kiểm tra cả thành công và destructive action.
3. Correlate `request_id` với log backend và request trace. Kiểm tra roles và
   permission hiệu lực, reason, `change_data.before/after`, IP và kết quả.
4. Khi API không dùng được, query read-only:

```sql
SELECT created_at, player_id, actor_username, action, effective_roles,
       effective_permission, entity_type, entity_id, reason, request_id,
       ip_address, success, error_message, change_data
FROM audit_logs
WHERE created_at >= :'from'::timestamptz
  AND created_at < :'to'::timestamptz
  AND (:'actor' = '' OR actor_username = :'actor')
ORDER BY created_at, id;
```

5. Export evidence read-only, checksum file, lưu theo retention policy. Không sửa
   audit để “sửa” incident; thêm một audit/incident note mới nếu cần đính chính.

## Rollback import lỗi

Trước mỗi import production, ưu tiên một dataset/lần, lưu CSV đã duyệt và backup:

```bash
backup="gu_db-pre-import-$(date -u +%Y%m%dT%H%M%SZ).dump"
docker compose -f docker-compose.local.yml exec -T db \
  pg_dump -U gu_admin -d gu_db -Fc > "$backup"
pg_restore --list "$backup" >/dev/null
```

1. Dừng import tiếp theo; lưu response, `preflight_id`, `source_digest`, plan,
   request ID và audit/log liên quan.
2. Nếu preflight/commit trả lỗi trước `200`, transaction commit đã rollback.
   Xác minh dữ liệu và audit; không restore DB chỉ vì preflight thất bại.
3. Nếu commit thành công nhưng `reload_ok=false`/`restart_required=true`, restart
   backend trước rồi kiểm tra registry và dữ liệu. Đây có thể chỉ là lỗi reload,
   không phải lỗi transaction.
4. Với sửa logic, import CSV đã biết tốt qua **preflight mới + commit mới**, ghi
   incident trong reason. Import là UPSERT; re-import file cũ không xóa row mới,
   nên không được coi là rollback bit-for-bit.
5. Nếu phải xóa row mới hoặc hoàn nguyên chính xác, mở maintenance window, dừng
   writer, thử restore trên DB tách biệt, rồi restore backup pre-import theo quy
   trình PostgreSQL của môi trường. Full restore là destructive và cần phê duyệt.
6. Khởi động lại backend để reload registry, chạy smoke test và lưu audit/timeline.

## Redis hoặc database không khả dụng

Backend cần cả PostgreSQL và Redis khi khởi động. Không dựa vào endpoint admin
khi DB đang lỗi vì chính authorization cũng cần DB.

```bash
docker compose -f docker-compose.local.yml ps
docker compose -f docker-compose.local.yml logs --since=30m backend db redis
docker compose -f docker-compose.local.yml exec -T db pg_isready -U gu_admin -d gu_db
docker compose -f docker-compose.local.yml exec -T db psql -U gu_admin -d gu_db -c 'SELECT 1'
docker compose -f docker-compose.local.yml exec -T redis redis-cli PING
```

1. Xác định dependency lỗi trước; kiểm tra disk, connection limit, DNS/TLS và
   credential. Không flush Redis và không reset DB để thử vận may.
2. Khôi phục dependency bị lỗi. Local compose có thể dùng:

```bash
docker compose -f docker-compose.local.yml restart redis db
# Chỉ sau khi pg_isready/SELECT 1 và PING đều thành công:
docker compose -f docker-compose.local.yml restart backend
```

3. Với DB ngoài compose: chạy `pg_isready` bằng `DB_HOST`, `DB_PORT`, `DB_USER`,
   `DB_NAME`. Với Redis ngoài compose: dùng `redis-cli` theo host/TLS của môi
   trường; tránh đặt password trực tiếp trên command line/history.
4. Sau recovery, kiểm tra `/api/admin/monitor/system`, error rate, worker backlog,
   WebSocket và một read/write smoke test. Đối chiếu log
   `admin_permission_resolution_error`: lỗi DB permission phải trả `503`, không
   được diễn giải là người dùng thiếu quyền.
