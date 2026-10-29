"use client";

import { useState, useEffect } from "react";
import {
  Settings2,
  Wrench,
  Swords,
  Sliders,
  RotateCcw,
  Sparkles,
  Volume2,
  Monitor,
  Info,
} from "lucide-react";
import { useGameStore, type AutoCombatConfig, type Player } from "@/store/useGameStore";
import { QuickActionCustomizer } from "./QuickActionCustomizer";
import { RealtimeHotbarAssignment } from "./RealtimeHotbarAssignment";
import { SkillLoadoutEditor, HotbarKeybindEditor } from "./SkillLoadoutEditor";
import { ClearCacheButton } from "@/components/ClearCacheModal";
import { resetUiPreferences, updateUiPreferences, useUiPreferences } from "@/lib/ui-preferences";

export type MMOSettingsTab = "quick_actions" | "combat" | "general";

interface MMOSettingsPanelProps {
  initialTab?: MMOSettingsTab;
  onLoadoutSaved?: () => void;
}

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

export function MMOSettingsPanel({ initialTab = "quick_actions", onLoadoutSaved }: MMOSettingsPanelProps) {
  const store = useGameStore();
  const player = store.player;
  const [activeTab, setActiveTab] = useState<MMOSettingsTab>(initialTab);

  // Combat settings state
  const [combatDraft, setCombatDraft] = useState<RealtimeAutomationDraft>(() =>
    realtimeDraftFrom(store.combatSettings ?? player?.combat_settings),
  );
  const [savingCombat, setSavingCombat] = useState(false);

  // General settings state
  const uiPreferences = useUiPreferences();
  const soundEnabled = uiPreferences.soundEnabled;
  const showOtherPlayerNames = uiPreferences.showOtherPlayerNames;

  useEffect(() => {
    if (!store.combatSettings && !store.combatSettingsLoading) {
      void store.fetchCombatSettings();
    }
  }, [store.combatSettings, store.combatSettingsLoading]);

  useEffect(() => {
    setCombatDraft(realtimeDraftFrom(store.combatSettings ?? player?.combat_settings));
  }, [store.combatSettings, player?.combat_settings]);

  const saveCombat = async () => {
    setSavingCombat(true);
    const source = store.combatSettings ?? player?.combat_settings;
    const settings: AutoCombatConfig = {
      realtime_left_click_slot: combatDraft.leftClickSlot,
      realtime_right_click_action: combatDraft.rightClickAction,
      realtime_right_click_slot: combatDraft.rightClickAction === "skill" ? combatDraft.rightClickSlot : 0,
      realtime_auto_farm: combatDraft.autoFarm,
      realtime_farm_range: combatDraft.farmRange,
      realtime_farm_stop_hp_percent: combatDraft.farmStopHPPercent,
      realtime_farm_use_skills: combatDraft.farmUseSkills,
    };
    const ok = await store.saveCombatSettings(settings);
    if (ok) store.addToast("Đã lưu thiết lập chiến đấu realtime", "success");
    setSavingCombat(false);
  };

  const selectClass =
    "mt-1 w-full rounded-xl border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-200 focus:border-amber-600 focus:outline-none";
  const toggleClass = "h-4 w-4 accent-amber-500";

  return (
    <div className="space-y-4 text-sm text-zinc-300 font-serif">
      {/* Top Tab Switcher */}
      <div className="flex border-b border-[#a88952]/30 pb-2 gap-1.5">
        <button
          type="button"
          onClick={() => setActiveTab("quick_actions")}
          className={`flex items-center gap-2 rounded-xl px-3 py-2 text-xs font-bold transition ${
            activeTab === "quick_actions"
              ? "bg-[#6b4728] text-amber-200 border border-[#b89456] shadow-sm"
              : "bg-[#251a12]/60 text-zinc-400 hover:text-[#f4e8cb] hover:bg-[#342419]"
          }`}
        >
          <Sparkles className="h-3.5 w-3.5 text-amber-400" />
          Tiện ích
        </button>

        <button
          type="button"
          onClick={() => setActiveTab("combat")}
          className={`flex items-center gap-2 rounded-xl px-3 py-2 text-xs font-bold transition ${
            activeTab === "combat"
              ? "bg-[#6b4728] text-amber-200 border border-[#b89456] shadow-sm"
              : "bg-[#251a12]/60 text-zinc-400 hover:text-[#f4e8cb] hover:bg-[#342419]"
          }`}
        >
          <Swords className="h-3.5 w-3.5 text-amber-400" />
          Chiến đấu & Điều khiển
        </button>

        <button
          type="button"
          onClick={() => setActiveTab("general")}
          className={`flex items-center gap-2 rounded-xl px-3 py-2 text-xs font-bold transition ${
            activeTab === "general"
              ? "bg-[#6b4728] text-amber-200 border border-[#b89456] shadow-sm"
              : "bg-[#251a12]/60 text-zinc-400 hover:text-[#f4e8cb] hover:bg-[#342419]"
          }`}
        >
          <Sliders className="h-3.5 w-3.5 text-amber-400" />
          Cài đặt Chung
        </button>
      </div>

      {/* Tab 1: Tiện ích (thanh ghim + menu yêu thích) */}
      {activeTab === "quick_actions" && (
        <div className="space-y-4 animate-in fade-in duration-200">
          <QuickActionCustomizer inline />
          <p className="text-[11px] text-zinc-500">
            Phím tắt hotbar chiến đấu và thứ tự sát chiêu / Trận Kích nằm trong tab “Chiến đấu &amp; Điều khiển”.
          </p>
        </div>
      )}

      {/* Tab 2: Chiến đấu & Điều khiển Realtime */}
      {activeTab === "combat" && (
        <div className="space-y-4 animate-in fade-in duration-200">
          {!player && (
            <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 text-xs text-zinc-400">
              Đăng nhập nhân vật để chỉnh cài đặt chiến đấu realtime.
            </section>
          )}

          {player && <RealtimeHotbarAssignment player={player} onSaved={onLoadoutSaved} />}

          <SkillLoadoutEditor />

          {player && <HotbarKeybindEditor player={player} />}

          <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
            <h3 className="font-serif text-sm font-bold text-zinc-100">Điều khiển chuột & phím</h3>
            <p className="mt-1 text-[11px] text-zinc-500">
              Các ô Cổ 1–6 dùng theo thứ tự hotbar; ô 7 biến thân và ô 8 dùng chiêu trong dạng khi Cổ hỗ trợ (phím mặc định 1–8). Thiết lập chuột được lưu cùng điều khiển realtime.
            </p>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              <label className="text-xs text-zinc-400">
                Chuột trái
                <select
                  className={selectClass}
                  value={combatDraft.leftClickSlot}
                  onChange={(event) =>
                    setCombatDraft((value) => ({ ...value, leftClickSlot: Number(event.target.value) }))
                  }
                >
                  <option value={0}>Dùng Cổ đang chọn bằng phím 1–6; nếu chưa chọn thì di chuyển</option>
                  {Array.from({ length: 6 }, (_, index) => (
                    <option key={index + 1} value={index + 1}>
                      Dùng Cổ ở phím {index + 1}
                    </option>
                  ))}
                </select>
              </label>
              <label className="text-xs text-zinc-400">
                Chuột phải
                <select
                  className={selectClass}
                  value={combatDraft.rightClickAction}
                  onChange={(event) =>
                    setCombatDraft((value) => ({
                      ...value,
                      rightClickAction: event.target.value as "attack" | "skill",
                    }))
                  }
                >
                  <option value="attack">Đánh thường</option>
                  <option value="skill">Dùng Cổ theo phím</option>
                </select>
              </label>
              {combatDraft.rightClickAction === "skill" && (
                <label className="text-xs text-zinc-400">
                  Cổ dùng bằng chuột phải
                  <select
                    className={selectClass}
                    value={combatDraft.rightClickSlot}
                    onChange={(event) =>
                      setCombatDraft((value) => ({ ...value, rightClickSlot: Number(event.target.value) }))
                    }
                  >
                    {Array.from({ length: 6 }, (_, index) => (
                      <option key={index + 1} value={index + 1}>
                        Phím {index + 1}
                      </option>
                    ))}
                  </select>
                </label>
              )}
            </div>
          </section>

          <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h3 className="font-serif text-sm font-bold text-zinc-100">Tự farm quái realtime</h3>
                <p className="mt-1 text-[11px] text-zinc-500">
                  Nhân vật tự tìm quái gần, chạy vào tầm đánh và đánh trên server. Chỉ hoạt động khi tab game đang mở.
                </p>
              </div>
              <input
                aria-label="Bật tự farm quái realtime"
                className={toggleClass}
                type="checkbox"
                checked={combatDraft.autoFarm}
                onChange={(event) =>
                  setCombatDraft((value) => ({ ...value, autoFarm: event.target.checked }))
                }
              />
            </div>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              <label className="text-xs text-zinc-400">
                Tầm tìm quái: {combatDraft.farmRange} ô
                <input
                  className="mt-2 w-full accent-amber-400"
                  type="range"
                  min={4}
                  max={32}
                  value={combatDraft.farmRange}
                  onChange={(event) =>
                    setCombatDraft((value) => ({ ...value, farmRange: Number(event.target.value) }))
                  }
                />
              </label>
              <label className="text-xs text-zinc-400">
                Dừng khi HP ≤ {combatDraft.farmStopHPPercent}%
                <input
                  className="mt-2 w-full accent-rose-400"
                  type="range"
                  min={5}
                  max={80}
                  step={5}
                  value={combatDraft.farmStopHPPercent}
                  onChange={(event) =>
                    setCombatDraft((value) => ({
                      ...value,
                      farmStopHPPercent: Number(event.target.value),
                    }))
                  }
                />
                <span className="mt-1 block text-[10px] text-zinc-600">
                  Tạm dừng đánh khi HP xuống ngưỡng này.
                </span>
              </label>
            </div>
            <label className="mt-3 flex items-center gap-2 text-xs text-zinc-300">
              <input
                className={toggleClass}
                type="checkbox"
                checked={combatDraft.farmUseSkills}
                onChange={(event) =>
                  setCombatDraft((value) => ({ ...value, farmUseSkills: event.target.checked }))
                }
              />
              Tự dùng Cổ theo thứ tự hotbar khi đủ mana/hết hồi chiêu
            </label>
          </section>

          <button
            type="button"
            disabled={savingCombat || store.actionLoading}
            onClick={() => void saveCombat()}
            className="w-full rounded-xl bg-gradient-to-r from-emerald-700 to-teal-600 py-2.5 text-xs font-mono font-bold text-white hover:brightness-110 active:scale-98 disabled:opacity-50"
          >
            {savingCombat || store.actionLoading ? "Đang lưu…" : "Lưu thiết lập chiến đấu"}
          </button>
        </div>
      )}

      {/* Tab 3: Cài đặt Chung & Hệ thống (Dành cho sau này cài đặt chung chung) */}
      {activeTab === "general" && (
        <div className="space-y-4 animate-in fade-in duration-200">
          {/* Hiển thị & Đồ họa */}
          <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
            <div className="flex items-center gap-2 mb-2">
              <Monitor className="w-4 h-4 text-amber-400" />
              <h3 className="font-serif text-sm font-bold text-zinc-100">Hiển thị & Giao diện</h3>
            </div>
            <div className="space-y-3">
              <label className="flex items-center justify-between text-xs text-zinc-300 cursor-pointer">
                <span>Hiện tên và thanh máu người chơi khác</span>
                <input
                  type="checkbox"
                  className={toggleClass}
                  checked={showOtherPlayerNames}
                  onChange={(e) => updateUiPreferences({ showOtherPlayerNames: e.target.checked })}
                />
              </label>
              <p className="text-[10px] text-zinc-500">
                Tắt bớt thông tin người chơi khi đông người để tăng tốc độ khung hình.
              </p>
            </div>
          </section>

          {/* Âm thanh */}
          <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
            <div className="flex items-center gap-2 mb-2">
              <Volume2 className="w-4 h-4 text-amber-400" />
              <h3 className="font-serif text-sm font-bold text-zinc-100">Âm thanh (Hệ thống)</h3>
            </div>
            <div className="space-y-3">
              <label className="flex items-center justify-between text-xs text-zinc-300 cursor-pointer">
                <span>Bật âm thanh & hiệu ứng chiêu</span>
                <input
                  type="checkbox"
                  className={toggleClass}
                  checked={soundEnabled}
                  onChange={(e) => updateUiPreferences({ soundEnabled: e.target.checked })}
                />
              </label>
              <p className="text-[10px] text-zinc-500">
                Tính năng nhạc nền và âm thanh tiếng xuất chiêu đang được tích hợp thêm.
              </p>
            </div>
          </section>

          {/* Bộ nhớ đệm & Khôi phục */}
          <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
            <div className="flex items-center gap-2 mb-2">
              <RotateCcw className="w-4 h-4 text-amber-400" />
              <h3 className="font-serif text-sm font-bold text-zinc-100">Bộ nhớ đệm & Khôi phục</h3>
            </div>
            <p className="text-xs text-zinc-400 mb-3">
              Nếu gặp hiện tượng hiển thị sai lệch sau khi game cập nhật phiên bản mới, bạn có thể xóa cache client hoặc đặt lại thanh phím tắt.
            </p>
            <div className="flex flex-wrap gap-2">
              <ClearCacheButton className="text-xs px-3 py-1.5" />
              <button
                type="button"
                onClick={() => {
                  resetUiPreferences();
                  store.addToast("Đã khôi phục tiện ích và phím tắt về mặc định", "success");
                }}
                className="rounded-lg border border-[#8f6d40]/60 bg-[#2b1f15] px-3 py-1.5 text-xs text-amber-200 hover:bg-[#432d1c] transition"
              >
                Đặt lại Thanh Tiện Ích
              </button>
            </div>
          </section>

          {/* Thông tin phiên bản */}
          <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
            <div className="flex items-center gap-2 mb-1">
              <Info className="w-4 h-4 text-amber-400" />
              <h3 className="font-serif text-sm font-bold text-zinc-100">Thông tin trò chơi</h3>
            </div>
            <p className="text-xs text-zinc-400">
              GuWorld MMO Engine · Phiên bản Realtime Thế giới Cổ Giới 0.1.0
            </p>
          </section>
        </div>
      )}
    </div>
  );
}
