import React from "react";
import Link from "next/link";
import { AlertTriangle, Clock } from "lucide-react";
import { WikiPage } from "./types";

export const characterPages: WikiPage[] = [
  {
    id: "gameplay-character",
    title: "Nhân Vật",
    category: "gameplay",
    keywords: ["nhân vật", "character", "hồ sơ", "profile", "chỉ số", "linh căn", "kỹ năng", "thông thạo", "hồn phách", "nhân khí", "tư chất"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Hồ sơ nhân vật chứa toàn bộ thông tin tu vi, không khiếu, các chỉ số cốt lõi, Hồn Phách và các Body Mods vĩnh viễn tích lũy từ dung hợp Cổ trùng. Đây là &quot;gương mặt&quot; của đạo hữu trong giới tu tiên: thân phận, chủng tộc, thể chất, cảnh giới lưu phái lẫn ngụy trang đều được phản chiếu tại đây.
        </p>
        <div className="p-4 rounded-2xl bg-amber-950/30 border border-amber-500/20 text-xs text-[var(--text-secondary)] leading-relaxed">
          <strong className="text-amber-200">Tổng quan một Cổ sư:</strong> Mỗi đạo hữu là một phàm nhân thức tỉnh Không Khiếu (hoặc Cổ Tiên sở hữu Tiên Khiếu) được định hình bởi ba lớp: <strong>tu vi</strong> (Cảnh giới + Tiểu Cảnh Giới), <strong>thân thể</strong> (chủng tộc, giới tính, thể chất, trạng thái) và <strong>linh hồn</strong> (Hồn Phách, thông thạo lưu phái, Đạo Ngân).
        </div>
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Nhấp vào các thẻ bên dưới để xem hướng dẫn chi tiết từng mục:
        </p>
      </div>
    ),
    cards: [
      {
        title: "Chỉ số",
        targetId: "gameplay-character-chiso",
        description: "Cảnh giới, Sinh Lực, Chân Nguyên, Hồn Phách, Nhân Khí, Thọ nguyên."
      },
      {
        title: "Thuộc tính & Áp lực",
        targetId: "gameplay-character-thuoctinh",
        description: "Chủng tộc, thể chất, trạng thái và Áp lực vách khiếu."
      },
      {
        title: "Linh Căn (Tư chất)",
        targetId: "gameplay-character-linhcan",
        description: "Phẩm cấp Giáp/Ất/Bính/Đinh và cách nâng tư chất Không khiếu."
      },
      {
        title: "Xung Đột Đạo Ngân & Cân Bằng Lưu Phái",
        targetId: "gameplay-dao-conflict",
        description: "10 Nhóm Đạo, cơ chế triệt tiêu xung đột và màng kháng tự nhiên."
      },
      {
        title: "Thông thạo (Cảnh giới)",
        targetId: "gameplay-character-thongthao",
        description: "Cảnh giới lưu phái và hệ thống Đạo Ngân 49 lưu phái."
      },
      {
        title: "Kỹ năng (Cổ & Sát chiêu)",
        targetId: "gameplay-character-kynang",
        description: "Điều khiển Cổ trùng công, thủ, trị liệu và Sát Chiêu phối hợp."
      }
    ]
  },
  {
    id: "gameplay-character-chiso",
    title: "Chỉ Số Nhân Vật",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["chỉ số", "stats", "hp", "mana", "chân nguyên", "tu vi", "level", "rank", "subrank", "hồn phách", "nhân khí", "thọ nguyên", "lifespan", "soul"],
    content: (
      <div className="space-y-6">
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Các chỉ số tu luyện</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Bảng dưới liệt kê toàn bộ chỉ số phản ánh trên hồ sơ nhân vật. Phàm nhân (Nhất Chuyển – Ngũ Chuyển) dùng Không Khiếu, còn Cổ Tiên (Lục Chuyển trở lên) chuyển sang Tiên Khiếu với quy luật riêng.
          </p>

          <div className="overflow-hidden rounded-2xl border border-[var(--border)]">
            <table className="w-full text-left text-xs font-mono">
              <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                <tr>
                  <th className="p-3">Tên chỉ số</th>
                  <th className="p-3">Mô tả chi tiết</th>
                  <th className="p-3">Tác dụng</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Cảnh Giới</td>
                  <td className="p-3">Đại cảnh giới tu vi của nhân vật, từ Nhất Chuyển đến Cửu Chuyển. Nhất–Ngũ Chuyển là phàm nhân, Lục Chuyển trở lên là Cổ Tiên.</td>
                  <td className="p-3">Mở khóa tính năng mới (Lục Chuyển mở Tiên Khiếu, v.v.).</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tiểu Cảnh Giới (Sub-Rank)</td>
                  <td className="p-3">Tiểu cảnh giới: Sơ Giai, Trung Giai, Cao Giai, Đỉnh Phong. Chỉ tồn tại ở phàm nhân (Chuyển 1–5).</td>
                  <td className="p-3">Mốc để trùng kích vách khiếu đột phá lên Chuyển mới.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Sinh Lực (HP / Base HP)</td>
                  <td className="p-3">Sức chịu đựng của cơ thể. Phàm nhân khởi đầu cố định ở mức rất thấp (100 Sinh Lực).</td>
                  <td className="p-3">Nếu tụt về 0 trong chiến đấu, bạn sẽ bị trọng thương hoặc chết vĩnh viễn.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Chân Nguyên / Tiên Nguyên</td>
                  <td className="p-3">Năng lượng trong Không Khiếu (phàm nhân) hoặc Tiên Khiếu (Cổ Tiên). Dùng để kích hoạt Cổ trùng ra chiêu.</td>
                  <td className="p-3">Tiêu hao liên tục trong chiến đấu và hồi phục tự động theo tư chất (Cổ Tiên hồi chậm hơn nhưng trữ lượng khổng lồ).</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Hồn Phách (Soul HP / Max Soul)</td>
                  <td className="p-3">Nội lực tinh thần giới hạn số lượng hoang thú, Cổ trùng và đạo pháp khống chế mà linh hồn chịu đựng được.</td>
                  <td className="p-3">Tráng Hồn tăng 10 điểm; đạt giới hạn tầng Hồn phải Luyện Hồn tại Cốc Lạc Phách để bứt phá tầng mới.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tầng Hồn (Soul Tier)</td>
                  <td className="p-3">Phẩm cấp linh hồn: Nhân Hồn ➔ Bách Nhân Hồn ➔ Thiên Nhân Hồn ➔ Vạn Nhân Hồn ➔ Bách Vạn Nhân Hồn ➔ Thiên Vạn Nhân Hồn.</td>
                  <td className="p-3">Mỗi tầng nâng giới hạn tối đa của Hồn Phách lên một bậc.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Nhân Khí (Human Qi)</td>
                  <td className="p-3">Tài nguyên phản ánh hoạt động tích cực của đạo hữu trong giới tu tiên, giới hạn tối đa 500.000.</td>
                  <td className="p-3">Quyết định tỉ lệ Thăng Tiên thành công (mỗi 100k Nhân Khí +20% tỉ lệ, cap 90%) và phẩm cấp Tiên Khiếu khi mở.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tư chất Không Khiếu</td>
                  <td className="p-3">Phẩm cấp Linh Căn (Giáp/Ất/Bính/Đinh kèm hệ số phần trăm).</td>
                  <td className="p-3">Ảnh hưởng trữ lượng Chân Nguyên tối đa, tốc độ hồi phục và hiệu suất phá vách.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Thọ nguyên (Lifespan)</td>
                  <td className="p-3">Tuổi thọ còn lại của cơ thể, hao mòn dần theo thời gian thực (tỷ lệ thuận với hệ số gia tốc thời gian tu luyện).</td>
                  <td className="p-3">Về 0 đồng nghĩa sinh mệnh tàn lụi (<span className="text-red-400 font-semibold">is_dead</span>) — bị khóa mọi hoạt động tu luyện &amp; chiến đấu. Cần dùng Thọ Nguyên Cổ hoặc chuyển thành Cương Thi.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Áp lực vách khiếu</td>
                  <td className="p-3">Mức độ tổn thương của Không Khiếu khi trùng kích quá độ.</td>
                  <td className="p-3">Vượt 80% làm suy giảm hiệu quả; chạm 100% Không Khiếu bị nứt vỡ (rớt tu vi, mất vĩnh viễn Chân Nguyên tối đa).</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div className="space-y-4 pt-4 border-t border-zinc-900">
            <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Bảng Quy Đổi Đơn Vị Số Lượng (Vạn, Ức, Kinh...)</h3>
            <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
              Hệ thống hiển thị chỉ số chiến đấu, sát thương, khiên ảo và tài nguyên theo thang đơn vị truyền thống Á Đông (mỗi bậc tăng 10.000 lần = 10⁴).
            </p>

            <div className="overflow-hidden rounded-2xl border border-[var(--border)]">
              <table className="w-full text-left text-xs font-mono">
                <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                  <tr>
                    <th className="p-2.5">Đơn vị</th>
                    <th className="p-2.5">Lũy thừa ($10^n$)</th>
                    <th className="p-2.5">Giá trị bằng số</th>
                    <th className="p-2.5">Diễn giải</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Vạn</td>
                    <td className="p-2.5">10⁴</td>
                    <td className="p-2.5">10.000</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">10 nghìn</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Ức</td>
                    <td className="p-2.5">10⁸</td>
                    <td className="p-2.5">100.000.000</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">100 triệu (1 vạn vạn)</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Triệu</td>
                    <td className="p-2.5">10¹²</td>
                    <td className="p-2.5">1.000.000.000.000</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">1 nghìn tỷ (1 triệu triệu / 1 vạn ức)</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Kinh</td>
                    <td className="p-2.5">10¹⁶</td>
                    <td className="p-2.5">10.000.000.000.000.000</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">10 triệu tỷ (10,000 Triệu)</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Cai</td>
                    <td className="p-2.5">10²⁰</td>
                    <td className="p-2.5">10²⁰</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">100 nghìn tỷ tỷ</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Tử</td>
                    <td className="p-2.5">10²⁴</td>
                    <td className="p-2.5">10²⁴</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">1 triệu tỷ tỷ</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Nhưỡng</td>
                    <td className="p-2.5">10²⁸</td>
                    <td className="p-2.5">10²⁸</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">10 tỷ tỷ tỷ</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Câu</td>
                    <td className="p-2.5">10³²</td>
                    <td className="p-2.5">10³²</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">100 triệu tỷ tỷ tỷ</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Giản</td>
                    <td className="p-2.5">10³⁶</td>
                    <td className="p-2.5">10³⁶</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">1 trạch / 1 tỷ tỷ tỷ tỷ</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Chính</td>
                    <td className="p-2.5">10⁴⁰</td>
                    <td className="p-2.5">10⁴⁰</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">10,000 Giản</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Tải</td>
                    <td className="p-2.5">10⁴⁴</td>
                    <td className="p-2.5">10⁴⁴</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">10,000 Chính</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Cực</td>
                    <td className="p-2.5">10⁴⁸</td>
                    <td className="p-2.5">10⁴⁸</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">10,000 Tải</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Hằng Hà Sa</td>
                    <td className="p-2.5">10⁵²</td>
                    <td className="p-2.5">10⁵²</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">Nhiều như số cát sông Hằng</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">A Tăng Kỳ</td>
                    <td className="p-2.5">10⁵⁶</td>
                    <td className="p-2.5">10⁵⁶</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">Vô số lượng lớn (Asamkhyeya)</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Na Do Tha</td>
                    <td className="p-2.5">10⁶⁰</td>
                    <td className="p-2.5">10⁶⁰</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">Vô lượng số (Nayuta)</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Bất Khả Tư Nghị</td>
                    <td className="p-2.5">10⁶⁴</td>
                    <td className="p-2.5">10⁶⁴</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">Vượt xa suy tư tư duy</td>
                  </tr>
                  <tr>
                    <td className="p-2.5 font-bold text-amber-400">Vô Lượng</td>
                    <td className="p-2.5">10⁶⁸</td>
                    <td className="p-2.5">10⁶⁸</td>
                    <td className="p-2.5 text-[var(--text-secondary)]">Cực hạn vô lượng</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "gameplay-character-thuoctinh",
    title: "Thuộc tính & Áp lực vách khiếu",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["stress", "áp lực", "vách khiếu", "không khiếu", "aperture stress", "nứt", "rớt cấp", "tịnh thủy", "chủng tộc", "thể chất", "trạng thái"],
    content: (
      <div className="space-y-6">
        <div className="p-4 rounded-2xl bg-amber-950/30 border border-amber-500/20 flex gap-3 items-start">
          <AlertTriangle className="w-5 h-5 text-amber-500 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-amber-200">Cơ chế bảo vệ không khiếu</h4>
            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              Việc ép chân nguyên liên tục xông phá vách khiếu sẽ gây tổn thương nặng nề lên Không khiếu. Bạn cần chú ý chỉ số <strong>Aperture Stress</strong> để tránh hậu quả đáng tiếc.
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Hệ thống áp lực</h3>
          <ul className="list-disc list-inside text-xs text-[var(--text-secondary)] space-y-2 leading-relaxed">
            <li>Mỗi lần click nút <strong>Trùng Kích Vách Khiếu</strong>, thanh áp lực tăng thêm <strong>+5</strong>.</li>
            <li>Thanh áp lực tự động suy giảm theo thời gian thực: cứ <strong>mỗi 2 giờ đời thật</strong>, áp lực giảm đi <strong>phân nửa</strong>.</li>
            <li>Nếu áp lực vượt mốc <strong>80%</strong>, hiệu quả trùng kích chỉ còn <strong>25%</strong> so với bình thường.</li>
            <li>Nếu áp lực chạm mốc <strong>100%</strong>, Không khiếu bị nứt: đạo hữu <strong>rớt 1 Tiểu cảnh giới (hoặc 1 Chuyển nếu đang ở Sơ Giai)</strong>, tiến độ vách khiếu về 0, áp lực giảm về 50%, và <strong>mất vĩnh viễn 5% Chân Nguyên tối đa</strong>!</li>
          </ul>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] text-xs leading-relaxed text-[var(--text-secondary)]">
            💡 <strong>Mẹo:</strong> Sử dụng vật phẩm tiêu hao <strong>Tịnh Thủy Cổ</strong> để lập tức thanh lọc, đưa thanh Áp lực vách khiếu về 0% ngay tức khắc.
          </div>
        </div>

        {/* Lifespan & Death / Cương Thi Section */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Clock className="w-4 h-4 text-amber-400" />
            <span>Hệ Thống Tuổi Thọ, Trạng Thái Tử Vong & Tiên Cương</span>
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2">
              <h4 className="font-bold text-red-400 font-serif">1. Tuổi Thọ & Trạng Thái Tử Vong (is_dead)</h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                • Tuổi thọ của Cổ sư bị hao mòn liên tục theo thời gian thực (nhân với hệ số gia tốc <span className="font-mono text-amber-300">time_speed_multiplier</span>).<br />
                • Khi tuổi thọ tụt về <strong>0</strong>, nhân vật đi vào trạng thái tử vong (<span className="text-red-400 font-mono">is_dead = true</span>).<br />
                • Cổ sư tử vong bị phong tỏa hoàn toàn các thao tác chiến đấu, khai thác, thám hiểm và tu luyện cho đến khi dùng <strong>Thọ Nguyên Cổ</strong> kéo dài tuổi thọ hoặc biến đổi chủng tộc.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2">
              <h4 className="font-bold text-emerald-400 font-serif">2. Chủng Tộc Cương Thi (Tiên Cương)</h4>
              <p className="text-[var(--text-secondary)] leading-relaxed">
                • <strong>Bất tử tuổi thọ:</strong> Thọ nguyên không bao giờ suy giảm theo thời gian.<br />
                • <strong>Debuff Tử Thi:</strong> Bị phong tỏa hoàn toàn tự hồi phục HP, Chân Nguyên / Tiên Nguyên, Nguyên Lực và Thần Hồn.<br />
                • <strong>Tiên Khiếu Chết:</strong> Gia tốc thời gian cá nhân bị cố định ở <span className="font-mono text-amber-300">1.0</span> (không thể tăng tốc), đồng thời Tiên Khiếu bị suy thoái giảm dần diện tích (<span className="font-mono">area_size</span>) và tốc độ thời gian (<span className="font-mono">time_rate</span>) theo thời gian thực.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Thuộc tính thân thể</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Ngoài tu vi, hồ sơ còn ghi nhận <strong>Chủng tộc</strong> (Nhân Tộc, Cương Thi, Người Lông, Nhân Ngư, Thạch Nhân, Long Nhân...), <strong>Giới tính</strong>, <strong>Thể chất</strong> (Hoang Cổ Nguyệt Thể, Thanh Liên Đại Nhật Thể...) và mảng <strong>Trạng thái</strong> (ví dụ Trọng Thương giảm 20% chiến lực). Các giá trị này mang hiệu ứng riêng lên tu luyện và chiến đấu, xem chi tiết trong mục Hồ Sơ.
          </p>
        </div>
      </div>
    )
  },
  {
    id: "gameplay-character-linhcan",
    title: "Linh Căn & Tư Chất Không Khiếu",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["tư chất", "linh căn", "aptitude", "giáp", "ất", "bính", "đinh", "hệ số", "multiplier", "hiến tế"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Khi đăng ký nhân vật, bạn sẽ được thức tỉnh không khiếu ngẫu nhiên với một trong bốn phẩm cấp tư chất (Linh Căn). Tư chất quyết định lượng chân nguyên lưu trữ tối đa, tốc độ hồi phục và đặc biệt là hiệu suất phá vách khiếu tu luyện.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-[var(--bg-card)] border border-emerald-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-emerald-400">Giáp (A)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Thượng đẳng</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>1.4x</strong> · Hồi Chân Nguyên <strong>8/giờ</strong>. Tỉ lệ mở ra 80–99%.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-cyan-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-cyan-400">Ất (B)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Trung đẳng</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>1.1x</strong> · Hồi Chân Nguyên <strong>6/giờ</strong>. Tỉ lệ mở ra 60–79%.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-amber-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-amber-500">Bính (C)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Hạ đẳng</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>0.85x</strong> · Hồi Chân Nguyên <strong>4/giờ</strong>. Tỉ lệ mở ra 40–59%.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-rose-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-rose-500">Đinh (D)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Cực hạ</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>0.6x</strong> · Hồi Chân Nguyên <strong>2/giờ</strong>. Tỉ lệ mở ra 20–39%.
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Cách nâng tư chất</h3>
          <ul className="list-disc list-inside text-xs text-[var(--text-secondary)] space-y-2 leading-relaxed">
            <li><strong>Hiến tế NPC</strong> bằng Huyết Sọ Cổ: nghiền nát cốt nhục đệ tử môn phái để tăng <strong>+5% tư chất</strong> (tối đa 99%), nhưng làm tăng <strong>+40 áp lực vách khiếu</strong>.</li>
            <li><strong>Hiệu ứng Cổ:</strong> một số Cổ trùng có hiệu ứng tăng trực tiếp tư chất Không khiếu (tối đa 99%).</li>
            <li>Khi tư chất thay đổi, phẩm cấp Linh Căn tự động cập nhật theo ngưỡng: &ge;80% = Giáp, &ge;60% = Ất, &ge;40% = Bính, dưới 40% = Đinh.</li>
            <li>Gợi ý cho đạo hữu Bính/Đinh: tận dụng <strong>Tửu Trùng</strong> và <strong>Xá Lợi Cổ</strong> để bù lại tốc độ tu luyện chậm chạp.</li>
          </ul>
        </div>
      </div>
    )
  },
  {
    id: "gameplay-character-thongthao",
    title: "Thông Thạo & Cảnh Giới Lưu Phái",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["cảnh giới", "thông thạo", "attainment", "đại sư", "tông sư", "đại tông sư", "vô thượng", "chân ý", "master", "grandmaster", "đạo ngân"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Cảnh giới lưu phái thể hiện sự thấu hiểu của linh hồn bạn đối với quy luật vạn vật thế gian. Tích lũy cảnh giới lưu phái thông qua việc rèn luyện sát chiêu chiến đấu hoặc hấp thu Chân Ý khi thám hiểm Mộng Cảnh. Song song đó, <strong>Đạo Ngân</strong> (vết khắc của quy luật tự nhiên) ghi lại độ lĩnh ngộ theo từng lưu phái.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Các mốc cảnh giới lưu phái</h3>

          <div className="space-y-3">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>1. Chuẩn Đại Sư / Đại Sư (100 Chân Ý)</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Trực giác siêu nhạy bén. Giảm <strong>20% xung đột Đạo Ngân</strong> ngoại hệ và tăng <strong>+10% hiệu suất khuếch đại Đạo Ngân</strong> cho lưu phái này.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-indigo-300 flex items-center gap-2">
                <span>2. Chuẩn Tông Sư / Tông Sư (1.000 Chân Ý)</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Đạt khả năng &quot;Loại suy&quot; (Thấu triệt tương sinh tương khắc). Giảm <strong>40% xung đột Đạo Ngân</strong> và tăng <strong>+20% hiệu suất khuếch đại</strong>. Điều kiện bắt buộc để đột phá Thất Chuyển.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-yellow-500 flex items-center gap-2">
                <span>3. Chuẩn Đại Tông Sư / Đại Tông Sư (10.000 Chân Ý)</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Cộng hưởng hoàn hảo với Đạo ngân của thiên nhiên. Tự nhiên hóa Đạo Ngân, giảm <strong>60% xung đột</strong> và tăng <strong>+30% hiệu suất khuếch đại</strong>. Điều kiện bắt buộc để đột phá Bát Chuyển.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-red-400 flex items-center gap-2">
                <span>4. Vô Thượng Đại Tông Sư (100.000 Chân Ý)</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Cảnh giới chí cao vô thượng. Áp chế tối đa <strong>80% xung đột</strong>, đẩy hiệu suất khuếch đại lên cực hạn <strong>+45%</strong>, mở khóa năng lực Thôn tính Tiên khiếu đồng hệ. Điều kiện bước vào Cửu Chuyển Tôn Giả.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Đạo Ngân (Dao Marks)</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Đạo Ngân là vết khắc quy luật tự nhiên tích lũy trên cơ thể và Tiên Khiếu qua <strong>49 lưu phái</strong> (chia thành 10 Nhóm Đạo). Mỗi điểm Đạo Ngân gia tăng sát thương và phòng ngự cho Cổ/Kỹ năng thuộc lưu phái đó.
          </p>
          <div className="overflow-hidden rounded-2xl border border-[var(--border)]">
            <table className="w-full text-left text-xs font-mono">
              <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                <tr>
                  <th className="p-3">Nguồn nhận Đạo Ngân</th>
                  <th className="p-3">Mô tả</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Thăng Tiên</td>
                  <td className="p-3">Khởi đầu nhận Đạo Ngân tương ứng với Thiên Địa Nhị Khí hội tụ.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tai Kiếp (Thiên Kiếp / Địa Tai)</td>
                  <td className="p-3">Vượt qua kiếp nạn Tiên Khiếu nhận hàng ngàn đến hàng vạn Đạo Ngân.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Sát Chiêu & Luyện Cổ</td>
                  <td className="p-3">Rèn luyện sát chiêu và luyện Tiên Cổ tích lũy Đạo Ngân vào lưu phái cốt lõi.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Nuốt Tiên Khiếu</td>
                  <td className="p-3">Thôn tính phúc địa / động thiên của Cổ Tiên khác để kế thừa toàn bộ Đạo Ngân.</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="p-4 rounded-2xl bg-indigo-950/20 border border-indigo-500/20 text-xs leading-relaxed text-indigo-300">
            👉 Xem chi tiết ma trận 10 nhóm Đạo, cơ chế triệt tiêu xung đột và màng kháng tự nhiên tại trang <Link href="/wiki/gameplay-dao-conflict" className="text-amber-400 font-bold underline">Xung Đột Đạo Ngân & Cân Bằng Lưu Phái</Link>.
          </div>
        </div>
      </div>
    )
  },
  {
    id: "gameplay-character-kynang",
    title: "Kỹ năng & Sát Chiêu",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["kỹ năng", "sát chiêu", "combo", "active", "passive", "body mods", "killer moves", "cổ hạch tâm"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Trong game, Cổ sư không học kỹ năng phép thuật trực tiếp. Thay vào đó, sức mạnh chiến đấu của bạn phụ thuộc vào việc kết hợp các con Cổ trùng khác nhau (Sát chiêu) hoặc dùng Cổ trùng thụ động cải tạo thân thể.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Cấu trúc Sát Chiêu</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Sát Chiêu (Killer Move) là kỹ năng ghép từ <strong>1 Cổ Hạch Tâm (Core Gu)</strong> và <strong>0–8 Cổ Phụ Trợ (Support Gu)</strong>. Cổ phụ trợ có thể tăng sát thương, giảm hao Chân Nguyên, thêm khống chế (choáng, đóng băng, câm lặng) hoặc rút ngắn thời gian tụ lực.
          </p>
          <ul className="list-disc list-inside text-xs text-[var(--text-secondary)] space-y-2 leading-relaxed">
            <li><strong>Cổ Hạch Tâm</strong> quyết định lưu phái và hiệu ứng chính của sát chiêu; sát chiêu rèn luyện tích lũy Đạo Ngân vào lưu phái cốt lõi.</li>
            <li><strong>Body Mods</strong> là cải tạo thân thể vĩnh viễn tích lũy từ dung hợp Cổ trùng, phản ánh qua các chỉ số thân thể trên hồ sơ.</li>
            <li>Trạng thái thân thể (ví dụ Trọng Thương) làm suy giảm sức mạnh sát chiêu cho đến khi hồi phục.</li>
          </ul>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Sát Chiêu Phối Hợp</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Bằng cách trang bị và kích hoạt đồng thời một nhóm Cổ trùng bổ trợ nhau trong lượt đánh, bạn sẽ tạo ra sát chiêu cộng hưởng sức mạnh khủng khiếp:
          </p>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-2 text-[var(--text-secondary)] leading-relaxed">
            <li>💥 <strong>Lực Đạo combo:</strong> Trang bị đồng thời <strong>Toàn Lực Ứng Phó Cổ</strong> (kích hoạt 100% thú ảnh) và <strong>Khổ Lực Cổ</strong> (tăng sát thương dựa trên Sinh Lực đã mất) để tối đa lượng sát thương bộc phát.</li>
            <li>❤️ <strong>Trị liệu combo:</strong> Sử dụng <strong>Tự Lực Cánh Sinh Cổ</strong> hồi phục sinh lực dựa trên chỉ số Sức mạnh thể chất của bạn. Sức mạnh càng lớn hồi máu càng nhiều.</li>
          </div>
        </div>
      </div>
    )
  },
];
