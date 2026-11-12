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
  { code: "ON_TURN_START", label: "Đầu Lượt", group: "event", when: "Kích hoạt ở đầu lượt hành động của đơn vị trong trận đấu theo lượt.", note: "Thường dùng để hồi Chân Nguyên, tích nộ khí hoặc phát tác độc thương." },
  { code: "ON_DEATH", label: "Khi Tử Vong", group: "event", when: "Chạy khi HP về 0, trước khi trận đấu kết thúc.", note: "Có thể hồi sinh và tiếp tục trận." },
  { code: "ON_TICK", label: "Mỗi Chu Kỳ", group: "scheduled", when: "Chạy theo worker định kỳ (3 giờ/lần cho tài nguyên) hoặc theo giờ cho hồi phục.", note: "Không chạy trong lúc giao chiến." },
  { code: "CAPABILITY", label: "Năng Lực", group: "capability", when: "Không tự kích hoạt; hệ thống kiểm tra khi người chơi dùng một chức năng (chợ, chat, cổng, bắt thú...).", note: "Cổ chỉ cần nằm trong túi là đủ, trừ khi ghi chú khác." },
  { code: "COMBAT_PASSIVE", label: "Nội Tại Chiến Đấu", group: "event", when: "Hiệu ứng bị động được engine đọc trong trận (ví dụ nhân đôi sát thương, đảo tai ách).", note: "Khác ALWAYS_ON: chỉ có ý nghĩa khi đang giao chiến." },
  { code: "PASSIVE", label: "Nội Tại (tên cũ)", group: "event", when: "Tên cũ của COMBAT_PASSIVE, dữ liệu cũ vẫn dùng.", note: "Dữ liệu mới nên dùng COMBAT_PASSIVE." },
  { code: "ON_BATTLE_START", label: "Vào Trận", group: "event", when: "Kích hoạt ngay khi bắt đầu bước vào trận đấu theo lượt.", note: "Thường dùng để tạo khiên mở màn, tăng tốc độ hiệp đầu hoặc thi triển trận pháp." },
  { code: "BEFORE_ENEMY_ATTACK", label: "Trước Khi Địch Đánh", group: "event", when: "Chạy ngay trước đòn của kẻ địch nhắm vào người mang." },
  { code: "ON_CAST", label: "Khi Niệm Chiêu", group: "event", when: "Chạy khi Cổ sư thôi động Cổ trong trận, trước khi đòn đánh được xử lý.", note: "Dùng cho biến thân/buff/danh sách chiêu đặc biệt." },
  { code: "ON_CONSUME", label: "Khi Nuốt Cổ", group: "action", when: "Chạy khi dùng Cổ Tiêu Hao qua túi đồ; Cổ biến mất sau khi hiệu ứng áp dụng.", note: "Cần Cổ đủ no, đủ Chân Nguyên và đúng điều kiện cảnh giới." },
  { code: "ON_USE", label: "Khi Kích Hoạt", group: "action", when: "Chạy khi dùng Cổ Hỗ Trợ hoặc Sát Chiêu ngoài trận (nút Dùng).", note: "Sát Chiêu hỗ trợ hồi HP, hoàn Chân Nguyên, Phi Hành, Ẩn Thân tăng DISGUISE và HASTE tăng tốc độ trên bản đồ; effect khác cần executor world riêng." },
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
  legacy: "Legacy — chỉ trong trận theo lượt cũ",
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
  { code: "INSTANT_WIN", label: "Chiến Thắng Tuyệt Đối", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Đầu trận hạ toàn bộ phe địch ngay Lượt 0, bỏ qua khiên, miễn nhiễm và Bất Tử." },
  { code: "SEIZE_INITIATIVE", label: "Đoạt Tiên Cơ", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Lượt 1 được đi trước nếu thọ nguyên địch chính không vượt quá lifespan_ratio lần thọ nguyên của mình (mặc định 3)." },
  { code: "SUPPRESS_PRIORITY", label: "Cấm Cổ Trấn Áp", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Vô hiệu Tiên Thủ/Né Lượt 1 của địch; nếu chặn được hiệu ứng địch thì người mang được đi trước." },
  { code: "SAT_KHI_GUARANTEE", label: "Sát Khí Đồ Sát", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Khi Sát Khí trên 10 tầng: xuyên khiên, không thể bị né, phá cả Bất Hoại lẫn Bất Tử." },
  { code: "REVERSE_CALAMITY", label: "Đảo Ngược Tai Ách", trigger: "COMBAT_PASSIVE", status: "active", group: "combat", behavior: "Mọi trạng thái xấu, khống chế, Lãng Quên và cướp Cổ nhắm vào chủ đều bị đảo về hư vô; chỉ sát thương đi qua." },
  { code: "REBORN", label: "Hồi Sinh Khi Chết", trigger: "ON_DEATH", status: "active", group: "combat", behavior: "Xác suất sống lại: tua về Lượt 1 hoặc đứng dậy tại chỗ với % HP, xóa debuff và buff thêm. Mỗi trận tối đa 5 lần hồi sinh tua lại." },
  { code: "FIRST_TURN_EVADE", label: "Né Lượt Đầu", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Né hoàn toàn đòn đánh đầu tiên ở Lượt 1; đòn nhiều hit/AoE sau đó vẫn trúng. Bị Tất Trúng hoặc Cấm Cổ vô hiệu." },
  { code: "GUARANTEED_HIT", label: "Tất Trúng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Đòn không thể bị né hoặc làm hụt, phá cả Né Lượt Đầu và hiệu ứng giảm độ chính xác." },
  { code: "POEM_INTENT", label: "Thi Ý", trigger: "ON_TURN_START", status: "legacy", group: "combat", behavior: "Mỗi lượt +1 Thi Ý và giảm 40% sát thương nhận; 3 tầng xuyên 30% khiên mỗi lượt, 10 tầng phá sạch khiên địch." },
  { code: "BORROW_PAST_POWER", label: "Mượn Lực Quá Khứ", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Tăng 40% sát thương gây ra trong 3 lượt." },
  { code: "SWORD_DRAGON_TRANSFORM", label: "Kiếm Long Biến", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Chuyển 1 HP thành Khiên Ảo và tăng 50% sát thương trong 3 lượt." },
  { code: "FULL_SOUL_HEAL", label: "Đầy Hồn Phục", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Hồi đầy HP nhưng mất lượt hành động hiện tại." },
  { code: "SUMMON_WOLF_FRONTLINE", label: "Triệu Hồi Sói Tiền Hàng", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Triệu hồi Địa Lang Vương có 50% MaxHP, 1/3 sát thương và 1/2 phòng thủ của chủ; đẩy chủ về hàng giữa." },
  { code: "SUMMON_MONSTER", label: "Triệu Hồi Quái", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Triệu hồi quái theo monster_id, mạnh dần theo rank Cổ và Đạo Ngân; giới hạn 2× HP và 3× sát thương của chủ." },
  { code: "FORGET_INTENT_INTERRUPT", label: "Đoán Ý & Ngắt Chiêu", trigger: "BEFORE_ENEMY_ATTACK", status: "legacy", group: "combat", behavior: "Đòn hiện tại vẫn trúng nhưng phong ấn Cổ địch N lượt; vượt số lần thành công tối đa thì tự bị phong ấn." },
  { code: "FORGET_WILL_CONFUSE", label: "Ý Niệm Hỗn Loạn", trigger: "BEFORE_ENEMY_ATTACK", status: "legacy", group: "combat", behavior: "Có thể khiến đòn địch đánh lạc sang chính nó hoặc hụt hẳn; có thể phản phệ phong ấn Cổ của mình." },
  { code: "UNRAVEL_RIDDLE", label: "Giải Đoán Câu Đố", trigger: "ON_CAST", status: "partial", group: "combat", behavior: "Xóa tối đa 2 buff của mục tiêu đầu tiên; phần 'ngắt chiêu tụ lực' hiện chỉ ghi log." },
  { code: "CHAOS_GAMBLE", label: "Hỗn Độn Đánh Cược", trigger: "ON_CAST", status: "partial", group: "combat", behavior: "Ngẫu nhiên một trong ba: ×5 sát thương đòn này, Bất Hoại 1 lượt, hoặc lấy đi 90% HP hiện tại của địch. 'Cái giá' hiện chưa bị trừ." },
  { code: "LUCK_ROULETTE", label: "Cát Hung Chi Biến", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Vận Đạo: Quay thưởng khí vận (Đại Cát ×3.0 dmg + hoàn Chân Nguyên, Trung Cát ×1.5 dmg, Bình Vận, hoặc Đại Hung tự tổn 10% HP và tăng vọt tỉ lệ Đại Cát hiệp sau)." },
  { code: "LETHAL_LUCK_WARD", label: "Cẩu Thỉ Vận", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Vận Đạo: Khi nhận sát thương kết liễu, có tỉ lệ thoát chết trong gang tấc (HP giữ lại 1 và nhận thêm Chân Nguyên Giáp, 1 lần/trận)." },
  { code: "DREAM_UNRAVEL", label: "Giải Mộng Hóa Thần", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Mộng Đạo: Thanh tẩy toàn bộ khống chế & mê chướng tâm lý, chuyển hóa thành hồi Chân Nguyên và tạo Mộng Giáp hộ thể." },
  { code: "CALCULATED_STRIKE", label: "Động Tất Sơ Hở", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Trí Đạo: Nhìn thấu toàn bộ sơ hở và đường đi của địch, đòn đánh tất trúng không thể né và tăng 35% uy lực." },
  { code: "SOUL_SHOCK", label: "Đãng Hồn Kích", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Hồn Đạo: Đòn đánh chấn động thẳng vào hồn phách mục tiêu, bỏ qua Khiên Ảo và gán Hồn Thương." },
  { code: "SOUL_DEVOUR", label: "Phệ Hồn Hóa Lực", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Hồn Đạo: Cắn nuốt hồn lực địch, hồi phục 20% MaxHP và tích tầng tăng 10% uy lực tấn công (tối đa 5 tầng)." },
  { code: "BEAST_PHANTOM_STRIKE", label: "Thú Lực Hư Ảnh", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Lực Đạo: Xác suất 45% triệu hồi Cổ Thú Hư Ảnh bồi thêm một đòn bạt sơn bạt hải gây 75% sát thương thêm." },
  { code: "BERSERK_STATE", label: "Cuồng Man Bá Thể", trigger: "COMBAT_PASSIVE", status: "active", group: "combat", behavior: "Lực Đạo: Máu càng thấp chiến ý càng mạnh (tăng tới +80% sát thương và +50% kiên cường), dưới 30% HP miễn nhiễm Choáng và Đóng Băng." },
  { code: "HEAVY_LIFESTEAL", label: "Huyết Thao Thiết", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Huyết Đạo: Hóa thân huyết hải thao thiết, chuyển hóa 50% sát thương gây ra thành hồi phục sinh mệnh." },
  { code: "BLOOD_SACRIFICE", label: "Huyết Tế Cuồng Bạo", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Huyết Đạo: Tự trích 15% HP hiến tế để kích phát huyết sát, nhân đôi sát thương chiêu thức (x2.0)." },
  { code: "BLEED_BURST", label: "Huyết Bạo Trùng Điệp", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Huyết Đạo: Kích nổ toàn bộ các tầng Xuất Huyết (Bleed) trên người mục tiêu thành sát thương nổ cực mạnh." },
  { code: "MANA_STEAL", label: "Thâu Chân Nguyên", trigger: "ON_ACTION", status: "active", group: "steal", behavior: "Trộm Đạo: Diệu thủ không không, đánh cắp 25% Chân Nguyên hiện có của mục tiêu chuyển hóa cho bản thân." },
  { code: "BUFF_STEAL", label: "Thâu Thiên Hoán Nhật", trigger: "ON_ACTION", status: "active", group: "steal", behavior: "Trộm Đạo: Đoạt lấy 1 hiệu ứng có lợi (Buff) đang có của kẻ địch chuyển sang người mình." },
  { code: "TIME_REWIND", label: "Quang Âm Đảo Lưu", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Thời Đạo: Nghịch chuyển dòng sông thời gian cục bộ, hồi 35% MaxHP, 40% Chân Nguyên và tiêu trừ khống chế tinh thần." },
  { code: "PACK_FRENZY", label: "Quân Đoàn Hợp Kích", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Nô Đạo: Mỗi dị thú triệu hồi trên sân tăng thêm 15% uy lực sát thương hiệp đồng tác chiến." },
  { code: "FORM_BEAR", label: "Hóa Thân Cự Hùng", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Biến Hóa Đạo: Hóa thân Viễn Cổ Cự Hùng, tăng 30% Phòng thủ và ngưng tụ 25% MaxHP Hùng Giáp hộ thể." },
  { code: "FORM_EAGLE", label: "Hóa Thân Kim Điêu", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Biến Hóa Đạo: Hóa thân Vân Lạc Kim Điêu, giương cánh tăng vọt 40% Tốc độ và 25% Né tránh." },
  { code: "DAO_ADAPTIVE_RESIST", label: "Vạn Vật Thích Ứng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Biến Hóa Đạo: Thích ứng cơ thể với đạo ngân công kích, tăng 25% giảm thương nhận vào." },
  { code: "POISON_DETONATE", label: "Vạn Độc Bộc Phát", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Độc Đạo: Kích nổ toàn bộ các tầng kịch độc trên người đối thủ thành sát thương độc bạo liệt." },
  { code: "SWORD_ARMOR_PIERCE", label: "Vô Kiên Bất Tồi", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Kiếm Đạo: Kiếm khí ngưng tụ vô kiên bất tồi, xuyên phá 40% phòng ngự và tăng 25% tỉ lệ bạo kích." },
  { code: "EXECUTE_STRIKE", label: "Trảm Diệt Đao Ý", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Đao Đạo: Khi đối thủ suy yếu dưới 35% HP, chém ra Tuyệt Mệnh Nhất Đao nhân đôi x2.0 uy lực trảm sát." },
  { code: "BONE_SPIKE", label: "Cốt Thứ Xuyên Tâm", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Cốt Đạo: Phóng thích gai xương đâm xuyên huyết nhục mục tiêu, gây Xuất Huyết và Phá Giáp 30%." },
  { code: "BONE_THORNS", label: "Bạch Cốt Gai Khôi", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Cốt Đạo: Ngưng tụ cốt giáp gai nhọn, tăng 20% Phòng thủ, 35% Phản thương và nhận 20% MaxHP Cốt Giáp." },
  { code: "MANA_EAT", label: "Thôn Phệ Bổ Khiếu", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Thực Đạo: Thôn phệ năng lượng dư thừa, chuyển hóa thành 25% Chân Nguyên tối đa nạp đầy Không Khiếu." },
  { code: "METAL_AEGIS", label: "Cương Thiết Kim Thân", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Kim Đạo: Ngưng tụ kim quang hộ thể, tăng 35% Phòng thủ và giảm 15% sát thương nhận vào." },
  { code: "ARMOR_BREAK_HEAVY", label: "Phá Giáp Kim Mang", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Kim Đạo: Đòn đánh xuyên phá hộ giáp, giảm 40% phòng thủ mục tiêu trong 3 lượt." },
  { code: "WOOD_REGEN_BURST", label: "Sinh Sinh Bất Tức", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Mộc Đạo: Vận khởi sinh cơ, hồi 15% Chân Nguyên và hồi 10% MaxHP mỗi lượt trong 4 lượt." },
  { code: "WOOD_BIND", label: "Mộc Lao Trói Buộc", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Mộc Đạo: Cổ thụ rễ cây trói chặt mục tiêu trong 2 hiệp, phong tỏa hành động và triệt tiêu né tránh." },
  { code: "WATER_CLEANSE", label: "Thủy Lưu Triều Tịch", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Thủy Đạo: Dòng nước tinh khiết bao bọc bản thân, cuốn trôi toàn bộ độc tố, thiêu đốt và xuất huyết." },
  { code: "WATER_MIRROR_SHIELD", label: "Thủy Kính Hóa Giải", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Thủy Đạo: Ngưng tụ gương nước phản xạ, tạo 20% MaxHP Khiên Ảo và hồi phục 10% HP." },
  { code: "FIRE_INFERNO", label: "Liệt Hỏa Liêu Nguyên", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Hỏa Đạo: Ngọn lửa hừng hực thiêu đốt đối phương, mỗi lượt cắn nuốt 8% sinh lực." },
  { code: "IGNITE_BURST", label: "Nộ Hỏa Bạo Phát", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Hỏa Đạo: Nếu mục tiêu đang bị Thiêu Đốt, đòn đánh nổ tung tăng thêm 60% sát thương bạo kích." },
  { code: "EARTH_SHIELD", label: "Đại Địa Hộ Thuẫn", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Thổ Đạo: Dẫn động địa khí ngưng tụ 30% MaxHP Thổ Nham Giáp và tăng 25% phòng thủ." },
  { code: "EARTH_GRAVITY", label: "Địa Chấn Trọng Lực", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Thổ Đạo: Trọng lực đại địa nghiền ép, làm chậm 40% tốc độ mục tiêu trong 3 lượt." },
  { code: "WIND_HASTE_BURST", label: "Tật Phong Tốc Độ", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Phong Đạo: Ngự phong phi hành, thân pháp mau lẹ tăng 40% Tốc độ và 25% Né tránh." },
  { code: "LIGHTNING_STUN_BURST", label: "Lôi Đình Vạn Quân", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Lôi Đạo: Giáng cuồng lôi tăng 30% uy lực và có 40% xác suất làm Choáng mục tiêu 1 lượt." },
  { code: "CHAIN_LIGHTNING", label: "Lôi Điện Truyền Dẫn", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Lôi Đạo: Sấm sét phóng lan truyền dẫn sang mục tiêu lân cận gây thêm sát thương điện giật." },
  { code: "ICE_DEEP_FREEZE", label: "Băng Phong Vạn Lý", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Băng Đạo: Hàn khí ngưng kết đóng băng đối thủ thành tượng tuyết, mất lượt hoàn toàn." },
  { code: "ICE_FROSTBITE", label: "Hàn Băng Tê Cóng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Băng Đạo: Băng sương ăn mòn da thịt mục tiêu, vừa cấu rỉa máu vừa làm chậm 35% tốc độ." },
  { code: "LIGHT_BLINDING_FLASH", label: "Manh Mục Cực Quang", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Quang Đạo: Cực quang chói lòa chiếu thẳng vào mắt đối thủ, giảm mạnh 60% độ chính xác." },
  { code: "LIGHT_PURIFY", label: "Thánh Quang Phục Thể", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Quang Đạo: Ánh sáng ấm áp bao phủ bản thân, hồi phục 25% HP và thanh tẩy khống chế tinh thần." },
  { code: "DARK_STEALTH", label: "Ám Dạ Ẩn Nấp", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Ám Đạo: Hòa tan vào bóng đêm vô ảnh vô hình, chuẩn bị đòn đánh bạo kích (+40% crit)." },
  { code: "DARK_CURSE_WEAK", label: "Ám Ảnh Hủ Thực", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Ám Đạo: Khí tức u tối nguyền rủa, làm suy yếu 30% sát thương và ăn mòn 15% Chân Nguyên của địch." },
  { code: "STAR_FIRE_RAIN", label: "Tinh Quang Lạc Trần", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Tinh Đạo: Hàng ngàn ngôi sao sa oanh tạc phe địch (+30% damage) và tạo 15% MaxHP Khiên Tinh Quang." },
  { code: "STAR_THOUGHT_RESTORE", label: "Tinh Niệm Tụ Hồn", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Tinh Đạo: Câu thông vạn dặm tinh hà, niệm đầu thông suốt phục hồi 30% Chân Nguyên." },
  { code: "SOUND_DISRUPT_SILENCE", label: "Ma Âm Quán Nhĩ", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Âm Đạo: Sóng âm chói tai đâm xuyên màng nhĩ, khóa kỹ năng chỉ có thể đánh thường trong 2 lượt." },
  { code: "SOUND_RESONANCE", label: "Ba Động Chấn Thần", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Âm Đạo: Dao động âm ba cộng hưởng nội tạng, bỏ qua Khiên Ảo và gán Hồn Thương trực tiếp." },
  { code: "CLOUD_MIST", label: "Vân Vụ Ảo Mộng", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Vân Đạo: Mây mù huyền ảo cuộn quanh, tăng 35% Né tránh và giảm 20% sát thương nhận vào." },
  { code: "HEAVEN_SUPPRESSION", label: "Thiên Ý Khó Dò", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Thiên Đạo: Thiên Ý tổn có thừa mà bù không đủ, xóa bỏ 2 buff ngẫu nhiên của địch và tăng 35% uy lực giáng phạt." },
  { code: "TIEP_THIEN_DIA", label: "Tiếp Thiên Tiếp Địa", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Thiên Đạo: Câu thông ý chí Thiên Đạo, giải trừ toàn bộ khống chế và hồi 25% Chân Nguyên." },
  { code: "HUMAN_WILL", label: "Nhân Định Thắng Thiên", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Nhân Đạo: Nghịch cảnh sinh hy vọng, khi dưới 40% HP bùng nổ tăng 50% uy lực chiêu thức." },
  { code: "VAN_DAN_DONG_TAM", label: "Vạn Dân Đồng Tâm", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Nhân Đạo: Ngưng tụ hồng trần nhân tâm, tạo 35% MaxHP Khiên Nhân Đạo kiên cố hộ thể." },
  { code: "QI_WALL", label: "Khí Tường Hộ Thể", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Khí Đạo: Dựng Nguyên Khí Tường vô hình, cản phá 30% sát thương và tạo 30% MaxHP Khí Giáp." },
  { code: "QI_EXPLOSION", label: "Khí Bạo Trảm Thiên", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Khí Đạo: Nén ép thiên địa nguyên khí bùng nổ, sóng khí kình cuộn trào tăng 40% sát thương." },
  { code: "SPATIAL_SHIFT", label: "Hư Không Thuấn Di", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Không Đạo: Bẻ cong không gian bước vào kẽ hở hư không, né tránh 75% đòn tấn công." },
  { code: "DIMENSION_SLASH", label: "Thứ Nguyên Trảm", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Không Đạo: Đao kiếm xé rách không gian, chém xuyên phòng tuyến giáng thẳng lên mục tiêu tăng 25% uy lực." },
  { code: "RULE_IMPOSITION", label: "Định Quy Đặt Củ", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Quy Tắc Đạo: Vô quy củ bất thành phương viên, thiết lập điều luật chiến trường xóa sạch toàn bộ Buff có lợi của địch." },
  { code: "JUSTICE_VERDICT", label: "Phán Quyết Công Lý", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Quy Tắc Đạo: Thiết diện vô tư, công lý phán xét trừng phạt mục tiêu tăng thêm 35% sát thương." },
  { code: "FORMATION_DOMAIN", label: "Trận Địa Khống Chế", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Trận Đạo: Cổ Trận khởi động bao phủ sàn đấu, tăng 25% phòng thủ và tạo 20% MaxHP Trận Giáp." },
  { code: "BLINDING_MIRAGE", label: "Huyễn Ảnh Mê Trận", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Huyễn Đạo: Hư ảnh trùng trùng điệp điệp, khiến đối thủ rơi vào Mê Loạn và Mù quáng trong 2 lượt." },
  { code: "PAINTING_IMPRISONMENT", label: "Họa Địa Vi Lao", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Họa Đạo: Vung bút điểm mặc, nhốt chặt mục tiêu vào trong bức tranh thủy mặc phong tỏa hành động." },
  { code: "PASSION_OVERLOAD", label: "Tình Cổ Vô Hối", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Tình Đạo: Nghịch cảnh sinh tình, khi dưới 35% HP bộc phát 50% uy lực và kháng khống chế." },
  { code: "GU_DETONATION", label: "Tự Bạo Cổ Trùng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Luyện Đạo: Quyết đoán kích nổ Chân Nguyên Cổ trùng, tạo làn sóng xung kích hủy diệt tăng 60% sát thương." },

  // ── Sát Chiêu, Tiên Cổ Phòng & Trận Pháp (Iconic Reverend Insanity Lore) ──
  { code: "REVERSE_FLOW_SEAL", label: "Nghịch Lưu Hộ Thân Quỹ", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Sát Chiêu Thủy Đạo: Mượn uy lực Nghịch Lưu Hà phản phệ hoàn toàn 100% sát thương nhận vào ngược lại kẻ tấn công." },
  { code: "GHOST_HAND_STEAL", label: "Đâu Thiên Thâu Thủ", trigger: "ON_ACTION", status: "active", group: "steal", behavior: "Sát Chiêu Trộm Đạo: Quỷ thủ vô hình xuyên vào Không Khiếu kẻ địch, cướp đoạt 30% Chân Nguyên và bóc đi 50% Khiên Ảo." },
  { code: "LEAD_SOUL_TO_DREAM", label: "Dẫn Hồn Nhập Mộng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Mộng Đạo: Kéo hồn phách kẻ địch vào cõi mộng vĩnh hằng, gây sát thương Hồn và giam cầm mục tiêu trong Mộng Cảnh Giam Cầm." },
  { code: "MYRIAD_SELF_GIANT_HAND", label: "Vạn Ngã Đại Thủ Ấn", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Lực Đạo: Triệu hoán bàn tay khổng lồ che trời đập xuống, đập nát hoàn toàn Khiên Ảo và tăng 75% uy lực đòn đánh." },
  { code: "FAMILIAR_FACE", label: "Gặp Mặt Từng Quen Biết", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Sát Chiêu Biến Hóa Đạo: Ngụy trang dung mạo và khí tức hoàn mỹ, tăng 50% Né Tránh và đòn đánh đầu chắc chắn Bạo Kích." },
  { code: "FIVE_FINGER_SWORD", label: "Ngũ Chỉ Quyền Tâm Kiếm", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Kiếm Đạo: Năm ngón tay liên hoàn bắn kiếm khí, tăng 65% sát thương và bỏ qua 50% phòng thủ mục tiêu." },
  { code: "TIME_CUTTER", label: "Tuế Nguyệt Như Đao", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Thời Đạo: Năm tháng như đao chém thiên kiêu! Tăng 50% sát thương, tước đoạt 25% Chân Nguyên và làm chậm 35% tuyệt đối." },
  { code: "SPRING_SCISSORS_SUMMER_FAN", label: "Xuân Tiễn Hạ Phiến", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Thời Đạo: Kéo mùa xuân cắt đứt phòng tuyến, quạt mùa hè thiêu rụi phòng thủ! Tăng 55% sát thương và phá 25% giáp." },
  { code: "DRAGON_BREATH_SWORD", label: "Long Tức Kiếm", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Băng Kiếm: Long ngâm vang dội, hàn băng kiếm khí đóng băng hoàn toàn mục tiêu và tăng 45% sát thương." },
  { code: "MYRIAD_YEAR_WARSHIP", label: "Vạn Niên Đấu Phi Thuyền", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Tiên Cổ Ốc Thời Đạo: Bẻ cong dòng chảy thời gian, tăng 50% Tốc độ cho toàn đội và mỗi hiệp bắn phi kiếm Nhật Chiếu chi viện." },
  { code: "DRAGON_PALACE_DOMINION", label: "Long Cung Ngự Hồn", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Tiên Cổ Ốc Mộng/Nô Đạo: Khói mộng điệp lượn lờ khiến kẻ tấn công có 35% xác suất rơi vào Mê Loạn trước khi đánh trúng." },
  { code: "DIVINE_BEAN_ABUNDANCE", label: "Thần Đậu Cung Sinh Cơ", trigger: "ON_TURN_START", status: "active", group: "combat", behavior: "Tiên Cổ Ốc Mộc Đạo: Đầu mỗi hiệp nảy mầm sinh cơ dạt dào, thanh tẩy dị trạng bất lợi và hồi 10% MaxHP cho toàn đội." },
  { code: "EIGHT_EXTREMITIES_LUCK", label: "Bát Cực Quán Trấn Vận", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Tiên Cổ Ốc Vận Đạo: Trấn Vận Cổ Trận trấn áp khí vận phe địch, giảm 20% Tỉ lệ Bạo Kích và 25% Né Tránh của toàn bộ quân địch." },
  { code: "THUNDER_GOD_CITY", label: "Lôi Thần Thành Phạt Tội", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Tiên Cổ Ốc Lôi Đạo: Pháo đài lôi điện tự động phóng cuồng lôi oanh tạc kẻ địch có lượng sinh lực thấp nhất trên sàn đấu." },
  { code: "FIVE_REGIONAL_BOUNDARY", label: "Ngũ Giới Phân Tranh Trận", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Chiến Trường Sát Chiêu: Giới Vực Bích dâng trào giam cầm toàn bộ quân địch, mỗi lần ra chiêu bị phản phệ tự tổn 5% MaxHP." },
  { code: "NINETY_NINE_ARRAY", label: "Cửu Cửu Liên Hoàn Trận", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Đại Trận Hộ Thể: 99 tầng trận giáp ngưng tụ 40% MaxHP Hộ Thuẫn và hấp thụ 50% toàn bộ sát thương nhận vào." },
  { code: "SOUL_CONCEALMENT_ARRAY", label: "Yểm Hồn Ma Trận", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Đại Trận Huyễn Hoặc: Sương mù linh hồn che phủ toàn đội trong trạng thái Ẩn Thân và tăng 35% Né Tránh." },
  { code: "SIX_EXTREMITIES_ARRAY", label: "Lục Cực Tuyệt Sát Trận", trigger: "ON_TURN_START", status: "active", group: "combat", behavior: "Sát Trận Tuyệt Diệt: Trận nhãn xoay chuyển giáng sát thương hủy diệt lên toàn bộ quân địch, sát thương tăng dần theo số hiệp (Turn × 2% MaxHP)." },

  // ── Sát Chiêu Tiên Đạo (Đợt 6 - Reverend Insanity Canonical Moves) ────
  { code: "SONG_OF_SEPARATION", label: "Ly Ca", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Âm Đạo: Khúc ca ly tán vạn vật, phá hủy 2 buff, bóc 30% giáp ảo, tăng 65% sát thương và gây Suy Yếu 25%." },
  { code: "SONG_OF_DESTINY", label: "Vận Mệnh Ca", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Âm/Vận Đạo: Khúc ca vận mệnh chấn động, buff toàn đội +40% sát thương và hồi 20% chân nguyên, giảm 30% sát thương quân địch." },
  { code: "FAREWELL_FRIEND_WIND", label: "Tống Hữu Phong", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Phong Đạo: Gió tiễn bạn xuyên thấu phòng ngự tăng 50% sát thương, nguyền rủa Tống Hữu Chi Phong trừ 5% MaxHP mỗi hiệp trong 4 hiệp." },
  { code: "INFINITE_WIND", label: "Vô Hạn Phong", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Phong Đạo: Gió vô cùng vô tận bùng nổ, tăng 75% sát thương, +60% Thần Tốc và +35% Né Tránh trong 3 hiệp." },
  { code: "BLOOD_STAINED_ROBE", label: "Huyết Nhiễm Chinh Bào", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Huyết Đạo: Chiến bào đỏ thẫm hấp thụ 40% sát thương nhận vào chuyển hóa thành Huyết Thuẫn hộ thể." },
  { code: "BLOOD_DEMON_DISINTEGRATION", label: "Huyết Ma Giải Thể", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Huyết Đạo: Tự tổn 20% HP hiện tại giải thể ma huyết, tăng bộc phát 120% sát thương và nhận 50% Hút Máu." },
  { code: "YAMA_EMPEROR", label: "Diêm Đế Quỷ Quan Y", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Hồn Đạo: Hóa thân Diêm La Đại Đế, tăng 50% Phòng Ngự và kích hoạt Diêm La Tử cắn nuốt hồn phách địch mỗi hiệp." },
  { code: "IMMEASURABLE_QI_SEA", label: "Vô Lượng Khí Hải", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Khí Đạo: Khí triều mênh mông như biển, tăng 70% sát thương toàn diện, giảm 30% Tấn Công và làm chậm 30% toàn bộ quân địch." },
  { code: "THREE_QI_RETURN", label: "Tam Khí Quy Lai", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Khí Đạo: Thiên Địa Nhân tam khí quy nhất, hồi 30% HP, đầy Chân Nguyên, tăng 85% sát thương và bỏ qua 50% giáp." },
  { code: "THREE_MEN_MAKE_TIGER", label: "Tam Nhân Thành Hổ", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Tín Đạo: Ba người nói thành hổ, hung hổ ảo ảnh vồ nát tâm trí mục tiêu, tăng 50% sát thương, áp đặt Sợ Hãi và Suy Yếu." },
  { code: "WORDS_ARE_FRIGHTENING", label: "Nhân Ngôn Khả Úy", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Tín Đạo: Lời đồn thế gian xé lòng, sát thương tăng theo số lượng debuff trên người mục tiêu (+25%/debuff) và Câm Lặng 2 hiệp." },
  { code: "FATE_VANQUISHED", label: "Mệnh Bại", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Thiên/Thời Đạo: Hồng Liên xé rách số mệnh, phá vỡ hoàn toàn Bất Tử, Bất Hoại và Giáp ảo, giáng đòn x2.2 sát thương." },
  { code: "HEAVEN_NET_RESTRICTION", label: "Lưới Trời Tuy Thưa", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Thiên Đạo: Khôi Khôi Thiên Võng phong tỏa toàn trường, giảm 50% né tránh phe địch, khi thi triển khóa cứng né tránh và làm chậm 40%." },
  { code: "FUTURE_SELF", label: "Vị Lai Thân", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Trụ Đạo: Mượn sức mạnh từ tương lai đỉnh phong, tăng vọt 80% Tấn Công và 50% Tốc độ trong 3 hiệp." },
  { code: "LANDSCAPE_AS_BEFORE", label: "Giang Sơn Như Cố", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Trụ Đạo: Cảnh cũ người xưa tái hiện, hồi phục 40% MaxHP Giáp và tẩy sạch toàn bộ hiệu ứng suy yếu." },
  { code: "EATING_LOSS_BLESSING", label: "Cật Khuy Thị Phúc", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Thực Đạo: Chịu thiệt là phúc, mỗi lần bị đánh tích lũy phúc lực (+5 Công, +5 Thủ, max 10 tầng), khi kích hoạt nổ sát thương và hồi HP." },
  { code: "MYRIAD_DISMANTLE", label: "Vạn Giải", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Luật Đạo: Vạn vật quy nhất tiêu giải pháp tắc, hủy toàn bộ Hộ Thuẫn, xóa sạch mọi Buff và đánh xuyên 100% phòng ngự." },
  { code: "YOU_DU_STRENGTH_TIGER", label: "Đồ Đằng U Đô Lực Bưu", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Đồ Đằng: Dực Hổ gầm vang, tăng 85% sát thương, xé toạc sinh mệnh gây Xuất Huyết nặng và Choáng 1 hiệp." },
  { code: "NIRVANA_FIRE", label: "Niết Bàn Hỏa", trigger: "ON_DEATH", status: "active", group: "combat", behavior: "Viêm Đạo: Khi nhận đòn chí mạng dục hỏa trùng sinh, hồi sinh 40% HP, tẩy uế toàn bộ và nổ hỏa diễm dư chấn thiêu đốt phe địch." },

  // ── Đại Điển Sát Chiêu Tiên Đạo (Đợt 7 - Reverend Insanity Canonical Moves 2) ──
  { code: "SWORD_WAVE_TRIPLE", label: "Kiếm Lãng Tam Điệp", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Kiếm Đạo: Ba đợt sóng kiếm trào dâng, tăng 80% sát thương (3 hit liên kích) và gây Xuất Huyết." },
  { code: "SWORD_MARK_DEATH", label: "Kiếm Ngân Tác Mệnh", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Kiếm Đạo: Khắc sâu kiếm ngân vào hồn phách địch, ăn mòn sinh mệnh mỗi hiệp và trảm sát khi dưới 25% HP." },
  { code: "SOUL_EXPLOSION_DANG_HUN", label: "Hồn Bạo Đãng Hồn Sơn", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Hồn Đạo: Chấn động Đãng Hồn Sơn vang vọng, bộc phá sát thương Hồn và Choáng toàn bộ phe địch 1 hiệp." },
  { code: "SOUL_BEAST_SUMMON", label: "Hồn Thú Triệu Lai", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Hồn Đạo: Triệu hoán Thái Cổ Hồn Thú nhập thể hộ vệ, nhận 35% MaxHP Hộ Thuẫn và tăng 30% sát thương." },
  { code: "STEAL_LIFE", label: "Thâu Sinh", trigger: "ON_ACTION", status: "active", group: "steal", behavior: "Thâu Đạo: Cướp đoạt thọ mệnh sinh cơ, hút thẳng 15% MaxHP của đối phương chuyển hóa thành máu hồi phục bản thân." },
  { code: "CHAOS_CAVE_DRILL", label: "Hỗn Động Toản", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Vũ Đạo: Mũi khoan không gian xoắn nát hư không, hủy sạch Hộ Thuẫn, xuyên 100% giáp và tăng 80% sát thương." },
  { code: "NO_EXTREME_SPACE", label: "Vô Gian Đạo", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Vũ Đạo: Hòa tan thân thể vào tầng lớp không gian vô tận, tăng 50% Né Tránh trong 3 hiệp." },
  { code: "LIGHT_TIME_FLYING_BLADE", label: "Quang Âm Phi Nhận", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Trụ Đạo: Lưỡi đao thời gian phóng ra từ Trường Hà, tăng 65% sát thương, tước 30% Chân Nguyên và làm chậm 50%." },
  { code: "ALL_PEOPLE_UNITED", label: "Vạn Chúng Nhất Tâm", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Nhân Đạo: Hiệu triệu vạn dân đồng lòng, toàn đội nhận Khiên 30% HP và tăng 35% Sát Thương." },
  { code: "HERO_AMONG_MEN", label: "Nhân Trung Hào Kiệt", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Nhân Đạo: Nghịch cảnh chuyển thương thành dũng, máu hao hụt càng nhiều sát thương bộc phát càng khủng khiếp." },
  { code: "QI_JUE_RETURN_LIFE", label: "Khí Tuyệt Phùng Sinh", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Khí Đạo: Mượn Hy Vọng Khí chuyển bại thành thắng, khi máu dưới 30% lập tức hồi 40% HP và thanh tẩy toàn bộ debuff." },
  { code: "RAGING_ANGRY_BIRD", label: "Phẫn Nộ Đích Tiểu Điểu", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Viêm Đạo: Hỏa điểu phẫn nộ phát nổ kinh hoàng, tăng 50% sát thương và gây Thiêu Đốt nặng 3 hiệp." },
  { code: "EXTREME_ICE_CRYSTAL_COFFIN", label: "Cực Băng Trấn Tiên Quan", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Băng Tuyết Đạo: Quan tài băng tinh trấn áp, Đóng Băng và Câm Lặng hoàn toàn mục tiêu trong 2 hiệp." },
  { code: "THIRTY_THREE_HEAVEN_LIGHT", label: "Tam Thập Tam Thiên Quang", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Quang Đạo: 33 tầng thiên quang rọi sáng, thanh tẩy toàn bộ phe ta và làm Mù Lòa toàn bộ quân địch." },
  { code: "GREAT_NIGHT", label: "Đại Dạ Di Thiên", trigger: "ON_BATTLE_START", status: "active", group: "combat", behavior: "Ám Đạo: Màn đêm vĩnh hằng buông xuống, toàn đội Ẩn Thân, +50% Bạo Kích và giảm 40% Né Tránh phe địch." },
  { code: "DRY_WOOD_SPRING", label: "Khô Mộc Phùng Xuân", trigger: "ON_TURN_START", status: "active", group: "combat", behavior: "Mộc Đạo: Cây khô tái sinh nảy lộc, hồi phục 15% MaxHP mỗi hiệp liên tục trong 3 hiệp." },
  { code: "IMMORTAL_BINDING_VINE", label: "Khổn Tiên Đằng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Mộc Đạo: Dây leo tiên đạo trói chặt mục tiêu trong 2 hiệp và hút 20% Chân Nguyên." },
  { code: "BLOOD_GOD_CHILD", label: "Huyết Thần Tử", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Huyết Đạo: Huyết phân thân xuất thế trợ chiến, tăng 50% sát thương và nhận 40% Hút Máu." },
  { code: "TOTEM_KUN_PENG", label: "Đồ Đằng Côn Bằng", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Sát Chiêu Đồ Đằng: Côn Bằng tung cánh bạt sơn cửu thiên, tăng 75% sát thương và nhận 50% Thần Tốc." },
  { code: "TOTEM_GOLDEN_BELL", label: "Đồ Đằng Kim Chung", trigger: "BEFORE_HIT", status: "active", group: "combat", behavior: "Sát Chiêu Đồ Đằng: Chuông vàng viễn cổ buông rủ ngưng tụ hào quang, kích hoạt 35% Phản thương và kháng bạo kích." },
  { code: "REFINE_SELF", label: "Luyện Kỷ", trigger: "ON_CAST", status: "active", group: "combat", behavior: "Luyện Đạo: Đỉnh cao Luyện Đạo lấy thân làm phôi, tăng vĩnh viễn 20% Công và 20% Thủ suốt trận chiến." },
  { code: "MYRIAD_STAR_FIREFLY", label: "Vạn Tinh Phi Huỳnh", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Trí Đạo: Đom đóm ánh sao bùng cháy thiêu rụi ý niệm, tăng 60% sát thương và thiêu rụi 40% Chân Nguyên đối phương." },
  { code: "FIVE_RESTRICTION_DEEP_LIGHT", label: "Ngũ Cấm Huyền Quang Khí", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Cấm Đạo: Luồng sáng năm màu phong tỏa ngũ hành, phong ấn Cổ trùng (Phong Cổ) của địch trong 3 hiệp." },
  { code: "CHARM_GHOST_COME", label: "Tử Quỷ Khoái Lai", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Mị Đạo: Tiếng gọi ma mị quyến rũ mê hồn, áp đặt Mê Hoặc khiến đối phương thần hồn điên đảo trong 2 hiệp." },
  { code: "DEVOUR_HEAVEN_EARTH", label: "Thôn Thiên Địa", trigger: "ON_ACTION", status: "active", group: "combat", behavior: "Thực Đạo: Cắn nuốt thiên địa càn khôn, nuốt 50% Khiên Ảo của địch chuyển hóa thành HP hồi phục bản thân." },
  { code: "PERSIST_STATUS_ON_HIT", label: "Lưu Trạng Thái Khi Trúng Đòn", trigger: "ON_HIT", status: "partial", group: "combat", behavior: "Đòn trúng trong PvP thật gán vĩnh viễn một trạng thái lên địch, mỗi trận một lần." },
  { code: "IMMUNE_DREAM_STATUS", label: "Miễn Nhiễm Mộng Đạo", trigger: "ALWAYS_ON", status: "active", group: "combat", behavior: "Miễn nhiễm tuyệt đối mọi trạng thái thuộc Mộng đạo." },
  { code: "CLEANSE_REGIONAL_WALL", label: "Phá Áp Chế Giới Vực", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "Xóa Áp Chế Giới Vực của bản thân ngay đầu trận trước khi tính chỉ số." },
  { code: "FAKE_REGIONAL_WALL", label: "Ngụy Áp Chế Giới Vực", trigger: "ON_BATTLE_START", status: "legacy", group: "combat", behavior: "PvP: giảm nửa Áp Chế Giới Vực tác động lên địch trong trận; sau trận gán Áp Chế lên đối thủ 15 + 5×rank phút." },

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
  { code: "SPRING_EGG_HATCH", label: "Trứng Nguyên Tuyền", trigger: "ON_CONSUME", status: "active", group: "growth", behavior: "Cấy một Phàm Tuyền ngay dưới chân; hoạt động cả qua vật phẩm Trứng Nguyên Tuyền và Cổ tiêu hao GU_SPRING_EGG. Không bị giới hạn cap 31 Phàm Tuyền hay cap đại vực; không tiêu hao khi ô đã có địa điểm, vùng an toàn hoặc trong bí cảnh." },

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
  { code: "PORTAL_CREATE", label: "Tạo Cổng Dịch Chuyển", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Dùng Cổ Tiêu Hao trong Túi Cổ để mở modal nhập tọa độ; máy chủ kiểm tra tầm theo range hoặc rank Cổ rồi mới dịch chuyển và tiêu hao Cổ." },
  { code: "RANDOM_PORTAL", label: "Cổng Ngẫu Nhiên", trigger: "CAPABILITY", status: "active", group: "world", behavior: "Dùng Cổ Tiêu Hao trong Túi Cổ để xác nhận dịch chuyển đến tọa độ ngẫu nhiên hợp lệ trong tầm range hoặc tầm theo rank Cổ; Cổ chỉ bị tiêu hao khi dịch chuyển thành công." },
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
  { id: "STEALTH", label: "Ẩn Thân", category: "Buff / bản đồ", behavior: "Trong combat: không bị chọn bởi đòn đơn mục tiêu. Dùng ON_USE ngoài bản đồ: tăng DISGUISE, giảm tỉ lệ phục kích.", turns: "2 / phút" },
  { id: "HASTE", label: "Tăng Tốc", category: "Buff / bản đồ", behavior: "Trong combat: tăng tốc độ theo lượt. Dùng ON_USE ngoài bản đồ: tăng tốc độ người chơi theo phần trăm trong thời hạn status.", turns: "3 lượt / phút" },
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
  { id: "THOUGHT_CHAOS", label: "Tạp Niệm", category: "Không Khiếu", behavior: "Trí Đạo: Gấp đôi tiêu hao Chân Nguyên khi dùng Cổ và có 30% tỉ lệ phát chiêu thất bại.", turns: "3" },
  { id: "DAO_CONFLICT", label: "Xung Đột Đạo Ngân", category: "Xung đột đạo", behavior: "Cờ debuff đã gán; engine chưa xử lý riêng.", turns: "3" },
  { id: "BACKLASH", label: "Phản Phệ", category: "Bất tử", behavior: "Gây sát thương ngay khi dính.", turns: "1" },
  { id: "FORGET_EGO", label: "Lãng Quên Bản Ngã", category: "Khống chế tinh thần", behavior: "Phong ấn Cổ, không dùng được kỹ năng Cổ.", turns: "1" },
  { id: "FORGET_REALITY", label: "Lãng Quên Thực Tại", category: "Khống chế tinh thần", behavior: "Giảm % sát thương gây ra.", turns: "2" },
  { id: "DREAM_PRISON", label: "Mộng Cảnh Giam Cầm", category: "Khống chế tinh thần", behavior: "Mộng Đạo: Kéo đối thủ vào cõi mộng sâu thẳm, mất lượt hoàn toàn và không bị đánh thức bởi sát thương thường.", turns: "2" },
  { id: "DREAM_ILLUSION", label: "Mộng Ảnh Ẩn Thân", category: "Buff", behavior: "Mộng Đạo: Ẩn thân trong ảo mộng, kẻ địch không thể nhắm mục tiêu bằng đòn đánh đơn lẻ.", turns: "2" },
  { id: "LUCK_SIPHON", label: "Đoạt Vận", category: "Khống chế tinh thần", behavior: "Vận Đạo: Tước đoạt 15% Né Tránh & Bạo Kích của mục tiêu chuyển hóa thành vận khí cho bản thân.", turns: "3" },
  { id: "LUCK_CURSE_REVERSE", label: "Họa Hề Phúc Sở Ỷ", category: "Buff", behavior: "Vận Đạo: Biến họa thành phúc, tăng vọt tỉ lệ quay ra Đại Cát trong hiệp kế tiếp.", turns: "2" },
  { id: "BERSERK", label: "Cuồng Bạo", category: "Buff", behavior: "Lực Đạo: Tăng 35% sát thương và 30% kiên cường (rút ngắn/tự giải khống chế).", turns: "3" },
  { id: "CALCULATED", label: "Động Tất", category: "Buff", behavior: "Trí Đạo: Nhìn thấu đối thủ, tăng 25% sát thương kỹ năng.", turns: "2" },
  { id: "BLOOD_SACRIFICE", label: "Huyết Tế", category: "Buff", behavior: "Huyết Đạo: Kích phát huyết sát cuồng nộ, tăng 50% sát thương gây ra.", turns: "2" },
  { id: "BEAR_FORM", label: "Hóa Thân Cự Hùng", category: "Buff", behavior: "Biến Hóa Đạo: Tăng 30% phòng thủ và ngưng tụ 25% MaxHP Hùng Giáp.", turns: "3" },
  { id: "EAGLE_FORM", label: "Hóa Thân Kim Điêu", category: "Buff", behavior: "Biến Hóa Đạo: Tăng 40% tốc độ và 25% né tránh.", turns: "3" },
  { id: "ADAPTIVE_RESIST", label: "Thích Ứng Đạo Pháp", category: "Buff", behavior: "Biến Hóa Đạo: Thích ứng cơ thể, giảm 25% sát thương nhận vào.", turns: "3" },
  { id: "BONE_ARMOR", label: "Bạch Cốt Giáp", category: "Buff", behavior: "Cốt Đạo: Tăng 20% phòng thủ, 35% phản thương và nhận 20% MaxHP Cốt Giáp.", turns: "3" },
  { id: "METAL_AEGIS", label: "Cương Thiết Hộ Thể", category: "Buff", behavior: "Kim Đạo: Tăng 35% phòng thủ và giảm 15% sát thương nhận vào.", turns: "3" },
  { id: "CLOUD_MIST", label: "Vân Vụ Ảo Mộng", category: "Buff", behavior: "Vân Đạo: Tăng 35% né tránh và giảm 20% sát thương nhận vào.", turns: "3" },
  { id: "REVERSE_FLOW_SEAL", label: "Nghịch Lưu Hộ Thân Quỹ", category: "Bất tử", behavior: "Sát Chiêu Thủy Đạo: Mượn uy lực Nghịch Lưu Hà phản phệ hoàn toàn 100% sát thương nhận vào ngược lại kẻ tấn công.", turns: "3" },
  { id: "FAMILIAR_FACE", label: "Gặp Mặt Từng Quen Biết", category: "Buff", behavior: "Sát Chiêu Biến Hóa Đạo: Ngụy trang hoàn mỹ hòa mình vào hư không, tăng 50% Né Tránh và đòn đánh đầu chắc chắn Bạo Kích.", turns: "3" },
  { id: "TIME_SPEED_SURGE", label: "Phi Thuyền Lưu Tốc", category: "Buff", behavior: "Tiên Cổ Ốc Thời Đạo: Bẻ cong thời gian quanh chiến hạm, tăng 50% Tốc độ hành động.", turns: "5" },
  { id: "FIVE_REGIONAL_SUPPRESSION", label: "Ngũ Giới Áp Chế", category: "Xung đột đạo", behavior: "Chiến Trường Sát Chiêu: Giới Vực Bích áp chế toàn diện, mỗi lần ra chiêu bị phản phệ tự tổn 5% MaxHP.", turns: "5" },
  { id: "NINETY_NINE_SHIELD", label: "Cửu Cửu Liên Hoàn Giáp", category: "Buff", behavior: "Đại Trận Hộ Thể: 99 tầng kết giới trận giáp hấp thụ 50% toàn bộ sát thương nhận vào.", turns: "5" },
  { id: "YAMA_EMPEROR", label: "Diêm Đế Thể", category: "Bất tử", behavior: "Hồn Đạo: Diêm Đế giáng thế, tăng 50% Phòng Ngự và cắn nuốt hồn địch mỗi hiệp.", turns: "4" },
  { id: "BLOOD_ROBE", label: "Huyết Nhiễm Chinh Bào", category: "Buff", behavior: "Huyết Đạo: Hấp thụ 40% sát thương nhận vào chuyển hóa thành Huyết Thuẫn.", turns: "3" },
  { id: "FUTURE_SELF", label: "Vị Lai Thân", category: "Buff", behavior: "Trụ Đạo: Mượn sức mạnh tương lai, tăng 80% Tấn Công và 50% Tốc độ.", turns: "3" },
  { id: "EATING_LOSS", label: "Cật Khuy Thị Phúc", category: "Buff", behavior: "Thực Đạo: Chịu thiệt rèn giũa, tăng tích tầng Công/Thủ mỗi khi bị đánh.", turns: "5" },
  { id: "FAREWELL_CURSE", label: "Tống Hữu Chi Phong", category: "Xung đột đạo", behavior: "Phong Đạo: Gió tiễn bạn vần vũ ăn mòn cốt nhục, trừ 5% MaxHP mỗi hiệp trực tiếp vào HP.", turns: "4" },
  { id: "SWORD_MARK", label: "Kiếm Ngân Tác Mệnh", category: "DoT vật lý", behavior: "Kiếm Đạo: Kiếm ngân róc thịt gọt xương, trừ máu mỗi hiệp và trảm sát tức khắc khi dưới 25% HP.", turns: "4" },
  { id: "REFINE_SELF", label: "Luyện Kỷ Thể", category: "Buff", behavior: "Luyện Đạo: Thân thể tôi luyện thành hình, tăng vĩnh viễn 20% Công và 20% Thủ suốt trận.", turns: "Vĩnh viễn" },
  { id: "ALL_PEOPLE_UNITED", label: "Vạn Chúng Nhất Tâm", category: "Buff", behavior: "Nhân Đạo: Ý chí triệu người như một, nhận Khiên 30% HP và tăng 35% sát thương.", turns: "4" },
  { id: "DRY_WOOD_SPRING", label: "Khô Mộc Phùng Xuân", category: "Buff", behavior: "Mộc Đạo: Gốc mục đâm chồi, hồi phục 15% MaxHP mỗi hiệp liên tục.", turns: "3" },
  { id: "ICE_COFFIN", label: "Cực Băng Trấn Tiên Quan", category: "Khống chế tâm thần", behavior: "Băng Tuyết Đạo: Đóng băng và câm lặng trong quan tài băng tinh.", turns: "2" },
  { id: "TOTEM_GOLDEN_BELL", label: "Đồ Đằng Kim Chung", category: "Buff", behavior: "Đồ Đằng: Chuông vàng hộ thể, phản phệ 35% sát thương nhận vào và miễn nhiễm bạo kích.", turns: "4" },
  { id: "GREAT_NIGHT", label: "Đại Dạ Di Thiên", category: "Buff", behavior: "Ám Đạo: Màn đêm vĩnh cửu bao phủ, ẩn thân và tăng 50% tỉ lệ bạo kích.", turns: "4" },
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
