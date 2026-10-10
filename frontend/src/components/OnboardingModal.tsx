"use client";

import React, { useState } from "react";
import Link from "next/link";
import {
  Sparkles,
  Compass,
  Moon,
  Droplet,
  Skull,
  Shield,
  Leaf,
  Gem,
  Clock,
} from "lucide-react";

const STARTING_ATTACK_GUS = [
  {
    id: "GU_MOONLIGHT",
    name: "Nguyệt Quang Cổ",
    description: "Bắn ra nguyệt nhận sắc lẹm màu xanh lam. Tiêu hao ít chân nguyên, gây chảy máu nhẹ.",
    activeColor: "border-blue-500 shadow-[0_0_15px_rgba(59,130,246,0.3)] bg-blue-950/40 text-blue-300",
    icon: Moon,
  },
  {
    id: "GU_STREAM",
    name: "Khê Lưu Cổ",
    description: "Thủy đạo cổ trùng, phun trào dòng nước mạnh mẽ gây sát thương và đẩy lui địch.",
    activeColor: "border-cyan-500 shadow-[0_0_15px_rgba(6,182,212,0.3)] bg-cyan-950/40 text-cyan-300",
    icon: Droplet,
  },
  {
    id: "GU_BONE_SPEAR",
    name: "Cốt Thương Cổ",
    description: "Ngưng kết thương xương trắng đâm xuyên kẻ địch. Sát thương tốt kèm độc tố nhẹ.",
    activeColor: "border-amber-500 shadow-[0_0_15px_rgba(245,158,11,0.3)] bg-zinc-900/40 text-zinc-100",
    icon: Skull,
  },
];

const STARTING_DEFENSE_GUS = [
  {
    id: "GU_STONE_SKIN",
    name: "Thạch Bì Cổ",
    description: "Tạo lớp da đá cứng cáp bao phủ cơ thể, tăng đáng kể phòng ngự vật lý.",
    activeColor: "border-amber-500 shadow-[0_0_15px_rgba(245,158,11,0.3)] bg-amber-950/40 text-amber-300",
    icon: Gem,
  },
  {
    id: "GU_IRON_SKIN",
    name: "Thiết Bì Cổ",
    description: "Cường hóa lớp biểu bì cứng cáp như thép nguội, chịu đòn cực tốt.",
    activeColor: "border-slate-400 shadow-[0_0_15px_rgba(148,163,184,0.3)] bg-slate-950/40 text-slate-200",
    icon: Shield,
  },
  {
    id: "GU_GREEN_SILK",
    name: "Thanh Ti Cổ",
    description: "Dệt lớp tơ gỗ xanh dai nhách bao phủ toàn thân, giảm nhẹ sát thương nhận vào.",
    activeColor: "border-emerald-500 shadow-[0_0_15px_rgba(16,185,129,0.3)] bg-emerald-950/40 text-emerald-300",
    icon: Leaf,
  },
  {
    id: "GU_BEAST_SKIN",
    name: "Thú Bì Cổ",
    description: "Tạo lớp thú bì bảo vệ cơ thể, có thể thăng luyện lên nhiều hướng khác nhau.",
    activeColor: "border-orange-500 shadow-[0_0_15px_rgba(249,115,22,0.3)] bg-orange-950/40 text-orange-300",
    icon: Compass,
  },
  {
    id: "GU_FISH_SCALE",
    name: "Ngư Lân Cổ",
    description: "Trải đều một lớp vảy cá trơn trượt trên da, giảm lực tấn công vật lý.",
    activeColor: "border-teal-500 shadow-[0_0_15px_rgba(20,184,166,0.3)] bg-teal-950/40 text-teal-300",
    icon: Droplet,
  },
];

const REGIONS = [
  { id: "hub_nam_cuong", name: "Nam Cương", desc: "Thành trì an toàn giữa thập vạn đại sơn, nơi các gia tộc Nam Cương giao thương và nghỉ chân." },
  { id: "hub_trung_chau", name: "Trung Châu", desc: "Kinh đô phồn hoa của Trung Châu, khu vực cấm tranh đấu tuyệt đối, nơi tụ hội của Thập Đại Cổ Phái." },
  { id: "hub_bac_nguyen", name: "Bắc Nguyên", desc: "Trấn an toàn giữa thảo nguyên mênh mông Bắc Nguyên, nơi các bộ lạc Hoàng Kim dừng chân." },
  { id: "hub_tay_mac", name: "Tây Mạc", desc: "Ốc đảo an toàn giữa sa mạc Tây Mạc, trạm dừng chân của các thương đội." },
  { id: "hub_dong_hai", name: "Đông Hải", desc: "Cảng an toàn bên bờ Đông Hải, cửa ngõ giao thương đường biển của tộc Giao Nhân." },
];

