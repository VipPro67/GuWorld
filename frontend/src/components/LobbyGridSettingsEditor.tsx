"use client";

import React from "react";
import { ArrowDown, ArrowUp, Plus, RotateCcw, X } from "lucide-react";
import {
  DEFAULT_LOBBY_GRID_ITEMS,
  DEFAULT_LOBBY_GRID_LIMIT,
  isLobbyGroupId,
  LOBBY_ENTRY_LABELS,
  LOBBY_FEATURE_IDS,
  LOBBY_GRID_LIMIT_OPTIONS,
  LOBBY_MENU_GROUPS,
  type LobbyGridLimit,
} from "@/lib/lobby-menu";
import { updateUiPreferences, useUiPreferences } from "@/lib/ui-preferences";

const ALL_CONFIGURABLE_IDS = [
  ...LOBBY_MENU_GROUPS.map((group) => group.id),
  ...LOBBY_FEATURE_IDS,
];

export const LobbyGridSettingsEditor: React.FC = () => {
  const { lobbyGridItems, lobbyGridLimit } = useUiPreferences();

  const effectiveLimit = lobbyGridLimit === 0 ? Infinity : lobbyGridLimit;
  const availableIds = ALL_CONFIGURABLE_IDS.filter((id) => !lobbyGridItems.includes(id));

  const handleLimitChange = (limit: LobbyGridLimit) => {
    updateUiPreferences({ lobbyGridLimit: limit });
  };

  const handleMove = (index: number, delta: -1 | 1) => {
    const target = index + delta;
    if (target < 0 || target >= lobbyGridItems.length) return;
    const next = [...lobbyGridItems];
    [next[index], next[target]] = [next[target], next[index]];
    updateUiPreferences({ lobbyGridItems: next });
  };

  const handleRemove = (id: string) => {
    updateUiPreferences({ lobbyGridItems: lobbyGridItems.filter((item) => item !== id) });
  };

  const handleAdd = (id: string) => {
    if (lobbyGridItems.includes(id)) return;
    updateUiPreferences({ lobbyGridItems: [...lobbyGridItems, id] });
  };

  const handleReset = () => {
    updateUiPreferences({
      lobbyGridItems: [...DEFAULT_LOBBY_GRID_ITEMS],
      lobbyGridLimit: DEFAULT_LOBBY_GRID_LIMIT,
    });
  };

  return (
    <div className="space-y-4 text-sm text-zinc-300">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="font-serif font-semibold text-amber-200">Lưới Lobby &amp; Thứ Tự Menu</h3>
          <p className="text-xs text-zinc-500">Chọn số lượng tối đa, sắp xếp thứ tự hoặc ghim thêm tính năng ra ngoài Lobby.</p>
        </div>
        <button
          type="button"
          onClick={handleReset}
          className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-red-900/50 bg-red-950/30 px-2.5 py-1.5 text-xs font-semibold text-red-300 transition hover:bg-red-950/60 cursor-pointer"
        >
          <RotateCcw className="h-3.5 w-3.5" />
          <span>Khôi phục mặc định</span>
        </button>
      </div>

      <div>
        <div className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
          1. Số lượng tối đa hiển thị ngoài Lobby
        </div>
        <div className="flex flex-wrap gap-2">
          {LOBBY_GRID_LIMIT_OPTIONS.map((option) => {
            const active = lobbyGridLimit === option;
            return (
              <button
                key={option}
                type="button"
                onClick={() => handleLimitChange(option)}
                className={`min-h-9 rounded-lg border px-3 py-1.5 text-xs font-semibold transition cursor-pointer ${
                  active
                    ? "border-amber-400 bg-amber-950/50 text-amber-200"
                    : "border-zinc-800 bg-zinc-900/60 text-zinc-400 hover:bg-zinc-800"
                }`}
              >
                {option === 0 ? "Tất cả" : `${option} ô`}
              </button>
            );
          })}
        </div>
      </div>

      <div>
        <div className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
          2. Các mục đang bật ({lobbyGridItems.length})
        </div>
        {lobbyGridItems.length === 0 ? (
          <p className="rounded-lg border border-dashed border-zinc-800 bg-zinc-900/40 p-3 text-xs text-zinc-500">
            Chưa có mục nào trên Lưới Lobby. Hãy thêm mục bên dưới hoặc bấm &quot;Khôi phục mặc định&quot;.
          </p>
        ) : (
          <div className="space-y-1.5">
            {lobbyGridItems.map((id, idx) => {
              const isGroup = isLobbyGroupId(id);
              const isCutoff = idx >= effectiveLimit;
              return (
                <div
                  key={id}
                  className={`flex items-center justify-between gap-2 rounded-lg border px-2.5 py-2 text-xs ${
                    isCutoff
                      ? "border-dashed border-zinc-800 bg-zinc-900/30 opacity-60"
                      : "border-zinc-800 bg-zinc-900/70"
                  }`}
                >
                  <div className="flex min-w-0 items-center gap-2">
                    <span className="w-5 shrink-0 font-mono text-[11px] text-zinc-500">{idx + 1}.</span>
                    <span className="truncate font-semibold text-zinc-200">{LOBBY_ENTRY_LABELS[id] || id}</span>
                    {isGroup && (
                      <span className="shrink-0 rounded bg-amber-500/15 px-1.5 py-0.5 text-[10px] font-semibold text-amber-300">
                        Nhóm
                      </span>
                    )}
                    {isCutoff && (
                      <span className="shrink-0 rounded bg-rose-500/15 px-1.5 py-0.5 text-[10px] font-semibold text-rose-300">
                        Ngoài top {lobbyGridLimit}
                      </span>
                    )}
                  </div>
                  <div className="flex shrink-0 items-center gap-1">
                    <button
                      type="button"
                      disabled={idx === 0}
                      onClick={() => handleMove(idx, -1)}
                      aria-label={`Đưa ${LOBBY_ENTRY_LABELS[id] || id} lên`}
                      className="inline-flex h-8 w-8 items-center justify-center rounded border border-zinc-800 bg-zinc-950 text-zinc-300 hover:bg-zinc-800 disabled:opacity-30 cursor-pointer disabled:cursor-not-allowed"
                    >
                      <ArrowUp className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      disabled={idx === lobbyGridItems.length - 1}
                      onClick={() => handleMove(idx, 1)}
                      aria-label={`Đưa ${LOBBY_ENTRY_LABELS[id] || id} xuống`}
                      className="inline-flex h-8 w-8 items-center justify-center rounded border border-zinc-800 bg-zinc-950 text-zinc-300 hover:bg-zinc-800 disabled:opacity-30 cursor-pointer disabled:cursor-not-allowed"
                    >
                      <ArrowDown className="h-3.5 w-3.5" />
                    </button>
                    <button
                      type="button"
                      onClick={() => handleRemove(id)}
                      aria-label={`Ẩn ${LOBBY_ENTRY_LABELS[id] || id}`}
                      className="inline-flex h-8 w-8 items-center justify-center rounded border border-zinc-800 bg-zinc-950 text-rose-400 hover:bg-rose-950/40 cursor-pointer"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div>
        <div className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-400">
          3. Thêm mục hoặc nhóm vào Lưới Lobby ({availableIds.length})
        </div>
        {availableIds.length === 0 ? (
          <p className="text-xs text-zinc-500">Tất cả mục và nhóm đã được đưa ra Lưới Lobby.</p>
        ) : (
          <div className="flex flex-wrap gap-1.5">
            {availableIds.map((id) => {
              const isGroup = isLobbyGroupId(id);
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => handleAdd(id)}
                  className={`inline-flex min-h-8 items-center gap-1 rounded-lg border px-2.5 py-1 text-xs font-medium transition cursor-pointer ${
                    isGroup
                      ? "border-amber-500/30 bg-amber-950/20 text-amber-200 hover:bg-amber-950/40"
                      : "border-zinc-800 bg-zinc-900/60 text-zinc-300 hover:bg-zinc-800"
                  }`}
                >
                  <Plus className="h-3 w-3" />
                  <span>{LOBBY_ENTRY_LABELS[id] || id}</span>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
};
