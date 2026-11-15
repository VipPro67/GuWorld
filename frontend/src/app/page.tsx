"use client";

import { useEffect, useState, useSyncExternalStore } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { useShallow } from "zustand/react/shallow";
import { useGameStore } from "@/store/useGameStore";
import { useUIStore } from "@/store/useUIStore";
import { Registration } from "@/components/Registration";
import { OnboardingModal } from "@/components/OnboardingModal";
import { LevelZeroView } from "@/components/LevelZeroView";
import { Sparkles, X, Handshake, Store, HelpCircle, LogOut, Compass, Gem, Swords } from "@/lib/icons";
import { GlobalChatOverlay } from "@/components/GlobalChatOverlay";
import { NotificationBell } from "@/components/NotificationBell";
import { RespawnChoiceModal } from "@/components/RespawnChoiceModal";
import { MaintenanceView } from "@/components/MaintenanceView";
import { PlayerProfileModal } from "@/components/PlayerProfileModal";
import { DivinationModal } from "@/components/DivinationModal";
import { TrackingRadarDrawer } from "@/components/TrackingRadarDrawer";
import { TradeRoomModal } from "@/components/TradeRoomModal";
import { GameLoadingScreen } from "@/components/GameLoadingScreen";
import { ClearCacheButton } from "@/components/ClearCacheModal";
import { installPlayerLifecycleReconciliation } from "@/lib/player-reconciliation";
import { updateUiPreferences } from "@/lib/ui-preferences";
import { uiRealtimeManager } from "@/lib/realtime/UIRealtimeManager";
import { installHttpFallbackPolling } from "@/lib/http-fallback";

const subscribeConnection = (notify: () => void) => uiRealtimeManager.onConnectionChange(notify);
const getConnection = () => uiRealtimeManager.getConnectionState();
const getServerConnection = () => "disconnected" as const;

const Aperture = dynamic(() => import("@/components/Aperture").then(m => m.Aperture), { ssr: false });
const GuInventory = dynamic(() => import("@/components/GuInventory").then(m => m.GuInventory), { ssr: false });
const RefiningFurnace = dynamic(() => import("@/components/RefiningFurnace").then(m => m.RefiningFurnace), { ssr: false });
const RefineSessionPanel = dynamic(() => import("@/components/RefineSessionPanel").then(m => m.RefineSessionPanel), { ssr: false });
const CaravanShop = dynamic(() => import("@/components/CaravanShop").then(m => m.CaravanShop), { ssr: false });
const MerchantTradeView = dynamic(() => import("@/components/MerchantTradeView").then(m => m.MerchantTradeView), { ssr: false });
const PremiumStoreView = dynamic(() => import("@/components/PremiumStoreView").then(m => m.PremiumStoreView), { ssr: false });
const BaoHoangMarket = dynamic(() => import("@/components/BaoHoangMarket").then(m => m.BaoHoangMarket), { ssr: false });
const ArenaView = dynamic(() => import("@/components/ArenaView").then(m => m.ArenaView), { ssr: false });
const InfiniteTowerView = dynamic(() => import("@/components/InfiniteTowerView").then(m => m.InfiniteTowerView), { ssr: false });
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
const CombatScreen = dynamic(() => import("@/components/CombatScreen").then(m => m.CombatScreen), { ssr: false });
const FeedbackView = dynamic(() => import("@/components/FeedbackView").then(m => m.FeedbackView), { ssr: false });
const AuditLogsView = dynamic(() => import("@/components/AuditLogsView").then(m => m.AuditLogsView), { ssr: false });
const LeaderboardView = dynamic(() => import("@/components/LeaderboardView").then(m => m.LeaderboardView), { ssr: false });
const TrainingRoom = dynamic(() => import("@/components/TrainingRoom").then(m => m.TrainingRoom), { ssr: false });
const UtilitiesView = dynamic(() => import("@/components/UtilitiesView").then(m => m.UtilitiesView), { ssr: false });
const ExplorationMap = dynamic(() => import("@/components/ExplorationMap").then(m => m.ExplorationMap), { ssr: false });
const LobbyView = dynamic(() => import("@/components/LobbyView").then(m => m.LobbyView), { ssr: false });

