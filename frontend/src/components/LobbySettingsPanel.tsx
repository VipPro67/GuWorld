"use client";

import React from "react";
import {
  useUiPreferences,
  updateUiPreferences,
  MENU_THEME_PRESETS,
  type MenuThemePreset,
} from "@/lib/ui-preferences";

export const LobbySettingsPanel: React.FC = () => {
  const prefs = useUiPreferences();

  const handleThemeChange = (preset: MenuThemePreset) => {
    updateUiPreferences({
      menuColor: {
        ...prefs.menuColor,
        preset,
      },
    });
  };

  const handleSoundToggle = () => {
    updateUiPreferences({
      soundEnabled: !prefs.soundEnabled,
    });
  };

  const handlePlayerNamesToggle = () => {
    updateUiPreferences({
      showOtherPlayerNames: !prefs.showOtherPlayerNames,
    });
  };

  return (
    <div className="space-y-6 text-sm text-zinc-300">
      <div>
        <h3 className="font-serif font-semibold text-amber-200 mb-2">Chủ đề giao diện (Menu Theme)</h3>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
          {MENU_THEME_PRESETS.map((item) => {
            const isSelected = prefs.menuColor.preset === item.id;
            return (
              <button
                key={item.id}
                type="button"
                onClick={() => handleThemeChange(item.id)}
                className={`flex flex-col items-start p-2.5 rounded-xl border text-left transition-all ${
                  isSelected
                    ? "border-amber-400 bg-amber-950/40 text-amber-100 shadow-md"
                    : "border-zinc-800 bg-zinc-900/60 hover:bg-zinc-800/80 text-zinc-400"
                }`}
              >
                <div className="flex items-center gap-2 mb-1">
                  <span
                    className="w-3.5 h-3.5 rounded-full border border-white/20"
                    style={{ backgroundColor: item.color }}
                  />
                  <span className="font-semibold text-xs">{item.name}</span>
                </div>
                <span className="text-[10px] text-zinc-500">{item.desc}</span>
              </button>
            );
          })}
        </div>
      </div>

      <div className="border-t border-zinc-800 pt-4 space-y-3">
        <h3 className="font-serif font-semibold text-amber-200">Âm thanh & Hiển thị</h3>
        <label className="flex items-center justify-between cursor-pointer p-2 rounded-lg bg-zinc-900/40 hover:bg-zinc-900 border border-zinc-800">
          <span>Âm thanh trò chơi</span>
          <input
            type="checkbox"
            checked={prefs.soundEnabled}
            onChange={handleSoundToggle}
            className="rounded border-zinc-700 text-amber-500 focus:ring-amber-500 h-4 w-4"
          />
        </label>

        <label className="flex items-center justify-between cursor-pointer p-2 rounded-lg bg-zinc-900/40 hover:bg-zinc-900 border border-zinc-800">
          <span>Hiện danh xưng người chơi khác</span>
          <input
            type="checkbox"
            checked={prefs.showOtherPlayerNames}
            onChange={handlePlayerNamesToggle}
            className="rounded border-zinc-700 text-amber-500 focus:ring-amber-500 h-4 w-4"
          />
        </label>
      </div>
    </div>
  );
};
