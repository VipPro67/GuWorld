-- Gift code test các Cổ Biến Thân rank < 6 (Biến Hóa Đạo)
-- Dùng được 999.999 lần, mỗi player được nhận tối đa 99 lần.
INSERT INTO gift_codes (code, rewards, max_uses, max_per_player, expires_at, is_active)
VALUES (
    'TEST_BIENHOA',
    '[
        {"type": "gu", "id": "GU_TRANSFORM_WOLF_1", "rank": 1, "amount": 1, "label": "Nhất Chuyển Lang Biến Cổ (Rank 1)"},
        {"type": "gu", "id": "GU_TRANSFORM_FISH_2", "rank": 2, "amount": 1, "label": "Nhị Chuyển Ngư Biến Cổ (Rank 2)"},
        {"type": "gu", "id": "GU_TRANSFORM_BEAR_3", "rank": 3, "amount": 1, "label": "Tam Chuyển Hùng Biến Cổ (Rank 3)"},
        {"type": "gu", "id": "GU_EAGLE_WINGS", "rank": 3, "amount": 1, "label": "Tam Chuyển Ưng Dực Cổ (Rank 3)"},
        {"type": "gu", "id": "GU_TRANSFORM_EAGLE_4", "rank": 4, "amount": 1, "label": "Tứ Chuyển Ưng Biến Cổ (Rank 4)"},
        {"type": "gu", "id": "GU_TRANSFORM_CHIMERA_5", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Bách Thú Biến Cổ (Rank 5)"},
        {"type": "gu", "id": "GU_CHANGE_FORM", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Biến Hóa Cổ (Rank 5)"},
        {"type": "primeval_stones", "amount": 100000, "label": "100.000 Nguyên Thạch"}
    ]'::jsonb,
    999999,
    99,
    NOW() + INTERVAL '365 days',
    true
)
ON CONFLICT (code) DO UPDATE
SET rewards = EXCLUDED.rewards,
    is_active = true,
    expires_at = EXCLUDED.expires_at;
