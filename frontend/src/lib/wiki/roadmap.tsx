import React from "react";
import { Sparkles } from "lucide-react";
import { WikiPage } from "./types";

export const roadmapPages: WikiPage[] = [
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
                <span>Phase 6 (Cơ Chế Sát Chiêu & Xung Đột Đạo Ngân)</span>
                <span className="px-1.5 py-0.5 rounded bg-indigo-950 text-indigo-400 font-mono text-[9px]">Đang phát triển</span>
              </h4>
              <p className="text-[var(--text-secondary)] mt-1 leading-relaxed">
                Hoàn thiện cơ chế <strong>Sát Chiêu</strong> và hệ thống <strong>Xung Đột Đạo Ngân (Dao Mark Conflict)</strong> cùng 10 nhóm Đạo Ngân, cơ chế điều hòa theo cảnh giới lưu phái.
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
                <span>Phiên Bản v0.8.5 (Hiện tại)</span>
                <span className="px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 font-mono text-[9px]">
                  Mới nhất
                </span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Cơ Chế Xung Đột Đạo Ngân (Dao Mark Conflict):</strong> Phân chia 49 Lưu phái thành 10 Nhóm Đạo (Dao Groups). Đạo ngân ngoại hệ không tương thích sẽ tạo ra Xung Đột làm suy giảm sát thương nếu Cảnh giới Lưu phái chưa đủ cao.
                </li>
                <li>
                  <strong>Hệ Thống Điều Hòa Cảnh Giới:</strong> Chân Ý được tính riêng theo lưu phái đang thi triển, điều hòa xung đột từ 10% ở Đại Sư đến 100% ở Vô Thượng Đại Tông Sư. Đạo Ngân ngoại hệ vẫn có thể gây phản phệ khi dùng một phái chưa đủ Chân Ý.
                </li>
                <li>
                  <strong>Phòng Thủ Đạo Ngân Tự Nhiên (Natural Dao Resistance):</strong> Đạo ngân của Cổ Tiên tạo màng kháng tự nhiên (lên tới 30%) trước các đòn đánh không tương thích của đối thủ.
                </li>
                <li>
                  <strong>Thiên Đạo Tiêu Hủy (Void Destruction):</strong> Tăng hao hụt tử đạo lên 40% (tiền, nguyên liệu) và 30% (Cổ trùng, bí phương). 40% trong số đó bị tiêu hủy vĩnh viễn khỏi server; 60% còn lại rơi bãi đất tồn tại 10 phút.
                </li>
                <li>
                  <strong>Cơ chế Sát Khí & Safe Zone:</strong> Sát Khí &gt; 10 bị tước quyền bảo hộ trong Safe Zone/Hub. Sát Khí &gt; 5 có thể rơi Cổ đang trang bị khi chết (Cổ Bản Mệnh luôn được bảo toàn).
                </li>
                <li>
                  <strong>Miễn Chiến Hồi Sinh:</strong> Nạn nhân bị tấn công và thua trận nhận 10 phút Miễn Chiến hồi sinh.
                </li>
              </ul>
            </div>

            <div className="relative">
              <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
              <h4 className="font-serif font-bold text-[var(--text-primary)] flex items-center gap-2">
                <span>Phiên Bản v0.8.0</span>
              </h4>

              <ul className="list-disc list-inside text-[var(--text-secondary)] mt-1.5 space-y-1.5 leading-relaxed">
                <li>
                  <strong>Tối ưu lưu trữ:</strong> Kho Cổ trùng được lưu trữ gọn gàng hơn, mở kho, trang bị và cho ăn nhanh mượt hơn.
                </li>
                <li>
                  <strong>Cho ăn Cổ trùng ổn định:</strong> Độ no giảm đều theo thời gian, tránh lag khi máy chủ xử lý toàn bộ người chơi cùng lúc.
                </li>
                <li>
                  <strong>Đạo Ngân vận hành tốt hơn:</strong> Đạo Ngân hiển thị và áp dụng hiệu ứng chính xác cho cả đồng đội lẫn địch trong chiến đấu tự động.
                </li>
                <li>
                  <strong>Thăng Tiên theo Nhân Khí:</strong> Tỉ lệ thành công = Xác suất gốc Tư chất (Giáp: 20%, Ất/Bính/Đinh: 5%) + Bonus Nhân Khí (+20% mỗi 100k Nhân Khí, cap 90%).
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
                  <strong>Bảng Xếp Hạng:</strong> Hệ thống BXH tổng hợp với 4 tiêu chí: Tu Luyện, Nguyên Thạch, Cổ Trùng (cấp cao nhất) và Tiên Nguyên.
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
                  <strong>Mở đá:</strong> Hệ thống quay thưởng ngẫu nhiên nhận Nguyên Thạch, Tiên Nguyên Thạch, Thọ Nguyên, Cổ trùng và Đạo Ấn.
                </li>
                <li>
                  <strong>Mã Quà (Gift Code):</strong> Nhập mã tại tab "Phúc Lợi" trong Không Khiếu để nhận thưởng.
                </li>
                <li>
                  <strong>Thành Tựu & Danh Hiệu:</strong> Hệ thống thành tựu với 23 mốc và danh hiệu hiển thị khung chat.
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    ),
  },
];
