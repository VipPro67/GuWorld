import React, { useState } from "react";
import {
  Search,
  Sparkles,
  HelpCircle,
  ChevronDown,
  ChevronRight,
  BookOpen,
  User,
  Shield,
  Coins,
  Compass,
  Swords,
  Flame,
  Crown,
  BookMarked,
  ArrowRight,
  Info,
  AlertTriangle,
  Heart
} from "lucide-react";

interface WikiPage {
  id: string;
  title: string;
  category: string;
  subCategory?: string;
  keywords: string[];
  content: React.ReactNode;
  cards?: { title: string; targetId: string; description: string }[];
}

export const WikiView: React.FC = () => {
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedPageId, setSelectedPageId] = useState("roadmap");
  const [expandedCategories, setExpandedCategories] = useState<Record<string, boolean>>({
    "gameplay": true,
    "gameplay-character": true,
  });

  const toggleCategory = (catId: string) => {
    setExpandedCategories(prev => ({
      ...prev,
      [catId]: !prev[catId]
    }));
  };

  const pages: WikiPage[] = [
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
              <p className="text-xs text-zinc-400 mt-1 leading-relaxed">
                GuWorld hướng tới việc tái hiện trọn vẹn sự tàn khốc, triết lý và tư duy sinh tồn trong bối cảnh tiểu thuyết <strong>Cổ Chân Nhân</strong>. Lộ trình phát triển được phân chia theo các giai đoạn nâng cấp đồng bộ cả hệ thống dữ liệu lẫn trải nghiệm nghe nhìn của người chơi.
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Các Phase cập nhật</h3>
            
            <div className="relative border-l border-zinc-800 pl-5 ml-2.5 space-y-6 text-xs">
              <div className="relative">
                <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-emerald-500 border-2 border-zinc-950" />
                <h4 className="font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <span>Giai Đoạn 4 & 5 (Hiện tại)</span>
                  <span className="px-1.5 py-0.5 rounded bg-emerald-950 text-emerald-400 font-mono text-[9px]">Hoàn thành</span>
                </h4>
                <p className="text-zinc-500 mt-1 leading-relaxed">
                  Xây dựng hệ thống phàm nhân cơ bản, trùng kích vách khiếu, các cổ trùng rank 1-5, thương đội caravan, lò luyện cổ và hệ thống đấu giá Bảo Hoàng Thiên.
                </p>
              </div>

              <div className="relative">
                <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-indigo-500 border-2 border-zinc-950" />
                <h4 className="font-serif font-bold text-zinc-200 flex items-center gap-2">
                  <span>Phase 6 & 7 (Sắp tới)</span>
                  <span className="px-1.5 py-0.5 rounded bg-indigo-950 text-indigo-400 font-mono text-[9px]">Đang phát triển</span>
                </h4>
                <p className="text-zinc-400 mt-1 leading-relaxed">
                  Hoàn thiện cơ chế <strong>Thăng Tiên (Cấp 5 đỉnh phong thăng lên Cấp 6)</strong> và <strong>Độ Kiếp (Cổ Tiên vượt Địa Tai, Thiên Kiếp)</strong>. Xây dựng sinh thái Tiên Khiếu, trồng trọt chăn nuôi và khai thác dị nhân.
                </p>
              </div>

              <div className="relative">
                <div className="absolute -left-[26px] top-1.5 w-3 h-3 rounded-full bg-zinc-700 border-2 border-zinc-950" />
                <h4 className="font-serif font-bold text-zinc-400">Phase 8 & 9</h4>
                <p className="text-zinc-500 mt-1 leading-relaxed">
                  Bổ sung phó bản <strong>Mộng Cảnh</strong> đầy đủ, nâng cấp hệ thống <strong>Hồn Phách</strong> (Tráng Hồn ở núi Đãng Hồn, Luyện Hồn ở cốc Lạc Phách). Mở rộng PvP đấu trường liên lục địa.
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Trong thế giới Cổ sư, tiền tệ không chỉ dùng để mua bán mà bản thân chúng còn là kết tinh năng lượng tinh khiết giúp hồi phục năng lượng và chế tạo vật phẩm.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4 flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-yellow-950/50 flex items-center justify-center border border-yellow-500/20">
                  <Coins className="w-4 h-4 text-yellow-500" />
                </div>
                <h4 className="text-sm font-serif font-bold text-zinc-200">Nguyên Thạch</h4>
              </div>
              <p className="text-[11px] text-zinc-500 leading-relaxed mt-1">
                Tiền tệ phổ biến của phàm nhân (Cấp 1-5). Chứa chân nguyên tự nhiên để hồi phục năng lượng cho Cổ sư và là nguyên liệu bắt buộc để kích hoạt/tăng hiệu suất trùng kích vách khiếu.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4 flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-purple-950/50 flex items-center justify-center border border-purple-500/20">
                  <Coins className="w-4 h-4 text-purple-400" />
                </div>
                <h4 className="text-sm font-serif font-bold text-zinc-200">Tiên Nguyên Thạch</h4>
              </div>
              <p className="text-[11px] text-zinc-500 leading-relaxed mt-1">
                Đơn vị tiền tệ tối cao của giới Cổ Tiên. Dùng để giao dịch các vật phẩm hiếm bậc nhất tại Chợ Bảo Hoàng Thiên, hoặc trực tiếp nghiền nát để bổ sung lượng lớn Tiên Nguyên chiến đấu.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-zinc-900 rounded-2xl p-4 flex flex-col gap-2">
              <div className="flex items-center gap-2">
                <div className="w-8 h-8 rounded-lg bg-rose-950/50 flex items-center justify-center border border-rose-500/20">
                  <Swords className="w-4 h-4 text-rose-500" />
                </div>
                <h4 className="text-sm font-serif font-bold text-zinc-200">Chiến Công</h4>
              </div>
              <p className="text-[11px] text-zinc-500 leading-relaxed mt-1">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Thay thế chế độ gia tộc dựa trên huyết thống, Tông môn là tổ chức thế lực tuyển chọn đệ tử dựa trên năng lực, phẩm hạnh và sự trung thành, vận hành bằng hệ thống sư đồ và điểm cống hiến nghiêm ngặt.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">1. Phân cấp đệ tử & Khảo hạch</h3>
            <p className="text-xs text-zinc-500 leading-relaxed">
              Bạn sẽ bắt đầu là một Đệ tử ngoại môn và leo dần lên các bậc cao hơn. Mỗi bậc đều giới hạn số lượng người chơi tối đa và yêu cầu vượt qua các kỳ thi định kỳ:
            </p>
            <div className="p-4 rounded-2xl bg-zinc-900/50 border border-zinc-900 space-y-3 text-xs">
              <div className="flex justify-between border-b border-zinc-800 pb-2">
                <span className="font-bold text-zinc-300">Tiểu Khảo</span>
                <span className="text-zinc-500">Mỗi 3 ngày thực tế • Điều kiện thăng cấp Nội Môn</span>
              </div>
              <div className="flex justify-between border-b border-zinc-800 pb-2">
                <span className="font-bold text-zinc-300">Trung Khảo</span>
                <span className="text-zinc-500">Mỗi 8 ngày thực tế • Quyết đấu chọn đệ tử Tinh Anh</span>
              </div>
              <div className="flex justify-between pb-1">
                <span className="font-bold text-zinc-300">Đại Khảo</span>
                <span className="text-zinc-500">Mỗi 15 ngày thực tế • Sinh tồn chọn đệ tử Chân Truyền</span>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">2. Điểm Cống Hiến & Tàng Kinh Các</h3>
            <p className="text-xs text-zinc-400 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
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
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Các chỉ số tu luyện</h3>
            
            <div className="overflow-hidden rounded-2xl border border-zinc-900">
              <table className="w-full text-left text-xs font-mono">
                <thead className="bg-zinc-950 text-zinc-400 border-b border-zinc-900">
                  <tr>
                    <th className="p-3">Tên chỉ số</th>
                    <th className="p-3">Mô tả chi tiết</th>
                    <th className="p-3">Tác dụng</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-900 text-zinc-300 bg-zinc-950/20">
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Cảnh Giới</td>
                    <td className="p-3">Đại cảnh giới tu vi của nhân vật, từ Cấp 1 đến Cấp 9.</td>
                    <td className="p-3">Mở khóa tính năng mới (Cấp 6 mở Tiên Khiếu, v.v.).</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Sub-Rank</td>
                    <td className="p-3">Tiểu cảnh giới: Sơ Giai, Trung Giai, Cao Giai, Đỉnh Phong.</td>
                    <td className="p-3">Mốc để trùng kích vách khiếu đột phá lên cấp mới.</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Sinh Lực</td>
                    <td className="p-3">Sức chịu đựng của cơ thể. Phàm nhân mặc định cố định ở mức rất thấp (100 HP).</td>
                    <td className="p-3">Nếu tụt về 0 trong chiến đấu, bạn sẽ bị trọng thương hoặc chết vĩnh viễn.</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Chân Nguyên / Tiên Nguyên</td>
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
              <p className="text-xs text-zinc-400 mt-1 leading-relaxed">
                Việc ép chân nguyên liên tục xông phá vách khiếu sẽ gây tổn thương nặng nề lên Không khiếu. Bạn cần chú ý chỉ số <strong>Aperture Stress</strong> để tránh hậu quả đáng tiếc.
              </p>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Hệ thống áp lực</h3>
            <ul className="list-disc list-inside text-xs text-zinc-400 space-y-2 leading-relaxed">
              <li>Mỗi lần click nút <strong>Trùng Kích Vách Khiếu</strong>, thanh áp lực tăng thêm <strong>20%</strong>.</li>
              <li>Thanh áp lực tự động giảm dần theo thời gian thực (ví dụ: giảm 10% mỗi giờ đời thật).</li>
              <li>Nếu áp lực vượt mốc <strong>80%</strong>, tỉ lệ trùng kích đột phá thành công bị suy giảm nghiêm trọng.</li>
              <li>Nếu áp lực chạm mốc <strong>100%</strong>, Không khiếu bị nứt: Nhân vật bị **mất vĩnh viễn 5% Max Mana** và có nguy cơ **rớt cấp tu vi hiện tại**!</li>
            </ul>
            <div className="p-4 rounded-2xl bg-zinc-900/50 border border-zinc-900 text-xs leading-relaxed text-zinc-400">
              💡 <strong>Mẹo:</strong> Sử dụng vật phẩm tiêu hao <strong>Tịnh Thủy Cổ</strong> để lập tức thanh lọc, đưa thanh Áp lực vách khiếu về 0% ngay tức khắc.
            </div>
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Khi đăng ký nhân vật, bạn sẽ được thức tỉnh không khiếu ngẫu nhiên với một trong bốn phẩm cấp tư chất (Linh Căn). Tư chất quyết định lượng chân nguyên lưu trữ tối đa, tốc độ hồi phục và đặc biệt là hiệu suất phá vách khiếu tu luyện.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div className="bg-zinc-900/40 border border-emerald-500/20 rounded-2xl p-4 text-center">
              <span className="text-2xl font-bold text-emerald-400">Giáp (A)</span>
              <h4 className="text-xs text-zinc-300 mt-2 font-bold">Tư chất Thượng đẳng</h4>
              <p className="text-[10px] text-zinc-500 mt-1 leading-relaxed">
                Hệ số tu luyện: <strong>1.4x</strong>. Chân Nguyên dồi dào, vách khiếu vững chắc nhất.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-cyan-500/20 rounded-2xl p-4 text-center">
              <span className="text-2xl font-bold text-cyan-400">Ất (B)</span>
              <h4 className="text-xs text-zinc-300 mt-2 font-bold">Tư chất Trung đẳng</h4>
              <p className="text-[10px] text-zinc-500 mt-1 leading-relaxed">
                Hệ số tu luyện: <strong>1.1x</strong>. Tốc độ đột phá ổn định, dễ dàng vượt qua phàm nhân.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-amber-500/20 rounded-2xl p-4 text-center">
              <span className="text-2xl font-bold text-amber-500">Bính (C)</span>
              <h4 className="text-xs text-zinc-300 mt-2 font-bold">Tư chất Hạ đẳng</h4>
              <p className="text-[10px] text-zinc-500 mt-1 leading-relaxed">
                Hệ số tu luyện: <strong>0.85x</strong>. Cần nỗ lực nhiều hoặc dùng ngoại lực hỗ trợ.
              </p>
            </div>

            <div className="bg-zinc-900/40 border border-rose-500/20 rounded-2xl p-4 text-center">
              <span className="text-2xl font-bold text-rose-500">Đinh (D)</span>
              <h4 className="text-xs text-zinc-300 mt-2 font-bold">Tư chất Cực hạ</h4>
              <p className="text-[10px] text-zinc-500 mt-1 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Cảnh giới lưu phái thể hiện sự thấu hiểu của linh hồn bạn đối với quy luật vạn vật thế gian. Tích lũy cảnh giới lưu phái thông qua việc rèn luyện sát chiêu chiến đấu hoặc hấp thu Chân Ý khi thám hiểm Mộng Cảnh.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Các mốc cảnh giới lưu phái</h3>
            
            <div className="space-y-3">
              <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 text-xs">
                <h4 className="font-bold text-zinc-200 flex items-center gap-2">
                  <span>1. Chuẩn Đại Sư / Đại Sư</span>
                </h4>
                <p className="text-zinc-500 mt-1 leading-relaxed">
                  Trực giác siêu nhạy bén. Trong chiến đấu, nếu đối phương sử dụng sát chiêu thuộc hệ phái bạn là Đại Sư, giao diện sẽ hiển thị gợi ý né tránh hoặc chỉ ra nhược điểm của chiêu đó.
                </p>
              </div>

              <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 text-xs">
                <h4 className="font-bold text-indigo-300 flex items-center gap-2">
                  <span>2. Chuẩn Tông Sư / Tông Sư</span>
                </h4>
                <p className="text-zinc-500 mt-1 leading-relaxed">
                  Đạt khả năng &quot;Loại suy&quot;. Loại bỏ hoàn toàn hình phạt mâu thuẫn giảm nửa sát thương khi sử dụng chéo hệ Cổ trùng (Ví dụ: Cổ sư Lực đạo dùng Cổ Thủy đạo vẫn nhận 100% hiệu năng).
                </p>
              </div>

              <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 text-xs">
                <h4 className="font-bold text-yellow-500 flex items-center gap-2">
                  <span>3. Chuẩn Đại Tông Sư / Đại Tông Sư</span>
                </h4>
                <p className="text-zinc-500 mt-1 leading-relaxed">
                  Cộng hưởng hoàn hảo với Đạo ngân của thiên nhiên. Tự động nhận thêm 50% sát thương/phòng ngự khi chiến đấu tại địa hình tương đồng hệ phái của bạn mà không cần kích hoạt cổ buff.
                </p>
              </div>

              <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 text-xs">
                <h4 className="font-bold text-red-400 flex items-center gap-2">
                  <span>4. Vô Thượng Đại Tông Sư</span>
                </h4>
                <p className="text-zinc-500 mt-1 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Trong game, Cổ sư không học kỹ năng phép thuật trực tiếp. Thay vào đó, sức mạnh chiến đấu của bạn phụ thuộc vào việc kết hợp các con Cổ trùng khác nhau (Sát chiêu) hoặc dùng Cổ trùng thụ động cải tạo thân thể.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Sát Chiêu Phối Hợp</h3>
            <p className="text-xs text-zinc-500 leading-relaxed">
              Bằng cách trang bị và kích hoạt đồng thời một nhóm Cổ trùng bổ trợ nhau trong lượt đánh, bạn sẽ tạo ra sát chiêu cộng hưởng sức mạnh khủng khiếp:
            </p>
            <div className="p-4 rounded-2xl bg-zinc-900/50 border border-zinc-900 text-xs space-y-2 text-zinc-400 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Cổ trùng được chia làm 9 Chuyển. Cấp 1-5 là Phàm Cổ, Cấp 6-9 là Tiên Cổ (độc nhất vô nhị). Chúng được chia thành các nhóm chính: Công kích, Phòng ngự, Trinh sát, Trị liệu, Phụ trợ và Di động.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Hệ thống nuôi Cổ trùng</h3>
            <div className="p-4 rounded-2xl bg-rose-950/20 border border-rose-500/10 text-xs text-rose-300 leading-relaxed">
              ⚠️ <strong>Lưu ý cực kỳ quan trọng:</strong> Mỗi con Cổ trang bị đều có thanh độ no. Độ no giảm dần sau mỗi lượt chiến đấu hoặc theo thời gian thực tế. Nếu độ no về 0, Cổ sẽ rơi vào trạng thái <strong>Hấp Hối</strong> và sẽ <strong>chết vĩnh viễn</strong> nếu không được cho ăn kịp thời!
            </div>
            
            <p className="text-xs text-zinc-500 leading-relaxed mt-2">
              Bạn cần thu thập thức ăn tương ứng của từng hệ để nuôi dưỡng chúng:
            </p>
            <div className="overflow-hidden rounded-2xl border border-zinc-900">
              <table className="w-full text-left text-xs font-mono">
                <thead className="bg-zinc-950 text-zinc-400 border-b border-zinc-900">
                  <tr>
                    <th className="p-3">Tên Cổ Trùng</th>
                    <th className="p-3">Thức ăn yêu cầu</th>
                    <th className="p-3">Nguồn gốc thức ăn</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-900 text-zinc-300 bg-zinc-950/20">
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Nguyệt Quang Cổ</td>
                    <td className="p-3">Cánh Hoa Nguyệt Lan</td>
                    <td className="p-3">Trồng tại Tiên Khiếu hoặc mua tại Tiệm Caravan</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Tửu Trùng</td>
                    <td className="p-3">Rượu ngon (Thanh Trúc, Hầu Nhi...)</td>
                    <td className="p-3">Sản xuất từ quầy chưng cất, dã ngoại</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Bạch Thỉ Cổ / Hắc Thỉ Cổ</td>
                    <td className="p-3">Thịt lợn rừng</td>
                    <td className="p-3">Săn bắn dã ngoại hoang dã</td>
                  </tr>
                  <tr>
                    <td className="p-3 font-bold text-zinc-200">Tiên Cổ Thái Độ</td>
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Các trận đấu trong game áp dụng cơ chế sắp xếp đội hình trên lưới 3x3 đối lập nhau. Sát thương nhận vào sẽ ưu tiên trừ trực tiếp vào lớp Cổ phòng ngự của bạn.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">1. Vị trí và tầm đánh AoE</h3>
            <p className="text-xs text-zinc-500 leading-relaxed">
              Bạn có thể xếp quân ở Tiền tuyến (X=1), Trung tuyến (X=2) hoặc Hậu phương (X=3). Trừ khi kỹ năng có tầm đánh đặc thù, các sát chiêu đơn mục tiêu bắt buộc phải đánh trúng và dọn sạch hàng Tiền tuyến của đối phương trước mới có thể đánh vào hàng sau.
            </p>
            <div className="p-4 rounded-xl bg-zinc-900/40 border border-zinc-900 text-xs space-y-2 text-zinc-400">
              <li>💥 <strong>Đánh Ngang:</strong> Quét sạch 3 ô nằm trên cùng một hàng ngang X.</li>
              <li>💥 <strong>Đánh Dọc:</strong> Xuyên từ trước ra sau trên cùng một cột dọc Y.</li>
              <li>💥 <strong>Toàn bản đồ:</strong> Tác động đồng thời lên cả 9 ô của kẻ địch.</li>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">2. Khiên ảo & Tốc độ</h3>
            <p className="text-xs text-zinc-400 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Luyện Cổ là con đường chế tạo Cổ trùng thông qua Lò Luyện Cổ nhân tạo hoặc dựa vào linh khí thiên địa.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Cơ chế Lò Luyện</h3>
            <p className="text-xs text-zinc-500 leading-relaxed">
              Mỗi con Cổ nâng cấp hoặc hợp thành đều cần một bản vẽ <strong>Cổ Phương</strong> cụ thể. Bạn cần đưa Cổ trùng nguyên liệu chính và các nguyên liệu phụ trợ (Hàn Ngọc, Địa Tinh, Quặng Sắt...) kèm theo lượng lớn Nguyên Thạch để kích hoạt.
            </p>
            <div className="p-4 rounded-2xl bg-amber-950/20 border border-amber-500/10 text-xs text-amber-300 leading-relaxed">
              ⚠️ <strong>Rủi ro luyện cổ:</strong> Mọi quá trình luyện cổ đều có tỷ lệ thất bại tự nhiên. Khi thất bại, bạn sẽ mất toàn bộ nguyên liệu phụ và có xác suất bị hư hỏng hoặc mất vĩnh viễn Cổ trùng nguyên liệu chính!
            </div>
            <p className="text-xs text-zinc-400 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Game cung cấp các hoạt động phiêu lưu ngoài bản đồ và các phó bản cốt truyện đặc thù nhằm thu thập tài nguyên cải thiện thực lực.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Các phó bản đặc thù</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
              <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 space-y-2">
                <h4 className="font-serif font-bold text-zinc-200 flex items-center gap-1.5">
                  <Compass className="w-4 h-4 text-emerald-400" />
                  <span>Truyền Thừa</span>
                </h4>
                <p className="text-zinc-500 leading-relaxed">
                  Phó bản solo giải đố dành cho phàm nhân. Bạn phải đối mặt với các cạm bẫy, câu đố chữ hoặc yêu cầu sử dụng đúng thuộc tính Cổ trùng nhất định để vượt qua.
                </p>
              </div>

              <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 space-y-2">
                <h4 className="font-serif font-bold text-zinc-200 flex items-center gap-1.5">
                  <Flame className="w-4 h-4 text-purple-400" />
                  <span>Mộng Cảnh</span>
                </h4>
                <p className="text-zinc-500 leading-relaxed">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Nô Đạo là lưu phái sử dụng linh hồn và các cổ trùng khống chế để hiệu lệnh, dẫn dắt đại quân hoang thú chiến đấu thay bản thân mình.
          </p>

          <div className="space-y-4">
            <h3 className="text-sm font-bold uppercase tracking-wider text-zinc-300 font-mono">Giới hạn khống chế & Hồn Phách</h3>
            <p className="text-xs text-zinc-500 leading-relaxed">
              Bạn không thể dẫn dắt một quân đội vô hạn. Sức chịu đựng tinh thần của bạn phụ thuộc hoàn toàn vào điểm số <strong>Nội tình Hồn Phách</strong>:
            </p>
            <div className="p-4 rounded-2xl bg-zinc-900/50 border border-zinc-900 text-xs space-y-3 text-zinc-400">
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
          <p className="text-xs text-zinc-400 leading-relaxed">
            Thế giới ngoài loài người còn tồn tại 10 chủng tộc dị nhân. Khi lên làm Cổ Tiên, bạn có thể chiêu mộ và thả các dị nhân vào Tiên Khiếu để làm việc sản xuất.
          </p>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 space-y-2 text-xs">
              <h4 className="font-bold text-zinc-200">Người Lông</h4>
              <p className="text-zinc-500 leading-relaxed">
                Thiên phú luyện Cổ bẩm sinh. Nuôi dưỡng Người Lông trong Tiên Khiếu giúp tăng mạnh tốc độ sản xuất và tăng tỉ lệ thành công khi luyện cổ trùng tự động.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 space-y-2 text-xs">
              <h4 className="font-bold text-zinc-200">Người Đá</h4>
              <p className="text-zinc-500 leading-relaxed">
                Thân hình cứng rắn, ngủ sâu nuôi dưỡng hồn phách. Rất thích hợp làm lao động đào quặng dưới lòng đất Tiên Khiếu, giúp tăng sản lượng khai thác quặng nguyên thạch.
              </p>
            </div>

            <div className="p-4 rounded-xl bg-zinc-900/30 border border-zinc-900 space-y-2 text-xs">
              <h4 className="font-bold text-zinc-200">Người Tuyết</h4>
              <p className="text-zinc-500 leading-relaxed">
                Tộc nhân sinh sống tại băng nguyên. Khóc ra Lệ Băng (tiên tài trân quý Băng đạo) dùng để luyện các loại Tiên Cổ hệ Băng Tuyết.
              </p>
            </div>
          </div>
        </div>
      )
    }
  ];

  // Helper to filter and search pages
  const filteredPages = pages.filter(p => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      p.title.toLowerCase().includes(query) ||
      p.keywords.some(k => k.includes(query))
    );
  });

  const currentPage = pages.find(p => p.id === selectedPageId) || pages[0];

  const handleCardClick = (targetId: string) => {
    setSelectedPageId(targetId);
    // Expand category if needed
    const targetPage = pages.find(p => p.id === targetId);
    if (targetPage && targetPage.category) {
      setExpandedCategories(prev => ({
        ...prev,
        [targetPage.category]: true
      }));
    }
  };

  return (
    <div className="w-full max-w-6xl mx-auto bg-zinc-950/70 border border-zinc-900 rounded-3xl overflow-hidden shadow-2xl backdrop-blur-xl flex flex-col md:flex-row h-[680px]">
      {/* 1. LEFT SIDEBAR */}
      <aside className="w-full md:w-64 border-r border-zinc-900 flex flex-col shrink-0 bg-zinc-950/80">
        {/* Search Input Box */}
        <div className="p-4 border-b border-zinc-900/60">
          <div className="relative flex items-center">
            <Search className="w-4 h-4 text-zinc-600 absolute left-3 pointer-events-none" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Tìm kiếm tài liệu..."
              className="w-full bg-zinc-900 border border-zinc-900 focus:border-zinc-800 rounded-xl pl-9 pr-3 py-2 text-xs text-zinc-100 placeholder-zinc-600 focus:outline-none transition-all font-mono"
            />
          </div>
        </div>

        {/* Sidebar Nav Items */}
        <nav className="flex-grow overflow-y-auto p-3 space-y-1 scrollbar-thin scrollbar-thumb-zinc-850">
          
          {/* Section: Lộ trình cập nhật */}
          <button
            onClick={() => setSelectedPageId("roadmap")}
            className={`w-full text-left px-3 py-2.5 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "roadmap"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <Sparkles className="w-3.5 h-3.5 shrink-0" />
            <span>Dự kiến cập nhật</span>
          </button>

          {/* Section: Game Play Dropdown */}
          <div className="space-y-0.5">
            <button
              onClick={() => toggleCategory("gameplay")}
              className="w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-bold text-zinc-500 hover:text-zinc-300 flex items-center justify-between cursor-pointer"
            >
              <span className="uppercase tracking-wider">Game Play</span>
              {expandedCategories["gameplay"] ? (
                <ChevronDown className="w-3.5 h-3.5" />
              ) : (
                <ChevronRight className="w-3.5 h-3.5" />
              )}
            </button>

            {expandedCategories["gameplay"] && (
              <div className="pl-3 border-l border-zinc-900/80 ml-2 space-y-0.5">
                <button
                  onClick={() => setSelectedPageId("gameplay-baothach")}
                  className={`w-full text-left px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                    selectedPageId === "gameplay-baothach"
                      ? "text-indigo-400 font-bold"
                      : "text-zinc-400 hover:text-zinc-200"
                  }`}
                >
                  Bảo Thạch
                </button>

                <button
                  onClick={() => setSelectedPageId("gameplay-tongmon")}
                  className={`w-full text-left px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                    selectedPageId === "gameplay-tongmon"
                      ? "text-indigo-400 font-bold"
                      : "text-zinc-400 hover:text-zinc-200"
                  }`}
                >
                  Tông Môn
                </button>

                {/* Sub Menu: Nhân vật */}
                <div className="space-y-0.5">
                  <button
                    onClick={() => {
                      setSelectedPageId("gameplay-character");
                      toggleCategory("gameplay-character");
                    }}
                    className={`w-full text-left px-3 py-1.5 rounded-lg text-xs font-semibold transition-all flex items-center justify-between cursor-pointer ${
                      selectedPageId.startsWith("gameplay-character")
                        ? "text-indigo-400 font-bold"
                        : "text-zinc-400 hover:text-zinc-200"
                    }`}
                  >
                    <span>Nhân vật</span>
                    {expandedCategories["gameplay-character"] ? (
                      <ChevronDown className="w-3 h-3" />
                    ) : (
                      <ChevronRight className="w-3 h-3" />
                    )}
                  </button>

                  {expandedCategories["gameplay-character"] && (
                    <div className="pl-3 border-l border-zinc-900 ml-2 space-y-0.5 text-[11px]">
                      {[
                        { id: "gameplay-character-chiso", label: "Chỉ số" },
                        { id: "gameplay-character-thuoctinh", label: "Thuộc tính" },
                        { id: "gameplay-character-linhcan", label: "Linh Căn" },
                        { id: "gameplay-character-thongthao", label: "Thông thạo" },
                        { id: "gameplay-character-kynang", label: "Kỹ năng" }
                      ].map(sub => (
                        <button
                          key={sub.id}
                          onClick={() => setSelectedPageId(sub.id)}
                          className={`w-full text-left px-3 py-1 rounded-md transition-all cursor-pointer ${
                            selectedPageId === sub.id
                              ? "bg-indigo-600/10 text-indigo-400 font-bold border-l-2 border-indigo-500 pl-2.5"
                              : "text-zinc-500 hover:text-zinc-300"
                          }`}
                        >
                          {sub.label}
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>

          {/* Section: Trang Bị */}
          <button
            onClick={() => setSelectedPageId("equipment")}
            className={`w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "equipment"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <User className="w-3.5 h-3.5 shrink-0" />
            <span>Trang bị</span>
          </button>

          {/* Section: Chiến Đấu */}
          <button
            onClick={() => setSelectedPageId("combat")}
            className={`w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "combat"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <Swords className="w-3.5 h-3.5 shrink-0" />
            <span>Chiến đấu</span>
          </button>

          {/* Section: Nghề */}
          <button
            onClick={() => setSelectedPageId("jobs")}
            className={`w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "jobs"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <Flame className="w-3.5 h-3.5 shrink-0" />
            <span>Nghề</span>
          </button>

          {/* Section: Hoạt Động */}
          <button
            onClick={() => setSelectedPageId("activities")}
            className={`w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "activities"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <Compass className="w-3.5 h-3.5 shrink-0" />
            <span>Hoạt động</span>
          </button>

          {/* Section: Linh Thú */}
          <button
            onClick={() => setSelectedPageId("beasts")}
            className={`w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "beasts"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <Crown className="w-3.5 h-3.5 shrink-0" />
            <span>Linh Thú</span>
          </button>

          {/* Section: Đồng Hành */}
          <button
            onClick={() => setSelectedPageId("companions")}
            className={`w-full text-left px-3 py-2 rounded-xl font-serif text-xs font-semibold transition-all flex items-center gap-2 cursor-pointer ${
              selectedPageId === "companions"
                ? "bg-indigo-600 text-white shadow-md shadow-indigo-600/15"
                : "text-zinc-400 hover:bg-zinc-900/40 hover:text-zinc-200"
            }`}
          >
            <Heart className="w-3.5 h-3.5 shrink-0" />
            <span>Đồng Hành</span>
          </button>

        </nav>
      </aside>

      {/* 2. MAIN CONTENT AREA */}
      <main className="flex-grow flex flex-col min-w-0 bg-zinc-950/20 overflow-y-auto p-6 md:p-8">
        
        {/* Navigation Breadcrumb */}
        <div className="flex items-center gap-1.5 text-[10px] font-mono text-zinc-600 uppercase tracking-widest mb-2">
          <span>Tài Liệu Hướng Dẫn</span>
          <ChevronRight className="w-3 h-3" />
          <span>{currentPage.category.replace("-", " ")}</span>
          {currentPage.subCategory && (
            <>
              <ChevronRight className="w-3 h-3" />
              <span>{currentPage.subCategory.replace("gameplay-", "")}</span>
            </>
          )}
        </div>

        {/* Page Title */}
        <h1 className="text-xl md:text-2xl font-bold font-serif text-zinc-100 border-b border-zinc-900/80 pb-4 mb-6">
          {currentPage.title}
        </h1>

        {/* Page Main Content */}
        <div className="flex-grow space-y-6 text-zinc-300 font-serif leading-relaxed">
          {currentPage.content}

          {/* Card Links if Subpages Exist */}
          {currentPage.cards && currentPage.cards.length > 0 && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-6">
              {currentPage.cards.map((card, idx) => (
                <div
                  key={idx}
                  onClick={() => handleCardClick(card.targetId)}
                  className="p-4 bg-zinc-950/80 border border-zinc-900 hover:border-violet-500/20 hover:bg-zinc-900/20 rounded-2xl cursor-pointer group transition-all duration-300 flex items-center justify-between"
                >
                  <div className="space-y-1">
                    <h4 className="text-xs font-bold font-serif text-zinc-200 group-hover:text-indigo-400 transition-colors">
                      {card.title}
                    </h4>
                    <p className="text-[10px] text-zinc-500 font-mono">
                      {card.description}
                    </p>
                  </div>
                  <ArrowRight className="w-4 h-4 text-zinc-700 group-hover:text-indigo-400 group-hover:translate-x-1 transition-all shrink-0" />
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Footer info box */}
        <div className="border-t border-zinc-900/60 pt-4 mt-8 flex items-center justify-between text-[10px] text-zinc-600 font-mono">
          <span className="flex items-center gap-1">
            <Info className="w-3.5 h-3.5 text-zinc-700" />
            Bản dịch & Tổng hợp theo nguyên tác Cổ Chân Nhân.
          </span>
          <span>Cổ Giới Wiki v1.0</span>
        </div>

      </main>
    </div>
  );
};
