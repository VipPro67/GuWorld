/**
 * Từ điển hiệu ứng người chơi (nguồn cho /wiki/effects*).
 *
 * Nguồn chuẩn: backend/internal/effectcatalog/block_manifest.go (trigger/op/
 * target/condition/stat/resource) và hành vi runtime trong backend/internal/
 * {handlers,simulator,gamefx}. Khi thêm trigger/op/native hook mới, cập nhật
 * file này — test src/lib/wiki/effects.test.ts sẽ fail nếu Gu.csv có mã lạ.
 */

export type TimingGroup = "always" | "action" | "event" | "scheduled" | "capability";

export interface GlossaryTrigger {
  code: string;
  label: string;
  group: TimingGroup;
  when: string;
  note?: string;
}

export const TIMING_GROUPS: { id: TimingGroup; label: string; summary: string }[] = [
  { id: "always", label: "Thường trực", summary: "Chỉ cần sở hữu/trang bị là có hiệu lực, đọc lại mỗi lần tính chỉ số." },
  { id: "action", label: "Khi hành động", summary: "Chạy lúc Cổ sư ra chiêu hoặc dùng/nuốt Cổ." },
  { id: "event", label: "Theo sự kiện trận", summary: "Chạy khi trận đấu bắn ra mốc tương ứng (vào trận, trúng đòn, chết...)." },
  { id: "scheduled", label: "Theo chu kỳ", summary: "Chạy theo đồng hồ: tick worker hoặc hết hạn trạng thái." },
  { id: "capability", label: "Năng lực", summary: "Không tự chạy; mở khóa một hành động (chợ, chat, bắt thú, dịch chuyển...)." },
];

export const EFFECT_TRIGGERS: GlossaryTrigger[] = [
  { code: "ALWAYS_ON", label: "Nội Tại", group: "always", when: "Luôn có hiệu lực khi Cổ được trang bị ACTIVE (hoặc chủng tộc/thể chất/trạng thái đang sở hữu).", note: "Hầu hết hiệu ứng chỉ số nằm ở đây." },
  { code: "ON_ACTION", label: "Khi Ra Chiêu", group: "action", when: "Kích hoạt khi Cổ sư dùng Cổ làm sát chiêu trong trận, hoặc dùng Cổ hồi phục ngoài trận.", note: "Đây là trigger phổ biến nhất của Cổ chủ động." },
  { code: "ON_HIT", label: "Khi Đánh Trúng", group: "event", when: "Chạy sau khi đòn đánh đã trúng mục tiêu.", note: "Không chạy nếu đòn bị né hoặc miễn nhiễm." },
  { code: "BEFORE_HIT", label: "Trước Khi Bị Đánh", group: "event", when: "Chạy ngay trước khi sát thương của đòn đánh giáng xuống người mang.", note: "Dùng cho hiệu ứng né/đỡ đòn." },
  { code: "ON_TURN_START", label: "Đầu Lượt", group: "event", when: "Chạy ở đầu mỗi lượt của người mang, trước khi chọn hành động." },
  { code: "ON_DEATH", label: "Khi Tử Vong", group: "event", when: "Chạy khi HP về 0, trước khi trận đấu kết thúc.", note: "Có thể hồi sinh và tiếp tục trận." },
  { code: "ON_TICK", label: "Mỗi Chu Kỳ", group: "scheduled", when: "Chạy theo worker định kỳ (3 giờ/lần cho tài nguyên) hoặc theo giờ cho hồi phục.", note: "Không chạy trong lúc giao chiến." },
  { code: "CAPABILITY", label: "Năng Lực", group: "capability", when: "Không tự kích hoạt; hệ thống kiểm tra khi người chơi dùng một chức năng (chợ, chat, cổng, bắt thú...).", note: "Cổ chỉ cần nằm trong túi là đủ, trừ khi ghi chú khác." },
  { code: "COMBAT_PASSIVE", label: "Nội Tại Chiến Đấu", group: "event", when: "Hiệu ứng bị động được engine đọc trong trận (ví dụ nhân đôi sát thương, đảo tai ách).", note: "Khác ALWAYS_ON: chỉ có ý nghĩa khi đang giao chiến." },
  { code: "PASSIVE", label: "Nội Tại (tên cũ)", group: "event", when: "Tên cũ của COMBAT_PASSIVE, dữ liệu cũ vẫn dùng.", note: "Dữ liệu mới nên dùng COMBAT_PASSIVE." },
  { code: "ON_BATTLE_START", label: "Vào Trận", group: "event", when: "Chạy đúng một lần khi trận bắt đầu, trước Lượt 1.", note: "Dùng cho triệu hồi, đoạt tiên cơ, thắng tuyệt đối..." },
  { code: "BEFORE_ENEMY_ATTACK", label: "Trước Khi Địch Đánh", group: "event", when: "Chạy ngay trước đòn của kẻ địch nhắm vào người mang." },
  { code: "ON_CAST", label: "Khi Niệm Chiêu", group: "event", when: "Chạy khi Cổ sư thôi động Cổ trong trận, trước khi đòn đánh được xử lý.", note: "Dùng cho biến thân/buff/danh sách chiêu đặc biệt." },
  { code: "ON_CONSUME", label: "Khi Nuốt Cổ", group: "action", when: "Chạy khi dùng Cổ Tiêu Hao qua túi đồ; Cổ biến mất sau khi hiệu ứng áp dụng.", note: "Cần Cổ đủ no, đủ Chân Nguyên và đúng điều kiện cảnh giới." },
  { code: "ON_USE", label: "Khi Kích Hoạt", group: "action", when: "Chạy khi dùng Cổ Hỗ Trợ ngoài trận (nút Dùng).", note: "Chỉ một số Cổ hỗ trợ có hiệu ứng này." },
];

export interface GlossarySubType {
  code: string;
  label: string;
  note: string;
}

export interface GlossaryOperation {
  code: string;
  label: string;
  summary: string;
  subTypes: GlossarySubType[];
  fields: string[];
  note?: string;
}

