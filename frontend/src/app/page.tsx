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
    inspectedProfile,
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

  // 2. Real-time Status Polling (Every 5 seconds) - Skip polling if actively in combat to avoid lag/race conditions
  useEffect(() => {
    if (!player || player.active_combat) return;

    const interval = setInterval(() => {
      fetchPlayerStatus(player.id);
    }, 5000);

    return () => clearInterval(interval);
  }, [player, fetchPlayerStatus]);

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

          {player && !inCombat && currentView !== "dashboard" && (
            <button
              onClick={() => setView("dashboard")}
              className="px-3 py-1.5 rounded-xl bg-zinc-900 hover:bg-zinc-800 text-xs text-zinc-300 hover:text-white font-serif transition-colors border border-zinc-800 cursor-pointer flex items-center gap-1"
            >
              <span>⬅</span>
              <span>Trở về Sảnh</span>
            </button>
          )}
        </div>

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

          <div className="flex items-center gap-1.5 text-[10px] sm:text-xs text-zinc-500 border border-zinc-900 bg-zinc-950/20 px-2 sm:px-3 py-1.5 rounded-xl">
            <HelpCircle className="w-3 h-3 sm:w-3.5 sm:h-3.5 text-zinc-600 shrink-0" />
            <span className="hidden sm:inline">Bản Thử Nghiệm (Phase 4 & 5)</span>
            <span className="sm:hidden">Beta</span>
          </div>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="flex-grow flex items-start justify-center w-full mx-auto">
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
                onLogout={handleLogout}
                guDictionary={guDictionary}
              />
          </div>
        ) : currentView === "inventory" ? (
          <div className="w-full">
            <GuInventory
              player={player}
              onFeed={feedGu}
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
    </div>
  );
}