interface OnboardingModalProps {
  onComplete: (data: {
    starting_region: string;
    attack_gu_id: string;
    defense_gu_id: string;
    vital_gu_choice: string;
  }) => Promise<boolean>;
}

export const OnboardingModal: React.FC<OnboardingModalProps> = ({ onComplete }) => {
  const [selectedRegion, setSelectedRegion] = useState("hub_nam_cuong");
  const [selectedAttackGu, setSelectedAttackGu] = useState("GU_MOONLIGHT");
  const [selectedDefenseGu, setSelectedDefenseGu] = useState("GU_STONE_SKIN");
  const [selectedVital, setSelectedVital] = useState("ATTACK");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleComplete = async () => {
    setIsSubmitting(true);
    await onComplete({
      starting_region: selectedRegion,
      attack_gu_id: selectedAttackGu,
      defense_gu_id: selectedDefenseGu,
      vital_gu_choice: selectedVital,
    });
    setIsSubmitting(false);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/90 backdrop-blur-md overflow-y-auto">
      <div className="glass-panel-gold rounded-3xl p-8 w-full max-w-lg relative overflow-hidden shadow-[0_0_50px_rgba(234,179,8,0.15)] my-8">
        <div className="absolute -left-12 -bottom-12 w-28 h-28 rounded-full bg-emerald-500/10 blur-2xl pointer-events-none" />
        <div className="absolute -right-12 -top-12 w-28 h-28 rounded-full bg-amber-500/10 blur-2xl pointer-events-none" />

        {/* Header */}
        <div className="text-center mb-6">
          <div className="inline-flex p-3 rounded-2xl bg-amber-950/40 border border-amber-500/20 text-amber-400 mb-4 animate-pulse">
            <Sparkles className="w-6 h-6" />
          </div>
          <h2 className="text-2xl font-bold text-zinc-100 font-serif tracking-wide">
            Thiết Lập Ban Đầu
          </h2>
          <p className="text-xs text-zinc-500 mt-2 font-mono">
            Thiên địa ưu ái, tư chất sẽ được bộc lộ ngẫu nhiên
          </p>
          <div className="mt-3">
            <Link
              href="/wiki"
              className="inline-flex items-center gap-1 text-[11px] font-mono text-amber-400 hover:text-amber-300 hover:underline transition-colors"
            >
              📖 Xem hướng dẫn trên Wiki
            </Link>
          </div>
        </div>

        <div className="space-y-6">
          {/* Lifespan & Tiên Cương Notice */}
          <div className="p-3.5 rounded-2xl bg-amber-950/40 border border-amber-500/30 text-amber-200 text-xs space-y-1.5">
            <div className="flex items-center gap-2 font-bold font-serif text-amber-300">
              <Clock className="w-4 h-4 text-amber-400 shrink-0" />
              <span>Cơ Chế Tuổi Thọ & Tiên Cương (Cương Thi)</span>
            </div>
            <p className="text-[11px] text-zinc-300 leading-relaxed font-serif">
              • <strong>Tuổi Thọ:</strong> Cổ sư bị tiêu hao thọ nguyên theo thời gian thực (tăng tốc theo <span className="text-amber-400 font-mono">time_speed</span>). Khi thọ nguyên giảm về 0, nhân vật sẽ rơi vào cõi chết và bị phong tỏa các hoạt động.
              <br />
              • <strong>Cương Thi (Tiên Cương):</strong> Chủng tộc sở hữu <span className="text-emerald-400 font-semibold">Bất Tử Tuổi Thọ</span> nhưng bị phong tỏa hoàn toàn khả năng tự hồi phục HP/Chân Nguyên/Hồn Phách. Tiên Khiếu Cương Thi bị khóa gia tốc thời gian ở <span className="text-amber-400 font-mono">1.0</span> và bị suy thoái theo thời gian.
            </p>
          </div>

          {/* 1. Region */}
          <div>
            <label className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-2">
              Khu Vực Khởi Đầu
            </label>
            <div className="grid grid-cols-1 gap-1.5">
              {REGIONS.map((r) => (
                <button
                  key={r.id}
                  type="button"
                  onClick={() => setSelectedRegion(r.id)}
                  className={`w-full text-left p-2.5 rounded-xl border transition-all cursor-pointer text-xs ${
                    selectedRegion === r.id
                      ? "border-amber-500 shadow-[0_0_12px_rgba(234,179,8,0.15)] bg-amber-950/30 text-amber-300"
                      : "border-zinc-800 bg-zinc-950/40 text-zinc-400 hover:border-zinc-700"
                  }`}
                >
                  <span className="font-serif font-semibold">{r.name}</span>
                  <span className="text-zinc-600 mx-1.5">·</span>
                  <span className="text-zinc-500">{r.desc}</span>
                </button>
              ))}
            </div>
          </div>

          {/* 2. Attack Gu */}
          <div>
            <label className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-2">
              Cổ Trùng Tấn Công (Cấp 1)
            </label>
            <div className="grid grid-cols-3 gap-2">
              {STARTING_ATTACK_GUS.map((item) => {
                const IconComponent = item.icon;
                const isSelected = selectedAttackGu === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setSelectedAttackGu(item.id)}
                    className={`flex flex-col items-center gap-1.5 p-2.5 rounded-xl border text-center transition-all cursor-pointer ${
                      isSelected
                        ? item.activeColor
                        : "border-zinc-800 bg-zinc-950/40 text-zinc-400 hover:text-zinc-300 hover:border-zinc-700"
                    }`}
                  >
                    <IconComponent className="w-5 h-5" />
                    <span className="text-[11px] font-semibold font-serif">{item.name.replace(" Cổ", "")}</span>
                  </button>
                );
              })}
            </div>
            <p className="text-[10px] text-zinc-500 font-serif mt-1.5 italic min-h-[28px]">
              {STARTING_ATTACK_GUS.find((g) => g.id === selectedAttackGu)?.description}
            </p>
          </div>

          {/* 3. Defense Gu */}
          <div>
            <label className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-2">
              Cổ Trùng Phòng Thủ (Cấp 1)
            </label>
            <div className="grid grid-cols-5 gap-1.5">
              {STARTING_DEFENSE_GUS.map((item) => {
                const IconComponent = item.icon;
                const isSelected = selectedDefenseGu === item.id;
                return (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setSelectedDefenseGu(item.id)}
                    title={item.name}
                    className={`flex flex-col items-center gap-1 p-2 rounded-xl border text-center transition-all cursor-pointer ${
                      isSelected
                        ? item.activeColor
                        : "border-zinc-800 bg-zinc-950/40 text-zinc-400 hover:text-zinc-300 hover:border-zinc-700"
                    }`}
                  >
                    <IconComponent className="w-4 h-4" />
                    <span className="text-[9px] font-semibold font-serif whitespace-nowrap overflow-hidden text-ellipsis w-full">
                      {item.name.replace(" Cổ", "")}
                    </span>
                  </button>
                );
              })}
            </div>
            <p className="text-[10px] text-zinc-500 font-serif mt-1.5 italic min-h-[28px]">
              {STARTING_DEFENSE_GUS.find((g) => g.id === selectedDefenseGu)?.description}
            </p>
          </div>

          {/* 4. Vital Gu */}
          <div>
            <label className="block text-[10px] font-mono text-zinc-500 uppercase tracking-wider mb-1.5">
              Bản Mệnh Cổ
            </label>
            <select
              value={selectedVital}
              onChange={(e) => setSelectedVital(e.target.value)}
              className="w-full bg-zinc-950/80 border border-zinc-800 rounded-xl py-2.5 px-4 text-sm text-zinc-100 focus:outline-none focus:border-amber-500/50 transition-all cursor-pointer"
            >
              <option value="ATTACK">Đặt Cổ Tấn Công làm Bản Mệnh</option>
              <option value="DEFENSE">Đặt Cổ Phòng Thủ làm Bản Mệnh</option>
              <option value="NONE">Chưa thiết lập (Để chọn sau)</option>
            </select>
          </div>
        </div>

        {/* Submit */}
        <button
          type="button"
          onClick={handleComplete}
          disabled={isSubmitting}
          className="w-full py-3.5 px-6 rounded-xl bg-gradient-to-r from-amber-600 to-yellow-600 text-zinc-950 text-sm font-serif font-semibold tracking-wider hover:from-amber-500 hover:to-yellow-500 hover:shadow-[0_0_25px_rgba(234,179,8,0.25)] active:scale-[0.98] transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed mt-6 flex items-center justify-center gap-2"
        >
          {isSubmitting ? (
            <>
              <div className="w-4 h-4 border-2 border-zinc-950 border-t-transparent rounded-full animate-spin" />
              Đang thiết lập...
            </>
          ) : (
            <>
              <Sparkles className="w-4 h-4" />
              Bắt Đầu Hành Trình
            </>
          )}
        </button>
      </div>
    </div>
  );
};
