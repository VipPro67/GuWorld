"use client";

import { useEffect, useState } from "react";
import { AlertTriangle, Clock, ShieldAlert, Swords, Leaf, Flame, Star, Wrench } from "lucide-react";

export const MaintenanceView = () => {
  const [timeLeft, setTimeLeft] = useState("");

  useEffect(() => {
    setTimeLeft("CHƯA RÕ");
  }, []);

  return (
    <div className="w-full max-w-3xl mx-auto py-8 sm:py-12 animate-fade-in">
      <div className="bg-gradient-to-b from-zinc-950 to-zinc-900 border border-red-900/30 rounded-3xl p-6 sm:p-10 shadow-2xl relative overflow-hidden">
        {/* Background Effects */}
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-full max-w-lg h-32 bg-red-600/10 rounded-full blur-[100px] pointer-events-none" />
        <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-full max-w-lg h-32 bg-amber-600/10 rounded-full blur-[100px] pointer-events-none" />

        {/* Header */}
        <div className="text-center mb-10 relative z-10">
          <div className="w-20 h-20 rounded-full bg-red-950/50 border border-red-500/20 flex items-center justify-center mx-auto mb-6 shadow-[0_0_30px_rgba(220,38,38,0.2)]">
            <AlertTriangle className="w-10 h-10 text-red-500 animate-pulse" />
          </div>
          <h1 className="text-2xl sm:text-3xl lg:text-4xl font-serif font-bold text-red-500 mb-4 tracking-wide uppercase">
            🚨 [Cổ Giới Reset] 🚨
          </h1>
          <h2 className="text-xl sm:text-2xl font-serif font-bold text-zinc-100 mb-4 bg-gradient-to-r from-amber-200 to-amber-500 bg-clip-text text-transparent">
            KỶ NGUYÊN MỚI CHÍNH THỨC BẮT ĐẦU! ⚔️
          </h2>
          <p className="text-base sm:text-lg text-zinc-300 font-serif max-w-2xl mx-auto leading-relaxed">
            Các vị Đạo hữu thân mến! 🌌
            <br />
            Cổ Giới sẽ chính thức <strong className="text-red-400">RESET TOÀN BỘ</strong>, đưa tất cả trở về vạch xuất phát để bắt đầu một kỷ nguyên mới.
          </p>
        </div>

        {/* Countdown */}
        <div className="bg-black/40 border border-zinc-800 rounded-2xl p-6 mb-10 text-center relative z-10">
          <div className="flex items-center justify-center gap-2 mb-3 text-amber-500">
            <Clock className="w-5 h-5" />
            <span className="font-serif font-semibold tracking-wide uppercase text-sm">Thời Gian Dự Kiến Mở Lại</span>
          </div>
          <div className="text-5xl sm:text-6xl lg:text-7xl font-mono font-bold text-zinc-100 tracking-wider text-shadow-glow">
            {timeLeft}
          </div>
        </div>

        {/* Timeline */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-10 relative z-10">
          <div className="bg-zinc-950/60 border border-zinc-800/60 rounded-xl p-4 flex flex-col items-center text-center gap-2">
            <ShieldAlert className="w-6 h-6 text-red-400" />
            <div className="text-sm font-mono text-zinc-400">Chưa rõ</div>
            <div className="text-sm font-bold text-zinc-200">Tiến hành Reset</div>
          </div>
          <div className="bg-zinc-950/60 border border-zinc-800/60 rounded-xl p-4 flex flex-col items-center text-center gap-2">
            <Wrench className="w-6 h-6 text-amber-400" />
            <div className="text-sm font-mono text-zinc-400">Chưa rõ</div>
            <div className="text-sm font-bold text-zinc-200">Bảo trì & Cập nhật</div>
          </div>
          <div className="bg-zinc-950/60 border border-emerald-900/30 rounded-xl p-4 flex flex-col items-center text-center gap-2">
            <Star className="w-6 h-6 text-emerald-400" />
            <div className="text-sm font-mono text-emerald-400/80">Chưa rõ</div>
            <div className="text-sm font-bold text-emerald-300">Dự kiến mở server</div>
          </div>
        </div>

        {/* Warning Section */}
        <div className="bg-red-950/10 border border-red-900/20 rounded-2xl p-6 sm:p-8 mb-10 relative z-10">
          <h3 className="flex items-center gap-2 text-lg sm:text-xl font-bold font-serif text-red-400 mb-6 border-b border-red-900/30 pb-3">
            <Flame className="w-5 h-5" />
            Kỷ nguyên mới sẽ khắc nghiệt hơn!
          </h3>
          <ul className="space-y-4 mb-6">
            <li className="flex items-start gap-3">
              <span className="text-red-500 font-bold shrink-0">❌</span>
              <span className="text-zinc-300 text-sm sm:text-base">Không buff miễn phí</span>
            </li>
            <li className="flex items-start gap-3">
              <span className="text-red-500 font-bold shrink-0">❌</span>
              <span className="text-zinc-300 text-sm sm:text-base">Không ưu ái người chơi</span>
            </li>
            <li className="flex items-start gap-3">
              <span className="text-red-500 font-bold shrink-0">❌</span>
              <span className="text-zinc-300 text-sm sm:text-base">Không còn con đường dễ dàng</span>
            </li>
            <li className="flex items-start gap-3">
              <span className="text-amber-500 font-bold shrink-0 mt-0.5"><Swords className="w-4 h-4" /></span>
              <span className="text-zinc-300 text-sm sm:text-base">Tài nguyên quý giá hơn, tu luyện chậm hơn và chiến đấu nguy hiểm hơn.</span>
            </li>
          </ul>
          <p className="text-zinc-400 text-sm sm:text-base leading-relaxed p-4 bg-zinc-950/50 rounded-xl border border-zinc-900">
            Tất cả sẽ bắt đầu lại từ con số 0. Không còn lợi thế từ nhân vật cũ, không còn khoảng cách giữa người chơi cũ và mới.
          </p>
        </div>

        {/* Path to Ascension */}
        <div className="text-center mb-8 relative z-10">
          <div className="inline-flex flex-wrap justify-center items-center gap-2 sm:gap-3 px-4 py-3 bg-emerald-950/20 border border-emerald-900/30 rounded-xl">
            <span className="text-emerald-400 font-semibold text-xs sm:text-sm flex items-center gap-1"><Leaf className="w-3.5 h-3.5" /> Săn Cổ Trùng</span>
            <span className="text-zinc-600">→</span>
            <span className="text-zinc-300 font-semibold text-xs sm:text-sm">Tích lũy tài nguyên</span>
            <span className="text-zinc-600">→</span>
            <span className="text-zinc-300 font-semibold text-xs sm:text-sm">Tu luyện</span>
            <span className="text-zinc-600">→</span>
            <span className="text-red-400 font-semibold text-xs sm:text-sm">Tranh đấu</span>
            <span className="text-zinc-600">→</span>
            <span className="text-amber-400 font-semibold text-xs sm:text-sm">Đột phá cảnh giới</span>
            <span className="text-zinc-600">→</span>
            <span className="text-amber-300 font-bold text-sm sm:text-base uppercase flex items-center gap-1"><Star className="w-4 h-4" /> Thăng Tiên!</span>
          </div>
        </div>

        {/* Footer Text */}
        <div className="text-center relative z-10">
          <p className="text-zinc-400 font-serif italic mb-2">Một kỷ nguyên kết thúc. Một kỷ nguyên mới bắt đầu.</p>
          <p className="text-red-400 font-serif font-bold uppercase tracking-widest text-sm">
            🔥 Không buff. Không ưu ái. Không dễ dàng. 🔥
          </p>
        </div>
      </div>
    </div>
  );
};
