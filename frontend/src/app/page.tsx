"use client";

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useShallow } from "zustand/react/shallow";
import { useGameStore } from "@/store/useGameStore";
import { useUIStore } from "@/store/useUIStore";
import { Registration } from "@/components/Registration";
import { OnboardingModal } from "@/components/OnboardingModal";
import { LevelZeroView } from "@/components/LevelZeroView";
import { Sparkles, X, Handshake, Store } from "@/lib/icons";
import { GlobalChatOverlay } from "@/components/GlobalChatOverlay";
import { NotificationBell } from "@/components/NotificationBell";
import { MaintenanceView } from "@/components/MaintenanceView";
import { PlayerProfileModal } from "@/components/PlayerProfileModal";
import { DivinationModal } from "@/components/DivinationModal";
import { TrackingRadarDrawer } from "@/components/TrackingRadarDrawer";
import { TradeRoomModal } from "@/components/TradeRoomModal";
import { GameLoadingScreen } from "@/components/GameLoadingScreen";
import { ClearCacheButton } from "@/components/ClearCacheModal";
import { installPlayerLifecycleReconciliation } from "@/lib/player-reconciliation";

const Aperture = dynamic(() => import("@/components/Aperture").then(m => m.Aperture), { ssr: false });
const GuInventory = dynamic(() => import("@/components/GuInventory").then(m => m.GuInventory), { ssr: false });
const RefiningFurnace = dynamic(() => import("@/components/RefiningFurnace").then(m => m.RefiningFurnace), { ssr: false });
const CombatScreen = dynamic(() => import("@/components/CombatScreen").then(m => m.CombatScreen), { ssr: false });
const CaravanShop = dynamic(() => import("@/components/CaravanShop").then(m => m.CaravanShop), { ssr: false });
const MerchantTradeView = dynamic(() => import("@/components/MerchantTradeView").then(m => m.MerchantTradeView), { ssr: false });
const PremiumStoreView = dynamic(() => import("@/components/PremiumStoreView").then(m => m.PremiumStoreView), { ssr: false });
const BaoHoangMarket = dynamic(() => import("@/components/BaoHoangMarket").then(m => m.BaoHoangMarket), { ssr: false });
const ArenaView = dynamic(() => import("@/components/ArenaView").then(m => m.ArenaView), { ssr: false });
const DeathArenaView = dynamic(() => import("@/components/DeathArena/DeathArenaView"), { ssr: false });
const MartialArenaView = dynamic(() => import("@/components/MartialArena/MartialArenaView"), { ssr: false });
const EntertainmentPanel = dynamic(() => import("@/components/EntertainmentPanel").then(m => m.EntertainmentPanel), { ssr: false });
const AchievementsPanel = dynamic(() => import("@/components/AchievementsPanel"), { ssr: false });
const ImmortalApertureView = dynamic(() => import("@/components/ImmortalApertureView").then(m => m.ImmortalApertureView), { ssr: false });
const DungeonView = dynamic(() => import("@/components/DungeonView").then(m => m.DungeonView), { ssr: false });
const DreamRealmView = dynamic(() => import("@/components/DreamRealmView").then(m => m.DreamRealmView), { ssr: false });
const PetPanel = dynamic(() => import("@/components/PetPanel").then(m => m.PetPanel), { ssr: false });
const SectView = dynamic(() => import("@/components/SectView").then(m => m.SectView), { ssr: false });
const PartyView = dynamic(() => import("@/components/PartyView").then(m => m.PartyView), { ssr: false });
const CombatHistoryView = dynamic(() => import("@/components/CombatHistoryView").then(m => m.CombatHistoryView), { ssr: false });
const FeedbackView = dynamic(() => import("@/components/FeedbackView").then(m => m.FeedbackView), { ssr: false });
const AuditLogsView = dynamic(() => import("@/components/AuditLogsView").then(m => m.AuditLogsView), { ssr: false });
const LeaderboardView = dynamic(() => import("@/components/LeaderboardView").then(m => m.LeaderboardView), { ssr: false });
const TrainingRoom = dynamic(() => import("@/components/TrainingRoom").then(m => m.TrainingRoom), { ssr: false });
const UtilitiesView = dynamic(() => import("@/components/UtilitiesView").then(m => m.UtilitiesView), { ssr: false });
const PhaserWorldHost = dynamic(() => import("@/components/game/PhaserWorldHost").then(m => m.PhaserWorldHost), { ssr: false });

