import React from "react";
import {
  Sparkles,
  Coins,
  Swords,
  AlertTriangle,
  Compass,
  Flame,
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
                Xây dựng hệ thống phàm nhân cơ bản, trùng kích vách khiếu, các cổ trùng rank 1-5, thương đội Caravan, lò luyện cổ và chợ đấu giá Bảo Hoàng Thiên.
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
                Hoàn thiện cơ chế <strong>Thăng Tiên (Cấp 5 đỉnh phong thăng lên Cấp 6)</strong> và <strong>Độ Kiếp (Cổ Tiên vượt Địa Tai, Thiên Kiếp)</strong>. Xây dựng sinh thái Tiên Khiếu độc lập, chăn nuôi, trồng trọt và khai thác dị nhân.
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
                  <strong>Thăng Tiên theo Nhân Khí:</strong> Tỉ lệ thành công tỉ lệ thuận với Nhân Khí (0 → 0%, 100.000 → chắc chắn thành công). Thành công lên Lục Chuyển Cổ Tiên, nhận thưởng và khai mở Tiên Khiếu — cấp bậc Tiên Khiếu được định đoạt bởi Nhân Khí lúc thăng tiên. Thất bại: Nhân Khí về 0, tu vi phế bỏ rơi về Rank 1.
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
              Tiền tệ phổ biến của phàm nhân (Cấp 1-5). Chứa chân nguyên tự nhiên để hồi phục năng lượng cho Cổ sư và là nguyên liệu bắt buộc để kích hoạt/tăng hiệu suất trùng kích vách khiếu.
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
    keywords: ["nhân vật", "character", "hồ sơ", "profile", "chỉ số", "linh căn", "kỹ năng", "thông thạo"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Hồ sơ nhân vật chứa toàn bộ thông tin tu vi, không khiếu, các chỉ số cốt lõi và các Body Mods vĩnh viễn tích lũy từ dung hợp Cổ trùng. Nhấp vào các thẻ bên dưới để xem hướng dẫn chi tiết từng mục:
        </p>
      </div>
    ),
    cards: [
      {
        title: "Chỉ số",
        targetId: "gameplay-character-chiso",
        description: "Các chỉ số cơ bản của Cổ sư phàm nhân và Cổ tiên."
      },
      {
        title: "Thuộc tính & Áp lực",
        targetId: "gameplay-character-thuoctinh",
        description: "Tình trạng Không Khiếu, Áp lực vách khiếu và hình phạt."
      },
      {
        title: "Linh Căn (Tư chất)",
        targetId: "gameplay-character-linhcan",
        description: "Hệ thống tư chất không khiếu: Giáp, Ất, Bính, Đinh."
      },
      {
        title: "Thông thạo (Cảnh giới)",
        targetId: "gameplay-character-thongthao",
        description: "Cảnh giới lưu phái: Đại sư, Tông sư, Đại tông sư."
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
    keywords: ["chỉ số", "stats", "hp", "mana", "chân nguyên", "tu vi", "level", "rank", "subrank"],
    content: (
      <div className="space-y-6">
        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Các chỉ số tu luyện</h3>

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
                  <td className="p-3">Đại cảnh giới tu vi của nhân vật, từ Cấp 1 đến Cấp 9.</td>
                  <td className="p-3">Mở khóa tính năng mới (Cấp 6 mở Tiên Khiếu, v.v.).</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Sub-Rank</td>
                  <td className="p-3">Tiểu cảnh giới: Sơ Giai, Trung Giai, Cao Giai, Đỉnh Phong.</td>
                  <td className="p-3">Mốc để trùng kích vách khiếu đột phá lên cấp mới.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Sinh Lực</td>
                  <td className="p-3">Sức chịu đựng của cơ thể. Phàm nhân mặc định cố định ở mức rất thấp (100 HP).</td>
                  <td className="p-3">Nếu tụt về 0 trong chiến đấu, bạn sẽ bị trọng thương hoặc chết vĩnh viễn.</td>
                </tr>
                <tr>
                  <td className="p-3 font-bold text-[var(--text-primary)]">Chân Nguyên / Tiên Nguyên</td>
                  <td className="p-3">Năng lượng trong không khiếu. Dùng để kích hoạt Cổ trùng ra chiêu.</td>
                  <td className="p-3">Tiêu hao liên tục trong chiến đấu và hồi phục tự động theo tư chất.</td>
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
    keywords: ["stress", "áp lực", "vách khiếu", "không khiếu", "aperture stress", "nứt", "rớt cấp", "tịnh thủy"],
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
            <li>Mỗi lần click nút <strong>Trùng Kích Vách Khiếu</strong>, thanh áp lực tăng thêm <strong>20%</strong>.</li>
            <li>Thanh áp lực tự động giảm dần theo thời gian thực (ví dụ: giảm 10% mỗi giờ đời thật).</li>
            <li>Nếu áp lực vượt mốc <strong>80%</strong>, tỉ lệ trùng kích đột phá thành công bị suy giảm nghiêm trọng.</li>
            <li>Nếu áp lực chạm mốc <strong>100%</strong>, Không khiếu bị nứt: Nhân vật bị **mất vĩnh viễn 5% Max Mana** và có nguy cơ <strong>rớt cấp tu vi hiện tại</strong>!</li>
          </ul>
        </div>
      </div>
    )
  },
  {
    id: "gameplay-character-linhcan",
    title: "Linh Căn & Tư Chất Không Khiếu",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["tư chất", "linh căn", "aptitude", "giáp", "ất", "bính", "đinh", "hệ số", "multiplier"],
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
              Hệ số tu luyện: <strong>1.4x</strong>. Chân Nguyên dồi dào, vách khiếu vững chắc nhất.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-cyan-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-cyan-400">Ất (B)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Trung đẳng</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>1.1x</strong>. Tốc độ đột phá ổn định, dễ dàng vượt qua phàm nhân.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-amber-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-amber-500">Bính (C)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Hạ đẳng</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>0.85x</strong>. Cần nỗ lực nhiều hoặc dùng ngoại lực hỗ trợ.
            </p>
          </div>

          <div className="bg-[var(--bg-card)] border border-rose-500/20 rounded-2xl p-4 text-center">
            <span className="text-2xl font-bold text-rose-500">Đinh (D)</span>
            <h4 className="text-xs text-[var(--text-primary)] mt-2 font-bold">Tư chất Cực hạ</h4>
            <p className="text-[10px] text-[var(--text-muted)] mt-1 leading-relaxed">
              Hệ số tu luyện: <strong>0.6x</strong>. Vô cùng khó khăn. Khuyên dùng Tửu Trùng và Xá Lợi Cổ để tu luyện.
            </p>
          </div>
        </div>
      </div>
    )
  },
  {
    id: "gameplay-character-thongthao",
    title: "Thông Thạo & Cảnh Giới Lưu Phái",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["cảnh giới", "thông thạo", "attainment", "đại sư", "tông sư", "đại tông sư", "vô thượng", "chân ý", "master", "grandmaster"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Cảnh giới lưu phái thể hiện sự thấu hiểu của linh hồn bạn đối với quy luật vạn vật thế gian. Tích lũy cảnh giới lưu phái thông qua việc rèn luyện sát chiêu chiến đấu hoặc hấp thu Chân Ý khi thám hiểm Mộng Cảnh.
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
      </div>
    )
  },
  {
    id: "gameplay-character-kynang",
    title: "Kỹ năng & Sát Chiêu",
    category: "gameplay",
    subCategory: "gameplay-character",
    keywords: ["kỹ năng", "sát chiêu", "combo", "active", "passive", "body mods", "killer moves"],
    content: (
      <div className="space-y-6">
        <p className="text-xs text-[var(--text-secondary)] leading-relaxed">
          Trong game, Cổ sư không học kỹ năng phép thuật trực tiếp. Thay vào đó, sức mạnh chiến đấu của bạn phụ thuộc vào việc kết hợp các con Cổ trùng khác nhau (Sát chiêu) hoặc dùng Cổ trùng thụ động cải tạo thân thể.
        </p>

        <div className="space-y-4">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Sát Chiêu Phối Hợp</h3>
          <p className="text-xs text-[var(--text-muted)] leading-relaxed">
            Bằng cách trang bị và kích hoạt đồng thời một nhóm Cổ trùng bổ trợ nhau trong lượt đánh, bạn sẽ tạo ra sát chiêu cộng hưởng sức mạnh khủng khiếp:
          </p>
          <div className="p-4 rounded-2xl bg-[var(--bg-card)] border border-[var(--border)] text-xs space-y-2 text-[var(--text-secondary)] leading-relaxed">
            <li>💥 <strong>Lực Đạo combo:</strong> Trang bị đồng thời <strong>Toàn Lực Ứng Phó Cổ</strong> (kích hoạt 100% thú ảnh) và <strong>Khổ Lực Cổ</strong> (tăng sát thương dựa trên HP đã mất) để tối đa lượng damage bộc phát.</li>
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
          Cổ trùng được chia làm 9 Chuyển. Cấp 1-5 là Phàm Cổ, Cấp 6-9 là Tiên Cổ (độc nhất vô nhị). Chúng được chia thành các nhóm chính: Công kích, Phòng ngự, Trinh sát, Trị liệu, Phụ trợ và Di động.
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
              <li><strong>Thành tựu chiến đấu:</strong> Tân Thủ Nhập Môn, các mốc Rank 2-9.</li>
              <li><strong>Thành tựu độ kiếp:</strong> Vượt 1, 5, 10 kiếp nạn.</li>
              <li><strong>Thành tựu Cổ trùng:</strong> Sở hữu 10, 25, 50 loại Cổ khác nhau.</li>
              <li><strong>Thành tựu khám phá:</strong> Khám phá 100, 500, 1.000 ô địa đồ.</li>
              <li><strong>Thành tựu kinh tế:</strong> Tích lũy 1.000.000 Nguyên Thạch.</li>
              <li><strong>Thành tựu tu luyện:</strong> Nhân Khí, Hồn Phách, Phá Vách.</li>
            </ul>
            <p>
              Một số thành tựu cấp cao (từ Rank 4 trở lên, 10 kiếp, 50 Cổ, 1.000 ô địa đồ, 1 triệu Nguyên Thạch, 10.000 Nhân Khí, Phá Vách 100%) mở khóa <strong>Khung Frame</strong> và <strong>Danh Hiệu</strong>. Có thể trang bị danh hiệu và frame tại tab Thành Tựu — danh hiệu sẽ hiển thị cạnh tên trong khung chat.
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
            <li><strong>Sửa MaxHP:</strong> Không còn bị ghi đè cứng ở 100. Growth HP + Passive HP giờ hoạt động đúng.</li>
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
