-- Gift code: trao bộ 12 Cổ Trùng chiến đấu tiêu biểu Rank 4 và Rank 5 đủ các hệ phái + 100.000 Nguyên Thạch để test
--
-- Code: TESTGU45
--
-- Thực thi trên database gu_db.
-- Script idempotent: chạy lại sẽ cập nhật rewards mà không reset used_count.

BEGIN;

INSERT INTO gift_codes (code, rewards, max_uses, max_per_player, expires_at, is_active)
VALUES (
    'TESTGU45',
    '[
        {"type": "gu", "id": "GU_BEAM_SWORD_4", "rank": 4, "amount": 1, "label": "Tứ Chuyển Mang Kiếm Cổ (Rank 4 - Kiếm Đạo Bắn Tia)"},
        {"type": "gu", "id": "GU_FIRE_SEA", "rank": 4, "amount": 1, "label": "Hỏa Hải Cổ (Rank 4 - Hỏa Đạo Vùng Lửa AOE)"},
        {"type": "gu", "id": "GU_COLD_SWORD_4", "rank": 4, "amount": 1, "label": "Tứ Chuyển Hàn Kiếm Cổ (Rank 4 - Băng Đạo Làm Chậm)"},
        {"type": "gu", "id": "GU_CRIMSON_TIDE", "rank": 4, "amount": 1, "label": "Hồng Triều Cổ (Rank 4 - Huyết/Thủy Triều Sóng Dâng)"},
        {"type": "gu", "id": "GU_WATER_CLEANSING", "rank": 4, "amount": 1, "label": "Tuyệt Tẩy Cổ (Rank 4 - Thủy Đạo Hồi Máu & Tẩy Debuff)"},
        {"type": "gu", "id": "GU_ANCIENT_BODY_4", "rank": 4, "amount": 1, "label": "Tứ Chuyển Cổ Thân Cổ (Rank 4 - Lực Đạo Tạo Khiên 600)"},
        {"type": "gu", "id": "GU_TREASURE_MOONLIGHT_KING", "rank": 5, "amount": 1, "label": "Bảo Nguyệt Quang Vương Cổ (Rank 5 - Quang Đạo Trảm Kích Xuyên Thấu)"},
        {"type": "gu", "id": "GU_CLOUD_THUNDER_BURST_5", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Vân Lôi Bạo Cổ (Rank 5 - Lôi Đạo Bộc Phá Sấm Sét)"},
        {"type": "gu", "id": "GU_FLYING_SWORD_5", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Ngự Phong Kiếm Cổ (Rank 5 - Phong/Kiếm Đạo Phi Kiếm)"},
        {"type": "gu", "id": "GU_BURNING_FIRE_5", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Phần Hỏa Cổ (Rank 5 - Hỏa/Hủy Diệt Thiêu Đốt)"},
        {"type": "gu", "id": "GU_FISH_SCALE_WATER_5", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Ngư Lân Thủy Cổ (Rank 5 - Thủy Đạo Vùng Đầm Lầy)"},
        {"type": "gu", "id": "GU_BARRIER_DISPEL_5", "rank": 5, "amount": 1, "label": "Ngũ Chuyển Chướng Phá Cổ (Rank 5 - Hộ Thuẫn Cực Đại 3.000)"},
        {"type": "primeval_stones", "amount": 100000, "label": "100.000 Nguyên Thạch"}
    ]'::jsonb,
    1000000,
    1,
    NOW() + INTERVAL '365 days',
    TRUE
)
ON CONFLICT (code) DO UPDATE
SET rewards        = EXCLUDED.rewards,
    max_uses       = EXCLUDED.max_uses,
    max_per_player = EXCLUDED.max_per_player,
    expires_at     = EXCLUDED.expires_at,
    is_active      = TRUE;

COMMIT;
