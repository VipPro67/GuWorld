"use client";

import React, { useState } from "react";
import {
  Sparkles,
  ChevronLeft,
  ChevronRight,
  Moon,
  Compass,
  Droplet,
  Swords,
  Users,
  Infinity,
} from "lucide-react";

interface LevelZeroViewProps {
  onComplete: () => Promise<boolean>;
}

const STEPS = [
  {
    icon: Compass,
    title: "Giới thiệu",
    heading: "Cổ Chân Nhân — Con đường tu luyện",
    body: (
      <>
        <p>
          Giữa thập vạn đại sơn, <strong className="text-amber-300">Cổ giới</strong> là thế giới của các Cổ Sư
          liệu pháp sinh mệnh. Tư chất của bạn vừa thức tỉnh một trong bốn phẩm cấp:
          <strong className="text-amber-300"> Giáp, Ất, Bính, Đinh</strong>. Trước khi mơ về Lục Chuyển
          Tiên Khiếu, ta là <strong className="text-amber-300">Phàm Nhân (rank 0)</strong> — chưa có Cổ
          trùng, chưa chọn được vùng đầu quân.
        </p>
      </>
    ),
  },
  {
    icon: Infinity,
    title: "Không Khiếu",
    heading: "Trùng kích Vách Khiếu (Wall)",
    body: (
      <p>
        Năng lượng chiến đấu là <strong className="text-amber-300">Chân Nguyên</strong> trong Không Khiếu.
        Dùng Nguyên Thạch trùng kích mở rộng Vách Khiếu qua 5 Đại cảnh giới phàm nhân:
        <strong> Nhất Chuyển → Ngũ Chuyển</strong> (rồi mới có thể thăng Tiên Lục Chuyển).
      </p>
    ),
  },
  {
    icon: Droplet,
    title: "Cổ Trùng",
    heading: "Thú ăn nhập khẩu sinh mệnh chiến đấu",
    body: (
      <>
        <p>
          Mỗi Cổ Trùng có lưu phái (Hỏa đạo, Thủy đạo, Lực đạo...) và chỉ hung khi được tiếp liền
          <strong className="text-amber-300"> Độ No</strong> đúng thức ăn. Độ No giảm mỗi 3 giờ; về 0
          thì Cổ sẽ <strong className="text-red-300">chết vĩnh viễn</strong>. Chọn một Cổ làm Bản Mệnh
          để nó liên kết sinh mệnh, không mất khi thất bại dã ngoại.
        </p>
      </>
    ),
  },
  {
    icon: Moon,
    title: "Yếu tố thực cảnh",
    heading: "Khai thác & Thám hiển",
    body: (
      <p>
        Hãy thử <strong className="text-amber-300">đào Nguyên Thạch dã ngoại</strong> và mở bản đồ sương
        mù để nhặt dược thảo — và để ý các điểm Nguyên Tuyền 💠, đứng đúng ô là thu hoạch được Nguyên Thạch
        rất nhiều ở giai đoạn đầu.
      </p>
    ),
  },
  {
    icon: Swords,
    title: "Chiến đấu & Thương hội",
    heading: "Luyện Cổ, Sát Chiêu, Tông Môn",
    body: (
      <p>
        Tự do phối hợp Cổ Hạch Tâm + Cổ Phụ Trợ để ghép <strong className="text-amber-300">Sát Chiêu</strong>,
        rèn luyện chúng tại lò luyện và thử trong Phòng Tập trước khi vào Võ Đài hay Đấu Trường Sinh Tử.
        Trung thành với một <strong className="text-amber-300">Tông Môn</strong> để dùng bảo khố và
        tham gia chiến tranh vùng lãnh thổ.
      </p>
    ),
  },
  {
    icon: Users,
    title: "Tiên Khiếu",
    heading: "Thăng Tiên & Thế giới riêng",
    body: (
      <p>
        Khi đủ Nhân Khí và vượt Kiếp Nạn, ta từ Phàm Nhân thành Cổ Tiên, mở
        <strong className="text-amber-300"> Tiên Khiếu</strong> — một thế giới nhỏ độc lập ngay trong cơ
        thể, nơi dòng thời gian chảy chậm hơn và có thể dựng node, gầy dựng thị trấn Dị Nhân.
      </p>
    ),
  },
];