export default function Home() {
  const router = useRouter();
  const [worldConnected, setWorldConnected] = useState(false);
  const connection = useSyncExternalStore(subscribeConnection, getConnection, getServerConnection);
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
      state.dictionaryLoaded,
      state.dictionaryError,
    ])
  );

  const unequipAllGu = useGameStore((s) => s.unequipAllGu);
  const logout = useGameStore((s) => s.logout);
  const merchantAtVillage = useGameStore((s) => s.merchantAtVillage);
  const nearbyMerchants = useGameStore((s) => s.merchants);
  const fetchMerchants = useGameStore((s) => s.fetchMerchants);
  const activeCombatReplay = useGameStore((s) => s.activeCombatReplay);
  const fleeCombat = useGameStore((s) => s.fleeCombat);

  const currentView = useUIStore((s) => s.currentView);
  const isExploreView = ["explore", "mmo-world"].includes(currentView);
  const setView = useUIStore((s) => s.setView);
  const showChangelog = useUIStore((s) => s.showChangelog);
  const setShowChangelog = useUIStore((s) => s.setShowChangelog);

  useEffect(() => {
    if (currentView === "wiki") router.replace("/wiki");
  }, [currentView, router]);

  // Vô tab Luyện Cổ khi đang có phiên luyện → tự mở panel phiên (WF-REF-07).
  const hasActiveRefineView = useGameStore((s) => !!s.refineSession && s.refineSession.status === "ACTIVE");
  const openRefinePanel = useGameStore((s) => s.openRefinePanel);
  useEffect(() => {
    if (currentView === "refine" && hasActiveRefineView) {
      openRefinePanel();
    }
  }, [currentView, hasActiveRefineView, openRefinePanel]);

  const [dontShowToday, setDontShowToday] = useState(false);

  const handleCloseChangelog = () => {
    if (dontShowToday) {
      const today = new Date().toISOString().split('T')[0];
      updateUiPreferences({ changelogHideUntil: today });
    }
    setDontShowToday(false);
    setShowChangelog(false);
  };

  // 1. Session Restoration on Mount
  useEffect(() => {
    void fetchDictionary();
    void initializeSession();
  }, [initializeSession, fetchDictionary]);

  const playerId = player?.id;
  useEffect(() => { setWorldConnected(false); }, [playerId]);
  useEffect(() => {
    if (!playerId) return;
    return installPlayerLifecycleReconciliation(window, document, () => void reconcilePlayer(true));
  }, [playerId, reconcilePlayer]);

  // 2. Authoritative status reconciliation is event-driven (`player` channel)
  // with an HTTP fallback poller while the realtime session is down, so no
  // timer is needed while WebTransport is connected.

  // Push outages must not strand business screens or incoming trade updates.
  useEffect(() => {
    if (!playerId) return;
    let nextPlayerRefresh = 0;
    return installHttpFallbackPolling({
      connection: uiRealtimeManager,
      activityTarget: window,
      visibilityTarget: document,
      isOnline: () => navigator.onLine !== false,
      refresh: async () => {
        const state = useGameStore.getState();
        if (state.player?.id !== playerId || !state.token) return;
        const tasks: Promise<unknown>[] = [state.fetchActiveTrade()];
        if (Date.now() >= nextPlayerRefresh) {
          nextPlayerRefresh = Date.now() + 15000;
          tasks.push(state.reconcilePlayer(true), state.fetchUnreadNotificationsCount());
        }
        if (state.chatOpen) {
          if (state.activeChannel === "private") {
            tasks.push(state.fetchPrivateConversations());
            if (state.activePrivateUsername) tasks.push(state.fetchPrivateThread(state.activePrivateUsername));
          } else {
            tasks.push(state.fetchBaoHoangMessages(state.activeChannel));
          }
        }
        await Promise.allSettled(tasks);
      },
    });
  }, [playerId]);

  // 3. Trade realtime subscription. This replaces the old /api/trade/active polling loop.
  const isPlayerReady = !!player;
  useEffect(() => {
    if (!isPlayerReady) return;

    subscribeTradeEvents();
    return () => unsubscribeTradeEvents();
    // Only subscribe once when player becomes available, not on every player state change

  }, [isPlayerReady]);

  // Load merchant proximity when entering/changing world region, or when viewing shop.
  // The merchant panel refreshes its own list while open; polling here duplicated requests
  // for players standing idle on the world page.
  useEffect(() => {
    if (!playerId) return;
    if (isExploreView || currentView === "shop") {
      void fetchMerchants();
    }
  }, [playerId, player?.current_region, isExploreView, currentView, fetchMerchants]);

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

        <a href="#main-content" className="skip-link">Bỏ qua đến nội dung chính</a>
        <main id="main-content" tabIndex={-1} className="flex-grow flex items-center justify-center w-full max-w-[1600px] px-2 sm:px-8 mx-auto">
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

  const isFullBleedView = currentView === "practice-room" || isExploreView;
  const isWideFeatureView = ["aperture", "sect", "shop", "auction", "premium-store", "premium-shop", "topup", "arena", "infinite-tower", "death-arena", "martial-arena"].includes(currentView);

  const viewTitles: Record<string, string> = {
    dashboard: "Sảnh chính",
    explore: "Khám phá dã ngoại",
    "practice-room": "Phòng tập",
    cultivation: "Tu luyện",
    character: "Nhân vật",
    inventory: "Túi đồ",
    "inventory-gu": "Cổ trùng",
    party: "Tổ đội",
    refine: "Luyện Cổ",
    topup: "Nạp",
    "premium-shop": "Cửa hàng tiên ngọc",
    "premium-store": "Cửa hàng",
    shop: "Thương hội",
    auction: "Chợ Tiên",
    arena: "Đấu trường",
    "infinite-tower": "Tháp Chiến Thuật",
    "death-arena": "Sinh tử đài",
    "martial-arena": "Đại Hội Võ Thuật",
    leaderboard: "Bảng xếp hạng",
    gacha: "Tiên duyên",
    achievements: "Thành tựu",
    pets: "Dị nhân",
    aperture: "Tiên khiếu",
    "dream-realm": "Mộng Cảnh",
    dungeon: "Bí cảnh & Truyền thừa",
    sect: "Tông môn",
    "combat-history": "Lịch sử chiến đấu",
    feedback: "Góp ý",
    "audit-logs": "Nhật ký hệ thống",
    "mmo-world": "Khám phá dã ngoại",
    utilities: "Tiện ích",
  };

  const viewArtwork: Record<string, string> = {
    character: "/assets/world/form-reckless-savage.webp",
    inventory: "/assets/world/spirit-beast.webp",
    "inventory-gu": "/assets/world/spirit-beast.webp",
    refine: "/assets/world/spirit-herb.webp",
    cultivation: "/assets/world/spirit-herb.webp",
    aperture: "/assets/world/ground-moss.webp",
    "premium-store": "/assets/items/icons/chest.webp",
    "premium-shop": "/assets/items/icons/crystal.webp",
    shop: "/assets/items/icons/gourd.webp",
    auction: "/assets/items/icons/crystal.webp",
    pets: "/assets/world/spirit-beast.webp",
    sect: "/assets/world/cultivator.webp",
    arena: "/assets/world/form-six-arm-ice-demon.webp",
    "infinite-tower": "/assets/world/form-reckless-savage.webp",
    "death-arena": "/assets/world/form-six-arm-ice-demon.webp",
    "martial-arena": "/assets/world/form-reckless-savage.webp",
    achievements: "/assets/items/icons/star.webp",
    utilities: "/assets/items/icons/formation.webp",
    gacha: "/assets/items/icons/star.webp",
    leaderboard: "/assets/items/icons/star.webp",
    "dream-realm": "/assets/world/spirit-beast.webp",
    dungeon: "/assets/world/great-regions-atlas.png",
    feedback: "/assets/items/icons/scroll.webp",
    "audit-logs": "/assets/items/icons/map.webp",
    "combat-history": "/assets/items/icons/blade.webp",
    party: "/assets/world/spirit-beast.webp",
    topup: "/assets/items/icons/coin.webp",
  };

  return (
    <div className="mmo-paper-theme h-[100dvh] min-h-[100dvh] max-h-[100dvh] flex flex-col overflow-hidden p-0 relative">
      <a href="#main-content" className="skip-link">Bỏ qua đến nội dung chính</a>
      {/* Top Header Navigation */}
      <header className="w-full max-w-[1600px] px-2 sm:px-6 mx-auto py-2.5 sm:py-3 border-b border-[var(--border-default)] flex justify-between items-center gap-3 shrink-0 bg-zinc-950/80 backdrop-blur-md sticky top-0 z-30">
        <div className="flex items-center gap-3 sm:gap-6">
          <button
            type="button"
            onClick={() => {
              if (player && currentView !== "dashboard") {
                setView("dashboard");
              }
            }}
            disabled={!player || currentView === "dashboard"}
            className={`flex items-center gap-2 rounded focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-amber-300 ${player && currentView !== "dashboard" ? "cursor-pointer hover:opacity-80 transition-opacity" : "cursor-default"}`}
            title={player && currentView !== "dashboard" ? "Trở về Sảnh Chính" : undefined}
          >
            <div className="w-8 h-8 rounded-lg bg-[var(--brand-dim)] border border-[var(--brand-primary)] flex items-center justify-center shrink-0">
              <Sparkles className="w-4 h-4 text-emerald-400" />
            </div>
            <h1 className="text-lg sm:text-xl font-bold font-serif tracking-wider text-[var(--text-primary)]">
              Cổ Giới
            </h1>
          </button>

        </div>

        <div className="flex items-center gap-2 sm:gap-3">
          {player && (
            <Link
              href="/wiki"
              target="_blank"
              rel="noopener noreferrer"
              className="px-2.5 py-1.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 text-xs text-emerald-400 hover:text-emerald-300 font-serif transition-colors border border-zinc-800 inline-flex items-center gap-1"
            >
              <HelpCircle className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
              <span className="hidden sm:inline">Wiki</span>
            </Link>
          )}

          {player && <NotificationBell />}

          {player && (
            <button
              onClick={logout}
              className="px-2.5 py-1.5 rounded-xl bg-zinc-900 hover:bg-red-950/30 text-xs text-red-400 hover:text-red-300 font-serif transition-colors border border-zinc-800 hover:border-red-900/30 cursor-pointer inline-flex items-center gap-1"
              title="Đăng Xuất"
            >
              <LogOut className="w-3.5 h-3.5" />
              <span className="hidden sm:inline">Đăng Xuất</span>
            </button>
          )}

          <ClearCacheButton />
        </div>
      </header>

      {/* Main Content Area */}
      <main id="main-content" tabIndex={-1} className={`w-full flex-1 min-h-0 max-w-none px-0 mx-0 flex flex-col ${player ? "overflow-hidden" : "overflow-y-auto"}`}>
        {!player ? (
          <div className="relative isolate flex-1 min-h-full w-full overflow-hidden bg-[#101814]">
            <div
              aria-hidden="true"
              className="absolute inset-0 opacity-45"
              style={{
                backgroundImage: "linear-gradient(90deg, rgba(10, 18, 14, 0.3), rgba(10, 18, 14, 0.86)), url('/assets/world/great-regions-atlas.png')",
                backgroundSize: "cover, 700% 100%",
                backgroundPosition: "center, left center",
              }}
            />
            <div aria-hidden="true" className="absolute inset-0 bg-[radial-gradient(ellipse_at_24%_42%,rgba(72,113,80,0.28),transparent_44%),linear-gradient(180deg,rgba(8,13,11,0.16),#101814_100%)]" />

            <div className="relative z-10 mx-auto grid min-h-full w-full max-w-[1440px] grid-cols-1 items-center gap-8 px-4 py-7 sm:px-8 sm:py-10 lg:grid-cols-[minmax(0,1.15fr)_minmax(390px,0.78fr)] lg:gap-10 lg:px-12">
              <section className="relative hidden min-h-[580px] flex-col justify-between overflow-hidden rounded-[2rem] border border-emerald-100/10 bg-[#111a15]/45 p-8 shadow-[0_28px_90px_rgba(0,0,0,0.38)] backdrop-blur-[2px] lg:flex xl:min-h-[640px] xl:p-10">
                <div aria-hidden="true" className="absolute inset-0 bg-gradient-to-br from-emerald-950/25 via-transparent to-black/35" />
                <div aria-hidden="true" className="absolute inset-0 opacity-[0.12] [background-image:linear-gradient(rgba(244,232,203,0.18)_1px,transparent_1px),linear-gradient(90deg,rgba(244,232,203,0.18)_1px,transparent_1px)] [background-size:52px_52px] [mask-image:linear-gradient(90deg,black,transparent_85%)]" />

                <div className="relative z-10 flex items-center gap-3 text-[10px] font-mono uppercase tracking-[0.28em] text-emerald-100/75">
                  <span className="h-px w-8 bg-amber-300/75" />
                  Thiên địa rộng lớn · Đạo hữu tự chọn đường đi
                </div>

                <div className="relative z-10 max-w-[540px] pb-20 pt-10">
                  <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-amber-200/20 bg-black/25 px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.2em] text-amber-100/85">
                    <Sparkles className="h-3.5 w-3.5 text-amber-300" />
                    Web game nhập vai tu tiên
                  </div>
                  <h2 className="font-serif text-5xl font-bold leading-[1.08] tracking-wide text-[#fff3d8] drop-shadow-[0_4px_24px_rgba(0,0,0,0.65)] xl:text-6xl">
                    Khai mở
                    <span className="mt-2 block text-amber-300">Không Khiếu.</span>
                  </h2>
                  <p className="mt-5 max-w-md text-sm leading-7 text-stone-200/80">
                    Bước vào thế giới Cổ Trùng, tự mình tu luyện, thám hiểm và viết nên con đường nghịch thiên của đạo hữu.
                  </p>

                  <div className="mt-8 flex flex-wrap gap-2.5">
                    <span className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-black/25 px-3 py-2 text-xs text-stone-100/90">
                      <Compass className="h-4 w-4 text-emerald-300" /> Khám phá thế giới
                    </span>
                    <span className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-black/25 px-3 py-2 text-xs text-stone-100/90">
                      <Gem className="h-4 w-4 text-sky-300" /> Thu thập Cổ Trùng
                    </span>
                    <span className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-black/25 px-3 py-2 text-xs text-stone-100/90">
                      <Swords className="h-4 w-4 text-rose-300" /> Giao đấu
                    </span>
                  </div>
                </div>

                <Image
                  src="/assets/world/cultivator.webp"
                  alt=""
                  width={494}
                  height={511}
                  priority
                  aria-hidden="true"
                  className="pointer-events-none absolute -bottom-4 right-0 z-[1] w-[48%] max-w-[360px] object-contain drop-shadow-[0_12px_42px_rgba(0,0,0,0.65)] xl:right-4 xl:w-[45%]"
                />

                <div className="relative z-10 grid max-w-[580px] grid-cols-3 gap-3 border-t border-white/10 pt-5">
                  <div>
                    <p className="font-serif text-lg font-bold text-amber-200">Tu luyện</p>
                    <p className="mt-1 text-[10px] uppercase tracking-[0.16em] text-stone-300/60">Từng bước đột phá</p>
                  </div>
                  <div className="border-l border-white/10 pl-4">
                    <p className="font-serif text-lg font-bold text-amber-200">Luyện Cổ</p>
                    <p className="mt-1 text-[10px] uppercase tracking-[0.16em] text-stone-300/60">Tìm kiếm cơ duyên</p>
                  </div>
                  <div className="border-l border-white/10 pl-4">
                    <p className="font-serif text-lg font-bold text-amber-200">Tranh đấu</p>
                    <p className="mt-1 text-[10px] uppercase tracking-[0.16em] text-stone-300/60">Định đoạt thắng bại</p>
                  </div>
                </div>
              </section>

              <div className="flex w-full justify-center lg:justify-end">
                <Registration
                  onRegister={register}
                  onLogin={login}
                  onSendOTP={sendOTP}
                  isLoading={isLoading}
                  error={error}
                  onClearError={clearError}
                />
              </div>
            </div>
          </div>
        ) : player.active_combat || activeCombatReplay ? (
          <div className="w-full flex-1 min-h-0 overflow-y-auto overscroll-contain">
            <CombatScreen
              player={player}
              enemy={player.active_combat || activeCombatReplay!.enemy}
              guDictionary={guDictionary}
              actionLoading={actionLoading}
              isPracticeMode={((player.active_combat || activeCombatReplay?.enemy)?.enemy_type === "PRACTICE")}
              onRetreat={fleeCombat}
            />
          </div>
        ) : player.rank === 0 ? (
          <div className="py-12 w-full flex justify-center overflow-y-auto">
            <LevelZeroView onComplete={completeTutorial} />
          </div>
        ) : player.needs_onboarding ? (
          <div className="py-12 w-full flex justify-center overflow-y-auto">
            <OnboardingModal onComplete={completeOnboarding} />
          </div>
        ) : (
          <div className="relative isolate w-full h-full min-h-0 flex-1 overflow-hidden bg-zinc-950">
            {isExploreView ? (
              <div className="w-full h-full overflow-y-auto py-2 sm:py-4">
                <ExplorationMap player={player} />
              </div>
            ) : (
              <div className="w-full h-full overflow-y-auto py-2 sm:py-4 px-2 sm:px-6">
                <LobbyView />
              </div>
            )}
            {!isExploreView && currentView !== "dashboard" && (
              <div className={`absolute inset-0 z-40 flex items-center justify-center ${isFullBleedView ? "" : "bg-black/55 backdrop-blur-[2px] md:p-4"}`}>
                <section
                  role="dialog"
                  aria-modal="true"
                  aria-label={viewTitles[currentView] || "Chức năng"}
                  className={`mmo-panel-theme mmo-paper-theme mmo-paper-surface relative flex h-full w-full flex-col overflow-hidden text-[var(--text-primary,#f4e8cb)] ${isFullBleedView ? "" : `md:h-[min(90dvh,900px)] md:max-h-[90dvh] md:rounded-2xl ${isWideFeatureView ? "md:w-[min(76vw,1100px)]" : "md:w-[min(64vw,820px)]"}`}`}
                >
                  {!isFullBleedView && (
                  <div className="mmo-paper-heading relative isolate flex min-h-14 shrink-0 items-center justify-between gap-3 overflow-hidden px-4 sm:px-5">
                    {viewArtwork[currentView] && <img src={viewArtwork[currentView]} alt="" aria-hidden="true" loading="lazy" className="pointer-events-none absolute right-12 top-0 h-full w-20 object-contain opacity-25 sm:right-16 sm:w-24" />}
                    <div className="relative z-[1] min-w-0">
                      <div className="truncate font-serif text-base font-semibold text-[var(--text-primary,#f4e8cb)]">{viewTitles[currentView] || "Chức năng"}</div>
                      <div className="hidden text-[10px] uppercase tracking-[.2em] text-[#795a34] sm:block">Cổ Giới · Bảng điều khiển</div>
                    </div>
                    <div className="relative z-[1] flex shrink-0 items-center gap-2">
                      <button
                        onClick={() => setView("dashboard")}
                        aria-label="Đóng bảng điều khiển"
                        className="mmo-bamboo-action grid h-9 w-9 shrink-0 place-items-center cursor-pointer"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                  )}
                  {isExploreView && (
                    <button
                      onClick={() => setView("dashboard")}
                      aria-label="Thoát khám phá"
                      title="Thoát khám phá"
                      className="absolute right-3 top-[calc(env(safe-area-inset-top)+0.75rem)] z-50 grid h-10 w-10 place-items-center rounded-xl border border-white/15 bg-zinc-950/80 text-zinc-100 shadow-lg backdrop-blur-sm transition-colors hover:bg-zinc-800"
                    >
                      <X className="h-5 w-5" />
                    </button>
                  )}
                  <div className={`min-h-0 flex-1 overflow-y-auto overscroll-contain p-2 pb-28 sm:p-4 sm:pb-8 ${isFullBleedView ? "p-0" : ""}`}>
        {currentView === "practice-room" ? (
          <div className="w-full">
            <TrainingRoom onClose={() => setView("dashboard")} />
          </div>
        ) : currentView === "character" || currentView === "cultivation" ? (
          <div className="mmo-paper-aperture w-full">
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
        ) : currentView === "infinite-tower" ? (
          <InfiniteTowerView />
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
              </div>
            )}
          </div>
        )}
      </main>

      {/* Phiên Luyện Cổ chạy nền — HUD nổi toàn cục (WF-REF-07) */}
      {player && <RefineSessionPanel />}

      {/* Footer Details */}
      {!player && (
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
          <div className="w-full max-w-xl max-h-[90vh] sm:max-h-[85vh] flex flex-col bg-[var(--bg-surface)] border border-amber-300/20 rounded-3xl p-4 sm:p-5 shadow-[0_28px_90px_rgba(0,0,0,.65)] overflow-hidden animate-scale-up" onClick={e => e.stopPropagation()}>
            <div className="absolute top-0 right-0 w-32 h-32 bg-amber-500/5 rounded-full blur-xl pointer-events-none" />
            <div className="flex items-center justify-between mb-4 shrink-0">
              <div className="flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                <h3 className="text-base font-serif font-bold text-[var(--text-primary)]">Cập Nhật Mới!</h3>
              </div>
              <button onClick={handleCloseChangelog} className="text-zinc-500 hover:text-zinc-300 cursor-pointer"><X className="w-4 h-4" /></button>
            </div>
            <div className="flex-1 overflow-y-auto space-y-3 pr-1 custom-scrollbar">
              <section className="relative isolate overflow-hidden rounded-2xl border border-amber-300/20 bg-gradient-to-br from-amber-950/70 via-zinc-950/80 to-emerald-950/50 p-4 sm:p-5">
                <img src="/assets/world/form-six-arm-ice-demon.webp" alt="" aria-hidden="true" className="pointer-events-none absolute -right-4 -top-8 h-36 w-40 object-contain opacity-35 sm:h-44 sm:w-48" />
                <div aria-hidden="true" className="pointer-events-none absolute inset-0 bg-gradient-to-r from-zinc-950/10 via-zinc-950/30 to-zinc-950/75" />
                <div className="relative z-[1] max-w-[78%]">
                  <span className="inline-flex items-center gap-1.5 rounded-full border border-amber-300/25 bg-amber-400/10 px-2 py-1 text-[9px] font-mono uppercase tracking-[.18em] text-amber-200"><Sparkles className="h-3 w-3" /> Bản cập nhật mới</span>
                  <h4 className="mt-2 font-serif text-xl font-bold text-amber-100">Ngũ Vực dậy sóng</h4>
                  <p className="mt-1 text-[11px] leading-relaxed text-stone-300/80">PvP, hồi sinh và chiến lợi phẩm được tinh chỉnh.</p>
                </div>
                <span className="absolute bottom-3 right-4 font-mono text-[10px] text-amber-100/50">v0.8.4</span>
              </section>
              <div className="grid gap-2 sm:grid-cols-2">
                <article className="rounded-xl border border-sky-400/20 bg-sky-950/30 p-3">
                  <span className="text-lg">⚔️</span><h4 className="mt-1 text-xs font-bold text-sky-200">Giao chiến theo lượt</h4>
                  <p className="mt-1 text-[10px] leading-relaxed text-slate-300/75">Chạm trán quái khi bước vào ô có quái.</p>
                </article>
                <article className="rounded-xl border border-rose-400/20 bg-rose-950/30 p-3">
                  <span className="text-lg">💀</span><h4 className="mt-1 text-xs font-bold text-rose-200">Tử đạo có giá</h4>
                  <p className="mt-1 text-[10px] leading-relaxed text-slate-300/75">Một phần chiến lợi phẩm bị hủy; phần còn lại rơi tại chỗ.</p>
                </article>
                <article className="rounded-xl border border-orange-400/20 bg-orange-950/30 p-3">
                  <span className="text-lg">🔥</span><h4 className="mt-1 text-xs font-bold text-orange-200">Sát khí & vùng an toàn</h4>
                  <p className="mt-1 text-[10px] leading-relaxed text-slate-300/75">Sát khí cao ảnh hưởng bảo hộ và nguy cơ rơi Cổ khi bại trận.</p>
                </article>
                <article className="rounded-xl border border-emerald-400/20 bg-emerald-950/30 p-3">
                  <span className="text-lg">🛡️</span><h4 className="mt-1 text-xs font-bold text-emerald-200">Hồi sinh & miễn chiến</h4>
                  <p className="mt-1 text-[10px] leading-relaxed text-slate-300/75">Hồi sinh gần khu an toàn; nạn nhân PvP được miễn chiến 10 phút.</p>
                </article>
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

      <RespawnChoiceModal />

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
      {player && !isExploreView && <GlobalChatOverlay />}
    </div>
  );
}
