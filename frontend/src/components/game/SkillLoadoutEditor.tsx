"use client";

import { useCallback, useEffect, useMemo, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { ChevronDown, ChevronUp, Keyboard, RotateCcw, Swords, Trash2 } from "lucide-react";
import { guSpecFor, useGameStore, type Player } from "@/store/useGameStore";
import {
  bindingFromEvent,
  bindingsEqual,
  defaultHotbarKeybinds,
  formatHotbarBinding,
  hotbarKeyLabel,
  isReservedHotbarBinding,
  isUnsafeHotbarBinding,
  setHotbarKeybinds,
  useHotbarKeybinds,
} from "@/lib/ui-preferences";
import type { CombatSkillCandidate, CombatSkillKind } from "@/store/slices/killerMovesSlice";

type LoadoutEntry = { kind: CombatSkillKind; id: string };

function entryKey(entry: LoadoutEntry) {
  return `${entry.kind}:${entry.id}`;
}

/**
 * One loadout UI for combat skills. Slots 1-6 are Gu (server-backed elsewhere);
 * this editor owns the shared 9-12 pool for sát chiêu and Tiên Cổ Ốc Trận Kích.
 * Only bound skills are rendered — no empty placeholder slots.
 */
export function SkillLoadoutEditor() {
  const token = useGameStore((state) => state.token);
  const loadout = useGameStore((state) => state.skillLoadout);
  const loading = useGameStore((state) => state.skillLoadoutLoading);
  const fetchLoadout = useGameStore((state) => state.fetchSkillLoadout);
  const saveLoadout = useGameStore((state) => state.saveSkillLoadout);
  const toast = useGameStore((state) => state.addToast);
  const keybinds = useHotbarKeybinds();

  const [entries, setEntries] = useState<LoadoutEntry[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (token) void fetchLoadout();
  }, [token, fetchLoadout]);

  useEffect(() => {
    if (!loadout) return;
    setEntries(loadout.slots.map((slot) => ({ kind: slot.kind, id: slot.id })));
  }, [loadout]);

  const meta = useMemo(() => {
    const map = new Map<string, CombatSkillCandidate>();
    for (const candidate of loadout?.available ?? []) map.set(entryKey(candidate), candidate);
    return map;
  }, [loadout]);

  const firstSlot = loadout?.firstSlot ?? 9;
  const maxSlots = loadout?.maxSlots ?? 4;
  const addable = useMemo(
    () => (loadout?.available ?? []).filter((candidate) => candidate.slot === 0),
    [loadout],
  );

  const commit = useCallback(async (next: LoadoutEntry[]) => {
    setSaving(true);
    const ok = await saveLoadout(next);
    if (ok) setEntries(next);
    setSaving(false);
    return ok;
  }, [saveLoadout]);

  const move = (index: number, direction: -1 | 1) => {
    const target = index + direction;
    if (target < 0 || target >= entries.length) return;
    const next = [...entries];
    [next[index], next[target]] = [next[target], next[index]];
    void commit(next);
  };

  const remove = (index: number) => {
    void commit(entries.filter((_, current) => current !== index));
  };

  const add = (candidate: CombatSkillCandidate) => {
    if (entries.length >= maxSlots) {
      toast(`Chỉ có ${maxSlots} ô chiêu thức`, "error");
      return;
    }
    void commit([...entries, { kind: candidate.kind, id: candidate.id }]);
  };

  if (!token) {
    return (
      <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 text-xs text-zinc-400">
        Đăng nhập nhân vật để chỉnh hotbar sát chiêu.
      </section>
    );
  }

  return (
    <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4 space-y-3">
      <div>
        <div className="flex items-center gap-2">
          <Swords className="h-4 w-4 text-amber-400" />
          <h3 className="font-serif text-sm font-bold text-zinc-100">Sát chiêu &amp; Trận Kích Tiên Cổ Ốc</h3>
        </div>
        <p className="mt-1 text-[11px] text-zinc-500">
          Sát chiêu tự sáng tạo và Trận Kích của Tiên Cổ Ốc dùng chung {maxSlots} phím Alt. Thứ tự dưới đây là thứ tự phím.
        </p>
      </div>

      {loading && !loadout && <p className="py-3 text-center text-xs text-zinc-500">Đang tải hotbar chiêu thức…</p>}

      {loadout && entries.length === 0 && (
        <p className="rounded-xl border border-dashed border-zinc-800 px-3 py-3 text-center text-[11px] text-zinc-500">
          Chưa ghim chiêu nào vào phím Alt. Thêm từ danh sách bên dưới.
        </p>
      )}

      {entries.length > 0 && (
        <div className="space-y-1">
          {entries.map((entry, index) => {
            const candidate = meta.get(entryKey(entry));
            const slot = firstSlot + index;
            const name = candidate?.name ?? entry.id;
            return (
              <div key={entryKey(entry)} className="flex items-center gap-2 rounded-xl bg-black/30 px-2.5 py-1.5">
                <kbd className="rounded border border-amber-300/25 bg-zinc-950 px-1.5 py-0.5 font-mono text-[10px] text-amber-200">
                  {formatHotbarBinding(keybinds[slot]) || `Alt+${index + 1}`}
                </kbd>
                <span className="rounded bg-white/5 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-zinc-400">
                  {entry.kind === "house" ? "Trận Kích" : "Sát chiêu"}
                </span>
                <span className="min-w-0 flex-1 truncate text-xs text-zinc-200" title={name}>{name}</span>
                <span className="text-[10px] text-zinc-500">R{candidate?.rank ?? "?"}</span>
                <button type="button" title="Lên" aria-label={`Đưa ${name} lên`} disabled={saving || index === 0} onClick={() => move(index, -1)} className="rounded p-1 text-zinc-500 hover:text-white disabled:opacity-20">
                  <ChevronUp className="h-3.5 w-3.5" />
                </button>
                <button type="button" title="Xuống" aria-label={`Đưa ${name} xuống`} disabled={saving || index === entries.length - 1} onClick={() => move(index, 1)} className="rounded p-1 text-zinc-500 hover:text-white disabled:opacity-20">
                  <ChevronDown className="h-3.5 w-3.5" />
                </button>
                <button type="button" title="Bỏ ghim" aria-label={`Bỏ ghim ${name}`} disabled={saving} onClick={() => remove(index)} className="rounded p-1 text-zinc-500 hover:text-rose-300 disabled:opacity-20">
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            );
          })}
        </div>
      )}

      {addable.length > 0 && entries.length < maxSlots && (
        <div className="space-y-1 border-t border-white/5 pt-2">
          <h4 className="text-[9px] font-semibold uppercase tracking-widest text-zinc-500">Thêm chiêu vào hotbar</h4>
          {addable.map((candidate) => {
            const key = entryKey(candidate);
            return (
              <div key={key} className="flex items-center gap-2 rounded-xl bg-white/[0.03] px-2.5 py-1.5">
                <span className="rounded bg-white/5 px-1.5 py-0.5 text-[9px] uppercase tracking-wide text-zinc-400">
                  {candidate.kind === "house" ? "Trận Kích" : "Sát chiêu"}
                </span>
                <span className="min-w-0 flex-1 truncate text-xs text-zinc-300" title={candidate.name}>{candidate.name}</span>
                <span className="text-[10px] text-zinc-500">R{candidate.rank}</span>
                <button
                  type="button"
                  disabled={!candidate.supported || saving}
                  title={candidate.supported ? "Ghim vào phím Alt" : candidate.reason || "Chiêu chưa hỗ trợ realtime"}
                  onClick={() => add(candidate)}
                  className="rounded-lg border border-emerald-900/60 bg-emerald-950/40 px-2 py-1 text-[10px] text-emerald-200 hover:bg-emerald-900/50 disabled:cursor-not-allowed disabled:opacity-40"
                >
                  Ghim
                </button>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

/** Editable hotbar keybinds. Rows only exist for slots that currently hold a skill. */
export function HotbarKeybindEditor({ player }: { player: Player | null }) {
  const keybinds = useHotbarKeybinds();
  const loadout = useGameStore((state) => state.skillLoadout);
  const toast = useGameStore((state) => state.addToast);
  const [captureSlot, setCaptureSlot] = useState<number | null>(null);
  const [isTouchDevice, setIsTouchDevice] = useState(false);
  const [showTouchKeybinds, setShowTouchKeybinds] = useState(false);

  useEffect(() => {
    setIsTouchDevice(
      typeof window !== "undefined" &&
        (window.matchMedia("(max-width: 767px)").matches || "ontouchstart" in window || navigator.maxTouchPoints > 0),
    );
  }, []);

  const rows = useMemo(() => {
    const result: Array<{ slot: number; name: string }> = [];
    const inventory = player?.gu_inventory ?? [];
    const active = inventory.filter((item) => item.equipped && item.status === "ACTIVE");
    const activeIds = new Set(active.map((item) => item.instance_id));
    const orderedIds = (player?.equipped_gus ?? []).filter((id) => activeIds.has(id));
    for (const id of activeIds) if (!orderedIds.includes(id)) orderedIds.push(id);
    for (let index = 0; index < Math.min(6, orderedIds.length); index += 1) {
      const gu = inventory.find((item) => item.instance_id === orderedIds[index]);
      if (!gu) continue;
      result.push({ slot: index + 1, name: guSpecFor(gu.gu_id, gu.rank)?.name ?? gu.gu_id });
    }
    for (const entry of loadout?.slots ?? []) {
      result.push({ slot: entry.slot, name: entry.name });
    }
    return result.sort((a, b) => a.slot - b.slot);
  }, [player, loadout]);

  const capture = (event: ReactKeyboardEvent<HTMLButtonElement>, slot: number) => {
    event.preventDefault();
    event.stopPropagation();
    if (event.key === "Escape") {
      setCaptureSlot(null);
      return;
    }
    const binding = bindingFromEvent(event);
    if (!binding) return;
    if (isReservedHotbarBinding(binding)) {
      toast("Phím này dành cho di chuyển/điều khiển cố định", "error");
      return;
    }
    if (isUnsafeHotbarBinding(binding)) {
      toast("Phím này bị trình duyệt chiếm; chọn phím khác", "error");
      return;
    }
    const conflict = Object.entries(keybinds).find(([rawSlot, value]) => Number(rawSlot) !== slot && bindingsEqual(value, binding));
    if (conflict) {
      toast(`Phím này đã dùng cho ô ${conflict[0]}`, "error");
      return;
    }
    setHotbarKeybinds({ ...keybinds, [slot]: binding });
    setCaptureSlot(null);
  };

  if (isTouchDevice && !showTouchKeybinds) {
    return (
      <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
        <button
          type="button"
          onClick={() => setShowTouchKeybinds(true)}
          className="flex w-full items-center justify-between text-left text-xs text-zinc-400 hover:text-amber-200 transition"
        >
          <div className="flex items-center gap-2">
            <Keyboard className="h-4 w-4 text-zinc-500" />
            <span>Phím tắt bàn phím PC (Đang ẩn trên thiết bị di động/cảm ứng)</span>
          </div>
          <span className="text-[11px] font-medium text-amber-500/80 hover:text-amber-400">Mở</span>
        </button>
      </section>
    );
  }

  return (
    <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4 space-y-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <Keyboard className="h-4 w-4 text-amber-400" />
            <h3 className="font-serif text-sm font-bold text-zinc-100">Phím tắt bàn phím PC</h3>
            {isTouchDevice && (
              <button
                type="button"
                onClick={() => setShowTouchKeybinds(false)}
                className="text-[10px] text-zinc-500 hover:text-zinc-300 ml-1 underline"
              >
                (Thu gọn)
              </button>
            )}
          </div>
          <p className="mt-1 text-[11px] text-zinc-500">
            Gán nút bấm bàn phím máy tính cho từng ô kỹ năng. Bấm vào nút phím rồi nhấn phím mới · Esc để hủy.
          </p>
        </div>
        <button
          type="button"
          onClick={() => setHotbarKeybinds(defaultHotbarKeybinds())}
          title="Khôi phục phím mặc định"
          className="flex shrink-0 items-center gap-1 rounded-lg border border-[#8f6d40]/60 bg-[#2b1f15] px-2 py-1 text-[10px] text-amber-200 hover:bg-[#432d1c]"
        >
          <RotateCcw className="h-3 w-3" />
          Mặc định
        </button>
      </div>

      {rows.length === 0 ? (
        <p className="rounded-xl border border-dashed border-zinc-800 px-3 py-3 text-center text-[11px] text-zinc-500">
          Chưa có chiêu nào trong hotbar để gán phím.
        </p>
      ) : (
        <div className="grid gap-1.5 sm:grid-cols-2">
          {rows.map((row) => (
            <div key={row.slot} className="flex items-center gap-2 rounded-xl bg-black/30 px-2.5 py-1.5">
              <span className="shrink-0 rounded bg-black/40 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-amber-300">
                Ô {row.slot}
              </span>
              <span className="min-w-0 flex-1 truncate text-xs text-zinc-300" title={row.name}>{row.name}</span>
              <button
                type="button"
                onClick={() => setCaptureSlot(row.slot)}
                onKeyDown={(event) => capture(event, row.slot)}
                title={captureSlot === row.slot ? "Nhấn phím mới · Esc để hủy" : "Đổi phím"}
                className={`rounded-lg border px-2 py-1 font-mono text-[10px] transition ${
                  captureSlot === row.slot
                    ? "border-amber-400 bg-amber-950/50 text-amber-100"
                    : "border-zinc-700 bg-zinc-950 text-zinc-300 hover:border-amber-500/60"
                }`}
              >
                {captureSlot === row.slot ? "…" : `Phím ${hotbarKeyLabel(row.slot, keybinds) || "—"}`}
              </button>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}