export const EFFECT_OPERATIONS: GlossaryOperation[] = [
  {
    code: "DAMAGE",
    label: "Sát Thương",
    summary: "Gây sát thương thường lên mục tiêu; bị giảm bởi phòng thủ, khiên và chênh lệch cảnh giới.",
    subTypes: [
      { code: "FLAT", label: "Sát thương cố định", note: "Cộng thẳng vào đòn đánh." },
      { code: "MULTIPLIER", label: "Hệ số sát thương", note: "value 1.5 = đòn mạnh ×1.5." },
      { code: "MAX_HP_PERCENT", label: "% Máu tối đa", note: "Gây sát thương theo % MaxHP của mục tiêu; vẫn bị giảm nếu cảnh giới địch cao hơn." },
    ],
    fields: ["value", "unit", "path", "chance", "target"],
  },
  {
    code: "DIRECT_HP",
    label: "Sát Thương Trực Tiếp",
    summary: "Trừ thẳng HP mục tiêu, bỏ qua phòng thủ và thường bỏ qua khiên — mạnh nhưng hiếm.",
    subTypes: [
      { code: "MAX_HP_PERCENT", label: "% Máu tối đa", note: "value 0.4 = lấy 40% MaxHP. Bị Bất Hoại chặn, Bất Tử giữ lại 1 HP." },
      { code: "SHIELD_STRIP", label: "Bóc khiên", note: "Lấy đi % khiên ảo hiện có của mục tiêu, không gây HP." },
      { code: "FLAT", label: "Số cố định", note: "Trừ đúng value HP." },
    ],
    fields: ["value", "unit", "target"],
    note: "Sát thương trực tiếp bị giảm khi cảnh giới mục tiêu cao hơn cảnh giới Cổ.",
  },
  {
    code: "SHIELD",
    label: "Khiên Ảo",
    summary: "Tạo lớp khiên hấp thụ sát thương đòn đánh trước khi trừ HP.",
    subTypes: [
      { code: "FLAT", label: "Khiên cố định", note: "Cộng đúng value." },
      { code: "MULTIPLIER", label: "Hệ số khiên", note: "Nhân khiên của chiêu/kỹ năng." },
    ],
    fields: ["value", "unit", "turns"],
  },
  {
    code: "BARRIER",
    label: "Kết Giới",
    summary: "Lớp phòng thủ bền hơn khiên, tồn tại nhiều lượt và có thể dựa trên % MaxHP.",
    subTypes: [
      { code: "FLAT", label: "Kết giới cố định", note: "Cộng đúng value." },
      { code: "PERCENT", label: "% Máu tối đa", note: "value 0.2 = 20% MaxHP thành kết giới." },
    ],
    fields: ["value", "unit", "turns", "stack_policy"],
  },
  {
    code: "HEAL",
    label: "Hồi Phục",
    summary: "Hồi HP (hoặc tài nguyên) cho bản thân/đồng đội; vượt MaxHP khi hút máu sẽ chuyển thành khiên.",
    subTypes: [
      { code: "FLAT", label: "Hồi cố định", note: "Cộng đúng value." },
      { code: "PERCENT", label: "% Máu tối đa", note: "value 0.15 = 15% MaxHP." },
      { code: "LIFESTEAL", label: "Hút máu", note: "Hồi theo % sát thương gây ra." },
      { code: "RESOURCE", label: "Hồi tài nguyên", note: "Dùng với resource = MANA/SOUL/THOUGHT." },
    ],
    fields: ["value", "unit", "resource", "target"],
  },
  {
    code: "STAT_MOD",
    label: "Đổi Chỉ Số",
    summary: "Cộng/nhân/đặt một chỉ số chiến đấu trong thời hạn hiệu ứng.",
    subTypes: [
      { code: "ADD_FLAT", label: "Cộng thẳng", note: "value là số điểm." },
      { code: "MULTIPLY_PERCENT", label: "Nhân %", note: "value 0.2 = +20% chỉ số." },
      { code: "SET", label: "Đặt giá trị", note: "Ghi đè chỉ số bằng value." },
      { code: "GROWTH_CAP", label: "Ngưỡng trưởng thành", note: "Chỉ số tăng dần tới mức này khi nuôi Cổ." },
    ],
    fields: ["stat", "value", "unit", "turns", "stack_policy", "path"],
    note: "Mỗi stat có cách cộng dồn riêng: buff cùng nguồn thường lấy max hoặc cộng dồn theo stack_policy.",
  },
  {
    code: "STATUS",
    label: "Gắn Trạng Thái",
    summary: "Gán buff/debuff/khống chế lên mục tiêu trong N lượt.",
    subTypes: [],
    fields: ["status_id", "turns", "chance", "value", "stack_policy"],
    note: "Tỉ lệ dính gốc lấy từ Status.csv; mỗi chênh lệch cảnh giới của Cổ so với rank 1 cộng thêm 5%, trần 95%.",
  },
  {
    code: "CLEANSE",
    label: "Giải Trạng Thái",
    summary: "Xóa debuff/trạng thái xấu trên bản thân hoặc đồng đội.",
    subTypes: [
      { code: "ALL", label: "Tất cả", note: "Xóa mọi trạng thái xấu." },
      { code: "DEBUFF", label: "Debuff", note: "Chỉ xóa debuff." },
      { code: "STATUS", label: "Theo mã", note: "Chỉ xóa status_id được khai báo." },
    ],
    fields: ["sub_type", "status_id", "target"],
  },
  {
    code: "UNIQUE",
    label: "Cơ Chế Đặc Biệt",
    summary: "Gọi một mã native hook do engine/handler xử lý riêng; xem trang Cơ Chế Đặc Biệt.",
    subTypes: [],
    fields: ["code", "params", "target"],
    note: "Không thể đoán tác dụng chỉ từ op — phải đọc bảng native hook.",
  },
];

export interface GlossaryTarget {
  code: string;
  label: string;
  note: string;
}

export const EFFECT_TARGETS: GlossaryTarget[] = [
  { code: "SELF", label: "Bản thân", note: "Người dùng hoặc người mang hiệu ứng." },
  { code: "TARGET", label: "Mục tiêu", note: "Một địch thủ do hệ thống chọn (thường là mục tiêu đang nhắm)." },
  { code: "SINGLE", label: "Một mục tiêu", note: "Tương đương TARGET trong dữ liệu Cổ." },
  { code: "ROW", label: "Hàng ngang", note: "Toàn bộ một hàng trong đội hình địch." },
  { code: "COL", label: "Hàng dọc", note: "Toàn bộ một cột trong đội hình địch." },
  { code: "ALL_ENEMIES", label: "Toàn bộ địch", note: "Mọi địch thủ còn sống." },
  { code: "ALL_ALLIES", label: "Toàn bộ đồng đội", note: "Mọi đồng đội còn sống, gồm bản thân nếu cùng phe." },
  { code: "ALL", label: "Toàn trận", note: "Cả hai phe." },
];

