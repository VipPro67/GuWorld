"use client";

import React, { useState, useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import type { ScoreDetail } from "@/store/slices/scoutingSlice";
import { formatNumber } from "@/lib/utils";
import {
  Compass,
  Eye,
  ShieldAlert,
  Sparkles,
  UserCheck,
  X,
  Radar,
  Scissors,
  Bug,
  MapPin,
  Clock,
  Shield,
  BookOpen,
  Building2,
  Store,
  User,
} from "lucide-react";

const MyScoreBreakdown: React.FC<{ score?: ScoreDetail }> = ({ score }) => {
  if (!score) return null;
  // Cảnh giới lưu phái là trụ số một, nên tách riêng khỏi realm_dao để
  // người chơi thấy rõ mình đang thiếu ở đâu.
  const rows: Array<[string, number]> = [
    ["Cảnh giới lưu phái", score.attain_base ?? 0],
    ["Tu vi + Cổ trinh sát", score.stat],
    ["Cảnh giới × Đạo ngân", score.realm_dao - (score.attain_base ?? 0)],
    ["Manh mối", score.clue],
    ["Vận số", score.luck_roll],
  ];
  const divisor = score.rank_divisor && score.rank_divisor > 1 ? score.rank_divisor : 0;
  const cell = (label: string, value: number, highlight = false) => (
    <div
      className={
        highlight
          ? "rounded-lg border border-amber-400/20 bg-amber-950/35 px-2.5 py-2"
          : "rounded-lg border border-white/5 bg-black/25 px-2.5 py-2"
      }
    >
      <div className="text-neutral-400/80">{label}</div>
      <div
        className={`font-mono font-semibold ${
          highlight ? "text-amber-300" : value < 0 ? "text-red-300" : "text-emerald-300"
        }`}
      >
        {value < 0 ? "−" : "+"}
        {formatNumber(Math.abs(value), 1)}
      </div>
    </div>
  );
  return (
    <div className="mb-3 grid grid-cols-2 gap-2 text-[11px] sm:grid-cols-3">
      {rows.map(([label, value]) => (
        <div key={label}>{cell(label, value)}</div>
      ))}
      <div>{cell("Tổng điểm ta", score.total, true)}</div>
      {divisor > 0 && (
        <div className="col-span-2 rounded-lg border border-white/5 bg-black/25 px-2.5 py-2 text-neutral-400 sm:col-span-3">
          Đã chia theo bậc Cổ: <span className="font-mono font-semibold text-cyan-300">÷{formatNumber(divisor, 1)}</span>
        </div>
      )}
    </div>
  );
};

export const DivinationModal: React.FC = () => {
  const {
    isDivinationModalOpen,
    closeDivinationModal,
    divinationTargetPreset,
    activeScoutingTab,
    setActiveScoutingTab,
    executeDivination,
    executeDivineGu,
    divinationLoading,
    guDivinationLoading,
    activeDivinationReport,
    activeGuDivinationReport,
    applyTrackingMark,
    trackedTargets,
    fetchTrackedTargets,
    severClues,
    setView,
    player,
  } = useGameStore();

  // Player Divination Form
  const [targetUsername, setTargetUsername] = useState("");
  const [suspectedRegion, setSuspectedRegion] = useState("");
  const [suspectedSect, setSuspectedSect] = useState("");
  const [suspectedGu, setSuspectedGu] = useState("");
  const [playerResourceType, setPlayerResourceType] = useState<"primeval_stone" | "immortal_stone">("primeval_stone");
  const [playerResourceAmount, setPlayerResourceAmount] = useState(200);
  const [applyingMark, setApplyingMark] = useState(false);

  // Gu Divination Form
  const [guQuery, setGuQuery] = useState("");
  const [targetRank, setTargetRank] = useState<number>(0);
  const [guResourceType, setGuResourceType] = useState<"primeval_stone" | "immortal_stone">("primeval_stone");
  const [guResourceAmount, setGuResourceAmount] = useState(200);

  // Sever clues action
  const [severing, setSevering] = useState(false);

  useEffect(() => {
    if (divinationTargetPreset) {
      setTargetUsername(divinationTargetPreset);
      setActiveScoutingTab("player");
    }
  }, [divinationTargetPreset, setActiveScoutingTab]);

  useEffect(() => {
    if (isDivinationModalOpen && activeScoutingTab === "radar") {
      fetchTrackedTargets();
    }
  }, [isDivinationModalOpen, activeScoutingTab, fetchTrackedTargets]);

  if (!isDivinationModalOpen) return null;

  const isImmortal = (player?.rank ?? 1) >= 6;

  const handlePlayerDivinationSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetUsername.trim()) return;

    await executeDivination({
      target_username: targetUsername.trim(),
      suspected_region: suspectedRegion.trim() || undefined,
      suspected_sect: suspectedSect.trim() || undefined,
      suspected_gu: suspectedGu.trim() || undefined,
      resource_type: playerResourceType,
      resource_amount: playerResourceAmount,
    });
  };

  const handleGuDivinationSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!guQuery.trim()) return;

    await executeDivineGu({
      gu_query: guQuery.trim(),
      target_rank: targetRank > 0 ? targetRank : undefined,
      resource_type: guResourceType,
      resource_amount: guResourceAmount,
    });
  };

  const handleApplyMark = async (markType: "wisdom" | "blood") => {
    if (!activeDivinationReport?.divination_report?.target_id) return;
    setApplyingMark(true);
    await applyTrackingMark(activeDivinationReport.divination_report.target_id, markType);
    setApplyingMark(false);
  };

  const handleLocateTarget = (_target?: { current_region?: string; x?: number; y?: number }) => {
    setView("mmo-world");
    closeDivinationModal();
  };

  const handleSeverClues = async () => {
    setSevering(true);
    await severClues();
    setSevering(false);
  };

  const pReport = activeDivinationReport?.divination_report;
  const pStatus = activeDivinationReport?.status_type;

  const gReport = activeGuDivinationReport?.report;
  const gStatus = activeGuDivinationReport?.status_type;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-[#080907]/85 p-2 backdrop-blur-md sm:p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="divination-title"
        className="relative flex h-[calc(100dvh-1rem)] max-h-[820px] w-full max-w-4xl flex-col overflow-hidden rounded-2xl border border-amber-500/35 bg-[#15120e] shadow-[0_24px_100px_rgba(0,0,0,0.7),0_0_48px_rgba(190,130,38,0.12)] md:h-auto md:max-h-[88vh]"
      >
        {/* Header */}
        <div className="relative flex shrink-0 items-center justify-between overflow-hidden border-b border-amber-500/20 bg-[radial-gradient(ellipse_at_0%_0%,rgba(139,91,36,0.25),transparent_55%),linear-gradient(110deg,#21190f,#151311_60%,#171321)] px-4 py-3.5 sm:px-6 sm:py-4">
          <div aria-hidden="true" className="pointer-events-none absolute -right-10 -top-20 h-48 w-48 rounded-full border border-amber-200/10 shadow-[0_0_0_18px_rgba(224,175,88,0.025),0_0_0_38px_rgba(224,175,88,0.02)]" />
          <div className="relative flex min-w-0 items-center gap-3">
            <div className="relative grid h-11 w-11 shrink-0 place-items-center rounded-2xl border border-amber-300/40 bg-gradient-to-br from-amber-500/25 via-violet-950/60 to-indigo-950/70 shadow-[0_0_22px_rgba(202,144,47,0.15)]">
              <Compass className="h-5 w-5 text-amber-200" />
              <Sparkles className="absolute -right-1 -top-1 h-3.5 w-3.5 text-violet-200" />
            </div>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-x-2 gap-y-0.5">
                <h2 id="divination-title" className="font-serif text-lg font-bold tracking-wide text-amber-200 sm:text-xl">Thiên Cơ Các</h2>
                <span className="hidden items-center gap-1 rounded-full border border-emerald-500/20 bg-emerald-950/35 px-2 py-0.5 text-[9px] font-mono uppercase tracking-wider text-emerald-300/80 sm:inline-flex">
                  <span className="h-1.5 w-1.5 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.8)]" />
                  Thiên cơ vận hành
                </span>
              </div>
              <p className="mt-0.5 max-w-xl text-[10px] leading-relaxed text-neutral-400 sm:text-xs">
                Suy diễn thiên cơ · Tầm nã Cổ Trùng · Khóa chặt ấn ký
              </p>
            </div>
          </div>
          <button
            onClick={closeDivinationModal}
            type="button"
            aria-label="Đóng Thiên Cơ Các"
            className="relative ml-3 grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-white/10 bg-black/20 text-neutral-400 transition hover:border-amber-400/40 hover:bg-amber-950/40 hover:text-amber-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-400/70"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Tab Navigation */}
        <div
          role="tablist"
          aria-label="Phương thức suy diễn"
          className="scrollbar-none flex shrink-0 gap-1 overflow-x-auto border-b border-amber-100/10 bg-[#100e0c] px-2 text-[11px] font-semibold sm:px-4 sm:text-xs"
        >
          <button
            type="button"
            role="tab"
            aria-selected={activeScoutingTab === "player"}
            onClick={() => setActiveScoutingTab("player")}
            className={`flex shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-3 py-3.5 transition sm:px-4 ${
              activeScoutingTab === "player"
                ? "border-amber-400 bg-amber-500/[0.06] text-amber-200"
                : "border-transparent text-neutral-400 hover:bg-white/[0.03] hover:text-neutral-200"
            }`}
          >
            <Eye className={`h-4 w-4 ${activeScoutingTab === "player" ? "text-amber-300" : ""}`} />
            Suy Diễn Cổ Sư
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeScoutingTab === "gu"}
            onClick={() => setActiveScoutingTab("gu")}
            className={`flex shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-3 py-3.5 transition sm:px-4 ${
              activeScoutingTab === "gu"
                ? "border-violet-400 bg-violet-500/[0.06] text-violet-200"
                : "border-transparent text-neutral-400 hover:bg-white/[0.03] hover:text-neutral-200"
            }`}
          >
            <Bug className={`h-4 w-4 ${activeScoutingTab === "gu" ? "text-violet-300" : ""}`} />
            Suy Tính Cổ Trùng
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeScoutingTab === "radar"}
            onClick={() => setActiveScoutingTab("radar")}
            className={`flex shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-3 py-3.5 transition sm:px-4 ${
              activeScoutingTab === "radar"
                ? "border-cyan-400 bg-cyan-500/[0.06] text-cyan-200"
                : "border-transparent text-neutral-400 hover:bg-white/[0.03] hover:text-neutral-200"
            }`}
          >
            <Radar className={`h-4 w-4 ${activeScoutingTab === "radar" ? "text-cyan-300" : ""}`} />
            Radar Truy Tung ({trackedTargets.length})
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={activeScoutingTab === "sever"}
            onClick={() => setActiveScoutingTab("sever")}
            className={`flex shrink-0 items-center gap-2 whitespace-nowrap border-b-2 px-3 py-3.5 transition sm:px-4 ${
              activeScoutingTab === "sever"
                ? "border-rose-400 bg-rose-500/[0.06] text-rose-200"
                : "border-transparent text-neutral-400 hover:bg-white/[0.03] hover:text-neutral-200"
            }`}
          >
            <Scissors className={`h-4 w-4 ${activeScoutingTab === "sever" ? "text-rose-300" : ""}`} />
            Chém Đứt Manh Mối
          </button>
        </div>

        {/* Tab Content */}
        <div
          role="tabpanel"
          className="flex-1 space-y-5 overflow-y-auto overscroll-contain bg-[radial-gradient(ellipse_at_50%_-20%,rgba(125,82,30,0.1),transparent_65%)] p-3 text-sm sm:p-4 md:p-6"
        >
          {/* TAB 1: PLAYER DIVINATION */}
          {activeScoutingTab === "player" && (
            <div className="space-y-4">
              <form onSubmit={handlePlayerDivinationSubmit} className="space-y-4">
                <div>
                  <label className="mb-1.5 block text-xs font-semibold text-amber-100/85">
                    Danh xưng mục tiêu (Username) <span className="text-red-400">*</span>
                  </label>
                  <input
                    type="text"
                    value={targetUsername}
                    onChange={(e) => setTargetUsername(e.target.value)}
                    placeholder="Nhập tên cổ sư cần suy diễn..."
                    required
                    className="min-h-11 w-full rounded-xl border border-amber-100/10 bg-black/35 px-3.5 py-2.5 text-sm text-amber-50 placeholder:text-neutral-600 transition focus:border-amber-400/60 focus:outline-none focus:ring-2 focus:ring-amber-400/10"
                  />
                </div>

                <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                  <div>
                    <label className="mb-1.5 block text-[11px] font-medium text-neutral-400">Vùng nghi ngờ</label>
                    <input
                      type="text"
                      value={suspectedRegion}
                      onChange={(e) => setSuspectedRegion(e.target.value)}
                      placeholder="VD: tan_thu_coc_nam_cuong"
                      className="min-h-10 w-full rounded-lg border border-white/8 bg-black/30 px-3 text-xs text-amber-50 placeholder:text-neutral-600 transition focus:border-amber-400/50 focus:outline-none focus:ring-2 focus:ring-amber-400/10"
                    />
                  </div>
                  <div>
                    <label className="mb-1.5 block text-[11px] font-medium text-neutral-400">Tông môn nghi ngờ</label>
                    <input
                      type="text"
                      value={suspectedSect}
                      onChange={(e) => setSuspectedSect(e.target.value)}
                      placeholder="VD: Cổ Nguyệt Sơn Trại"
                      className="min-h-10 w-full rounded-lg border border-white/8 bg-black/30 px-3 text-xs text-amber-50 placeholder:text-neutral-600 transition focus:border-amber-400/50 focus:outline-none focus:ring-2 focus:ring-amber-400/10"
                    />
                  </div>
                  <div>
                    <label className="mb-1.5 block text-[11px] font-medium text-neutral-400">Cổ trùng nghi vấn</label>
                    <input
                      type="text"
                      value={suspectedGu}
                      onChange={(e) => setSuspectedGu(e.target.value)}
                      placeholder="VD: GU_MOONLIGHT"
                      className="min-h-10 w-full rounded-lg border border-white/8 bg-black/30 px-3 text-xs text-amber-50 placeholder:text-neutral-600 transition focus:border-amber-400/50 focus:outline-none focus:ring-2 focus:ring-amber-400/10"
                    />
                  </div>
                </div>

                <div className="rounded-2xl border border-amber-300/15 bg-[linear-gradient(145deg,rgba(65,44,21,0.28),rgba(12,12,12,0.72))] p-3.5 sm:p-4">
                  <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <span className="text-xs font-semibold text-amber-100/90">Tài nguyên đầu tư suy diễn</span>
                    {isImmortal && (
                      <div className="flex max-w-full gap-1 rounded-xl border border-white/5 bg-black/25 p-1 text-[10px] sm:text-xs">
                        <button
                          type="button"
                          onClick={() => {
                            setPlayerResourceType("primeval_stone");
                            setPlayerResourceAmount(200);
                          }}
                          className={`rounded-lg px-2.5 py-1.5 transition ${
                            playerResourceType === "primeval_stone" ? "bg-amber-500/20 text-amber-100 shadow-sm" : "text-neutral-400 hover:text-neutral-200"
                          }`}
                        >
                          Nguyên Thạch
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setPlayerResourceType("immortal_stone");
                            setPlayerResourceAmount(2);
                          }}
                          className={`rounded-lg px-2.5 py-1.5 transition ${
                            playerResourceType === "immortal_stone" ? "bg-violet-500/20 text-violet-100 shadow-sm" : "text-neutral-400 hover:text-neutral-200"
                          }`}
                        >
                          Tiên Nguyên Thạch
                        </button>
                      </div>
                    )}
                  </div>
                  <div className="flex items-center gap-3">
                    <input
                      type="range"
                      min={playerResourceType === "primeval_stone" ? 100 : 1}
                      max={playerResourceType === "primeval_stone" ? Math.max(500, Math.floor(player?.primeval_stones || 500)) : 20}
                      value={playerResourceAmount}
                      onChange={(e) => setPlayerResourceAmount(Number(e.target.value))}
                      aria-label="Mức đầu tư suy diễn Cổ Sư"
                      className="h-2 min-w-0 flex-1 cursor-pointer accent-amber-400"
                    />
                    <span className="w-28 shrink-0 text-right font-mono text-[11px] font-semibold leading-snug text-amber-200 sm:w-32 sm:text-xs">
                      {playerResourceAmount} {playerResourceType === "primeval_stone" ? "Nguyên Thạch" : "TNT"}
                    </span>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={divinationLoading || !targetUsername.trim()}
                  className="flex min-h-11 w-full items-center justify-center gap-2 rounded-xl border border-amber-200/20 bg-gradient-to-r from-amber-600 to-amber-700 py-2.5 font-semibold tracking-wide text-amber-50 shadow-[0_8px_24px_rgba(115,66,15,0.22)] transition hover:-translate-y-0.5 hover:from-amber-500 hover:to-amber-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-amber-300/60 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <Sparkles className="h-4 w-4" />
                  {divinationLoading ? "Đang Vận Chuyển Thiên Cơ..." : "Bắt Đầu Suy Diễn"}
                </button>
              </form>

              {/* Player Report */}
              {activeDivinationReport && (
                <div
                  className={`rounded-2xl border p-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] transition-all sm:p-5 ${
                    pStatus === "full"
                      ? "border-emerald-500/50 bg-emerald-950/20"
                      : pStatus === "partial"
                      ? "border-amber-500/50 bg-amber-950/20"
                      : pStatus === "backfire"
                      ? "border-red-500/50 bg-red-950/20"
                      : "border-neutral-700 bg-neutral-900"
                  }`}
                >
                  <div className="mb-3 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {pStatus === "backfire" ? (
                        <ShieldAlert className="h-5 w-5 text-red-400" />
                      ) : (
                        <Eye className="h-5 w-5 text-amber-400" />
                      )}
                      <h3 className="font-bold text-white">
                        {pStatus === "full" && "Thiên Cơ: Đại Thành Công"}
                        {pStatus === "partial" && "Thiên Cơ: Tiểu Thành Công (Mờ Ảo)"}
                        {pStatus === "fogged" && "Thiên Cơ: Thất Bại (Màn Sương)"}
                        {pStatus === "backfire" && "Thiên Cơ: Phản Phệ!"}
                      </h3>
                    </div>
                    <span className="font-mono text-xs text-neutral-400">
                      Điểm chênh lệch: {activeDivinationReport.delta_score.toFixed(1)}
                    </span>
                  </div>

                  <p className="mb-4 text-xs text-neutral-300">{activeDivinationReport.message}</p>

                  <MyScoreBreakdown score={activeDivinationReport.my_score} />

                  {pReport && (
                    <div className="grid grid-cols-2 gap-3 text-xs">
                      <div className="rounded bg-neutral-900/80 p-2.5">
                        <span className="text-neutral-400">Danh xưng:</span>{" "}
                        <span className="font-semibold text-white">{pReport.username}</span>
                      </div>
                      <div className="rounded bg-neutral-900/80 p-2.5">
                        <span className="text-neutral-400">Tu vi ước lượng:</span>{" "}
                        <span className="font-semibold text-amber-300">
                          Bậc {pReport.approx_rank} {pReport.sub_rank ? `(Chuyển ${pReport.sub_rank})` : ""}
                        </span>
                      </div>
                      <div className="rounded bg-neutral-900/80 p-2.5">
                        <span className="text-neutral-400">Vùng tọa lạc:</span>{" "}
                        <span className="font-semibold text-cyan-300">{pReport.region || "Chưa rõ"}</span>
                      </div>
                      <div className="rounded bg-neutral-900/80 p-2.5">
                        <span className="text-neutral-400">Tọa độ:</span>{" "}
                        <span className="font-semibold text-yellow-300">
                          {pReport.coordinates ? `(${pReport.coordinates.x}, ${pReport.coordinates.y})` : "Bị che giấu"}
                        </span>
                      </div>
                      <div className="col-span-2 rounded bg-neutral-900/80 p-2.5">
                        <span className="text-neutral-400">Lưu phái:</span>{" "}
                        <span className="font-semibold text-purple-300">{pReport.dominant_path || "Chưa rõ"}</span>
                      </div>
                      {pReport.detected_gus && pReport.detected_gus.length > 0 && (
                        <div className="col-span-2 rounded bg-neutral-900/80 p-2.5">
                          <span className="text-neutral-400">Cổ trùng phát hiện:</span>{" "}
                          <div className="mt-1 flex flex-wrap gap-1">
                            {pReport.detected_gus.map((g, idx) => (
                              <span key={idx} className="rounded bg-amber-900/50 px-2 py-0.5 font-mono text-[10px] text-amber-200">
                                {g}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  )}

                  {activeDivinationReport.can_apply_tracking_mark && pReport && (
                    <div className="mt-4 flex items-center justify-between border-t border-neutral-800 pt-3">
                      <span className="text-xs text-emerald-300">Đã khóa chặt khí tức! Có thể gieo ấn ký.</span>
                      <button
                        onClick={() => handleApplyMark("wisdom")}
                        disabled={applyingMark}
                        className="flex items-center gap-1 rounded-lg bg-amber-700 px-3 py-1.5 text-xs font-bold text-white hover:bg-amber-600 disabled:opacity-50"
                      >
                        <UserCheck className="h-3.5 w-3.5" />
                        Gieo Trí Niệm Ấn
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* TAB 2: GU DIVINATION */}
          {activeScoutingTab === "gu" && (
            <div className="space-y-4">
              <form onSubmit={handleGuDivinationSubmit} className="space-y-4">
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
                  <div className="sm:col-span-2">
                    <label className="mb-1.5 block text-xs font-semibold text-violet-100/85">
                      Tên hoặc ID Cổ Trùng <span className="text-red-400">*</span>
                    </label>
                    <input
                      type="text"
                      value={guQuery}
                      onChange={(e) => setGuQuery(e.target.value)}
                      placeholder="VD: Xuân Thu Thiền, Nguyệt Quang Cổ, xuan_bing_gu..."
                      required
                      className="min-h-11 w-full rounded-xl border border-violet-100/10 bg-black/35 px-3.5 py-2.5 text-sm text-violet-50 placeholder:text-neutral-600 transition focus:border-violet-400/60 focus:outline-none focus:ring-2 focus:ring-violet-400/10"
                    />
                  </div>
                  <div>
                    <label className="mb-1.5 block text-[11px] font-medium text-neutral-400">
                      Phẩm Bậc (Tùy chọn)
                    </label>
                    <select
                      value={targetRank}
                      onChange={(e) => setTargetRank(Number(e.target.value))}
                      className="min-h-11 w-full rounded-xl border border-violet-100/10 bg-black/35 px-3 py-2.5 text-xs text-violet-50 transition focus:border-violet-400/60 focus:outline-none focus:ring-2 focus:ring-violet-400/10"
                    >
                      <option value={0}>Tất cả phẩm bậc</option>
                      <option value={1}>1 Chuyển (Nhất)</option>
                      <option value={2}>2 Chuyển (Nhị)</option>
                      <option value={3}>3 Chuyển (Tam)</option>
                      <option value={4}>4 Chuyển (Tứ)</option>
                      <option value={5}>5 Chuyển (Ngũ)</option>
                      <option value={6}>6 Chuyển (Lục Tiên)</option>
                      <option value={7}>7 Chuyển (Thất Tiên)</option>
                      <option value={8}>8 Chuyển (Bát Tiên)</option>
                      <option value={9}>9 Chuyển (Cửu Tôn)</option>
                    </select>
                  </div>
                </div>

                <div className="rounded-2xl border border-violet-300/15 bg-[linear-gradient(145deg,rgba(49,33,73,0.28),rgba(12,12,12,0.72))] p-3.5 sm:p-4">
                  <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
                    <span className="text-xs font-semibold text-violet-100/90">Tài nguyên suy tính</span>
                    {isImmortal && (
                      <div className="flex max-w-full gap-1 rounded-xl border border-white/5 bg-black/25 p-1 text-[10px] sm:text-xs">
                        <button
                          type="button"
                          onClick={() => {
                            setGuResourceType("primeval_stone");
                            setGuResourceAmount(200);
                          }}
                          className={`rounded-lg px-2.5 py-1.5 transition ${
                            guResourceType === "primeval_stone" ? "bg-violet-500/20 text-violet-100 shadow-sm" : "text-neutral-400 hover:text-neutral-200"
                          }`}
                        >
                          Nguyên Thạch
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setGuResourceType("immortal_stone");
                            setGuResourceAmount(2);
                          }}
                          className={`rounded-lg px-2.5 py-1.5 transition ${
                            guResourceType === "immortal_stone" ? "bg-purple-500/20 text-purple-100 shadow-sm" : "text-neutral-400 hover:text-neutral-200"
                          }`}
                        >
                          Tiên Nguyên Thạch
                        </button>
                      </div>
                    )}
                  </div>
                  <div className="flex items-center gap-3">
                    <input
                      type="range"
                      min={guResourceType === "primeval_stone" ? 100 : 1}
                      max={guResourceType === "primeval_stone" ? Math.max(500, Math.floor(player?.primeval_stones || 500)) : 20}
                      value={guResourceAmount}
                      onChange={(e) => setGuResourceAmount(Number(e.target.value))}
                      aria-label="Mức đầu tư suy tính Cổ Trùng"
                      className="h-2 min-w-0 flex-1 cursor-pointer accent-violet-400"
                    />
                    <span className="w-28 shrink-0 text-right font-mono text-[11px] font-semibold leading-snug text-violet-200 sm:w-32 sm:text-xs">
                      {guResourceAmount} {guResourceType === "primeval_stone" ? "Nguyên Thạch" : "TNT"}
                    </span>
                  </div>
                </div>

                <button
                  type="submit"
                  disabled={guDivinationLoading || !guQuery.trim()}
                  className="flex min-h-11 w-full items-center justify-center gap-2 rounded-xl border border-violet-200/20 bg-gradient-to-r from-violet-600 to-indigo-700 py-2.5 font-semibold tracking-wide text-violet-50 shadow-[0_8px_24px_rgba(74,49,135,0.24)] transition hover:-translate-y-0.5 hover:from-violet-500 hover:to-indigo-600 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-violet-300/60 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <Sparkles className="h-4 w-4" />
                  {guDivinationLoading ? "Đang Thấu Thị Thiên Địa..." : "Suy Tính Cổ Trùng"}
                </button>
              </form>

              {/* Gu Divination Report */}
              {activeGuDivinationReport && (
                <div
                  className={`rounded-2xl border p-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)] transition-all sm:p-5 ${
                    gStatus === "full"
                      ? "border-emerald-500/50 bg-emerald-950/20"
                      : gStatus === "partial"
                      ? "border-purple-500/50 bg-purple-950/20"
                      : gStatus === "backfire"
                      ? "border-red-500/50 bg-red-950/20"
                      : "border-neutral-700 bg-neutral-900"
                  }`}
                >
                  <div className="mb-3 flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {gStatus === "backfire" ? (
                        <ShieldAlert className="h-5 w-5 text-red-400" />
                      ) : (
                        <Bug className="h-5 w-5 text-purple-400" />
                      )}
                      <h3 className="font-bold text-white">
                        {gStatus === "full" && "Suy Tính Cổ Trùng: Thấu Triệt"}
                        {gStatus === "partial" && "Suy Tính Cổ Trùng: Mờ Ảo"}
                        {gStatus === "fogged" && "Suy Tính Cổ Trùng: Bị Che Khuất"}
                        {gStatus === "backfire" && "Suy Tính Cổ Trùng: Bị Phản Phệ!"}
                      </h3>
                    </div>
                    <span className="font-mono text-xs text-neutral-400">
                      Điểm chênh lệch: {activeGuDivinationReport.delta_score.toFixed(1)}
                    </span>
                  </div>

                  <p className="mb-4 text-xs text-neutral-300">{activeGuDivinationReport.message}</p>

                  <MyScoreBreakdown score={activeGuDivinationReport.my_score} />

                  {gReport && (
                    <div className="space-y-3 text-xs">
                      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                        <div className="rounded bg-neutral-900/80 p-2.5">
                          <span className="text-neutral-400">Tên Cổ:</span>{" "}
                          <span className="font-semibold text-white">{gReport.gu_name}</span>
                        </div>
                        <div className="rounded bg-neutral-900/80 p-2.5">
                          <span className="text-neutral-400">Phẩm Bậc:</span>{" "}
                          <span className={`font-semibold ${gReport.rank >= 6 ? "text-purple-400" : "text-amber-300"}`}>
                            {gReport.rank >= 6 ? `Lục Tiên+ (${gReport.rank} Chuyển)` : `${gReport.rank} Chuyển`}
                          </span>
                        </div>
                        <div className="rounded bg-neutral-900/80 p-2.5">
                          <span className="text-neutral-400">Lưu phái:</span>{" "}
                          <span className="font-semibold text-cyan-300">{gReport.path || "Tự nhiên"}</span>
                        </div>
                        <div className="rounded bg-neutral-900/80 p-2.5">
                          <span className="text-neutral-400">Tổng Số Lượng:</span>{" "}
                          <span className="font-semibold text-yellow-300">
                            {gReport.world_count} con
                          </span>
                        </div>
                      </div>

                      {/* Multi-Domain Breakdown Chips */}
                      <div className="grid grid-cols-3 gap-2 text-center text-xs">
                        <div className="rounded-lg border border-amber-500/20 bg-neutral-900/70 p-2">
                          <div className="flex items-center justify-center gap-1 text-amber-400 mb-0.5">
                            <User className="h-3.5 w-3.5" />
                            <span className="font-semibold">Cổ Sư Sở Hữu</span>
                          </div>
                          <span className="text-base font-bold text-white">{gReport.player_count ?? 0}</span>
                        </div>
                        <div className="rounded-lg border border-purple-500/20 bg-neutral-900/70 p-2">
                          <div className="flex items-center justify-center gap-1 text-purple-400 mb-0.5">
                            <Building2 className="h-3.5 w-3.5" />
                            <span className="font-semibold">Kho Tông Môn</span>
                          </div>
                          <span className="text-base font-bold text-white">{gReport.sect_vault_count ?? 0}</span>
                        </div>
                        <div className="rounded-lg border border-emerald-500/20 bg-neutral-900/70 p-2">
                          <div className="flex items-center justify-center gap-1 text-emerald-400 mb-0.5">
                            <Store className="h-3.5 w-3.5" />
                            <span className="font-semibold">Sàn Đấu Giá</span>
                          </div>
                          <span className="text-base font-bold text-white">{gReport.market_count ?? 0}</span>
                        </div>
                      </div>

                      {/* Status Lore Banner */}
                      <div className={`rounded-lg p-3 ${gReport.is_refined ? "bg-amber-950/40 border border-amber-500/30 text-amber-200" : "bg-emerald-950/40 border border-emerald-500/30 text-emerald-200"}`}>
                        <span className="font-bold">Trạng thái Thiên Địa: </span>
                        {gReport.lore_status}
                      </div>

                      {/* Recipe & Food */}
                      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                        <div className="rounded bg-neutral-900/80 p-3">
                          <div className="flex items-center gap-1.5 font-semibold text-neutral-300 mb-1">
                            <BookOpen className="h-3.5 w-3.5 text-amber-400" />
                            <span>Bí Phương Luyện Chế</span>
                          </div>
                          <p className="text-neutral-400">{gReport.recipe_hint}</p>
                        </div>
                        <div className="rounded bg-neutral-900/80 p-3">
                          <div className="flex items-center gap-1.5 font-semibold text-neutral-300 mb-1">
                            <Bug className="h-3.5 w-3.5 text-emerald-400" />
                            <span>Thức Ăn & Sinh Trưởng</span>
                          </div>
                          <p className="text-neutral-400">{gReport.food_hint}</p>
                        </div>
                      </div>

                      {/* Multi-Domain Holders list */}
                      {gReport.holders && gReport.holders.length > 0 ? (
                        <div className="rounded bg-neutral-900/80 p-3">
                          <div className="flex items-center justify-between mb-2">
                            <span className="font-semibold text-neutral-300">Vết Tích Nơi Lưu Trữ Trong Thiên Địa:</span>
                            <span className="text-[11px] text-neutral-500">{gReport.holders.length} vị trí phát hiện</span>
                          </div>
                          <div className="space-y-2">
                            {gReport.holders.map((h, idx) => {
                              const isPlayer = h.holder_type === "player" || (!h.holder_type && h.holder_username);
                              const isSectVault = h.holder_type === "sect_vault";
                              const isAuction = h.holder_type === "auction_listing";

                              return (
                                <div
                                  key={idx}
                                  className="flex items-center justify-between rounded bg-neutral-950 p-2.5 border border-neutral-800 hover:border-neutral-700 transition"
                                >
                                  <div className="space-y-1">
                                    <div className="flex items-center gap-2">
                                      {isPlayer && (
                                        <span className="flex items-center gap-1 rounded bg-amber-950/80 px-2 py-0.5 text-[10px] text-amber-300 border border-amber-800/40">
                                          <User className="h-3 w-3" /> Cổ Sư
                                        </span>
                                      )}
                                      {isSectVault && (
                                        <span className="flex items-center gap-1 rounded bg-purple-950/80 px-2 py-0.5 text-[10px] text-purple-300 border border-purple-800/40">
                                          <Building2 className="h-3 w-3" /> Kho Tông Môn
                                        </span>
                                      )}
                                      {isAuction && (
                                        <span className="flex items-center gap-1 rounded bg-emerald-950/80 px-2 py-0.5 text-[10px] text-emerald-300 border border-emerald-800/40">
                                          <Store className="h-3 w-3" /> Sàn Giao Dịch
                                        </span>
                                      )}
                                      <span className="font-bold text-white">
                                        {h.holder_username || h.holder_name || "Thế lực ẩn danh"}
                                      </span>
                                      {h.quantity && h.quantity > 1 && (
                                        <span className="rounded bg-neutral-800 px-1.5 py-0.2 text-[10px] font-mono text-yellow-300">
                                          x{h.quantity}
                                        </span>
                                      )}
                                    </div>
                                    <div className="flex items-center gap-2 text-[11px] text-neutral-400">
                                      {h.region && <span>Vùng: <strong className="text-neutral-300">{h.region}</strong></span>}
                                      {h.is_vital && <span className="text-rose-400 font-semibold">[Bản Mệnh Cổ]</span>}
                                      {h.equipped && <span className="text-amber-400">[Đang Trang Bị]</span>}
                                      {h.location_state && !h.equipped && !h.is_vital && (
                                        <span className="text-neutral-400">[{h.location_state === "inventory" ? "Trong Túi" : h.location_state === "sleeping" ? "Ngủ Say" : h.location_state}]</span>
                                      )}
                                      {h.price_info && (
                                        <span className="text-yellow-400 font-mono">[{h.price_info}]</span>
                                      )}
                                    </div>
                                  </div>
                                  {h.coordinates ? (
                                    <div className="flex items-center gap-1 font-mono text-xs text-cyan-300 bg-cyan-950/40 border border-cyan-800/40 rounded px-2 py-1">
                                      <MapPin className="h-3 w-3 text-cyan-400" />
                                      <span>({h.coordinates.x}, {h.coordinates.y})</span>
                                    </div>
                                  ) : (
                                    <span className="text-[10px] text-neutral-600 italic">Tọa độ mờ ảo</span>
                                  )}
                                </div>
                              );
                            })}
                          </div>
                        </div>
                      ) : gReport.world_count > 0 ? (
                        <div className="rounded bg-neutral-900/80 p-3 text-center text-neutral-500 italic">
                          Cổ trùng đã xuất thế trong thiên địa nhưng khí tức bị che giấu hoặc chưa đủ độ chính xác để định vị danh xưng.
                        </div>
                      ) : null}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* TAB 3: RADAR TRACKED TARGETS */}
          {activeScoutingTab === "radar" && (
            <div className="space-y-4">
              <div className="flex items-center justify-between gap-3 rounded-2xl border border-cyan-400/15 bg-[linear-gradient(110deg,rgba(8,55,64,0.28),rgba(17,17,17,0.75))] p-3.5 sm:p-4">
                <div className="flex min-w-0 items-center gap-2.5 text-cyan-200">
                  <div className="grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-cyan-300/20 bg-cyan-950/50">
                    <Radar className="h-4 w-4 text-cyan-300" />
                  </div>
                  <span className="text-xs font-semibold sm:text-sm">Mục Tiêu Đang Bị Khóa Khí Tức</span>
                </div>
                <button
                  onClick={() => fetchTrackedTargets()}
                  type="button"
                  className="shrink-0 rounded-lg border border-cyan-200/10 bg-black/25 px-3 py-2 text-[11px] font-medium text-cyan-100/80 transition hover:border-cyan-300/30 hover:bg-cyan-950/50"
                >
                  Làm Mới
                </button>
              </div>

              {trackedTargets.length === 0 ? (
                <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-cyan-200/15 bg-cyan-950/[0.08] px-5 py-10 text-center text-neutral-500">
                  <div className="mb-3 grid h-14 w-14 place-items-center rounded-2xl border border-cyan-200/10 bg-black/20">
                    <Radar className="h-7 w-7 text-cyan-300/60" />
                  </div>
                  <p className="text-xs font-medium text-neutral-300">Thiên Cơ chưa khóa mục tiêu nào</p>
                  <p className="mt-1 max-w-sm text-[10px] leading-relaxed text-neutral-500">
                    Hãy dùng Suy Diễn Cổ Sư mức độ cao hoặc sát chiêu chiến đấu để gieo ấn ký.
                  </p>
                </div>
              ) : (
                <div className="space-y-2.5">
                  {trackedTargets.map((target) => (
                    <div
                      key={target.id}
                      className="flex flex-col gap-3 rounded-2xl border border-cyan-300/15 bg-[linear-gradient(110deg,rgba(8,38,44,0.35),rgba(12,12,12,0.72))] p-3.5 transition hover:border-cyan-300/35 sm:flex-row sm:items-center sm:justify-between sm:p-4"
                    >
                      <div className="min-w-0 space-y-1">
                        <div className="flex items-center gap-2">
                          <span className="font-bold text-white">{target.target_username || "Mục tiêu bí ẩn"}</span>
                          <span className="rounded bg-cyan-950 px-2 py-0.5 text-[10px] text-cyan-300 uppercase border border-cyan-800/40">
                            {target.mark_type}
                          </span>
                        </div>
                        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-neutral-400">
                          <div className="flex min-w-0 items-center gap-1">
                            <MapPin className="h-3 w-3 text-cyan-400" />
                            <span>{target.current_region || "Chưa rõ"}</span>
                            {target.x !== undefined && target.y !== undefined && (
                              <span className="font-mono text-cyan-300">({target.x}, {target.y})</span>
                            )}
                          </div>
                          <div className="flex items-center gap-1">
                            <Clock className="h-3 w-3 text-amber-400" />
                            <span>Hết hạn: {new Date(target.expires_at).toLocaleTimeString()}</span>
                          </div>
                        </div>
                      </div>

                      {target.x !== undefined && target.y !== undefined && (
                        <button
                          onClick={() => handleLocateTarget(target)}
                          type="button"
                          className="flex min-h-10 items-center justify-center gap-1.5 rounded-xl border border-cyan-200/15 bg-cyan-700/80 px-4 py-2 text-xs font-semibold text-white shadow transition hover:bg-cyan-600 sm:min-h-9"
                        >
                          <MapPin className="h-3.5 w-3.5" />
                          Định Vị
                        </button>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* TAB 4: SEVER CLUES & CONCEALMENT */}
          {activeScoutingTab === "sever" && (
            <div className="space-y-4">
              <div className="rounded-2xl border border-rose-300/20 bg-[radial-gradient(ellipse_at_0%_0%,rgba(127,29,29,0.2),transparent_60%),rgba(30,10,12,0.42)] p-4 sm:p-5">
                <div className="mb-2 flex items-center gap-2.5 text-rose-200">
                  <div className="grid h-9 w-9 place-items-center rounded-xl border border-rose-300/20 bg-rose-950/50">
                    <Scissors className="h-4 w-4" />
                  </div>
                  <h3 className="font-serif text-sm font-bold sm:text-base">Chém Đứt Manh Mối</h3>
                </div>
                <p className="pl-0 text-xs leading-relaxed text-neutral-300 sm:pl-11">
                  Thi triển sát chiêu Trí Đạo / Ám Đạo để xóa sạch toàn bộ dấu vết di chuyển, chiến đấu còn lưu lại trong thiên địa, đồng thời phá vỡ các ấn ký truy tung đang bám trên thân mình.
                </p>
                <div className="mt-4 flex flex-col gap-3 border-t border-rose-200/10 pt-3 sm:flex-row sm:items-center sm:justify-between">
                  <span className="text-[11px] leading-relaxed text-neutral-400">Tiêu hao Chân Nguyên / Thần Hồn để tẩy sạch nhân quả.</span>
                  <button
                    onClick={handleSeverClues}
                    type="button"
                    disabled={severing}
                    className="flex min-h-10 items-center justify-center gap-1.5 rounded-xl border border-rose-200/15 bg-rose-700/80 px-4 py-2 text-xs font-semibold text-white shadow-lg transition hover:bg-rose-600 disabled:cursor-not-allowed disabled:opacity-50 sm:min-h-9"
                  >
                    <Scissors className="h-4 w-4" />
                    {severing ? "Đang Thanh Tẩy..." : "Chém Đứt Toàn Bộ Manh Mối"}
                  </button>
                </div>
              </div>

              <div className="rounded-2xl border border-amber-200/10 bg-black/20 p-4">
                <div className="mb-3 flex items-center gap-2 text-neutral-200">
                  <Shield className="h-4 w-4 text-amber-300" />
                  <span className="text-xs font-semibold">Trạng Thái Phòng Ngự Thiên Cơ</span>
                </div>
                <div className="grid grid-cols-2 gap-3 text-xs">
                  <div className="rounded-xl border border-white/5 bg-black/25 p-3">
                    <span className="text-neutral-400">Ngụy trang:</span>{" "}
                    <span className="font-semibold text-white">
                      {player?.is_disguised ? "Đang Bật Ngụy Trang" : "Bình Thường"}
                    </span>
                  </div>
                  <div className="rounded-xl border border-white/5 bg-black/25 p-3">
                    <span className="text-neutral-400">Lớp Che Giấu:</span>{" "}
                    <span className="font-semibold text-emerald-400">
                      {player?.active_concealment ? "Đang Kích Hoạt" : "Không"}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
