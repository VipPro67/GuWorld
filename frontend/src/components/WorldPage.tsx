"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { useWorldStore } from "@/store/useWorldStore";
import { useGameStore } from "@/store/useGameStore";
import { WorldLogViewer } from "./WorldLogViewer";

const WorldCanvas = dynamic(() => import("@/components/WorldCanvas"), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full flex items-center justify-center text-zinc-500 font-serif">
      Đang tải thế giới...
    </div>
  ),
});

const TYPE_CONFIG: Record<string, { icon: string; label: string; color: string; borderColor: string; bgColor: string }> = {
  Boss:     { icon: "⚔️",  label: "Boss",        color: "text-red-300",     borderColor: "border-red-500/30",     bgColor: "bg-red-950/60" },
  Monster:  { icon: "🐺",  label: "Yêu Thú",     color: "text-red-300",     borderColor: "border-red-500/30",     bgColor: "bg-red-950/60" },
  Resource: { icon: "💎",  label: "Linh Tài",    color: "text-emerald-300", borderColor: "border-emerald-500/30", bgColor: "bg-emerald-950/60" },
  WildGu:   { icon: "🐛",  label: "Cổ Trùng",    color: "text-purple-300",  borderColor: "border-purple-500/30",  bgColor: "bg-purple-950/60" },
  Empty:    { icon: "🏯",  label: "Địa Điểm",    color: "text-amber-300",   borderColor: "border-amber-500/30",   bgColor: "bg-amber-950/60" },
  Player:   { icon: "🥷",  label: "Cổ Sư",       color: "text-purple-300",  borderColor: "border-purple-500/30",  bgColor: "bg-purple-950/60" },
};
const DEFAULT_CONFIG = { icon: "📌", label: "Địa Điểm", color: "text-zinc-300", borderColor: "border-zinc-500/30", bgColor: "bg-zinc-800/60" };

