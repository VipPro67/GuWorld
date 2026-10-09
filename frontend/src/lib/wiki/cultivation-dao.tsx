import React from "react";
import { AlertTriangle, BookOpen, Shield, Swords, Zap } from "lucide-react";
import { WikiPage } from "./types";

export const cultivationDaoPages: WikiPage[] = [
  {
    id: "gameplay-dao-conflict",
    title: "Xung Đột Đạo Ngân & Phản Phệ Trong Chiến Đấu",
    category: "cultivation",
    keywords: [
      "đạo ngân", "dao mark", "xung đột đạo ngân", "dao conflict", "phản phệ", "bạo thể",
      "lưu phái", "path mastery", "chân ý", "đánh thường", "đạo ngân đa tu"
    ],
    content: (
      <div className="space-y-6 text-xs leading-relaxed text-[var(--text-secondary)]">
        <section className="p-4 rounded-2xl bg-gradient-to-r from-amber-950/40 via-purple-950/30 to-indigo-950/40 border border-amber-500/30">
          <h3 className="text-sm font-serif font-bold text-amber-200 flex items-center gap-2">
            <BookOpen className="w-4 h-4" /> Đạo Ngân làm gì trong chiến đấu?
          </h3>
          <p className="mt-2">
            Đạo Ngân của đúng lưu phái làm tăng uy lực đòn đánh thuộc lưu phái đó. Đạo Ngân ngoại hệ có thể cản trở hành động đang dùng; mức ảnh hưởng được tính riêng cho từng hành động dựa trên lưu phái của nó, lượng Đạo Ngân và Chân Ý ở chính lưu phái đó. Vì vậy, một nhân vật có độ thuần Đạo Ngân cao vẫn có thể bị phản phệ khi dùng một lưu phái khác.
          </p>
          <p className="mt-2 text-amber-200"><strong>Điểm cần nhớ:</strong> nếu đòn đánh không khai báo lưu phái, hệ thống tính nó là <strong>Strength (Lực Đạo)</strong>. Cổ Thủy đang trang bị không tự biến đòn đánh thường thành đòn Thủy.
          </p>
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Zap className="w-4 h-4 text-amber-400" /> Cách hệ thống tính xung đột
          </h3>
          <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-[var(--border)] space-y-3">
            <ol className="list-decimal list-inside space-y-2">
              <li><strong>Chọn lưu phái của hành động.</strong> Hệ thống dùng lưu phái được khai báo đầu tiên trên kỹ năng/Cổ; nếu không có thì dùng Strength.</li>
              <li><strong>Tính ngưỡng dung nạp:</strong> ngưỡng = max(1.000, 2% tổng Đạo Ngân dương). Đạo Ngân của phái khác chỉ gây xung đột khi <strong>lớn hơn</strong> ngưỡng; bằng ngưỡng vẫn được bỏ qua.</li>
              <li><strong>So độ tương thích.</strong> Xung đột thô của mỗi phái khác = Đạo Ngân đang hoạt động × (1 − độ tương thích với phái của hành động). Cặp càng kỵ nhau thì xung đột càng cao.</li>
              <li><strong>Tính ảnh hưởng đa tu.</strong> Nhiều phái ít tương thích làm xung đột cộng dồn mạnh hơn. Tỉ lệ Đạo Ngân của chính phái đang dùng so với tổng Đạo Ngân cũng ảnh hưởng mức này.</li>
              <li><strong>Áp dụng Chân Ý của phái đang dùng.</strong> Chân Ý không làm sạch mọi lưu phái cùng lúc; nó điều hòa xung đột cho hành động thuộc đúng phái đã luyện.</li>
            </ol>
            <div className="p-3 rounded-lg bg-zinc-950/80 border border-zinc-800 font-mono text-[11px] text-[var(--text-primary)] space-y-1">
              <div>Fđa-tu = 1 + (NhiềuPháiHiệuDụng − 1)<sup>1,3</sup> × (1 − TỉLệĐạoNgânPháiĐangDùng)<sup>2</sup> × 4</div>
              <div>XungĐộtHiệuDụng = XungĐộtThô × Fđa-tu × (1 − MứcĐiềuHòaChânÝ)</div>
              <div>TỉLệXungĐột = XungĐộtHiệuDụng / [ĐạoNgânPháiĐangDùng + max(1.000, 10% TổngĐạoNgân)]</div>
              <div className="text-amber-300">M = max(0,08; 1 / (1 + TỉLệXungĐột × 1,6))</div>
            </div>
            <p><strong>M</strong> là hệ số còn lại sau xung đột, trong khoảng 0,08–1,00. M = 1 nghĩa là không bị giảm bởi xung đột; M = 0,08 là mức giảm tối đa.
            </p>
          </div>
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <Swords className="w-4 h-4 text-indigo-400" /> Chân Ý điều hòa theo từng lưu phái
          </h3>
          <div className="overflow-hidden rounded-xl border border-[var(--border)]">
            <table className="w-full text-left">
              <thead className="bg-zinc-950 text-[var(--text-secondary)]"><tr><th className="p-3">Cảnh giới</th><th className="p-3">Mốc điểm</th><th className="p-3">Xung đột được điều hòa</th></tr></thead>
              <tbody className="divide-y divide-zinc-900 bg-[var(--surface)] text-[var(--text-primary)]">
                <tr><td className="p-3">Phàm Thai</td><td className="p-3">0</td><td className="p-3">0%</td></tr>
                <tr><td className="p-3">Đại Sư</td><td className="p-3">10.000</td><td className="p-3">10%</td></tr>
                <tr><td className="p-3">Tông Sư</td><td className="p-3">100.000</td><td className="p-3">25%</td></tr>
                <tr><td className="p-3">Đại Tông Sư</td><td className="p-3">1.000.000</td><td className="p-3">45%</td></tr>
                <tr><td className="p-3">Vô Thượng Đại Tông Sư</td><td className="p-3">10.000.000</td><td className="p-3">100%</td></tr>
              </tbody>
            </table>
          </div>
          <p className="p-3 rounded-lg bg-indigo-950/20 border border-indigo-500/20">Tỉ lệ điều hòa này áp dụng cho <strong>phái đang thi triển</strong>. Ví dụ, Vô Thượng Strength có thể hóa giải xung đột khi đánh bằng Strength; nó không tự hóa giải một kỹ năng Thủy nếu Chân Ý Thủy chưa đạt Vô Thượng.
          </p>
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-400" /> Ảnh hưởng lên đòn đánh và phản phệ
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-amber-500/20 space-y-2">
              <h4 className="font-bold text-amber-300">Sát thương và Chân Nguyên</h4>
              <p>Phần khuếch đại Đạo Ngân của đòn đánh được nhân với M. Đạo Ngân của phái khác <strong>không tăng</strong> khuếch đại cho đòn này; chúng chỉ có thể gây xung đột. Nếu hành động có phí Chân Nguyên, phí được nhân với <strong>1 + (1 − M) × 1,5</strong>, tối đa 2,38 lần.</p>
            </div>
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-rose-500/20 space-y-2">
              <h4 className="font-bold text-rose-300">Tán loạn kỹ năng</h4>
              <p>Chỉ kỹ năng Cổ và sát chiêu có thể tán loạn; kiểm tra khi M &lt; 0,70. Tỉ lệ tối đa 25% tại M = 0,08. Chân Nguyên đã bị trừ trước lần gieo này. Nếu kỹ năng tán loạn, lượt đó kết thúc và không gieo phản phệ tiếp.</p>
            </div>
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-red-500/20 space-y-2">
              <h4 className="font-bold text-red-300">Phản phệ nội thương</h4>
              <p>Khi M &lt; 0,60, mỗi hành động có tối đa 30% xác suất phản phệ tại M = 0,08. Đánh thường cũng có thể bị phản phệ nếu hành động được tính theo Strength và xung đột Strength đủ nặng. Khi xảy ra, người ra đòn mất trực tiếp 8–12% HP tối đa; đòn đó gây một nửa sát thương.</p>
            </div>
            <div className="p-4 rounded-xl bg-[var(--bg-card)] border border-blue-500/20 space-y-2">
              <h4 className="font-bold text-blue-300 flex items-center gap-2"><Shield className="w-4 h-4" /> Bạo thể vong mạng</h4>
              <p>Không có một lần gieo riêng để quyết định tử vong. Nếu HP về 0 sau khi nhận sát thương phản phệ, nhân vật tử vong. Vì vậy thanh HP thấp khiến phản phệ nguy hiểm hơn dù bảng Đạo Ngân cho thấy một phái đang rất thuần.</p>
            </div>
          </div>
          <p className="text-[var(--text-muted)]">Một số Cổ trùng đang ACTIVE có hiệu ứng miễn xung đột đủ mạnh có thể bỏ qua hình phạt. Kháng Đạo Ngân tự nhiên của bên phòng thủ là cơ chế riêng, không ngăn người tấn công tự phản phệ.
          </p>
        </section>

        <section className="space-y-3">
          <h3 className="text-sm font-bold uppercase tracking-wider text-[var(--text-primary)] font-mono">Ví dụ thực tế</h3>
          <div className="space-y-3">
            <article className="p-4 rounded-xl bg-[var(--bg-card)] border border-emerald-500/20">
              <h4 className="font-bold text-emerald-300">Hỏa 1.000, Thủy 0 — dùng kỹ năng Thủy</h4>
              <p>Tổng Đạo Ngân = 1.000 nên ngưỡng cũng là 1.000. Hỏa bằng ngưỡng, không lớn hơn ngưỡng, vì vậy không gây xung đột: M = 1, không tăng phí, không tán loạn và không phản phệ. Đòn Thủy cũng không được khuếch đại bởi Đạo Ngân Thủy vì nhân vật có 0 Đạo Ngân Thủy.</p>
            </article>
            <article className="p-4 rounded-xl bg-[var(--bg-card)] border border-rose-500/20">
              <h4 className="font-bold text-rose-300">Hỏa 1.001, Thủy 0 — dùng kỹ năng Thủy</h4>
              <p>Hỏa vượt ngưỡng 1.000 một điểm. Do Hỏa–Thủy tương thích 0,55, M xấp xỉ 0,236 nếu không có Chân Ý/bonus khác. Phí Chân Nguyên khoảng 2,15 lần; kỹ năng có 18,7% nguy cơ tán loạn. Nếu qua bước đó, phản phệ có khoảng 21,0% xác suất và gây mất 8–12% HP tối đa.</p>
            </article>
            <article className="p-4 rounded-xl bg-[var(--bg-card)] border border-amber-500/20">
              <h4 className="font-bold text-amber-300">Hỏa 2.000, Strength 0 — đánh thường</h4>
              <p>Đánh thường không khai báo lưu phái nên dùng Strength; Cổ Thủy trong túi/trang bị không đổi quy tắc đó. Hỏa–Strength tương thích 0,85 nên xung đột nhẹ hơn Hỏa–Thủy. Với Chân Ý Strength ở Phàm Thai và không có bonus khác, M xấp xỉ 0,532: đánh thường không tán loạn, nhưng có khoảng 4% nguy cơ phản phệ mỗi đòn. Nếu đòn có phí Chân Nguyên, phí khoảng 1,70 lần.</p>
            </article>
            <article className="p-4 rounded-xl bg-[var(--bg-card)] border border-indigo-500/20">
              <h4 className="font-bold text-indigo-300">Cùng Hỏa 2.000, nhưng Chân Ý Strength Vô Thượng</h4>
              <p>Đòn đánh thường vẫn dùng Strength, và Chân Ý Strength Vô Thượng điều hòa 100% xung đột cho hành động đó nên M = 1. Kỹ năng Thủy vẫn phải xét Chân Ý Thủy riêng.</p>
            </article>
          </div>
        </section>
      </div>
    ),
  },
];
