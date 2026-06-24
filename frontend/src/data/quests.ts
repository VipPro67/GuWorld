export interface QuestReward {
  type: string;
  id?: string;
  amount: number;
  label: string;
}

export interface QuestDef {
  id: string;
  title: string;
  desc: string;
  req_desc: string;
  rewards: QuestReward[];
  check: () => boolean;
  icon: string;
}

export function formatRewards(rewards: QuestReward[]): string {
  if (!rewards || rewards.length === 0) return "";
  const labels = rewards.map((r) => r.label).filter(Boolean);
  return labels.length > 0 ? labels.join(", ") : "";
}

export function defaultQuests(checkCtx: {
  rank: number;
  guInventory: { gu_id: string; is_vital: boolean; equipped: boolean }[];
  clearedCells: Record<string, string>;
  guDictionary: Record<string, { rank: number }>;
}): QuestDef[] {
  const { rank, guInventory, clearedCells, guDictionary } = checkCtx;
  return [
    {
      id: "quest_1_awaken",
      title: "Giác Tỉnh Không Khiếu",
      desc: "Khai mở không khiếu thành công",
      req_desc: "Cảnh giới Nhất Chuyển Sơ Giai trở lên",
      rewards: [
        { type: "gu", id: "GU_GREEN_SILK", amount: 1, label: "Thanh Ti Cổ" },
        { type: "item", id: "ITEM_HEALING_PELLET_SMALL", amount: 3, label: "3 Tiểu Hồi Huyết Đan" },
      ],
      check: () => rank >= 1,
      icon: "sparkles",
    },
    {
      id: "quest_2_vital_gu",
      title: "Cổ Trùng Bản Mệnh",
      desc: "Luyện hóa hoặc trang bị Cổ trùng",
      req_desc: "Đang trang bị hoặc có Cổ trùng bản mệnh",
      rewards: [
        { type: "gu", id: "GU_WHITE_JADE", amount: 1, label: "Bạch Ngọc Cổ" },
        { type: "item", id: "ITEM_MEDICINAL_PASTE", amount: 2, label: "2 Kim Sang Dược" },
      ],
      check: () => guInventory?.some((g) => g.is_vital || g.equipped),
      icon: "flame",
    },
    {
      id: "quest_3_rank_2",
      title: "Trùng Kích Nhị Chuyển",
      desc: "Đột phá vách khiếu thăng Nhị Chuyển",
      req_desc: "Tu vi đạt Nhị Chuyển",
      rewards: [
        { type: "gu", id: "GU_WATER_SHIELD", amount: 1, label: "Thủy Tráo Cổ" },
        { type: "item", id: "ITEM_HEALING_PELLET_MEDIUM", amount: 3, label: "3 Trung Hồi Huyết Đan" },
      ],
      check: () => rank >= 2,
      icon: "zap",
    },
    {
      id: "quest_4_refine_gu",
      title: "Luyện Chế Cổ Trùng Nhị Chuyển",
      desc: "Luyện thành công Cổ trùng bậc 2 trở lên",
      req_desc: "Có Cổ trùng bậc 2 trở lên",
      rewards: [
        { type: "gu", id: "GU_CANOPY", amount: 1, label: "Thiên Bồng Cổ" },
        { type: "item", id: "ITEM_BLOOD_GINSENG", amount: 2, label: "2 Huyết Sâm" },
      ],
      check: () =>
        guInventory?.some((g) => {
          const spec = guDictionary?.[g.gu_id];
          return spec && spec.rank >= 2;
        }),
      icon: "atom",
    },
    {
      id: "quest_5_fame",
      title: "Danh Chấn Thanh Bản Thôn",
      desc: "Chiến thắng quái thú dọn sạch 5 địa khu",
      req_desc: `Đã quét dọn ${Object.keys(clearedCells || {}).length}/5 ô`,
      rewards: [
        { type: "gu", id: "GU_FLYING_BONE_SHIELD", amount: 1, label: "Phi Cốt Thuẫn Cổ" },
        { type: "item", id: "mine_chest", amount: 3, label: "3 Rương Mỏ Linh Thạch" },
      ],
      check: () => Object.keys(clearedCells || {}).length >= 5,
      icon: "crown",
    },
    {
      id: "quest_6_rank_5",
      title: "Đỉnh Phong Chuẩn Bị",
      desc: "Tu vi đỉnh phong, sẵn sàng độ kiếp thăng tiên",
      req_desc: "Tu vi đạt Ngũ Chuyển",
      rewards: [
        { type: "gu", id: "GU_GOLDEN_BELL_SHIELD", amount: 1, label: "Kim Chung Tráo Cổ" },
        { type: "item", id: "ITEM_BLOOD_GINSENG", amount: 5, label: "5 Huyết Sâm" },
      ],
      check: () => rank >= 5,
      icon: "shield_alert",
    },
  ];
}