export function WorldPage() {
  const regions = useGameStore((s) => s.regions);
  const regionsLoaded = useGameStore((s) => s.regionsLoaded);
  const fetchRegions = useGameStore((s) => s.fetchRegions);
  const gamePlayer = useGameStore((s) => s.player);
  const error = useWorldStore((s) => s.error);
  const loading = useWorldStore((s) => s.loading);
  const selectedObject = useWorldStore((s) => s.selectedObject);
  const selectObject = useWorldStore((s) => s.selectObject);
  const isMoving = useWorldStore((s) => s.isMoving);
  const setMovementRange = useWorldStore((s) => s.setMovementRange);
  const liveX = useWorldStore((s) => s.liveX);
  const liveY = useWorldStore((s) => s.liveY);
  const fetchGrid = useGameStore((s) => s.fetchGrid);
  
  const monsterDictionary = useGameStore((s) => s.monsterDictionary);
  const itemDictionary = useGameStore((s) => s.itemDictionary);
  const guDictionary = useGameStore((s) => s.guDictionary);
  const fetchPlayerProfile = useGameStore((s) => s.fetchPlayerProfile);
  const movePlayer = useGameStore((s) => s.movePlayer);

  const [mapMode, setMapMode] = useState<"select" | "drag">("select");

  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
  }, [regionsLoaded, fetchRegions]);

  const setInvestigationRange = useWorldStore((s) => s.setInvestigationRange);

  // Movement and Investigation ranges come from the same stats as the grid viewport.
  useEffect(() => {
    fetchGrid().then((data) => {
      if (data?.stats) {
        if (data.stats.movement) setMovementRange(data.stats.movement);
        if (data.stats.investigation) setInvestigationRange(data.stats.investigation);
      }
    });
  }, [fetchGrid, setMovementRange, setInvestigationRange]);

  const currentZone = gamePlayer?.current_region
    ? regions.find((r) => r.id === gamePlayer.current_region)
    : undefined;

  const cfg = selectedObject ? TYPE_CONFIG[selectedObject.cell_type] || DEFAULT_CONFIG : DEFAULT_CONFIG;

  return (
    <div className="w-full h-full relative">
      {loading && (
        <div className="absolute top-20 sm:top-16 left-1/2 -translate-x-1/2 px-4 py-2 bg-zinc-900/80 rounded-xl border border-zinc-800 text-xs text-zinc-400 font-serif">
          Đang khám phá thế giới...
        </div>
      )}
      {error && (
        <div className="absolute top-20 sm:top-16 left-1/2 -translate-x-1/2 px-4 py-2 bg-red-950/80 rounded-xl border border-red-800/40 text-xs text-red-400 font-serif">
          {error}
        </div>
      )}

      <div className="w-full h-[calc(100vh-8rem)] min-h-[480px]">
        <WorldCanvas mode={mapMode} />
      </div>

      <div className="absolute top-20 sm:top-16 left-4 flex flex-col gap-2 pointer-events-none">
        {gamePlayer && (
          <div className="px-3 py-1.5 flex flex-col items-center bg-zinc-900/85 backdrop-blur-sm rounded-xl border border-zinc-800/40 text-xs text-zinc-300 font-serif shadow-lg pointer-events-auto">
            <div className="text-emerald-400 font-bold text-center">
              {currentZone?.name || gamePlayer.current_region || "Ngoài bản đồ"}
            </div>
            <div className="text-[10px] text-zinc-400 font-mono mt-0.5">
              X:{Math.round(liveX)} Y:{Math.round(liveY)}
            </div>
          </div>
        )}

        <div className="flex flex-col gap-1.5 p-1.5 bg-zinc-900/85 backdrop-blur-sm rounded-xl border border-zinc-800/40 shadow-lg pointer-events-auto w-fit">
          <button
            onClick={() => setMapMode("select")}
            title="Di chuyển"
            className={`w-9 h-9 flex items-center justify-center rounded-lg transition-all cursor-pointer ${
              mapMode === "select"
                ? "bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 shadow-inner"
                : "text-zinc-400 hover:text-zinc-200 border border-transparent hover:bg-zinc-800/50"
            }`}
          >
            <span className="text-lg">🎯</span>
          </button>
          <button
            onClick={() => setMapMode("drag")}
            title="Xem bản đồ"
            className={`w-9 h-9 flex items-center justify-center rounded-lg transition-all cursor-pointer ${
              mapMode === "drag"
                ? "bg-amber-500/20 text-amber-300 border border-amber-500/30 shadow-inner"
                : "text-zinc-400 hover:text-zinc-200 border border-transparent hover:bg-zinc-800/50"
            }`}
          >
            <span className="text-lg">🖐️</span>
          </button>
          <div className="border-t border-zinc-700/40 my-0.5 mx-1" />
          <button
            onClick={() => useWorldStore.getState().requestCenterCamera()}
            title="Về vị trí"
            className="w-9 h-9 flex items-center justify-center rounded-lg transition-all cursor-pointer text-sky-400 hover:text-sky-300 border border-transparent hover:bg-sky-500/10"
          >
            <span className="text-lg">📍</span>
          </button>
        </div>
      </div>
      {selectedObject && (
        <div className="absolute bottom-28 left-1/2 -translate-x-1/2 pointer-events-auto animate-fade-in z-20">
          <div className={`relative w-72 px-4 py-4 rounded-2xl border ${cfg.borderColor} ${cfg.bgColor} backdrop-blur-md shadow-2xl`}>
            <button
              onClick={() => selectObject(null)}
              className="absolute top-2 right-2 w-6 h-6 flex items-center justify-center rounded-full bg-zinc-800/60 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs transition-colors cursor-pointer"
            >
              ✕
            </button>

            <div className="flex items-center gap-2.5 mb-3">
              <div className={`w-9 h-9 rounded-xl ${cfg.bgColor} border ${cfg.borderColor} flex items-center justify-center text-lg shrink-0`}>
                {cfg.icon}
              </div>
              <div>
                <div className={`font-bold font-serif text-sm ${cfg.color}`}>
                  {selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss"
                    ? monsterDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "Resource"
                    ? itemDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "WildGu"
                    ? guDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.content_id}
                </div>
                <div className="text-[10px] text-zinc-500 font-serif">
                  {cfg.label} {selectedObject.rank ? `• Rank ${selectedObject.rank}` : ""}
                </div>
              </div>
            </div>

            <div className="flex gap-3 text-[10px] text-zinc-500 font-mono mb-3">
              <span>X: {selectedObject.x}</span>
              <span>Y: {selectedObject.y}</span>
            </div>

            {selectedObject.cell_type === "Boss" && selectedObject.cleared && (
              <div className="px-2 py-1.5 rounded-lg bg-zinc-800/60 text-[10px] text-zinc-400 font-serif mb-3">
                💀 Quái vật đang hồi sinh — hãy quay lại sau.
              </div>
            )}

            <div className="flex gap-2">
              {selectedObject.cell_type === "Player" ? (
                <>
                  <button
                    disabled={isMoving}
                    onClick={async () => {
                      if (gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y) {
                        movePlayer(selectedObject.x, selectedObject.y, selectedObject.id);
                      } else {
                        window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                      }
                    }}
                    className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl bg-red-600 hover:bg-red-500 shadow-red-950/30 disabled:opacity-60 text-white text-xs font-serif font-bold transition-all cursor-pointer shadow-lg"
                  >
                    <span>⚔️</span>
                    <span>{isMoving ? "Đang di chuyển..." : (gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y) ? "Tấn Công" : "Tiếp Cận"}</span>
                  </button>
                  <button
                    onClick={() => fetchPlayerProfile(selectedObject.content_id)}
                    className="flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl bg-purple-600 hover:bg-purple-500 shadow-purple-950/30 text-white text-xs font-serif font-bold transition-all cursor-pointer shadow-lg"
                  >
                    <span>👁️</span>
                    <span>Trinh Thám</span>
                  </button>
                </>
              ) : (
                <button
                  disabled={isMoving || !!selectedObject.cleared}
                  onClick={() => {
                    window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                  }}
                  className={`flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-xl ${
                    selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss"
                      ? "bg-red-600 hover:bg-red-500 shadow-red-950/30"
                      : "bg-emerald-600 hover:bg-emerald-500 shadow-emerald-950/30"
                  } disabled:opacity-60 text-white text-xs font-serif font-bold transition-all cursor-pointer shadow-lg`}
                >
                  <span>{selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss" ? "⚔️" : "🏃"}</span>
                  <span>{isMoving ? "Đang di chuyển..." : selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss" ? "Chiến Đấu" : "Tiếp Cận"}</span>
                </button>
              )}
              <button
                onClick={() => selectObject(null)}
                className="px-3 py-2 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs font-serif transition-all cursor-pointer border border-zinc-700/40"
              >
                Đóng
              </button>
            </div>
          </div>
        </div>
      )}

      <WorldLogViewer />
    </div>
  );
}
