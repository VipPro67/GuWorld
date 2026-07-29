"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Registration } from "@/components/Registration";
import { OnboardingModal } from "@/components/OnboardingModal";
import { Aperture } from "@/components/Aperture";
import { LogConsole } from "@/components/LogConsole";
import { GuInventory } from "@/components/GuInventory";
import { RefiningFurnace } from "@/components/RefiningFurnace";
import { ExplorationMap } from "@/components/ExplorationMap";
import { CombatScreen } from "@/components/CombatScreen";
import { CaravanShop } from "@/components/CaravanShop";
import { PremiumStoreView } from "@/components/PremiumStoreView";
import { Sparkles, HelpCircle, X } from "lucide-react";
import Link from "next/link";
import { BaoHoangMarket } from "@/components/BaoHoangMarket";
import { ArenaView } from "@/components/ArenaView";
import { MineView } from "@/components/MineView";
import { GachaPanel } from "@/components/GachaPanel";
import AchievementsPanel from "@/components/AchievementsPanel";
import { LobbyView } from "@/components/LobbyView";
import { GlobalChatOverlay } from "@/components/GlobalChatOverlay";
import { ToastContainer } from "@/components/ToastContainer";
import { ConfirmDialog } from "@/components/ConfirmDialog";
import { ImmortalApertureView } from "@/components/ImmortalApertureView";
import { DungeonView } from "@/components/DungeonView";
import { SectView } from "@/components/SectView";
import { WikiView } from "@/components/WikiView";
import { CombatHistoryView } from "@/components/CombatHistoryView";
import { PlayerProfileModal } from "@/components/PlayerProfileModal";
import { FeedbackView } from "@/components/FeedbackView";
import { TradeRoomModal } from "@/components/TradeRoomModal";
import { Handshake } from "lucide-react";
import { KillerMoveBuilder } from "@/components/killer-moves/KillerMoveBuilder";
import { AuditLogsView } from "@/components/AuditLogsView";
import { AdminView } from "@/components/AdminView";
import { LeaderboardView } from "@/components/LeaderboardView";
import { ThemeToggle } from "@/components/ThemeToggle";
export default function Home() {
  const {
    player,
    isLoading,
    actionLoading,
    error,
    login,
    register,
    sendOTP,
    completeOnboarding,
    fetchPlayerStatus,
    initializeSession,
    logout,
    explore,
    feedGu,
    feedGuGroup,
    clearError,
    guDictionary,
    itemDictionary,
    fetchDictionary,
    recipes,
    fetchRecipes,
    fetchQuests,
    refineGu,
    exploreNode,
    retreat,
    fleeCombat,
    buyShopItem,
    sellShopItem,
    equipGu,
    updateGuStatus,
    useItem,
    setVitalGu,
    useGu,
    dropGu,
    executeCombatTurn,
    currentView,
    setView,
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
    showChangelog,
    setShowChangelog,
  } = useGameStore();

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
    fetchRecipes();
    fetchQuests();
    initializeSession();
  }, [initializeSession, fetchDictionary, fetchRecipes, fetchQuests]);

  // 2. Sparse authoritative status sync. Mana is rendered from a local snapshot between server actions.
  useEffect(() => {
    if (!player || player.active_combat) return;

    const interval = setInterval(() => {
      fetchPlayerStatus(player.id);
    }, 90000);

    return () => clearInterval(interval);
  }, [player, fetchPlayerStatus]);

  // 3. Trade realtime subscription. This replaces the old /api/trade/active polling loop.
  const isPlayerReady = !!player;
  useEffect(() => {
    if (!isPlayerReady) return;

    subscribeTradeEvents();
    return () => unsubscribeTradeEvents();
    // Only subscribe once when player becomes available, not on every player state change
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isPlayerReady]);

  const handleLogout = () => {
    logout();
  };

  // Force combat view if player is actively in combat
  const inCombat = player && player.active_combat;

  return (
    <div className="min-h-screen flex flex-col py-2 sm:py-8 px-2 sm:px-6 lg:px-8 relative">
      {/* Top Header Navigation */}
      <header className="max-w-7xl w-full mx-auto flex justify-between items-center pb-4 border-b border-zinc-900/50 mb-4">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-emerald-950/60 border border-emerald-500/20 flex items-center justify-center shrink-0">
              <Sparkles className="w-4 h-4 text-emerald-400" />
            </div>
            <h1 className="text-lg sm:text-xl font-bold font-serif tracking-wider text-zinc-100 bg-gradient-to-r from-zinc-100 to-zinc-400 bg-clip-text text-transparent">
              Cổ Giới
            </h1>
          </div>
        </div>
        {player && !inCombat && currentView !== "dashboard" && (
            <button
              onClick={() => setView("dashboard")}
              className="px-3 py-1.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 text-xs text-zinc-300 hover:text-white font-serif transition-colors border border-zinc-800 cursor-pointer flex items-center gap-1"
            >
              <span>⬅</span>
              <span>Trở về Sảnh</span>
            </button>
          )}

        <div className="flex items-center gap-2">
          {player && !inCombat && currentView !== "wiki" && (
            <Link
              href="/wiki"
              target="_blank"
              rel="noopener noreferrer"
              className="px-3 py-1.5 rounded-xl bg-emerald-950/40 hover:bg-emerald-900/60 text-xs text-emerald-400 hover:text-emerald-300 font-serif transition-colors border border-emerald-800/40 inline-flex items-center gap-1"
            >
              <HelpCircle className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
              <span>Wiki Game</span>
            </Link>
          )}

          {player && !inCombat && (
            <button
              onClick={handleLogout}
              className="px-3 py-1.5 rounded-xl bg-red-900/40 hover:bg-red-900/60 text-xs text-red-400 hover:text-red-300 font-serif transition-colors border border-red-800/40 cursor-pointer"
            >
              Đăng Xuất
            </button>
          )}
          <ThemeToggle />
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-grow flex items-start justify-center w-full max-w-6xl mx-auto">
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
        ) : player.needs_onboarding ? (
          <div className="py-12 w-full flex justify-center">
            <OnboardingModal onComplete={completeOnboarding} />
          </div>
        ) : inCombat ? (
          <div className="w-full">
            <CombatScreen
              player={player}
              enemy={player.active_combat!}
              guDictionary={guDictionary}
              actionLoading={actionLoading}
              onRetreat={fleeCombat}
            />
          </div>
        ) : currentView === "dashboard" ? (
          <div className="w-full">
            <LobbyView />
          </div>
        ) : currentView === "character" ? (
          <div className="w-full">
              <Aperture
                player={player}
                guDictionary={guDictionary}
              />
          </div>
        ) : currentView === "inventory" ? (
          <div className="w-full">
            <GuInventory
              player={player}
              onFeed={feedGu}
              onFeedGroup={feedGuGroup}
              actionLoading={actionLoading}
              guDictionary={guDictionary}
              itemDictionary={itemDictionary}
              onEquipToggle={equipGu}
              onUseItem={useItem}
              onUpdateStatus={updateGuStatus}
              onSetVital={setVitalGu}
              onUseGu={useGu}
              onDropGu={dropGu}
            />
          </div>
        ) : currentView === "explore" ? (
          <div className="w-full">
            <ExplorationMap
              player={player}
              actionLoading={actionLoading}
              onExplore={exploreNode}
              onBack={() => setView("dashboard")}
            />
          </div>
        ) : currentView === "refine" ? (
          <div className="w-full">
            <RefiningFurnace
              player={player}
              guDictionary={guDictionary}
              itemDictionary={itemDictionary}
              recipes={recipes}
              actionLoading={actionLoading}
              onRefine={refineGu}
            />
          </div>
        ) : currentView === "kill-moves" ? (
          <KillerMoveBuilder/>
        ) 
        : currentView === "topup" ? (
          <div className="w-full">
            <PremiumStoreView defaultTab="topup" />
          </div>
        ) : (currentView === "premium-shop" || currentView === "premium-store") ? (
          <div className="w-full">
            <PremiumStoreView defaultTab="packages" />
          </div>
        ) : currentView === "shop" ? (
          <div className="w-full">
            <CaravanShop
              player={player}
              actionLoading={actionLoading}
              onBuy={buyShopItem}
              onSell={sellShopItem}
            />
          </div>
        ) : currentView === "auction" ? (
          <BaoHoangMarket
            player={player}
            guInventory={player.gu_inventory || []}
            materials={player.materials_inventory || {}}
            actionLoading={actionLoading}
            listings={marketplaceListings as any[]}
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
        ) : currentView === "leaderboard" ? (
          <div className="w-full">
            <LeaderboardView />
          </div>
        ) : currentView === "gacha" ? (
          <div className="w-full max-w-4xl mx-auto">
            <GachaPanel />
          </div>
        ) : currentView === "achievements" ? (
          <div className="w-full max-w-4xl mx-auto">
            <AchievementsPanel />
          </div>
        ) : currentView === "mine" ? (
          <MineView />
        ) : currentView === "aperture" ? (
          <div className="w-full">
            <ImmortalApertureView />
          </div>
        ) : currentView === "dungeon" ? (
          <div className="w-full">
            <DungeonView />
          </div>
        ) : currentView === "sect" ? (
          <div className="w-full">
            <SectView />
          </div>
        ) : currentView === "wiki" ? (
          <div className="w-full">
            <WikiView />
          </div>
        ) : currentView === "combat-history" ? (
          <div className="w-full">
            <CombatHistoryView />
          </div>
        ) : currentView === "feedback" ? (
          <div className="w-full">
            <FeedbackView />
          </div>
        ) : currentView === "admin" ? (
          <div className="w-full">
            <AdminView />
          </div>
        ) : currentView === "audit-logs" ? (
          <div className="w-full">
            <AuditLogsView />
          </div>
        ) : (
          <div className="text-center py-12 text-zinc-500 font-mono">Giao diện không khả dụng</div>
        )}
      </main>

      {/* Footer Details */}
      <footer className="max-w-7xl w-full mx-auto text-center mt-8 pt-4 border-t border-zinc-950 flex flex-col items-center gap-2">
        <span className="px-2 py-0.5 text-[9px] sm:text-xs font-semibold font-mono tracking-wide rounded bg-amber-500/10 text-amber-500 border border-amber-500/20 uppercase">
          Đang ở Bản Beta sẽ reset dữ liệu khi ra mắt chính thức
        </span>
        <a href="https://www.facebook.com/profile.php?id=61587628762458" target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-semibold text-blue-400 hover:text-blue-300 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/20 transition-all">
          <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 24 24"><path d="M24 12.073c0-6.627-5.373-12-12-12s-12 5.373-12 12c0 5.99 4.388 10.954 10.125 11.854v-8.385H7.078v-3.47h3.047V9.43c0-3.007 1.792-4.669 4.533-4.669 1.312 0 2.686.235 2.686.235v2.953H15.83c-1.491 0-1.956.925-1.956 1.874v2.25h3.328l-.532 3.47h-2.796v8.385C19.612 23.027 24 18.062 24 12.073z"/></svg>
          Fanpage
        </a>
        <p className="text-xs text-zinc-600 font-mono">Cổ Chân Nhân Web Game © 2026 • Thiết Kế Trực Quan Sinh Động</p>
      </footer>

      {/* Toast Notification Container */}
      <ToastContainer />
      <ConfirmDialog />

      {/* Changelog Modal */}
      {player && showChangelog && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm animate-fade-in" onClick={handleCloseChangelog}>
          <div className="w-full max-w-lg bg-gradient-to-br from-zinc-950 to-zinc-900 border border-zinc-800 rounded-3xl p-6 shadow-2xl overflow-hidden animate-scale-up" onClick={e => e.stopPropagation()}>
            <div className="absolute top-0 right-0 w-32 h-32 bg-amber-500/5 rounded-full blur-xl pointer-events-none" />
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                <h3 className="text-base font-serif font-bold text-zinc-100">Cập Nhật Mới!</h3>
              </div>
              <button onClick={handleCloseChangelog} className="text-zinc-500 hover:text-zinc-300 cursor-pointer"><X className="w-4 h-4" /></button>
            </div>
            <div className="space-y-3 text-xs text-zinc-300 leading-relaxed">
              <p className="text-zinc-400">Phiên bản <strong className="text-amber-400">v0.7.2</strong> — Sửa lỗi chiến đấu & Gift Code:</p>
              <div className="space-y-2">
                <div className="flex gap-2.5 items-start bg-amber-950/20 border border-amber-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">⚔️</span>
                  <div>
                    <span className="font-bold text-amber-300">Sửa chiến đấu</span>
                    <p className="text-zinc-400 mt-0.5">Tốc độ giờ ảnh hưởng thứ tự đánh. MaxHP không còn bị kẹt ở 100. Passive + Growth auto-apply từ toàn bộ Cổ đang trang bị.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-emerald-950/20 border border-emerald-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🎯</span>
                  <div>
                    <span className="font-bold text-emerald-300">Auto-Combat chỉ hiện Cổ có Active Skill</span>
                    <p className="text-zinc-400 mt-0.5">Bộ lọc giờ chỉ hiện Cổ có hiệu ứng chủ động (tấn công, hồi máu, khiên...).</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-amber-950/20 border border-amber-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">📈</span>
                  <div>
                    <span className="font-bold text-amber-300">Trưởng Thành </span>
                    <p className="text-zinc-400 mt-0.5">Sửa lỗi Cổ cũ không có thông tin phát triển.</p>
                  </div>
                </div>
                <div className="flex gap-2.5 items-start bg-fuchsia-950/20 border border-fuchsia-500/10 rounded-lg p-3">
                  <span className="text-lg shrink-0 mt-0.5">🎁</span>
                  <div>
                    <span className="font-bold text-fuchsia-300">Gift Code mới!</span>
                    <p className="text-zinc-400 mt-0.5">Nhập <strong className="text-fuchsia-300">CHODAUGIA</strong>, <strong className="text-fuchsia-300">PHUONGNGUYEN</strong>, <strong className="text-fuchsia-300">BACHNGUNGBANG</strong> tại tab Phúc Lợi để nhận quà!</p>
                  </div>
                </div>
              </div>
            </div>
            <label className="flex items-center gap-2 mt-4 mb-1 text-xs text-zinc-400 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={dontShowToday}
                onChange={e => setDontShowToday(e.target.checked)}
                className="rounded border-zinc-600 bg-zinc-800 text-amber-500 focus:ring-amber-500"
              />
              Không hiển thị lại hôm nay
            </label>
            <button onClick={handleCloseChangelog} className="w-full mt-2 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-xl text-xs font-semibold transition-all cursor-pointer">
              Đã hiểu!
            </button>
          </div>
        </div>
      )}

      {/* Inspect Player Profile Modal */}
      {inspectedProfile && <PlayerProfileModal />}

      {/* Direct Trade Overlays */}
      {activeTrade && activeTrade.status === "PENDING" && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/80 backdrop-blur-sm animate-fade-in">
          <div className="w-full max-w-md bg-zinc-950 border border-zinc-900 rounded-3xl p-6 shadow-2xl relative overflow-hidden text-center flex flex-col gap-5 animate-scale-up">
            <div className="absolute top-0 right-0 w-24 h-24 bg-emerald-500/5 rounded-full blur-xl pointer-events-none" />
            
            <div className="w-12 h-12 rounded-full bg-emerald-950/50 border border-emerald-500/20 flex items-center justify-center mx-auto">
              <Handshake className="w-6 h-6 text-emerald-400 animate-pulse" />
            </div>

            <div className="space-y-2">
              <h3 className="text-lg font-serif font-bold text-zinc-100">
                {player?.username === activeTrade.receiver_username ? "Yêu Cầu Giao Dịch" : "Đang Kết Nối"}
              </h3>
              <p className="text-xs text-zinc-400 font-serif leading-relaxed">
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

      {/* Global Floating Chat Box */}
      {player && <GlobalChatOverlay />}
    </div>
  );
}
