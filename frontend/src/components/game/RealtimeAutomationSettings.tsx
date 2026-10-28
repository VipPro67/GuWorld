"use client";

import { useEffect, useState } from "react";
import { useGameStore, type AutoCombatConfig, type Player } from "@/store/useGameStore";
import { RealtimeHotbarAssignment } from "./RealtimeHotbarAssignment";
import { QuickActionCustomizer } from "./QuickActionCustomizer";

interface RealtimeAutomationDraft {
  leftClickSlot: number;
  rightClickAction: "attack" | "skill";
  rightClickSlot: number;
  autoFarm: boolean;
  farmRange: number;
  farmStopHPPercent: number;
  farmUseSkills: boolean;
}

const DEFAULT_REALTIME_DRAFT: RealtimeAutomationDraft = {
  leftClickSlot: 0,
  rightClickAction: "attack",
  rightClickSlot: 1,
  autoFarm: false,
  farmRange: 14,
  farmStopHPPercent: 25,
  farmUseSkills: false,
};

function realtimeDraftFrom(settings: AutoCombatConfig | null | undefined): RealtimeAutomationDraft {
  return {
    leftClickSlot: settings?.realtime_left_click_slot ?? DEFAULT_REALTIME_DRAFT.leftClickSlot,
    rightClickAction: settings?.realtime_right_click_action === "skill" ? "skill" : "attack",
    rightClickSlot: settings?.realtime_right_click_slot ?? DEFAULT_REALTIME_DRAFT.rightClickSlot,
    autoFarm: settings?.realtime_auto_farm ?? DEFAULT_REALTIME_DRAFT.autoFarm,
    farmRange: settings?.realtime_farm_range ?? DEFAULT_REALTIME_DRAFT.farmRange,
    farmStopHPPercent: Math.max(5, settings?.realtime_farm_stop_hp_percent ?? DEFAULT_REALTIME_DRAFT.farmStopHPPercent),
    farmUseSkills: settings?.realtime_farm_use_skills ?? DEFAULT_REALTIME_DRAFT.farmUseSkills,
  };
}

