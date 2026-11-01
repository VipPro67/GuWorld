-- Gift codes for testing realtime dash and haste Gu.
-- Idempotent: re-running updates rewards without resetting used_count.
-- Apply to gu_db after importing Gu.csv through the Admin panel.

BEGIN;

INSERT INTO gift_codes (code, rewards, max_uses, max_per_player, expires_at, is_active)
VALUES
(
    'TEST_LUOT_HASTE',
    '[
      {"type":"gu","id":"GU_BLACK_BOAR","rank":1,"amount":1,"label":"Hắc Thỉ Cổ - lướt va chạm"},
      {"type":"gu","id":"GU_GREEN_WIND_WHEEL","rank":1,"amount":1,"label":"Thanh Phong Luân Cổ - Tăng Tốc 1.35x"},
      {"type":"gu","id":"GU_FAST_BRIDGE_5","rank":5,"amount":1,"label":"Ngũ Chuyển Độn Kiều Cổ - Tăng Tốc 1.5x"},
      {"type":"gu","id":"GU_MIND_WORLD_I","rank":7,"amount":1,"label":"Tâm Giới Tiên Cổ - Tăng Tốc 1.75x"}
    ]'::jsonb,
    1000000, 1, NOW() + INTERVAL '365 days', TRUE
)
ON CONFLICT (code) DO UPDATE
SET rewards = EXCLUDED.rewards,
    max_uses = EXCLUDED.max_uses,
    max_per_player = EXCLUDED.max_per_player,
    expires_at = EXCLUDED.expires_at,
    is_active = TRUE;

COMMIT;

SELECT code, rewards, max_uses, used_count, max_per_player, expires_at, is_active
FROM gift_codes
WHERE code = 'TEST_LUOT_HASTE';
