import React from "react";
import { Coins, Swords, AlertTriangle, Flame, Clock } from "lucide-react";
import { WikiPage } from "./types";

export const gameplayPages: WikiPage[] = [
  {
    id: "gameplay-ui-navigation",
    title: "Bắt Đầu & Điều Hướng Game",
    category: "gameplay",
    keywords: ["cách chơi", "giao diện", "đại sảnh", "bắt đầu", "tân thủ"],
    content: (
      <div className="space-y-5 text-xs text-[var(--text-secondary)] leading-relaxed">
        <div className="p-4 rounded-2xl bg-emerald-950/20 border border-emerald-500/20">
          <h3 className="text-sm font-bold text-emerald-300 mb-2">Bắt đầu trong 5 bước</h3>
          <ol className="list-decimal list-inside space-y-2">
            <li>Hoàn tất tạo nhân vật, chọn Đại Vực và bộ Cổ ban đầu.</li>
            <li>Mở <strong>Nhân Vật</strong> để xem HP, Chân Nguyên, tư chất và áp lực vách khiếu.</li>
            <li>Vào <strong>Túi Cổ</strong>, trang bị Cổ, chọn Bản Mệnh và kiểm tra Độ No.</li>
            <li>Làm Nhiệm Vụ, khai thác Mỏ và khám phá Ngũ Vực để tích tài nguyên.</li>
            <li>Luyện Cổ, thử đội hình trong Phòng Tập rồi mới tham gia Võ Đài/PvP.</li>
          </ol>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]">
            <h3 className="font-bold text-indigo-300 mb-2">Quản lý</h3>
            <ul className="list-disc list-inside space-y-1"><li>Nhân Vật: tu luyện, hồ sơ, bộ Cổ, phúc lợi và thành tựu.</li><li>Túi Cổ: cho ăn, trang bị, ngủ, phong ấn, dùng hoặc loại bỏ.</li><li>Nô Thú: cho ăn, xếp đội, ngủ, thả và gộp linh thú.</li><li>Tiên Khiếu mở từ Lục Chuyển.</li></ul>
          </div>
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]">
            <h3 className="font-bold text-amber-300 mb-2">Hoạt động</h3>
            <ul className="list-disc list-inside space-y-1"><li>Ngũ Vực: săn quái/Cổ hoang, nhặt tài nguyên và gặp người chơi.</li><li>Thương Đội/Bảo Hoàng Thiên: shop NPC, chợ và đấu giá.</li><li>Tông Môn: nhiệm vụ, bảo khố, cống hiến và lãnh địa.</li><li>Lịch sử/Ký Sự: kiểm tra trận đấu và biến động tài sản.</li></ul>
          </div>
        </div>
        <p className="p-3 rounded-xl bg-amber-950/10 border border-amber-500/20 text-amber-200"><strong>Lưu ý:</strong> ô Sát Chiêu hiện có thể bị khóa theo bản build. Hãy dựa vào trạng thái trên đại sảnh.</p>
      </div>
    ),
    cards: [
      { title: "Sinh tồn ngoài Ngũ Vực", targetId: "gameplay-world-survival", description: "Di chuyển, chiến đấu và nhặt hài cốt" },
      { title: "Truyền Tống Trận & Trận Đạo", targetId: "gameplay-world-teleport", description: "Bố trí trận pháp, kết nối không gian và giảm giá Trận Đạo" },
      { title: "Chu kỳ hằng ngày", targetId: "gameplay-daily-loop", description: "Checklist phát triển nhân vật" },
      { title: "Tiên Khiếu & Dị Nhân", targetId: "gameplay-aperture-society", description: "Kinh tế dành cho Cổ Tiên" },
      { title: "Trinh Sát & Phản Trinh Sát", targetId: "gameplay-scouting-divination", description: "Suy diễn thiên cơ, ấn ký và chém đứt manh mối" },
    ],
  },
  {
    id: "gameplay-world-teleport",
    title: "Truyền Tống Trận & Trận Đạo",
    category: "activities",
    subCategory: "gameplay-ui-navigation",
    keywords: ["truyền tống", "trận pháp", "trận đạo", "teleport", "vi trận", "tiên trận", "danh bạ", "két sắt", "bố trận"],
    content: (
      <div className="space-y-4 text-xs text-[var(--text-secondary)] leading-relaxed">
        <section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]">
          <h3 className="font-bold text-amber-300 mb-2">Tổng Quan & Phẩm Cấp Trận Pháp</h3>
          <p className="mb-2">
            Cổ Sư có thể bố trí các điểm Truyền Tống Trận cố định trên bản đồ Thế Giới Ngũ Vực để du hành nhanh giữa các khu vực hoặc xuyên Đại Vực.
          </p>
          <ul className="list-disc list-inside space-y-1">
            <li><strong>Vi Trận (Tier 1):</strong> Tầm 1,000 ô (nội bộ tiểu vực). Cần Rank 2. Gốc: 3,000 Nguyên Thạch, 20 Tinh Thể Vỡ Không Gian. Tỉ lệ gốc: 70%.</li>
            <li><strong>Đại Vực Trận (Tier 2):</strong> Tầm 10,000 ô (toàn bộ Đại Vực). Cần Rank 4, Trận Đạo tối thiểu <strong>Đại Sư</strong>. Gốc: 15,000 Nguyên Thạch, 10 Đá Vực Không Gian Vô Tận, 20 Tinh Thể Vỡ Không Gian. Tỉ lệ gốc: 60%.</li>
            <li><strong>Tiên Trận (Tier 3):</strong> Tầm Vô Hạn (Xuyên Ngũ Vực). Cần Rank 6 Tiên Nhân, Trận Đạo tối thiểu <strong>Tông Sư</strong>. Gốc: 100,000 Nguyên Thạch, 50 Tiên Nguyên Thạch, 50 Tinh Thể Vỡ Không Gian, 20 Đá Vực Không Gian Vô Tận, 10 Hư Không Thạch. Tỉ lệ gốc: 45%.</li>
          </ul>
        </section>

        <section className="p-4 rounded-xl bg-blue-950/20 border border-blue-500/20">
          <h3 className="font-bold text-blue-300 mb-2">Ảnh Hưởng Của Cảnh Giới Trận Đạo</h3>
          <ul className="list-disc list-inside space-y-1">
            <li><strong>Giảm Giá Chi Phí:</strong> Nhập Môn 0%, Đại Sư (100 pts) <strong>-15%</strong>, Tông Sư (1,000 pts) <strong>-30%</strong>, Đại Tông Sư (10,000 pts) <strong>-45%</strong>, Vô Thượng (100,000 pts) <strong>-60%</strong>.</li>
            <li><strong>Tỉ Lệ Thành Công:</strong> Tăng thêm 15% mỗi bậc Trận Đạo, cộng thêm từ Tu Vi (Rank vượt chuẩn +3%/cấp) và Đạo Ngân Trận Đạo (+1% mỗi 500 Đạo Ngân).</li>
            <li><strong>Sụp Đổ Trận Văn (Thất Bại):</strong> Tiêu hao 50% nguyên thạch/tiên thạch và vật liệu. Cổ trùng <strong>KHÔNG</strong> bị tổn hại. Nhận 50% điểm Chân Ý Trận Đạo.</li>
          </ul>
        </section>

        <section className="p-4 rounded-xl bg-emerald-950/20 border border-emerald-500/20">
          <h3 className="font-bold text-emerald-300 mb-2">Quyền Hạn & Bảo Mật Danh Bạ Không Gian</h3>
          <ul className="list-disc list-inside space-y-1">
            <li><strong>Công Cộng (PUBLIC):</strong> Hiển thị trong tab Tất Cả, mở cho mọi Cổ Sư. Chủ trận có thể bật thu phí dịch chuyển.</li>
            <li><strong>Tông Môn (SECT):</strong> Chỉ thành viên cùng môn phái mới tra cứu được trong tab Tông Môn và sử dụng. Không hiển thị cho người ngoài để bảo mật tọa độ căn cứ.</li>
            <li><strong>Riêng Tư (PERSONAL):</strong> Chỉ chủ sở hữu mới thấy trong tab Của Ta và sử dụng. Tọa độ được ẩn hoàn toàn trước toàn bộ người chơi khác.</li>
          </ul>
        </section>

        <section className="p-4 rounded-xl bg-purple-950/20 border border-purple-500/20">
          <h3 className="font-bold text-purple-300 mb-2">Vận Hành, Thu Phí & Bảo Trì</h3>
          <ul className="list-disc list-inside space-y-1">
            <li><strong>Hao mòn độ bền:</strong> Mỗi lượt truyền tống tiêu hao 1 độ bền. Hết độ bền cần sửa chữa để tiếp tục sử dụng.</li>
            <li><strong>Sửa Chữa & Bạo Kích:</strong> Giảm giá sửa tối đa 50% theo Trận Đạo. Tỉ lệ Bạo Kích Sửa Chữa (+25% độ bền và cơ hội tăng Max Durability).</li>
            <li><strong>Két Sắt Doanh Thu:</strong> Thu phí dịch chuyển từ người khác, rút về túi đồ bất cứ lúc nào.</li>
            <li><strong>Thu Hồi:</strong> Phá dỡ trận pháp, hoàn trả 50% nguyên liệu (theo % độ bền) và rút toàn bộ số dư két sắt.</li>
          </ul>
        </section>
      </div>
    ),
  },
  {
    id: "gameplay-world-survival",
    title: "Ngũ Vực: Di Chuyển & Sinh Tồn",
    category: "activities",
    subCategory: "gameplay-ui-navigation",
    keywords: ["ngũ vực", "bản đồ", "pvp", "hài cốt", "dịch chuyển"],
    content: (
      <div className="space-y-4 text-xs text-[var(--text-secondary)] leading-relaxed">
        <section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]"><h3 className="font-bold text-sky-300 mb-2">Điều khiển bản đồ</h3><ul className="list-disc list-inside space-y-1"><li>Chuyển giữa click để đi và kéo bản đồ; nút định vị đưa camera về nhân vật.</li><li>Bật/tắt nhãn Quái & Cổ, Boss, HUB, Khu Vực và Ngũ Vực.</li><li>Click đích mới để đổi hướng; click ô đang đứng để dừng.</li><li>Dịch chuyển tức thời sử dụng Cổ trùng có khả năng Dịch Chuyển / Tạo Cổng (như Định Tiên Du, Môn Vực Cổ,...) đang trang bị và ở trạng thái ACTIVE.</li><li>Bố trí và sử dụng <strong>Truyền Tống Trận</strong> để di chuyển nhanh chóng giữa các khu vực hoặc xuyên suốt Ngũ Vực.</li></ul></section>
        <section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]"><h3 className="font-bold text-rose-300 mb-2">Gặp mục tiêu</h3><p>Chỉ bước đúng ô quái/Boss mới mở chiến đấu. Chọn người chơi để xem hồ sơ, mời tổ đội hoặc tiếp cận. Phạm vi PvP cơ bản là 2 ô và có thể tăng bằng chỉ số tầm đánh/tầm chiến đấu; không thể tấn công trong vùng an toàn.</p></section>
        <section className="p-4 rounded-xl bg-rose-950/15 border border-rose-500/20"><h3 className="font-bold text-rose-300 mb-2">Tử vong & hài cốt</h3><p>Thua trận có thể làm rơi thạch, nguyên liệu, Bí phương và Cổ không trang bị/không phải Bản Mệnh. Đồ nằm trên ô bản đồ. Nếu có người hạ gục, thời gian bảo hộ ưu tiên người đó; chiến lợi phẩm PvE không có người hạ gục không được khóa riêng. Luôn xem trạng thái hiển thị trước khi nhặt.</p></section>
      </div>
    ),
  },
  {
    id: "gameplay-daily-loop",
    title: "Chu Kỳ Chơi Hằng Ngày",
    category: "activities",
    subCategory: "gameplay-ui-navigation",
    keywords: ["hằng ngày", "nhiệm vụ", "võ đài", "mỏ", "thành tựu"],
    content: (
      <div className="space-y-4 text-xs text-[var(--text-secondary)] leading-relaxed">
        <section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]"><h3 className="font-bold text-emerald-300 mb-2">Mỗi lần đăng nhập</h3><ol className="list-decimal list-inside space-y-1"><li>Đọc thông báo; kiểm tra HP, Chân Nguyên và Hồn Phách.</li><li>Kiểm tra Độ No của Cổ/linh thú và cho ăn trước khi về 0.</li><li>Nhận thưởng nhiệm vụ, thành tựu, mỏ và node Tiên Khiếu.</li><li>Luyện Cổ, cập nhật bộ chiến đấu và thử trong Phòng Tập.</li></ol></section>
        <section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]"><h3 className="font-bold text-amber-300 mb-2">Theo dõi tiến độ</h3><ul className="list-disc list-inside space-y-1"><li>Lịch Sử Chiến Đấu có kết quả và replay log.</li><li>Ký Sự Tu Hành cho tìm/lọc hành động và biến động tài sản.</li><li>Thành tựu cho nhận thưởng và trang bị danh hiệu/khung.</li><li>BXH gồm Tu Vi, Nguyên Thạch, Cổ mạnh nhất, Tiên Nguyên và Sát Khí.</li></ul></section>
      </div>
    ),
  },
  {
    id: "gameplay-aperture-society",
    title: "Tiên Khiếu & Xã Hội Dị Nhân",
    category: "companions",
    subCategory: "gameplay-ui-navigation",
    keywords: ["tiên khiếu", "dị nhân", "node", "lao động", "khu định cư"],
    content: (
      <div className="space-y-4 text-xs text-[var(--text-secondary)] leading-relaxed">
        <p className="p-4 rounded-xl bg-indigo-950/20 border border-indigo-500/20"><strong>Vòng lặp:</strong> đặt node → phân công dân số → thu sản lượng → sửa/nâng cấp → phát triển khu định cư. Node bị phá hủy không sản xuất.</p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4"><section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]"><h3 className="font-bold text-emerald-300 mb-2">Node & sản xuất</h3><ul className="list-disc list-inside space-y-1"><li>Node tối đa cấp 5; nâng cấp tăng HP và hiệu quả.</li><li>Gán dân số còn rảnh vào node tài nguyên còn sống.</li><li>Hàng đợi luyện chế dùng bí phương, số lượng và ưu tiên.</li></ul></section><section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]"><h3 className="font-bold text-amber-300 mb-2">Xã hội</h3><ul className="list-disc list-inside space-y-1"><li>LOW/NORMAL/HIGH/EXPLOIT đổi sản lượng lấy ổn định.</li><li>Thiếu lương, hạnh phúc thấp và áp bức cao tăng tử vong/phản loạn.</li><li>Khu định cư quản lý sức chứa, nhà ở, kho lương và an ninh.</li></ul></section></div>
        <p className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]">Quần thể đủ lớn có thể sinh Cổ Sư dị nhân với rank, thiên phú, trung thành và lưu phái riêng. Có thể giao vai trò hoặc giải tán NPC. Mô phỏng chạy nền nên số liệu tiếp tục thay đổi khi đóng giao diện.</p>
      </div>
    ),
  },
  {
    id: "gameplay-death-arena",
    title: "Đấu Trường Sinh Tử",
    category: "gameplay",
    keywords: ["đấu trường sinh tử", "death arena", "pvp", "giải đấu", "tournament", "thứ 7", "saturday"],
    content: (
      <div className="space-y-6">
        <div className="p-4 rounded-2xl bg-red-950/30 border border-red-500/20 flex gap-3 items-start">
          <Swords className="w-5 h-5 text-red-400 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-red-200">Đại Chiến Sinh Tử Hàng Tuần</h4>
            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              <strong>Đấu Trường Sinh Tử</strong> là giải đấu PvP tổ đội 6v6 loại trực tiếp đỉnh cao diễn ra định kỳ vào <strong>chiều Thứ 7 hàng tuần</strong>. Nơi các Cổ sư tinh anh cùng tổ đội tranh đoạt vinh quang và phần thưởng Nguyên Thạch khổng lồ.
            </p>
          </div>
        </div>

        <div className="space-y-4 text-xs text-[var(--text-secondary)] leading-relaxed">
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-red-400 flex items-center gap-2">
              <Clock className="w-4 h-4" /> Thời Gian & Lịch Trình
            </h3>
            <ul className="list-disc pl-5 space-y-1.5">
              <li><strong>Thời gian mở đăng ký:</strong> Từ <strong>16:00 đến 17:00 chiều Thứ 7</strong>.</li>
              <li><strong>Thời gian khai chiến:</strong> Đúng <strong>17:00 (5h chiều) Thứ 7</strong>.</li>
              <li><strong>Địa điểm tham gia:</strong> Vào mục <em>Đấu Trường Sinh Tử</em> từ màn hình chính hoặc gõ lệnh <code>/death-arena</code>.</li>
            </ul>
          </div>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-amber-400 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4" /> Quy Mô & Giới Hạn Đăng Ký
            </h3>
            <ul className="list-disc pl-5 space-y-1.5">
              <li><strong>Quy mô đội:</strong> Mỗi đội gồm <strong>6 Cổ sư</strong> (hỗ trợ tự động ghép đội khi đăng ký cá nhân).</li>
              <li><strong>Giới hạn giải đấu:</strong> Tối đa <strong>16 đội</strong> (tổng cộng <strong>96 người chơi</strong>).</li>
              <li>Khi đủ 16 đội hoặc 96 người, cổng đăng ký sẽ tự động đóng sớm.</li>
            </ul>
          </div>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-indigo-400 flex items-center gap-2">
              <Flame className="w-4 h-4" /> Thể Thức Thi Đấu & Nhịp Độ Vòng Đấu
            </h3>
            <ul className="list-disc pl-5 space-y-1.5">
              <li><strong>Loại trực tiếp (Single Elimination):</strong> Bảng đấu gồm 4 vòng (Vòng 1 ➔ Tứ kết ➔ Bán kết ➔ Chung kết). Đội thua bị loại ngay lập tức.</li>
              <li><strong>Nhịp độ vòng đấu:</strong> Cứ mỗi <strong>5 phút chia đôi số đội 1 lần</strong> (16 ➔ 8 ➔ 4 ➔ 2 ➔ 1).</li>
              <li><strong>Nghỉ giữa hiệp:</strong> Sau khi hoàn tất các trận của mỗi vòng, hệ thống sinh nhánh đấu vòng tiếp theo và đếm ngược thời gian nghỉ để các đạo hữu mở xem <strong>Combat Replay</strong> (diễn biến trận đấu vừa xong) và chuẩn bị chiến thuật.</li>
            </ul>
          </div>

          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <h3 className="text-sm font-serif font-bold text-yellow-400 flex items-center gap-2">
              <Coins className="w-4 h-4" /> Phần Thưởng
            </h3>
            <ul className="list-disc pl-5 space-y-1.5">
              <li><strong>Thưởng qua từng vòng:</strong> Mỗi thành viên đội chiến thắng nhận Nguyên Thạch tăng lũy tiến qua từng vòng (Vòng 1: 50, Vòng 2: 100, Vòng 3: 200, Chung kết: 400 Nguyên Thạch).</li>
              <li><strong>Quán Quân & Á Quân:</strong> Nhận thêm Tiên Nguyên Thạch, danh hiệu độc quyền và phần thưởng vật phẩm quý giá gửi về hòm thư.</li>
            </ul>
          </div>
        </div>

        {/* Glossary Table */}
        <div className="space-y-4 pt-6 border-t border-[var(--border)]">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">📖 Glossary / Bảng Thuật Ngữ</h3>
          <div className="overflow-hidden rounded-xl border border-[var(--border)]">
            <table className="w-full text-left text-xs font-mono">
              <thead className="bg-zinc-950 text-[var(--text-secondary)] border-b border-[var(--border)]">
                <tr>
                  <th className="p-3">Thuật Ngữ</th>
                  <th className="p-3">Giải Thích</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-900 text-[var(--text-primary)] bg-[var(--surface)]">
                <tr><td className="p-3 font-bold text-amber-400">damage_flat</td><td className="p-3">Sát thương thuần từ Cổ</td></tr>
                <tr><td className="p-3 font-bold text-amber-400">damage_mult</td><td className="p-3">Hệ số nhân sát thương</td></tr>
                <tr><td className="p-3 font-bold text-cyan-400">shield_flat</td><td className="p-3">Giá trị khiên ảo gốc từ Cổ</td></tr>
                <tr><td className="p-3 font-bold text-cyan-400">shield_mult</td><td className="p-3">Hệ số nhân khiên</td></tr>
                <tr><td className="p-3 font-bold text-emerald-400">daoMult</td><td className="p-3">Khuếch đại Đạo Ngân theo công thức logarit &amp; cảnh giới</td></tr>
                <tr><td className="p-3 font-bold text-violet-400">dmgMultiplier</td><td className="p-3">Buff/debuff tạm thời (Kiếm Long +50%, weaken -20%...)</td></tr>
                <tr><td className="p-3 font-bold text-red-400">netDmg</td><td className="p-3">hitDmg &minus; targetDef; floor = 1 (không bao giờ dưới 1)</td></tr>
                <tr><td className="p-3 font-bold text-orange-400">targetDef</td><td className="p-3">BaseDef &times; DefenseMultiplier &minus; ignore_defense</td></tr>
                <tr><td className="p-3 font-bold text-blue-400">daoConflictMult</td><td className="p-3">Hệ số suy giảm do xung đột Đạo Ngân (0.10 &rarr; 1.00)</td></tr>
                <tr><td className="p-3 font-bold text-purple-400">naturalResistance</td><td className="p-3">Kháng Đạo Ngân tự nhiên từ Cổ Tiên phòng thủ (cap 30%)</td></tr>
                <tr><td className="p-3 font-bold text-pink-400">LifeStealPct</td><td className="p-3">Tỉ lệ hồi HP từ sát thương gây ra (heal = totalDmg &times; LifeStealPct)</td></tr>
                <tr><td className="p-3 font-bold text-green-400">critRateBonus</td><td className="p-3">Tỉ lệ chí mạng; nếu roll thành công &rarr; damage &times; 1.5</td></tr>
                <tr><td className="p-3 font-bold text-yellow-400">MultiHit</td><td className="p-3">Số đòn đánh trong cùng lượt; đòn thứ 2+ &times; 0.7</td></tr>
                <tr><td className="p-3 font-bold text-lime-400">BYPASS_SHIELD</td><td className="p-3">Cơ chế bỏ qua khiên; giảm targetDef đi 30 trước khi tính</td></tr>
                <tr><td className="p-3 font-bold text-teal-400">VirtualShield</td><td className="p-3">Khiên ảo hiện tại — trừ trước HP, không hồi giữa lượt</td></tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    ),
  },
  {
    id: "gameplay-baothach",
    title: "Bảo Thạch & Tiền Tệ",
    category: "gameplay",
    keywords: ["bảo thạch", "nguyên thạch", "tiên nguyên thạch", "tiền tệ", "currency", "stones", "money"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Trong thế giới Cổ sư, tiền tệ không chỉ dùng để mua bán mà bản thân chúng còn là kết tinh năng lượng tinh khiết giúp hồi phục năng lượng và chế tạo vật phẩm.
        </p>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-2xl p-4 flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-yellow-950/50 flex items-center justify-center border border-yellow-500/20">
                <Coins className="w-4 h-4 text-yellow-500" />
              </div>
              <h4 className="text-sm font-serif font-bold text-[var(--text-primary)]">Nguyên Thạch</h4>
            </div>
            <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-1">
              Tiền tệ phổ biến của phàm nhân (Nhất Chuyển đến Ngũ Chuyển). Chứa chân nguyên tự nhiên để hồi phục năng lượng cho Cổ sư và là nguyên liệu bắt buộc để kích hoạt/tăng hiệu suất trùng kích vách khiếu.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-2xl p-4 flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-purple-950/50 flex items-center justify-center border border-purple-500/20">
                <Coins className="w-4 h-4 text-purple-400" />
              </div>
              <h4 className="text-sm font-serif font-bold text-[var(--text-primary)]">Tiên Nguyên Thạch</h4>
            </div>
            <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-1">
              Đơn vị tiền tệ tối cao của giới Cổ Tiên. Dùng để giao dịch các vật phẩm hiếm bậc nhất tại Chợ Bảo Hoàng Thiên, hoặc trực tiếp nghiền nát để bổ sung lượng lớn Tiên Nguyên chiến đấu.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-[var(--border)] rounded-2xl p-4 flex flex-col gap-2">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-rose-950/50 flex items-center justify-center border border-rose-500/20">
                <Swords className="w-4 h-4 text-rose-500" />
              </div>
              <h4 className="text-sm font-serif font-bold text-[var(--text-primary)]">Chiến Công</h4>
            </div>
            <p className="text-[11px] text-[var(--text-muted)] leading-relaxed mt-1">
              Tiền tệ tạm thời được phát hành bởi các gia tộc lớn hoặc liên minh môn phái trong thời chiến (như lang triều). Dùng để đổi vật tư, thực phẩm hoặc mượn các cổ trùng chiến thuật.
            </p>
          </div>
        </div>
      </div>
    ),
  },
  {
    id: "gameplay-tongmon",
    title: "Tông Môn Thế Lực",
    category: "gameplay",
    keywords: ["tông môn", "sect", "gia tộc", "cống hiến", "khảo hạch", "hierarchy", "sư đồ", "master"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Thay thế chế độ gia tộc dựa trên huyết thống, Tông môn là tổ chức thế lực tuyển chọn đệ tử dựa trên năng lực, phẩm hạnh và sự trung thành, vận hành bằng hệ thống sư đồ và điểm cống hiến nghiêm ngặt.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">1. Phân cấp chức vụ & Khảo hạch</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Hệ thống chức vụ: <strong>Ngoại Môn ➔ Nội Môn ➔ Tinh Anh ➔ Chân Truyền ➔ Trưởng Lão ➔ Hộ Pháp ➔ Chưởng Môn</strong>. Chi phí thăng cấp bằng Điểm cống hiến lần lượt là: <strong>100 / 500 / 2.000 / 10.000 / 30.000</strong>. Đệ tử ngoại môn và nội môn cần vượt qua các kỳ khảo hạch:
          </p>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3 text-xs">
            <div className="flex justify-between border-b border-zinc-800 pb-2">
              <span className="font-bold text-[var(--text-primary)]">Tiểu Khảo</span>
              <span className="text-[var(--text-muted)]">Mỗi 3 ngày thực tế • Thăng cấp Nội Môn</span>
            </div>
            <div className="flex justify-between border-b border-zinc-800 pb-2">
              <span className="font-bold text-[var(--text-primary)]">Trung Khảo</span>
              <span className="text-[var(--text-muted)]">Mỗi 8 ngày thực tế • Quyết đấu chọn Tinh Anh</span>
            </div>
            <div className="flex justify-between pb-1">
              <span className="font-bold text-[var(--text-primary)]">Đại Khảo</span>
              <span className="text-[var(--text-muted)]">Mỗi 15 ngày thực tế • Sinh tồn chọn Chân Truyền</span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">2. Lãnh địa mỏ Tông môn & Quỹ</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Môn phái có thể chiếm Lãnh địa để tự sinh <strong>Quỹ tông môn</strong> và <strong>Linh tài</strong> theo giờ. Lãnh đạo (Trưởng Lão, Hộ Pháp, Chưởng Môn) được quyền bố trí đệ tử hoặc thuê thần thú canh giữ, đồng thời phát động tấn công mỏ của tông môn khác. Đệ tử phòng thủ nhận +20 cống hiến/giờ, các thành viên khác +2 cống hiến/giờ (tối đa 200/ngày).
          </p>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">3. Điểm Cống Hiến & Bảo Khố</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Điểm cống hiến nhận được khi quyên góp cổ trùng, nguyên thạch, bí phương hoặc làm nhiệm vụ tông môn.
          </p>
          <div className="p-4 rounded-2xl bg-indigo-950/20 border border-indigo-500/10 text-xs leading-relaxed text-indigo-300">
            💡 <strong>Sử dụng Điểm cống hiến:</strong> Đổi lấy các công thức, Tiên Cổ từ kho tông môn, sách bí phương hoặc dùng để thăng cấp chức vụ.
          </div>
        </div>
      </div>
    ),
  },
  {
    id: "gameplay-scouting-divination",
    title: "Trinh Sát & Phản Trinh Sát",
    category: "gameplay",
    subCategory: "gameplay-ui-navigation",
    keywords: ["trinh sát", "suy diễn", "thiên cơ", "ấn ký", "phản trinh sát", "scouting", "divination", "sever", "tracking"],
    content: (
      <div className="space-y-4 text-xs text-[var(--text-secondary)] leading-relaxed">
        <section className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)]">
          <h3 className="font-bold text-amber-300 mb-2">Ba trụ điểm — cảnh giới là trụ số một</h3>
          <p className="mb-2 text-neutral-400">
            Sức dò chia làm ba trụ, <strong>càng đầu bảng càng quan trọng</strong>:
          </p>
          <ul className="list-disc list-inside space-y-1.5">
            <li>
              <strong className="text-purple-300">① Cảnh giới lưu phái (Trí đạo) — trụ số một.</strong> Mỗi bậc cộng thẳng vào điểm, không phụ thuộc tu vi:
              {" "}<strong>Phàm +0 · Đại Sư +50 · Tông Sư +140 · Đại Tông Sư +280 · Vô Thượng +470</strong>, cộng thêm một hệ số nhân nhẹ ×1.00 → ×1.30.
              Đây là con đường <strong>đáng đầu tư nhất</strong>: mỗi bậc lên đều đáng hơn cả trăm điểm, và không bị tu vi thấp kéo lùi.
            </li>
            <li>
              <strong className="text-cyan-300">② Cổ trùng.</strong> Mỗi Cổ Trinh Sát đang trang bị cộng theo phẩm bậc (Cổ 3 chuyển +15, Cổ 7 chuyển +35, Cổ 9 chuyển +45). Xếp sau cảnh giới nhưng vẫn rõ ràng — trang bị đủ Cổ là bước nhảy rõ rệt.
            </li>
            <li>
              <strong className="text-amber-300">③ Đạo ngân — yếu nhất, bão hòa rất sớm.</strong> Đạo ngân Trí/Tín chỉ tăng tối đa <strong>+20%</strong> điểm nền, và đạt trần khi khoảng 3.000 điểm đạo ngân. Một bậc cảnh giới luôn đáng hơn toàn bộ đạo ngân, nên <strong>đừng dồn đạo ngân thay vì luyện cảnh giới</strong>.
            </li>
          </ul>
          <p className="mt-2 text-neutral-400">
            Ngoài ra còn có <strong>Manh mối</strong> (tên đúng +20, đúng vùng +15, đúng tông môn +10, Cổ nghi vấn đúng +10, khí tức đúng +30, dò mù −40 — chỉ manh mối <em>đúng thật</em> mới tính) và <strong>Vận số</strong> gieo mỗi lần dò. Mỗi lần dò tốn <strong>Nguyên Thạch</strong> (tối thiểu 100) hoặc <strong>Tiên Nguyên Thạch</strong> (Tiên nhân) — không tốn Chân Nguyên.
          </p>
        </section>
        <section className="p-4 rounded-xl bg-emerald-950/20 border border-emerald-500/20">
          <h3 className="font-bold text-emerald-300 mb-2">Bốn kết quả (so điểm tổng cuối)</h3>
          <ul className="list-disc list-inside space-y-1">
            <li><strong>Đại thành công</strong> (≥+30 người / +20 Cổ): tọa độ thật, tu vi thật, toàn bộ Cổ trang bị; được gieo ấn ký.</li>
            <li><strong>Tiểu thành công</strong>: vùng + tu vi ước lượng + 2 Cổ, tọa độ bị giấu.</li>
            <li><strong>Màn sương</strong>: mất tài nguyên, trắng tay.</li>
            <li><strong>Phản phệ</strong>: mất <strong>15% HP và 15% Hồn tối đa</strong> cùng 20 Chân Nguyên, và <strong>lộ ngược tọa độ cho mục tiêu</strong>.</li>
          </ul>
          <p className="mt-2">Chênh lệch tu vi được tính theo tỉ lệ dịu, nên lệch 2 bậc chỉ khiến bạn <em>màn sương</em> chứ chưa chắc chắn phản phệ. Sau mỗi lần dò, bảng <strong>điểm của chính đạo hữu</strong> hiện đầy đủ để biết thiếu ở đâu — điểm thủ của địch không bao giờ lộ.</p>
        </section>
        <section className="p-4 rounded-xl bg-purple-950/20 border border-purple-500/20">
          <h3 className="font-bold text-purple-300 mb-2">Suy tính Cổ & Ấn ký</h3>
          <ul className="list-disc list-inside space-y-1">
            <li>Cổ trong tay người mang khí tức chủ: trang bị được che full, nằm kho/ngủ được che một nửa. Tự sở hữu Cổ đó thì gần như luôn thấu triệt.</li>
            <li>Ấn ký truy tung: tối đa 3 mục tiêu (phàm) / 10 (tiên), cho tọa độ realtime trong radar.</li>
            <li><strong>Đoạn Tuyệt Thiên Cơ:</strong> tốn 20 Chân Nguyên (cố định), chém ấn ký trên người + xóa tàn dư khí tức tại ô đang đứng.</li>
          </ul>
        </section>
      </div>
    ),
  },
];