export const EFFECT_CONDITIONS: GlossaryTarget[] = [
  { code: "HP_PERCENT", label: "% HP người dùng", note: "So sánh HP hiện tại / MaxHP." },
  { code: "ACTOR_HP_PERCENT", label: "% HP người hành động", note: "Dùng khi hiệu ứng phản ứng với đòn của địch." },
  { code: "TARGET_HP_PERCENT", label: "% HP mục tiêu", note: "Điều kiện kiểu kết liễu khi địch thấp máu." },
  { code: "MANA_PERCENT", label: "% Chân Nguyên", note: "So sánh Chân Nguyên hiện tại / tối đa." },
  { code: "TURN", label: "Lượt hiện tại", note: "Chỉ kích hoạt ở lượt chỉ định." },
  { code: "STACKS", label: "Số tầng", note: "Số tầng hiệu ứng/trạng thái đang có." },
  { code: "RANK", label: "Cảnh giới", note: "So sánh cảnh giới người dùng hoặc Cổ." },
  { code: "MASTERY", label: "Chân Ý", note: "Mức lưu phái/chân ý liên quan." },
];

export const EFFECT_CONDITION_OPERATORS: GlossaryTarget[] = [
  { code: "EQ", label: "Bằng", note: "=" },
  { code: "NE", label: "Khác", note: "≠" },
  { code: "LT", label: "Nhỏ hơn", note: "<" },
  { code: "LTE", label: "Nhỏ hơn hoặc bằng", note: "≤" },
  { code: "GT", label: "Lớn hơn", note: ">" },
  { code: "GTE", label: "Lớn hơn hoặc bằng", note: "≥" },
];

export const EFFECT_UNITS: GlossaryTarget[] = [
  { code: "FLAT", label: "Số cố định", note: "Cộng/trừ thẳng giá trị." },
  { code: "PERCENT", label: "Phần trăm", note: "Tính theo % của một mốc (MaxHP, MaxMana...)." },
  { code: "RATIO", label: "Tỉ lệ", note: "Hệ số nhân, 0.1 = 10%." },
  { code: "POINTS", label: "Điểm", note: "Giá trị điểm nội bộ (Chân Ý, Đạo Ngân...)." },
];

export const EFFECT_STACK_POLICIES: GlossaryTarget[] = [
  { code: "REPLACE", label: "Thay thế", note: "Hiệu ứng mới ghi đè hiệu ứng cũ cùng nguồn." },
  { code: "REFRESH", label: "Làm mới", note: "Giữ hiệu ứng cũ, làm mới thời hạn." },
  { code: "STACK", label: "Cộng dồn", note: "Nhiều tầng cộng lại với nhau." },
  { code: "MAX", label: "Lấy cao nhất", note: "Chỉ giữ giá trị mạnh nhất." },
  { code: "IGNORE", label: "Bỏ qua", note: "Không áp dụng nếu đã có hiệu ứng cùng loại." },
];

export const EFFECT_STATS: GlossaryTarget[] = [
  { code: "ATTACK", label: "Công", note: "Sát thương đòn đánh thường và nền của kỹ năng." },
  { code: "DEFENSE", label: "Thủ", note: "Giảm sát thương nhận vào." },
  { code: "HP", label: "Sinh Lực", note: "Máu tối đa." },
  { code: "SPEED", label: "Tốc Độ", note: "Quyết định thứ tự ra đòn." },
  { code: "DAMAGE_DEALT", label: "Sát Thương Gây Ra", note: "Hệ số nhân sát thương đầu ra." },
  { code: "CRIT_RATE", label: "Bạo Kích", note: "Tỉ lệ chí mạng." },
  { code: "TENACITY", label: "Kiên Cường", note: "Tỉ lệ tự giải debuff đầu lượt, rút ngắn khống chế." },
  { code: "CC_RESIST", label: "Kháng Khống Chế", note: "Giảm tỉ lệ dính debuff/khống chế." },
  { code: "LUCK", label: "Vận Khí", note: "Ảnh hưởng luyện cổ, trinh sát và một số roll may mắn." },
  { code: "ALL", label: "Toàn Chỉ Số", note: "Áp dụng đồng thời cho Công/Thủ/Máu/Tốc." },
  { code: "MANA", label: "Chân Nguyên", note: "Tài nguyên thi triển kỹ năng." },
  { code: "MOVEMENT", label: "Di Chuyển", note: "Số ô tối đa mỗi lần di chuyển trên bản đồ." },
  { code: "INVESTIGATION", label: "Điều Tra", note: "Khả năng do thám/điều tra." },
  { code: "HEARING", label: "Thính Giác", note: "Phát hiện mục tiêu/ẩn thân." },
  { code: "DISGUISE", label: "Ngụy Trang", note: "Mức che giấu khí tức, chống trinh sát." },
  { code: "TIME_SPEED", label: "Tốc Độ Thời Gian", note: "Hệ số trôi thời gian trong Không Khiếu." },
  { code: "REFINE_SUCCESS", label: "Tỷ Lệ Luyện Cổ", note: "Tăng/giảm cơ hội hợp luyện." },
  { code: "REFINE_MANA_TIER", label: "Tinh Luyện Chân Nguyên", note: "Cấp tinh luyện, cộng sát thương kỹ năng." },
  { code: "STORAGE", label: "Sức Chứa", note: "Số ô/khối lượng lưu trữ." },
  { code: "SOUL_REFINEMENT", label: "Luyện Hồn", note: "Hiệu quả liên quan Hồn đạo." },
  { code: "CLEANSING_POWER", label: "Thanh Tẩy", note: "Khả năng giải trạng thái xấu." },
  { code: "DAO_MARK", label: "Đạo Ngân", note: "Điểm sức mạnh lưu phái." },
  { code: "PATH_CONFLICT_REDUCTION", label: "Giảm Xung Đột Đạo", note: "Giảm phản phệ khi đa tu." },
  { code: "PATH_RESISTANCE", label: "Kháng Lưu Phái", note: "Giảm sát thương theo lưu phái cụ thể." },
];

