import React from "react";
import { Sparkles, Compass, Flame } from "lucide-react";
import { WikiPage } from "./types";

export const equipmentJobsPages: WikiPage[] = [
  {
    id: "equipment",
    title: "Trang bị & Cổ Trùng",
    category: "equipment",
    keywords: ["trang bị", "cổ trùng", "gu", "moonlight", "boar", "relic", "liquor", "nuôi cổ", "hunger", "thức ăn"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Cổ trùng được chia làm 9 Chuyển. Nhất Chuyển đến Ngũ Chuyển là Phàm Cổ, Lục Chuyển đến Cửu Chuyển là Tiên Cổ (độc nhất vô nhị). Chúng được chia thành các nhóm chính: Công kích, Phòng ngự, Trinh sát, Trị liệu, Phụ trợ và Di động.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Hệ thống nuôi Cổ trùng</h3>
          <div className="p-4 rounded-2xl bg-rose-950/20 border border-rose-500/10 text-xs text-rose-300 leading-relaxed">
            ⚠️ <strong>Lưu ý cực kỳ quan trọng:</strong> Mỗi con Cổ (dù đang trang bị hay trong Bảo Khố) đều có thanh độ no. Độ no giảm dần <strong>sau mỗi 3 giờ thực tế</strong>. Nếu độ no về 0, Cổ sẽ rơi vào trạng thái <strong>Hấp Hối</strong> và sẽ <strong>chết vĩnh viễn</strong> nếu không được cho ăn kịp thời!
          </div>

          <p className="text-xs text-[var(--text-muted)] leading-relaxed mt-2">
            Bạn cần thu thập thức ăn tương ứng của từng hệ để nuôi dưỡng chúng:
          </p>
          <div className="overflow-hidden rounded-2xl border border-[var(--border)]">
            <table className="w-full text-left text-xs font-mono">
              <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                <tr>
                  <th className="p-3">Tên Cổ Trùng</th>
                  <th className="p-3">Thức ăn yêu cầu</th>
                  <th className="p-3">Nguồn gốc thức ăn</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Nguyệt Quang Cổ</td>
                  <td className="p-3">Cánh Hoa Nguyệt Lan</td>
                  <td className="p-3">Trồng tại Tiên Khiếu hoặc mua tại Tiệm Caravan</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tửu Trùng</td>
                  <td className="p-3">Rượu ngon (Thanh Trúc, Hầu Nhi...)</td>
                  <td className="p-3">Sản xuất từ quầy chưng cất, dã ngoại</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Bạch Thỉ Cổ / Hắc Thỉ Cổ</td>
                  <td className="p-3">Thịt lợn rừng</td>
                  <td className="p-3">Săn bắn dã ngoại hoang dã</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tiên Cổ Thái Độ</td>
                  <td className="p-3">Quả Lưu Quang (số lượng lớn)</td>
                  <td className="p-3">Khai thác tại các bí cảnh tiên đạo cực nam</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "jobs",
    title: "Nghề & Luyện Cổ",
    category: "jobs",
    keywords: ["nghề", "luyện cổ", "refine", "chế tạo", "lò luyện", "furnace", "cổ phương", "recipe", "tỉ lệ"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Luyện Cổ là con đường chế tạo Cổ trùng thông qua Lò Luyện Cổ nhân tạo hoặc dựa vào linh khí thiên địa.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Cơ chế Lò Luyện</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Tỉ lệ Luyện Cổ thành công được quyết định bởi 10 yếu tố: Tỷ lệ gốc của bí phương, Độ hoàn thiện của phương thức, Độ chính xác tổ hợp nguyên liệu, Chất lượng nguyên liệu đầu vào (+2% mỗi bậc chênh lệch), Đạo Ngân Luyện Đạo (không giới hạn, tăng chậm dần theo hàm căn bậc hai, nhân hệ số Chân Ý lưu phái đích và chia theo hệ số Rank P), Cảnh giới Luyện Đạo (lên tới +25%), Cảnh giới Lưu Phái của Cổ (max +10%), Khí vận (max +10%), Cổ phụ trợ kích hoạt, và Môi trường Đại Vực (+5%). Lắp sai nguyên liệu lập tức tỷ lệ về 0%.
            <br /><br />
            <strong className="text-cyan-400">Tiêu hao Năng Lượng:</strong> Cổ sư luyện Cổ đồng cấp tốn 20 Chân Nguyên (hoặc 10 Tiên Nguyên nếu là Tiên Cổ). Chi phí này nhân lên hoặc giảm đi 10 lần (với Phàm) và 100 lần (với Tiên) cho mỗi bậc chênh lệch. Phàm nhân không thể luyện Tiên Cổ vì tiêu hao lập tức cạn kiệt khiếu.
            <br /><br />
            <strong className="text-cyan-400">Tiên Cổ Giảm Trừ:</strong> Khi luyện Tiên Cổ (Rank 6 - 9), các chỉ số cộng thẳng ngoại vật (Chất lượng vật liệu, Cảnh giới, Khí vận, Phụ trợ, Môi trường) sẽ bị ép chia nhỏ theo độ khó. Rank 6: chia 2, Rank 7: chia 5, Rank 8: chia 20, Rank 9: chia 100.
          </p>

          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono mt-6">Suy Diễn Bí Phương</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Sử dụng <strong>Niệm Lực</strong> và Nguyên thạch để tăng % độ hoàn thiện của tàn phương. Độ khó của việc suy diễn phụ thuộc cực kỳ lớn vào Rank của Cổ, chia nhỏ tỷ lệ nhận được: Rank 5 (chia 100), Rank 6 (chia 1.000), tới Rank 9 (chia 10.000.000).
            <br /><br />
            Để có thể suy diễn Tiên Cổ trong thời gian thực tế, đạo hữu bắt buộc phải dựa vào <strong>Cảnh Giới</strong> (Trí Đạo x3, Luyện Đạo x2, Lưu Phái Đích x1) và <strong>Đạo Ngân Trí Đạo</strong> (Gia tốc tư duy). Một Trí Đạo Vô Thượng sở hữu lượng lớn Đạo Ngân có thể suy diễn nhanh gấp hàng vạn lần phàm nhân. Cảnh giới Trí Đạo cũng tăng vọt cơ hội nhận <strong>Linh quang chớp lóe</strong> (lập tức hoàn thiện 100%).
          </p>

          <div className="p-4 rounded-2xl bg-amber-950/20 border border-amber-500/10 text-xs text-amber-300 leading-relaxed mt-4">
            ⚠️ <strong>Hậu quả luyện cổ:</strong> Có 4 mốc hậu quả:
            <ul className="list-disc list-inside mt-2 space-y-1">
              <li><strong>Thành công:</strong> Nhận Cổ trùng, tăng Chân Ý Luyện Đạo theo bậc Cổ (R1: +2đ; R2-3: +5~15đ; R4-5: +40~100đ; Tiên Cổ R6+: +500~50.000đ; không tăng Chân Ý lưu phái khác).</li>
              <li><strong>Thất bại (Safe):</strong> Cảnh giới Luyện Đạo cao cứu vãn, giữ lại nguyên liệu chính, mất 50% nguyên liệu phụ.</li>
              <li><strong>Thất bại (Normal):</strong> Mất 100% nguyên liệu. Cổ Bản Mệnh sẽ chuyển sang trạng thái Hấp Hối thay vì chết.</li>
              <li><strong>Cháy Nổ (Backfire):</strong> Tỉ lệ tột cùng xui xẻo khiến lò nổ, mất 50% HP max và 30% Hồn Phách max.</li>
            </ul>
          </div>

          <div className="p-4 rounded-2xl bg-red-950/20 border border-red-500/10 text-xs text-red-300 leading-relaxed mt-4">
            💤 <strong>Mệt Mỏi & Phản Phệ khi Luyện Cổ:</strong>
            <ul className="list-disc list-inside mt-2 space-y-1">
              <li><strong>Điều kiện:</strong> Chỉ bị ảnh hưởng khi <em>Luyện Tiên Cổ (Rank 6+)</em> hoặc <em>Luyện Cổ Rank &ge; Rank Cảnh Giới bản thân</em>. Luyện Cổ bậc thấp hơn bản thân ("cổ cùi") không làm tổn hao tâm thần.</li>
              <li><strong>Tâm Thần Mệt Mỏi (refine_fatigue):</strong> Giảm <strong>25%</strong> tỉ lệ Luyện Cổ thành công trong 10 phút.</li>
              <li><strong>Luyện Cổ Phản Phệ (refine_backfire):</strong> Xuất hiện khi cố Luyện Cổ trong lúc đang mệt mỏi hoặc bị nổ lò. Giảm <strong>50%</strong> tỉ lệ Luyện Cổ thành công trong 20 phút (tiếp tục luyện sẽ gia hạn 20 phút).</li>
            </ul>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "activities",
    title: "Hoạt động & Dã Ngoại",
    category: "activities",
    keywords: ["hoạt động", "dã ngoại", "khám phá", "mỏ", "phó bản", "mộng cảnh", "explore", "mine", "dungeon", "dream realm"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Game cung cấp các hoạt động phiêu lưu ngoài bản đồ và các phó bản cốt truyện đặc thù nhằm thu thập tài nguyên cải thiện thực lực.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Các phó bản đặc thù</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2">
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-1.5">
                <Compass className="w-4 h-4 text-emerald-400" />
                <span>Truyền Thừa</span>
              </h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Phó bản solo giải đố dành cho phàm nhân. Bạn phải đối mặt với các cạm bẫy, câu đố chữ hoặc yêu cầu sử dụng đúng thuộc tính Cổ trùng nhất định để vượt qua.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2">
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-1.5">
                <Flame className="w-4 h-4 text-purple-400" />
                <span>Mộng Cảnh</span>
              </h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Vượt ải sinh tồn tinh thần. Khi vào, tu vi và đồ của bạn bị khóa, bạn phải dùng chỉ số Hồn Phách để di chuyển. Trả lời sai hoặc thua trận sẽ trực tiếp trừ Hồn Phách. Thành công nhận về <strong>Chân Ý</strong> tăng mạnh Cảnh Giới lưu phái.
              </p>
            </div>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "beasts",
    title: "Linh Thú & Nô Đạo",
    category: "beasts",
    keywords: ["linh thú", "thú", "nô đạo", "hồn thú", "enslavement", "beasts", "soul", "hồn phách"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Nô Đạo là lưu phái sử dụng linh hồn và các cổ trùng khống chế để hiệu lệnh, dẫn dắt đại quân hoang thú chiến đấu thay bản thân mình.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Giới hạn khống chế & Hồn Phách</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Bạn không thể dẫn dắt một quân đội vô hạn. Sức chịu đựng tinh thần của bạn phụ thuộc hoàn toàn vào điểm số <strong>Nội tình Hồn Phách</strong>:
          </p>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-3 text-[var(--text-secondary)]">
            <div className="flex justify-between border-b border-zinc-800 pb-2">
              <span>Bách Nhân Hồn (100)</span>
              <span>Khống chế tối đa cấp độ Bách Thú Vương</span>
            </div>
            <div className="flex justify-between border-b border-zinc-800 pb-2">
              <span>Vạn Nhân Hồn (10,000)</span>
              <span>Dẫn dắt vạn quân thú vương dã ngoại</span>
            </div>
            <div className="flex justify-between">
              <span>Thiên Vạn Nhân Hồn (10M+)</span>
              <span>Gánh vác và điều khiển Hoang Thú Thái Cổ cấp Cổ Tiên</span>
            </div>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "companions",
    title: "Đồng Hành & Dị Nhân",
    category: "companions",
    keywords: ["đồng hành", "dị nhân", "người lông", "người đá", "người tuyết", "hairy men", "rockmen", "variant"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Thế giới ngoài loài người còn tồn tại 10 chủng tộc dị nhân. Khi lên làm Cổ Tiên, bạn có thể chiêu mộ và thả các dị nhân vào Tiên Khiếu để làm việc sản xuất.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
            <h4 className="font-bold text-[var(--text-primary)]">Người Lông</h4>
            <p className="text-[var(--text-muted)] leading-relaxed">
              Thiên phú luyện Cổ bẩm sinh. Nuôi dưỡng Người Lông trong Tiên Khiếu giúp tăng mạnh tốc độ sản xuất và tăng tỉ lệ thành công khi luyện cổ trùng tự động.
            </p>
          </div>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
            <h4 className="font-bold text-[var(--text-primary)]">Người Đá</h4>
            <p className="text-[var(--text-muted)] leading-relaxed">
              Thân hình cứng rắn, ngủ sâu nuôi dưỡng hồn phách. Rất thích hợp làm lao động đào quặng dưới lòng đất Tiên Khiếu, giúp tăng sản lượng khai thác quặng nguyên thạch.
            </p>
          </div>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
            <h4 className="font-bold text-[var(--text-primary)]">Người Tuyết</h4>
            <p className="text-[var(--text-muted)] leading-relaxed">
              Tộc nhân sinh sống tại băng nguyên. Khóc ra Lệ Băng (tiên tài trân quý Băng đạo) dùng để luyện các loại Tiên Cổ hệ Băng Tuyết.
            </p>
          </div>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
            <h4 className="font-bold text-emerald-400">Cương Thi (Tiên Cương)</h4>
            <p className="text-[var(--text-muted)] leading-relaxed">
              Chủng tộc tử thi bất tử thọ nguyên. Cấm tự hồi sinh mệnh/chân nguyên/thần hồn, khóa time_speed về 1.0 và Tiên Khiếu suy thoái theo thời gian thực.
            </p>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "tinh-nang-moi",
    title: "Tính Năng Mới",
    category: "gameplay",
    keywords: ["tính năng mới", "gacha", "cổ từ", "gift code", "mã quà", "thành tựu", "danh hiệu", "new features"],
    content: (
      <div className="space-y-6">
        <div className="p-4 rounded-2xl bg-gradient-to-br from-amber-950/30 to-rose-950/30 border border-amber-500/20 flex gap-3 items-start">
          <Sparkles className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-amber-200">Tổng Quan</h4>
            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              Phiên bản v0.7.0 &amp; v0.8.x giới thiệu các tính năng lớn nhằm tăng trải nghiệm: Hệ thống <strong>Mở đá</strong>, <strong>Mã Quà (Gift Code)</strong>, <strong>Thành Tựu & Danh Hiệu</strong> và <strong>Cân Bằng Xung Đột Đạo Ngân</strong>.
            </p>
          </div>
        </div>

        {/* Cổ Từ */}
        <div className="bg-gradient-to-br from-pink-950/20 to-rose-950/20 border border-pink-500/20 rounded-xl p-4 space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-2xl">🎴</span>
            <h3 className="text-sm font-serif font-bold text-pink-200">Mở đá</h3>
          </div>
          <div className="space-y-2 text-xs text-[var(--text-secondary)] leading-relaxed">
            <p>
              Hệ thống quay thưởng ngẫu nhiên với 3 Pool quay, mỗi pool có tỉ lệ ra phần thưởng riêng:
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li><strong>Pool Tinh Túy:</strong> 100-50.000 Nguyên Thạch, tỉ lệ ra 85%. Giá 1 lần quay: 1.000 Nguyên Thạch. 10 lần: 9.000 Nguyên Thạch.</li>
              <li><strong>Pool Thọ Nguyên:</strong> 1-200 Thọ Nguyên, tỉ lệ ra 85%. Giá 1 lần quay: 5.000 Nguyên Thạch. 10 lần: 45.000 Nguyên Thạch.</li>
              <li><strong>Pool Vận Mệnh:</strong> Tiên Nguyên Thạch, Cổ trùng, Đạo Ấn, tỉ lệ ra 70%. Giá 1 lần quay: 2.000 Nguyên Thạch. 10 lần: 18.000 Nguyên Thạch.</li>
            </ul>
            <p>
              Có thể vào hub "Cổ Từ" ở sảnh chính để mở. Animation quay thưởng với hiệu ứng ánh sáng và thông báo kết quả chi tiết.
            </p>
          </div>
        </div>

        {/* Mã Quà */}
        <div className="bg-gradient-to-br from-emerald-950/20 to-teal-950/20 border border-emerald-500/20 rounded-xl p-4 space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-2xl">🎁</span>
            <h3 className="text-sm font-serif font-bold text-emerald-200">Mã Quà (Gift Code)</h3>
          </div>
          <div className="space-y-2 text-xs text-[var(--text-secondary)] leading-relaxed">
            <p>
              Hệ thống mã quà tặng cho phép Admin tạo các mã code có phần thưởng bên trong. Người chơi nhập mã tại tab <strong>"Phúc Lợi"</strong> trong Không Khiếu.
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li>Mỗi mã có số lượt sử dụng tối đa và giới hạn mỗi người chơi.</li>
              <li>Phần thưởng có thể là: Nguyên Thạch, Tiên Nguyên Thạch, Nhân Khí, Vật phẩm, Cổ trùng, Thọ Nguyên, Phá Vách, Đạo Ấn.</li>
              <li>Admin tạo và quản lý mã tại mục "Mã Quà" trong Admin Panel.</li>
              <li>UI tạo mã mới có picker chọn vật phẩm trực quan thay vì nhập ID thủ công.</li>
            </ul>
          </div>
        </div>

        {/* Thành Tựu */}
        <div className="bg-gradient-to-br from-amber-950/20 to-yellow-950/20 border border-amber-500/20 rounded-xl p-4 space-y-3">
          <div className="flex items-center gap-2">
            <span className="text-2xl">🏆</span>
            <h3 className="text-sm font-serif font-bold text-amber-200">Thành Tựu & Danh Hiệu</h3>
          </div>
          <div className="space-y-2 text-xs text-[var(--text-secondary)] leading-relaxed">
            <p>
              Hệ thống thành tựu với 23 mốc thành tựu, mỗi thành tựu có điều kiện mở khóa và phần thưởng riêng. Quản lý tại tab <strong>"Thành Tựu"</strong> trong Không Khiếu.
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li><strong>Thành tựu chiến đấu:</strong> Tân Thủ Nhập Môn, các mốc Nhị Chuyển đến Cửu Chuyển.</li>
              <li><strong>Thành tựu độ kiếp:</strong> Vượt 1, 5, 10 kiếp nạn.</li>
              <li><strong>Thành tựu Cổ trùng:</strong> Sở hữu 10, 25, 50 loại Cổ khác nhau.</li>
              <li><strong>Thành tựu khám phá:</strong> Khám phá 100, 500, 1.000 ô địa đồ.</li>
              <li><strong>Thành tựu kinh tế:</strong> Tích lũy 1.000.000 Nguyên Thạch.</li>
              <li><strong>Thành tựu tu luyện:</strong> Nhân Khí, Hồn Phách, Phá Vách.</li>
            </ul>
            <p>
              Một số thành tựu cấp cao mở khóa <strong>Khung Frame</strong> và <strong>Danh Hiệu</strong> hiển thị cạnh tên trong khung chat.
            </p>
          </div>
        </div>

        {/* v0.7.2 - Combat Fixes + Gift Codes */}
        <div className="bg-gradient-to-br from-amber-950/20 to-rose-950/20 border border-amber-500/20 rounded-xl p-4 space-y-2">
          <div className="flex items-center gap-2">
            <span className="text-lg shrink-0">🛠️</span>
            <h3 className="text-sm font-serif font-bold text-amber-200">v0.7.2 — Sửa lỗi Chiến đấu & Gift Code</h3>
          </div>
          <ul className="list-disc list-inside space-y-1 text-xs text-[var(--text-secondary)] leading-relaxed">
            <li><strong>Sửa Speed:</strong> Tốc độ từ Growth và Passive giờ ảnh hưởng thứ tự đánh trong combat.</li>
            <li><strong>Sửa MaxHP:</strong> Sinh Lực tối đa không còn bị ghi đè cứng ở 100.</li>
            <li><strong>Growth cho legacy Cổ:</strong> Worker tự tạo EquippedAt cho Cổ đã equip trước khi feature growth ra mắt.</li>
            <li><strong>Auto-Combat loadout:</strong> Chỉ hiện Cổ có Active Skill. Passive + Growth auto-apply từ toàn bộ Cổ đang trang bị.</li>
            <li><strong>Gift Code mới:</strong> <span className="font-mono text-emerald-400">CHODAUGIA</span>, <span className="font-mono text-emerald-400">PHUONGNGUYEN</span>, <span className="font-mono text-amber-400">BACHNGUNGBANG</span>. Nhập tại tab Phúc Lợi trong Không Khiếu.</li>
          </ul>
        </div>
      </div>
    )
  }
];
