import React from "react";
import { Sparkles, Shield, Swords, Zap, Compass, AlertTriangle, BookOpen, Layers } from "lucide-react";
import { WikiPage } from "./types";

export const cultivationDaoPages: WikiPage[] = [
  {
    id: "gameplay-dao-conflict",
    title: "Xung Đột Đạo Ngân & Cân Bằng Lưu Phái",
    category: "cultivation",
    keywords: [
      "đạo ngân", "dao mark", "xung đột đạo ngân", "dao conflict", "lưu phái", "10 nhóm đạo", "dao groups", 
      "cảnh giới", "chân ý", "attainment", "phòng ngự tự nhiên", "natural resistance", "chuyên tu", "song tu", "đa tu", "chí tôn"
    ],
    content: (
      <div className="space-y-6">
        {/* Banner */}
        <div className="p-4 rounded-2xl bg-gradient-to-r from-amber-950/40 via-purple-950/30 to-indigo-950/40 border border-amber-500/30 flex gap-3 items-start">
          <Sparkles className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-amber-200">Quy Luật Tự Nhiên & Cân Bằng Cổ Tiên</h4>
            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              Trong thế giới Cổ Chân Nhân, <strong>Đạo Ngân</strong> là sự cô đọng của Thiên Địa Quy Tắc trên thân thể Cổ Tiên. Đạo ngân cùng nhóm sẽ tương sinh hỗ trợ lẫn nhau, nhưng đạo ngân dị hệ sẽ sinh ra <strong>Xung Đột Đạo Ngân (Dao Mark Conflict)</strong> triệt tiêu uy lực sát chiêu. Muốn dung hòa nhiều lưu phái, Cổ Tiên bắt buộc phải nâng cao <strong>Cảnh Giới Chân Ý (Attainment)</strong>.
            </p>
          </div>
        </div>

        {/* Core Principles */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Zap className="w-4 h-4 text-amber-400" />
            <span>1. Ba Trụ Cột Cơ Chế Đạo Ngân</span>
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-amber-500/20 space-y-2 text-xs">
              <div className="flex items-center gap-2 text-amber-300 font-bold">
                <Swords className="w-4 h-4" />
                <span>Khuếch Đại Đồng Hệ</span>
              </div>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Đạo ngân cùng hệ với chiêu thức sẽ khuếch đại sát thương, khiên và trị liệu. Công thức khuếch đại 2 giai đoạn (Tuyến tính &le; 10k + Logarit bậc cao) được nâng tầm bởi <strong>Hiệu Suất Cảnh Giới (+10% &rarr; +45%)</strong>.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-rose-500/20 space-y-2 text-xs">
              <div className="flex items-center gap-2 text-rose-300 font-bold">
                <AlertTriangle className="w-4 h-4" />
                <span>Xung Đột Ngoại Hệ</span>
              </div>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Khi thi triển sát chiêu của lưu phái A, nếu cơ thể tồn tại lượng lớn Đạo ngân của lưu phái B không tương thích, uy lực chiêu thức sẽ bị triệt tiêu từ <strong>1.00 &rarr; 0.10</strong> (tối đa mất 90% sát thương nếu không có cảnh giới).
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-blue-500/20 space-y-2 text-xs">
              <div className="flex items-center gap-2 text-blue-300 font-bold">
                <Shield className="w-4 h-4" />
                <span>Phòng Ngự Tự Nhiên</span>
              </div>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Đạo ngân trên người Cổ Tiên đóng vai trò như một lớp màng bài xích tự nhiên. Khi bị tấn công bởi lưu phái xung đột, lượng Đạo ngân phòng thủ sẽ tạo ra <strong>Màng Kháng Tự Nhiên (lên đến 30%)</strong>.
              </p>
            </div>
          </div>
        </div>

        {/* 10 Dao Groups */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Layers className="w-4 h-4 text-indigo-400" />
            <span>2. Bảng Phân Loại 10 Nhóm Đạo (Dao Groups)</span>
          </h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Toàn bộ 49 lưu phái trong game được xếp vào <strong>10 Nhóm Đạo</strong>. Các lưu phái trong cùng một nhóm hoàn toàn tương dung <strong>100% (Xung đột = 0.0)</strong>.
          </p>

          <div className="overflow-hidden rounded-2xl border border-[var(--border)]">
            <table className="w-full text-left text-xs font-mono">
              <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                <tr>
                  <th className="p-3">Nhóm Đạo (Dao Group)</th>
                  <th className="p-3">Các Lưu Phái Thành Viên</th>
                  <th className="p-3">Đặc Tính Quy Tắc</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                <tr>
                  <td className="p-3 font-bold text-amber-400">1. ELEMENTAL (Nguyên Tố)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Hỏa, Thủy, Băng, Phong, Lôi, Thổ, Vân, Ám, Quang, Nguyệt, Nhật, Tinh</td>
                  <td className="p-3 text-emerald-400">Tương sinh biến chuyển nguyên tố tự nhiên</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-sky-400">2. SPACE_TIME (Thời Không)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Vũ (Không gian), Trụ (Thời gian), Hư (Hư vô)</td>
                  <td className="p-3 text-emerald-400">Quy tắc kết cấu cội nguồn không thời gian</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-purple-400">3. RULE_ORDER (Quy Tắc & Hạn Chế)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Luật, Giới (Giới hạn), Phong (Phong ấn), Trận</td>
                  <td className="p-3 text-emerald-400">Trật tự thiết lập luật lệ và phong tỏa</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-rose-400">4. KINETIC_WEAPON (Vũ Khí & Động Năng)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Kiếm, Đao, Binh, Thanh (Âm thanh), Chấn (Chấn động)</td>
                  <td className="p-3 text-emerald-400">Sát phạt vật lý, chém kích và cộng hưởng dao động</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-emerald-400">5. BODY_LIFE (Sinh Mệnh & Thể Phách)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Lực, Biến Hóa, Cốt, Cương (Thân thể/Cương thi)</td>
                  <td className="p-3 text-emerald-400">Nhục thân cường hóa, biến hóa hình thể</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-violet-400">6. SOUL_MIND (Linh Hồn & Tâm Thức)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Hồn, Trí, Nô (Nô dịch), Mộng, Mị (Mị hoặc), Ảo, Tình (Tình cảm)</td>
                  <td className="p-3 text-emerald-400">Thần thức, tâm trí khống chế và mộng cảnh ảo cảnh</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-cyan-400">7. INFORMATION_STEALTH (Ẩn Thao & Tình Báo)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Tín (Tin tức), Ẩn (Tàng hình), Huyễn (Biến ảo), Đạo (Trộm cắp)</td>
                  <td className="p-3 text-emerald-400">Dò thám thông tin, ẩn thân và trộm đoạt quy tắc</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-red-500">8. ORGANIC_CORRUPTION (Ăn Mòn & Huyết Độc)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Huyết, Độc, Mộc, Thực (Ăn nuốt)</td>
                  <td className="p-3 text-emerald-400">Sinh học hữu cơ, lây lan độc tính và huyết mạch</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-yellow-300">9. CRAFT_ALCHEMY (Chế Tác & Vật Chất)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Luyện, Đan, Kim (Kim loại)</td>
                  <td className="p-3 text-emerald-400">Luyện Cổ, đúc đan, chế tác tiên tài bảo vật</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-fuchsia-400">10. HEAVEN_HUMAN (Chí Cao Vận Mệnh)</td>
                  <td className="p-3 text-[var(--text-secondary)]">Thiên, Nhân, Vận, Họa (Tai họa/Vẽ), Hạn (Hạn hán)</td>
                  <td className="p-3 text-emerald-400">Chí cao pháp tắc, thiên ý, nhân tâm và vận mệnh</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* Pair Overrides */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Swords className="w-4 h-4 text-rose-400" />
            <span>3. Ngoại Lệ Khắc Chế & Tương Trợ Đặc Thù (Pair Overrides)</span>
          </h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Một số cặp lưu phái có quy luật tương sinh tương khắc vượt ra ngoài ranh giới nhóm Đạo:
          </p>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
            <div className="p-3 rounded-xl bg-rose-950/20 border border-rose-500/20 space-y-1">
              <span className="font-bold text-rose-300">Tương Khắc Cực Độ (Xung đột = 1.0 - 1.2):</span>
              <ul className="list-disc list-inside text-[var(--text-secondary)] space-y-0.5">
                <li><strong>Hỏa &harr; Thủy / Băng:</strong> Lửa và Nước/Băng triệt tiêu gay gắt.</li>
                <li><strong>Quang &harr; Ám:</strong> Ánh sáng thanh trừng bóng tối.</li>
                <li><strong>Độc &harr; Mộc:</strong> Độc ăn mòn sinh cơ thực vật.</li>
                <li><strong>Kiếm / Đao &harr; Trận:</strong> Trảm kích phá vỡ kết cấu trận nhãn.</li>
                <li><strong>Trí &harr; Tình:</strong> Lý trí và Cảm xúc đối nghịch.</li>
              </ul>
            </div>

            <div className="p-3 rounded-xl bg-emerald-950/20 border border-emerald-500/20 space-y-1">
              <span className="font-bold text-emerald-300">Cộng Hưởng Bổ Trợ (Xung đột = 0.2 - 0.3):</span>
              <ul className="list-disc list-inside text-[var(--text-secondary)] space-y-0.5">
                <li><strong>Lực &harr; Biến Hóa:</strong> Thân thể hóa hình thú nhận trọn vẹn lực đạo.</li>
                <li><strong>Hồn &harr; Nô:</strong> Hồn phách hùng mạnh nô dịch vạn thú dễ dàng.</li>
                <li><strong>Hư &harr; Ẩn:</strong> Hư không che giấu thân ảnh hoàn hảo.</li>
                <li><strong>Luyện &harr; Lực / Hỏa:</strong> Rèn luyện đúc Cổ cần hỏa hầu và thể lực.</li>
                <li><strong>Vận &harr; Thiên / Nhân:</strong> Vận mệnh gắn liền thiên ý và nhân tâm.</li>
              </ul>
            </div>
          </div>
        </div>

        {/* Formulas & Attainment Mitigation */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Compass className="w-4 h-4 text-cyan-400" />
            <span>4. Công Thức Tính Xung Đột & Điều Hòa Cảnh Giới</span>
          </h3>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3 text-xs">
            <div className="space-y-1">
              <h4 className="font-bold text-amber-300">A. Ngưỡng Dung Lượng An Toàn (Grace Threshold)</h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                Đạo ngân ngoại hệ &le; <span className="font-mono text-emerald-400">max(1.000, 2% tổng Đạo Ngân)</span> sẽ được Tiên Khiếu tự nhiên dung nạp mà <strong>hoàn toàn không gây xung đột</strong>. Bạn có thể thoải mái tích lũy một lượng nhỏ Đạo ngân từ các phần thưởng phụ.
              </p>
            </div>

            <div className="space-y-1 pt-2 border-t border-zinc-800">
              <h4 className="font-bold text-amber-300">B. Hệ Số Điều Hòa Cảnh Giới (Attainment Realm Control)</h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                Khi Cổ Tiên đạt Chân Ý cao ở lưu phái tấn công, khả năng điều phối quy tắc thế giới giúp <strong>triệt tiêu phần lớn xung đột</strong> và <strong>tăng uy lực khuếch đại</strong>:
              </p>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center pt-1 font-mono">
                <div className="p-2 rounded bg-zinc-900/80 border border-zinc-800">
                  <div className="text-zinc-300 font-bold">Đại Sư</div>
                  <div className="text-emerald-400 text-[11px]">-20% xung đột</div>
                  <div className="text-amber-400 text-[11px]">+10% uy lực</div>
                </div>
                <div className="p-2 rounded bg-zinc-900/80 border border-indigo-900/50">
                  <div className="text-indigo-300 font-bold">Tông Sư</div>
                  <div className="text-emerald-400 text-[11px]">-40% xung đột</div>
                  <div className="text-amber-400 text-[11px]">+20% uy lực</div>
                </div>
                <div className="p-2 rounded bg-zinc-900/80 border border-amber-900/50">
                  <div className="text-yellow-400 font-bold">Đại Tông Sư</div>
                  <div className="text-emerald-400 text-[11px]">-60% xung đột</div>
                  <div className="text-amber-400 text-[11px]">+30% uy lực</div>
                </div>
                <div className="p-2 rounded bg-zinc-900/80 border border-rose-900/50">
                  <div className="text-red-400 font-bold">Vô Thượng</div>
                  <div className="text-emerald-400 text-[11px]">-80% xung đột</div>
                  <div className="text-amber-400 text-[11px]">+45% uy lực</div>
                </div>
              </div>
            </div>

            <div className="space-y-1 pt-2 border-t border-zinc-800">
              <h4 className="font-bold text-amber-300">C. Màng Phòng Thủ Đạo Ngân Tự Nhiên (Natural Dao Resistance)</h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                Đạo ngân của Cổ Tiên phòng thủ tạo ra màng kháng giảm sát thương từ các đòn đánh dị hệ của đối phương:
              </p>
              <div className="p-2.5 rounded bg-zinc-950 font-mono text-[11px] text-blue-300">
                Resistance = (DefenseConflictDao / (DefenseConflictDao + 50.000)) &times; 30% (Trần tối đa 30%)
              </div>
            </div>
          </div>
        </div>

        {/* Calculation Examples */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Zap className="w-4 h-4 text-amber-400" />
            <span>5. Ví Dụ Tính Toán Chi Tiết (Step-by-Step)</span>
          </h3>

          <div className="space-y-3 text-xs">
            {/* Example 1 */}
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-amber-500/20 space-y-2">
              <h4 className="font-bold text-amber-300 font-serif">
                Ví dụ 1: Khuếch Đại Đạo Ngân &amp; Hiệu Suất Cảnh Giới (Attainment Efficiency)
              </h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                Giả sử Cổ Tiên Hỏa Đạo sở hữu <strong>2.000 Đạo Ngân Hỏa</strong>.
              </p>
              <div className="bg-zinc-950/80 border border-zinc-800 rounded-lg p-3 font-mono text-[11px] text-[var(--text-primary)] space-y-1">
                <div>• Khuếch đại gốc: <span className="text-amber-400">daoAmp = 1.0 + 2.000 / 100.0 = 21.0</span> (+2.000% sát thương gốc)</div>
                <div>• <strong>Phàm Thai (Eff = 1.00):</strong> totalDaoMult = 1.0 + (21.0 &minus; 1.0) &times; 1.00 = <span className="text-emerald-400 font-bold">21.0 lần</span></div>
                <div>• <strong>Đại Sư (Eff = 1.05):</strong> totalDaoMult = 1.0 + (21.0 &minus; 1.0) &times; 1.05 = <span className="text-emerald-400 font-bold">22.0 lần</span> (+100% bonus từ Chân Ý)</div>
                <div>• <strong>Tông Sư (Eff = 1.12):</strong> totalDaoMult = 1.0 + (21.0 &minus; 1.0) &times; 1.12 = <span className="text-emerald-400 font-bold">23.4 lần</span></div>
                <div>• <strong>Vô Thượng (Eff = 1.45):</strong> totalDaoMult = 1.0 + (21.0 &minus; 1.0) &times; 1.45 = <span className="text-emerald-400 font-bold">30.0 lần</span>!</div>
              </div>
            </div>

            {/* Example 2 */}
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-rose-500/20 space-y-2">
              <h4 className="font-bold text-rose-300 font-serif">
                Ví dụ 2: Xung Đột Đạo Ngân &amp; Khả Năng Hóa Giải Của Cảnh Giới (Attainment Control)
              </h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                Cổ Tiên Hỏa Đạo có <strong>10.000 Đạo Ngân Hỏa</strong>, nhưng tích lũy <strong>20.000 Đạo Ngân Thủy</strong> (Thủy &amp; Hỏa tương khắc sâu, compatibility = 0.55 &rarr; xung đột = 1.0 &minus; 0.55 = 0.45).
              </p>
              <div className="bg-zinc-950/80 border border-zinc-800 rounded-lg p-3 font-mono text-[11px] text-[var(--text-primary)] space-y-1.5">
                <div>• Xung đột thô: rawConflict = 20.000 &times; 0.45 = <span className="text-amber-400">9.000</span></div>
                <div className="pt-1 border-t border-zinc-900">
                  <span className="text-rose-400 font-bold">Trường hợp A: Phàm Thai (RealmControl = 0%)</span>
                  <div>• effConflict = 9.000 &times; (1.0 &minus; 0.0) = 9.000</div>
                  <div>• conflictRatio = 9.000 / (10.000 + 50.000) = 0.15</div>
                  <div>• conflictMult = 1.0 / (1.0 + 0.15 &times; 1.6) = <span className="text-rose-400 font-bold">0.806</span> (&rarr; <strong>Mất 19.4% sát thương</strong> do xung đột Thủy Hỏa!)</div>
                </div>
                <div className="pt-1 border-t border-zinc-900">
                  <span className="text-emerald-400 font-bold">Trường hợp B: Vô Thượng Đại Tông Sư (RealmControl = 80%)</span>
                  <div>• effConflict = 9.000 &times; (1.0 &minus; 0.80) = 1.800</div>
                  <div>• conflictRatio = 1.800 / 60.000 = 0.03</div>
                  <div>• conflictMult = 1.0 / (1.0 + 0.03 &times; 1.6) = <span className="text-emerald-400 font-bold">0.954</span> (&rarr; <strong>Chỉ giảm 4.6% sát thương</strong>, Chân Ý hóa giải 76% tổn thất!)</div>
                </div>
              </div>
            </div>

            {/* Example 3 */}
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-blue-500/20 space-y-2">
              <h4 className="font-bold text-blue-300 font-serif">
                Ví dụ 3: Màng Phòng Thủ Đạo Ngân Tự Nhiên (Natural Dao Resistance)
              </h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                Cổ Tiên Thủy Đạo sở hữu <strong>100.000 Đạo Ngân Thủy</strong> bị tấn công bởi sát chiêu Hỏa Đạo (compatibility = 0.55 &rarr; xung đột thủ = 0.45).
              </p>
              <div className="bg-zinc-950/80 border border-zinc-800 rounded-lg p-3 font-mono text-[11px] text-[var(--text-primary)] space-y-1">
                <div>• Xung đột phòng thủ: defConflict = 100.000 &times; 0.45 = <span className="text-amber-400">45.000</span></div>
                <div>• Tỉ lệ kháng tự nhiên: Resistance = 0.30 &times; [45.000 / (45.000 + 50.000)] = 0.30 &times; (45.000 / 95.000) = <span className="text-blue-400 font-bold">14.21%</span></div>
                <div className="text-[var(--text-muted)]">&rarr; Đòn đánh Lửa của đối phương tự động bị triệt tiêu 14.21% sát thương ngay khi chạm vào lớp màng Thủy Đạo tự nhiên!</div>
              </div>
            </div>

            {/* Example 4 */}
            <div className="p-4 rounded-xl bg-gradient-to-r from-purple-950/20 to-indigo-950/20 border border-indigo-500/20 space-y-2">
              <h4 className="font-bold text-indigo-300 font-serif">
                Ví dụ 4: Tính Sát Thương Thực Tế Toàn Diện (A &rarr; Z)
              </h4>
              <div className="text-[var(--text-secondary)] space-y-1.5 leading-relaxed">
                <p><strong>Bên Tấn Công:</strong> Sát thương gốc = 1.000, Chiêu thức DamageMult = 2.0, Buff = 1.2. Có 5.000 Đạo Ngân Lực Đạo (daoAmp = 51.0), Tông Sư Lực Đạo (Eff = 1.12, Ctrl = 25%), mang 10.000 Đạo Ngân Hồn Đạo ngoại hệ (comp = 0.88 &rarr; xung đột = 0.12).</p>
                <div className="bg-zinc-950/80 border border-zinc-800 rounded-lg p-3 font-mono text-[11px] space-y-1 text-zinc-300">
                  <div>1. totalDaoMult = 1.0 + (51.0 &minus; 1.0) &times; 1.12 = <span className="text-amber-400">57.0</span></div>
                  <div>2. effConflict = (10.000 &times; 0.12) &times; (1.0 &minus; 0.25) = 900</div>
                  <div>3. conflictRatio = 900 / (5.000 + 50.000) = 0.01636 &rarr; conflictMult = <span className="text-rose-400">0.974</span></div>
                  <div>4. hitDmg = 1.000 &times; 2.0 &times; 1.2 &times; 57.0 &times; 0.974 = <span className="text-amber-400 font-bold">133.243</span> sát thương</div>
                </div>
                <p><strong>Bên Phòng Thủ:</strong> Phòng thủ BaseDef = 2.000, Màng kháng tự nhiên Resistance = 10%.</p>
                <div className="bg-zinc-950/80 border border-zinc-800 rounded-lg p-3 font-mono text-[11px] text-emerald-400 font-bold">
                  netDmg = (133.243 &minus; 2.000) &times; (1.0 &minus; 0.10) = 131.243 &times; 0.90 = 118.118 Sinh Lực mất thực tế!
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Build Strategies */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <BookOpen className="w-4 h-4 text-emerald-400" />
            <span>6. Ba Hướng Xây Dựng Nhân Vật (Build Paths)</span>
          </h3>

          <div className="space-y-3 text-xs">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-emerald-500/20 space-y-1.5">
              <h4 className="font-bold text-emerald-300 flex items-center gap-2">
                <span>1. Chuyên Tu (Pure Focus) - Lối Tu Truyền Thống Của Cổ Tiên</span>
              </h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                • <strong>Cách chơi:</strong> Tập trung 100% Đạo Ngân và Chân Ý vào duy nhất 1 lưu phái (hoặc các phái trong cùng 1 Nhóm Đạo).<br />
                • <strong>Ưu điểm:</strong> Xung đột bằng 0 (Conflict Multiplier = 1.0). Sát thương đồng hệ được khuếch đại cực đại, dễ dàng đạt ngưỡng hủy diệt.<br />
                • <strong>Nhược điểm:</strong> Bị động khi gặp đối thủ có kháng cao hoặc thiên địch khắc chế hệ phái của mình.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-indigo-500/20 space-y-1.5">
              <h4 className="font-bold text-indigo-300 flex items-center gap-2">
                <span>2. Song Tu / Phụ Tu Bổ Trợ (Dual Synergic) - Lối Chơi Chiến Thuật</span>
              </h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                • <strong>Cách chơi:</strong> Chọn 1 phái Chính và 1 phái Phụ có độ tương thích cao (Ví dụ: Lực + Biến Hóa, Hồn + Nô, Luyện + Hỏa). Đưa phái chính lên Tông Sư / Đại Tông Sư.<br />
                • <strong>Ưu điểm:</strong> Bộ chiêu thức đa dạng, công thủ toàn diện. Xung đột chỉ ở mức rất thấp (5% &ndash; 15%) nhờ Chân Ý điều hòa.<br />
                • <strong>Khuyên dùng:</strong> Phù hợp cho phần lớn Cổ Tiên leo tháp, vượt ải phó bản và thi đấu PvP Đấu Trường Sinh Tử.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-purple-500/20 space-y-1.5">
              <h4 className="font-bold text-purple-300 flex items-center gap-2">
                <span>3. Đa Tu / Toàn Năng (Omni Master & Chí Tôn Tiên Thể)</span>
              </h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                • <strong>Cách chơi:</strong> Tu luyện nhiều lưu phái đối nghịch cùng lúc. Đòi hỏi phải cày Chân Ý Mộng Cảnh lên mốc <strong>Đại Tông Sư / Vô Thượng</strong> ở tất cả các phái.<br />
                • <strong>Đặc tính Chí Tôn (Non-Conflicting Body):</strong> Cổ Tiên sở hữu thể chất đặc biệt không xung đột sẽ bỏ qua 100% hình phạt xung đột khi tấn công (luôn gây 100% sát thương dù mang bao nhiêu Đạo Ngân khác hệ). Đổi lại, họ cũng <strong>không có Màng Kháng Đạo Ngân Tự Nhiên</strong> khi phòng thủ.
              </p>
            </div>
          </div>
        </div>
      </div>
    ),
  },
];
