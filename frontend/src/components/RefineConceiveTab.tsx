"use client";

import React, { useState, useEffect, useMemo, useCallback } from "react";
import { Player, GuSpec, ItemSpec, useGameStore, ConceiveCandidate, ConceiveResult } from "@/store/useGameStore";
import { pathLabels, translatePath } from "@/lib/translate";
import { getItemIconUrl } from "@/lib/item-icons";
import { 
  Sparkles, AlertCircle, Check, X, RefreshCw, Coins, Zap, Brain, 
  Search, ShieldAlert, ArrowRight, Skull, Dna, Info, Flame 
} from "lucide-react";

interface RefineConceiveTabProps {
  player: Player;
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
  preselectedGuId?: string | null;
  onSelectRecipeForRefine?: (recipeId: string) => void;
}

export const RefineConceiveTab: React.FC<RefineConceiveTabProps> = ({
  player,
  guDictionary,
  itemDictionary,
  preselectedGuId,
  onSelectRecipeForRefine,
}) => {
  const fetchConceiveCandidates = useGameStore((state) => state.fetchConceiveCandidates);
  const conceiveRecipe = useGameStore((state) => state.conceiveRecipe);
  const addToast = useGameStore((state) => state.addToast);

  // Filter States
  const [selectedPath, setSelectedPath] = useState<string>("");
  const [selectedRank, setSelectedRank] = useState<number>(0);
  const [searchKeyword, setSearchKeyword] = useState<string>("");

  // Candidate Data State
  const [candidates, setCandidates] = useState<ConceiveCandidate[]>([]);
  const [selectedCandidate, setSelectedCandidate] = useState<ConceiveCandidate | null>(null);
  const [loadingCandidates, setLoadingCandidates] = useState<boolean>(false);

  // Catalyst State
  const [catalystMode, setCatalystMode] = useState<"none" | "gu" | "material">("none");
  const [selectedGuInstanceId, setSelectedGuInstanceId] = useState<string>("");
  const [selectedMaterialId, setSelectedMaterialId] = useState<string>("");

  // Action & Modal State
  const [isConceiving, setIsConceiving] = useState<boolean>(false);
  const [resultModal, setResultModal] = useState<ConceiveResult | null>(null);

  // Load Candidates
  const loadCandidates = useCallback(async () => {
    setLoadingCandidates(true);
    try {
      const list = await fetchConceiveCandidates({
        path: selectedPath || undefined,
        rank: selectedRank > 0 ? selectedRank : undefined,
        search: searchKeyword.trim() || undefined,
      });
      setCandidates(list);

      // Handle preselection or maintain selected candidate
      if (preselectedGuId) {
        const found = list.find((c) => c.id === preselectedGuId);
        if (found) {
          setSelectedCandidate(found);
        } else if (list.length > 0 && !selectedCandidate) {
          setSelectedCandidate(list[0]);
        }
      } else if (list.length > 0 && (!selectedCandidate || !list.some((c) => c.id === selectedCandidate.id))) {
        setSelectedCandidate(list[0]);
      } else if (list.length === 0) {
        setSelectedCandidate(null);
      }
    } catch {
      addToast("Không thể tải danh sách Cổ trùng khả dụng", "error");
    } finally {
      setLoadingCandidates(false);
    }
  }, [fetchConceiveCandidates, selectedPath, selectedRank, searchKeyword, preselectedGuId, selectedCandidate, addToast]);

  useEffect(() => {
    const timer = setTimeout(() => {
      loadCandidates();
    }, 200);
    return () => clearTimeout(timer);
  }, [selectedPath, selectedRank, searchKeyword, loadCandidates]);

  // Reset catalyst when selected candidate changes
  useEffect(() => {
    setCatalystMode("none");
    setSelectedGuInstanceId("");
    setSelectedMaterialId("");
  }, [selectedCandidate?.id]);

  // Eligible Living Specimen Gus in Inventory
  const eligibleGus = useMemo(() => {
    if (!selectedCandidate) return [];
    const targetCleanPath = selectedCandidate.primary_path.toLowerCase();
    return (player.gu_inventory || []).filter((gu) => {
      if (gu.is_vital || gu.equipped || gu.is_locked || gu.status === "DEAD" || gu.status === "SEALED") return false;
      if (gu.rank > selectedCandidate.rank) return false;
      const spec = guDictionary[gu.gu_id];
      if (!spec || !spec.path) return false;
      const paths = Array.isArray(spec.path) ? spec.path : [String(spec.path)];
      return paths.some((p) => p.toLowerCase().includes(targetCleanPath));
    });
  }, [player.gu_inventory, selectedCandidate, guDictionary]);

  // Eligible Materials in Inventory
  const eligibleMaterials = useMemo(() => {
    if (!selectedCandidate) return [];
    const targetCleanPath = selectedCandidate.primary_path.toLowerCase();
    const matInv = player.materials_inventory || {};

    return Object.entries(matInv)
      .filter(([id, qty]) => {
        if (qty <= 0) return false;
        const spec = itemDictionary[id];
        if (!spec || spec.item_type !== "Material" || (spec.rank != null && spec.rank > selectedCandidate.rank)) return false;
        const paths = Array.isArray(spec.path) ? spec.path : [String(spec.path)];
        return paths.some((p) => p.toLowerCase().includes(targetCleanPath));
      })
      .map(([id, qty]) => ({
        id,
        qty,
        spec: itemDictionary[id],
      }));
  }, [player.materials_inventory, selectedCandidate, itemDictionary]);

  // Realtime Accuracy & Forecast Calculation
  const forecast = useMemo(() => {
    if (!selectedCandidate) return null;
    let catalystBonus = 0;

    if (catalystMode === "gu" && selectedGuInstanceId) {
      const g = eligibleGus.find((inst) => inst.instance_id === selectedGuInstanceId);
      if (g) {
        if (g.rank >= selectedCandidate.rank) catalystBonus = 0.20;
        else if (g.rank === selectedCandidate.rank - 1) catalystBonus = 0.16;
        else catalystBonus = 0.12;
      }
    } else if (catalystMode === "material" && selectedMaterialId) {
      const m = eligibleMaterials.find((mat) => mat.id === selectedMaterialId);
      if (m && m.spec) {
        if (m.spec.rank != null && m.spec.rank >= selectedCandidate.rank) catalystBonus = 0.12;
        else catalystBonus = 0.08;
      }
    }

    const accuracy = Math.min(0.88, Math.max(0.15, selectedCandidate.estimated_accuracy + catalystBonus));
    const deviationRisk = Math.max(0, Math.round((1 - accuracy) * 100));

    return {
      accuracyPercent: Math.round(accuracy * 100),
      deviationRiskPercent: deviationRisk,
      catalystBonusPercent: Math.round(catalystBonus * 100),
    };
  }, [selectedCandidate, catalystMode, selectedGuInstanceId, selectedMaterialId, eligibleGus, eligibleMaterials]);

  // Handle Action Click
  const handleConceive = async () => {
    if (!selectedCandidate || isConceiving) return;

    // Check thought & stones upfront
    if (player.current_thought < selectedCandidate.required_thought) {
      addToast(`Không đủ Niệm Lực! Cần ${selectedCandidate.required_thought} Niệm Lực.`, "error");
      return;
    }
    if (player.primeval_stones < selectedCandidate.required_stones) {
      addToast(`Không đủ Nguyên Thạch! Cần ${selectedCandidate.required_stones.toLocaleString()} Nguyên Thạch.`, "error");
      return;
    }
    if (selectedCandidate.required_immortal_stones > 0 && player.immortal_stones < selectedCandidate.required_immortal_stones) {
      addToast(`Không đủ Tiên Nguyên Thạch! Cần ${selectedCandidate.required_immortal_stones} Tiên Thạch.`, "error");
      return;
    }

    setIsConceiving(true);
    try {
      const res = await conceiveRecipe({
        target_gu_id: selectedCandidate.id,
        target_rank: selectedCandidate.rank,
        catalyst_gu_instance_id: catalystMode === "gu" ? selectedGuInstanceId : undefined,
        catalyst_item_id: catalystMode === "material" ? selectedMaterialId : undefined,
      });

      if (res) {
        setResultModal(res);
        // Refresh candidate progress in list
        loadCandidates();
      }
    } catch {
      addToast("Lỗi trong quá trình khởi niệm suy tính", "error");
    } finally {
      setIsConceiving(false);
    }
  };

  return (
    <div className="space-y-4">
      {/* Top Resources Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 p-3.5 bg-neutral-900/80 border border-neutral-800 rounded-xl">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <Brain className="w-5 h-5 text-indigo-400" />
            <div>
              <div className="text-xs text-neutral-400">Niệm Lực</div>
              <div className="text-sm font-semibold text-indigo-200">
                {Math.floor(player.current_thought)} / {player.max_thought || 100}
              </div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Coins className="w-5 h-5 text-amber-400" />
            <div>
              <div className="text-xs text-neutral-400">Nguyên Thạch</div>
              <div className="text-sm font-semibold text-amber-200">
                {Math.floor(player.primeval_stones).toLocaleString()}
              </div>
            </div>
          </div>

          {player.rank >= 6 && (
            <div className="flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-cyan-400" />
              <div>
                <div className="text-xs text-neutral-400">Tiên Nguyên Thạch</div>
                <div className="text-sm font-semibold text-cyan-200">
                  {Math.floor(player.immortal_stones || 0).toLocaleString()}
                </div>
              </div>
            </div>
          )}
        </div>

        <div className="text-xs text-neutral-400 flex items-center gap-1.5">
          <Info className="w-4 h-4 text-neutral-500" />
          Khai sáng tiêu hao Niệm Lực và Nguyên Thạch cố định theo phẩm giai Cổ đích.
        </div>
      </div>

      {/* Filter Controls Bar */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 p-3 bg-neutral-900/60 border border-neutral-800 rounded-xl">
        {/* Path Selector */}
        <div>
          <label className="block text-xs font-medium text-neutral-400 mb-1">Lưu Phái</label>
          <select
            value={selectedPath}
            onChange={(e) => setSelectedPath(e.target.value)}
            className="w-full bg-neutral-950 border border-neutral-800 text-neutral-200 text-xs rounded-lg px-2.5 py-2 focus:outline-none focus:border-indigo-500"
          >
            <option value="">Tất cả lưu phái</option>
            {Object.keys(pathLabels).map((p) => (
              <option key={p} value={p}>
                {translatePath(p)} ({p.replace(" Path", "")})
              </option>
            ))}
          </select>
        </div>

        {/* Rank Selector */}
        <div>
          <label className="block text-xs font-medium text-neutral-400 mb-1">Phẩm Giai</label>
          <select
            value={selectedRank}
            onChange={(e) => setSelectedRank(Number(e.target.value))}
            className="w-full bg-neutral-950 border border-neutral-800 text-neutral-200 text-xs rounded-lg px-2.5 py-2 focus:outline-none focus:border-indigo-500"
          >
            <option value={0}>Tất cả phẩm giai (1 - 9)</option>
            {[1, 2, 3, 4, 5, 6, 7, 8, 9].map((r) => (
              <option key={r} value={r}>
                Chuyển {r} {r >= 6 ? "(Tiên Cổ)" : "(Phàm Cổ)"}
              </option>
            ))}
          </select>
        </div>

        {/* Search */}
        <div>
          <label className="block text-xs font-medium text-neutral-400 mb-1">Tìm Kiếm</label>
          <div className="relative">
            <input
              type="text"
              placeholder="Nhập tên Cổ..."
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              className="w-full bg-neutral-950 border border-neutral-800 text-neutral-200 text-xs rounded-lg pl-8 pr-3 py-2 focus:outline-none focus:border-indigo-500"
            />
            <Search className="w-4 h-4 text-neutral-500 absolute left-2.5 top-2.5" />
          </div>
        </div>
      </div>

      {/* Main Content Split: Left Candidates List / Right Setup Altar */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4">
        {/* Left Column: Candidates List (5 Cols) */}
        <div className="lg:col-span-5 flex flex-col h-[540px] bg-neutral-900/60 border border-neutral-800 rounded-xl overflow-hidden">
          <div className="p-3 border-b border-neutral-800 flex items-center justify-between text-xs font-medium text-neutral-400">
            <span>Danh Sách Cổ Khả Dụng ({candidates.length})</span>
            {loadingCandidates && <RefreshCw className="w-3.5 h-3.5 animate-spin text-neutral-500" />}
          </div>

          <div className="flex-1 overflow-y-auto p-2 space-y-2">
            {candidates.length === 0 ? (
              <div className="text-center py-16 text-neutral-500 text-xs">
                {loadingCandidates ? "Đang dò tìm thiên cơ..." : "Không tìm thấy Cổ trùng phù hợp bộ lọc."}
              </div>
            ) : (
              candidates.map((c) => {
                const isSelected = selectedCandidate?.id === c.id;
                return (
                  <div
                    key={`${c.id}_${c.rank}`}
                    onClick={() => setSelectedCandidate(c)}
                    className={`p-2.5 rounded-lg border transition-all cursor-pointer flex items-center justify-between gap-3 ${
                      isSelected
                        ? "bg-indigo-950/40 border-indigo-500/80 shadow-sm shadow-indigo-500/20"
                        : "bg-neutral-950/40 border-neutral-800/80 hover:bg-neutral-800/40 hover:border-neutral-700"
                    }`}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="w-10 h-10 rounded-lg bg-neutral-900 border border-neutral-700 flex items-center justify-center overflow-hidden flex-shrink-0">
                        {c.image_url ? (
                          <img src={getItemIconUrl(c)} alt={c.name} className="w-full h-full object-cover" />
                        ) : (
                          <Dna className="w-5 h-5 text-neutral-500" />
                        )}
                      </div>

                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="text-xs font-semibold text-neutral-100 truncate">{c.name}</span>
                          <span className={`text-[10px] px-1.5 py-0.5 rounded font-mono ${
                            c.rank >= 6 ? "bg-amber-950 text-amber-300 border border-amber-800" : "bg-neutral-800 text-neutral-300"
                          }`}>
                            R{c.rank}
                          </span>
                        </div>
                        <div className="text-[11px] text-neutral-400 truncate mt-0.5">
                          {translatePath(c.primary_path)} • Cần {c.required_thought} 🧠
                        </div>
                      </div>
                    </div>

                    <div className="text-right flex-shrink-0">
                      {c.has_backlash_risk ? (
                        <div className="flex items-center gap-1 text-[11px] text-rose-400 font-medium">
                          <Skull className="w-3.5 h-3.5" />
                          <span>Nguy hiểm</span>
                        </div>
                      ) : (
                        <div className="text-[11px] text-emerald-400 font-medium">
                          ~{Math.round(c.estimated_accuracy * 100)}%
                        </div>
                      )}
                      {c.existing_completion_rate > 0 && (
                        <div className="text-[10px] text-amber-400/80 font-mono mt-0.5">
                          Tàn phương: {c.existing_completion_rate}%
                        </div>
                      )}
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* Right Column: Conception Setup & Forecast (7 Cols) */}
        <div className="lg:col-span-7 flex flex-col h-[540px] bg-neutral-900/60 border border-neutral-800 rounded-xl overflow-hidden p-4">
          {selectedCandidate ? (
            <div className="flex flex-col h-full justify-between space-y-4 overflow-y-auto pr-1">
              {/* Selected Gu Profile */}
              <div className="p-3.5 bg-neutral-950/80 border border-neutral-800 rounded-xl flex items-start gap-4">
                <div className="w-16 h-16 rounded-xl bg-neutral-900 border border-neutral-700 flex items-center justify-center overflow-hidden flex-shrink-0">
                  {selectedCandidate.image_url ? (
                    <img src={getItemIconUrl(selectedCandidate)} alt={selectedCandidate.name} className="w-full h-full object-cover" />
                  ) : (
                    <Dna className="w-8 h-8 text-neutral-500" />
                  )}
                </div>

                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-bold text-neutral-100">{selectedCandidate.name}</h3>
                    <span className="text-[10px] px-2 py-0.5 rounded font-mono bg-indigo-950 text-indigo-300 border border-indigo-800">
                      Chuyển {selectedCandidate.rank}
                    </span>
                    <span className="text-[10px] px-2 py-0.5 rounded bg-neutral-800 text-neutral-300">
                      {translatePath(selectedCandidate.primary_path)}
                    </span>
                  </div>

                  <p className="text-xs text-neutral-400 mt-1 line-clamp-2">
                    {selectedCandidate.description || "Chưa có ghi chép chi tiết trong cổ phổ."}
                  </p>
                </div>
              </div>

              {/* Catalyst Picker Slot */}
              <div className="p-3.5 bg-neutral-950/60 border border-neutral-800 rounded-xl space-y-3">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-neutral-300 flex items-center gap-1.5">
                    <Sparkles className="w-4 h-4 text-amber-400" />
                    Vật Dẫn Nghiên Cứu (Tăng Tỉ Lệ Trúng Đích)
                  </span>

                  <div className="flex items-center gap-1 bg-neutral-900 p-0.5 rounded-lg border border-neutral-800 text-[11px]">
                    <button
                      onClick={() => setCatalystMode("none")}
                      className={`px-2 py-1 rounded ${catalystMode === "none" ? "bg-neutral-800 text-white font-medium" : "text-neutral-400 hover:text-white"}`}
                    >
                      Không
                    </button>
                    <button
                      onClick={() => setCatalystMode("gu")}
                      className={`px-2 py-1 rounded ${catalystMode === "gu" ? "bg-indigo-900 text-indigo-200 font-medium" : "text-neutral-400 hover:text-white"}`}
                    >
                      Cổ Phôi ({eligibleGus.length})
                    </button>
                    <button
                      onClick={() => setCatalystMode("material")}
                      className={`px-2 py-1 rounded ${catalystMode === "material" ? "bg-amber-900 text-amber-200 font-medium" : "text-neutral-400 hover:text-white"}`}
                    >
                      Nguyên Liệu ({eligibleMaterials.length})
                    </button>
                  </div>
                </div>

                {catalystMode === "gu" && (
                  <div>
                    {eligibleGus.length === 0 ? (
                      <div className="text-xs text-neutral-500 py-2">
                        Không có Cổ phôi cùng hệ hợp lệ trong Không Khiếu (phải là Cổ sống, chưa trang bị, không bản mệnh).
                      </div>
                    ) : (
                      <select
                        value={selectedGuInstanceId}
                        onChange={(e) => setSelectedGuInstanceId(e.target.value)}
                        className="w-full bg-neutral-900 border border-neutral-800 text-xs text-neutral-200 rounded-lg px-2.5 py-2 focus:outline-none focus:border-indigo-500"
                      >
                        <option value="">-- Chọn Cổ phôi hiến tế (+12% đến +20% tỉ lệ) --</option>
                        {eligibleGus.map((inst) => {
                          const spec = guDictionary[inst.gu_id];
                          return (
                            <option key={inst.instance_id} value={inst.instance_id}>
                              {spec?.name || inst.gu_id} (Chuyển {inst.rank})
                            </option>
                          );
                        })}
                      </select>
                    )}
                  </div>
                )}

                {catalystMode === "material" && (
                  <div>
                    {eligibleMaterials.length === 0 ? (
                      <div className="text-xs text-neutral-500 py-2">
                        Không có nguyên liệu cùng hệ hợp lệ trong Túi đồ.
                      </div>
                    ) : (
                      <select
                        value={selectedMaterialId}
                        onChange={(e) => setSelectedMaterialId(e.target.value)}
                        className="w-full bg-neutral-900 border border-neutral-800 text-xs text-neutral-200 rounded-lg px-2.5 py-2 focus:outline-none focus:border-indigo-500"
                      >
                        <option value="">-- Chọn nguyên liệu dẫn dắt (+8% đến +12% tỉ lệ) --</option>
                        {eligibleMaterials.map((m) => (
                          <option key={m.id} value={m.id}>
                            {m.spec?.name || m.id} (Hiện có: {m.qty})
                          </option>
                        ))}
                      </select>
                    )}
                  </div>
                )}
              </div>

              {/* Forecast & Risk Panel */}
              <div className="p-3.5 bg-neutral-950/60 border border-neutral-800 rounded-xl space-y-3">
                <div className="text-xs font-medium text-neutral-300">Dự Toán Thiên Cơ</div>

                {/* Accuracy Progress Bar */}
                {forecast && (
                  <div className="space-y-1.5">
                    <div className="flex justify-between text-xs">
                      <span className="text-neutral-400">Tỉ lệ trúng đích Cổ mong muốn:</span>
                      <span className="font-semibold text-emerald-400 font-mono">
                        {forecast.accuracyPercent}%
                        {forecast.catalystBonusPercent > 0 && (
                          <span className="text-amber-400 text-[10px] ml-1">
                            (+{forecast.catalystBonusPercent}% vật dẫn)
                          </span>
                        )}
                      </span>
                    </div>

                    <div className="w-full h-2 bg-neutral-800 rounded-full overflow-hidden">
                      <div
                        className="h-full bg-gradient-to-r from-indigo-500 via-emerald-500 to-amber-400 transition-all duration-300"
                        style={{ width: `${forecast.accuracyPercent}%` }}
                      />
                    </div>

                    <div className="flex justify-between text-[11px] text-neutral-500">
                      <span>Rủi ro rẽ nhánh sang Cổ liên quan: {forecast.deviationRiskPercent}%</span>
                      <span>Trần thiên cơ: 88%</span>
                    </div>
                  </div>
                )}

                {/* Backlash Warning Banner */}
                {selectedCandidate.has_backlash_risk && (
                  <div className="p-2.5 bg-rose-950/40 border border-rose-800/80 rounded-lg flex items-start gap-2.5 text-rose-300 text-xs">
                    <ShieldAlert className="w-4 h-4 text-rose-400 flex-shrink-0 mt-0.5" />
                    <div>
                      <div className="font-semibold text-rose-200">
                        Cảnh báo Phản Phệ ({Math.round(selectedCandidate.backlash_risk_chance * 100)}% nguy cơ)!
                      </div>
                      <div className="text-[11px] text-rose-300/80 mt-0.5">
                        Cố suy tính Cổ vượt tu vi hoặc non nớt cảnh giới có thể khiến Niệm Lực cạn sạch và mất 50% HP/Chân Nguyên!
                      </div>
                    </div>
                  </div>
                )}

                {/* Cost Summary */}
                <div className="flex items-center justify-between text-xs text-neutral-400 pt-1 border-t border-neutral-800">
                  <span>Chi phí cần tiêu hao:</span>
                  <div className="flex items-center gap-3 font-medium">
                    <span className="text-indigo-300">{selectedCandidate.required_thought} Niệm Lực</span>
                    <span className="text-amber-300">{selectedCandidate.required_stones.toLocaleString()} Nguyên Thạch</span>
                    {selectedCandidate.required_immortal_stones > 0 && (
                      <span className="text-cyan-300">{selectedCandidate.required_immortal_stones} Tiên Thạch</span>
                    )}
                  </div>
                </div>
              </div>

              {/* Action Button */}
              <button
                onClick={handleConceive}
                disabled={isConceiving}
                className="w-full py-3 bg-gradient-to-r from-indigo-600 via-purple-600 to-indigo-700 hover:from-indigo-500 hover:to-indigo-600 text-white font-semibold rounded-xl text-xs sm:text-sm shadow-lg shadow-indigo-600/30 flex items-center justify-center gap-2 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isConceiving ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Đang giao cảm thiên cơ & thôi diễn...</span>
                  </>
                ) : (
                  <>
                    <Flame className="w-4 h-4 text-amber-300" />
                    <span>Khởi Niệm Khai Sáng Bí Phương</span>
                  </>
                )}
              </button>
            </div>
          ) : (
            <div className="h-full flex flex-col items-center justify-center text-center text-neutral-500 space-y-2">
              <Brain className="w-12 h-12 text-neutral-600 stroke-[1.5]" />
              <div className="text-sm font-medium text-neutral-400">Chưa chọn Cổ trùng mục tiêu</div>
              <div className="text-xs max-w-sm text-neutral-500">
                Hãy chọn một Cổ trùng từ danh sách bên trái để thiết lập vật dẫn và dự toán thiên cơ trước khi khởi niệm thôi diễn.
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Result Pop-up Modal */}
      {resultModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
          <div className={`w-full max-w-md p-5 rounded-2xl border bg-neutral-900 shadow-2xl relative animate-in fade-in zoom-in-95 duration-200 ${
            resultModal.status === "BACKLASH"
              ? "border-rose-500/80 shadow-rose-950/50"
              : resultModal.is_instant_100
              ? "border-amber-400 shadow-amber-950/50"
              : resultModal.is_deviated
              ? "border-purple-500 shadow-purple-950/50"
              : "border-emerald-500 shadow-emerald-950/50"
          }`}>
            <button
              onClick={() => setResultModal(null)}
              className="absolute top-4 right-4 text-neutral-400 hover:text-white"
            >
              <X className="w-5 h-5" />
            </button>

            {/* Modal Header */}
            <div className="flex items-center gap-3 mb-4">
              <div className={`w-12 h-12 rounded-xl flex items-center justify-center ${
                resultModal.status === "BACKLASH"
                  ? "bg-rose-950 text-rose-400 border border-rose-800"
                  : resultModal.is_instant_100
                  ? "bg-amber-950 text-amber-300 border border-amber-700 animate-pulse"
                  : resultModal.is_deviated
                  ? "bg-purple-950 text-purple-300 border border-purple-800"
                  : "bg-emerald-950 text-emerald-300 border border-emerald-800"
              }`}>
                {resultModal.status === "BACKLASH" ? (
                  <Skull className="w-6 h-6" />
                ) : resultModal.is_instant_100 ? (
                  <Sparkles className="w-6 h-6" />
                ) : (
                  <Check className="w-6 h-6" />
                )}
              </div>

              <div>
                <h3 className="text-base font-bold text-neutral-100">
                  {resultModal.status === "BACKLASH"
                    ? "Trí Đạo Phản Phệ!"
                    : resultModal.is_instant_100
                    ? "✨ Linh Quang Chớp Lóe!"
                    : resultModal.is_deviated
                    ? "Thiên Cơ Rẽ Nhánh!"
                    : "Khai Sáng Thành Công!"}
                </h3>
                <div className="text-xs text-neutral-400 mt-0.5">
                  {resultModal.status === "BACKLASH"
                    ? "Cưỡng cầu thiên cơ bất thành"
                    : resultModal.is_deviated
                    ? "Lĩnh ngộ Cổ liên quan cùng hệ"
                    : "Lĩnh ngộ tàn phương Cổ đích"}
                </div>
              </div>
            </div>

            {/* Message Body */}
            <p className="text-xs text-neutral-300 bg-neutral-950/60 p-3 rounded-xl border border-neutral-800 leading-relaxed">
              {resultModal.message}
            </p>

            {/* Backlash Penalties Detail */}
            {resultModal.backlash && (
              <div className="mt-3 p-3 bg-rose-950/30 border border-rose-900/60 rounded-xl space-y-1.5 text-xs text-rose-300">
                <div className="font-semibold text-rose-200">Tổn thất chịu phạt:</div>
                <div>• Sinh mệnh tiêu giảm: -{resultModal.backlash.hp_lost} HP</div>
                <div>• Chân nguyên hao hụt: -{resultModal.backlash.mana_lost}</div>
                <div>• Niệm Lực bị hút cạn về 0</div>
                {resultModal.backlash.soul_injury_cooldown_minutes > 0 && (
                  <div className="text-amber-300">• Chấn thương hồn phách: Hồi chiêu {resultModal.backlash.soul_injury_cooldown_minutes} phút</div>
                )}
              </div>
            )}

            {/* Recipe Progress Detail if not backlash */}
            {resultModal.status !== "BACKLASH" && resultModal.gu_name && (
              <div className="mt-3 p-3 bg-neutral-950/60 border border-neutral-800 rounded-xl space-y-2">
                <div className="flex justify-between text-xs">
                  <span className="font-medium text-neutral-300">
                    [{resultModal.gu_name}] (Chuyển {resultModal.rank})
                  </span>
                  <span className="font-mono font-bold text-emerald-400">
                    {resultModal.completion_rate}%
                  </span>
                </div>

                <div className="w-full h-2 bg-neutral-800 rounded-full overflow-hidden">
                  <div
                    className={`h-full transition-all duration-500 ${
                      resultModal.completion_rate >= 100 ? "bg-amber-400" : "bg-emerald-500"
                    }`}
                    style={{ width: `${resultModal.completion_rate}%` }}
                  />
                </div>
              </div>
            )}

            {/* Action Buttons */}
            <div className="mt-5 flex items-center justify-end gap-3">
              <button
                onClick={() => setResultModal(null)}
                className="px-4 py-2 bg-neutral-800 hover:bg-neutral-700 text-neutral-200 text-xs font-medium rounded-xl transition-all"
              >
                Tiếp Tục Khai Sáng
              </button>

              {resultModal.status !== "BACKLASH" && resultModal.recipe_id && onSelectRecipeForRefine && (
                <button
                  onClick={() => {
                    const recId = resultModal.recipe_id!;
                    setResultModal(null);
                    onSelectRecipeForRefine(recId);
                  }}
                  className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded-xl flex items-center gap-1.5 transition-all shadow-md shadow-indigo-600/30"
                >
                  <span>Đi Đến Lò Luyện</span>
                  <ArrowRight className="w-3.5 h-3.5" />
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
