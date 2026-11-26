"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import type { EnvDaoStatus } from "@/store/slices/worldSlice";

interface EnvDaoPanelProps {
  regionId?: string;
  compactBanner?: boolean;
}

export const EnvDaoPanel: React.FC<EnvDaoPanelProps> = ({ regionId, compactBanner = false }) => {
  const player = useGameStore((s) => s.player);
  const envDaoStatus = useGameStore((s) => s.envDaoStatus) as EnvDaoStatus | null;
  const fetchEnvDaoStatus = useGameStore((s) => s.fetchEnvDaoStatus);
  const refineEnvDao = useGameStore((s) => s.refineEnvDao);
  const releaseEnvDao = useGameStore((s) => s.releaseEnvDao);
  const [selectedPath, setSelectedPath] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void fetchEnvDaoStatus(regionId);
  }, [regionId, player?.current_region, fetchEnvDaoStatus]);

  if (!envDaoStatus) return null;

  const dom = envDaoStatus.dominion;
  const isOwner = !!dom && !!player?.id && dom.owner_player_id === player.id;
  const isVenerable = (player?.rank ?? 0) >= 9;
  const naturalEntries = Object.entries(envDaoStatus.natural_dao_marks || {})
    .filter(([, v]) => v > 0)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 3);

  if (compactBanner) {
    return (
      <div className="mb-3 rounded-lg border border-amber-500/30 bg-amber-950/20 px-3 py-2 text-xs text-amber-100 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="font-semibold text-amber-300">Môi Trường:</span>
          <span>{envDaoStatus.region_name}</span>
          {dom && dom.control_pct >= envDaoStatus.domain_activation ? (
            <span className="rounded-full border border-amber-400/50 bg-amber-500/15 px-2 py-0.5 text-[11px] font-semibold text-amber-200">
              Đạo Vực {dom.path} ({dom.owner_name} · {Math.round(dom.control_pct)}%)
            </span>
          ) : naturalEntries.length > 0 ? (
            <span className="text-amber-200/80">
              Đạo Ngân Tự Nhiên: {naturalEntries.map(([k, v]) => `${k} (${Math.round(v)})`).join(", ")}
            </span>
          ) : (
            <span className="text-slate-400">Đạo Ngân cân bằng</span>
          )}
        </div>
        {dom && dom.control_pct >= envDaoStatus.domain_activation && (
          <span className={isOwner ? "text-emerald-300 font-semibold" : "text-rose-300 font-semibold"}>
            {isOwner ? "Bá chủ Đạo Vực (Ưu thế áp đảo)" : "Đang chịu áp chế Đạo Vực"}
          </span>
        )}
      </div>
    );
  }

  const level4Paths = Object.entries(player?.path_masteries || {})
    .filter(([, pts]) => Number(pts) >= 10_000_000)
    .map(([p]) => p);

  const handleRefine = async () => {
    setBusy(true);
    try {
      await refineEnvDao(selectedPath || level4Paths[0] || "");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="rounded-xl border border-amber-500/25 bg-slate-950/80 p-3 text-xs text-slate-200 space-y-2">
      <div className="flex items-center justify-between gap-2">
        <span className="font-semibold text-amber-300">Đạo Ngân Môi Trường · {envDaoStatus.region_name}</span>
        {envDaoStatus.is_exempt ? (
          <span className="rounded-full border border-slate-600 px-2 py-0.5 text-[10px] text-slate-400">Khu vực miễn trừ</span>
        ) : dom ? (
          <span className="rounded-full border border-amber-400/40 bg-amber-500/10 px-2 py-0.5 text-[10px] text-amber-200">
            {dom.owner_name} · {dom.path} ({Math.round(dom.control_pct)}%)
          </span>
        ) : (
          <span className="rounded-full border border-sky-500/30 bg-sky-500/10 px-2 py-0.5 text-[10px] text-sky-300">Vô chủ</span>
        )}
      </div>

      {naturalEntries.length > 0 && (
        <div className="text-slate-400">
          Đạo Ngân Tự Nhiên:{" "}
          <span className="text-slate-200">{naturalEntries.map(([k, v]) => `${k} (${Math.round(v).toLocaleString()})`).join(" · ")}</span>
        </div>
      )}

      {dom && (
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-800">
          <div
            className="h-full bg-gradient-to-r from-amber-600 to-amber-300 transition-all"
            style={{ width: `${Math.min(100, Math.max(0, dom.control_pct))}%` }}
          />
        </div>
      )}

      {isVenerable && !envDaoStatus.is_exempt && (
        <div className="flex flex-wrap items-center gap-2 pt-1 border-t border-slate-800/80">
          <span className="text-[11px] text-amber-200/80">
            Đạo Vực ({envDaoStatus.my_domains?.length ?? 0}/{envDaoStatus.max_domains})
          </span>
          {level4Paths.length > 1 && !dom && (
            <select
              value={selectedPath || level4Paths[0]}
              onChange={(e) => setSelectedPath(e.target.value)}
              className="rounded border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] text-slate-200"
            >
              {level4Paths.map((p) => (
                <option key={p} value={p}>{p}</option>
              ))}
            </select>
          )}
          <button
            type="button"
            disabled={busy || (isOwner && (dom?.control_pct ?? 0) >= 100)}
            onClick={handleRefine}
            className={`rounded px-2.5 py-1 text-[11px] font-semibold transition ${
              dom && !isOwner
                ? "bg-rose-900/70 text-rose-100 border border-rose-500/50 hover:bg-rose-800"
                : "bg-amber-700/80 text-amber-50 border border-amber-400/50 hover:bg-amber-600"
            } disabled:opacity-40`}
          >
            {dom && !isOwner
              ? "Phá Vực (-25%)"
              : isOwner
                ? (dom?.control_pct ?? 0) >= 100
                  ? "Đã hoàn thiện (100%)"
                  : "Gia Cố Đạo Vực (+25%)"
                : "Luyện Hóa Đạo Ngân (+25%)"}
          </button>
          {isOwner && (
            <button
              type="button"
              disabled={busy}
              onClick={() => releaseEnvDao(envDaoStatus.region_id)}
              className="rounded border border-slate-700 bg-slate-900 px-2 py-1 text-[11px] text-slate-300 hover:border-slate-500"
            >
              Giải Phóng
            </button>
          )}
        </div>
      )}
    </div>
  );
};
