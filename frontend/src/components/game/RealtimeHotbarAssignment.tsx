"use client";

import { useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { guSpecFor, useGameStore, type Player } from "@/store/useGameStore";

const SLOT_COUNT = 6;

/**
 * Realtime hotbar assignment is an explicit persisted setting
 * (players.realtime_hotbar): slot index = ô, "" = ô trống. Pins survive
 * equipped/unequipped churn: a pinned Gu reclaims its slot when it becomes
 * valid again, and an empty slot stays empty until the player fills it.
 */
function deriveHotbarLayout(player: Player): string[] {
  const activeSet = new Set(
    player.gu_inventory
      .filter((gu) => gu.equipped && gu.status === "ACTIVE")
      .map((gu) => gu.instance_id),
  );
  const orderedIds = player.equipped_gus.filter((id) => activeSet.has(id));
  for (const id of activeGuIds(player)) if (!orderedIds.includes(id)) orderedIds.push(id);

  const layout = new Array<string>(SLOT_COUNT).fill("");
  const saved = player.hotbar_gus ?? [];
  saved.slice(0, SLOT_COUNT).forEach((id, index) => {
    layout[index] = id;
  });
  const used = new Set(layout.filter(Boolean));
  let next = 0;
  for (let i = 0; i < SLOT_COUNT; i++) {
    if (layout[i]) continue;
    while (next < orderedIds.length && used.has(orderedIds[next])) next++;
    if (next >= orderedIds.length) break;
    layout[i] = orderedIds[next];
    used.add(orderedIds[next]);
    next++;
  }
  return layout;
}

function activeGuIds(player: Player): string[] {
  return player.gu_inventory
    .filter((gu) => gu.equipped && gu.status === "ACTIVE")
    .map((gu) => gu.instance_id);
}

export function RealtimeHotbarAssignment({ player, onSaved }: { player: Player; onSaved?: () => void }) {
  const store = useGameStore();

  // Re-derive when the saved layout, equipped order or Gu validity changes:
  // a pin whose Gu flips valid/invalid must refresh its reserved display.
  const validityKey = player.gu_inventory
    .map((gu) => `${gu.instance_id}:${gu.equipped}:${gu.status}`)
    .join("|");
  const deriveKey = [
    ...(player.hotbar_gus ?? []),
    "\u0000",
    ...player.equipped_gus,
    "\u0000",
    validityKey,
  ].join("|");
  const [slots, setSlots] = useState<string[]>(() => deriveHotbarLayout(player));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [dirty, setDirty] = useState(false);

  // Server state is the source of truth: until the user starts editing, keep
  // the displayed slots in sync with the latest snapshot.
  useEffect(() => {
    if (!dirty) {
      setSlots(deriveHotbarLayout(player));
    }
  }, [deriveKey, dirty]);

  const assignedIds = [...activeGuIds(player)];

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
      const body = await response.json().catch(() => ({})) as {
        error?: string; message?: string; equipped_gus?: string[]; hotbar_gus?: string[];
      };
      if (!response.ok) throw new Error(body.error || "Không thể lưu hotbar realtime");
      store.setPlayerFromServer({
 ...(Array.isArray(body.equipped_gus) ? { equipped_gus: body.equipped_gus } : {}),
        ...(Array.isArray(body.hotbar_gus) ? { hotbar_gus: body.hotbar_gus } : {}),
      });
      store.addToast(body.message || "Đã lưu thứ tự Cổ trên Hotbar", "success");
      setDirty(false);
      onSaved?.();
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : String(caught);
      setError(message);
      store.addToast(message, "error");
    } finally {
      setSaving(false);
    }
  };

  const onSelectSlot = (index: number, targetId: string) => {
    setDirty(true);
    const next = [...slots];
    if (targetId) {
      const existingIdx = next.indexOf(targetId);
      if (existingIdx !== -1 && existingIdx !== index) {
        next[existingIdx] = "";
      }
    }
    next[index] = targetId;
    setSlots(next);
  };

  const hasDuplicatesOrInvalid =
    slots.filter(Boolean).length !== new Set(slots.filter(Boolean)).size;

  return (
    <section className="rounded-xl border border-emerald-300/15 bg-black/25 p-3">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-sm font-semibold text-emerald-100">Thứ tự Cổ trên Hotbar (Ô 1 – 6)</h3>
          <p className="mt-1 text-[11px] text-zinc-400">
            Chọn Cổ trùng xuất hiện tại từng ô trên thanh kỹ năng (Hotbar), hoặc để ô trống. Thiết lập được lưu riêng: Cổ gán vẫn giữ nguyên ô dù trang bị/tháo thêm Cổ khác, và Cổ tạm tháo sẽ tự chiếm lại ô cũ khi hợp lệ trở lại.
          </p>
        </div>
        <button
          type="button"
          onClick={() => void save()}
          disabled={saving || hasDuplicatesOrInvalid}
          className="rounded-lg border border-emerald-300/25 bg-emerald-500/10 px-3 py-1.5 text-xs font-medium text-emerald-100 transition hover:bg-emerald-500/20 disabled:opacity-40"
        >
          {saving ? "Đang lưu…" : "Lưu thứ tự"}
        </button>
      </div>
      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: SLOT_COUNT }, (_, index) => {
          const current = slots[index] ?? "";
          const options = current && !assignedIds.includes(current) ? [current, ...assignedIds] : assignedIds;
          return (
            <label key={index + 1} className="flex items-center gap-2 rounded-lg border border-white/5 bg-black/25 px-2 py-1.5">
              <span className="shrink-0 rounded bg-amber-500/15 px-1.5 py-0.5 font-sans text-xs font-semibold text-amber-300">Ô {index + 1}</span>
              <select
                value={current}
                onChange={(event) => onSelectSlot(index, event.target.value)}
                className="min-w-0 flex-1 bg-transparent text-xs text-zinc-200 outline-none"
              >
                <option value="" className="bg-zinc-900">— Trống —</option>
                {options.map((instanceId) => (
                  <option key={instanceId} value={instanceId} className="bg-zinc-900">{guLabel(instanceId)}</option>
                ))}
              </select>
            </label>
          );
        })}
      </div>
      {error && <p role="alert" className="mt-2 text-xs text-red-300">{error}</p>}
    </section>
  );
}
