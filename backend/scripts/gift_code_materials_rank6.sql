-- Gift code: trao TOÀN BỘ nguyên liệu (Material) rank >= 6, mỗi loại 100 cái.
--
-- Reward type "item" của backend (backend/internal/handlers/gift_codes.go)
-- cộng thẳng vào player_inventory_items với category = 'materials'.
-- Vì vậy chỉ cần tạo gift code, không cần sửa schema hay code.
--
-- Điều kiện: đã import Item.csv (bảng item_dictionary) qua Admin panel.
-- Chạy script trên database của game (mặc định gu_db).
--
-- Lưu ý: đổi code / amount / max_uses / thời hạn bên dưới nếu cần.
-- Script idempotent: chạy lại sẽ cập nhật rewards cho code đã tồn tại,
-- không reset used_count (không cho player đổi lại).

BEGIN;

INSERT INTO gift_codes (code, rewards, max_uses, max_per_player, expires_at, is_active)
SELECT
    'NGUYENLIEU6' AS code,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'type',   'item',
                'id',     i.id,
                'amount', 100,
                'label',  i.name || ' (Rank ' || i.rank::text || ')'
            )
            ORDER BY i.rank, i.name, i.id
        ),
        '[]'::jsonb
    ) AS rewards,
    1000000 AS max_uses,     -- tổng số lượt đổi (đủ cho toàn bộ player)
    1 AS max_per_player,     -- mỗi player chỉ đổi được 1 lần
    NOW() + INTERVAL '365 days' AS expires_at,
    TRUE AS is_active
FROM item_dictionary i
WHERE i.item_type = 'Material'
  AND i.rank >= 6
ON CONFLICT (code) DO UPDATE
SET rewards        = EXCLUDED.rewards,
    max_uses       = EXCLUDED.max_uses,
    max_per_player = EXCLUDED.max_per_player,
    expires_at     = EXCLUDED.expires_at,
    is_active      = TRUE;

COMMIT;

-- Kiểm tra kết quả:
-- Số nguyên liệu rank >= 6 trong dữ liệu và số reward đã nhúng vào code.
SELECT
    (SELECT COUNT(*) FROM item_dictionary WHERE item_type = 'Material' AND rank >= 6) AS materials_rank6plus,
    jsonb_array_length(rewards) AS rewards_in_code,
    code,
    max_uses,
    max_per_player,
    expires_at,
    is_active
FROM gift_codes
WHERE code = 'NGUYENLIEU6';
