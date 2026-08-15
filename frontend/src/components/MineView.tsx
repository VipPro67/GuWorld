import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Sword, LogOut, Coins, Flame, Gift, Clock, User, ShieldAlert, Loader2, Skull } from "lucide-react";

export const MineView: React.FC = () => {
  const {
    player,
    mineSlots,
    actionLoading,
    fetchMineSlots,
    occupyMineSlot,
    claimMineRewards,
    abandonMineSlot,
    attackMineSlot,
  } = useGameStore();

  const [time, setTime] = useState(Date.now());

  // Fetch slots on mount and set polling interval
  useEffect(() => {
    fetchMineSlots();
    const interval = setInterval(() => {
      fetchMineSlots();
    }, 10000); // Poll every 10s
    return () => clearInterval(interval);
  }, [fetchMineSlots]);

  // Keep a local timer to update the duration strings in real-time
  useEffect(() => {
    const timer = setInterval(() => {
      setTime(Date.now());
    }, 30000); // update every 30s
    return () => clearInterval(timer);
  }, []);

  const formatDuration = (occupiedAtStr: string | undefined): string => {
    if (!occupiedAtStr) return "0 phút";
    const diffMs = time - Date.parse(occupiedAtStr);
    const diffMins = Math.floor(diffMs / 60000);
    if (diffMins < 60) {
      return `${diffMins} phút`;
    }
    const hours = Math.floor(diffMins / 60);
    const mins = diffMins % 60;
    return `${hours} giờ ${mins} phút`;
  };

  const getRarityStyles = (rarity: string) => {
    switch (rarity) {
      case "Siêu Cấp":
        return {
          textColor: "text-amber-300 font-extrabold shadow-amber-900/50",
          glowColor: "shadow-[0_0_25px_rgba(251,191,36,0.5)] border-amber-400/40 bg-amber-950/20",
          crystalFill: "url(#grad-superior)",
          crystalColor: "#fbbf24",
          bgGradient: "from-amber-950/40 via-zinc-900/90 to-zinc-950/90 border-amber-400/30",
        };
      case "Truyền Thuyết":
        return {
          textColor: "text-rose-400 font-extrabold shadow-rose-900/50",
          glowColor: "shadow-[0_0_20px_rgba(244,63,94,0.45)] border-rose-500/30 bg-rose-950/20",
          crystalFill: "url(#grad-legendary)",
          crystalColor: "#f43f5e",
          bgGradient: "from-rose-950/40 via-zinc-900/90 to-zinc-950/90 border-rose-500/20",
        };
      case "Sử Thi":
        return {
          textColor: "text-violet-400 font-bold",
          glowColor: "shadow-[0_0_15px_rgba(139,92,246,0.35)] border-violet-500/30 bg-violet-950/20",
          crystalFill: "url(#grad-epic)",
          crystalColor: "#8b5cf6",
          bgGradient: "from-violet-950/40 via-zinc-900/90 to-zinc-950/90 border-violet-500/20",
        };
      case "Hiếm":
        return {
          textColor: "text-cyan-400 font-semibold",
          glowColor: "shadow-[0_0_12px_rgba(6,182,212,0.3)] border-cyan-500/30 bg-cyan-950/20",
          crystalFill: "url(#grad-rare)",
          crystalColor: "#06b6d4",
          bgGradient: "from-cyan-950/40 via-zinc-900/90 to-zinc-950/90 border-cyan-500/20",
        };
      default: // Thường
        return {
          textColor: "text-zinc-400 font-normal",
          glowColor: "shadow-[0_0_8px_rgba(161,161,170,0.2)] border-zinc-700/30 bg-zinc-900/20",
          crystalFill: "url(#grad-common)",
          crystalColor: "#a1a1aa",
          bgGradient: "from-zinc-800/40 via-zinc-900/90 to-zinc-950/90 border-zinc-800/20",
        };
    }
  };

  const getInitials = (username: string): string => {
    if (!username) return "?";
    return username.slice(0, 2).toUpperCase();
  };

  const getAvatarBg = (username: string): string => {
    let hash = 0;
    for (let i = 0; i < username.length; i++) {
      hash = username.charCodeAt(i) + ((hash << 5) - hash);
    }
    const colors = [
      "from-rose-500 to-red-600",
      "from-violet-500 to-indigo-600",
      "from-cyan-500 to-blue-600",
      "from-emerald-500 to-teal-600",
      "from-amber-500 to-orange-600",
      "from-pink-500 to-fuchsia-600",
    ];
    return colors[Math.abs(hash) % colors.length];
  };

  const userOccupiedSlot = mineSlots.find(
    (slot) => slot.occupier_id === player?.id
  );

  return (
    <div className="w-full mx-auto px-1 sm:px-4 py-2">
      {/* SVG gradients definition */}
      <svg className="absolute w-0 h-0" width="0" height="0">
        <defs>
          <linearGradient id="grad-superior" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#fef3c7" />
            <stop offset="30%" stopColor="#fbbf24" />
            <stop offset="60%" stopColor="#f59e0b" />
            <stop offset="100%" stopColor="#92400e" />
          </linearGradient>
          <linearGradient id="grad-legendary" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#fda4af" />
            <stop offset="50%" stopColor="#f43f5e" />
            <stop offset="100%" stopColor="#9f1239" />
          </linearGradient>
          <linearGradient id="grad-epic" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#ddd6fe" />
            <stop offset="50%" stopColor="#8b5cf6" />
            <stop offset="100%" stopColor="#5b21b6" />
          </linearGradient>
          <linearGradient id="grad-rare" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#c5f2ff" />
            <stop offset="50%" stopColor="#06b6d4" />
            <stop offset="100%" stopColor="#0891b2" />
          </linearGradient>
          <linearGradient id="grad-common" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="#fafafa" />
            <stop offset="50%" stopColor="#d4d4d8" />
            <stop offset="100%" stopColor="#52525b" />
          </linearGradient>
        </defs>
      </svg>

      {/* Header Summary */}
      <div className="bg-zinc-950/60 border border-zinc-900 rounded-3xl p-4 sm:p-6 mb-8 flex flex-col md:flex-row items-center justify-between gap-6 shadow-xl backdrop-blur-md">
        <div className="flex flex-col gap-2">
          <h2 className="text-xl sm:text-2xl font-bold font-serif text-zinc-100 flex items-center gap-2.5">
            <Coins className="w-6 h-6 text-yellow-500 animate-pulse" />
            Mỏ Linh Thạch Vạn Giới
          </h2>
          <p className="text-xs sm:text-sm text-zinc-400">
            Chiếm đóng, bảo vệ các tinh mạch linh thạch để tích lũy Nguyên Thạch và Nhân Khí liên tục theo thời gian.
          </p>
        </div>

        {userOccupiedSlot ? (
          <div className="flex items-center gap-4 bg-zinc-900/60 border border-zinc-800 px-5 py-3 rounded-2xl">
            <div className="flex flex-col">
              <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono">Đang chiếm giữ</span>
              <span className={`text-sm font-serif ${getRarityStyles(userOccupiedSlot.rarity).textColor}`}>
                Mỏ {userOccupiedSlot.rarity} #{userOccupiedSlot.id}
              </span>
            </div>
            <div className="h-8 w-px bg-zinc-800" />
            <div className="flex flex-col text-right">
              <span className="text-[10px] text-zinc-500 uppercase tracking-widest font-mono">Đã giữ</span>
              <span className="text-sm font-semibold text-zinc-300 font-mono">
                {formatDuration(userOccupiedSlot.occupied_at)}
              </span>
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-2 border border-zinc-800 bg-zinc-900/20 px-4 py-2.5 rounded-2xl text-xs sm:text-sm text-zinc-500 font-mono">
            <ShieldAlert className="w-4 h-4 text-amber-500 shrink-0" />
            Đạo hữu chưa chiếm giữ linh mạch nào. Hãy chiếm 1 vị trí!
          </div>
        )}
      </div>

      {/* Grid of Slots */}
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
        {mineSlots.map((slot) => {
          const styles = getRarityStyles(slot.rarity);
          const isOwnSlot = slot.occupier_id === player?.id;
          const isOccupied = !!slot.occupier_id;
          const hasMonster = !!slot.monster_id && !slot.occupier_id;

          return (
            <div
              key={slot.id}
              className={`bg-gradient-to-b ${styles.bgGradient} border rounded-3xl p-5 flex flex-col items-center relative transition-all duration-300 hover:scale-[1.03] hover:shadow-2xl shadow-lg ${styles.glowColor}`}
            >
              {/* Slot ID Badge */}
              <div className="absolute top-3 left-3 bg-zinc-950/70 border border-zinc-800 text-[10px] text-zinc-400 px-2 py-0.5 rounded-full font-mono">
                #{slot.id}
              </div>

              {/* Crystal graphic with circular avatar inside */}
              <div className="w-32 h-32 relative flex items-center justify-center mb-2">
                <svg
                  viewBox="0 0 100 100"
                  className="w-full h-full drop-shadow-[0_4px_8px_rgba(0,0,0,0.5)]"
                >
                  {/* Styled central crystal shape using path */}
                  <path
                    d="M 50 10 L 80 40 L 70 85 L 30 85 L 20 40 Z"
                    fill={styles.crystalFill}
                    stroke={`${styles.crystalColor}33`}
                    strokeWidth="1.5"
                  />
                  <path
                    d="M 50 10 L 50 85"
                    stroke="#ffffff22"
                    strokeWidth="1"
                  />
                  <path
                    d="M 20 40 L 80 40"
                    stroke="#ffffff11"
                    strokeWidth="1"
                  />
                </svg>

                {/* Occupier Avatar in the center of crystal */}
                <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-14 h-14 rounded-full border-2 border-zinc-950/80 shadow-inner overflow-hidden flex items-center justify-center">
                  {hasMonster ? (
                    <div className="w-full h-full bg-red-950/80 flex items-center justify-center">
                      <Skull className="w-7 h-7 text-red-400" />
                    </div>
                  ) : isOccupied ? (
                    <div
                      className={`w-full h-full bg-gradient-to-br ${getAvatarBg(
                        slot.occupier_name || ""
                      )} flex items-center justify-center text-zinc-100 font-bold text-xs`}
                      title={slot.occupier_name}
                    >
                      {getInitials(slot.occupier_name || "")}
                    </div>
                  ) : (
                    <div className="w-full h-full bg-zinc-950/60 border border-dashed border-zinc-800 flex items-center justify-center">
                      <User className="w-5 h-5 text-zinc-700" />
                    </div>
                  )}
                </div>
              </div>

              {/* Rarity & Status */}
              <div className="text-center mb-1">
                <h3 className={`text-base font-serif tracking-wider ${styles.textColor}`}>
                  {slot.rarity}
                </h3>
                <span className="text-[11px] text-zinc-400 font-mono">
                  {hasMonster ? "Yêu Quái canh giữ" : isOccupied ? slot.occupier_name : "Trống"}
                </span>
              </div>

              {/* Occupied Timer */}
              <div className="text-xs text-zinc-500 font-mono mb-4 flex items-center gap-1.5 justify-center">
                <Clock className="w-3.5 h-3.5 text-zinc-600" />
                {hasMonster ? "Có quái thủ giữ" : isOccupied ? formatDuration(slot.occupied_at) : "Đang mở cửa"}
              </div>

              {/* Rewards Indicators */}
              <div className="w-full bg-zinc-950/40 border border-zinc-900 rounded-2xl py-2 px-3 flex items-center justify-around gap-2 mb-5">
                <div className="flex items-center gap-1 text-[10px] text-zinc-300 font-mono" title="Nguyên Thạch">
                  <Coins className="w-3.5 h-3.5 text-yellow-500" />
                  <span>{slot.accumulated_stones}</span>
                </div>
                <div className="w-px h-4 bg-zinc-900" />
                <div className="flex items-center gap-1 text-[10px] text-zinc-300 font-mono" title="Nhân Khí">
                  <Flame className="w-3.5 h-3.5 text-orange-500" />
                  <span>{slot.accumulated_qi.toFixed(0)}</span>
                </div>
                {slot.accumulated_chests > 0 && (
                  <>
                    <div className="w-px h-4 bg-zinc-900" />
                    <div className="flex items-center gap-1 text-[10px] text-zinc-300 font-mono" title="Rương Mỏ Linh Thạch">
                      <Gift className="w-3.5 h-3.5 text-red-500" />
                      <span>{slot.accumulated_chests}</span>
                    </div>
                  </>
                )}
              </div>

              {/* Action Buttons */}
              <div className="w-full mt-auto">
                {actionLoading ? (
                  <button
                    disabled
                    className="w-full bg-zinc-900 border border-zinc-800 text-zinc-500 py-2.5 px-4 rounded-xl flex items-center justify-center gap-2 text-xs font-mono"
                  >
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                    Đang xử lý...
                  </button>
                ) : hasMonster ? (
                  <button
                    onClick={() => attackMineSlot(slot.id)}
                    className="w-full bg-red-950/60 border border-red-500/20 hover:bg-red-900/60 hover:border-red-500/40 text-red-400 hover:text-red-300 py-2.5 px-4 rounded-xl text-xs font-serif font-bold tracking-wider transition-all shadow-md active:scale-95 cursor-pointer flex items-center justify-center gap-1.5"
                  >
                    <Skull className="w-3.5 h-3.5" />
                    Thách Đấu
                  </button>
                ) : !isOccupied ? (
                  <button
                    onClick={() => occupyMineSlot(slot.id)}
                    className="w-full bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 hover:border-zinc-700 text-zinc-200 hover:text-white py-2.5 px-4 rounded-xl text-xs font-serif font-bold tracking-wider transition-all shadow-md active:scale-95 cursor-pointer"
                  >
                    Chiếm Đóng
                  </button>
                ) : isOwnSlot ? (
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      onClick={() => claimMineRewards(slot.id)}
                      className="bg-emerald-950/60 border border-emerald-500/20 hover:bg-emerald-900/60 hover:border-emerald-500/40 text-emerald-400 hover:text-emerald-300 py-2 px-3 rounded-xl text-[11px] font-serif font-bold tracking-wide transition-all active:scale-95 cursor-pointer"
                    >
                      Nhận Thưởng
                    </button>
                    <button
                      onClick={() => abandonMineSlot(slot.id)}
                      className="bg-zinc-900/80 border border-zinc-800 hover:bg-red-950/40 hover:border-red-900/40 text-zinc-400 hover:text-red-400 py-2 px-3 rounded-xl text-[11px] font-serif font-semibold tracking-wide transition-all active:scale-95 cursor-pointer flex items-center justify-center gap-1"
                    >
                      <LogOut className="w-3 h-3" />
                      Rời Mỏ
                    </button>
                  </div>
                ) : (
                  <button
                    onClick={() => attackMineSlot(slot.id)}
                    className="w-full bg-red-950/60 border border-red-500/20 hover:bg-red-900/60 hover:border-red-500/40 text-red-400 hover:text-red-300 py-2.5 px-4 rounded-xl text-xs font-serif font-bold tracking-wider transition-all shadow-md active:scale-95 cursor-pointer flex items-center justify-center gap-1.5"
                  >
                    <Sword className="w-3.5 h-3.5" />
                    Thách Đấu
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
