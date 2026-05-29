"use client";

import { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Registration } from "@/components/Registration";
import { Aperture } from "@/components/Aperture";
import { PlayerStats } from "@/components/PlayerStats";
import { LogConsole } from "@/components/LogConsole";
import { GuInventory } from "@/components/GuInventory";
import { RefiningFurnace } from "@/components/RefiningFurnace";
import { ExplorationMap } from "@/components/ExplorationMap";
import { CombatScreen } from "@/components/CombatScreen";
import { CaravanShop } from "@/components/CaravanShop";
import { Sparkles, HelpCircle } from "lucide-react";
import { BaoHoangMarket } from "@/components/BaoHoangMarket";

export default function Home() {
  const {
    player,
    isLoading,
    actionLoading,
    error,
    logs,
    login,
    register,
    fetchPlayerStatus,
    initializeSession,
    temperAperture,
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
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-emerald-950/60 border border-emerald-500/20 flex items-center justify-center shrink-0">
            <Sparkles className="w-4 h-4 text-emerald-400" />
          </div>
          <h1 className="text-lg sm:text-xl font-bold font-serif tracking-wider text-zinc-100 bg-gradient-to-r from-zinc-100 to-zinc-400 bg-clip-text text-transparent">
            GuWorld
          </h1>
        </div>

        <div className="flex items-center gap-1.5 text-[10px] sm:text-xs text-zinc-500 border border-zinc-900 bg-zinc-950/20 px-2 sm:px-3 py-1.5 rounded-xl">
          <HelpCircle className="w-3 h-3 sm:w-3.5 sm:h-3.5 text-zinc-600 shrink-0" />
          <span className="hidden sm:inline">Bản Thử Nghiệm (Phase 4 & 5)</span>
          <span className="sm:hidden">Beta</span>
        </div>
      </header>

      {/* Tab Switcher - scrollable on mobile */}
      {player && !inCombat && (
        <div className="max-w-7xl w-full mx-auto border-b border-zinc-900 mb-6 shrink-0 overflow-x-auto">
          <div className="flex min-w-max sm:min-w-0 sm:justify-center gap-1 sm:gap-4 px-1">
            {[
              { id: "dashboard", label: "TU LUYỆN", shortLabel: "Tu Luyện", activeColor: "text-emerald-400", barColor: "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]" },
              { id: "explore", label: "KHÁM PHÁ BẢN ĐỒ", shortLabel: "Khám Phá", activeColor: "text-cyan-400", barColor: "bg-cyan-500 shadow-[0_0_8px_rgba(34,211,238,0.5)]" },
              { id: "refine", label: "LUYỆN CỔ THẤT", shortLabel: "Luyện Cổ", activeColor: "text-amber-400", barColor: "bg-amber-500 shadow-[0_0_8px_rgba(234,179,8,0.5)]" },
              { id: "shop", label: "GIẢ GIA THƯƠNG ĐỘI", shortLabel: "Thương Đội", activeColor: "text-amber-500", barColor: "bg-amber-600 shadow-[0_0_8px_rgba(217,119,6,0.5)]" },
              { id: "auction", label: "BẢO HOÀNG THIÊN", shortLabel: "Bảo Hoàng", activeColor: "text-violet-400", barColor: "bg-violet-500 shadow-[0_0_8px_rgba(139,92,246,0.5)]" },
            ].map((tab) => (
              <button
                key={tab.id}
                onClick={() => setView(tab.id)}
                className={`pb-3 px-3 sm:px-4 font-serif text-[10px] sm:text-sm font-semibold tracking-wider transition-all relative cursor-pointer whitespace-nowrap ${
                  currentView === tab.id ? `${tab.activeColor} font-bold` : "text-zinc-500 hover:text-zinc-300"
                }`}
              >
                <span className="hidden sm:inline">{tab.label}</span>
                <span className="sm:hidden">{tab.shortLabel}</span>
                {currentView === tab.id && (
                  <span className={`absolute bottom-0 left-0 right-0 h-0.5 ${tab.barColor}`} />
                )}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Main Content Area */}
      <main className="flex-grow flex items-start justify-center max-w-7xl w-full mx-auto">
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
          <div className="grid grid-cols-1 xl:grid-cols-12 gap-4 sm:gap-6 w-full items-start">
            {/* Left Column: Aperture + PlayerStats — side by side on md, stacked on mobile */}
            <div className="xl:col-span-5 grid grid-cols-1 md:grid-cols-2 xl:grid-cols-1 gap-4 sm:gap-6 w-full">
              <Aperture
                currentMana={player.current_mana}
                maxMana={player.max_mana}
                aptitudeType={player.aptitude_type}
                aptitudePercent={player.aptitude_percent}
                rank={player.rank}
                subRank={player.sub_rank}
                guInventory={player.gu_inventory}
                guDictionary={guDictionary}
                actionLoading={actionLoading}
                cultivationProgress={player.cultivation_progress}
                onTemper={temperAperture}
              />
              <PlayerStats
                player={player}
                onLogout={handleLogout}
                guDictionary={guDictionary}
                onEquipToggle={equipGu}
              />
            </div>

            {/* Right Column: Log + Inventory */}
            <div className="xl:col-span-7 flex flex-col gap-4 sm:gap-6 w-full">
              <LogConsole logs={logs} />
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
        ) : (
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
        )}
      </main>

      {/* Footer Details */}
      <footer className="max-w-7xl w-full mx-auto text-center mt-8 pt-4 border-t border-zinc-950 text-xs text-zinc-600 font-mono">
        <p>Cổ Chân Nhân Web Game © 2026 • Thiết Kế Trực Quan Sinh Động</p>
      </footer>
    </div>
  );
}