export const LevelZeroView: React.FC<LevelZeroViewProps> = ({ onComplete }) => {
  const [step, setStep] = useState(0);
  const [isFinishing, setIsFinishing] = useState(false);
  const CurrentIcon = STEPS[step].icon;

  const handleFinish = async () => {
    setIsFinishing(true);
    try {
      await onComplete();
    } finally {
      setIsFinishing(false);
    }
  };

  return (
    <div className="glass-panel-gold rounded-3xl p-8 w-full max-w-lg relative overflow-hidden shadow-[0_0_50px_rgba(234,179,8,0.15)] my-8">
      <div className="absolute -left-12 -bottom-12 w-28 h-28 rounded-full bg-emerald-500/10 blur-2xl pointer-events-none" />
      <div className="absolute -right-12 -top-12 w-28 h-28 rounded-full bg-amber-500/10 blur-2xl pointer-events-none" />

      <div className="text-center mb-5">
        <div className="inline-flex p-3 rounded-2xl bg-amber-950/40 border border-amber-500/20 text-amber-400 mb-3 animate-pulse">
          <Sparkles className="w-6 h-6" />
        </div>
        <h2 className="text-2xl font-bold text-zinc-100 font-serif tracking-wide">
          Lộ Trình Tân Thủ — Phàm Nhân
        </h2>
        <p className="text-[11px] font-mono text-zinc-500 mt-1">
          Bước {step + 1} / {STEPS.length} · Hoàn thành để thức tỉnh Không Khiếu
        </p>
      </div>

      <div className="flex items-center justify-center gap-1.5 mb-6">
        {STEPS.map((_, i) => (
          <button
            key={i}
            type="button"
            aria-label={`Bước ${i + 1}`}
            onClick={() => setStep(i)}
            className={`h-1.5 rounded-full transition-all ${
              i === step ? "w-8 bg-amber-400" : "w-3 bg-zinc-800"
            }`}
          />
        ))}
      </div>

      <div className="p-4 rounded-2xl bg-amber-950/20 border border-amber-500/20 text-zinc-200 min-h-[190px]">
        <div className="flex items-center gap-2 mb-3">
          <CurrentIcon className="w-5 h-5 text-amber-400" />
          <h3 className="font-serif font-bold text-amber-300">{STEPS[step].heading}</h3>
        </div>
        <div className="text-xs leading-relaxed text-zinc-300 space-y-2">{STEPS[step].body}</div>
      </div>

      <div className="mt-6 flex items-center justify-between gap-3">
        <button
          type="button"
          onClick={() => setStep((s) => Math.max(0, s - 1))}
          disabled={step === 0}
          className="inline-flex items-center gap-1 px-3 py-2 rounded-xl text-xs text-zinc-400 hover:text-zinc-200 disabled:opacity-30 cursor-pointer disabled:cursor-default"
        >
          <ChevronLeft className="w-4 h-4" /> Trước
        </button>

        {step < STEPS.length - 1 ? (
          <button
            type="button"
            onClick={() => setStep((s) => Math.min(STEPS.length - 1, s + 1))}
            className="inline-flex items-center gap-1 px-5 py-2.5 rounded-xl bg-gradient-to-r from-amber-600 to-yellow-600 text-zinc-950 text-xs font-bold tracking-wide hover:from-amber-500 hover:to-yellow-500 cursor-pointer"
          >
            Tiếp <ChevronRight className="w-4 h-4" />
          </button>
        ) : (
          <button
            type="button"
            onClick={handleFinish}
            disabled={isFinishing}
            className="inline-flex items-center gap-1 px-5 py-2.5 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 text-zinc-950 text-xs font-bold tracking-wide hover:from-emerald-500 hover:to-teal-500 cursor-pointer disabled:opacity-50"
          >
            <Sparkles className="w-4 h-4" />
            {isFinishing ? "Đang thức tỉnh..." : "Thức Tỉnh Không Khiếu"}
          </button>
        )}
      </div>
    </div>
  );
};
