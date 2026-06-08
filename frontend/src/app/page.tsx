"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Registration } from "@/components/Registration";
import { Aperture } from "@/components/Aperture";
import { LogConsole } from "@/components/LogConsole";
import { GuInventory } from "@/components/GuInventory";
import { RefiningFurnace } from "@/components/RefiningFurnace";
import { ExplorationMap } from "@/components/ExplorationMap";
import { CombatScreen } from "@/components/CombatScreen";
import { CaravanShop } from "@/components/CaravanShop";
import { Sparkles, HelpCircle } from "lucide-react";
import { BaoHoangMarket } from "@/components/BaoHoangMarket";
import { ArenaView } from "@/components/ArenaView";
import { MineView } from "@/components/MineView";
import { LobbyView } from "@/components/LobbyView";
import { ToastContainer } from "@/components/ToastContainer";
import { ImmortalApertureView } from "@/components/ImmortalApertureView";
import { DungeonView } from "@/components/DungeonView";
import { SectView } from "@/components/SectView";
import { WikiView } from "@/components/WikiView";
import { PlayerProfileModal } from "@/components/PlayerProfileModal";
import { FeedbackView } from "@/components/FeedbackView";
import { TradeRoomModal } from "@/components/TradeRoomModal";
import { Handshake } from "lucide-react";




export default function Home() {
  const {
    player,
    isLoading,
    actionLoading,
    error,
    login,
    register,
    fetchPlayerStatus,
    initializeSession,
    logout,
    explore,
    feedGu,
    feedGuGroup,
    clearError,
    guDictionary,
    itemDictionary,
    dictionaryLoaded,
    fetchDictionary,
    recipes,
    recipesLoaded,
    fetchRecipes,
    refineGu,
    exploreNode,
    retreat,
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
  } = useGameStore();

  const [checkingSession, setCheckingSession] = useState(true);

  // 1. Session Restoration on Mount
  useEffect(() => {
    fetchDictionary();
    fetchRecipes();
    initializeSession().finally(() => {
      setCheckingSession(false);
    });
  }, [initializeSession, fetchDictionary, fetchRecipes]);

  // 2. Sparse authoritative status sync. Mana is rendered from a local snapshot between server actions.
  useEffect(() => {
    if (!player || player.active_combat) return;

    const interval = setInterval(() => {
      fetchPlayerStatus(player.id);
    }, 90000);

    return () => clearInterval(interval);
  }, [player, fetchPlayerStatus]);

  // 3. Trade realtime subscription. This replaces the old /api/trade/active polling loop.
  useEffect(() => {
    if (!player) return;

    subscribeTradeEvents();
    return () => unsubscribeTradeEvents();
  }, [player, subscribeTradeEvents, unsubscribeTradeEvents]);

  const handleLogout = () => {
    logout();
  };

  if (checkingSession || !dictionaryLoaded || !recipesLoaded) {
    return (
      <div className="min-h-screen flex items-center justify-center flex-col gap-4">
        <div className="w-12 h-12 border-4 border-emerald-500 border-t-transparent rounded-full animate-spin" />
        <p className="text-sm font-mono text-zinc-500 uppercase tracking-widest animate-pulse">
          Đang kết nối không khiếu...
        </p>
      </div>
    );
  }

  // Force combat view if player is actively in combat
  const inCombat = player && player.active_combat;

  return (
    <div className="min-h-screen flex flex-col py-4 sm:py-8 px-3 sm:px-6 lg:px-8 relative">
      {/* Top Header Navigation */}
      <header className="max-w-7xl w-full mx-auto flex justify-between items-center pb-4 border-b border-zinc-900/50 mb-4">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-emerald-950/60 border border-emerald-500/20 flex items-center justify-center shrink-0">
              <Sparkles className="w-4 h-4 text-emerald-400" />
            </div>
            <h1 className="text-lg sm:text-xl font-bold font-serif tracking-wider text-zinc-100 bg-gradient-to-r from-zinc-100 to-zinc-400 bg-clip-text text-transparent">
              GuWorld
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
            <button
              onClick={() => setView("wiki")}
              className="px-3 py-1.5 rounded-xl bg-emerald-950/40 hover:bg-emerald-900/60 text-xs text-emerald-400 hover:text-emerald-300 font-serif transition-colors border border-emerald-800/40 cursor-pointer flex items-center gap-1"
            >
              <HelpCircle className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
              <span>Wiki Game</span>
            </button>
          )}

          {player && !inCombat && (
            <button
              onClick={handleLogout}
              className="px-3 py-1.5 rounded-xl bg-red-900/40 hover:bg-red-900/60 text-xs text-red-400 hover:text-red-300 font-serif transition-colors border border-red-800/40 cursor-pointer"
            >
              Đăng Xuất
            </button>
          )}
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-grow flex items-start justify-center w-full max-w-6xl mx-auto">
        {!player ? (
          <div className="py-12 w-full flex justify-center">
            <Registration
              onRegister={register}
              onLogin={login}
              isLoading={isLoading}
              error={error}
              onClearError={clearError}
            />
          </div>
        ) : inCombat ? (
          <div className="w-full">
            <CombatScreen
              player={player}
              enemy={player.active_combat!}
              guDictionary={guDictionary}
              actionLoading={actionLoading}
              onAttack={executeCombatTurn}
              onRetreat={retreat}
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
            guInventory={player.gu_inventory}
            materials={player.materials_inventory}
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
        ) : currentView === "feedback" ? (
          <div className="w-full">
            <FeedbackView />
          </div>
        ) : (
          <div className="text-center py-12 text-zinc-500 font-mono">Giao diện không khả dụng</div>
        )}
      </main>

      {/* Footer Details */}
      <footer className="max-w-7xl w-full mx-auto text-center mt-8 pt-4 border-t border-zinc-950 text-xs text-zinc-600 font-mono">
        <p>Cổ Chân Nhân Web Game © 2026 • Thiết Kế Trực Quan Sinh Động</p>
      </footer>

      {/* Toast Notification Container */}
      <ToastContainer />

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
    </div>
  );
}
