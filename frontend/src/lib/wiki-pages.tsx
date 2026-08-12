import React from "react";
import {
  Sparkles,
  Coins,
  Swords,
  AlertTriangle,
  Compass,
  Flame,
  HelpCircle,
  Clock,
} from "lucide-react";

export interface WikiPage {
  id: string;
  title: string;
  category: string;
  subCategory?: string;
  keywords: string[];
  content: React.ReactNode;
  cards?: { title: string; targetId: string; description: string }[];
}

export const wikiPages: WikiPage[] = [
  {
    id: "roadmap",
    title: "Dự Định Cập Nhật & Lộ Trình Phát Triển",
    category: "roadmap",
    keywords: ["roadmap", "cập nhật", "update", "lộ trình", "phát triển", "future"],
    content: (
      <div className="space-y-6">
        <div className="p-4 rounded-2xl bg-indigo-950/30 border border-indigo-500/20 flex gap-3 items-start">
          <Sparkles className="w-5 h-5 text-indigo-400 shrink-0 mt-0.5" />
          <div>
            <h4 className="text-sm font-serif font-bold text-indigo-200">Định hướng cốt lõi</h4>
            <p className="text-xs text-[var(--text-secondary)] mt-1 leading-relaxed">
              GuWorld hướng tới việc tái hiện trọn vẹn sự tàn khốc, triết lý và tư duy sinh tồn trong bối cảnh tiểu thuyết <strong>Cổ Chân Nhân</strong>. Lộ trình phát triển được phân chia theo các giai đoạn nâng cấp đồng bộ cả hệ thống dữ liệu lẫn trải nghiệm của người chơi.
            </p>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Các Phase cập nhật</h3>

          <div className="relative border-l border-zinc-800 pl-5 ml-2.5 space-y-6 text-xs">
            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phase 1, 2 & 3 (Khởi Đầu)</span>
                <span className="px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 font-mono text-[9px]">Hoàn thành</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Thiết lập nền tảng cốt lõi của thế giới Cổ sư, hệ thống tài khoản, giao diện cơ bản và các cơ chế chiến đấu phàm nhân sơ cấp.
              </p>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phase 4 & 5 (Mở Rộng Sinh Thái)</span>
                <span className="px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 font-mono text-[9px]">Hoàn thành</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Xây dựng hệ thống phàm nhân cơ bản, trùng kích vách khiếu, các Cổ trùng từ Nhất Chuyển đến Ngũ Chuyển, thương đội Caravan, lò luyện cổ và chợ đấu giá Bảo Hoàng Thiên.
              </p>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Tối Ưu Trải Nghiệm (Song Song)</span>
                <span className="px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 font-mono text-[9px]">Đang thực hiện</span>
              </h4>
              <p className="text-[var(--text-secondary)] mt-1 leading-relaxed">
                Tối ưu hóa các tính năng đang có, cải tiến giao diện người dùng (như chế độ sáng/tối, khung chat toàn cục, thu phóng kích thước), đồng bộ thông báo tự động từ NPC Hệ Thống. Lắng nghe phản hồi và tiếp thu các yêu cầu thiết thực từ người chơi để nâng cấp game.
              </p>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-indigo-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phase 6 (Cơ Chế Sát Chiêu)</span>
                <span className="px-1.5 py-0.5 rounded bg-indigo-950 text-indigo-400 font-mono text-[9px]">Đang phát triển</span>
              </h4>
              <p className="text-[var(--text-secondary)] mt-1 leading-relaxed">
                Hoàn thiện cơ chế <strong>Sát Chiêu</strong> (các chiêu thức đặc biệt của Cổ trùng).
              </p>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-indigo-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phase 7 (Thăng Tiên & Độ Kiếp)</span>
              </h4>
              <p className="text-[var(--text-secondary)] mt-1 leading-relaxed">
                Hoàn thiện cơ chế <strong>Thăng Tiên (Ngũ Chuyển Đỉnh Phong thăng lên Lục Chuyển)</strong> và <strong>Độ Kiếp (Cổ Tiên vượt Địa Tai, Thiên Kiếp)</strong>. Xây dựng sinh thái Tiên Khiếu độc lập, chăn nuôi, trồng trọt và khai thác dị nhân.
              </p>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-secondary)]">Phase 8 & 9 (Mộng Cảnh & Hồn Phách)</h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Bổ sung phó bản <strong>Mộng Cảnh</strong> đầy đủ, nâng cấp hệ thống <strong>Hồn Phách</strong> (Tráng Hồn ở núi Đãng Hồn, Luyện Hồn ở cốc Lạc Phách). Mở rộng PvP đấu trường liên lục địa.
              </p>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-secondary)]">Phase 10 (Tiên cổ phòng)</h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Hoàn thiện cơ chế <strong>Tiên Cổ Phòng</strong>
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4 pt-4 border-t border-zinc-900">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Nhật Ký Cập Nhật</h3>

          <div className="relative border-l border-zinc-800 pl-5 ml-2.5 space-y-6 text-xs">

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.8.0 (Hiện tại)</span>
                <span className="px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 font-mono text-[9px]">
                  Mới nhất
                </span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Tối ưu lưu trữ:</strong> Kho Cổ trùng được lưu trữ gọn gàng hơn, mở kho, trang bị và cho ăn nhanh mượt hơn kể cả khi sở hữu rất nhiều Cổ.
                </li>
                <li>
                  <strong>Cho ăn Cổ trùng ổn định:</strong> Độ no giảm đều theo thời gian, tránh lag khi máy chủ xử lý toàn bộ người chơi cùng lúc.
                </li>
                <li>
                  <strong>Đạo Ngân vận hành tốt hơn:</strong> Đạo Ngân hiển thị và áp dụng hiệu ứng chính xác cho cả đồng đội lẫn địch trong chiến đấu tự động.
                </li>
                <li>
                  <strong>Thăng Tiên theo Nhân Khí:</strong> Tỉ lệ thành công tỉ lệ thuận với Nhân Khí (0 → 0%, 100.000 → chắc chắn thành công). Thành công lên Lục Chuyển Cổ Tiên, nhận thưởng và khai mở Tiên Khiếu — phẩm cấp Tiên Khiếu được định đoạt bởi Nhân Khí lúc thăng tiên. Thất bại: Nhân Khí về 0, tu vi phế bỏ rơi về Nhất Chuyển.
                </li>
              </ul>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.7.1</span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Bảng Xếp Hạng:</strong> Hệ thống BXH tổng hợp với 4 tiêu chí: Tu Luyện, Nguyên Thạch, Cổ Trùng (cấp cao nhất) và Tiên Nguyên. Người chơi có thể chủ động rút khỏi hoặc tham gia BXH. Admin bị loại khỏi bảng xếp hạng.
                </li>
              </ul>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.7.0</span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Mở đá:</strong> Hệ thống quay thưởng ngẫu nhiên với. Mở từ 1 hoặc 10 lần, nhận Nguyên Thạch, Tiên Nguyên Thạch, Thọ Nguyên, Cổ trùng và Đạo Ấn. Ra mắt hub mới "Mở đá" tại sảnh chính.
                </li>
                <li>
                  <strong>Mã Quà (Gift Code):</strong> Admin tạo mã quà tặng, người chơi nhập mã tại tab "Phúc Lợi" trong Không Khiếu để nhận thưởng. Admin panel có mục "Mã Quà" riêng.
                </li>
                <li>
                  <strong>Thành Tựu & Danh Hiệu:</strong> Hệ thống thành tựu với 23 mốc, mỗi thành tựu có phần thưởng Nguyên Thạch/Tiên Nguyên Thạch. Một số thành tựu mở khóa khung Frame và Danh Hiệu hiển thị cạnh tên trong khung chat. Quản lý tại tab "Thành Tựu" trong Không Khiếu.
                </li>
                <li>
                  <strong>Tối Ưu Quản Trị:</strong> Tạo mã quà mới có UI chọn vật phẩm bằng picker thay vì nhập tay ID.
                </li>
              </ul>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.6.8</span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Cân Bằng Gameplay:</strong> Cập nhật thời gian giảm độ no của Cổ trùng (Hunger Decay Background Worker) lên 3 giờ/lần.
                </li>
                <li>
                  <strong>Tính Năng Mới:</strong> Thêm Cửa hàng Đấu trường (Arena Shop) và hệ thống Nhiệm vụ hàng ngày (Daily Challenges) cùng với các thành phần Frontend và Handler tương ứng.
                </li>
                <li>
                  <strong>Tối Ưu Trải Nghiệm:</strong> Nâng cấp logic cho Cổ ăn, thêm thông báo (toast) cảnh báo chi tiết khi cho ăn đơn và ăn nhóm nếu không đủ thức ăn.
                </li>
              </ul>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.6.7.1</span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Tối Ưu Đăng Ký Nhân Vật:</strong> Tinh gọn giao diện tạo nhân vật, giúp quá trình khởi tạo trực quan hơn.
                </li>

                <li>
                  <strong>Cải Tiến Hướng Dẫn Tân Thủ:</strong> Refactor luồng onboarding, tối ưu quản lý trạng thái và bổ sung cơ chế đánh dấu hoàn tất hướng dẫn sau khi người chơi thiết lập nhân vật.
                </li>

                <li>
                  <strong>Tông Môn:</strong> Refactor giao diện quản lý Tông Môn (Sect View), cải thiện cấu trúc mã nguồn, tăng khả năng bảo trì và thống nhất logic sắp xếp dữ liệu.
                </li>

                <li>
                  <strong>Tối Ưu Mã Nguồn:</strong> Dọn dẹp các đoạn mã không còn sử dụng, giúp giảm độ phức tạp, cải thiện hiệu năng và thuận tiện cho việc phát triển các phiên bản tiếp theo.
                </li>
              </ul>
            </div>
            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.6.7</span>
              </h4>
              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li><strong>NPC Hệ Thống:</strong> Gửi tin nhắn sự kiện thời gian thực lên kênh Bảo Hoàng Thiên (khi có vật phẩm mới rao bán/đấu giá) và kênh Tông Môn (khi lập tông, xin gia nhập, rời tông, bị trục xuất, quyên góp tài sản hoặc nhập hàng bảo khố).</li>
                <li><strong>Khung Chat Toàn Cục (Global Chat):</strong> Di chuyển khung chat ra ngoài trang chính, cho phép chat liền mạch xuyên suốt tất cả các tính năng mà không bị ngắt kết nối.</li>
                <li><strong>Thu Phóng Khung Chat:</strong> Thêm tính năng thay đổi chiều ngang (↔) và chiều cao (↕) linh hoạt cho khung chat.</li>
                <li><strong>Nâng Cấp Giao Diện Sáng (Light Mode):</strong> Điều chỉnh tông màu chữ, viền và hình nền dịu hơn, tăng độ tương phản giúp đạo hữu dễ dàng đọc tin nhắn.</li>
                <li><strong>Tải Tin Nhắn Tức Thì:</strong> Tự động kéo dữ liệu tin nhắn cũ qua REST API ngay khi click mở tab để loại bỏ độ trễ của kết nối SSE.</li>
              </ul>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-secondary)] flex items-center gap-2">
                <span>Phiên Bản v0.6.0</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Khởi chạy tính năng Tông môn bản thử nghiệm, Quyên góp, Quản lý tài nguyên bang hội và mở rộng đấu giá Bảo Hoàng Thiên.
              </p>
            </div>
          </div>
        </div>
      </div>
    )
  },
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
              <h4 className="font-bold text-indigo-300">Đạo Ngân (Dao Marks)</h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Vết khắc của quy luật tự nhiên tích lũy trên cơ thể Cổ Tiên gồm 49 lưu phái (Lực đạo, Hỏa đạo, Hồn đạo, Luyện đạo...). Mỗi điểm Đạo Ngân tăng trực tiếp <strong>+1%</strong> sát thương, phòng ngự, trị liệu hoặc khiên bảo hộ cho các kỹ năng thuộc lưu phái đó.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-2 text-xs">
              <h4 className="font-bold text-indigo-300">Cảnh Giới Lưu Phái (Attainment / Chân Ý)</h4>
              <p className="text-[var(--text-muted)] leading-relaxed">
                Độ hiểu biết về quy luật thế giới. Chia làm 5 cảnh giới: <strong>Phàm Thai (Ordinary)</strong> ➔ <strong>Đại Sư (Master)</strong> ➔ <strong>Tông Sư (Grandmaster)</strong> ➔ <strong>Đại Tông Sư (Great Grandmaster)</strong> ➔ <strong>Vô Thượng Đại Tông Sư (Supreme Grandmaster)</strong>. Điểm tích lũy theo từng lưu phái từ sát chiêu thành công, luyện Cổ và vượt ải phó bản; cảnh giới suy ra theo mốc điểm (Đại Sư=100, Tông Sư=500, Đại Tông Sư=2.500, Vô Thượng=10.000). **Để đột phá lên Rank 7/8/9 khi vượt Kiếp Nạn, cần ít nhất 1 lưu phái đạt Tông Sư/Đại Tông Sư/Vô Thượng tương ứng.** Cảnh giới càng cao, sát chiêu lưu phái đó càng uy lực và tỉ lệ luyện Cổ thành công càng lớn.
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
                ⚠️ <strong>Quy luật sinh tồn:</strong> Cổ trùng tiêu hao <strong>Độ No (Hunger)</strong> liên tục <strong>mỗi 3 giờ</strong>. Hãy tích lũy đúng loại thức ăn (thịt thú, khoáng vật, ngọc lộ...) và cho chúng ăn kịp thời. Cổ trùng đói về 0 độ no sẽ <strong>chết vĩnh viễn và biến mất khỏi hòm đồ</strong>!
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
                Tích lũy <strong>Nhân Khí (Human Qi)</strong> thông qua các hoạt động hàng ngày trước khi thăng tiên để gia tăng tỉ lệ thành công (tối đa 100k Nhân Khí = chắc chắn 100% thành công). Nếu thăng tiên thất bại, tu vi sẽ bị phế bỏ rơi về Nhất Chuyển.
              </p>
              <p>
                Sau khi thăng tiên, đạo hữu sẽ sở hữu <strong>Tiên Khiếu</strong> riêng biệt và phải đối đầu với các <strong>Kiếp Nạn</strong> (Địa Tai, Thiên Kiếp...) định kỳ. Kiếp đầu tiên giáng lâm sau <strong>10 ngày đời thật</strong>. Vượt qua kiếp nạn thành công sẽ nhận được lượng lớn Đạo Ngân để gia tăng sức mạnh vượt bậc.
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
                    <td className="p-3 font-bold text-[var(--text-primary)]">Sau 10 ngày thăng tiên</td>
                    <td className="p-3">Kiếp nạn đầu tiên giáng lâm — chuẩn bị phòng thủ Tiên Khiếu và chiến lực.</td>
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
    )
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
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">1. Phân cấp đệ tử & Khảo hạch</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Bạn sẽ bắt đầu là một Đệ tử ngoại môn và leo dần lên các bậc cao hơn. Mỗi bậc đều giới hạn số lượng người chơi tối đa và yêu cầu vượt qua các kỳ thi định kỳ:
          </p>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3 text-xs">
            <div className="flex justify-between border-b border-zinc-800 pb-2">
              <span className="font-bold text-[var(--text-primary)]">Tiểu Khảo</span>
              <span className="text-[var(--text-muted)]">Mỗi 3 ngày thực tế • Điều kiện thăng cấp Nội Môn</span>
            </div>
            <div className="flex justify-between border-b border-zinc-800 pb-2">
              <span className="font-bold text-[var(--text-primary)]">Trung Khảo</span>
              <span className="text-[var(--text-muted)]">Mỗi 8 ngày thực tế • Quyết đấu chọn đệ tử Tinh Anh</span>
            </div>
            <div className="flex justify-between pb-1">
              <span className="font-bold text-[var(--text-primary)]">Đại Khảo</span>
              <span className="text-[var(--text-muted)]">Mỗi 15 ngày thực tế • Sinh tồn chọn đệ tử Chân Truyền</span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">2. Điểm Cống Hiến & Tàng Kinh Các</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Đây là kinh tế cốt lõi trong tông môn. Điểm cống hiến nhận được khi bạn quyên góp cổ trùng, nguyên thạch hoặc làm nhiệm vụ tông môn tại Bảng Nhiệm Vụ (thu thập dược liệu, luyện bồi tập đấu, chinh phạt hoang thú dã ngoại).
          </p>
          <div className="p-4 rounded-2xl bg-indigo-950/20 border border-indigo-500/10 text-xs leading-relaxed text-indigo-300">
            💡 <strong>Sử dụng Điểm cống hiến:</strong> Đổi lấy các công thức cổ phương độc quyền, mượn Tiên Cổ từ kho tông môn theo ngày, hoặc đổi lấy sách bí phương sát chiêu tại Tàng Kinh Các.
          </div>
        </div>
      </div>
    )
  },
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
                  <td className="p-3">Tài nguyên phản ánh hoạt động tích cực của đạo hữu trong giới tu tiên, giới hạn tối đa 100.000.</td>
                  <td className="p-3">Quyết định tỉ lệ Thăng Tiên thành công và phẩm cấp Tiên Khiếu khi mở.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tư chất Không Khiếu</td>
                  <td className="p-3">Phẩm cấp Linh Căn (Giáp/Ất/Bính/Đinh kèm hệ số phần trăm).</td>
                  <td className="p-3">Ảnh hưởng trữ lượng Chân Nguyên tối đa, tốc độ hồi phục và hiệu suất phá vách.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Thọ nguyên (Lifespan)</td>
                  <td className="p-3">Tuổi thọ còn lại của cơ thể, hao mòn dần theo thời gian thực (nhân với hệ số thời gian tu luyện).</td>
                  <td className="p-3">Về 0 đồng nghĩa sinh mệnh tàn lụi — cần tăng tu vi hoặc dùng ngoại lực kéo dài.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Áp lực vách khiếu</td>
                  <td className="p-3">Mức độ tổn thương của Không Khiếu khi trùng kích quá độ.</td>
                  <td className="p-3">Vượt 80% làm suy giảm hiệu quả; chạm 100% Không Khiếu bị nứt vỡ (rớt tu vi, mất vĩnh viễn Chân Nguyên tối đa).</td>
                </tr>
              </tbody>
            </table>
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
            <li>Nếu áp lực chạm mốc <strong>100%</strong>, Không khiếu bị nứt: đạo hữu <strong>mất vĩnh viễn 5% Chân Nguyên tối đa</strong>, tiến độ vách khiếu về 0, áp lực giảm về 50% và <strong>rớt xuống một Tiểu Cảnh Giới (hoặc một Chuyển nếu đang ở Sơ Giai)</strong>!</li>
          </ul>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] text-xs leading-relaxed text-[var(--text-secondary)]">
            💡 <strong>Mẹo:</strong> Sử dụng vật phẩm tiêu hao <strong>Tịnh Thủy Cổ</strong> để lập tức thanh lọc, đưa thanh Áp lực vách khiếu về 0% ngay tức khắc.
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Thuộc tính thân thể</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Ngoài tu vi, hồ sơ còn ghi nhận <strong>Chủng tộc</strong> (Nhân Tộc, Người Lông, Nhân Ngư, Thạch Nhân, Long Nhân...), <strong>Giới tính</strong>, <strong>Thể chất</strong> (Hoang Cổ Nguyệt Thể, Thanh Liên Đại Nhật Thể...) và mảng <strong>Trạng thái</strong> (ví dụ Trọng Thương giảm 20% chiến lực). Các giá trị này mang hiệu ứng riêng lên tu luyện và chiến đấu, xem chi tiết trong mục Hồ Sơ.
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
            <li>Khi tư chất thay đổi, phẩm cấp Linh Căn tự động cập nhật theo ngưỡng: ≥80% = Giáp, ≥60% = Ất, ≥40% = Bính, dưới 40% = Đinh.</li>
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
                <span>1. Chuẩn Đại Sư / Đại Sư</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Trực giác siêu nhạy bén. Trong chiến đấu, nếu đối phương sử dụng sát chiêu thuộc hệ phái bạn là Đại Sư, giao diện sẽ hiển thị gợi ý né tránh hoặc chỉ ra nhược điểm của chiêu đó.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-indigo-300 flex items-center gap-2">
                <span>2. Chuẩn Tông Sư / Tông Sư</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Đạt khả năng &quot;Loại suy&quot;. Loại bỏ hoàn toàn hình phạt mâu thuẫn giảm nửa sát thương khi sử dụng chéo hệ Cổ trùng (Ví dụ: Cổ sư Lực đạo dùng Cổ Thủy đạo vẫn nhận 100% hiệu năng).
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-yellow-500 flex items-center gap-2">
                <span>3. Chuẩn Đại Tông Sư / Đại Tông Sư</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Cộng hưởng hoàn hảo với Đạo ngân của thiên nhiên. Tự động nhận thêm 50% sát thương/phòng ngự khi chiến đấu tại địa hình tương đồng hệ phái của bạn mà không cần kích hoạt cổ buff.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs">
              <h4 className="font-bold text-red-400 flex items-center gap-2">
                <span>4. Vô Thượng Đại Tông Sư</span>
              </h4>
              <p className="text-[var(--text-muted)] mt-1 leading-relaxed">
                Cảnh giới cao nhất. Sát chiêu tự động vận hành trong Tiên khiếu, mở khóa tính năng đặc biệt: Nuốt (thôn tính) Tiên khiếu của Cổ Tiên khác để tăng năng suất và né tránh tai kiếp.
              </p>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Đạo Ngân (Dao Marks)</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Đạo Ngân là điểm lĩnh ngộ tích lũy theo từng lưu phái trong số <strong>49 lưu phái</strong> (Lực đạo, Hỏa đạo, Thủy đạo, Phong đạo, Lôi đạo, Hồn đạo, Thời gian đạo, Không gian đạo...). Mỗi điểm Đạo Ngân gia tăng sát thương và phòng ngự cho Cổ/Kỹ năng thuộc lưu phái đó.
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
                  <td className="p-3">Ban đầu nhận 100 điểm mỗi lưu phái cho hai đường ngẫu nhiên.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Tai Kiếp</td>
                  <td className="p-3">Vượt qua Địa Tai, Thiên Kiếp, Hạo Kiếp... nhận lượng Đạo Ngân lớn theo độ khó.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Sát Chiêu</td>
                  <td className="p-3">Rèn luyện sát chiêu tích lũy Đạo Ngân vào lưu phái cốt lõi.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Hiệu ứng Cổ / vật phẩm</td>
                  <td className="p-3">Một số Cổ và vật phẩm có hiệu ứng tăng Đạo Ngân trực tiếp.</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] text-xs leading-relaxed text-[var(--text-secondary)]">
            💡 <strong>Công thức sát thương:</strong> Mỗi điểm Đạo Ngân = <strong>+1%</strong> sát thương/phòng ngự cho lưu phái đó, áp dụng cả trong chiến đấu thường lẫn tự động. Vượt quá 10.000 điểm, hiệu quả giảm xuống còn <strong>+0.2%/điểm</strong> (soft cap).
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
            ⚠️ <strong>Lưu ý cực kỳ quan trọng:</strong> Mỗi con Cổ trang bị đều có thanh độ no. Độ no giảm dần sau mỗi lượt chiến đấu hoặc theo thời gian thực tế. Nếu độ no về 0, Cổ sẽ rơi vào trạng thái <strong>Hấp Hối</strong> và sẽ <strong>chết vĩnh viễn</strong> nếu không được cho ăn kịp thời!
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
    id: "combat",
    title: "Chiến đấu Lưới 3x3",
    category: "combat",
    keywords: ["chiến đấu", "combat", "lưới", "3x3", "grid", "speed", "tốc độ", "khiên", "shield", "virtual shield", "aoe"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Các trận đấu trong game áp dụng cơ chế sắp xếp đội hình trên lưới 3x3 đối lập nhau. Sát thương nhận vào sẽ ưu tiên trừ trực tiếp vào lớp Cổ phòng ngự của bạn.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">1. Vị trí và tầm đánh AoE</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Bạn có thể xếp quân ở Tiền tuyến (X=1), Trung tuyến (X=2) hoặc Hậu phương (X=3). Trừ khi kỹ năng có tầm đánh đặc thù, các sát chiêu đơn mục tiêu bắt buộc phải đánh trúng và dọn sạch hàng Tiền tuyến của đối phương trước mới có thể đánh vào hàng sau.
          </p>
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-2 text-[var(--text-secondary)]">
            <li>💥 <strong>Đánh Ngang:</strong> Quét sạch 3 ô nằm trên cùng một hàng ngang X.</li>
            <li>💥 <strong>Đánh Dọc:</strong> Xuyên từ trước ra sau trên cùng một cột dọc Y.</li>
            <li>💥 <strong>Toàn bản đồ:</strong> Tác động đồng thời lên cả 9 ô của kẻ địch.</li>
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">2. Khiên ảo & Tốc độ</h3>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Sinh mạng cơ bản của Cổ sư cực kỳ mong manh (chỉ 100 Sinh Lực). Do đó, bạn bắt buộc phải trang bị Cổ phòng ngự để nhận được lớp <strong>Khiên ảo</strong> lớn. Khi bị tấn công, sát thương trừ vào khiên trước, khiên vỡ mới tính vào sinh lực cơ thể.
          </p>
          <div className="p-4 rounded-2xl bg-indigo-950/20 border border-indigo-500/10 text-xs leading-relaxed text-indigo-300">
            ⚡ <strong>Tốc độ lượt đấu:</strong> Lượt đánh của toàn bộ mục tiêu được sắp xếp theo chỉ số Tốc độ từ cao tới thấp. Nếu mục tiêu của bạn bị tiêu diệt trước bởi đồng đội có tốc độ cao hơn, lượt đánh của bạn sẽ bị hủy bỏ hoặc chuyển hướng ngẫu nhiên.
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
            Mỗi con Cổ nâng cấp hoặc hợp thành đều cần một bản vẽ <strong>Cổ Phương</strong> cụ thể. Bạn cần đưa Cổ trùng nguyên liệu chính và các nguyên liệu phụ trợ (Hàn Ngọc, Địa Tinh, Quặng Sắt...) kèm theo lượng lớn Nguyên Thạch để kích hoạt.
          </p>
          <div className="p-4 rounded-2xl bg-amber-950/20 border border-amber-500/10 text-xs text-amber-300 leading-relaxed">
            ⚠️ <strong>Rủi ro luyện cổ:</strong> Mọi quá trình luyện cổ đều có tỷ lệ thất bại tự nhiên. Khi thất bại, bạn sẽ mất toàn bộ nguyên liệu phụ và có xác suất bị hư hỏng hoặc mất vĩnh viễn Cổ trùng nguyên liệu chính!
          </div>
          <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
            Nâng cao cảnh giới <strong>Luyện Đạo</strong> hoặc thu phục chủng tộc dị nhân <strong>Người Lông</strong> làm việc trong Tiên Khiếu sẽ giúp gia tăng tỉ lệ thành công đáng kể.
          </p>
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
              Phiên bản v0.7.0 giới thiệu ba tính năng lớn nhằm tăng trải nghiệm và thu hút người chơi: Hệ thống <strong>Mở đá</strong>, <strong>Mã Quà (Gift Code)</strong> và <strong>Thành Tựu & Danh Hiệu</strong>.
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
              Một số thành tựu cấp cao (từ Tứ Chuyển trở lên, 10 kiếp, 50 Cổ, 1.000 ô địa đồ, 1 triệu Nguyên Thạch, 10.000 Nhân Khí, Phá Vách 100%) mở khóa <strong>Khung Frame</strong> và <strong>Danh Hiệu</strong>. Có thể trang bị danh hiệu và frame tại tab Thành Tựu — danh hiệu sẽ hiển thị cạnh tên trong khung chat.
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
            <li><strong>Sửa Speed:</strong> Tốc độ từ Growth và Passive giờ ảnh hưởng thứ tự đánh trong combat (trước bị ignore).</li>
            <li><strong>Sửa MaxHP:</strong> Sinh Lực tối đa không còn bị ghi đè cứng ở 100. Sinh Lực từ Growth và Passive giờ hoạt động đúng.</li>
            <li><strong>Growth cho legacy Cổ:</strong> Worker tự tạo EquippedAt cho Cổ đã equip trước khi feature growth ra mắt.</li>
            <li><strong>Auto-Combat loadout:</strong> Chỉ hiện Cổ có Active Skill. Passive + Growth auto-apply từ toàn bộ Cổ đang trang bị.</li>
            <li><strong>Gift Code mới:</strong> <span className="font-mono text-emerald-400">CHODAUGIA</span>, <span className="font-mono text-emerald-400">PHUONGNGUYEN</span>, <span className="font-mono text-amber-400">BACHNGUNGBANG</span>. Nhập tại tab Phúc Lợi trong Không Khiếu.</li>
          </ul>
        </div>
      </div>
    )
  }
];

export function getWikiPage(slug: string): WikiPage | undefined {
  return wikiPages.find(page => page.id === slug);
}
