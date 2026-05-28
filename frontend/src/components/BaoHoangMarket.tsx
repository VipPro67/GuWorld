"use client";

import { useEffect, useState } from "react";
import { GuInstance, Player, GuSpec, ItemSpec } from "@/store/useGameStore";
import { AlertCircle } from "lucide-react";

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
  guDictionary: Record<string, GuSpec>;
  itemDictionary: Record<string, ItemSpec>;
}

export function BaoHoangMarket({ player, guInventory, materials, actionLoading, listings, onRefresh, onCreate, onBid, onCancel, guDictionary, itemDictionary }: Props) {
  const [tab, setTab] = useState<"browse" | "sell">("browse");
  const [price, setPrice] = useState(100);
  const [durationHours, setDurationHours] = useState(24);
  const [itemType, setItemType] = useState<"Gu" | "Material">("Material");
  const [selectedGu, setSelectedGu] = useState("");
  const [selectedMaterial, setSelectedMaterial] = useState("");
  const [quantity, setQuantity] = useState(1);
  const [currency, setCurrency] = useState<"primeval" | "immortal">("primeval");
  const [bidAmounts, setBidAmounts] = useState<Record<string, number>>({});
  const [message, setMessage] = useState<{ type: "error" | "success"; text: string } | null>(null);

  useEffect(() => {
    onRefresh();
  }, [onRefresh]);

  useEffect(() => {
    if (!message) return;
    const timer = setTimeout(() => setMessage(null), 4000);
    return () => clearTimeout(timer);
  }, [message]);

  const materialKeys = Object.keys(materials);

  const getDisplayName = (itemData: string, itemType: "Gu" | "Material"): string => {
    if (itemType === "Gu") {
      const spec = guDictionary[itemData];
      return spec?.name || itemData.replace(/^GU_/, "").replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
    }
    const spec = itemDictionary[itemData];
    return spec?.name || itemData.replace(/_/g, " ").replace(/\b\w/g, c => c.toUpperCase());
  };

  const createListing = async () => {
    if (actionLoading) return;

    if (itemType === "Gu" && !selectedGu) {
      setMessage({ type: "error", text: "Vui lòng chọn Gu để đăng bán." });
      return;
    }

    if (itemType === "Material" && !selectedMaterial) {
      setMessage({ type: "error", text: "Vui lòng chọn nguyên liệu để đăng bán." });
      return;
    }

    if (quantity < 1) {
      setMessage({ type: "error", text: "Số lượng phải lớn hơn 0." });
      return;
    }

    if (price < 1) {
      setMessage({ type: "error", text: "Giá khởi điểm phải lớn hơn 0." });
      return;
    }

    if (durationHours < 1) {
      setMessage({ type: "error", text: "Thời gian phải lớn hơn 0 giờ." });
      return;
    }

    const ok = await onCreate({
      itemType,
      guInstanceId: itemType === "Gu" ? selectedGu : undefined,
      itemId: itemType === "Material" ? selectedMaterial : undefined,
      quantity,
      price,
      durationHours,
      currency: player.rank >= 6 ? currency : "primeval",
    });

    if (ok) {
      await onRefresh();
      setSelectedGu("");
      setSelectedMaterial("");
      setQuantity(1);
      setMessage({ type: "success", text: "Đã tạo phiên đấu giá thành công." });
      return;
    }

    setMessage({ type: "error", text: "Đăng bán thất bại. Hãy thử lại." });
  };

  const handleBid = async (listingId: string, amount: number, currentPrice: number) => {
    if (actionLoading) return;

    if (amount <= currentPrice) {
      setMessage({ type: "error", text: "Giá đặt phải cao hơn giá hiện tại." });
      return;
    }

    const ok = await onBid(listingId, amount);
    if (ok) {
      await onRefresh();
      setMessage({ type: "success", text: "Đã đặt giá thành công." });
      return;
    }

    setMessage({ type: "error", text: "Đặt giá thất bại. Hãy thử lại." });
  };

  const handleCancel = async (listingId: string) => {
    if (actionLoading) return;

    const ok = await onCancel(listingId);
    if (ok) {
      await onRefresh();
      setMessage({ type: "success", text: "Đã hủy phiên đấu giá." });
      return;
    }

    setMessage({ type: "error", text: "Hủy phiên đấu giá thất bại. Hãy thử lại." });
  };

  return <div className="w-full max-w-5xl mx-auto glass-panel-gold p-6 rounded-2xl border border-amber-900/30">
    <div className="flex items-center justify-between mb-6">
      <h2 className="text-xl font-serif text-amber-300">Bảo Hoàng Thiên</h2>
      <div className="flex gap-2">
        <button onClick={() => setTab("browse")} className={`px-3 py-1 rounded ${tab === "browse" ? "bg-cyan-900/60 text-cyan-300" : "text-zinc-400"}`}>Đấu Giá</button>
        <button onClick={() => setTab("sell")} className={`px-3 py-1 rounded ${tab === "sell" ? "bg-amber-900/60 text-amber-300" : "text-zinc-400"}`}>Đăng Bán</button>
      </div>
    </div>

    {message && (
      <div className={`flex items-start gap-2 p-3 rounded-xl border ${message.type === "error" ? "border-rose-500/40 bg-rose-500/10 text-rose-100" : "border-emerald-500/40 bg-emerald-500/10 text-emerald-100"}`}>
        <AlertCircle className="w-5 h-5 mt-0.5" />
        <div className="text-sm leading-5">{message.text}</div>
      </div>
    )}

    {tab === "sell" ? <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <label className="space-y-2">
          <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Loại hàng</span>
          <select value={itemType} onChange={(e) => setItemType(e.target.value as "Gu" | "Material")} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2">
            <option value="Material">Material</option>
            <option value="Gu">Gu</option>
          </select>
        </label>
        <label className="space-y-2">
          <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Giá khởi điểm</span>
          <input type="number" value={price} onChange={(e) => setPrice(Number(e.target.value))} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2" placeholder="Ví dụ: 100" />
        </label>
      </div>

      {player.rank >= 6 && (
        <label className="space-y-2">
          <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Loại tiền</span>
          <select value={currency} onChange={(e) => setCurrency(e.target.value as "primeval" | "immortal")} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2">
            <option value="primeval">Nguyên Thạch (Primeval)</option>
            <option value="immortal">Bất Tử (Immortal)</option>
          </select>
        </label>
      )}

      {itemType === "Gu" ? (
        <label className="space-y-2 block">
          <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Chọn Gu</span>
          <select value={selectedGu} onChange={(e) => setSelectedGu(e.target.value)} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2">
            <option value="">Chọn Gu</option>
            {guInventory.filter(g => !g.is_vital).map(g => <option key={g.instance_id} value={g.instance_id}>{getDisplayName(g.gu_id, "Gu")} ({g.instance_id.slice(0, 6)})</option>)}
          </select>
        </label>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <label className="space-y-2 block">
            <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Nguyên liệu</span>
            <select value={selectedMaterial} onChange={(e) => setSelectedMaterial(e.target.value)} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2">
              <option value="">Chọn nguyên liệu</option>
              {materialKeys.map((k) => <option key={k} value={k}>{getDisplayName(k, "Material")} (x{materials[k]})</option>)}
            </select>
          </label>
          <label className="space-y-2 block">
            <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Số lượng</span>
            <input type="number" value={quantity} min={1} onChange={(e) => setQuantity(Number(e.target.value))} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2" placeholder="1" />
          </label>
        </div>
      )}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <label className="space-y-2 block">
          <span className="text-xs uppercase text-zinc-500 font-semibold tracking-wider">Thời gian</span>
          <input type="number" min={1} value={durationHours} onChange={(e) => setDurationHours(Number(e.target.value))} className="w-full bg-zinc-900 border border-zinc-700 rounded p-2" placeholder="Số giờ" />
        </label>
        <button disabled={actionLoading} onClick={createListing} className="bg-amber-700 hover:bg-amber-600 rounded p-2 font-semibold">Đăng Bán</button>
      </div>
    </div> : <div className="space-y-3">
      <div className="flex justify-end"><button onClick={onRefresh} className="text-xs text-cyan-400">Làm mới</button></div>
      {listings.map((listing) => <div key={listing.id} className="border border-zinc-800 rounded-xl p-3 flex flex-col md:flex-row md:items-center md:justify-between gap-3">
        <div>
          <div className="text-zinc-100 font-semibold">{listing.item_type}: {getDisplayName(listing.item_data, listing.item_type)} {listing.quantity > 1 ? `x${listing.quantity}` : ""}</div>
          <div className="text-xs text-zinc-500">Giá hiện tại: {listing.current_price} ({listing.currency === "primeval" ? "Nguyên Thạch" : "Bất Tử"}) • Kết thúc: {new Date(listing.end_time).toLocaleString()}</div>
        </div>
        <div className="flex gap-2 items-center">
          {listing.seller_id === player.id ? <button disabled={actionLoading} onClick={() => handleCancel(listing.id)} className="px-3 py-1 rounded bg-red-900/60 text-red-300">Hủy</button> : <>
            <input type="number" className="w-28 bg-zinc-900 border border-zinc-700 rounded p-1" value={bidAmounts[listing.id] ?? listing.current_price + 1} onChange={(e) => setBidAmounts((prev) => ({ ...prev, [listing.id]: Number(e.target.value) }))} placeholder={String(listing.current_price + 1)} />
            <button disabled={actionLoading} onClick={() => handleBid(listing.id, bidAmounts[listing.id] ?? listing.current_price + 1, listing.current_price)} className="px-3 py-1 rounded bg-cyan-900/70 text-cyan-300">Đặt giá</button>
          </>}
        </div>
      </div>)}
      {listings.length === 0 && <p className="text-zinc-500 text-sm">Chưa có phiên đấu giá nào.</p>}
    </div>}
  </div>;
}