export default function Home() {
  const router = useRouter();
  const [
    player,
    isLoading,
    actionLoading,
    error,
    login,
    register,
    sendOTP,
    completeOnboarding,
    completeTutorial,
    fetchPlayerStatus,
    initializeSession,
    reconcilePlayer,
    feedGu,
    feedGuGroup,
    feedAllGu,
    clearError,
    guDictionary,
    itemDictionary,
    fetchDictionary,
    refineGu,
    fleeCombat,
    activeCombatReplay,
    buyShopItem,
    sellShopItem,
    sellShopItemsBulk,
    equipGu,
    updateGuStatus,
    useItem,
    useItems,
    setVitalGu,
    useGu,
    useSupportGu,
    useGus,
    dropGu,
    dropGus,
    marketplaceListings,
    fetchMarketplaceListings,
    createMarketplaceListing,
    placeMarketplaceBid,
    cancelMarketplaceListing,
    buyMarketplaceListing,
    inspectedProfile,
    activeTrade,
    subscribeTradeEvents,
    unsubscribeTradeEvents,
    acceptTrade,
    declineTrade,
    cancelTrade,
    playerPets,
    monsterDictionary,
    party,
    dictionaryLoaded,
    dictionaryError,
  ] = useGameStore(
    useShallow((state) => [
      state.player,
      state.isLoading,
      state.actionLoading,
      state.error,
      state.login,
      state.register,
      state.sendOTP,
      state.completeOnboarding,
      state.completeTutorial,
      state.fetchPlayerStatus,
      state.initializeSession,
      state.reconcilePlayer,
      state.feedGu,
      state.feedGuGroup,
      state.feedAllGu,
      state.clearError,
      state.guDictionary,
      state.itemDictionary,
      state.fetchDictionary,
      state.refineGu,
      state.fleeCombat,
      state.activeCombatReplay,
      state.buyShopItem,
      state.sellShopItem,
      state.sellShopItemsBulk,
      state.equipGu,
      state.updateGuStatus,
      state.useItem,
      state.useItems,
      state.setVitalGu,
      state.useGu,
      state.useSupportGu,
      state.useGus,
      state.dropGu,
      state.dropGus,
      state.marketplaceListings,
      state.fetchMarketplaceListings,
      state.createMarketplaceListing,
      state.placeMarketplaceBid,
      state.cancelMarketplaceListing,
      state.buyMarketplaceListing,
      state.inspectedProfile,
      state.activeTrade,
      state.subscribeTradeEvents,
      state.unsubscribeTradeEvents,
      state.acceptTrade,
      state.declineTrade,
      state.cancelTrade,
      state.playerPets,
      state.monsterDictionary,
      state.party,
      state.dictionaryLoaded,
      state.dictionaryError,
    ])
  );

  const unequipAllGu = useGameStore((s) => s.unequipAllGu);
  const merchantAtVillage = useGameStore((s) => s.merchantAtVillage);
  const nearbyMerchants = useGameStore((s) => s.merchants);
  const fetchMerchants = useGameStore((s) => s.fetchMerchants);

  const currentView = useUIStore((s) => s.currentView);
  const isGameWorldSurface = ["dashboard", "mmo-world"].includes(currentView);
  const setView = useUIStore((s) => s.setView);
  const showChangelog = useUIStore((s) => s.showChangelog);
  const setShowChangelog = useUIStore((s) => s.setShowChangelog);

  useEffect(() => {
    if (currentView === "wiki") router.replace("/wiki");
  }, [currentView, router]);

  const [dontShowToday, setDontShowToday] = useState(false);

  const handleCloseChangelog = () => {
    if (dontShowToday) {
      const today = new Date().toISOString().split('T')[0];
      localStorage.setItem("gu_changelog_hide_until", today);
    }
    setDontShowToday(false);
    setShowChangelog(false);
  };

  // 1. Session Restoration on Mount
  useEffect(() => {
    fetchDictionary();
    initializeSession();
  }, [initializeSession, fetchDictionary]);

  const playerId = player?.id;
  useEffect(() => {
    if (!playerId) return;
    return installPlayerLifecycleReconciliation(window, document, () => void reconcilePlayer(true));
  }, [playerId, reconcilePlayer]);

  // 2. Sparse authoritative status sync. PostgreSQL (via /player/status) is the
  // source of truth for active_combat; keep reconciling while the combat UI is
  // open so a missed/stale realtime event cannot trap the player there.
  const hasActiveCombat = Boolean(player?.active_combat);
  useEffect(() => {
    if (!playerId) return;

    if (hasActiveCombat) {
      void fetchPlayerStatus(playerId);
    }

    const interval = setInterval(() => {
      if (document.visibilityState === "visible") {
        void fetchPlayerStatus(playerId);
      }
    }, hasActiveCombat ? 10000 : 90000);

    return () => clearInterval(interval);
  }, [playerId, hasActiveCombat, fetchPlayerStatus]);

  // 3. Trade realtime subscription. This replaces the old /api/trade/active polling loop.
  const isPlayerReady = !!player;
  useEffect(() => {
    if (!isPlayerReady) return;

    subscribeTradeEvents();
    return () => unsubscribeTradeEvents();
    // Only subscribe once when player becomes available, not on every player state change

  }, [isPlayerReady]);

  // Merchant proximity only needs refresh while the world is open.
  useEffect(() => {
    if (!playerId || !isGameWorldSurface) return;
    void fetchMerchants();
    const interval = setInterval(() => {
      if (document.visibilityState === "visible") {
        void fetchMerchants();
      }
    }, 30000);
    return () => clearInterval(interval);
  }, [playerId, player?.current_region, isGameWorldSurface, fetchMerchants]);

  const isPartyMemberNotLeader = Boolean(player?.party_id) && party !== null && party.leader_id !== player?.id;
  // Force combat view if player is actively in combat (and is leader or solo) or if we have an active replay with logs
  const hasReplayLogs = Boolean(activeCombatReplay && activeCombatReplay.enemy && (activeCombatReplay.logs?.length ?? 0) > 0);
  const inCombat = (player && player.active_combat && !isPartyMemberNotLeader) || hasReplayLogs;


  const IS_MAINTENANCE = process.env.NEXT_PUBLIC_MAINTENANCE_MODE === "true";

  if (IS_MAINTENANCE) {
    return (
      <div className="min-h-screen flex flex-col py-2 sm:py-8 px-1 sm:px-6 lg:px-8 relative bg-[var(--bg-primary)]">
        <header className="w-full max-w-[1600px] px-2 sm:px-8 mx-auto flex justify-between items-center pb-4 border-b border-[var(--border-default)] mb-4">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <div className="w-8 h-8 rounded-lg bg-[var(--brand-dim)] border border-[var(--brand-primary)] flex items-center justify-center shrink-0">
                <Sparkles className="w-4 h-4 text-emerald-400" />
              </div>
              <h1 className="text-lg sm:text-xl font-bold font-serif tracking-wider text-[var(--text-primary)]">
                Cổ Giới
              </h1>
            </div>
          </div>
        </header>

        <main className="flex-grow flex items-center justify-center w-full max-w-[1600px] px-2 sm:px-8 mx-auto">
          <MaintenanceView />
        </main>

        <footer className="w-full max-w-[1600px] px-2 sm:px-8 mx-auto text-center mt-8 pt-4 border-t border-zinc-950 flex flex-col items-center gap-2">
          <div className="flex items-center gap-3">
            <a href="https://www.facebook.com/profile.php?id=61587628762458" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-blue-400 hover:text-blue-300 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/20 transition-all">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>
              Fanpage
            </a>
            <a href="https://www.facebook.com/groups/2036276400523995" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-sky-400 hover:text-sky-300 bg-sky-500/10 hover:bg-sky-500/20 border border-sky-500/20 transition-all">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>
              Cộng Đồng
            </a>
            <a href="https://discord.gg/pTMuYUFngg" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-indigo-400 hover:text-indigo-300 bg-indigo-500/10 hover:bg-indigo-500/20 border border-indigo-500/20 transition-all">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994.021-.041.001-.09-.041-.106a13.094 13.094 0 0 1-1.873-.894.077.077 0 0 1-.008-.128c.126-.093.252-.19.372-.287a.075.075 0 0 1 .077-.011c3.92 1.793 8.18 1.793 12.061 0a.073.073 0 0 1 .078.009c.12.099.246.195.373.289a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.894.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.156 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.156 2.418z"/>
              </svg>
              Nhóm Discord
            </a>
            <ClearCacheButton showLabel={true} className="!text-amber-500 dark:!text-amber-400 !bg-amber-500/10 hover:!bg-amber-500/20 !border-amber-500/20" />
          </div>
          <p className="text-xs text-zinc-600 font-mono">Cổ Chân Nhân Web Game © 2026 • Thiết Kế Trực Quan Sinh Động</p>
        </footer>
      </div>
    );
  }

  if (!dictionaryLoaded) {
    return (
      <GameLoadingScreen
        error={dictionaryError}
        onRetry={() => void fetchDictionary(true)}
      />
    );
  }

  // Phaser is the only persistent gameplay world. Old view IDs remain aliases
  // for navigation compatibility and never open a legacy map surface.
  const isGameWorldView = Boolean(player && !player.needs_onboarding && !inCombat);
  // Full-bleed modes take the whole surface instead of the side drawer.
  const isFullBleedView = currentView === "practice-room" || isGameWorldSurface;
  const isWideFeatureView = ["aperture", "sect", "shop", "auction", "premium-store", "premium-shop", "topup", "arena", "death-arena", "martial-arena"].includes(currentView);

  const viewTitles: Record<string, string> = {
    "practice-room": "Phòng tập", cultivation: "Tu luyện", character: "Nhân vật", inventory: "Túi đồ", "inventory-gu": "Cổ trùng", party: "Tổ đội",    refine: "Luyện Cổ", topup: "Nạp", "premium-shop": "Cửa hàng tiên ngọc",
    "premium-store": "Cửa hàng", shop: "Thương hội", auction: "Chợ Tiên",
    arena: "Đấu trường", "death-arena": "Sinh tử đài", "martial-arena": "Đại Hội Võ Thuật", leaderboard: "Bảng xếp hạng",
    gacha: "Tiên duyên", achievements: "Thành tựu", pets: "Dị nhân",
    aperture: "Tiên khiếu", "dream-realm": "Mộng Cảnh", dungeon: "Bí cảnh & Truyền thừa", sect: "Tông môn",
    "combat-history": "Lịch sử chiến đấu", feedback: "Góp ý", "audit-logs": "Nhật ký hệ thống",
    "mmo-world": "Thế giới Realtime",
    utilities: "Tiện ích",
  };

  return (
    <div className={`${isGameWorldView ? "h-[100dvh] min-h-[100dvh] max-h-[100dvh] flex flex-col overflow-hidden p-0" : "min-h-screen flex flex-col py-2 sm:py-8 px-1 sm:px-6 lg:px-8"} relative`}>
      {/* Top Header Navigation */}
      {!isGameWorldView && <header className="w-full max-w-[1600px] px-2 sm:px-8 mx-auto pb-4 border-b border-[var(--border-default)] mb-4 flex justify-between items-center">
        <div className="flex items-center gap-4">
          <div
            onClick={() => {
              if (player && !inCombat && currentView !== "dashboard") {
                setView("dashboard");
              }
            }}
            className={`flex items-center gap-2 ${player && !inCombat && currentView !== "dashboard" ? "cursor-pointer hover:opacity-80 transition-opacity" : ""}`}
            title={player && !inCombat && currentView !== "dashboard" ? "Trở về Sảnh" : undefined}
          >
              <div className="w-8 h-8 rounded-lg bg-[var(--brand-dim)] border border-[var(--brand-primary)] flex items-center justify-center shrink-0">
                <Sparkles className="w-4 h-4 text-emerald-400" />
              </div>
              <h1 className="text-lg sm:text-xl font-bold font-serif tracking-wider text-[var(--text-primary)]">
                Cổ Giới
              </h1>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {player && !inCombat && <NotificationBell />}
        </div>
      </header>}
      {/* Main Content Area */}
      <main className={`w-full ${isGameWorldView ? "flex-1 min-h-0 max-w-none px-0 mx-0 overflow-hidden flex flex-col" : "flex-grow flex items-start justify-center max-w-[1600px] px-2 sm:px-8 mx-auto"}`}>
        {!player ? (
          <div className="py-12 w-full flex justify-center">
            <Registration
              onRegister={register}
              onLogin={login}
              onSendOTP={sendOTP}
              isLoading={isLoading}
              error={error}
              onClearError={clearError}
            />
          </div>
        ) : player.rank === 0 ? (
          <div className="py-12 w-full flex justify-center">
            <LevelZeroView onComplete={completeTutorial} />
          </div>
        ) : player.needs_onboarding ? (
          <div className="py-12 w-full flex justify-center">
            <OnboardingModal onComplete={completeOnboarding} />
          </div>
        ) : inCombat ? (
          <div className="w-full">
            <CombatScreen
              player={player}
              enemy={player.active_combat || activeCombatReplay!.enemy}
              guDictionary={guDictionary}
              actionLoading={actionLoading}
              onRetreat={fleeCombat}
            />
          </div>
        ) : (
          <div className="relative isolate w-full h-full min-h-0 flex-1 overflow-hidden bg-zinc-950">
            <PhaserWorldHost worldActive={isGameWorldSurface || currentView === "practice-room"} onExit={() => setView("character")} />
            {!isGameWorldSurface && (
              <section
                role="dialog"
                aria-label={viewTitles[currentView] || "Chức năng"}
                className={`mmo-panel-theme absolute z-40 flex flex-col overflow-hidden border border-[#a88952]/60 bg-[#150f0bf2] text-[#f4e8cb] shadow-[0_24px_80px_rgba(0,0,0,.55)] backdrop-blur-xl ${
                  isFullBleedView
                    ? "inset-0"
                    : `inset-x-0 inset-y-0 md:inset-y-3 md:left-auto md:right-3 md:top-3 md:rounded-2xl ${isWideFeatureView ? "md:w-[min(58vw,760px)]" : "md:w-[min(46vw,560px)]"}`
                }`}
              >
                {!isFullBleedView && (
                  <div className="flex min-h-14 shrink-0 items-center justify-between gap-3 border-b border-[#a88952]/35 bg-[#2b2017]/70 px-4 sm:px-5">
                    <div className="min-w-0">
                      <div className="truncate font-serif text-base font-semibold text-[#f4e8cb]">{viewTitles[currentView] || "Chức năng"}</div>
                      <div className="hidden text-[10px] uppercase tracking-[.2em] text-[#d8b979]/70 sm:block">Cổ Giới · Bảng điều khiển</div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {player && !inCombat && <NotificationBell />}
                      <button
                        onClick={() => setView("mmo-world")}
                        aria-label="Đóng bảng điều khiển"
                        className="grid h-9 w-9 shrink-0 place-items-center rounded-xl border border-[#a88952]/40 bg-[#3a291d] text-[#ead6ad] transition hover:bg-[#624329] hover:text-white"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                )}
                <div className={`min-h-0 flex-1 overflow-y-auto overscroll-contain p-2 sm:p-4 pb-28 sm:pb-8 ${isFullBleedView ? "p-0" : ""}`}>
        {currentView === "practice-room" ? (
          <div className="w-full">
            <TrainingRoom onClose={() => setView("dashboard")} />
          </div>
        ) : currentView === "character" || currentView === "cultivation" ? (
          <div className="w-full">
              <Aperture
                key={currentView}
                player={player}
                guDictionary={guDictionary}
              />
          </div>
        ) : currentView === "inventory" || currentView === "inventory-gu" ? (
          <div className="w-full">
            <GuInventory
              player={player}
              initialTab={currentView === "inventory-gu" ? "gu" : "material"}
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
              onUseSupportGu={useSupportGu}
              onUseGus={useGus}
              onDropGu={dropGu}
              onDropGus={dropGus}
            />
          </div>
        ) : currentView === "refine" ? (
          <div className="w-full">
            <RefiningFurnace
              player={player}
              guDictionary={guDictionary}
              itemDictionary={itemDictionary}
              actionLoading={actionLoading}
              onRefine={refineGu}
            />
          </div>
        ) : currentView === "topup" ? (
          <div className="w-full">
            <PremiumStoreView defaultTab="topup" />
          </div>
        ) : (currentView === "premium-shop" || currentView === "premium-store") ? (
          <div className="w-full">
            <PremiumStoreView defaultTab="packages" />
          </div>
        ) : currentView === "shop" ? (
          <div className="w-full space-y-6">
            {merchantAtVillage && (
              <CaravanShop
                player={player}
                actionLoading={actionLoading}
                onBuy={buyShopItem}
                onSell={sellShopItem}
                onSellBulk={sellShopItemsBulk}
              />
            )}
            {nearbyMerchants.length > 0 && <MerchantTradeView player={player} actionLoading={actionLoading} />}
            {!merchantAtVillage && nearbyMerchants.length === 0 && (
              <div className="py-16 flex flex-col items-center justify-center gap-3 text-zinc-500">
                <Store className="w-10 h-10 text-zinc-700" />
                <span className="text-sm font-mono text-center">
                  Bạn không ở gần thương nhân nào.
                  <br />
                  Hãy tới <span className="text-amber-400">thành trấn / thôn làng</span> để mua bán, hoặc tìm <span className="text-amber-400">Thương Nhân Lang Thang</span> ngoài hoang dã.
                </span>
              </div>
            )}
          </div>
        ) : currentView === "auction" ? (
          <BaoHoangMarket
            player={player}
            guInventory={player.gu_inventory || []}
            materials={player.materials_inventory || {}}
            actionLoading={actionLoading}
            listings={marketplaceListings as ApiResponseData[]}
            onRefresh={fetchMarketplaceListings}
            onCreate={createMarketplaceListing}
            onBid={placeMarketplaceBid}
            onCancel={cancelMarketplaceListing}
            onBuy={buyMarketplaceListing}
            guDictionary={guDictionary}
            itemDictionary={itemDictionary}
          />
        ) : currentView === "arena" ? (
          <ArenaView />
        ) : currentView === "death-arena" ? (
          <div className="w-full">
            <DeathArenaView />
          </div>
        ) : currentView === "martial-arena" ? (
          <div className="w-full">
            <MartialArenaView />
          </div>
        ) : currentView === "leaderboard" ? (
          <div className="w-full">
            <LeaderboardView />
          </div>
        ) : currentView === "gacha" ? (
          <div className="w-full">
            <EntertainmentPanel />
          </div>
        ) : currentView === "achievements" ? (
          <div className="w-full">
            <AchievementsPanel />
          </div>
        ) : currentView === "pets" ? (
          <div className="w-full">
            <PetPanel
              player={player}
              pets={playerPets}
              itemDictionary={itemDictionary}
              monsterDictionary={monsterDictionary}
              actionLoading={actionLoading}
            />
          </div>
        ) : currentView === "aperture" ? (
          <div className="w-full">
            <ImmortalApertureView />
          </div>
        ) : currentView === "dream-realm" ? (
          <div className="w-full">
            <DreamRealmView />
          </div>
        ) : currentView === "dungeon" ? (
          <div className="w-full">
            <DungeonView />
          </div>
        ) : currentView === "sect" ? (
          <div className="w-full">
            <SectView />
          </div>
        ) : currentView === "party" ? (
          <div className="w-full">
            <PartyView />
          </div>
        ) : currentView === "combat-history" ? (
          <div className="w-full">
            <CombatHistoryView />
          </div>
        ) : currentView === "feedback" ? (
          <div className="w-full">
            <FeedbackView />
          </div>
        ) : currentView === "audit-logs" ? (
          <div className="w-full">
            <AuditLogsView />
          </div>
        ) : currentView === "utilities" ? (
          <div className="w-full">
            <UtilitiesView />
          </div>
        ) : (
          <div className="py-12 text-center font-mono text-zinc-500">Giao diện không khả dụng</div>
        )}
                </div>
              </section>
            )}
          </div>
        )}
      </main>

      {/* Footer Details */}
      {!isGameWorldView && (
        <footer className="w-full max-w-[1600px] px-2 sm:px-8 mx-auto text-center mt-8 pt-4 border-t border-primary flex flex-col items-center gap-2">
          <div className="flex items-center gap-3">
            <a href="https://www.facebook.com/profile.php?id=61587628762458" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-blue-400 hover:text-blue-300 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/20 transition-all">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>
              Fanpage
            </a>
            <a href="https://www.facebook.com/groups/2036276400523995" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-sky-400 hover:text-sky-300 bg-sky-500/10 hover:bg-sky-500/20 border border-sky-500/20 transition-all">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>
              Cộng Đồng
            </a>
            <a href="https://discord.gg/pTMuYUFngg" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-indigo-400 hover:text-indigo-300 bg-indigo-500/10 hover:bg-indigo-500/20 border border-indigo-500/20 transition-all">
              <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24">
                <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994.021-.041.001-.09-.041-.106a13.094 13.094 0 0 1-1.873-.894.077.077 0 0 1-.008-.128c.126-.093.252-.19.372-.287a.075.075 0 0 1 .077-.011c3.92 1.793 8.18 1.793 12.061 0a.073.073 0 0 1 .078.009c.12.099.246.195.373.289a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.894.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.156 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.156-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.156 2.418z"/>
              </svg>
              Nhóm Discord
            </a>
            <ClearCacheButton showLabel={true} className="!text-amber-500 dark:!text-amber-400 !bg-amber-500/10 hover:!bg-amber-500/20 !border-amber-500/20" />
          </div>
          <p className="text-xs text-zinc-600 font-mono">Cổ Chân Nhân Web Game © 2026 • Thiết Kế Trực Quan Sinh Động</p>
        </footer>
      )}

      {/* Changelog Modal */}
      {player && showChangelog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[var(--bg-primary)] animate-fade-in" onClick={handleCloseChangelog}>
          <div className="w-full max-w-lg max-h-[90vh] sm:max-h-[85vh] flex flex-col bg-[var(--bg-surface)] border border-[var(--border-default)] rounded-3xl p-5 sm:p-6 shadow-2xl overflow-hidden animate-scale-up" onClick={e => e.stopPropagation()}>
            <div className="absolute top-0 right-0 w-32 h-32 bg-amber-500/5 rounded-full blur-xl pointer-events-none" />
            <div className="flex items-center justify-between mb-4 shrink-0">
              <div className="flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                <h3 className="text-base font-serif font-bold text-[var(--text-primary)]">Cập Nhật Mới!</h3>
              </div>
              <button onClick={handleCloseChangelog} className="text-zinc-500 hover:text-zinc-300 cursor-pointer"><X className="w-4 h-4" /></button>
            </div>
            <div className="flex-1 overflow-y-auto space-y-3 text-xs text-[var(--text-secondary)] leading-relaxed pr-1 custom-scrollbar">
              <p className="text-[var(--text-secondary)]">Phiên bản <strong className="text-amber-600 dark:text-amber-400">v0.8.4</strong> — Cải Tiến PvP Ngũ Vực & Chống Lợi Dụng Chuyển Đồ:</p>
              <div className="space-y-2">
                <div className="flex gap-2.5 items-start bg-sky-950/20 border border-sky-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🐾</span>
                  <div>
                    <span className="font-bold text-sky-300">Quái Thế Giới Truy Đuổi Từ Xa</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Quái thường phát hiện người chơi trong 9–17 ô theo Rank, đuổi tới 15–23 ô quanh điểm sinh và di chuyển 0,9–1,7 ô/giây theo Rank. Elite phát hiện trong 14 ô, đuổi tối đa 22 ô với tốc độ nhanh hơn quái thường 0,2 ô/giây; boss săn mục tiêu trong toàn vùng. Khi quái áp sát, combat tự mở mà không cần bước vào đúng ô sinh quái.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-red-950/20 border border-red-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">💥</span>
                  <div>
                    <span className="font-bold text-red-400">Thiên Đạo Tiêu Hủy (40% Void Destruction)</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Tăng tỷ lệ tổn thất khi tử đạo lên 40% tiền, 40% nguyên liệu, 30% Cổ trùng và Bí phương. Trong lượng mất, <strong>40% bị chấn nát tiêu hủy vĩnh viễn khỏi server</strong>, chỉ 60% còn lại rơi ra bãi đất (tồn tại 10 phút). Chống triệt để việc mượn cái chết để tuồn chuyển tài nguyên.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-amber-950/20 border border-amber-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">⚔️</span>
                  <div>
                    <span className="font-bold text-amber-300">Cơ Chế Sát Khí & Safe Zone</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Kẻ đồ sát có Sát Khí &gt; 10 sẽ bị tước bỏ quyền bảo hộ trong Safe Zone/Hub (người khác có thể vây sát ngay trong vùng an toàn, trừ khi kẻ này trốn vào Lãnh địa Tông Môn). Khi Sát Khí &gt; 5, Cổ trùng đang trang bị có thể bị đánh rơi khi bại trận (Cổ Bản Mệnh luôn được bảo toàn).</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-emerald-950/20 border border-emerald-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🛡️</span>
                  <div>
                    <span className="font-bold text-emerald-300">Miễn Chiến Hồi Sinh</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Khi bị người khác tấn công và thua trận, nạn nhân nhận buff <strong>Miễn Chiến 10 phút</strong> (chủ động khai chiến sẽ mất buff). Sát Khí không còn giảm theo thời gian; mỗi lần đồ sát nhận thêm Sát Khí theo chênh lệch cảnh giới (rank càng cao hạ rank càng thấp càng nhiều).</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-purple-950/20 border border-purple-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🏰</span>
                  <div>
                    <span className="font-bold text-purple-300">Hồi Sinh Ngũ Vực & Tông Môn Chiến</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Khi tử đạo, đạo hữu sẽ hồi sinh tại HUB/SAFE_ZONE an toàn gần nhất (ưu tiên HUB, fallback SAFE ZONE) trong Đại Vực tử vong; khi không có HUB/SAFE ZONE hợp lệ trên tầng đó thì hồi sinh tại vị trí tử vong. Sau khi hồi sinh, nạn nhân PvP nhận Miễn Chiến 10 phút (chủ động khai chiến sẽ mất buff). Tán tu không tông có thể phát động Tông Môn Chiến (khi thắng: Cổ/item/bí phương chia đều, Quỹ cướp được sẽ đi vào hư vô; khi thua: toàn đội bị tử đạo rơi đồ). Đặc biệt: <strong>Khi Tông Môn đang trong trạng thái Hộ Sơn bảo hộ (shield_until), người chơi bên trong sẽ không được lãnh địa che chở và vẫn có thể bị tấn công</strong>.</p>
                  </div>
                </div>
              </div>

              <p className="text-[var(--text-secondary)]">Phiên bản <strong className="text-amber-600 dark:text-amber-400">v0.8.3</strong> — Tuổi Thọ, Trạng Thái Tử Vong & Tiên Cương:</p>
              <div className="space-y-2">
                <div className="flex gap-2.5 items-start bg-red-950/20 border border-red-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">⏳</span>
                  <div>
                    <span className="font-bold text-red-400">Cơ chế Tuổi Thọ & Trạng Thái Tử Vong</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Thọ nguyên tiêu hao dần theo thời gian thực (nhân hệ số gia tốc time_speed). Khi tuổi thọ giảm về 0, Cổ sư rơi vào cõi chết và bị phong tỏa mọi hoạt động chiến đấu, thám hiểm, tu luyện cho tới khi khôi phục tuổi thọ.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-emerald-950/20 border border-emerald-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🧟</span>
                  <div>
                    <span className="font-bold text-emerald-300">Chủng Tộc Cương Thi (Tiên Cương) & Tiên Khiếu Chết</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Cương Thi sở hữu năng lực Bất Tử Tuổi Thọ, đổi lại bị cấm tự hồi phục HP, Chân Nguyên/Tiên Nguyên, Nguyên Lực, Hồn Phách. Khi lên Tiên Cương, gia tốc thời gian khóa ở 1.0 và Tiên Khiếu bị suy thoái giảm dần diện tích và tốc độ thời gian.</p>
                  </div>
                </div>
              </div>

              <p className="text-[var(--text-secondary)]">Phiên bản <strong className="text-amber-600 dark:text-amber-400">v0.8.2</strong> — Hợp Luyện & Cảnh Giới Chân Ý:</p>
              <div className="space-y-2">
                <div className="flex gap-2.5 items-start bg-amber-950/20 border border-amber-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">📊</span>
                  <div>
                    <span className="font-bold text-amber-300">Hiện chi tiết % Luyện Cổ</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Hiển thị chi tiết đóng góp tỷ lệ % của từng thành phần (tỷ lệ gốc, đạo ngân, cảnh giới, vận khí, hỗ trợ, địa lợi...) khi lò luyện thành công hoặc thất bại.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-emerald-950/20 border border-emerald-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🌀</span>
                  <div>
                    <span className="font-bold text-emerald-300">Đạo Ngân vô hạn & Cảnh giới tương quan</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Bỏ giới hạn đạo ngân luyện cổ (unlimit), tăng chậm dần theo căn bậc hai và giảm dần theo bậc Cổ đích (chia theo Rank). Cảnh giới Chân Ý nhân trực tiếp vào đạo ngân giúp hóa giải độ khó luyện Cổ bậc cao!</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-purple-950/20 border border-purple-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🎓</span>
                  <div>
                    <span className="font-bold text-purple-300">Cập nhật mốc điểm Cảnh giới</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Đồng bộ lại điểm Chân Ý: Đại Sư (100), Tông Sư (1.000 - khắc chế Tiên Cổ R6), Đại Tông Sư (10.000 - R7), Vô Thượng (100.000 - R8).</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-cyan-950/20 border border-cyan-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🛡️</span>
                  <div>
                    <span className="font-bold text-cyan-300">Scale Chân Ý Luyện Đạo & Giới hạn Bậc</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Luyện Cổ chỉ tăng Chân Ý Luyện Đạo (scale theo cấp Cổ: R1 +2đ, R2-3 +5~15đ, R4-5 +40~100đ, Tiên Cổ +500~50.000đ). Cổ R1 chỉ cho điểm dưới 1.000đ, R2-3 dưới 10.000đ, R4+ trên 10.000đ.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-rose-950/20 border border-rose-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">📖</span>
                  <div>
                    <span className="font-bold text-rose-300">Đọc kỹ Wiki thường xuyên</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Cơ chế Luyện Cổ và thăng thiên vô cùng phức tạp, khuyến cáo đạo hữu hãy thường xuyên tra cứu Wiki để thấu suốt thiên cơ!</p>
                  </div>
                </div>
              </div>
              <p className="text-[var(--text-secondary)]">Phiên bản <strong className="text-amber-600 dark:text-amber-400">v0.8.1</strong> — Sủng vật & Hồn Phách:</p>
              <div className="space-y-2">
                <div className="flex gap-2.5 items-start bg-purple-950/20 border border-purple-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🐾</span>
                  <div>
                    <span className="font-bold text-purple-300">Gộp bầy giới hạn theo Hồn Phách</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Bầy sủng vật tối đa <strong>max_soul/10</strong> cá thể. Hiệu ứng cộng dồn khi gộp đã giảm — chỉ số tăng theo <strong>N^0.65</strong> (sub-linear), không còn bị lũy tiến quá mạnh khi gộp nhiều lần.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-indigo-950/20 border border-indigo-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">👻</span>
                  <div>
                    <span className="font-bold text-indigo-300">Hồn Phách tiêu hao khi ra trận</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Mỗi lần ra trận thật, mỗi cá thể sủng vật tiêu tốn <strong>1 soul_hp</strong>. Hồn Phách không đủ thì sủng vật đó bị bỏ qua, không xuất trận. Võ Đài (đấu ảo) miễn phí Hồn Phách.</p>
                  </div>
                </div>
              </div>
              <p className="text-[var(--text-secondary)]">Phiên bản <strong className="text-amber-600 dark:text-amber-400">v0.8.0</strong> — Tối ưu trải nghiệm & Thăng Tiên:</p>
              <div className="space-y-2">
                <div className="flex gap-2.5 items-start bg-amber-950/20 border border-amber-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🚀</span>
                  <div>
                    <span className="font-bold text-amber-300">Tối ưu lưu trữ</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Kho Cổ trùng giờ lưu trữ gọn gàng hơn: mở kho, trang bị, cho ăn nhanh mượt hơn dù sở hữu nhiều Cổ.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-emerald-950/20 border border-emerald-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🍽️</span>
                  <div>
                    <span className="font-bold text-emerald-300">Cho ăn Cổ trùng ổn định</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Chăm sóc Cổ trùng mượt mà hơn: độ no giảm đều theo thời gian, tránh tình trạng lag khi toàn máy chủ xử lý cùng lúc.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-amber-950/20 border border-amber-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">☯️</span>
                  <div>
                    <span className="font-bold text-amber-300">Đạo Ngân vận hành tốt hơn</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Đạo Ngân hiển thị và hoạt động chính xác, áp dụng hiệu ứng cho cả đồng đội lẫn địch trong chiến đấu tự động.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-fuchsia-950/20 border border-fuchsia-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🐉</span>
                  <div>
                    <span className="font-bold text-fuchsia-300">Thăng Tiên theo Nhân Khí</span>
                    <p className="text-[var(--text-secondary)] mt-0.5">Tỉ lệ thành công Thăng Tiên tính theo Tư chất (Giáp: 20% gốc, Ất/Bính/Đinh: 5% gốc) + Bonus Nhân Khí (+20% mỗi 100k Nhân Khí tích lũy, trần tối đa 500.000, cap 90% thành công). Thành công lên Lục Chuyển Cổ Tiên và khai mở Tiên Khiếu.</p>
                  </div>
                </div>
              </div>
            </div>
             <label className="flex items-center gap-2 mt-4 mb-1 text-xs text-[var(--text-secondary)] cursor-pointer select-none shrink-0">
              <input
                type="checkbox"
                checked={dontShowToday}
                onChange={e => setDontShowToday(e.target.checked)}
                className="rounded border-zinc-600 bg-zinc-800 text-amber-500 focus:ring-amber-500"
              />
              Không hiển thị lại hôm nay
            </label>
            <div className="flex gap-2.5 mt-2 shrink-0">
              <Link
                href="/wiki"
                className="flex-1 py-2 border border-amber-600/30 hover:border-amber-500/50 text-amber-400 hover:text-amber-300 rounded-xl text-xs font-semibold transition-all cursor-pointer text-center flex items-center justify-center gap-1.5"
              >
                📖 Xem Game Wiki
              </Link>
              <button onClick={handleCloseChangelog} className="flex-1 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-xl text-xs font-semibold transition-all cursor-pointer">
                Đã hiểu!
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Inspect Player Profile Modal */}
      {inspectedProfile && <PlayerProfileModal />}

      {/* Scouting & Divination Modals */}
      <DivinationModal />
      <TrackingRadarDrawer />

      {/* Direct Trade Overlays */}
      {activeTrade && activeTrade.status === "PENDING" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-[var(--bg-primary)] animate-fade-in">
          <div className="w-full max-w-md bg-[var(--bg-surface)] border border-[var(--border-default)] rounded-3xl p-6 shadow-2xl relative overflow-hidden text-center flex flex-col gap-5 animate-scale-up">
            <div className="absolute top-0 right-0 w-24 h-24 bg-emerald-500/5 rounded-full blur-xl pointer-events-none" />

            <div className="w-12 h-12 rounded-full bg-emerald-950/50 border border-emerald-500/20 flex items-center justify-center mx-auto">
              <Handshake className="w-6 h-6 text-emerald-400 animate-pulse" />
            </div>

            <div className="space-y-2">
              <h3 className="text-lg font-serif font-bold text-[var(--text-primary)]">
                {player?.username === activeTrade.receiver_username ? "Yêu Cầu Giao Dịch" : "Đang Kết Nối"}
              </h3>
              <p className="text-xs text-[var(--text-secondary)] font-serif leading-relaxed">
                {player?.username === activeTrade.receiver_username ? (
                  <>
                    Cổ sư <span className="text-emerald-400 font-bold">{activeTrade.sender_username}</span> đang phát tín hiệu Thông Thiên Cổ muốn giao dịch trực tiếp với đạo hữu.
                  </>
                ) : (
                  <>
                    Đang thiết lập tần số giao dịch với cổ sư <span className="text-emerald-400 font-bold">{activeTrade.receiver_username}</span> qua Thông Thiên Cổ...
                  </>
                )}
              </p>
            </div>

            <div className="flex gap-3 justify-center">
              {player?.username === activeTrade.receiver_username ? (
                <>
                  <button
                    onClick={() => declineTrade(activeTrade.id)}
                    className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-zinc-900 hover:bg-red-950/20 text-zinc-400 hover:text-red-400 border border-zinc-800 hover:border-red-900/10 transition-colors cursor-pointer"
                  >
                    Từ Chối
                  </button>
                  <button
                    onClick={() => acceptTrade(activeTrade.id)}
                    className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors cursor-pointer shadow-lg shadow-emerald-950/20 border border-emerald-500/25"
                  >
                    Chấp Nhận
                  </button>
                </>
              ) : (
                <button
                  onClick={() => cancelTrade(activeTrade.id)}
                  className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-zinc-900 hover:bg-red-950/20 text-zinc-400 hover:text-red-400 border border-zinc-800 hover:border-red-900/10 transition-colors cursor-pointer"
                >
                  Hủy Yêu Cầu
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {activeTrade && activeTrade.status === "ACCEPTED" && <TradeRoomModal />}

      {/* Dead Player Overlay — cõi chết is derived from lifespan on the server */}
      {player && player.lifespan <= 0 && (
        <div className="fixed inset-0 z-[9999] bg-black/90 backdrop-blur-md flex items-center justify-center p-4">
          <div className="bg-zinc-950 border border-red-900/50 rounded-2xl p-6 max-w-md w-full text-center space-y-4 shadow-2xl shadow-red-950/50">
            <div className="w-16 h-16 bg-red-950/50 border border-red-800/40 rounded-full flex items-center justify-center mx-auto text-red-500 text-3xl font-extrabold animate-pulse">
              ☠️
            </div>
            <h2 className="text-xl font-serif font-extrabold text-red-500 tracking-wide">
              Thọ Nguyên Đã Cạn!
            </h2>
            <p className="text-xs text-zinc-400 font-serif leading-relaxed">
              Tuổi thọ của cổ sư <span className="text-red-400 font-bold">{player.username}</span> đã giảm về 0. Thể xác và thần hồn của đạo hữu đã rơi vào cõi chết.
            </p>
            <div className="p-3 bg-red-950/20 border border-red-900/20 rounded-xl text-[11px] text-red-300 font-mono">
              Mọi hoạt động tu luyện, giao tranh và đi săn đã bị phong ấn. Hãy tìm kiếm sự hỗ trợ hoặc phương pháp tái sinh để hồi sinh bản thể!
            </div>
          </div>
        </div>
      )}

      {/* Global Floating Chat Box */}
      {player && !isGameWorldSurface && <GlobalChatOverlay />}
    </div>
  );
}
