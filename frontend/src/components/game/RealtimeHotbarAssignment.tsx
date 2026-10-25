"use client";

import { useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { guSpecFor, useGameStore, type Player } from "@/store/useGameStore";

export function RealtimeHotbarAssignment({ player, onSaved }: { player: Player; onSaved?: () => void }) {
  const store = useGameStore();
  const activeGuIds = player.gu_inventory
    .filter((gu) => gu.equipped && gu.status === "ACTIVE")
    .map((gu) => gu.instance_id);
  const activeSet = new Set(activeGuIds);
  const orderedIds = player.equipped_gus.filter((id) => activeSet.has(id));
  for (const id of activeGuIds) if (!orderedIds.includes(id)) orderedIds.push(id);
  const visibleSlots = Math.min(6, orderedIds.length);
  const orderKey = orderedIds.join("\u0000");
  const [slots, setSlots] = useState<string[]>(orderedIds.slice(0, visibleSlots));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => setSlots(orderedIds.slice(0, visibleSlots)), [orderKey, visibleSlots]);

  const guLabel = (instanceId: string) => {
    const gu = player.gu_inventory.find((entry) => entry.instance_id === instanceId);
    if (!gu) return instanceId;
    const spec = guSpecFor(gu.gu_id, gu.rank);
    return `${spec?.name ?? gu.gu_id} · Bậc ${gu.rank}`;
  };

  const save = async () => {
    setSaving(true);
    setError("");
    try {
      const response = await authFetch(`${BACKEND_URL}/api/inventory/realtime-hotbar`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ gu_instance_ids: slots }),
      });
      const body = await response.json().catch(() => ({})) as { error?: string; message?: string; equipped_gus?: string[] };
      if (!response.ok) throw new Error(body.error || "Không thể lưu hotbar realtime");
      if (Array.isArray(body.equipped_gus)) store.setPlayerFromServer({ equipped_gus: body.equipped_gus });
      store.addToast(body.message || "Đã lưu hotbar realtime", "success");
      onSaved?.();
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : String(caught);
      setError(message);
      store.addToast(message, "error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <section className="rounded-xl border border-emerald-300/15 bg-black/25 p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-emerald-100">Gán Cổ vào hotbar realtime</h3>
          <p className="mt-1 text-[11px] text-zinc-400">Gán thứ tự Cổ cho phím 1–6; chuột trái/phải sẽ dùng theo chiến thuật realtime.</p>
        </div>
        <button type="button" onClick={() => void save()} disabled={saving || visibleSlots === 0 || slots.length !== visibleSlots || new Set(slots).size !== slots.length} className="rounded-lg border border-emerald-300/25 bg-emerald-500/10 px-3 py-1.5 text-xs text-emerald-100 transition hover:bg-emerald-500/20 disabled:opacity-40">
          {saving ? "Đang lưu…" : "Lưu & kết nối lại"}
        </button>
      </div>
      {visibleSlots === 0 ? (
        <p className="mt-3 text-xs text-zinc-500">Trang bị Cổ trùng trạng thái Hoạt động trước khi gán nút.</p>
      ) : (
        <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: visibleSlots }, (_, index) => (
            <label key={index + 1} className="flex items-center gap-2 rounded-lg border border-white/5 bg-black/25 px-2 py-1.5">
              <span className="w-5 shrink-0 font-mono text-xs text-amber-200">{index + 1}</span>
              <select value={slots[index] ?? ""} onChange={(event) => setSlots((current) => current.map((id, slot) => slot === index ? event.target.value : id))} className="min-w-0 flex-1 bg-transparent text-xs text-zinc-200 outline-none">
                {orderedIds.map((instanceId) => (
                  <option key={instanceId} value={instanceId} className="bg-zinc-900">{guLabel(instanceId)}</option>
                ))}
              </select>
            </label>
          ))}
        </div>
      )}
      {error && <p role="alert" className="mt-2 text-xs text-red-300">{error}</p>}
    </section>
  );
}
