"use client";

import { useEffect } from "react";
import dynamic from "next/dynamic";
import { X, Settings2, Sparkles, Users, Package, Compass } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import { ClearCacheButton } from "@/components/ClearCacheModal";

const GuInventory = dynamic(() => import("@/components/GuInventory").then((m) => m.GuInventory), { ssr: false });
const Aperture = dynamic(() => import("@/components/Aperture").then((m) => m.Aperture), { ssr: false });
const PartyView = dynamic(() => import("@/components/PartyView").then((m) => m.PartyView), { ssr: false });
const WorldAtlasModal = dynamic(() => import("@/components/WorldAtlasModal").then((m) => m.WorldAtlasModal), { ssr: false });
const MMOSettingsPanel = dynamic(() => import("./MMOSettingsPanel").then((m) => m.MMOSettingsPanel), { ssr: false });

export type MMOPanel = "inventory" | "cultivation" | "party" | "settings" | "maps" | null;

interface MMOBusinessPanelsProps {
  panel: MMOPanel;
  onClose: () => void;
  onLoadoutSaved: () => void;
}

const PANEL_META: Record<Exclude<MMOPanel, null>, { title: string; icon: typeof Package }> = {
  inventory: { title: "Túi đồ & Cổ trùng", icon: Package },
  cultivation: { title: "Tu luyện", icon: Sparkles },
  party: { title: "Tổ đội", icon: Users },
  settings: { title: "Cài đặt & Tiện ích", icon: Settings2 },
  maps: { title: "Bản đồ Cổ Giới", icon: Compass },
};

export function MMOBusinessPanels({ panel, onClose, onLoadoutSaved }: MMOBusinessPanelsProps) {
  const store = useGameStore();
  useEffect(() => { if (panel === "maps") void store.fetchRegions(); }, [panel, store.fetchRegions]);
  if (!panel) return null;
  if (panel === "maps") return <WorldAtlasModal regions={store.regions} currentRegionId={store.player?.current_region ?? ""} x={store.player?.x ?? -1} y={store.player?.y ?? -1} onClose={onClose} />;
  const meta = PANEL_META[panel];
  const Icon = meta.icon;
  return (
    <div className="absolute inset-0 z-30 flex justify-end">
      <button
        type="button"
        aria-label="Đóng bảng điều khiển"
        className="absolute inset-0 bg-black/40 backdrop-blur-[1px]"
        onClick={onClose}
      />
      <aside
        role="dialog"
        aria-label={meta.title}
        className="mmo-panel-theme relative z-10 flex h-full w-full max-w-[min(94vw,640px)] flex-col border-l border-[#a88952]/60 bg-[#150f0bf2] text-[#f4e8cb] shadow-[-24px_0_80px_rgba(0,0,0,.55)] backdrop-blur-xl"
      >
        <header className="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-[#a88952]/35 bg-[#2b2017]/70 px-4">
          <div className="flex items-center gap-2">
            <Icon className="h-4 w-4 text-[#d8b979]" />
            <div className="font-serif text-base font-semibold text-[#f4e8cb]">{meta.title}</div>
          </div>
          <div className="flex items-center gap-2">
            <ClearCacheButton className="text-[10px]" />
            <button
              type="button"
              onClick={onClose}
              className="grid h-8 w-8 place-items-center rounded-lg border border-[#a88952]/40 bg-[#3a291d] text-[#ead6ad] transition hover:bg-[#624329] hover:text-white"
              aria-label="Đóng"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto overscroll-contain p-2 sm:p-4">
          {panel === "inventory" && store.player ? (
            <div className="space-y-3">
            <GuInventory
              player={store.player}
              initialTab="material"
              onFeed={store.feedGu}
              onFeedGroup={store.feedGuGroup}
              onFeedAll={store.feedAllGu}
              onUnequipAll={store.unequipAllGu}
              actionLoading={store.actionLoading}
              guDictionary={store.guDictionary}
              itemDictionary={store.itemDictionary}
              onEquipToggle={store.equipGu}
              onUseItem={store.useItem}
              onUseItems={store.useItems}
              onUpdateStatus={store.updateGuStatus}
              onSetVital={store.setVitalGu}
              onUseSupportGu={store.useSupportGu}
              onDropGu={store.dropGu}
              onDropGus={store.dropGus}
              onUseGus={store.useGus}
            />
            </div>
          ) : panel === "cultivation" && store.player ? (
            <Aperture player={store.player} guDictionary={store.guDictionary} onRealtimeHotbarSaved={onLoadoutSaved} />
          ) : panel === "settings" ? (
            <MMOSettingsPanel onLoadoutSaved={onLoadoutSaved} />
          ) : panel === "party" ? (
            <PartyView />
          ) : null}
        </div>
      </aside>
    </div>
  );
}
