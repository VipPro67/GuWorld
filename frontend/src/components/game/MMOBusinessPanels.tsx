"use client";

import { useEffect } from "react";
import dynamic from "next/dynamic";
import { X, Settings2, Sparkles, Users, Package, Compass, Gem } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import { ClearCacheButton } from "@/components/ClearCacheModal";

const GuInventory = dynamic(() => import("@/components/GuInventory").then((m) => m.GuInventory), { ssr: false });
const Aperture = dynamic(() => import("@/components/Aperture").then((m) => m.Aperture), { ssr: false });
const PartyView = dynamic(() => import("@/components/PartyView").then((m) => m.PartyView), { ssr: false });
const WorldAtlasModal = dynamic(() => import("@/components/WorldAtlasModal").then((m) => m.WorldAtlasModal), { ssr: false });
const MMOSettingsPanel = dynamic(() => import("./MMOSettingsPanel").then((m) => m.MMOSettingsPanel), { ssr: false });
const WorldResourcePointsPanel = dynamic(() => import("./WorldResourcePointsPanel").then((m) => m.WorldResourcePointsPanel), { ssr: false });

export type MMOPanel = "inventory" | "cultivation" | "party" | "settings" | "maps" | "resources" | null;

interface MMOBusinessPanelsProps {
  panel: MMOPanel;
  onClose: () => void;
  onLoadoutSaved: () => void;
  /** Realtime player position (map-local cells) for world-mine distance checks. */
  position?: { x: number; y: number } | null;
}

const PANEL_META: Record<Exclude<MMOPanel, null>, { title: string; icon: typeof Package }> = {
  inventory: { title: "Túi đồ & Cổ trùng", icon: Package },
  cultivation: { title: "Tu luyện", icon: Sparkles },
  party: { title: "Tổ đội", icon: Users },
  settings: { title: "Cài đặt & Tiện ích", icon: Settings2 },
  maps: { title: "Bản đồ Cổ Giới", icon: Compass },
  resources: { title: "Mỏ tài nguyên công khai", icon: Gem },
};

export function MMOBusinessPanels({ panel, onClose, onLoadoutSaved, position }: MMOBusinessPanelsProps) {
  const store = useGameStore();
  useEffect(() => { if (panel === "maps") void store.fetchRegions(); }, [panel, store.fetchRegions]);
  if (!panel) return null;
  if (panel === "maps") return <WorldAtlasModal regions={store.regions} currentRegionId={store.player?.current_region ?? ""} x={store.player?.x ?? -1} y={store.player?.y ?? -1} onClose={onClose} />;
  const meta = PANEL_META[panel];
  const Icon = meta.icon;
  return (
    <div className="absolute inset-0 z-30 flex items-center justify-center bg-black/75 backdrop-blur-sm">
      <button
        type="button"
        aria-label="Đóng bảng điều khiển"
        className="absolute inset-0"
        onClick={onClose}
      />
      <aside
        role="dialog"
        aria-modal="true"
        aria-label={meta.title}
        className="mmo-panel-theme mmo-paper-theme mmo-paper-surface relative z-10 flex h-full w-full flex-col border border-[#80603a] text-[#49331f] md:h-[min(90dvh,900px)] md:max-h-[90dvh] md:w-[min(92vw,1152px)] md:rounded-xl"
      >
        <header className="mmo-paper-heading flex h-14 shrink-0 items-center justify-between gap-3 px-4">
          <div className="flex items-center gap-2">
            <span className="mmo-paper-seal">令</span>
            <Icon className="h-4 w-4 text-[#79542f]" />
            <div className="font-serif text-base font-semibold text-[#49331f]">{meta.title}</div>
          </div>
          <div className="flex items-center gap-2">
            <ClearCacheButton className="text-[10px]" />
            <button
              type="button"
              onClick={onClose}
              className="mmo-bamboo-action grid h-8 w-8 place-items-center"
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
              onUseGu={store.useGu}
              onUseSupportGu={store.useSupportGu}
              onDropGu={store.dropGu}
              onDropGus={store.dropGus}
              onUseGus={store.useGus}
            />
            </div>
          ) : panel === "cultivation" && store.player ? (
            <div className="mmo-paper-aperture">
              <Aperture player={store.player} guDictionary={store.guDictionary} onRealtimeHotbarSaved={onLoadoutSaved} />
            </div>
          ) : panel === "settings" ? (
            <MMOSettingsPanel onLoadoutSaved={onLoadoutSaved} />
          ) : panel === "party" ? (
            <PartyView />
          ) : panel === "resources" ? (
            <WorldResourcePointsPanel position={position} />
          ) : null}
        </div>
      </aside>
    </div>
  );
}
