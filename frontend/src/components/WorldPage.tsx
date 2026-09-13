"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import { useWorldStore } from "@/store/useWorldStore";
import { useGameStore } from "@/store/useGameStore";
import TeleportModal from "./TeleportModal";
import { Menu, X, User, Briefcase, Flame, Sparkles, Shield, Store, Trophy, Gift, Swords, Gem, Compass, Zap, Users } from "lucide-react";

const WorldCanvas = dynamic(() => import("@/components/WorldCanvas"), {
  ssr: false,
  loading: () => (
    <div className="w-full h-full flex items-center justify-center text-zinc-500 font-serif">
      Đang tải thế giới...
    </div>
  ),
});

const GuInventory = dynamic(() => import("./GuInventory").then(m => m.GuInventory), { ssr: false });
const Aperture = dynamic(() => import("./Aperture").then(m => m.Aperture), { ssr: false });
const RefiningFurnace = dynamic(() => import("./RefiningFurnace").then(m => m.RefiningFurnace), { ssr: false });
const SectView = dynamic(() => import("./SectView").then(m => m.SectView), { ssr: false });
const BaoHoangMarket = dynamic(() => import("./BaoHoangMarket").then(m => m.BaoHoangMarket), { ssr: false });
const LeaderboardView = dynamic(() => import("./LeaderboardView").then(m => m.LeaderboardView), { ssr: false });
const EntertainmentPanel = dynamic(() => import("./EntertainmentPanel").then(m => m.EntertainmentPanel), { ssr: false });
const ArenaView = dynamic(() => import("./ArenaView").then(m => m.ArenaView), { ssr: false });
const PremiumStoreView = dynamic(() => import("./PremiumStoreView").then(m => m.PremiumStoreView), { ssr: false });
const DungeonView = dynamic(() => import("./DungeonView").then(m => m.DungeonView), { ssr: false });
const ImmortalApertureView = dynamic(() => import("./ImmortalApertureView").then(m => m.ImmortalApertureView), { ssr: false });
const PartyView = dynamic(() => import("./PartyView").then(m => m.PartyView), { ssr: false });
const GuQuickBar = dynamic(() => import("./GuQuickBar").then(m => m.GuQuickBar), { ssr: false });
const TargetHud = dynamic(() => import("./TargetHud").then(m => m.TargetHud), { ssr: false });

const TYPE_CONFIG: Record<string, { icon: string; label: string; color: string; borderColor: string; bgColor: string }> = {
  Boss:       { icon: "⚔️",  label: "Boss",            color: "text-red-300",     borderColor: "border-red-500/30",     bgColor: "bg-red-950/60" },
  Monster:    { icon: "🐺",  label: "Yêu Thú",         color: "text-red-300",     borderColor: "border-red-500/30",     bgColor: "bg-red-950/60" },
  Resource:   { icon: "💎",  label: "Linh Tài",        color: "text-emerald-300", borderColor: "border-emerald-500/30", bgColor: "bg-emerald-950/60" },
  WildGu:     { icon: "🐛",  label: "Cổ Trùng",        color: "text-purple-300",  borderColor: "border-purple-500/30",  bgColor: "bg-purple-950/60" },
  Empty:      { icon: "🏯",  label: "Địa Điểm",        color: "text-amber-300",   borderColor: "border-amber-500/30",   bgColor: "bg-amber-950/60" },
  Player:     { icon: "🥷",  label: "Cổ Sư",           color: "text-purple-300",  borderColor: "border-purple-500/30",  bgColor: "bg-purple-950/60" },
  GroundDrop: { icon: "🎁",  label: "Chiến Lợi Phẩm", color: "text-amber-300",   borderColor: "border-amber-500/40",   bgColor: "bg-amber-950/70" },
};
const DEFAULT_CONFIG = { icon: "📌", label: "Địa Điểm", color: "text-zinc-300", borderColor: "border-zinc-500/30", bgColor: "bg-zinc-800/60" };

