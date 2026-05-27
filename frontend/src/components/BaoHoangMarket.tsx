"use client";

import { useEffect, useState } from "react";
import { GuInstance, Player } from "@/store/useGameStore";

interface Listing {
  id: string;
  seller_id: string;
  item_type: "Gu" | "Material";
  item_data: string;
  quantity: number;
  currency: "primeval" | "immortal";
  start_price: number;
  current_price: number;
  highest_bidder_id: string;
  end_time: string;
}

interface Props {
  player: Player;
  guInventory: GuInstance[];
  materials: Record<string, number>;
  actionLoading: boolean;
  listings: Listing[];
  onRefresh: () => Promise<void>;
  onCreate: (payload: { itemType: "Gu" | "Material"; guInstanceId?: string; itemId?: string; quantity: number; price: number; durationHours: number; currency: "primeval" | "immortal" }) => Promise<boolean>;
  onBid: (listingId: string, amount: number) => Promise<boolean>;
  onCancel: (listingId: string) => Promise<boolean>;
}

export function BaoHoangMarket({ player, guInventory, materials, actionLoading, listings, onRefresh, onCreate, onBid, onCancel }: Props) {
  const [tab, setTab] = useState<"browse" | "sell">("browse");
  const [price, setPrice] = useState(100);
  const [durationHours, setDurationHours] = useState(24);
  const [itemType, setItemType] = useState<"Gu" | "Material">("Material");
  const [selectedGu, setSelectedGu] = useState("");
  const [selectedMaterial, setSelectedMaterial] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [bidAmounts, setBidAmounts] = useState<Record<string, number>>({});

  useEffect(() => { onRefresh(); }, [onRefresh]);

  const materialKeys = Object.keys(materials);

  const createListing = async () => {
    const ok = await onCreate({
      itemType,
      guInstanceId: itemType === "Gu" ? selectedGu : undefined,
      itemId: itemType === "Material" ? selectedMaterial : undefined,
      quantity,
      price,
      durationHours,
      currency: "primeval",
    });
    if (ok) {
      await onRefresh();
      setSelectedGu("");
      setSelectedMaterial("");
    }
  };

  return <div className="w-full max-w-5xl mx-auto glass-panel-gold p-6 rounded-2xl border border-amber-900/30">
    <div className="flex items-center justify-between mb-6">
      <h2 className="text-xl font-serif text-amber-300">Bảo Hoàng Thiên</h2>
      <div className="flex gap-2">
        <button onClick={() => setTab("browse")} className={`px-3 py-1 rounded ${tab === "browse" ? "bg-cyan-900/60 text-cyan-300" : "text-zinc-400"}`}>Đấu Giá</button>
        <button onClick={() => setTab("sell")} className={`px-3 py-1 rounded ${tab === "sell" ? "bg-amber-900/60 text-amber-300" : "text-zinc-400"}`}>Đăng Bán</button>
      </div>
    </div>

    {tab === "sell" ? <div className="space-y-3">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <select value={itemType} onChange={(e) => setItemType(e.target.value as "Gu" | "Material")} className="bg-zinc-900 border border-zinc-700 rounded p-2">
          <option value="Material">Material</option>
          <option value="Gu">Gu</option>
        </select>
        <input type="number" value={price} onChange={(e) => setPrice(Number(e.target.value))} className="bg-zinc-900 border border-zinc-700 rounded p-2" placeholder="Giá khởi điểm" />
      </div>
      {itemType === "Gu" ? (
        <select value={selectedGu} onChange={(e) => setSelectedGu(e.target.value)} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2">
          <option value="">Chọn Gu</option>
          {guInventory.filter(g => !g.is_vital).map(g => <option key={g.instance_id} value={g.instance_id}>{g.gu_id} ({g.instance_id.slice(0, 6)})</option>)}
        </select>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <select value={selectedMaterial} onChange={(e) => setSelectedMaterial(e.target.value)} className="bg-zinc-900 border border-zinc-700 rounded p-2">
            <option value="">Chọn material</option>
            {materialKeys.map((k) => <option key={k} value={k}>{k} (x{materials[k]})</option>)}
          </select>
          <input type="number" value={quantity} min={1} onChange={(e) => setQuantity(Number(e.target.value))} className="bg-zinc-900 border border-zinc-700 rounded p-2" />
        </div>
      )}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <input type="number" min={1} value={durationHours} onChange={(e) => setDurationHours(Number(e.target.value))} className="bg-zinc-900 border border-zinc-700 rounded p-2" placeholder="Thời gian giờ" />
        <button disabled={actionLoading} onClick={createListing} className="bg-amber-700 hover:bg-amber-600 rounded p-2 font-semibold">Đăng Bán</button>
      </div>
    </div> : <div className="space-y-3">
      <div className="flex justify-end"><button onClick={onRefresh} className="text-xs text-cyan-400">Làm mới</button></div>
      {listings.map((listing) => <div key={listing.id} className="border border-zinc-800 rounded-xl p-3 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <div className="text-zinc-100 font-semibold">{listing.item_type}: {listing.item_data} {listing.quantity > 1 ? `x${listing.quantity}` : ""}</div>
          <div className="text-xs text-zinc-500">Giá hiện tại: {listing.current_price} • Kết thúc: {new Date(listing.end_time).toLocaleString()}</div>
        </div>
        <div className="flex gap-2 items-center">
          {listing.seller_id === player.id ? <button disabled={actionLoading} onClick={() => onCancel(listing.id)} className="px-3 py-1 rounded bg-red-900/60 text-red-300">Hủy</button> : <>
            <input type="number" className="w-28 bg-zinc-900 border border-zinc-700 rounded p-1" value={bidAmounts[listing.id] ?? listing.current_price + 1} onChange={(e) => setBidAmounts((prev) => ({ ...prev, [listing.id]: Number(e.target.value) }))} />
            <button disabled={actionLoading} onClick={() => onBid(listing.id, bidAmounts[listing.id] ?? listing.current_price + 1)} className="px-3 py-1 rounded bg-cyan-900/70 text-cyan-300">Đặt giá</button>
          </>}
        </div>
      </div>)}
      {listings.length === 0 && <p className="text-zinc-500 text-sm">Chưa có phiên đấu giá nào.</p>}
    </div>}
  </div>;
}
