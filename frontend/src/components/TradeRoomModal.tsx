"use client";

import React, { useEffect, useState } from "react";
import { useGameStore, GuInstance } from "@/store/useGameStore";
import { 
  X, Coins, Sparkles, Lock, Unlock, CheckCircle2, 
  Trash2, Plus, Minus, RefreshCw, User, ShieldAlert 
} from "lucide-react";

export const TradeRoomModal: React.FC = () => {
  const { 
    player, 
    activeTrade, 
    cancelTrade, 
    updateTradeOffer, 
    lockTrade, 
    confirmTrade, 
    guDictionary, 
    itemDictionary,
    showConfirm,
  } = useGameStore();

  const [primevalInput, setPrimevalInput] = useState<number>(0);
  const [immortalInput, setImmortalInput] = useState<number>(0);
  const [selectedMat, setSelectedMat] = useState<string>("");
  const [selectedMatQty, setSelectedMatQty] = useState<number>(1);
  const [activeTab, setActiveTab] = useState<"stones" | "gus" | "materials">("stones");
  const [expandedGus, setExpandedGus] = useState<Record<string, boolean>>({});
  const toggleGuGroup = (guId: string) => {
    setExpandedGus((prev) => ({ ...prev, [guId]: !prev[guId] }));
  };

  // Safety checks
  if (!player || !activeTrade) return null;

  const isSender = player.id === activeTrade.sender_id;
  
  // Destructure statuses and offers
  const myOffer = isSender ? activeTrade.sender_offer : activeTrade.receiver_offer;
  const myLocked = isSender ? activeTrade.sender_locked : activeTrade.receiver_locked;
  const myConfirmed = isSender ? activeTrade.sender_confirmed : activeTrade.receiver_confirmed;

  const oppOffer = isSender ? activeTrade.receiver_offer : activeTrade.sender_offer;
  const oppLocked = isSender ? activeTrade.receiver_locked : activeTrade.sender_locked;
  const oppConfirmed = isSender ? activeTrade.receiver_confirmed : activeTrade.sender_confirmed;
  const oppUsername = isSender ? activeTrade.receiver_username : activeTrade.sender_username;

  // Initialize stone inputs from database offer
  useEffect(() => {
    if (myOffer) {
      setPrimevalInput(myOffer.primeval_stones || 0);
      setImmortalInput(myOffer.immortal_stones || 0);
    }
  }, [activeTrade]);

  // Sync Stone offers to backend
  const syncStones = () => {
    if (myLocked) return;
    
    // Clamp values
    const clampedPrimeval = Math.max(0, Math.min(player.primeval_stones, primevalInput));
    const clampedImmortal = Math.max(0, Math.min(player.immortal_stones, immortalInput));
    
    setPrimevalInput(clampedPrimeval);
    setImmortalInput(clampedImmortal);

    const updatedOffer = {
      primeval_stones: clampedPrimeval,
      immortal_stones: clampedImmortal,
      gus: myOffer?.gus || [],
      materials: myOffer?.materials || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Sync Gu selections
  const toggleGu = (guInstanceId: string) => {
    if (myLocked) return;

    const currentGus = myOffer?.gus || [];
    let updatedGus: string[];

    if (currentGus.includes(guInstanceId)) {
      updatedGus = currentGus.filter((id: string) => id !== guInstanceId);
    } else {
      updatedGus = [...currentGus, guInstanceId];
    }

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: updatedGus,
      materials: myOffer?.materials || {},
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Sync Material offers
  const addMaterial = () => {
    if (myLocked || !selectedMat) return;

    const currentMats = { ...(myOffer?.materials || {}) };
    const maxQty = player.materials_inventory[selectedMat] || 0;
    const currentQty = currentMats[selectedMat] || 0;
    const targetQty = Math.max(1, Math.min(maxQty, currentQty + selectedMatQty));

    currentMats[selectedMat] = targetQty;

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: currentMats,
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
    setSelectedMat("");
    setSelectedMatQty(1);
  };

  const removeMaterial = (materialId: string) => {
    if (myLocked) return;

    const currentMats = { ...(myOffer?.materials || {}) };
    delete currentMats[materialId];

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: currentMats,
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  const adjustMaterialQty = (materialId: string, delta: number) => {
    if (myLocked) return;

    const currentMats = { ...(myOffer?.materials || {}) };
    const currentQty = currentMats[materialId] || 0;
    const maxQty = player.materials_inventory[materialId] || 0;
    const newQty = currentQty + delta;

    if (newQty <= 0) {
      delete currentMats[materialId];
    } else {
      currentMats[materialId] = Math.min(maxQty, newQty);
    }

    const updatedOffer = {
      primeval_stones: myOffer?.primeval_stones || 0,
      immortal_stones: myOffer?.immortal_stones || 0,
      gus: myOffer?.gus || [],
      materials: currentMats,
    };

    updateTradeOffer(activeTrade.id, updatedOffer);
  };

  // Helpers
  const getGuName = (identifier: string) => {
    // If it is an instance ID found in gu_details
    const guIdFromDetails = activeTrade.gu_details?.[identifier];
    if (guIdFromDetails) {
      return guDictionary[guIdFromDetails]?.name || guIdFromDetails.replace(/^GU_/, "").replace(/_/g, " ");
    }
    // If it is a spec ID directly
    if (guDictionary[identifier]) {
      return guDictionary[identifier].name;
    }
    // If it is an instance ID from local inventory
    const localGu = (player.gu_inventory || []).find((g) => g.instance_id === identifier);
    if (localGu) {
      return guDictionary[localGu.gu_id]?.name || localGu.gu_id.replace(/^GU_/, "").replace(/_/g, " ");
    }
    // Fallback
    return identifier.replace(/^GU_/, "").replace(/_/g, " ");
  };

  const getMatName = (matId: string) => {
    return itemDictionary[matId]?.name || matId;
  };

  // Filter list of materials in inventory that are not fully added yet
  const availableMaterials = Object.keys(player.materials_inventory || {}).filter(
    (matId) => {
      const playerQty = player.materials_inventory[matId] || 0;
      const offeredQty = myOffer?.materials?.[matId] || 0;
      return playerQty > offeredQty;
    }
  );

  // Filter player Gus (Exclude Vital Gus, Equipped Gus and Dead Gus)
  const tradeableGus = (player.gu_inventory || []).filter(
    (g) => !g.is_vital && !g.equipped && g.status !== "DEAD"
  );

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-zinc-950/85 backdrop-blur-md animate-fade-in">
      <div className="w-full max-w-4xl bg-zinc-950 border border-zinc-900 rounded-3xl overflow-hidden shadow-2xl flex flex-col max-h-[90vh]">
        
        {/* Header */}
        <div className="px-6 py-4 border-b border-zinc-900 flex justify-between items-center bg-gradient-to-r from-zinc-950 to-zinc-900">
          <div className="flex items-center gap-2.5">
            <div className="w-9 h-9 rounded-xl bg-emerald-950/50 border border-emerald-500/20 flex items-center justify-center shrink-0">
              <RefreshCw className="w-4 h-4 text-emerald-400 animate-spin" style={{ animationDuration: "10s" }} />
            </div>
            <div>
              <h3 className="text-base font-serif font-bold text-zinc-100 flex items-center gap-2">
                Không Gian Giao Dịch Thông Thiên
              </h3>
              <p className="text-[10px] text-zinc-500 font-mono">
                Được liên kết an toàn qua tần số Thông Thiên Cổ
              </p>
            </div>
          </div>
          <button
            onClick={async () => {
              if (await showConfirm("Đạo hữu có chắc chắn muốn hủy kết nối giao dịch này không?")) {
                cancelTrade(activeTrade.id);
              }
            }}
            className="p-1.5 rounded-lg bg-zinc-900/60 hover:bg-red-950/40 text-zinc-400 hover:text-red-400 border border-zinc-900/60 hover:border-red-900/30 transition-all cursor-pointer"
            title="Hủy giao dịch"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Main Workspace split */}
        <div className="flex-1 overflow-y-auto grid grid-cols-1 md:grid-cols-2 divide-y md:divide-y-0 md:divide-x divide-zinc-900">
          
          {/* LEFT SIDE: Player (You) */}
          <div className="p-6 flex flex-col gap-5 bg-zinc-950/40">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-serif font-bold text-zinc-300 flex items-center gap-1.5">
                <User className="w-4 h-4 text-emerald-400" />
                Đạo Hữu
              </h4>
              <div className="flex items-center gap-1.5">
                {myLocked ? (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-emerald-950/50 text-emerald-400 border border-emerald-500/30 font-mono flex items-center gap-1 animate-pulse">
                    <Lock className="w-3 h-3" /> Đã Khóa
                  </span>
                ) : (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-zinc-900 text-zinc-400 border border-zinc-800 font-mono flex items-center gap-1">
                    <Unlock className="w-3 h-3" /> Đang Biên Soạn
                  </span>
                )}
                {myConfirmed && (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-blue-950/50 text-blue-400 border border-blue-500/30 font-mono flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" /> Xác Nhận
                  </span>
                )}
              </div>
            </div>

            {/* Current Offer Display Box */}
            <div className="bg-zinc-950 border border-zinc-900/80 rounded-2xl p-4 space-y-3 min-h-[120px] shadow-inner">
              <span className="text-[9px] uppercase tracking-wider text-zinc-500 font-mono block">Ưu đãi hiện tại của bạn:</span>
              
              {/* Stones offered */}
              <div className="flex flex-wrap gap-2 text-xs font-mono">
                {((myOffer?.primeval_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-yellow-950/35 border border-yellow-500/20 text-yellow-500 flex items-center gap-1">
                    <Coins className="w-3.5 h-3.5" />
                    {(myOffer.primeval_stones).toLocaleString()} Nguyên Thạch
                  </span>
                )}
                {((myOffer?.immortal_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-purple-950/35 border border-purple-500/20 text-purple-400 flex items-center gap-1">
                    <Sparkles className="w-3.5 h-3.5" />
                    {(myOffer.immortal_stones).toFixed(1)} Tiên Nguyên
                  </span>
                )}
              </div>

              {/* Gus offered */}
              {myOffer?.gus && myOffer.gus.length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Cổ trùng ({myOffer.gus.length}):</span>
                  <div className="flex flex-wrap gap-1.5">
                    {(myOffer.gus as string[]).map((instId: string) => {
                      const guName = getGuName(instId);
                      return (
                        <span key={instId} className="text-xs px-2.5 py-1 rounded-xl bg-emerald-950/30 border border-emerald-900/40 text-emerald-400 flex items-center gap-1">
                          {guName}
                          {!myLocked && (
                            <button onClick={() => toggleGu(instId)} className="hover:text-red-400 cursor-pointer ml-1">
                              <X className="w-3 h-3" />
                            </button>
                          )}
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Materials offered */}
              {myOffer?.materials && Object.keys(myOffer.materials).length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Nguyên liệu:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(myOffer.materials as Record<string, number>).map(([matId, qty]) => (
                      <span key={matId} className="text-xs px-2.5 py-1 rounded-xl bg-zinc-900 border border-zinc-800 text-zinc-300 flex items-center gap-1.5">
                        {getMatName(matId)} (x{qty})
                        {!myLocked && (
                          <div className="flex items-center gap-1 border-l border-zinc-800 pl-1.5 ml-1">
                            <button onClick={() => adjustMaterialQty(matId, -1)} className="hover:text-amber-500 cursor-pointer">
                              <Minus className="w-2.5 h-2.5" />
                            </button>
                            <button onClick={() => adjustMaterialQty(matId, 1)} className="hover:text-amber-500 cursor-pointer">
                              <Plus className="w-2.5 h-2.5" />
                            </button>
                            <button onClick={() => removeMaterial(matId)} className="hover:text-red-400 cursor-pointer pl-1 text-red-500/80">
                              <Trash2 className="w-2.5 h-2.5" />
                            </button>
                          </div>
                        )}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {!(myOffer?.primeval_stones || 0) && !(myOffer?.immortal_stones || 0) && (!myOffer?.gus || myOffer.gus.length === 0) && (!myOffer?.materials || Object.keys(myOffer.materials).length === 0) && (
                <p className="text-xs text-zinc-600 italic">Chưa chọn vật phẩm nào để giao dịch</p>
              )}
            </div>

            {/* Asset Selectors Panel (Disabled if locked) */}
            <div className="flex-1 flex flex-col border border-zinc-900 rounded-2xl overflow-hidden bg-zinc-950/20">
              
              {/* Selectors tabs */}
              <div className="flex border-b border-zinc-900 text-xs font-mono bg-zinc-950">
                <button
                  onClick={() => setActiveTab("stones")}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-all ${
                    activeTab === "stones"
                      ? "text-amber-400 border-amber-500/40 bg-amber-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Linh Thạch
                </button>
                <button
                  onClick={() => setActiveTab("gus")}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-all ${
                    activeTab === "gus"
                      ? "text-emerald-400 border-emerald-500/40 bg-emerald-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Cổ Trùng
                </button>
                <button
                  onClick={() => setActiveTab("materials")}
                  className={`flex-1 py-2.5 border-b-2 font-bold cursor-pointer transition-all ${
                    activeTab === "materials"
                      ? "text-violet-400 border-violet-500/40 bg-violet-950/5"
                      : "text-zinc-500 border-transparent hover:text-zinc-300"
                  }`}
                >
                  Nguyên Liệu
                </button>
              </div>

              {/* Selector Content Area */}
              <div className="flex-1 p-4 overflow-y-auto max-h-[200px]">
                {myLocked ? (
                  <div className="h-full flex items-center justify-center text-center flex-col gap-1 text-zinc-500 font-serif">
                    <Lock className="w-5 h-5 text-zinc-600" />
                    <p className="text-xs">Giao dịch đã được khóa</p>
                    <p className="text-[10px] text-zinc-600 font-mono">Vui lòng hủy khóa nếu muốn sửa ưu đãi</p>
                  </div>
                ) : (
                  <>
                    {/* Stones selectors */}
                    {activeTab === "stones" && (
                      <div className="space-y-4">
                        {/* Primeval Stones */}
                        <div className="space-y-1.5">
                          <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500">
                            <span className="flex items-center gap-1 text-yellow-500/95"><Coins className="w-3 h-3" /> Nguyên Thạch</span>
                            <span>Sở hữu: {(player.primeval_stones).toLocaleString()}</span>
                          </div>
                          <input
                            type="number"
                            value={primevalInput}
                            onChange={(e) => setPrimevalInput(parseInt(e.target.value) || 0)}
                            onBlur={syncStones}
                            onKeyDown={(e) => e.key === "Enter" && syncStones()}
                            min={0}
                            max={player.primeval_stones}
                            className="w-full bg-zinc-950 border border-zinc-900 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:border-zinc-800 font-mono"
                          />
                        </div>

                        {/* Immortal Stones (TIEN NGUYEN THACH - for Rank >= 6) */}
                        {player.rank >= 6 && (
                          <div className="space-y-1.5">
                            <div className="flex justify-between items-center text-[10px] font-mono text-zinc-500">
                              <span className="flex items-center gap-1 text-purple-400"><Sparkles className="w-3 h-3" /> Tiên Nguyên Thạch</span>
                              <span>Sở hữu: {player.immortal_stones.toFixed(1)}</span>
                            </div>
                            <input
                              type="number"
                              step="0.1"
                              value={immortalInput}
                              onChange={(e) => setImmortalInput(parseFloat(e.target.value) || 0)}
                              onBlur={syncStones}
                              onKeyDown={(e) => e.key === "Enter" && syncStones()}
                              min={0}
                              max={player.immortal_stones}
                              className="w-full bg-zinc-950 border border-zinc-900 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none focus:border-zinc-800 font-mono"
                            />
                          </div>
                        )}
                      </div>
                    )}

                    {/* Gus selectors */}
                    {activeTab === "gus" && (
                      <div className="space-y-1.5">
                        {tradeableGus.length > 0 ? (
                          (() => {
                            const grouped: Record<string, GuInstance[]> = {};
                            tradeableGus.forEach((gu) => {
                              if (!grouped[gu.gu_id]) grouped[gu.gu_id] = [];
                              grouped[gu.gu_id].push(gu);
                            });

                            return Object.entries(grouped).map(([guId, list]) => {
                              const name = guDictionary[guId]?.name || guId;
                              const rank = guDictionary[guId]?.rank || 1;
                              const isExpanded = !!expandedGus[guId];
                              
                              // Count how many instances in this group are already selected in myOffer
                              const selectedCountInGroup = list.filter(g => (myOffer?.gus || []).includes(g.instance_id)).length;

                              return (
                                <div key={guId} className="border border-zinc-900 bg-zinc-950/40 rounded-xl overflow-hidden">
                                  {/* Group Header Row */}
                                  <div
                                    onClick={() => toggleGuGroup(guId)}
                                    className="p-3 flex justify-between items-center cursor-pointer hover:bg-zinc-950/80 transition-all select-none font-serif text-xs"
                                  >
                                    <div className="flex items-center gap-2 flex-wrap">
                                      <span className="font-bold text-zinc-200">{name}</span>
                                      <span className="text-[10px] font-mono px-1.5 py-0.2 rounded-full bg-zinc-900 border border-zinc-850 text-zinc-400">
                                        x{list.length}
                                      </span>
                                      <span className="text-[8px] font-mono border border-amber-500/20 px-1 py-0.2 rounded text-amber-400 uppercase">
                                        Rank {rank}
                                      </span>
                                      {selectedCountInGroup > 0 && (
                                        <span className="text-[8px] font-mono font-bold text-emerald-450 bg-emerald-950/20 border border-emerald-500/25 px-1 py-0.2 rounded uppercase animate-pulse">
                                          Đã chọn {selectedCountInGroup}
                                        </span>
                                      )}
                                    </div>
                                    <span className="text-[10px] font-mono text-zinc-500">
                                      {isExpanded ? "Thu gọn" : "Xem cá thể"}
                                    </span>
                                  </div>

                                  {/* Expandable list of instances */}
                                  {isExpanded && (
                                    <div className="bg-zinc-950 border-t border-zinc-900 p-2 space-y-1.5">
                                      {list.map((gu) => {
                                        const selected = (myOffer?.gus || []).includes(gu.instance_id);
                                        const isSleeping = gu.status === "SLEEPING";
                                        const isSealed = gu.status === "SEALED";
                                        
                                        return (
                                          <button
                                            type="button"
                                            key={gu.instance_id}
                                            onClick={() => toggleGu(gu.instance_id)}
                                            className={`w-full text-left p-2.5 rounded-lg border text-xs font-mono flex justify-between items-center transition-all cursor-pointer ${
                                              selected
                                                ? "bg-emerald-950/20 border-emerald-500/40 text-emerald-400 shadow-md shadow-emerald-950/15"
                                                : "bg-zinc-950 border-zinc-900 hover:border-zinc-850 text-zinc-300"
                                            }`}
                                          >
                                            <div className="flex items-center gap-2">
                                              <span className="font-bold">#{gu.instance_id.slice(0, 6)}</span>
                                              <span className="text-[10px] text-zinc-550">Độ no: {gu.hunger}/100</span>
                                              {isSleeping && <span className="text-[8px] font-bold text-amber-500 uppercase bg-amber-955/20 px-1 rounded border border-amber-900/10">Ngủ</span>}
                                              {isSealed && <span className="text-[8px] font-bold text-cyan-400 uppercase bg-cyan-955/20 px-1 rounded border border-cyan-900/10">Ấn</span>}
                                            </div>
                                            <span className="text-[9px] font-bold uppercase">
                                              {selected ? "Hủy chọn" : "Tích chọn"}
                                            </span>
                                          </button>
                                        );
                                      })}
                                    </div>
                                  )}
                                </div>
                              );
                            });
                          })()
                        ) : (
                          <p className="text-xs text-zinc-650 italic text-center py-4">Không có Cổ trùng nào có thể giao dịch</p>
                        )}
                      </div>
                    )}

                    {/* Materials selectors */}
                    {activeTab === "materials" && (
                      <div className="space-y-3">
                        <div className="flex gap-2">
                          <select
                            value={selectedMat}
                            onChange={(e) => setSelectedMat(e.target.value)}
                            className="flex-grow bg-zinc-950 border border-zinc-900 rounded-xl p-3 text-xs text-zinc-300 focus:outline-none font-serif cursor-pointer"
                          >
                            <option value="">-- Chọn Nguyên Liệu --</option>
                            {availableMaterials.map((matId) => (
                              <option key={matId} value={matId}>
                                {getMatName(matId)} (x{player.materials_inventory[matId] - (myOffer?.materials?.[matId] || 0)})
                              </option>
                            ))}
                          </select>

                          {selectedMat && (
                            <input
                              type="number"
                              min={1}
                              max={player.materials_inventory[selectedMat] - (myOffer?.materials?.[selectedMat] || 0)}
                              value={selectedMatQty}
                              onChange={(e) => setSelectedMatQty(Math.max(1, parseInt(e.target.value) || 1))}
                              className="w-16 bg-zinc-950 border border-zinc-900 rounded-xl p-3 text-xs text-center font-mono text-zinc-300 focus:outline-none"
                            />
                          )}

                          <button
                            onClick={addMaterial}
                            disabled={!selectedMat}
                            className="px-4 bg-violet-600 hover:bg-violet-500 disabled:bg-zinc-900 disabled:text-zinc-600 text-zinc-100 rounded-xl text-xs font-serif font-bold transition-colors cursor-pointer"
                          >
                            Thêm
                          </button>
                        </div>

                        {availableMaterials.length === 0 && (
                          <p className="text-xs text-zinc-600 italic text-center py-2">Hành lý trống hoặc toàn bộ nguyên liệu đã được chọn</p>
                        )}
                      </div>
                    )}
                  </>
                )}
              </div>
            </div>
          </div>

          {/* RIGHT SIDE: Opponent (Partner) */}
          <div className="p-6 flex flex-col gap-5 bg-zinc-950/20">
            <div className="flex items-center justify-between">
              <h4 className="text-sm font-serif font-bold text-zinc-300 flex items-center gap-1.5">
                <User className="w-4 h-4 text-violet-400 animate-pulse" />
                Đối Phương ({oppUsername})
              </h4>
              <div className="flex items-center gap-1.5">
                {oppLocked ? (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-emerald-950/50 text-emerald-400 border border-emerald-500/30 font-mono flex items-center gap-1">
                    <Lock className="w-3 h-3 text-emerald-400" /> Đã Khóa
                  </span>
                ) : (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-rose-950/20 text-rose-400 border border-rose-900/10 font-mono flex items-center gap-1 animate-pulse">
                    <Unlock className="w-3 h-3 text-rose-400" /> Đang Biên Soạn
                  </span>
                )}
                {oppConfirmed && (
                  <span className="text-[10px] px-2.5 py-1 rounded-full bg-blue-950/50 text-blue-400 border border-blue-500/30 font-mono flex items-center gap-1">
                    <CheckCircle2 className="w-3 h-3" /> Xác Nhận
                  </span>
                )}
              </div>
            </div>

            {/* Opponent's Offer Display Box */}
            <div className="bg-zinc-950 border border-zinc-900 rounded-2xl p-5 space-y-4 flex-1 shadow-md">
              <span className="text-[9px] uppercase tracking-wider text-zinc-500 font-mono block">Ưu đãi đề xuất của đối phương:</span>

              {/* Stones offered */}
              <div className="flex flex-wrap gap-2 text-xs font-mono">
                {((oppOffer?.primeval_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-yellow-950/20 border border-yellow-500/10 text-yellow-500 flex items-center gap-1 select-none">
                    <Coins className="w-3.5 h-3.5" />
                    {(oppOffer.primeval_stones).toLocaleString()} Nguyên Thạch
                  </span>
                )}
                {((oppOffer?.immortal_stones || 0) > 0) && (
                  <span className="px-2.5 py-1.5 rounded-xl bg-purple-950/20 border border-purple-500/10 text-purple-400 flex items-center gap-1 select-none">
                    <Sparkles className="w-3.5 h-3.5" />
                    {(oppOffer.immortal_stones).toFixed(1)} Tiên Nguyên
                  </span>
                )}
              </div>

              {/* Gus offered */}
              {oppOffer?.gus && oppOffer.gus.length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Cổ trùng ({oppOffer.gus.length}):</span>
                  <div className="flex flex-wrap gap-1.5">
                    {(oppOffer.gus as string[]).map((instId: string, index: number) => {
                      const guName = getGuName(instId);
                      return (
                        <span key={index} className="text-xs px-2.5 py-1 rounded-xl bg-emerald-950/20 border border-emerald-900/20 text-emerald-400 select-none">
                          {guName}
                        </span>
                      );
                    })}
                  </div>
                </div>
              )}

              {/* Materials offered */}
              {oppOffer?.materials && Object.keys(oppOffer.materials).length > 0 && (
                <div className="space-y-1">
                  <span className="text-[9px] text-zinc-500 font-mono">Nguyên liệu:</span>
                  <div className="flex flex-wrap gap-1.5">
                    {Object.entries(oppOffer.materials as Record<string, number>).map(([matId, qty]) => (
                      <span key={matId} className="text-xs px-2.5 py-1 rounded-xl bg-zinc-900 border border-zinc-850 text-zinc-300 select-none">
                        {getMatName(matId)} (x{qty})
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {!(oppOffer?.primeval_stones || 0) && !(oppOffer?.immortal_stones || 0) && (!oppOffer?.gus || oppOffer.gus.length === 0) && (!oppOffer?.materials || Object.keys(oppOffer.materials).length === 0) && (
                <p className="text-xs text-zinc-650 italic py-6 select-none">Chưa đề xuất vật phẩm nào</p>
              )}
            </div>

            {/* Guidance panel */}
            <div className="px-4 py-3 bg-zinc-900/10 border-l border-zinc-800 text-[10px] text-zinc-500 font-mono rounded-r-xl flex items-start gap-2 leading-relaxed">
              <ShieldAlert className="w-4 h-4 text-zinc-600 shrink-0" />
              <div>
                <p className="font-bold text-zinc-400">Luật Bảo Mật:</p>
                <p>Mỗi khi một bên chỉnh sửa ưu đãi, toàn bộ khóa và xác nhận sẽ được đặt lại để đảm bảo an toàn, phòng ngừa lừa đảo.</p>
              </div>
            </div>

          </div>
        </div>

        {/* Footer actions bar */}
        <div className="px-6 py-5 border-t border-zinc-900 bg-zinc-950 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4">
          <div className="text-[10px] font-mono text-zinc-500 leading-tight">
            <span>Thực trạng liên kết: </span>
            {myLocked && oppLocked ? (
              <span className="text-emerald-400">Kết nối ổn định. Đã khóa cả hai bên. Sẵn sàng xác nhận.</span>
            ) : (
              <span className="text-amber-500">Đang đồng bộ hóa ưu đãi... Cần cả hai bên cùng khóa.</span>
            )}
          </div>

          <div className="flex items-center gap-3">
            {/* Lock Offer Button */}
            {!myLocked ? (
              <button
                onClick={() => lockTrade(activeTrade.id)}
                className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-emerald-600 hover:bg-emerald-500 text-zinc-100 transition-colors flex items-center gap-1.5 cursor-pointer shadow-lg shadow-emerald-950/25 border border-emerald-500/20"
              >
                <Lock className="w-3.5 h-3.5" /> Khóa Giao Dịch
              </button>
            ) : (
              <span className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-zinc-900 text-zinc-500 border border-zinc-800 flex items-center gap-1.5 select-none">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" /> Đã Khóa Của Bạn
              </span>
            )}

            {/* Confirm Trade Button (Enabled only when both locked and you haven't confirmed yet) */}
            {myLocked && oppLocked && !myConfirmed ? (
              <button
                onClick={() => confirmTrade(activeTrade.id)}
                className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-gradient-to-r from-blue-600 to-indigo-500 hover:from-blue-500 hover:to-indigo-400 text-zinc-100 transition-all cursor-pointer shadow-lg shadow-blue-950/25 border border-blue-500/20"
              >
                <CheckCircle2 className="w-3.5 h-3.5" /> Xác Nhận Giao Dịch
              </button>
            ) : myConfirmed ? (
              <span className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-indigo-950/40 text-indigo-400 border border-indigo-900/35 flex items-center gap-1.5 animate-pulse select-none">
                <RefreshCw className="w-3 h-3 animate-spin" /> Đang Chờ Đối Phương...
              </span>
            ) : (
              <button
                disabled
                className="px-5 py-2.5 rounded-xl text-xs font-serif font-bold bg-zinc-900 text-zinc-650 border border-zinc-950 flex items-center gap-1.5 select-none"
              >
                Xác Nhận
              </button>
            )}
          </div>
        </div>

      </div>
    </div>
  );
};