export const EFFECT_RESOURCES: GlossaryTarget[] = [
  { code: "HP", label: "Sinh Lực", note: "Máu." },
  { code: "MANA", label: "Chân Nguyên", note: "Năng lượng thi triển Cổ." },
  { code: "SOUL", label: "Hồn Phách", note: "Tài nguyên Hồn đạo." },
  { code: "THOUGHT", label: "Niệm Lực", note: "Tài nguyên Trí đạo." },
  { code: "APTITUDE", label: "Tư Chất", note: "Phẩm chất Không Khiếu." },
  { code: "LIFESPAN", label: "Thọ Nguyên", note: "Tuổi thọ còn lại." },
  { code: "STRESS", label: "Áp Lực Không Khiếu", note: "Căng thẳng làm giảm hiệu suất tu luyện." },
  { code: "PRIMEVAL_STONE", label: "Nguyên Thạch", note: "Tiền tệ phàm giới." },
  { code: "IMMORTAL_STONE", label: "Tiên Nguyên Thạch", note: "Tiền tệ tiên giới." },
];

export type HookStatus = "active" | "partial" | "legacy" | "unimplemented";
export type HookGroup = "combat" | "steal" | "growth" | "stat" | "world";

export const HOOK_STATUS_LABELS: Record<HookStatus, string> = {
  active: "Đang hoạt động",
  partial: "Hoạt động một phần",
  legacy: "Tên cũ / alias",
  unimplemented: "Chưa có hiệu lực",
};

export const HOOK_GROUP_LABELS: Record<HookGroup, string> = {
  combat: "Chiến đấu",
  steal: "Thâu đoạt & phòng vệ",
  growth: "Tu luyện, tài nguyên & tiêu hao",
  stat: "Chỉ số thụ động",
  world: "Thế giới & tiện ích",
};

export interface NativeHookEntry {
  code: string;
  label: string;
  trigger: string;
  status: HookStatus;
  group: HookGroup;
  behavior: string;
}

