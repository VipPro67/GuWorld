"use client";

import { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useGameStore } from "@/store/useGameStore";
import { Clock3, DoorOpen, Globe2, RefreshCw, Users } from "lucide-react";

interface SpatialRealm {
  template_id: string;
  name: string;
  description: string;
  required_rank: number;
  region_id: string;
  entry_region_id: string;
  entry_x: number;
  entry_y: number;
  duration_minutes: number;
  instance_id?: string;
  opened_at?: string;
  expires_at?: string;
  player_count: number;
}

export function SharedSpatialRealmControls({ enabled, currentRegion, x, y }: { enabled: boolean; currentRegion: string; x: number; y: number }) {
  const token = useGameStore((state) => state.token);
  const regions = useGameStore((state) => state.regions);
  const logout = useGameStore((state) => state.logout);
  const refreshPlayerSummary = useGameStore((state) => state.refreshPlayerSummary);
  const addToast = useGameStore((state) => state.addToast);
  const [realms, setRealms] = useState<SpatialRealm[]>([]);
  const [activeRealm, setActiveRealm] = useState<SpatialRealm | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    if (!token || !enabled) return;
    setLoading(true);
    try {
      const [listResponse, activeResponse] = await Promise.all([
        authFetch(`${BACKEND_URL}/api/instance/spatial`),
        authFetch(`${BACKEND_URL}/api/instance/spatial/active`),
      ]);
      if (listResponse.status === 401 || activeResponse.status === 401) {
        logout();
        return;
      }
      if (listResponse.ok) {
        const data = await listResponse.json();
        setRealms(data.realms || []);
      }
      if (activeResponse.ok) {
        const data = await activeResponse.json();
        setActiveRealm(data.realm || null);
      }
    } catch {
      // Keep the last snapshot visible during a brief network interruption.
    } finally {
      setLoading(false);
    }
  }, [enabled, logout, token]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    if (!enabled) return;
    const timer = window.setInterval(() => { void refresh(); }, 30000);
    return () => window.clearInterval(timer);
  }, [enabled, refresh]);

  const enter = async (templateID: string) => {
    if (!token || busy) return;
    setBusy(true);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/spatial/enter`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ template_id: templateID }),
      });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể vào bí cảnh.");
      await refreshPlayerSummary();
      await refresh();
      addToast("Đã tiến vào phiên bí cảnh chung.", "success");
    } catch (caught) {
      addToast(caught instanceof Error ? caught.message : "Không thể vào bí cảnh.", "error");
    } finally {
      setBusy(false);
    }
  };

  const exitAtPortal = async () => {
    if (!token || busy) return;
    setBusy(true);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/instance/spatial/exit`, { method: "POST" });
      if (response.status === 401) { logout(); return; }
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Hãy đến cổng thoát của bí cảnh.");
      if (!data.exited) return;
      await refreshPlayerSummary();
      await refresh();
      addToast("Đã trở về vị trí trước khi vào bí cảnh.", "success");
    } catch (caught) {
      addToast(caught instanceof Error ? caught.message : "Không thể thoát bí cảnh.", "error");
    } finally {
      setBusy(false);
    }
  };

  if (!enabled || (realms.length === 0 && !activeRealm)) return null;

  const secondsLeft = activeRealm?.expires_at ? Math.max(0, Math.floor((Date.parse(activeRealm.expires_at) - Date.now()) / 1000)) : null;

  return (
    <section className="rounded-xl border border-violet-900/50 bg-violet-950/15 p-3 space-y-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-violet-200">
          <Globe2 className="h-4 w-4" />
          <h3 className="text-xs font-bold">Bí Cảnh Bản Đồ · Chung Toàn Server</h3>
        </div>
        <button type="button" onClick={() => void refresh()} disabled={loading} className="rounded-md p-1 text-zinc-400 hover:text-white disabled:opacity-50" aria-label="Làm mới bí cảnh">
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
        </button>
      </div>

      {activeRealm ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-violet-900/40 bg-zinc-950/40 p-3">
          <div className="min-w-0">
            <p className="truncate text-xs font-bold text-zinc-100">Đang ở: {activeRealm.name}</p>
            <div className="mt-1 flex flex-wrap gap-x-3 text-[10px] text-zinc-400">
              <span className="inline-flex items-center gap-1"><Users className="h-3 w-3" />{activeRealm.player_count} người</span>
              {secondsLeft !== null && <span className="inline-flex items-center gap-1"><Clock3 className="h-3 w-3" />Còn {Math.floor(secondsLeft / 60)} phút</span>}
            </div>
          </div>
          <button type="button" onClick={() => void exitAtPortal()} disabled={busy} className="inline-flex items-center gap-1.5 rounded-lg border border-violet-800/60 px-3 py-2 text-[10px] font-bold text-violet-200 hover:bg-violet-900/30 disabled:opacity-50">
            <DoorOpen className="h-3.5 w-3.5" /> Thoát tại cổng
          </button>
        </div>
      ) : (
        <div className="grid gap-2 md:grid-cols-2">
          {realms.map((realm) => {
            const atGate = currentRegion === realm.entry_region_id && x === realm.entry_x && y === realm.entry_y;
            const entryRegionName = regions.find((region) => region.id === realm.entry_region_id)?.name || realm.entry_region_id;
            return (
              <article key={realm.template_id} className="rounded-lg border border-zinc-800 bg-zinc-950/40 p-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h4 className="truncate text-xs font-bold text-zinc-100">{realm.name}</h4>
                    <p className="mt-1 line-clamp-2 text-[10px] text-zinc-400">{realm.description}</p>
                    <p className="mt-1 text-[10px] text-violet-300">Cổng tại {entryRegionName} · ({realm.entry_x}, {realm.entry_y})</p>
                    <div className="mt-2 flex gap-3 text-[10px] text-zinc-500">
                      <span>Rank {realm.required_rank}+</span>
                      <span>{realm.duration_minutes} phút</span>
                      {realm.instance_id && <span className="inline-flex items-center gap-1"><Users className="h-3 w-3" />{realm.player_count} đang ở</span>}
                    </div>
                  </div>
                  <button type="button" onClick={() => void enter(realm.template_id)} disabled={busy || !atGate} className="shrink-0 rounded-lg bg-violet-700 px-3 py-2 text-[10px] font-bold text-white hover:bg-violet-600 disabled:opacity-50">
                    {atGate ? (realm.instance_id ? "Tham gia" : "Tiến vào") : "Đến cổng"}
                  </button>
                </div>
              </article>
            );
          })}
        </div>
      )}
      <p className="text-[10px] text-zinc-500">Cần đi tới đúng cổng được đánh dấu trên bản đồ thế giới mới vào được. Người chơi cùng bí cảnh sẽ gặp nhau trong một bản đồ; có thể rời bằng Cổ Trùng thoát hoặc đến cổng ra.</p>
    </section>
  );
}