export function RealtimeAutomationSettings({
  player,
  onHotbarSaved,
}: {
  player: Player | null;
  onHotbarSaved?: () => void;
}) {
  const store = useGameStore();
  const [draft, setDraft] = useState<RealtimeAutomationDraft>(() => realtimeDraftFrom(store.combatSettings ?? player?.combat_settings));
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!store.combatSettings && !store.combatSettingsLoading) void store.fetchCombatSettings();
  }, [store.combatSettings, store.combatSettingsLoading]);

  useEffect(() => {
    setDraft(realtimeDraftFrom(store.combatSettings ?? player?.combat_settings));
  }, [store.combatSettings, player?.combat_settings]);

  const save = async () => {
    setSaving(true);
    const source = store.combatSettings ?? player?.combat_settings;
    const settings: AutoCombatConfig = {
      realtime_left_click_slot: draft.leftClickSlot,
      realtime_right_click_action: draft.rightClickAction,
      realtime_right_click_slot: draft.rightClickAction === "skill" ? draft.rightClickSlot : 0,
      realtime_auto_farm: draft.autoFarm,
      realtime_farm_range: draft.farmRange,
      realtime_farm_stop_hp_percent: draft.farmStopHPPercent,
      realtime_farm_use_skills: draft.farmUseSkills,
    };
    const ok = await store.saveCombatSettings(settings);
    if (ok) store.addToast("Đã lưu điều khiển và auto farm realtime", "success");
    setSaving(false);
  };

  const selectClass = "mt-1 w-full rounded-xl border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-200 focus:border-zinc-700 focus:outline-none";
  const toggleClass = "h-4 w-4 accent-emerald-500";

  return (
    <div className="space-y-4 text-sm text-zinc-300">
      {!player && (
        <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 text-xs text-zinc-400">
          Đăng nhập nhân vật để chỉnh cài đặt chiến đấu realtime.
        </section>
      )}

      {player && <RealtimeHotbarAssignment player={player} onSaved={onHotbarSaved} />}

      <QuickActionCustomizer inline />

      <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
        <h3 className="font-serif text-sm font-bold text-zinc-100">Điều khiển chuột & phím</h3>
        <p className="mt-1 text-[11px] text-zinc-500">Phím 1–6 dùng Cổ theo thứ tự hotbar; phím 7 biến thân và phím 8 dùng chiêu trong dạng khi Cổ hỗ trợ.</p>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <label className="text-xs text-zinc-400">Chuột trái
            <select className={selectClass} value={draft.leftClickSlot} onChange={(event) => setDraft((value) => ({ ...value, leftClickSlot: Number(event.target.value) }))}>
              <option value={0}>Dùng Cổ đang chọn bằng phím 1–6; nếu chưa chọn thì di chuyển</option>
              {Array.from({ length: 6 }, (_, index) => <option key={index + 1} value={index + 1}>Dùng Cổ ở phím {index + 1}</option>)}
            </select>
          </label>
          <label className="text-xs text-zinc-400">Chuột phải
            <select className={selectClass} value={draft.rightClickAction} onChange={(event) => setDraft((value) => ({ ...value, rightClickAction: event.target.value as "attack" | "skill" }))}>
              <option value="attack">Đánh thường</option>
              <option value="skill">Dùng Cổ theo phím</option>
            </select>
          </label>
          {draft.rightClickAction === "skill" && (
            <label className="text-xs text-zinc-400">Cổ dùng bằng chuột phải
              <select className={selectClass} value={draft.rightClickSlot} onChange={(event) => setDraft((value) => ({ ...value, rightClickSlot: Number(event.target.value) }))}>
                {Array.from({ length: 6 }, (_, index) => <option key={index + 1} value={index + 1}>Phím {index + 1}</option>)}
              </select>
            </label>
          )}
        </div>
      </section>

      <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h3 className="font-serif text-sm font-bold text-zinc-100">Tự farm quái realtime</h3>
            <p className="mt-1 text-[11px] text-zinc-500">Nhân vật tự tìm quái gần, chạy vào tầm đánh và đánh trên server. Chỉ hoạt động khi tab game đang mở.</p>
          </div>
          <input aria-label="Bật tự farm quái realtime" className={toggleClass} type="checkbox" checked={draft.autoFarm} onChange={(event) => setDraft((value) => ({ ...value, autoFarm: event.target.checked }))} />
        </div>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          <label className="text-xs text-zinc-400">Tầm tìm quái: {draft.farmRange} ô
            <input className="mt-2 w-full accent-amber-400" type="range" min={4} max={32} value={draft.farmRange} onChange={(event) => setDraft((value) => ({ ...value, farmRange: Number(event.target.value) }))} />
          </label>
          <label className="text-xs text-zinc-400">Dừng khi HP ≤ {draft.farmStopHPPercent}%
            <input className="mt-2 w-full accent-rose-400" type="range" min={5} max={80} step={5} value={draft.farmStopHPPercent} onChange={(event) => setDraft((value) => ({ ...value, farmStopHPPercent: Number(event.target.value) }))} />
            <span className="mt-1 block text-[10px] text-zinc-600">Tạm dừng đánh khi HP xuống ngưỡng này.</span>
          </label>
        </div>
        <label className="mt-3 flex items-center gap-2 text-xs text-zinc-300">
          <input className={toggleClass} type="checkbox" checked={draft.farmUseSkills} onChange={(event) => setDraft((value) => ({ ...value, farmUseSkills: event.target.checked }))} />
          Tự dùng Cổ theo thứ tự hotbar khi đủ mana/hết hồi chiêu
        </label>
      </section>

      <button type="button" disabled={saving || store.actionLoading} onClick={() => void save()} className="w-full rounded-xl bg-gradient-to-r from-emerald-700 to-teal-600 py-2.5 text-xs font-mono font-bold text-white disabled:opacity-50">
        {saving || store.actionLoading ? "Đang lưu…" : "Lưu chiến thuật realtime"}
      </button>

      <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
        <h3 className="font-serif text-sm font-bold text-zinc-100">Điều khiển & phiên chơi</h3>
        <ul className="mt-2 space-y-1 font-mono text-xs text-zinc-400">
          <li>WASD / mũi tên · di chuyển; click đất · đi theo đường A*</li>
          <li>1–6 · dùng Cổ đã gán; 7 · biến thân; 8 · chiêu trong dạng; Tab · đổi mục tiêu</li>
          <li>Esc · bỏ chọn Cổ/aim/target; cuộn chuột · zoom camera</li>
          <li>F3 · bật/tắt debug HUD (tick/FPS/network/prediction)</li>
        </ul>
        <p className="mt-3 text-xs text-zinc-400">Auto farm gửi ý định di chuyển/đánh; server vẫn kiểm tra vị trí, tầm đánh, mana và hồi chiêu.</p>
      </section>
    </div>
  );
}