export const NATIVE_HOOKS: NativeHookEntry[] = [
  // ── Chiến đấu ──────────────────────────────────────────────────────────
  { code: "INSTANT_WIN", label: "Chiến Thắng Tuyệt Đối", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Đầu trận hạ toàn bộ phe địch ngay Lượt 0, bỏ qua khiên, miễn nhiễm và Bất Tử." },
  { code: "SEIZE_INITIATIVE", label: "Đoạt Tiên Cơ", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Lượt 1 được đi trước nếu thọ nguyên địch chính không vượt quá lifespan_ratio lần thọ nguyên của mình (mặc định 3)." },
  { code: "SUPPRESS_PRIORITY", label: "Cấm Cổ Trấn Áp", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Vô hiệu Tiên Thủ/Né Lượt 1 của địch; nếu chặn được hiệu ứng địch thì người mang được đi trước." },
  { code: "SAT_KHI_GUARANTEE", label: "Sát Khí Đồ Sát", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Khi Sát Khí trên 10 tầng: xuyên khiên, không thể bị né, phá cả Bất Hoại lẫn Bất Tử." },
  { code: "REVERSE_CALAMITY", label: "Đảo Ngược Tai Ách", trigger: "COMBAT_PASSIVE", status: "active", group: "combat", behavior: "Mọi trạng thái xấu, khống chế, Lãng Quên và cướp Cổ nhắm vào chủ đều bị đảo về hư vô; chỉ sát thương đi qua." },
  { code: "REBORN", label: "Hồi Sinh Khi Chết", trigger: "ON_DEATH", status: "active", group: "combat", behavior: "Xác suất sống lại: tua về Lượt 1 hoặc đứng dậy tại chỗ với % HP, xóa debuff và buff thêm. Mỗi trận tối đa 5 lần hồi sinh tua lại." },
  { code: "FIRST_TURN_EVADE", label: "Né Lượt Đầu", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Né hoàn toàn đòn đánh đầu tiên ở Lượt 1; đòn nhiều hit/AoE sau đó vẫn trúng. Bị Tất Trúng hoặc Cấm Cổ vô hiệu." },
  { code: "GUARANTEED_HIT", label: "Tất Trúng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Đòn không thể bị né hoặc làm hụt, phá cả Né Lượt Đầu và hiệu ứng giảm độ chính xác." },
  { code: "POEM_INTENT", label: "Thi Ý", trigger: "ON_TURN_START", status: "active", group: "combat", behavior: "Mỗi lượt +1 Thi Ý và giảm 40% sát thương nhận; 3 tầng xuyên 30% khiên mỗi lượt, 10 tầng phá sạch khiên địch." },
  { code: "BORROW_PAST_POWER", label: "Mượn Lực Quá Khứ", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Tăng 40% sát thương gây ra trong 3 lượt." },
  { code: "SWORD_DRAGON_TRANSFORM", label: "Kiếm Long Biến", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Chuyển 1 HP thành Khiên Ảo và tăng 50% sát thương trong 3 lượt." },
  { code: "FULL_SOUL_HEAL", label: "Đầy Hồn Phục", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Hồi đầy HP nhưng mất lượt hành động hiện tại." },
  { code: "SUMMON_WOLF_FRONTLINE", label: "Triệu Hồi Sói Tiền Hàng", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Triệu hồi Địa Lang Vương có 50% MaxHP, 1/3 sát thương và 1/2 phòng thủ của chủ; đẩy chủ về hàng giữa." },
  { code: "SUMMON_MONSTER", label: "Triệu Hồi Quái", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Triệu hồi quái theo monster_id, mạnh dần theo rank Cổ và Đạo Ngân; giới hạn 2× HP và 3× sát thương của chủ." },
  { code: "FORGET_INTENT_INTERRUPT", label: "Đoán Ý & Ngắt Chiêu", trigger: "BEFORE_ENEMY_ATTACK", status: "active", group: "combat", behavior: "Đòn hiện tại vẫn trúng nhưng phong ấn Cổ địch N lượt; vượt số lần thành công tối đa thì tự bị phong ấn." },
  { code: "FORGET_WILL_CONFUSE", label: "Ý Niệm Hỗn Loạn", trigger: "BEFORE_ENEMY_ATTACK", status: "active", group: "combat", behavior: "Có thể khiến đòn địch đánh lạc sang chính nó hoặc hụt hẳn; có thể phản phệ phong ấn Cổ của mình." },
  { code: "UNRAVEL_RIDDLE", label: "Giải Đoán Câu Đố", trigger: "ON_CAST", status: "partial", group: "combat", behavior: "Xóa tối đa 2 buff của mục tiêu đầu tiên; phần 'ngắt chiêu tụ lực' hiện chỉ ghi log." },
  { code: "CHAOS_GAMBLE", label: "Hỗn Độn Đánh Cược", trigger: "ON_CAST", status: "partial", group: "combat", behavior: "Ngẫu nhiên một trong ba: ×5 sát thương đòn này, Bất Hoại 1 lượt, hoặc lấy đi 90% HP hiện tại của địch. 'Cái giá' hiện chưa bị trừ." },
  { code: "PERSIST_STATUS_ON_HIT", label: "Lưu Trạng Thái Khi Trúng Đòn", trigger: "ON_HIT", status: "partial", group: "combat", behavior: "Đòn trúng trong PvP thật gán vĩnh viễn một trạng thái lên địch, mỗi trận một lần." },
  { code: "IMMUNE_DREAM_STATUS", label: "Miễn Nhiễm Mộng Đạo", trigger: "ALWAYS_ON", status: "active", group: "combat", behavior: "Miễn nhiễm tuyệt đối mọi trạng thái thuộc Mộng đạo." },
  { code: "CLEANSE_REGIONAL_WALL", label: "Phá Áp Chế Giới Vực", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Xóa Áp Chế Giới Vực của bản thân ngay đầu trận trước khi tính chỉ số." },
  { code: "FAKE_REGIONAL_WALL", label: "Ngụy Áp Chế Giới Vực", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "PvP: giảm nửa Áp Chế Giới Vực tác động lên địch trong trận; sau trận gán Áp Chế lên đối thủ 15 + 5×rank phút." },

  // ── Thâu đoạt & phòng vệ ──────────────────────────────────────────────
  { code: "PERM_STEAL_GU", label: "Thâu Cổ Vĩnh Viễn", trigger: "ON_BATTLE_START", status: "partial", group: "steal", behavior: "PvP thật: cướp vĩnh viễn một Cổ đang trang bị của địch; tỉ lệ ~25% +10%/rank chênh, Cổ Bản Mệnh khó hơn." },
  { code: "TEMP_STEAL_GU", label: "Thâu Cổ Tạm Thời", trigger: "ON_BATTLE_START", status: "partial", group: "steal", behavior: "PvP thật: đoạt tạm một Cổ để dùng trong trận; tỉ lệ ~40% +5%/rank chênh, hết trận trả lại." },

  // ── Tu luyện, tài nguyên & tiêu hao ──────────────────────────────────
  { code: "CULTIVATION_EFFICIENCY", label: "Hiệu Suất Tu Luyện", trigger: "ALWAYS_ON", status: "active", group: "growth", behavior: "Cộng % tiến độ tu luyện; chỉ tính khi Cổ ACTIVE cùng rank người chơi, nhiều Cổ khác ID cộng dồn." },
  { code: "DEDUCTION_INSTANT_FINISH", label: "Thôi Diễn Tức Thành", trigger: "ALWAYS_ON", status: "active", group: "growth", behavior: "Mỗi lần thôi diễn có thêm cơ hội hoàn thành ngay; tỉ lệ tăng theo chênh rank Cổ với mục tiêu, vẫn trừ chi phí." },
  { code: "GROWTH_DURATION_HOURS", label: "Thời Gian Trưởng Thành", trigger: "ALWAYS_ON", status: "active", group: "growth", behavior: "Đặt số giờ đeo Cổ để đạt chỉ số trưởng thành tối đa; tiến độ tăng tuyến tính." },
  { code: "MANA_REGEN_TICK", label: "Hồi Chân Nguyên Định Kỳ", trigger: "ON_TICK", status: "active", group: "growth", behavior: "Hồi thêm Chân Nguyên mỗi giờ (mặc định 0.5), giảm theo chênh rank Cổ so với người chơi." },
  { code: "THOUGHT_REGEN_TICK", label: "Hồi Niệm Lực Định Kỳ", trigger: "ON_TICK", status: "active", group: "growth", behavior: "Hồi thêm Niệm Lực mỗi giờ (mặc định 1), giảm theo chênh rank Cổ." },
  { code: "SOUL_REGEN_TICK", label: "Hồi Hồn Phách Định Kỳ", trigger: "ON_TICK", status: "active", group: "growth", behavior: "Hồi thêm Hồn HP mỗi giờ (mặc định 1), không vượt Hồn tối đa." },
  { code: "GENERATE_STONES_TICK", label: "Ngưng Tụ Nguyên Thạch", trigger: "ON_TICK", status: "active", group: "growth", behavior: "Cộng Nguyên Thạch mỗi 3 giờ; không giảm theo rank." },
  { code: "GENERATE_IMMORTAL_STONES_TICK", label: "Ngưng Tụ Tiên Nguyên Thạch", trigger: "ON_TICK", status: "active", group: "growth", behavior: "Cộng Tiên Nguyên Thạch mỗi 3 giờ; không giảm theo rank." },
  { code: "ADD_LIFESPAN_YEARS", label: "Tăng Thọ Nguyên", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Cộng vĩnh viễn số năm thọ nguyên; Cổ tiêu hao biến mất." },
  { code: "ADD_SOUL_MAX", label: "Tăng Hồn Phách Tối Đa", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Cộng vĩnh viễn Hồn tối đa (nền 10 nếu đang bằng 0)." },
  { code: "ADD_THOUGHT_MAX", label: "Tăng Niệm Lực Tối Đa", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Cộng vĩnh viễn Niệm Lực tối đa (nền 10 nếu đang bằng 0)." },
  { code: "RESTORE_SOUL_HP", label: "Hồi Hồn HP", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Hồi Hồn HP (mặc định 10), không vượt Hồn tối đa." },
  { code: "RESTORE_MENTAL", label: "Hồi Niệm Lực", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Hồi Niệm Lực (mặc định đầy), không vượt mức tối đa." },
  { code: "REDUCE_STRESS", label: "Giảm Áp Lực Không Khiếu", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Giảm Áp Lực Không Khiếu (mặc định 5), sàn 0." },
  { code: "INCREASE_TIME_SPEED", label: "Tăng Tốc Thời Gian", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Cộng vĩnh viễn vào hệ số tốc độ thời gian của Không Khiếu (mặc định +0.1)." },
  { code: "DECREASE_TIME_SPEED", label: "Giảm Tốc Thời Gian", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Giảm hệ số tốc độ thời gian, sàn 0.1. Lưu ý: dữ liệu Nhật/Nguyệt Cổ đang khai báo giá trị âm nên thực tế chỉ giảm 0.1." },
  { code: "RANK_PROGRESS", label: "Tăng Tiểu Cảnh Giới", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Tăng 1 tiểu cảnh giới khi rank Cổ trùng rank người chơi và chưa Đỉnh Phong; bị chặn khi Không Khiếu hóa đá." },
  { code: "MAN_BEAST_BURIAL", label: "Nhân Thú Táng Sinh", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Rank 1–4 đang Đỉnh Phong: tăng 1 đại cảnh giới, về Sơ Giai và giảm 10 Chân Nguyên tối đa (sàn 20)." },
  { code: "SACRIFICE_APTITUDE", label: "Hiến Tế Tư Chất", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Tăng tư chất Không Khiếu (mặc định 5, trần 99) và đổi loại tư chất; Cổ không mất, chỉ giảm 20 độ no." },
  { code: "SOVEREIGN_BODY_TRANSFORM", label: "Tiên Thể Chí Tôn", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Vĩnh viễn miễn nhiễm xung đột Đạo Ngân; chỉ dùng được một lần." },
  { code: "STONE_APERTURE", label: "Hóa Đá Không Khiếu", trigger: "ON_CONSUME", status: "legacy", group: "growth", behavior: "Tên cũ của APERTURE_PETRIFICATION: đẩy lên Đỉnh Phong nhưng hóa đá 12h × 2^(rank−1); không hồi Chân Nguyên, không đột phá trong thời gian đó." },
  { code: "THIRD_WATCH", label: "Tam Canh", trigger: "ON_CONSUME", status: "legacy", group: "growth", behavior: "Tên cũ của TIME_ACCELERATION: nhân hệ số tốc độ thời gian (mặc định ×3), cộng dồn mỗi lần dùng." },
  { code: "RECORD_MESSAGE", label: "Ghi Lại Hình Ảnh", trigger: "ON_USE", status: "legacy", group: "growth", behavior: "Tên cũ của RECORD: chỉ ghi log mô tả, không lưu dữ liệu hình ảnh/âm thanh." },
  { code: "POISON_VOW", label: "Độc Thệ", trigger: "ON_CONSUME", status: "legacy", group: "growth", behavior: "Tên cũ của CONTRACT_OATH: chỉ ghi log kết khế ước, chưa có cơ chế ràng buộc thực tế." },
  { code: "SPRING_EGG_HATCH", label: "Trứng Nguyên Tuyền", trigger: "ON_CONSUME", status: "partial", group: "growth", behavior: "Chỉ hoạt động qua vật phẩm Trứng Nguyên Tuyền (cấy Phàm Tuyền tại chỗ); dùng như Cổ sẽ báo chưa hỗ trợ." },

  // ── Chỉ số thụ động ───────────────────────────────────────────────────
  { code: "STAT_MODS", label: "Cộng Chỉ Số", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Bảng tỉ lệ chỉ số (0.1 = +10%). Chỉ đọc từ chủng tộc/thể chất/trạng thái; gắn trên Cổ thì bị bỏ qua — Cổ dùng op STAT_MOD." },
  { code: "LUCK", label: "Vận Khí", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Cộng Vận Khí từ chủng tộc/trạng thái. Trên Cổ phải dùng STAT_MOD LUCK." },
  { code: "CC_RESIST", label: "Kháng Khống Chế", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Giảm tỉ lệ dính debuff/khống chế; trên Cổ phải dùng STAT_MOD CC_RESIST." },
  { code: "TENACITY", label: "Kiên Cường", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Tỉ lệ tự giải debuff đầu lượt và rút ngắn khống chế; trên Cổ phải dùng STAT_MOD TENACITY." },
  { code: "HP_REGEN", label: "Hồi Sinh Lực", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Hồi HP mỗi giờ ngoài trận; chỉ đọc từ chủng tộc/trạng thái, gắn trên Cổ bị bỏ qua." },
  { code: "MENTAL_RESISTANCE", label: "Kháng Tinh Thần", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Tăng kháng khống chế tinh thần; chỉ đọc từ chủng tộc/trạng thái." },
  { code: "SOUL_DEFENSE", label: "Phòng Ngự Hồn", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Hiện chỉ ghi nhận vào chỉ số, chưa có công thức sát thương nào trừ theo." },
  { code: "DAO_MARK_GAIN", label: "Nhận Đạo Ngân", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Chỉ nhân Đạo Ngân nhận từ tai kiếp (đọc theo thể chất); gắn trên Cổ bị bỏ qua." },
  { code: "DAO_MARK_MODIFIER", label: "Gia Tăng Đạo Ngân", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Hệ số Đạo Ngân theo lưu phái (0.2 = ×1.2); chỉ đọc từ chủng tộc/trạng thái." },
  { code: "PATH_DAMAGE_MODIFIER", label: "Tăng Sát Thương Lưu Phái", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Cộng % sát thương theo lưu phái; chỉ đọc từ chủng tộc/trạng thái." },
  { code: "PATH_RESISTANCE_MODIFIER", label: "Kháng Sát Thương Lưu Phái", trigger: "ALWAYS_ON", status: "partial", group: "stat", behavior: "Giảm % sát thương nhận theo lưu phái; trên Cổ phải dùng STAT_MOD PATH_RESISTANCE." },

  // ── Thế giới & tiện ích ───────────────────────────────────────────────
  { code: "MARKETPLACE_ACCESS", label: "Truy Cập Chợ Trời", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Mở Bảo Hoàng Thiên: xem, đăng bán và mua vật phẩm/Cổ." },
  { code: "REMOTE_TRADE", label: "Giao Dịch Từ Xa", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Cho phép giao dịch khi hai bên ở xa; cả hai đều phải sở hữu Cổ này." },
  { code: "CHAT_CONNECT", label: "Kết Nối Thiên Cơ", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Cùng Cảm Ứng Thiên Cơ mở kênh chat chung; chỉ cần nằm trong túi." },
  { code: "CHAT_PERCEIVE", label: "Cảm Ứng Thiên Cơ", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Cùng Kết Nối Thiên Cơ mở kênh chat chung; thiếu một trong hai thì không gửi/đọc được." },
  { code: "PORTAL_CREATE", label: "Tạo Cổng Dịch Chuyển", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Dịch chuyển tới toạ độ tự chọn trong tầm range; không khai range thì dùng bảng tầm theo rank Cổ." },
  { code: "RANDOM_PORTAL", label: "Cổng Ngẫu Nhiên", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Dịch chuyển ngẫu nhiên trong tầm range; không khai range thì đi ngẫu nhiên khắp bản đồ." },
  { code: "IGNORE_MOVEMENT_RANGE", label: "Vượt Giới Hạn Di Chuyển", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Di chuyển không giới hạn số ô khi trang bị; Cổng không khai range cũng thành vô hạn." },
  { code: "LOOT_PREVIEW", label: "Thấu Thị Chiến Lợi Phẩm", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Xem trước bảng rơi đồ (vật phẩm, tỉ lệ, số lượng) khi chạm quái." },
  { code: "CAPTURE_BONUS", label: "Tăng Tỷ Lệ Bắt Thú", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Cộng thêm tỉ lệ bắt thú; cộng dồn nhiều Cổ, trần 100%, cần Cổ rank không thấp hơn quái." },
  { code: "DISGUISE", label: "Ngụy Trang Khí Tức", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Bật ngụy trang: chỉ có thể giả rank thấp hơn, tăng khả năng chống trinh sát." },
  { code: "PERFECT_DISGUISE", label: "Ngụy Trang Hoàn Mỹ", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Giả được rank 1–9 (kể cả cao hơn) và giả Bản Mệnh Cổ; hồ sơ hiện dữ liệu giả." },
  { code: "CHANGE_FORM_DISGUISE", label: "Biến Hóa Ngụy Trang", trigger: "CAPABILITY", status: "partial", group: "world", behavior: "Mở khóa đặt Bản Mệnh Cổ giả; muốn ẩn thân vẫn cần Ngụy Trang Khí Tức." },
  { code: "ARENA_IDENTITY_MASK", label: "Mặt Nạ Đấu Trường", trigger: "CAPABILITY", status: "unimplemented", group: "world", behavior: "Chưa có hiệu lực: chưa ẩn tên/ẩn danh trong đấu trường." },
];

export interface CombatStatusEntry {
  id: string;
  label: string;
  category: string;
  behavior: string;
  turns: string;
}

export const COMBAT_STATUSES: CombatStatusEntry[] = [
  { id: "BLEED", label: "Xuất Huyết", category: "Sát thương theo thời gian", behavior: "Mỗi lượt mất HP theo giá trị trạng thái (khiên ăn trước).", turns: "3" },
  { id: "POISON", label: "Độc", category: "Sát thương theo thời gian", behavior: "Mỗi lượt mất HP theo giá trị trạng thái.", turns: "3" },
  { id: "BURN", label: "Thiêu Đốt", category: "Sát thương theo thời gian", behavior: "Mỗi lượt mất HP theo giá trị trạng thái.", turns: "3" },
  { id: "SLOW_DAMAGE", label: "Tê Cóng", category: "Sát thương theo thời gian", behavior: "Mỗi lượt mất HP và giảm tốc độ.", turns: "3" },
  { id: "REGEN", label: "Hồi Phục", category: "Hồi phục", behavior: "Mỗi lượt hồi HP theo giá trị trạng thái, không vượt MaxHP.", turns: "3" },
  { id: "STUN", label: "Choáng", category: "Khống chế tinh thần", behavior: "Mất lượt hoàn toàn.", turns: "1" },
  { id: "FREEZE", label: "Đóng Băng", category: "Khống chế tinh thần", behavior: "Mất lượt, vỡ ngay khi bị đánh.", turns: "1" },
  { id: "SLEEP", label: "Ngủ", category: "Khống chế tinh thần", behavior: "Mất lượt, vỡ khi bị đánh.", turns: "2" },
  { id: "BIND", label: "Trói", category: "Khống chế tinh thần", behavior: "Mất lượt.", turns: "3" },
  { id: "SILENCE", label: "Khóa Chiêu", category: "Khống chế tinh thần", behavior: "Chỉ được đánh thường.", turns: "2" },
  { id: "FEAR", label: "Khiếp Sợ", category: "Khống chế tinh thần", behavior: "Chỉ được đánh thường.", turns: "2" },
  { id: "CONFUSE", label: "Mê Loạn", category: "Khống chế tinh thần", behavior: "40% xác suất đánh lệch mục tiêu.", turns: "2" },
  { id: "CHARM", label: "Mê Hoặc", category: "Khống chế tinh thần", behavior: "Quay sang đánh phe mình.", turns: "3" },
  { id: "BLIND", label: "Mù", category: "Khống chế tinh thần", behavior: "Giảm độ chính xác, dễ đánh hụt.", turns: "2" },
  { id: "WEAK", label: "Suy Yếu", category: "Khống chế tinh thần", behavior: "Giảm % sát thương gây ra.", turns: "3" },
  { id: "SLOW", label: "Làm Chậm", category: "Khống chế tinh thần", behavior: "Giảm % tốc độ.", turns: "3" },
  { id: "SLOW1", label: "Làm Chậm Tuyệt Đối", category: "Buff", behavior: "Giảm % tốc độ, không bị kháng.", turns: "3" },
  { id: "ATK_UP", label: "Tăng Công", category: "Buff", behavior: "Tăng % sát thương gây ra.", turns: "3" },
  { id: "DEF_UP", label: "Tăng Thủ", category: "Buff", behavior: "Tăng % phòng thủ.", turns: "3" },
  { id: "THORNS", label: "Phản Sát Thương", category: "Buff", behavior: "Phản lại % sát thương nhận vào.", turns: "3" },
  { id: "TAUNT", label: "Khiêu Khích", category: "Buff", behavior: "Buộc địch đánh vào người mang.", turns: "2" },
  { id: "STEALTH", label: "Ẩn Thân", category: "Buff", behavior: "Không bị chọn bởi đòn đơn mục tiêu.", turns: "2" },
  { id: "HASTE", label: "Tăng Tốc", category: "Buff", behavior: "Tăng % tốc độ.", turns: "3" },
  { id: "FLYING", label: "Phi Hành", category: "Bản đồ", behavior: "Bay trên bản đồ, bỏ qua quái mặt đất; không tác dụng trong combat.", turns: "Vĩnh viễn" },
  { id: "PHI_HANH", label: "Phi Hành (alias)", category: "Bản đồ", behavior: "Tên cũ của FLYING, cùng hiệu ứng bản đồ.", turns: "Vĩnh viễn" },
  { id: "ARMOR_BREAK", label: "Phá Giáp", category: "Debuff", behavior: "Giảm % phòng thủ.", turns: "3" },
  { id: "DAMAGE_REDUCTION_UP", label: "Giảm Sát Thương", category: "Buff", behavior: "Giảm % sát thương nhận; lấy giá trị cao nhất, không cộng dồn.", turns: "3" },
  { id: "INVINCIBLE", label: "Bất Hoại", category: "Bất tử", behavior: "Miễn nhiễm mọi sát thương; bị Sát Khí trên 10 tầng xuyên qua.", turns: "1" },
  { id: "LAST_STAND", label: "Bất Tử", category: "Bất tử", behavior: "Vẫn nhận sát thương nhưng HP không xuống dưới 1.", turns: "1" },
  { id: "EVASION", label: "Né Tránh", category: "Buff", behavior: "Tỉ lệ né đòn, trần 75%, cộng dồn bằng max.", turns: "1" },
  { id: "REVIVE_LOCK", label: "Khóa Hồi Sinh", category: "Bất tử", behavior: "Chặn mọi hiệu ứng hồi sinh.", turns: "3" },
  { id: "ESSENCE_LEAK", label: "Rò Rỉ Chân Nguyên", category: "Không Khiếu", behavior: "Mỗi lượt mất 5% Chân Nguyên tối đa.", turns: "3" },
  { id: "GU_SEAL", label: "Khóa Cổ", category: "Không Khiếu", behavior: "Không dùng được kỹ năng Cổ, chỉ đánh thường.", turns: "3" },
  { id: "SOUL_WOUND", label: "Hồn Thương", category: "Xung đột đạo", behavior: "Gây sát thương thực ngay khi dính (khiên ăn trước).", turns: "3" },
  { id: "THOUGHT_CHAOS", label: "Tạp Niệm", category: "Không Khiếu", behavior: "Cờ debuff đã gán; engine chưa xử lý riêng.", turns: "3" },
  { id: "DAO_CONFLICT", label: "Xung Đột Đạo Ngân", category: "Xung đột đạo", behavior: "Cờ debuff đã gán; engine chưa xử lý riêng.", turns: "3" },
  { id: "BACKLASH", label: "Phản Phệ", category: "Bất tử", behavior: "Gây sát thương ngay khi dính.", turns: "1" },
  { id: "FORGET_EGO", label: "Lãng Quên Bản Ngã", category: "Khống chế tinh thần", behavior: "Phong ấn Cổ, không dùng được kỹ năng Cổ.", turns: "1" },
  { id: "FORGET_REALITY", label: "Lãng Quên Thực Tại", category: "Khống chế tinh thần", behavior: "Giảm % sát thương gây ra.", turns: "2" },
];

export interface PersistentStatusEntry {
  key: string;
  name: string;
  description: string;
  duration: string;
  blocked?: string;
}

export const PERSISTENT_STATUSES: PersistentStatusEntry[] = [
  { key: "severely_wounded", name: "Trọng Thương", description: "Thân trúng trọng thương; giảm mạnh hợp luyện và một số hoạt động.", duration: "Đến khi hồi phục", blocked: "Không thể thám hiểm" },
  { key: "poisoned", name: "Trúng Độc", description: "Độc tố xâm nhập cơ thể.", duration: "30 phút" },
  { key: "exhausted", name: "Kiệt Sức", description: "Lực kiệt, không thể tu luyện.", duration: "10 phút", blocked: "Không thể tu luyện" },
  { key: "blessed", name: "Phúc Trạch", description: "Nhận phúc trạch, tăng may mắn.", duration: "60 phút" },
  { key: "death_penalty", name: "Hình Phạt Tử Vong", description: "Sát khí chưa tan sau khi chết.", duration: "30 phút" },
  { key: "refine_fatigue", name: "Tâm Thần Mệt Mỏi", description: "Giảm 50% tỉ lệ luyện cổ thành công.", duration: "10 phút" },
  { key: "refine_backfire", name: "Luyện Cổ Phản Phệ", description: "Giảm 95% tỉ lệ luyện cổ thành công.", duration: "20 phút" },
  { key: "sat_khi", name: "Sát Khí Đồ Sát", description: "Tích tụ oán khí đồ sát vĩnh viễn; khi bại trận rơi thêm 50–250% tài sản theo tầng Sát Khí.", duration: "Vĩnh viễn" },
  { key: "regional_wall_suppression", name: "Áp Chế Giới Vực", description: "Chịu áp chế khi vượt Giới Vực Bích; tu vi càng cao càng nặng.", duration: "15 phút" },
  { key: "pvp_immunity", name: "Miễn Chiến Hồi Sinh", description: "Thiên Đạo che chở sau hồi sinh, miễn mọi đòn PvP; chủ động khai chiến sẽ phá vỡ.", duration: "10 phút" },
  { key: "refine_insight", name: "Luyện Cổ Tâm Đắc R1–R9", description: "Mỗi lần hợp luyện đúng công thức thất bại tích 1 tầng (+1% tỉ lệ luyện rank đó); luyện thành công reset về 0.", duration: "Vĩnh viễn" },
  { key: "sinh_khi_tan", name: "Sinh Khí Tàn", description: "Sinh cơ tàn lụi theo tầng I–V (−10% đến −99% công/thủ/tốc); mỗi đòn trúng +1 tầng, thanh tẩy −1 tầng.", duration: "5 giờ" },
  { key: "aperture_petrified", name: "Không Khiếu Hóa Đá", description: "Không tự hồi Chân Nguyên và không thể đột phá; tự hồi phục sau 12h × 2^(rank−1).", duration: "Theo rank", blocked: "Không thể đột phá" },
  { key: "phi_hanh", name: "Phi Hành", description: "Đang bay; miễn nhiễm quái dưới đất khi di chuyển, không bị đuổi theo.", duration: "Đến khi hạ cánh" },
];
