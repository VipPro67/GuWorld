import React, { useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { ShoppingCart } from "lucide-react";

export const ArenaShopView: React.FC = () => {
  const { arenaShopItems, fetchArenaShop, buyArenaItem, actionLoading } = useGameStore();

  useEffect(() => {
    fetchArenaShop();
  }, []);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
        {arenaShopItems.map((item: any) => (
          <div key={item.id} className="bg-zinc-950/60 border border-zinc-800 rounded-2xl p-4 flex flex-col justify-between items-start hover:border-amber-900/50 transition-all group">
            <div>
              <h4 className="text-amber-500 font-bold font-serif text-sm group-hover:text-amber-400">{item.name}</h4>
              <p className="text-xs text-zinc-500 font-mono mt-1">Sổ lượng: {item.qty}</p>
            </div>
            
            <div className="mt-4 w-full flex items-center justify-between">
              <span className="text-xs font-mono font-bold text-amber-500 flex items-center gap-1">
                {item.price} <span className="text-[10px] text-zinc-500 uppercase tracking-widest">Xu</span>
              </span>
              <button
                disabled={actionLoading}
                onClick={() => buyArenaItem(item.id)}
                className="bg-zinc-900 hover:bg-amber-950/60 border border-zinc-800 hover:border-amber-900 text-zinc-300 hover:text-amber-500 px-3 py-1.5 rounded-xl text-xs font-bold font-mono transition-all flex items-center gap-1.5 cursor-pointer disabled:opacity-50"
              >
                <ShoppingCart className="w-3.5 h-3.5" />
                Đổi
              </button>
            </div>
          </div>
        ))}
        {arenaShopItems.length === 0 && (
          <div className="col-span-full text-center py-8 text-zinc-500 font-mono text-xs italic border border-zinc-900/50 rounded-2xl bg-zinc-950/20">
            Chưa có vật phẩm nào trong shop...
          </div>
        )}
      </div>
    </div>
  );
};
