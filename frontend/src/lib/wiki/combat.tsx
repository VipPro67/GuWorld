import React from "react";
import { Swords } from "lucide-react";
import { WikiPage } from "./types";

export const combatPages: WikiPage[] = [
  {
    id: "combat-mechanics",
    title: "Cơ Chế Chiến Thuật: Sát Thương, Khiên & Phòng Thủ",
    category: "combat",
    keywords: [
      "sát thương",
      "khiên",
      "phòng thủ",
      "công thức",
      "hit",
      "sát thương ròng",
      "dao ngân",
      "đạo nguyên",
      "chí mạng",
      "crit",
      "hút máu",
      "life steal",
      "đóng băng",
      "choáng",
      "buff",
      "debuff",
      "xung đột",
      "kháng tự nhiên",
    ],

    content: (
      <div className="space-y-6">
        {/* Intro */}
        <div className="p-4 rounded-2xl bg-indigo-950/30 border border-indigo-500/20 flex gap-3 items-start">
          <Swords className="w-5 h-5 text-indigo-400 shrink-0 mt-0.5" />

          <div>
            <h4 className="text-sm font-serif font-bold text-indigo-200">
              Cách tính sát thương & phòng thủ
            </h4>

            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              Hiểu rõ công thức sát thương, khiên ảo, Đạo Ngân, Xung Đột Đạo Ngân và chí mạng giúp
              bạn tối ưu đội hình Cổ trùng và tối đa hóa hiệu quả chiến đấu.
            </p>
          </div>
        </div>

        {/* 1. Damage */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">
            1. Công Thức Sát Thương Cơ Bản
          </h3>

          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Mỗi đòn tấn công (BASIC_ATTACK hoặc GU_SKILL) gây sát thương theo
            công thức:
          </p>

          <div className="bg-zinc-950/80 border border-[var(--border)] rounded-xl p-4 font-mono text-[11px] text-[var(--text-primary)] space-y-2 overflow-x-auto">
            <p>
              <span className="text-amber-400">hitDmg</span> ={" "}
              <span className="text-emerald-400">a.Damage</span> &times;{" "}
              <span className="text-emerald-400">a.DamageMult</span> &times;{" "}
              <span className="text-emerald-400">dmgMultiplier</span> &times;{" "}
              <span className="text-emerald-400">daoMult</span> &times;{" "}
              <span className="text-rose-400">daoConflictMult</span> &times; (1 + pathBonus)
            </p>

            <p className="text-[var(--text-muted)]">
              • <span className="text-amber-400">a.Damage</span>: Raw damage (DamageFlat from Gu + baseAttack)
            </p>

            <p className="text-[var(--text-muted)]">
              • <span className="text-amber-400">a.DamageMult</span>: Hệ số sát thương cộng dồn (damage_mult từ Cổ, gộp với cảnh giới lưu phái + buff chờ theo `1 + Σ(m−1)`)
            </p>

            <p className="text-[var(--text-muted)]">
              • <span className="text-amber-400">dmgMultiplier</span>: Temporary buff/debuff (Sword Dragon +50%, Borrow Past Power +40%, atk_up...)
            </p>

            <p className="text-[var(--text-muted)]">
              • <span className="text-emerald-400">daoMult</span>: Hệ số khuếch đại Đạo Ngân đồng hệ kèm hiệu suất cảnh giới (Attainment Efficiency: +10% &rarr; +45%)
            </p>

            <p className="text-[var(--text-muted)]">
              • <span className="text-rose-400">daoConflictMult</span>: Hệ số xung đột của lưu phái hành động (0.08 &rarr; 1.00). Chân Ý được xét riêng theo đúng lưu phái đang dùng; Vô Thượng điều hòa 100% xung đột cục bộ.
            </p>
          </div>

          <div className="p-4 rounded-xl bg-emerald-950/20 border border-emerald-500/20 text-xs space-y-2">
            <h4 className="font-bold text-emerald-300">📊 Ví dụ tính Sát Thương &amp; Xung Đột</h4>
            <div className="text-[var(--text-secondary)] space-y-1 font-mono text-[11px]">
              <div>• Gu: damage_flat = 500, damage_mult = 1.5, Buff: Kiếm Long = 1.5</div>
              <div>• Đạo Ngân: 3.000 điểm Kiếm Đạo &rarr; daoAmp = 31.0, Tông Sư (Eff = 1.12) &rarr; <span className="text-amber-400">daoMult = 34.6</span></div>
              <div>• Xung đột còn phụ thuộc ngưỡng dung nạp, độ tương thích, tổng các phái ngoại hệ và Chân Ý của lưu phái đang dùng.</div>
              <div className="bg-black/40 rounded-lg p-2 text-emerald-400 font-bold mt-1">
                Đạo Ngân Hỏa không khuếch đại đòn Kiếm; nó chỉ có thể gây xung đột nếu vượt ngưỡng.
              </div>
            </div>
          </div>
        </div>

        {/* 2. Net Damage */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">
            2. Tính Sát Thương Thực Tế (Net Damage & Kháng Tự Nhiên)
          </h3>

          <div className="bg-zinc-950/80 border border-[var(--border)] rounded-xl p-4 font-mono text-[11px] text-[var(--text-primary)] space-y-2 overflow-x-auto">
            <p>
              <span className="text-amber-400">targetDef</span> = target.BaseDef &times; target.DefenseMultiplier &minus; ignore_defense
            </p>

            <p className="mt-2">
              <span className="text-amber-400">netDmg</span> = (hitDmg &minus; targetDef) &times; (1 &minus; pathResistance) &times; (1 &minus; <span className="text-blue-400">naturalResistance</span>)
            </p>

            <p className="text-[var(--text-muted)]">
              • <span className="text-blue-400">naturalResistance</span>: Màng kháng Đạo Ngân tự nhiên từ Cổ Tiên phòng ngự (tối đa 30% khi Đạo Ngân thủ xung đột với đòn đánh)
            </p>
            <p>
              • Floor: netDmg tối thiểu = <strong>1</strong> (không bao giờ dưới 1)
            </p>
            <p className="mt-2">
              • <strong>BYPASS_SHIELD</strong>: Giảm targetDef đi 30, đánh thẳng vào máu bỏ qua khiên
            </p>
          </div>

          <div className="p-4 rounded-xl bg-blue-950/20 border border-blue-500/20 text-xs space-y-2">
            <h4 className="font-bold text-blue-300">📊 Ví dụ tính Thủ &amp; Kháng Tự Nhiên</h4>
            <div className="text-[var(--text-secondary)] space-y-1 font-mono text-[11px]">
              <div>• Kẻ địch: targetDef = 3.000</div>
              <div>• Màng Kháng Tự Nhiên (100.000 Đạo Ngân Thổ chống Kiếm Đạo): <span className="text-blue-400">naturalResistance = 6.9%</span></div>
              <div className="bg-black/40 rounded-lg p-2 text-red-400 font-bold mt-1">
                netDmg = (38.341 &minus; 3.000) &times; (1.0 &minus; 0.069) = 35.341 &times; 0.931 = 32.902 HP mất thực tế!
              </div>
            </div>
          </div>
        </div>

        {/* 3. Shield */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">
            3. Khiên Ảo (Virtual Shield)
          </h3>

          <div className="bg-zinc-950/80 border border-[var(--border)] rounded-xl p-4 font-mono text-[11px] text-[var(--text-primary)] space-y-2 overflow-x-auto">
            <p>
              <span className="text-amber-400">baseShield</span> = a.Defense
            </p>

            <p>
              • If <span className="text-amber-400">a.ShieldMult &gt; 1.0</span>: baseShield = a.Defense &times; a.ShieldMult
            </p>

            <p>
              • <span className="text-emerald-400">daoShield</span> = baseShield &times; (daoMult &minus; 1.0) &times; daoConflictMult
            </p>

            <p className="mt-2">
              actor.VirtualShield += baseShield + daoShield
            </p>

            <p className="text-[var(--text-muted)]">
              • Sát thương trừ vào khiên trước; khi khiên = 0, HP mới chịu sát thương.
            </p>

            <p className="text-[var(--text-muted)]">
              • <strong>Biến Thân Kiếm Long:</strong> Chuyển hóa HP thành khiên ảo +50% sát thương trong 3 lượt.
            </p>
          </div>
        </div>

        {/* 4. Speed & Turn */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">
            4. Tốc Độ & Thứ Tự Lượt Đi (Speed & Turn Order)
          </h3>

          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Các Cổ trùng và Cổ sư hành động theo thứ tự <strong>Tốc độ (Speed)</strong> giảm dần trong mỗi lượt. Tốc độ cao giúp ra tay trước, giành lợi thế khống chế hoặc kết liễu kẻ địch.
          </p>
        </div>

        {/* 5. Status Effects */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">
            5. Hiệu Ứng Trạng Thái Combat
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2">
              <h4 className="font-bold text-red-400">Gây hại (Debuff)</h4>

              <ul className="list-disc list-inside text-xs text-[var(--text-muted)] space-y-1">
                <li><strong>Stun (Choáng):</strong> Bỏ lượt</li>
                <li><strong>Freeze (Đóng băng):</strong> Bỏ lượt, 50% vỡ khi nhận damage</li>
                <li><strong>Sleep (Ngủ):</strong> Bỏ lượt</li>
                <li><strong>Bleed / Poison / Burn:</strong> Mất HP mỗi lượt (DOT)</li>
                <li><strong>Silence (Câm lặng):</strong> Không dùng sát chiêu</li>
                <li><strong>Charm (Mị hoặc):</strong> Tấn công đồng đội</li>
                <li><strong>Bind (Trói chân):</strong> Không thể di chuyển</li>
              </ul>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2">
              <h4 className="font-bold text-emerald-400">Có ích (Buff)</h4>

              <ul className="list-disc list-inside text-xs text-[var(--text-muted)] space-y-1">
                <li><strong>Shield (Khiên ảo):</strong> Hấp thụ sát thương</li>
                <li><strong>Heal (Trị liệu):</strong> Hồi phục sinh lực</li>
                <li><strong>Atk Up / Def Up:</strong> Tăng chỉ số tạm thời</li>
                <li><strong>Crit Up / Haste:</strong> Tăng chí mạng & tốc độ</li>
                <li><strong>Taunt (Khiêu khích):</strong> Kẻ địch buộc đánh bạn</li>
                <li><strong>Stealth (Ẩn thân):</strong> Tránh bị nhắm mục tiêu</li>
              </ul>
            </div>
          </div>
        </div>

        {/* 6. Life Steal */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">
            6. Hút Máu (Life Steal)
          </h3>

          <div className="bg-zinc-950/80 border border-[var(--border)] rounded-xl p-4 font-mono text-[11px] text-[var(--text-primary)] space-y-2 overflow-x-auto">
            <p>
              <strong>healFromLifesteal</strong> = totalDamage &times; a.LifeStealPct
            </p>
            <p>• Không bị giới hạn bởi max HP</p>
            <p>• Huyết đạo buff cho thêm +50% tỉ lệ hút máu cơ bản</p>
          </div>
        </div>
      </div>
    ),

    cards: [
      {
        title: "Xung Đột Đạo Ngân & Cân Bằng",
        targetId: "gameplay-dao-conflict",
        description: "10 Nhóm Đạo, cơ chế triệt tiêu xung đột và màng kháng tự nhiên",
      },
      {
        title: "Chiến đấu theo lượt",
        targetId: "combat",
        description: "Động cơ turn-based chuẩn xác, thứ tự lượt, Chân nguyên và sát chiêu",
      },
      {
        title: "Hướng dẫn Tân thủ",
        targetId: "guide",
        description: "Lộ trình hoạt động hằng ngày",
      },
    ],
  },
  {
    id: "combat",
    title: "Cơ chế Chiến đấu theo lượt",
    category: "combat",
    keywords: ["chiến đấu", "combat", "turnbase", "turn-based", "theo lượt", "tốc độ", "khiên", "shield", "sát chiêu"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Chiến đấu trong Cổ Giới vận hành theo cơ chế <strong>theo lượt xác định (Deterministic Turn-based Engine)</strong>. Mỗi hiệp đấu được máy chủ tính toán chuẩn xác dựa trên Tốc độ, Chân nguyên, hiệu ứng Cổ trùng và Sát chiêu.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">1. Thứ tự hành động và Hiệp đấu</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Mỗi hiệp (turn), các nhân vật (hoặc quái vật) hành động tuần tự theo chỉ số <strong>Tốc độ (Speed)</strong>. Đơn vị có tốc độ cao hơn sẽ ra đòn trước, cho phép áp dụng khống chế hoặc khiên bảo hộ kịp thời.
          </p>
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-2 text-[var(--text-secondary)]">
            <li>🎯 <strong>Thứ tự lượt:</strong> Xác định tự động vào đầu mỗi hiệp dựa trên tốc độ thực tế (bao gồm buff/debuff).</li>
            <li>⚡ <strong>Chân Nguyên:</strong> Mỗi Cổ trùng hoặc Sát chiêu khi kích hoạt tiêu hao Chân Nguyên tương ứng. Nếu Chân Nguyên không đủ, đòn đánh sẽ suy yếu hoặc chỉ dùng đòn đánh thường.</li>
            <li>🌪️ <strong>Sát chiêu phối hợp:</strong> Kích hoạt các hiệu ứng diện rộng (AoE), hàng ngang (Row) hoặc đơn thể theo công thức sát chiêu đã phối ngũ.</li>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">2. Trạng thái và Hiệu ứng kích hoạt</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Các hiệu ứng được kích hoạt chuẩn xác theo từng giai đoạn trong hiệp đấu:
          </p>
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-2 text-[var(--text-secondary)]">
            <li>🏁 <strong>ON_BATTLE_START:</strong> Kích hoạt khi vừa bước vào trận (khiên mở màn, bùa chúc phúc).</li>
            <li>🔄 <strong>ON_TURN_START:</strong> Kích hoạt ở đầu mỗi lượt của bản thân (hồi phục Chân Nguyên, độc tố phát tác).</li>
            <li>💥 <strong>ON_ACTION / ON_CAST:</strong> Kích hoạt khi thực hiện công kích hoặc thi triển sát chiêu.</li>
            <li>🛡️ <strong>POST_ACTION:</strong> Phản đòn, hút máu, hoặc tích tụ hộ thuẫn sau khi công kích.</li>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">3. Khiên Hộ Thể & Rơi Đồ Khi Bại Trận</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Khiên hấp thụ sát thương trực tiếp trước khi ảnh hưởng đến Sinh mệnh (HP). Khi HP về 0, nhân vật sẽ bị trọng thương hoặc tử vong.
          </p>
          <div className="p-4 rounded-2xl bg-amber-950/20 border border-amber-500/10 text-xs leading-relaxed text-amber-300">
            💀 <strong>Tử Đạo (Rơi đồ dã ngoại):</strong> Khi bại trận ở các khu vực dã ngoại nguy hiểm, đạo hữu có thể đánh rơi một phần tài sản mang theo: Nguyên Thạch, nguyên liệu, và Cổ trùng chưa khóa (ngoại trừ Cổ đang trang bị và Bản Mệnh Cổ).
          </div>
        </div>
      </div>
    )
  },
];