export function WorldPage() {
  const regions = useGameStore((s) => s.regions);
  const regionsLoaded = useGameStore((s) => s.regionsLoaded);
  const fetchRegions = useGameStore((s) => s.fetchRegions);
  const gamePlayer = useGameStore((s) => s.player);
  const error = useWorldStore((s) => s.error);
  const loading = useWorldStore((s) => s.loading);
  const selectedObject = useWorldStore((s) => s.selectedObject);
  const selectObject = useWorldStore((s) => s.selectObject);
  const isMoving = useWorldStore((s) => s.isMoving);
  const setMovementRange = useWorldStore((s) => s.setMovementRange);
  const liveX = useWorldStore((s) => s.liveX);
  const liveY = useWorldStore((s) => s.liveY);
  const fetchGrid = useGameStore((s) => s.fetchGrid);
  
  const monsterDictionary = useGameStore((s) => s.monsterDictionary);
  const itemDictionary = useGameStore((s) => s.itemDictionary);
  const guDictionary = useGameStore((s) => s.guDictionary);
  const fetchPlayerProfile = useGameStore((s) => s.fetchPlayerProfile);
  const movePlayer = useGameStore((s) => s.movePlayer);

  const feedGu = useGameStore((s) => s.feedGu);
  const feedGuGroup = useGameStore((s) => s.feedGuGroup);
  const feedAllGu = useGameStore((s) => s.feedAllGu);
  const unequipAllGu = useGameStore((s) => s.unequipAllGu);
  const actionLoading = useGameStore((s) => s.actionLoading);
  const equipGu = useGameStore((s) => s.equipGu);
  const useItem = useGameStore((s) => s.useItem);
  const useItems = useGameStore((s) => s.useItems);
  const updateGuStatus = useGameStore((s) => s.updateGuStatus);
  const setVitalGu = useGameStore((s) => s.setVitalGu);
  const useGu = useGameStore((s) => s.useGu);
  const useGus = useGameStore((s) => s.useGus);
  const dropGu = useGameStore((s) => s.dropGu);
  const dropGus = useGameStore((s) => s.dropGus);

  // RefiningFurnace
  const refineGu = useGameStore((s) => s.refineGu);

  // BaoHoangMarket
  const marketplaceListings = useGameStore((s) => s.marketplaceListings);
  const fetchMarketplaceListings = useGameStore((s) => s.fetchMarketplaceListings);
  const createMarketplaceListing = useGameStore((s) => s.createMarketplaceListing);
  const placeMarketplaceBid = useGameStore((s) => s.placeMarketplaceBid);
  const cancelMarketplaceListing = useGameStore((s) => s.cancelMarketplaceListing);
  const buyMarketplaceListing = useGameStore((s) => s.buyMarketplaceListing);

  // Party
  const fetchPartyStatus = useGameStore((s) => s.fetchPartyStatus);
  const fetchPartyInvitations = useGameStore((s) => s.fetchPartyInvitations);

  // Ground Drops
  const groundDrops = useGameStore((s) => s.groundDrops);
  const fetchGroundDrops = useGameStore((s) => s.fetchGroundDrops);
  const lootGroundDrop = useGameStore((s) => s.lootGroundDrop);

  const [mapMode, setMapMode] = useState<"select" | "drag">("select");
  const [menuOpen, setMenuOpen] = useState(false);
  const [activeModal, setActiveModal] = useState<string | null>(null);
  interface MapLabelSettings {
    monster: boolean;
    bossNearby: boolean;
    hub: boolean;
    khu_vuc: boolean;
    ngu_vuc: boolean;
  }

  const [labelSettings, setLabelSettings] = useState<MapLabelSettings>(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("gu_world_map_label_settings");
      if (saved) {
        try {
          const parsed = JSON.parse(saved);
          return {
            monster: !!parsed.monster,
            bossNearby: !!parsed.bossNearby,
            hub: parsed.hub !== false,
            khu_vuc: parsed.khu_vuc !== false,
            ngu_vuc: parsed.ngu_vuc !== false,
          };
        } catch (e) {}
      }
    }
    return { monster: false, bossNearby: false, hub: true, khu_vuc: true, ngu_vuc: true };
  });

  const [showLabelMenu, setShowLabelMenu] = useState(false);

  const updateLabelSetting = (key: keyof MapLabelSettings, val: boolean) => {
    setLabelSettings((prev) => {
      const next = { ...prev, [key]: val };
      localStorage.setItem("gu_world_map_label_settings", JSON.stringify(next));
      return next;
    });
  };
  const setView = useGameStore((s) => s.setView);
  const isImmortal = gamePlayer ? gamePlayer.rank >= 6 : false;

  const menuItems = [
    { id: "character", label: "Nhân Vật", icon: User, color: "text-emerald-400" },
    { id: "inventory", label: "Túi Đồ", icon: Briefcase, color: "text-cyan-400" },
    { id: "aperture", label: "Không Khiếu", icon: Flame, color: "text-rose-400" },
    { id: "refine", label: "Luyện Cổ", icon: Sparkles, color: "text-purple-400" },
    { id: "party", label: "Tổ Đội", icon: Users, color: "text-amber-400" },
    { id: "sect", label: "Tông Môn", icon: Shield, color: "text-blue-400" },
    { id: "market", label: "Chợ Tiên", icon: Store, color: "text-amber-400", locked: !isImmortal },
    { id: "teleport", label: "Di Chuyển", icon: Zap, color: "text-yellow-400" },
    // { id: "leaderboard", label: "Xếp Hạng", icon: Trophy, color: "text-yellow-400" },
    // { id: "gacha", label: "Rút Thưởng", icon: Gift, color: "text-pink-400" },
    // { id: "arena", label: "Đấu Trường", icon: Swords, color: "text-red-400" },
    // { id: "premium_store", label: "Cửa Hàng", icon: Gem, color: "text-sky-400" },
    // { id: "dungeon", label: "Bí Cảnh", icon: Compass, color: "text-indigo-400" },
  ];

  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
    fetchPartyStatus();
    fetchPartyInvitations();
    fetchGroundDrops();
  }, [regionsLoaded, fetchRegions, fetchPartyStatus, fetchPartyInvitations, fetchGroundDrops]);

  const setInvestigationRange = useWorldStore((s) => s.setInvestigationRange);

  // Movement and Investigation ranges come from the same stats as the grid viewport.
  useEffect(() => {
    fetchGrid().then((data) => {
      if (data?.stats) {
        if (data.stats.movement) setMovementRange(data.stats.movement);
        if (data.stats.investigation) setInvestigationRange(data.stats.investigation);
      }
    });
  }, [fetchGrid, setMovementRange, setInvestigationRange]);

  const currentZone = gamePlayer?.current_region
    ? regions.find((r) => r.id === gamePlayer.current_region)
    : undefined;

  const cfg = selectedObject ? TYPE_CONFIG[selectedObject.cell_type] || DEFAULT_CONFIG : DEFAULT_CONFIG;

  return (
    <div className="w-full h-full relative">
      {loading && (
        <div className="absolute top-20 sm:top-16 left-1/2 -translate-x-1/2 px-4 py-2 bg-zinc-900/80 rounded-xl border border-zinc-800 text-xs text-zinc-400 font-serif">
          Đang khám phá thế giới...
        </div>
      )}
      {error && (
        <div className="absolute top-20 sm:top-16 left-1/2 -translate-x-1/2 px-4 py-2 bg-red-950/80 rounded-xl border border-red-800/40 text-xs text-red-400 font-serif">
          {error}
        </div>
      )}

      <div className="w-full h-[calc(100vh-8rem)] min-h-[480px]">
        <WorldCanvas mode={mapMode} labelSettings={labelSettings} />
      </div>

      <TargetHud />

      <div className="absolute top-2 right-2 flex flex-col items-end gap-2 pointer-events-none z-30">
        <button
          onClick={() => setMenuOpen(!menuOpen)}
          className="w-10 h-10 flex items-center justify-center bg-zinc-900/85 backdrop-blur-sm rounded-xl border border-zinc-800/40 text-zinc-300 shadow-lg pointer-events-auto hover:bg-zinc-800 transition-colors cursor-pointer"
        >
          {menuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
        </button>

        {menuOpen && (
          <div className="flex flex-col gap-1.5 p-2 bg-zinc-900/95 backdrop-blur-md rounded-xl border border-zinc-700/50 shadow-2xl pointer-events-auto w-48 max-h-[70vh] overflow-y-auto animate-fade-in-down origin-top-right">
            {menuItems.map((item) => (
              <button
                key={item.id}
                onClick={() => {
                  setMenuOpen(false);
                  setActiveModal(item.id);
                }}
                className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-left transition-all cursor-pointer ${
                  item.locked 
                    ? "opacity-50 cursor-not-allowed bg-zinc-800/30 text-zinc-500" 
                    : "hover:bg-zinc-800/80 text-zinc-300 hover:text-white"
                }`}
                disabled={item.locked}
              >
                <item.icon className={`w-4 h-4 ${item.locked ? 'text-zinc-600' : item.color}`} />
                <span className="font-serif">{item.label}</span>
              </button>
            ))}
          </div>
        )}
      </div>

      {activeModal && (
        <div className="absolute inset-0 z-40 bg-black/70 flex items-center justify-center p-4 sm:p-8 backdrop-blur-sm overflow-hidden pointer-events-auto">
          <div className="relative w-full h-full max-w-7xl max-h-[90vh] bg-zinc-950/95 rounded-2xl border border-zinc-800 shadow-2xl flex flex-col overflow-y-auto">
            <button
              onClick={() => setActiveModal(null)}
              className="absolute top-4 right-6 z-50 p-2 rounded-xl bg-zinc-800/80 hover:bg-zinc-700 text-zinc-300 hover:text-white transition-colors border border-zinc-600 cursor-pointer shadow-lg"
              title="Đóng"
            >
              <X className="w-6 h-6" />
            </button>
            <div className="flex-1 w-full p-2 sm:p-4 overflow-y-auto relative">
              {activeModal === "character" && <Aperture player={gamePlayer!} guDictionary={guDictionary} />}
              {activeModal === "inventory" && (
                <GuInventory
                  player={gamePlayer!}
                  onFeed={feedGu}
                  onFeedGroup={feedGuGroup}
                  onFeedAll={feedAllGu}
                  onUnequipAll={unequipAllGu}
                  actionLoading={actionLoading}
                  guDictionary={guDictionary}
                  itemDictionary={itemDictionary}
                  onEquipToggle={equipGu}
                  onUseItem={useItem}
                  onUseItems={useItems}
                  onUpdateStatus={updateGuStatus}
                  onSetVital={setVitalGu}
                  onUseGu={useGu}
                  onUseGus={useGus}
                  onDropGu={dropGu}
                  onDropGus={dropGus}
                />
              )}
              {activeModal === "aperture" && <ImmortalApertureView />}
              {activeModal === "refine" && (
                <RefiningFurnace
                  player={gamePlayer!}
                  guDictionary={guDictionary}
                  itemDictionary={itemDictionary}
                  actionLoading={actionLoading}
                  onRefine={refineGu}
                />
              )}
              {activeModal === "party" && <PartyView />}
              {activeModal === "sect" && <SectView />}
              {activeModal === "market" && (
                <BaoHoangMarket
                  player={gamePlayer!}
                  guInventory={gamePlayer!.gu_inventory || []}
                  materials={gamePlayer!.materials_inventory || {}}
                  actionLoading={actionLoading}
                  listings={marketplaceListings}
                  onRefresh={fetchMarketplaceListings}
                  onCreate={createMarketplaceListing}
                  onBid={placeMarketplaceBid}
                  onCancel={cancelMarketplaceListing}
                  onBuy={buyMarketplaceListing}
                  guDictionary={guDictionary}
                  itemDictionary={itemDictionary}
                />
              )}
              {activeModal === "leaderboard" && <LeaderboardView />}
              {activeModal === "gacha" && <EntertainmentPanel />}
              {activeModal === "arena" && <ArenaView />}
              {activeModal === "premium_store" && <PremiumStoreView />}
              {activeModal === "dungeon" && <DungeonView />}
              {activeModal === "teleport" && (
                <TeleportModal isOpen={true} onClose={() => setActiveModal(null)} />
              )}
            </div>
          </div>
        </div>
      )}

      <div className="absolute top-2 left-2 flex flex-col gap-2 pointer-events-none z-10">
        {gamePlayer && (
          <div className="px-3 py-1.5 flex flex-col items-center bg-zinc-900/85 backdrop-blur-sm rounded-xl border border-zinc-800/40 text-xs text-zinc-300 font-serif shadow-lg pointer-events-auto">
            <div className="w-full flex flex-col gap-1 my-1">
              <div className="relative w-32 h-3 bg-zinc-800 rounded overflow-hidden" title={`Máu: ${Math.round(gamePlayer.hp)}`}>
                <div className="h-full bg-red-500 transition-all duration-300" style={{ width: `${Math.min(100, (gamePlayer.hp / (gamePlayer.base_hp || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[10px] font-bold text-white drop-shadow-md">
                  {Math.round(gamePlayer.hp).toLocaleString('vi-VN')}
                </span>
              </div>
              <div className="relative w-32 h-3 bg-zinc-800 rounded overflow-hidden" title={`Chân Nguyên: ${Math.floor(gamePlayer.current_mana)}`}>
                <div className="h-full bg-sky-500 transition-all duration-300" style={{ width: `${gamePlayer.rank >= 6 ? 100 : Math.min(100, (gamePlayer.current_mana / (gamePlayer.max_mana || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[10px] font-bold text-white drop-shadow-md">
                  {Math.floor(gamePlayer.current_mana).toLocaleString('vi-VN')}
                </span>
              </div>
              <div className="relative w-32 h-3 bg-zinc-800 rounded overflow-hidden" title={`Linh Hồn: ${Math.round(gamePlayer.soul_hp)}`}>
                <div className="h-full bg-purple-500 transition-all duration-300" style={{ width: `${gamePlayer.rank >= 6 ? 100 : Math.min(100, (gamePlayer.soul_hp / (gamePlayer.max_soul || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[10px] font-bold text-white drop-shadow-md">
                  {Math.round(gamePlayer.soul_hp).toLocaleString('vi-VN')}
                </span>
              </div>
              <div className="relative w-32 h-3 bg-zinc-800 rounded overflow-hidden" title={`Tinh Thần: ${Math.round(gamePlayer.current_thought)}`}>
                <div className="h-full bg-amber-500 transition-all duration-300" style={{ width: `${gamePlayer.rank >= 6 ? 100 : Math.min(100, (gamePlayer.current_thought / (gamePlayer.max_thought || 1)) * 100)}%` }} />
                <span className="absolute inset-0 flex items-center justify-center text-[10px] font-bold text-white drop-shadow-md">
                  {Math.round(gamePlayer.current_thought).toLocaleString('vi-VN')}
                </span>
              </div>
            </div>
            <div className="flex flex-col items-center justify-center mt-1">
              <div className="text-emerald-400 font-bold text-sm flex items-center gap-1.5">
                <span>{currentZone?.name || gamePlayer.current_region || "Ngoài bản đồ"}</span>
              </div>
              <div className="mt-1">
                {currentZone?.zone_type === "HUB" || currentZone?.zone_type === "SAFE_ZONE" ? (
                  <span className="px-2 py-0.5 text-[10px] font-mono font-semibold rounded-md bg-emerald-950/90 text-emerald-400 border border-emerald-500/40 shadow-sm" title="Vùng An Toàn: Nghiêm cấm đồ sát / Không thể PvP">
                    🛡️ Vùng An Toàn (Cấm PvP)
                  </span>
                ) : currentZone?.zone_type === "PVP_ZONE" ? (
                  <span className="px-2 py-0.5 text-[10px] font-mono font-semibold rounded-md bg-red-950/90 text-red-400 border border-red-500/40 animate-pulse shadow-sm" title="Vùng PvP Nguy Hiểm: Tự do đồ sát, nguy cơ rơi đồ cao!">
                    ⚔️ Vùng PvP (Tự Do Đồ Sát)
                  </span>
                ) : (
                  <span className="px-2 py-0.5 text-[10px] font-mono font-semibold rounded-md bg-amber-950/90 text-amber-300 border border-amber-500/40 shadow-sm" title="Vùng Dã Ngoại: Cho phép giao tranh PvP">
                    ⚠️ Vùng Dã Ngoại (PvP Mở)
                  </span>
                )}
              </div>
            </div>
            <div className="text-[10px] text-zinc-400 font-mono mt-1 text-center">
              X:{Math.round(liveX || gamePlayer.x)} Y:{Math.round(liveY || gamePlayer.y)}
            </div>
          </div>
        )}

        <div className="flex flex-col gap-1.5 p-1.5 bg-zinc-900/85 backdrop-blur-sm rounded-xl border border-zinc-800/40 shadow-lg pointer-events-auto w-fit">
          <button
            onClick={() => setMapMode(mapMode === "select" ? "drag" : "select")}
            title={mapMode === "select" ? "Chế độ: Di chuyển (Click để đổi sang Kéo bản đồ)" : "Chế độ: Kéo bản đồ (Click để đổi sang Di chuyển)"}
            className={`w-9 h-9 flex items-center justify-center rounded-lg transition-all cursor-pointer ${
              mapMode === "select"
                ? "bg-emerald-500/20 text-emerald-300 border border-emerald-500/30 shadow-inner"
                : "bg-amber-500/20 text-amber-300 border border-amber-500/30 shadow-inner"
            }`}
          >
            <span className="text-lg">{mapMode === "select" ? "🎯" : "🖐️"}</span>
          </button>
          <div className="border-t border-zinc-700/40 my-0.5 mx-1" />
          <button
            onClick={() => useWorldStore.getState().requestCenterCamera()}
            title="Về vị trí"
            className="w-9 h-9 flex items-center justify-center rounded-lg transition-all cursor-pointer text-sky-400 hover:text-sky-300 border border-transparent hover:bg-sky-500/10"
          >
            <span className="text-lg">📍</span>
          </button>
          <div className="border-t border-zinc-700/40 my-0.5 mx-1" />
          <div className="relative">
            <button
              onClick={() => setShowLabelMenu(!showLabelMenu)}
              title="Cài đặt hiển thị nhãn"
              className={`w-9 h-9 flex items-center justify-center rounded-lg transition-all cursor-pointer ${
                showLabelMenu || Object.values(labelSettings).some(Boolean)
                  ? "bg-purple-500/20 text-purple-300 border border-purple-500/30 shadow-inner"
                  : "text-zinc-400 hover:text-zinc-200 border border-transparent hover:bg-zinc-800/50"
              }`}
            >
              <span className="text-lg">🏷️</span>
            </button>
            {showLabelMenu && (
              <div className="absolute left-11 top-0 ml-1.5 p-3 bg-zinc-950/95 backdrop-blur-md rounded-xl border border-zinc-800 shadow-2xl z-50 w-48 flex flex-col gap-2 pointer-events-auto select-none">
                <div className="text-[10px] text-zinc-500 font-bold uppercase tracking-wider mb-1 font-serif">Hiển thị nhãn</div>
                <label className="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer hover:text-white">
                  <input
                    type="checkbox"
                    checked={labelSettings.monster}
                    onChange={(e) => updateLabelSetting("monster", e.target.checked)}
                    className="accent-purple-500 cursor-pointer"
                  />
                  <span className="font-serif">Quái & Cổ trùng</span>
                </label>
                <label className="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer hover:text-white">
                  <input
                    type="checkbox"
                    checked={labelSettings.bossNearby}
                    onChange={(e) => updateLabelSetting("bossNearby", e.target.checked)}
                    className="accent-red-500 cursor-pointer"
                  />
                  <span className="font-serif">Boss lân cận</span>
                </label>
                <label className="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer hover:text-white">
                  <input
                    type="checkbox"
                    checked={labelSettings.hub}
                    onChange={(e) => updateLabelSetting("hub", e.target.checked)}
                    className="accent-purple-500 cursor-pointer"
                  />
                  <span className="font-serif">Điểm tập trung (Hub)</span>
                </label>
                <label className="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer hover:text-white">
                  <input
                    type="checkbox"
                    checked={labelSettings.khu_vuc}
                    onChange={(e) => updateLabelSetting("khu_vuc", e.target.checked)}
                    className="accent-purple-500 cursor-pointer"
                  />
                  <span className="font-serif">Khu vực</span>
                </label>
                <label className="flex items-center gap-2 text-xs text-zinc-300 cursor-pointer hover:text-white">
                  <input
                    type="checkbox"
                    checked={labelSettings.ngu_vuc}
                    onChange={(e) => updateLabelSetting("ngu_vuc", e.target.checked)}
                    className="accent-purple-500 cursor-pointer"
                  />
                  <span className="font-serif">Ngự vực</span>
                </label>
              </div>
            )}
          </div>
        </div>
      </div>
      {selectedObject && (
        <div className="absolute bottom-28 left-1/2 -translate-x-1/2 pointer-events-auto animate-fade-in z-20">
          <div className={`relative w-72 px-4 py-4 rounded-2xl border ${cfg.borderColor} ${cfg.bgColor} backdrop-blur-md shadow-2xl`}>
            <button
              onClick={() => selectObject(null)}
              className="absolute top-2 right-2 w-6 h-6 flex items-center justify-center rounded-full bg-zinc-800/60 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-xs transition-colors cursor-pointer"
            >
              ✕
            </button>

            <div className="flex items-center gap-2.5 mb-3">
              <div className={`w-9 h-9 rounded-xl ${cfg.bgColor} border ${cfg.borderColor} flex items-center justify-center text-lg shrink-0`}>
                {cfg.icon}
              </div>
              <div>
                <div className={`font-bold font-serif text-sm ${cfg.color}`}>
                  {selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss"
                    ? monsterDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "Resource"
                    ? itemDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "WildGu"
                    ? guDictionary[selectedObject.content_id]?.name || selectedObject.content_id
                    : selectedObject.cell_type === "GroundDrop"
                    ? `Hài cốt: ${(selectedObject.dropData || groundDrops.find(d => d.id === selectedObject.id))?.loser_name || "Vô danh"}`
                    : selectedObject.content_id}
                </div>
                <div className="text-[10px] text-zinc-500 font-serif">
                  {cfg.label} {selectedObject.rank ? `• Rank ${selectedObject.rank}` : ""}
                </div>
              </div>
            </div>

            <div className="flex gap-3 text-[10px] text-zinc-500 font-mono mb-3">
              <span>X: {selectedObject.x}</span>
              <span>Y: {selectedObject.y}</span>
            </div>

            {selectedObject.cell_type === "Boss" && selectedObject.cleared && (
              <div className="px-2 py-1.5 rounded-lg bg-zinc-800/60 text-[10px] text-zinc-400 font-serif mb-3">
                💀 Quái vật đang hồi sinh — hãy quay lại sau.
              </div>
            )}

            {selectedObject.cell_type === "GroundDrop" && (() => {
              const drop = selectedObject.dropData || groundDrops.find(d => d.id === selectedObject.id);
              if (!drop) return null;
              const protExpiry = drop.protected_until ? new Date(drop.protected_until) : null;
              const isProtected = drop.is_protected || (protExpiry ? protExpiry.getTime() > Date.now() : false);
              const isKiller = gamePlayer?.id && drop.killer_id === gamePlayer.id;

              return (
                <div className="mb-3 space-y-1.5 text-xs font-serif">
                  <div className="flex items-center justify-between text-[11px] text-zinc-400 bg-zinc-900/60 p-2 rounded-lg border border-zinc-800">
                    <span>⚔️ Kẻ diệt: <strong className="text-amber-400">{drop.killer_name || "Yêu thú/Hệ thống"}</strong></span>
                    {isProtected ? (
                      isKiller ? (
                        <span className="text-emerald-400 text-[10px] font-bold">✨ Quyền riêng</span>
                      ) : (
                        <span className="text-rose-400 text-[10px] font-bold">🔒 Khóa Killer</span>
                      )
                    ) : (
                      <span className="text-amber-400 text-[10px] font-bold">🔓 Tự do nhặt</span>
                    )}
                  </div>
                  <div className="grid grid-cols-2 gap-1 text-[11px] text-amber-200/90 bg-amber-950/30 p-2 rounded-lg border border-amber-900/40 font-mono">
                    <div>💎 {drop.loot_data?.primeval_stones || 0} Đá NT</div>
                    <div>✨ {drop.loot_data?.immortal_stones || 0} Tiên Thạch</div>
                    {drop.loot_data?.items && drop.loot_data.items.length > 0 && (
                      <div>📦 {drop.loot_data.items.length} Vật phẩm</div>
                    )}
                    {drop.loot_data?.gus && drop.loot_data.gus.length > 0 && (
                      <div>🐛 {drop.loot_data.gus.length} Cổ trùng</div>
                    )}
                    {drop.loot_data?.recipes && drop.loot_data.recipes.length > 0 && (
                      <div>📜 {drop.loot_data.recipes.length} Công thức</div>
                    )}
                  </div>
                </div>
              );
            })()}

            <div className="flex gap-2.5">
              {selectedObject.cell_type === "Player" ? (
                (() => {
                  const dx = Math.abs((gamePlayer?.x ?? 0) - selectedObject.x);
                  const dy = Math.abs((gamePlayer?.y ?? 0) - selectedObject.y);
                  const dist = Math.max(dx, dy);
                  const inPvPRange = dist <= 2;

                  return (
                    <>
                      <button
                        disabled={isMoving}
                        title={isMoving ? "Đang di chuyển..." : inPvPRange ? "Tấn Công (PvP)" : "Tiếp Cận Đối Phương"}
                        onClick={async () => {
                          if (inPvPRange) {
                            movePlayer(gamePlayer!.x, gamePlayer!.y, selectedObject.id);
                          } else {
                            window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                          }
                        }}
                        className={`flex-1 flex items-center justify-center py-2.5 rounded-xl ${
                          inPvPRange ? "bg-red-600 hover:bg-red-500 shadow-red-950/30" : "bg-amber-600 hover:bg-amber-500 shadow-amber-950/30"
                        } disabled:opacity-60 text-white text-base transition-all cursor-pointer shadow-lg`}
                      >
                        <span>{inPvPRange ? "⚔️" : "🏃"}</span>
                      </button>
                  <button
                    title="Trinh Thám"
                    onClick={() => fetchPlayerProfile(selectedObject.content_id)}
                    className="flex-1 flex items-center justify-center py-2.5 rounded-xl bg-purple-600 hover:bg-purple-500 shadow-purple-950/30 text-white text-base transition-all cursor-pointer shadow-lg"
                  >
                    <span>👁️</span>
                  </button>
                  {gamePlayer && gamePlayer.username !== selectedObject.content_id && (
                    <button
                      title="Mời Tổ Đội"
                      onClick={async () => {
                        let currentParty = useGameStore.getState().party;
                        if (!currentParty) {
                          const created = await useGameStore.getState().createParty();
                          if (!created) return;
                          currentParty = useGameStore.getState().party;
                        }
                        if (currentParty && currentParty.leader_id !== gamePlayer.id) {
                          useGameStore.getState().addToast("Chỉ có Trưởng nhóm mới có quyền mời!", "error");
                          return;
                        }
                        await useGameStore.getState().invitePlayer(selectedObject.content_id);
                      }}
                      className="flex-1 flex items-center justify-center py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 shadow-amber-950/30 text-white text-base transition-all cursor-pointer shadow-lg"
                    >
                      <span>🤝</span>
                    </button>
                  )}
                </>
              );
                })()
              ) : selectedObject.cell_type === "GroundDrop" ? (
                gamePlayer?.x === selectedObject.x && gamePlayer?.y === selectedObject.y ? (
                  <button
                    onClick={async () => {
                      const drop = selectedObject.dropData || groundDrops.find(d => d.id === selectedObject.id);
                      const res = await lootGroundDrop(drop?.id || selectedObject.id!);
                      if (res?.success) {
                        selectObject(null);
                      }
                    }}
                    className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 shadow-amber-950/30 text-white text-sm font-bold font-serif transition-all cursor-pointer shadow-lg"
                  >
                    <span>🎁</span>
                    <span>Lụm</span>
                  </button>
                ) : (
                  <button
                    disabled={isMoving}
                    title="Tiếp Cận Chiến Lợi Phẩm"
                    onClick={() => {
                      window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                    }}
                    className="flex-1 flex items-center justify-center gap-2 py-2.5 rounded-xl bg-amber-600 hover:bg-amber-500 shadow-amber-950/30 text-white text-sm font-bold font-serif transition-all cursor-pointer shadow-lg"
                  >
                    <span>🏃</span>
                    <span>Tiếp Cận</span>
                  </button>
                )
              ) : (
                <button
                  disabled={isMoving || !!selectedObject.cleared}
                  title={isMoving ? "Đang di chuyển..." : selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss" ? "Chiến Đấu" : "Tiếp Cận"}
                  onClick={() => {
                    window.dispatchEvent(new CustomEvent("gu_world_move", { detail: { x: selectedObject.x, y: selectedObject.y } }));
                  }}
                  className={`flex-1 flex items-center justify-center py-2.5 rounded-xl ${
                    selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss"
                      ? "bg-red-600 hover:bg-red-500 shadow-red-950/30"
                      : "bg-emerald-600 hover:bg-emerald-500 shadow-emerald-950/30"
                  } disabled:opacity-60 text-white text-base transition-all cursor-pointer shadow-lg`}
                >
                  <span>{selectedObject.cell_type === "Monster" || selectedObject.cell_type === "Boss" ? "⚔️" : "🏃"}</span>
                </button>
              )}
              <button
                title="Đóng"
                onClick={() => selectObject(null)}
                className="px-4 py-2.5 rounded-xl bg-zinc-800 hover:bg-zinc-700 text-zinc-400 hover:text-zinc-200 text-sm font-bold transition-all cursor-pointer border border-zinc-700/40"
              >
                ✕
              </button>
            </div>
          </div>
        </div>
      )}

      <GuQuickBar />
    </div>
  );
}
