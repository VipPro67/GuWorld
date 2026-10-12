"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { Coins, Sparkles, Package, RefreshCw, ArrowLeftRight, Store } from "lucide-react";
import { GamePanel, GameButton, GameTabs } from "@/components/ui";
import { useGameStore, Player, formatRecipeName, guSpecFor } from "@/store/useGameStore";
import type { MerchantQuoteItem, MerchantSellLine, MerchantView } from "@/store/slices/inventorySlice";

interface Candidate {
  item_id: string;
  item_type: "Gu" | "Material" | "Recipe";
  rank: number;
  name: string;
  owned: number;
}

interface MerchantTradeViewProps {
  player: Player;
  actionLoading: boolean;
}

const tierLabel = (rank: number) => (rank >= 6 ? "Tiên Nguyên Thạch" : "Nguyên Thạch");

export const MerchantTradeView: React.FC<MerchantTradeViewProps> = ({ player, actionLoading }) => {
  const {
    merchants,
    fetchMerchants,
    quoteMerchant,
    sellToMerchant,
    buyFromMerchant,
    guDictionary,
    itemDictionary,
  } = useGameStore();

  const [selectedID, setSelectedID] = useState<string>("");
  const [tab, setTab] = useState<"buy" | "sell">("sell");
  const [quotes, setQuotes] = useState<Record<string, MerchantQuoteItem>>({});
  const [buyQty, setBuyQty] = useState<Record<string, number>>({});

  const region = player.current_region;

  useEffect(() => {
    void fetchMerchants();
  }, [fetchMerchants, region]);

  const selected: MerchantView | undefined = useMemo(
    () => merchants.find((m) => m.id === selectedID) ?? merchants[0],
    [merchants, selectedID],
  );

  // Sellable candidates from the player's inventory.
  const candidates: Candidate[] = useMemo(() => {
    const out: Candidate[] = [];
    for (const [id, qty] of Object.entries(player.materials_inventory || {})) {
      if (qty > 0) out.push({ item_id: id, item_type: "Material", rank: itemDictionary[id]?.rank || 1, name: itemDictionary[id]?.name || id, owned: qty });
    }
    const guAgg = new Map<string, { rank: number; count: number }>();
    for (const gu of player.gu_inventory || []) {
      if (gu.is_vital || gu.equipped || gu.status === "DEAD") continue;
      const key = `${gu.gu_id}@${gu.rank}`;
      const cur = guAgg.get(key) || { rank: gu.rank, count: 0 };
      cur.count += 1;
      guAgg.set(key, cur);
    }
    for (const [key, agg] of guAgg) {
      const guId = key.split("@")[0];
      const spec = guSpecFor(guId, agg.rank) || guDictionary[guId];
      out.push({ item_id: guId, item_type: "Gu", rank: agg.rank, name: spec?.name || guId, owned: agg.count });
    }
    for (const recipe of player.unlocked_recipes || []) {
      const rank = Number(recipe.recipe_id.match(/_(\d+)_\d+$/)?.[1]) || 1;
      out.push({ item_id: recipe.recipe_id, item_type: "Recipe", rank, name: formatRecipeName(recipe.recipe_id), owned: 1 });
    }
    return out;
  }, [player.materials_inventory, player.gu_inventory, player.unlocked_recipes, itemDictionary, guDictionary]);

  const refreshQuote = useCallback(async () => {
    if (!selected || candidates.length === 0) {
      setQuotes({});
      return;
    }
    const lines: MerchantSellLine[] = candidates.map((c) => ({ item_id: c.item_id, item_type: c.item_type, quantity: c.owned, rank: c.rank }));
    const result = await quoteMerchant(selected.id, lines);
    const map: Record<string, MerchantQuoteItem> = {};
    for (const q of result) map[q.item_id] = q;
    setQuotes(map);
  }, [selected, candidates, quoteMerchant]);

  useEffect(() => {
    void refreshQuote();
  }, [refreshQuote]);

  const handleSell = async (c: Candidate) => {
    if (!selected) return;
    const q = quotes[c.item_id];
    let qty = c.owned;
    if (q && q.quota_left >= 0) qty = Math.min(qty, q.quota_left);
    if (qty <= 0) return;
    const ok = await sellToMerchant(selected.id, [{ item_id: c.item_id, item_type: c.item_type, quantity: qty, rank: c.rank }]);
    if (ok) {
      await fetchMerchants();
      void refreshQuote();
    }
  };

  const handleBuy = async (itemID: string, rank: number) => {
    if (!selected) return;
    const qty = buyQty[`${itemID}@${rank}`] || 1;
    const ok = await buyFromMerchant(selected.id, [{ item_id: itemID, rank, quantity: qty }]);
    if (ok) await fetchMerchants();
  };

  return (
    <GamePanel variant="gold" className="rounded-3xl p-3 sm:p-6 w-full mx-auto space-y-5 animate-fade-in">
      <div className="flex items-center gap-2 border-b border-zinc-800 pb-3">
        <Store className="w-5 h-5 text-amber-500" />
        <h3 className="text-lg font-bold text-zinc-100 font-serif">Thương Nhân Địa Phương</h3>
        <span className="ml-auto text-[10px] font-mono text-zinc-500">{region}</span>
      </div>

      {merchants.length === 0 ? (
        <div className="py-10 flex flex-col items-center gap-2 text-zinc-600">
          <Store className="w-8 h-8 text-zinc-700" />
          <span className="text-xs font-mono">Khu vực này không có thương nhân. Hãy tới thành trấn/thôn làng.</span>
        </div>
      ) : (
        <>
          {merchants.length > 1 && (
            <div className="flex flex-wrap gap-2">
              {merchants.map((m) => (
                <button
                  key={m.id}
                  onClick={() => setSelectedID(m.id)}
                  className={`px-3 py-1.5 text-xs font-mono rounded-lg border transition-all cursor-pointer ${
                    selected?.id === m.id ? "bg-amber-500/10 text-amber-400 border-amber-500/30" : "text-zinc-400 border-zinc-800 hover:text-zinc-200"
                  }`}
                >
                  {m.name}
                </button>
              ))}
            </div>
          )}

          {selected && (
            <>
              <div className="text-[11px] text-zinc-400 leading-relaxed">
                {selected.description}
                <div className="mt-1 text-[10px] font-mono text-zinc-500">
                  Mua vào {Math.round(selected.buy_ratio * 100)}% → sàn {Math.round(selected.buy_floor_ratio * 100)}% · quota {selected.quota_per_item}/{selected.quota_period === "HOUR" ? "giờ" : "ngày"} (toàn server)
                </div>
              </div>

              <GameTabs
                items={[
                  { id: "sell", label: "Bán Cho Họ", icon: <ArrowLeftRight className="w-3.5 h-3.5" />, disabled: false },
                  { id: "buy", label: "Mua Của Họ", icon: <Package className="w-3.5 h-3.5" />, disabled: false },
                ]}
                activeId={tab}
                onChange={(id) => setTab(id as "buy" | "sell")}
                variant="pills"
                size="sm"
              />

              {tab === "sell" ? (
                candidates.length === 0 ? (
                  <div className="py-8 text-center text-xs font-mono text-zinc-600">Không có gì để bán.</div>
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {candidates.map((c) => {
                      const q = quotes[c.item_id];
                      const accepted = q?.accepted;
                      const sellable = accepted ? (q!.quota_left < 0 ? c.owned : Math.min(c.owned, q!.quota_left)) : 0;
                      return (
                        <div key={`${c.item_type}-${c.item_id}-${c.rank}`} className="bg-zinc-950/40 border border-zinc-900 rounded-xl p-3 flex flex-col gap-2">
                          <div className="flex items-center justify-between gap-2">
                            <span className="text-xs font-bold text-zinc-200 truncate">{c.name}</span>
                            <span className="text-[10px] font-mono text-zinc-500">x{c.owned}</span>
                          </div>
                          {q && accepted ? (
                            <div className="flex items-center justify-between text-[11px] font-mono">
                              <span className={c.rank >= 6 ? "text-violet-400" : "text-amber-400"}>
                                {c.rank >= 6 ? <Sparkles className="w-3 h-3 inline" /> : <Coins className="w-3 h-3 inline" />} {q.unit_price} {tierLabel(c.rank)}
                              </span>
                              <span className="text-zinc-500">còn nhận {q.quota_left < 0 ? "∞" : q.quota_left}</span>
                            </div>
                          ) : (
                            <span className="text-[10px] font-mono text-zinc-600">{q ? "Không thu mua loại này" : "Đang định giá..."}</span>
                          )}
                          <GameButton
                            size="sm"
                            variant="warning"
                            disabled={actionLoading || !accepted || sellable <= 0}
                            onClick={() => void handleSell(c)}
                          >
                            Bán {accepted && sellable > 0 ? `x${sellable}` : ""}
                          </GameButton>
                        </div>
                      );
                    })}
                  </div>
                )
              ) : selected.stock.length === 0 ? (
                <div className="py-8 text-center text-xs font-mono text-zinc-600">Thương nhân này không bán hàng.</div>
              ) : (
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                  {selected.stock.map((it) => {
                    const key = `${it.item_id}@${it.rank}`;
                    const qty = buyQty[key] || 1;
                    return (
                      <div key={key} className="bg-zinc-950/40 border border-zinc-900 rounded-xl p-3 flex flex-col gap-2">
                        <div className="flex items-center justify-between gap-2">
                          <span className="text-xs font-bold text-zinc-200 truncate">{it.name}</span>
                          <span className="text-[10px] font-mono text-zinc-500">Hạng {it.rank}</span>
                        </div>
                        <div className="flex items-center justify-between text-[11px] font-mono">
                          <span className="text-amber-400 flex items-center gap-1">
                            {it.rank >= 6 ? <Sparkles className="w-3 h-3" /> : <Coins className="w-3 h-3" />}
                            {it.price}
                          </span>
                          <span className="text-zinc-500">kho {it.stock < 0 ? "∞" : it.stock}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <input
                            type="number"
                            min={1}
                            value={qty}
                            onChange={(e) => setBuyQty((p) => ({ ...p, [key]: Math.max(1, parseInt(e.target.value, 10) || 1) }))}
                            className="w-16 bg-zinc-900/70 border border-zinc-800 rounded-lg px-2 py-1 text-xs font-mono text-zinc-200 focus:outline-none"
                          />
                          <GameButton size="sm" variant="warning" disabled={actionLoading} onClick={() => void handleBuy(it.item_id, it.rank)}>
                            Mua
                          </GameButton>
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}

              <button
                onClick={() => { void fetchMerchants(); void refreshQuote(); }}
                className="flex items-center gap-1 text-[10px] font-mono text-zinc-500 hover:text-zinc-300 transition-colors cursor-pointer"
              >
                <RefreshCw className="w-3 h-3" /> Làm mới
              </button>
            </>
          )}
        </>
      )}
    </GamePanel>
  );
};

export default MerchantTradeView;
