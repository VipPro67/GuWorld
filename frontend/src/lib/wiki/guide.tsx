import React from "react";
import { HelpCircle, Clock } from "lucide-react";
import { WikiPage } from "./types";

export const guidePages: WikiPage[] = [
  {
    id: "guide",
    title: "Hướng Dẫn Tân Thủ",
    category: "gameplay",
    keywords: ["hướng dẫn", "tân thủ", "new player", "guide", "bắt đầu", "hoạt động", "đạo ngân", "tiên khiếu", "sát chiêu", "kiếp nạn", "lộ trình", "hàng ngày", "timeline"],
    content: (
      <div className="space-y-6">
        <div className="p-4 rounded-2xl bg-indigo-950/30 border border-indigo-500/20 flex gap-3 items-start">
          <HelpCircle className="w-5 h-5 text-indigo-400 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-indigo-200">Chào mừng Đạo Hữu gia nhập GuWorld</h4>
            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              GuWorld là thế giới Cổ sư mô phỏng chân thực hành trình nghịch thiên cải mệnh theo nguyên tác tiểu thuyết <strong>Cổ Chân Nhân</strong>. Dưới đây là hướng dẫn chi tiết về các thuật ngữ cốt lõi và lộ trình hoạt động theo thời gian để đạo hữu dễ dàng bước lên con đường tu tiên.
            </p>
          </div>
        </div>

        {/* Terminology Glossary */}
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Thuật Ngữ Cốt Lõi Cho Cổ Sư</h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
              <h4 className="font-bold text-indigo-300">Không Khiếu & Tiên Khiếu (Aperture)</h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                • <strong>Không Khiếu:</strong> Nơi chứa Chân Nguyên của Cổ sư phàm nhân (Nhất Chuyển đến Ngũ Chuyển).<br />
                • <strong>Tiên Khiếu:</strong> Thế giới thu nhỏ độc lập bên trong cơ thể Cổ Tiên (Lục Chuyển trở lên). Tiên Khiếu tự động sản sinh <strong>Tiên Nguyên</strong> và có gia tốc thời gian riêng biệt. Tiên Khiếu được chia thành 4 phẩm cấp: <strong>Thấp, Trung, Cao, Cực Hạn</strong> phụ thuộc vào lượng Nhân Khí khi thăng tiên.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
              <h4 className="font-bold text-indigo-300">Đạo Ngân & Xung Đột Quy Tắc (Dao Marks & Conflict)</h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Đạo Ngân của lưu phái đang dùng làm tăng uy lực chiêu thức cùng hệ. Đạo Ngân ngoại hệ có thể gây <strong>Xung Đột Đạo Ngân</strong>; mức phạt được tính theo từng hành động và Chân Ý của chính lưu phái đó. Đánh thường mặc định tính là Strength (Lực Đạo), không tự đổi hệ theo Cổ đang trang bị. Xem trang <strong>Xung Đột Đạo Ngân &amp; Phản Phệ</strong> để biết ví dụ và cách tránh bạo thể.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
              <h4 className="font-bold text-indigo-300">Cảnh Giới Lưu Phái (Attainment / Chân Ý)</h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Độ hiểu biết về quy luật thế giới. Chia làm 5 cảnh giới: <strong>Phàm Thai (Ordinary)</strong> ➔ <strong>Đại Sư (Master)</strong> ➔ <strong>Tông Sư (Grandmaster)</strong> ➔ <strong>Đại Tông Sư (Great Grandmaster)</strong> ➔ <strong>Vô Thượng Đại Tông Sư (Supreme Grandmaster)</strong>. Chân Ý của một lưu phái điều hòa xung đột cho hành động dùng đúng lưu phái đó: từ 10% ở Đại Sư đến 100% ở Vô Thượng. Nó không tự bảo vệ các lưu phái khác.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
              <h4 className="font-bold text-indigo-300">Bản Mệnh Cổ (Vital Gu)</h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Loại Cổ trùng liên kết trực tiếp với linh hồn và sinh mệnh của Cổ sư. Bản Mệnh Cổ sẽ không bao giờ bị mất hoặc rơi ra khi đạo hữu thất bại trong chiến đấu dã ngoại.
              </p>
            </div>
          </div>
        </div>

        {/* Timeline Guide */}
        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Lộ Trình Tu Tiên Theo Thời Gian</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Hành trình của một Cổ sư diễn ra theo dòng thời gian thực. Dưới đây là lộ trình gợi ý từ lúc bước vào Cổ giới cho đến lúc thăng tiên và đối đầu kiếp nạn.
          </p>

          {/* Phase 1 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">1</span>
              Ngay Lập Tức: Tạo Nhân Vật & Thức Tỉnh Linh Căn
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                Bước vào Cổ giới, đạo hữu sẽ đặt tên Cổ sư của mình và chọn <strong>Chủng tộc</strong>, <strong>Giới tính</strong>. Hệ thống sẽ tự động thức tỉnh <strong>Linh Căn</strong> ngẫu nhiên với một trong bốn phẩm cấp: <strong>Giáp, Ất, Bính, Đinh</strong>.
              </p>
              <p>
                Linh Căn quyết định hướng phát triển ban đầu của bạn: tư chất Giáp/Ất giúp trùng kích vách khiếu nhanh hơn rõ rệt. Sử dụng các Cổ trùng có cùng hệ thuộc tính với Linh Căn sẽ giúp gia tăng đáng kể hiệu năng chiến đấu.
              </p>
              <div className="p-3 rounded-lg bg-indigo-950/20 border border-indigo-500/10 text-indigo-300">
                ⏱️ <strong>Gợi ý:</strong> Ngay khi tạo xong, hãy hoàn tất luồng onboarding, nhận Cổ trùng ban đầu và bắt đầu trùng kích vách khiếu — Chân Nguyên khởi đầu rất thấp nên mỗi giờ hồi phục đều đáng giá.
              </div>
            </div>
          </div>

          {/* Phase 2 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">2</span>
              Những Ngày Đầu: Trùng Kích & Khai Thác Tài Nguyên
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                Tu vi phàm nhân chia làm 5 Cảnh giới (Nhất Chuyển đến Ngũ Chuyển), mỗi Cảnh giới có 4 cấp bậc nhỏ (Sơ Giai, Trung Giai, Cao Giai, Đỉnh Phong). Mỗi lần trùng kích làm tăng <strong>+5 áp lực vách khiếu</strong> — hãy nhớ theo dõi thanh áp lực.
              </p>
              <ul className="list-disc list-inside pl-2 space-y-1">
                <li><strong>Khai Thác Mỏ (Mining):</strong> Đào khoáng và Nguyên Thạch dã ngoại. Đôi khi bạn sẽ chạm trán <strong>Quái Mỏ</strong> hung hãn bảo vệ quặng, cần tiêu diệt chúng để tiếp tục khai thác. Mỏ được tái lập mỗi ngày lúc 00:00.</li>
                <li><strong>Thám Hiểm (Exploration):</strong> Di chuyển qua các ô đất sương mù trên bản đồ để nhận khoáng sản quý, thảo dược và gặp gỡ các kỳ ngộ nhận thưởng. Mỗi ngày có <strong>tối đa 20 lượt thám hiểm</strong>.</li>
              </ul>
              <div className="p-3 rounded-lg bg-amber-950/20 border border-amber-500/20 text-amber-300">
                ⏱️ <strong>Quy luật thời gian:</strong> Áp lực vách khiếu giảm <strong>phân nửa mỗi 2 giờ đời thật</strong>. Chân Nguyên hồi tự động theo tư chất (Giáp 8/giờ, Ất 6/giờ, Bính 4/giờ, Đinh 2/giờ) — đừng để Chân Nguyên bị lãng phí vì thanh đầy.
              </div>
            </div>
          </div>

          {/* Phase 3 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">3</span>
              Mỗi Ngày: Cho Cổ Trùng Ăn & Trang Bị
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                Đạo hữu có thể trang bị tối đa 5 Cổ trùng vào Không Khiếu. Cổ trùng chia làm 3 dạng chính:
              </p>
              <ul className="list-disc list-inside pl-2 space-y-1">
                <li><strong>Cổ Chủ Động:</strong> Cung cấp các chiêu thức chiến đấu để kéo vào phím tắt kích hoạt thủ công.</li>
                <li><strong>Cổ Bị Động & Tăng Trưởng:</strong> Tự động cộng các chỉ số thuộc tính vĩnh viễn (Sinh Lực, Tấn công, Tốc độ) khi được trang bị.</li>
              </ul>
              <div className="p-3 rounded-lg bg-rose-950/20 border border-rose-500/10 text-rose-300">
                ⚠️ <strong>Quy luật sinh tồn:</strong> Cổ trùng tiêu hao <strong>Độ No (Hunger)</strong> liên tục <strong>mỗi 3 giờ</strong>. Hãy tích lũy đúng loại thức ăn (thịt thú, khoáng vật, ngọc lộ...) và cho chúng ăn kịp thời. Cổ trùng đói về 0 độ no sẽ <strong>chết vĩnh viễn và biến mất khỏi hòm đồ</strong>! Quy luật này áp dụng cả với Cổ nằm trong <strong>Bảo khố Tông Môn</strong> — Cổ trong kho vẫn hao Độ No và có thể chết trong kho.
              </div>
            </div>
          </div>

          {/* Phase 4 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">4</span>
              Sau Vài Ngày: Luyện Cổ & Lắp Ráp Sát Chiêu
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                • <strong>Luyện Cổ:</strong> Sử dụng các công thức (Recipe) tại lò luyện để dung hợp các loại Cổ trùng bậc thấp và nguyên liệu phụ để tạo ra Cổ trùng cấp cao hơn. Quá trình này có tỷ lệ thất bại tự nhiên làm mất nguyên liệu phụ.
              </p>
              <p>
                • <strong>Sát Chiêu (Killer Moves):</strong> Lắp ráp kỹ năng tối thượng bằng cách kết hợp <strong>1 Cổ Hạch Tâm (Core Gu)</strong> quyết định lưu phái/sát thương gốc cùng tối đa <strong>8 Cổ Phụ Trợ (Support Gu)</strong> để bổ sung các hiệu ứng khống chế (choáng, đóng băng, câm lặng), tăng phạm vi tấn công và giảm hao Chân Nguyên.
              </p>
              <div className="p-3 rounded-lg bg-indigo-950/20 border border-indigo-500/10 text-indigo-300">
                ⏱️ <strong>Gợi ý:</strong> Mỗi sát chiêu mới tạo ra nhận <strong>+5 Nhân Khí</strong> — hãy ưu tiên tạo vài sát chiêu căn bản sớm để vừa có chiêu thức dùng khi chiến đấu, vừa tích Nhân Khí phục vụ Thăng Tiên về sau.
              </div>
            </div>
          </div>

          {/* Phase 5 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">5</span>
              Hằng Ngày: Ải Cốt Truyện & Đấu Trường
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                • <strong>Ải Cốt Truyện:</strong> Sử dụng kỹ năng Cổ trùng và Sát Chiêu vượt các ải quái vật để nhận tu vi, Nguyên thạch và Cổ trùng sơ cấp. Mỗi trận chiến đều tích lũy Nhân Khí.
              </p>
              <p>
                • <strong>Đấu Trường:</strong> Khiêu chiến đội hình Cổ sư của người chơi khác để tích lũy <strong>Chiến Công</strong>. Chiến công dùng để đổi các bảo vật và Cổ trùng độc quyền tại Cửa Hàng Đấu Trường.
              </p>
              <div className="p-3 rounded-lg bg-amber-950/20 border border-amber-500/20 text-amber-300">
                ⏱️ <strong>Giới hạn hàng ngày:</strong> Mỗi ngày đạo hữu được khiêu chiến Võ Đài tối đa <strong>10 trận</strong>; hạn mức đấu trường được làm mới theo ngày. Thắng nhận <strong>50 Xu Đấu Trường</strong>, thua nhận <strong>10 Xu</strong>. Dùng hết lượt để tối đa hóa Chiến Công và Nhân Khí tích lũy.
              </div>
            </div>
          </div>

          {/* Phase 6 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">6</span>
              Định Kỳ: Giao Thương Bảo Hoàng Thiên & Tông Môn
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                • <strong>Bảo Hoàng Thiên:</strong> Chợ giao dịch tự do liên server. Đạo hữu có thể bày bán Cổ trùng, tiên tài, nguyên liệu để đổi lấy Nguyên Thạch hoặc Tiên Nguyên Thạch từ người chơi khác. Mỗi lần niêm yết hoặc giao dịch thành công đều nhận Nhân Khí.
              </p>
              <p>
                • <strong>Tông Môn (Bang Hội):</strong> Tham gia tông môn, thực hiện quyên góp tài sản để nhận Điểm Cống Hiến, cùng bang hội chiếm mỏ tài nguyên dã ngoại và dùng Điểm Cống Hiến để đổi đồ trong Bảo Khố Tông Môn.
              </p>
              <div className="p-3 rounded-lg bg-indigo-950/20 border border-indigo-500/10 text-indigo-300">
                ⏱️ <strong>Khảo hạch tông môn:</strong> <strong>Tiểu Khảo</strong> mỗi 3 ngày (lên Nội Môn), <strong>Trung Khảo</strong> mỗi 8 ngày (đệ tử Tinh Anh), <strong>Đại Khảo</strong> mỗi 15 ngày (đệ tử Chân Truyền). Hãy canh ngày tham gia đủ kỳ khảo hạch để không lỡ thăng cấp trong tông môn.
              </div>
            </div>
          </div>

          {/* Phase 7 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">7</span>
              Trung Kỳ: Dưỡng Hồn Phách & Tích Nhân Khí
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                • <strong>Tráng Hồn:</strong> Dùng Guts Gu để tăng <strong>+10 Hồn Phách</strong> (Soul HP & Max Soul), tăng khả năng khống chế hoang thú và đạo pháp.
              </p>
              <p>
                • <strong>Luyện Hồn:</strong> Khi đạt giới hạn tầng Hồn hiện tại, hãy đến <strong>Cốc Lạc Phách</strong> để luyện hồn bứt phá lên tầng mới (Nhân Hồn ➔ Bách Nhân Hồn ➔ Thiên Nhân Hồn...). Sau luyện hồn, Hồn Phách sẽ suy yếu mạnh và cần thời gian dưỡng hồi.
              </p>
              <div className="p-3 rounded-lg bg-amber-950/20 border border-amber-500/20 text-amber-300">
                ⏱️ <strong>Quy luật thời gian:</strong> Hồn Phách hồi phục theo thời gian thực (~10% Max Soul mỗi giờ, nhân với gia tốc thời gian). Hãy sắp xếp luyện hồn vào lúc có thời gian nghỉ dài để linh hồn kịp hồi phục.
              </div>
            </div>
          </div>

          {/* Phase 8 */}
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-300 flex items-center gap-2">
              <span className="flex items-center justify-center w-5 h-5 rounded-full bg-indigo-950 text-indigo-400 text-xs font-mono border border-indigo-500/20">8</span>
              Hậu Kỳ: Thăng Tiên & Đối Phó Kiếp Nạn
            </h3>
            <div className="text-xs text-[var(--text-secondary)] leading-relaxed space-y-2">
              <p>
                Khi đạt <strong>Ngũ Chuyển Đỉnh Phong</strong>, đạo hữu có thể tiến hành thăng tiên để đột phá thành <strong>Cổ Tiên (Lục Chuyển trở lên)</strong>.
              </p>
              <p>
                Yêu cầu tối thiểu <strong>100.000 Nhân Khí</strong> để tiến hành Thăng Tiên. Tỉ lệ thành công phụ thuộc vào <strong>Tư chất</strong> và <strong>Nhân Khí tích lũy</strong> (giới hạn tối đa 500.000 Nhân Khí, tổng tỉ lệ cap tại <strong>90%</strong>):
              </p>
              <div className="bg-zinc-950/80 border border-[var(--border)] rounded-lg p-3 my-2 space-y-1 font-mono text-[11px]">
                <div className="text-amber-400 font-bold">📊 Công thức: Tỉ lệ = Xác suất gốc Tư chất + (Nhân Khí ÷ 100.000) × 20%</div>
                <div className="text-[var(--text-secondary)]">• <strong>Tư chất Giáp:</strong> 20% gốc</div>
                <div className="text-[var(--text-secondary)]">• <strong>Tư chất Ất / Bính / Đinh:</strong> 5% gốc</div>
                <div className="text-emerald-400 border-t border-zinc-800 pt-1 mt-1">
                  💡 <strong>Ví dụ cụ thể:</strong><br />
                  • <strong>Tư chất Giáp (100k Nhân Khí):</strong> 20% + 20% = <strong>40% tỉ lệ thành công</strong>.<br />
                  • <strong>Tư chất Giáp (350k Nhân Khí):</strong> 20% + 70% = <strong>90% tỉ lệ thành công</strong> (Đạt trần 90%).<br />
                  • <strong>Tư chất Ất/Bính/Đinh (100k Nhân Khí):</strong> 5% + 20% = <strong>25% tỉ lệ thành công</strong>.<br />
                  • <strong>Tư chất Ất/Bính/Đinh (425k Nhân Khí):</strong> 5% + 85% = <strong>90% tỉ lệ thành công</strong> (Đạt trần 90%).
                </div>
              </div>
              <p className="text-rose-400/90 italic">
                ⚠️ Nếu thăng tiên thất bại, không khiếu vỡ nát, Nhân Khí về 0 và tu vi phế bỏ hoàn toàn về Nhất Chuyển Sơ Giai!
              </p>
              <p>
                Sau khi thăng tiên, đạo hữu sẽ sở hữu <strong>Tiên Khiếu</strong> riêng biệt và phải đối đầu với các <strong>Kiếp Nạn</strong> (Địa Tai, Thiên Kiếp...) định kỳ. Kiếp đầu tiên giáng lâm vào <strong>năm nội bộ thứ 10</strong> — tính ra khoảng <strong>2–10 giờ đời thật</strong> tùy phẩm chất Tiên Khiếu (1 năm nội bộ = 10 giờ đời thật ÷ hệ số gia tốc time_rate 10–55×). Vượt qua kiếp nạn thành công sẽ nhận được lượng lớn Đạo Ngân để gia tăng sức mạnh vượt bậc.
              </p>
            </div>
          </div>

          {/* Daily rhythm summary */}
          <div className="p-4 rounded-2xl bg-emerald-950/20 border border-emerald-500/20 space-y-3">
            <h3 className="text-sm font-serif font-bold text-emerald-300 flex items-center gap-2">
              <Clock className="w-4 h-4" /> Nhịp Điệu Sinh Hoạt Của Một Cổ Sư
            </h3>
            <div className="overflow-hidden rounded-2xl border border-[var(--border)]">
              <table className="w-full text-left text-xs font-mono">
                <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                  <tr>
                    <th className="p-3">Mốc thời gian</th>
                    <th className="p-3">Việc cần làm</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                  <tr>
                    <td className="p-3 font-bold text-[var(--text-primary)]">Liên tục (mỗi giờ)</td>
                    <td className="p-3">Chân Nguyên tự hồi theo tư chất (8/6/4/2 mỗi giờ). Đừng để thanh đầy lãng phí.</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-[var(--text-primary)]">Mỗi 2 giờ</td>
                    <td className="p-3">Áp lực vách khiếu tự giảm phân nửa — thời điểm lý tưởng để trùng kích tiếp.</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-[var(--text-primary)]">Mỗi 3 giờ</td>
                    <td className="p-3">Cổ trùng giảm Độ No — kiểm tra và cho ăn kịp thời tránh chết Cổ.</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-[var(--text-primary)]">Mỗi ngày (00:00)</td>
                    <td className="p-3">Làm mới lượt thám hiểm (20 lượt), khiêu chiến Đấu Trường (10 trận), tái lập mỏ khai thác.</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-[var(--text-primary)]">17:00 Thứ 7 hàng tuần</td>
                    <td className="p-3"><strong>Đấu Trường Sinh Tử:</strong> Đại chiến tổ đội 6v6 loại trực tiếp 16 đội. Mở đăng ký từ 16:00 - 17:00 Thứ 7, khai chiến lúc 17:00 (5h chiều Thứ 7).</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-[var(--text-primary)]">Vài giờ sau thăng tiên (tùy phẩm chất Tiên Khiếu)</td>
                    <td className="p-3">Kiếp nạn đầu tiên giáng lâm — chuẩn bị phòng thủ Tiên Khiếu và chiến lực.</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

        </div>
      </div>
    ),
  },
];
