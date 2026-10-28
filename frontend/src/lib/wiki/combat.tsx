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
        title: "Chiến đấu realtime",
        targetId: "combat",
        description: "Bố trí đội hình và lượt đánh",
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
    title: "Chiến đấu realtime",
    category: "combat",
    keywords: ["chiến đấu", "combat", "realtime", "hotbar", "cast", "khiên", "shield", "aoe"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Chiến đấu chạy trên map realtime do server mô phỏng theo tick. Vị trí, mục tiêu, thời gian tụ lực, hồi chiêu, trạng thái và sát thương đều được máy chủ xác nhận.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">1. Điều khiển và thi triển chiêu</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Di chuyển bằng WASD/chuột; dùng hotbar để chọn Cổ có profile realtime. Chiêu có thể cần chọn mục tiêu hoặc hướng, tụ lực, tiêu Chân Nguyên và chờ hồi chiêu tùy profile.
          </p>
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-2 text-[var(--text-secondary)]">
            <li>🎯 <strong>Mục tiêu:</strong> profile khai báo mục tiêu, tầm đánh và giới hạn số mục tiêu.</li>
            <li>💥 <strong>Vùng tác động:</strong> hình dạng, bán kính và hướng do profile realtime quyết định.</li>
            <li>⚡ <strong>Server authoritative:</strong> client gửi lệnh; máy chủ kiểm tra range, mana, cooldown và hit.</li>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">2. Trạng thái và hiệu ứng</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Status được áp dụng, cộng dồn và hết hạn theo tick realtime. Effect chỉ hoạt động khi có executor và profile realtime rõ ràng; dữ liệu turn-based cũ không tự đổi lượt thành giây hoặc tự phát sinh hành vi thay thế.
          </p>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">3. Khiên ảo & Sinh Mạng</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Sinh mạng và khiên được mô phỏng trong world runtime. Khiên hấp thụ sát thương theo quy tắc profile; khi HP về 0, server xử lý hạ gục, respawn/settlement và lịch sử theo chế độ chiến đấu.
          </p>
          <div className="p-4 rounded-2xl bg-amber-950/20 border border-amber-500/10 text-xs leading-relaxed text-amber-300">
            💀 <strong>Tử Đạo (Chết rơi đồ):</strong> Khi bại trận dã ngoại, bỏ chạy ngất xỉu, hay thua PvP, đạo hữu sẽ đánh rơi một phần tài sản đang giữ: Nguyên Thạch, nguyên liệu, Cổ trùng (trừ Cổ đang trang bị & Bản Mệnh Cổ) và Bí phương.
          </div>
        </div>
      </div>
    )
  },
];
