import React, { useEffect, useState, useRef } from "react";
import { estimateManaFromSnapshot, estimateHPFromSnapshot, useGameStore } from "@/store/useGameStore";
import {
  User,
  Briefcase,
  Flame,
  Store,
  Gem,
  Compass,
  Coins,
  Swords,
  Lock,
  Send,
  Loader2,
  Sparkles,
  Crown,
  Sword,
  History,
  MessageSquare,
  X,
} from "lucide-react";
import { LogConsole } from "./LogConsole";

interface HubItem {
  id: string;
  label: string;
  icon: React.ComponentType<any>;
  gradient: string;
  glow: string;
  borderColor: string;
  iconColor: string;
  locked?: boolean;
}

const formatRelativeTime = (createdAtStr: string): string => {
  if (!createdAtStr) return "";
  const date = new Date(createdAtStr);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.max(0, Math.floor(diffMs / 1000));
  
  if (diffSec < 60) return "vừa xong";
  
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}p trước`;
  
  const diffHour = Math.floor(diffMin / 60);
  if (diffHour < 24) return `${diffHour}g trước`;
  
  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay} ngày trước`;
};

export const LobbyView: React.FC = () => {
  const {
    player,
    manaSnapshot,
    logs,
    setView,
    baoHoangMessages,
    sendBaoHoangChat,
    actionLoading,
    fetchPlayerProfile,
    mySectStatus,
    addToast,
    activeChannel,
    setActiveChannel,
    chatOpen,
    setChatOpen,
    unreadChannels,
    subscribeAllBaoHoangMessages,
  } = useGameStore();
  const [chatInput, setChatInput] = useState("");
  const [manaNow, setManaNow] = useState(() => Date.now());

  useEffect(() => {
    if (!player || !manaSnapshot) return;

    const isManaFull = player.rank >= 6 ? false : (manaSnapshot.current_mana >= manaSnapshot.max_mana);
    const isHPFull = manaSnapshot.hp >= manaSnapshot.base_hp;

    if (isManaFull && isHPFull) return;

    const interval = setInterval(() => {
      setManaNow(Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, [player, manaSnapshot]);

  const displayedMana = player
    ? estimateManaFromSnapshot(manaSnapshot?.player_id === player.id ? manaSnapshot : null, manaNow) ?? player.current_mana
    : 0;
  const displayedMaxMana = player?.max_mana || 0;
  const displayedManaPercent = displayedMaxMana > 0 ? Math.min(100, (displayedMana / displayedMaxMana) * 100) : 0;

  const displayedHP = player
    ? estimateHPFromSnapshot(manaSnapshot?.player_id === player.id ? manaSnapshot : null, manaNow) ?? player.hp
    : 0;
  const displayedMaxHP = player?.base_hp || 0;
  const displayedHPPercent = displayedMaxHP > 0 ? Math.min(100, (displayedHP / displayedMaxHP) * 100) : 0;

  // Management items (Removed "Truyền Tin" as requested)
  const managementItems: HubItem[] = [
    {
      id: "character",
      label: "Nhân Vật",
      icon: User,
      gradient: "from-emerald-500 to-teal-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(16,185,129,0.4)]",
      borderColor: "border-emerald-400/20",
      iconColor: "text-emerald-100",
    },
    {
      id: "inventory",
      label: "Túi Đồ",
      icon: Briefcase,
      gradient: "from-cyan-500 to-blue-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(6,182,212,0.4)]",
      borderColor: "border-cyan-400/20",
      iconColor: "text-cyan-100",
    },
    {
      id: "refine",
      label: "Luyện Cổ",
      icon: Flame,
      gradient: "from-amber-500 to-orange-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(245,158,11,0.4)]",
      borderColor: "border-amber-400/20",
      iconColor: "text-amber-100",
    },
    // {
    //   id: "kill-moves",
    //   label: "Sát Chiêu",
    //   icon: Sword,
    //   gradient: "from-rose-500 to-red-600",
    //   glow: "group-hover:shadow-[0_0_20px_rgba(239,68,68,0.4)]",
    //   borderColor: "border-rose-400/20",
    //   iconColor: "text-rose-100",
    // },
    // {
    //   id: "premium-store",
    //   label: "Cát Bảo Các",
    //   icon: Sparkles,
    //   gradient: "from-violet-500 to-emerald-600",
    //   glow: "group-hover:shadow-[0_0_20px_rgba(139,92,246,0.4)]",
    //   borderColor: "border-violet-400/20",
    //   iconColor: "text-violet-100",
    // },
    {
      id: "shop",
      label: "Cửa Hàng",
      icon: Store,
      gradient: "from-yellow-400 to-amber-500",
      glow: "group-hover:shadow-[0_0_20px_rgba(234,179,8,0.4)]",
      borderColor: "border-yellow-400/20",
      iconColor: "text-yellow-100",
    },
    {
      id: "auction",
      label: "Chợ, Đấu Giá",
      icon: Gem,
      gradient: "from-violet-500 to-purple-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(139,92,246,0.4)]",
      borderColor: "border-violet-400/20",
      iconColor: "text-violet-100",
    },
    {
      id: "feedback",
      label: "Thỉnh Thiên Đạo",
      icon: Sparkles,
      gradient: "from-fuchsia-500 to-pink-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(217,70,239,0.4)]",
      borderColor: "border-fuchsia-400/20",
      iconColor: "text-fuchsia-100",
    },
    {
      id: "combat-history",
      label: "Lịch Sử Chiến Đấu",
      icon: History,
      gradient: "from-rose-500 to-red-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(239,68,68,0.4)]",
      borderColor: "border-rose-400/20",
      iconColor: "text-rose-100",
    },
    {
      id: "audit-logs",
      label: "Ký Sự Tu Hành",
      icon: History,
      gradient: "from-blue-500 to-indigo-600",
      glow: "group-hover:shadow-[0_0_20px_rgba(59,130,246,0.4)]",
      borderColor: "border-blue-400/20",
      iconColor: "text-blue-100",
    },
  ];

  const activityItems: HubItem[] = [
    {
      id: "explore",
      label: "Khám Phá",
      icon: Compass,
      gradient: "from-teal-500 to-emerald-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(20,184,166,0.45)]",
      borderColor: "border-teal-400/30",
      iconColor: "text-teal-500",
    },
    {
      id: "mine",
      label: "Mỏ Linh Thạch",
      icon: Coins,
      gradient: "from-yellow-500 to-amber-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(234,179,8,0.45)]",
      borderColor: "border-yellow-400/30",
      iconColor: "text-yellow-500",
    },
    {
      id: "arena",
      label: "Võ Đài",
      icon: Swords,
      gradient: "from-rose-500 to-red-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(239,68,68,0.45)]",
      borderColor: "border-rose-400/30",
      iconColor: "text-rose-500",
    },
    {
      id: "sect",
      label: "Tông Môn",
      icon: Crown,
      gradient: "from-blue-600 to-indigo-700",
      glow: "group-hover:shadow-[0_0_25px_rgba(37,99,235,0.45)]",
      borderColor: "border-blue-400/30",
      iconColor: "text-blue-400",
      locked: false,
    },
    {
      id: "dungeon",
      label: "Phó Bản",
      icon: Compass,
      gradient: "from-purple-500 to-indigo-600",
      glow: "group-hover:shadow-[0_0_25px_rgba(139,92,246,0.45)]",
      borderColor: "border-purple-400/30",
      iconColor: "text-purple-400",
      locked: false,
    },
    {
      id: "beasts",
      label: "Thần Thú",
      icon: Lock,
      gradient: "from-zinc-800 to-zinc-950",
      glow: "",
      borderColor: "border-zinc-800",
      iconColor: "text-zinc-600",
      locked: true,
    },
    {
      id: "aperture",
      label: "Tiên Khiếu",
      icon: Sparkles,
      gradient: player && player.rank >= 6 ? "from-indigo-500 to-purple-600" : "from-zinc-800 to-zinc-950",
      glow: player && player.rank >= 6 ? "group-hover:shadow-[0_0_25px_rgba(99,102,241,0.45)]" : "",
      borderColor: player && player.rank >= 6 ? "border-indigo-400/30" : "border-zinc-800",
      iconColor: player && player.rank >= 6 ? "text-indigo-400" : "text-zinc-600",
      locked: !player || player.rank < 6,
    },
  ];

  useEffect(() => {
    subscribeAllBaoHoangMessages();
  }, [subscribeAllBaoHoangMessages]);

  const chatContainerRef = useRef<HTMLDivElement>(null);

  // Scroll chat to bottom on new messages
  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [baoHoangMessages]);

  const handleItemClick = (item: HubItem) => {
    if (item.locked) return;
    setView(item.id);
  };

  const handleSendChat = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!chatInput.trim()) return;
    const success = await sendBaoHoangChat(chatInput.trim(), activeChannel);
    if (success) {
      setChatInput("");
    }
  };

  const getRankName = (rank: number, sub: number): string => {
    const ranks = ["Sơ giai", "Trung giai", "Cao giai", "Đỉnh phong"];
    const rankNames = [
      "Nhất Chuyển",
      "Nhị Chuyển",
      "Tam Chuyển",
      "Tứ Chuyển",
      "Ngũ Chuyển",
      "Lục Tiên",
      "Thất Tiên",
      "Bát Tiên",
      "Cửu Tôn",
    ];
    const rName = rankNames[rank - 1] || `${rank} Chuyển`;
    const sName = ranks[sub - 1] || `${sub} Giai`;
    return `${rName} • ${sName}`;
  };

  const getInitials = (username: string): string => {
    if (!username) return "?";
    return username.slice(0, 2).toUpperCase();
  };

  return (
    <div className="w-full max-w-4xl mx-auto px-2 sm:px-4 py-2 sm:py-4 flex flex-col gap-4 sm:gap-8">
      {/* 1. CHARACTER & RESOURCE HUD */}
      {player && (
        <div className="bg-zinc-950/60 border border-zinc-900 rounded-2xl sm:rounded-3xl p-3 sm:p-5 flex flex-col md:flex-row items-center justify-between gap-3 sm:gap-6 shadow-xl backdrop-blur-md">
          {/* Left Block: Avatar & Name & Tu vi */}
          <div className="flex items-center gap-3 sm:gap-4 w-full md:w-auto">
            <div className="w-12 h-12 sm:w-14 sm:h-14 rounded-full bg-gradient-to-br from-emerald-500 to-teal-600 flex items-center justify-center text-zinc-100 font-bold border-2 border-zinc-900 shrink-0 overflow-hidden">
              {player.avatar_url ? (
                <img src={player.avatar_url} alt={player.username} className="w-full h-full object-cover" />
              ) : (
                getInitials(player.username)
              )}
            </div>
            <div className="flex flex-col min-w-0">
              <h2 className="text-base sm:text-lg font-serif font-bold text-zinc-100 truncate">
                {player.username}
              </h2>
              <span className="text-xs text-zinc-500 font-mono font-semibold">
                {getRankName(player.rank, player.sub_rank)}
              </span>
            </div>
          </div>

          {/* Center Block: HP & Mana Bars */}
          <div className="flex flex-col gap-2.5 w-full md:w-80 grow max-w-md">
            {/* HP Bar */}
            <div className="flex items-center justify-between text-[10px] font-mono text-zinc-400">
              <span>Máu</span>
              <span>
                {Math.round(displayedHP)} / {Math.round(displayedMaxHP)}
              </span>
            </div>
            <div className="w-full bg-zinc-950 rounded-full h-2.5 border border-zinc-900 overflow-hidden">
              <div
                className="bg-gradient-to-r from-red-600 to-rose-500 h-full rounded-full transition-all duration-300"
                style={{ width: `${displayedHPPercent}%` }}
              />
            </div>

            {/* Mana Bar */}
            <div className="flex items-center justify-between text-[10px] font-mono text-zinc-400">
              <span>{player.rank >= 6 ? "Tiên Nguyên" : "Chân Nguyên"}</span>
              <span>
                {player.rank >= 6
                  ? `${Math.round(displayedMana)}`
                  : `${Math.round(displayedMana)} / ${Math.round(displayedMaxMana)}`}
              </span>
            </div>
            <div className="w-full bg-zinc-950 rounded-full h-2.5 border border-zinc-900 overflow-hidden">
              <div
                className="bg-gradient-to-r from-cyan-600 to-blue-500 h-full rounded-full transition-all duration-300"
                style={{ width: `${player.rank >= 6 ? 100 : displayedManaPercent}%` }}
              />
            </div>
          </div>
          {/* Right Block: Resources */}
          <div className="flex justify-around md:justify-end items-center gap-4 sm:gap-6 w-full md:w-auto border-t md:border-t-0 border-zinc-900/60 pt-3 md:pt-0 shrink-0">
            {/* Primeval Stones */}
            <div className="flex flex-col items-center">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest">Nguyên Thạch</span>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Coins className="w-4 h-4 text-yellow-500" />
                <span className="text-sm font-bold text-yellow-500/90 font-mono">
                  {player.primeval_stones?.toLocaleString()}
                </span>
              </div>
            </div>

            {/* Immortal Stones */}
            <div className="flex flex-col items-center">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest">Tiên Nguyên</span>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Sparkles className="w-4 h-4 text-violet-400" />
                <span className="text-sm font-bold text-purple-400 font-mono">
                  {player.immortal_stones.toLocaleString()}
                </span>
              </div>
            </div>

            {/* Human Qi */}
            <div className="flex flex-col items-center">
              <span className="text-[10px] text-zinc-500 font-mono uppercase tracking-widest">Nhân Khí</span>
              <div className="flex items-center gap-1.5 mt-0.5">
                <Flame className="w-4 h-4 text-orange-500" />
                <span className="text-sm font-bold text-orange-500 font-mono">
                  {player.human_qi?.toFixed(0)}
                </span>
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="flex sm:gap-6 w-full">
        <LogConsole logs={logs} />
      </div>

      {/* 2. Category: CHỨC NĂNG */}
      <div className="flex flex-col gap-3 sm:gap-5">
        <div className="flex items-center justify-between border-b border-zinc-900/60 pb-2">
          <h2 className="text-xs uppercase tracking-widest text-zinc-500 font-mono font-semibold">
            Chức năng chính
          </h2>
          <span className="text-[10px] text-zinc-600 font-mono">
            {managementItems.length} chức năng
          </span>
        </div>

        <div className="grid grid-cols-3 sm:grid-cols-5 gap-1.5 sm:gap-6 justify-center">
          {managementItems.map((item) => {
            const Icon = item.icon;
            return (
              <div
                key={item.id}
                onClick={() => handleItemClick(item)}
                className="flex flex-col items-center p-2 rounded-2xl transition-all duration-300 hover:bg-zinc-900/20 cursor-pointer group select-none"
              >
                <div
                  className={`w-16 h-16 sm:w-20 sm:h-20 rounded-full flex items-center justify-center bg-gradient-to-br ${item.gradient} border ${item.borderColor} transition-all duration-300 group-hover:scale-110 shadow-lg active:scale-95 relative ${item.glow}`}
                >
                  <Icon className={`w-7 h-7 sm:w-9 sm:h-9 ${item.iconColor} drop-shadow-[0_2px_4px_rgba(0,0,0,0.4)]`} />
                </div>
                <span className="font-serif text-[11px] sm:text-xs font-semibold text-zinc-400 mt-3 tracking-wide text-center group-hover:text-zinc-100 transition-colors">
                  {item.label}
                </span>
              </div>
            );
          })}
        </div>
      </div>

      {/* 3. Category: HOẠT ĐỘNG */}
      <div className="flex flex-col gap-3 sm:gap-5">
        <div className="flex items-center justify-between border-b border-zinc-900/60 pb-2">
          <h2 className="text-xs uppercase tracking-widest text-zinc-500 font-mono font-semibold">
            Hoạt động thế giới
          </h2>
          <span className="text-[10px] text-zinc-600 font-mono">
            {activityItems.length} hoạt động
          </span>
        </div>

        <div className="grid grid-cols-3 sm:grid-cols-6 gap-1.5 sm:gap-6 justify-center">
          {activityItems.map((item) => {
            const Icon = item.icon;
            return (
              <div
                key={item.id}
                onClick={() => handleItemClick(item)}
                className={`flex flex-col items-center p-2 rounded-2xl transition-all duration-300 select-none ${
                  item.locked
                    ? "opacity-50 cursor-not-allowed"
                    : "hover:bg-zinc-900/20 cursor-pointer group"
                }`}
              >
                <div
                  className={`w-16 h-16 sm:w-20 sm:h-20 rounded-full flex items-center justify-center ${
                    item.locked
                      ? "bg-zinc-950 border border-zinc-900"
                      : "bg-zinc-950 border border-zinc-900 hover:border-zinc-800 transition-all duration-300 group-hover:scale-110 shadow-md active:scale-95"
                  } relative ${item.glow}`}
                >
                  {!item.locked && (
                    <div className={`absolute inset-1 rounded-full border border-dashed border-zinc-800 opacity-60`} />
                  )}
                  <Icon className={`w-7 h-7 sm:w-9 sm:h-9 ${item.iconColor}`} />
                  {item.locked && (
                    <div className="absolute -bottom-1 -right-1 bg-zinc-900 border border-zinc-800 p-1 rounded-full">
                      <Lock className="w-3 h-3 text-zinc-500" />
                    </div>
                  )}
                </div>
                <span className={`font-serif text-[11px] sm:text-xs font-semibold mt-3 tracking-wide text-center transition-colors ${
                  item.locked
                    ? "text-zinc-600"
                    : "text-zinc-400 group-hover:text-zinc-100"
                }`}>
                  {item.label}
                </span>
              </div>
            );
          })}
        </div>
      </div>

      {/* 4. FLOATING MESSAGE BUTTON & CHAT OVERLAY */}
      <div className="fixed bottom-6 right-6 z-50">
        {/* Toggle Button */}
        {!chatOpen && (
          <button
            onClick={() => setChatOpen(true)}
            className="w-14 h-14 rounded-full flex items-center justify-center shadow-2xl transition-all duration-300 border relative bg-gradient-to-br from-emerald-600 to-teal-700 border-emerald-500/30 text-white hover:scale-105 hover:shadow-[0_0_20px_rgba(16,185,129,0.4)] active:scale-95 cursor-pointer"
          >
            <MessageSquare className="w-6 h-6" />

            {/* Red dot notification */}
            {(unreadChannels.world || unreadChannels.general || unreadChannels.sect) && (
              <span className="absolute top-0.5 right-0.5 flex h-3.5 w-3.5">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-3.5 w-3.5 bg-red-500"></span>
              </span>
            )}
          </button>
        )}

        {/* Chat Overlay Panel */}
        {chatOpen && (
          <div className="absolute bottom-0 right-0 w-[320px] sm:w-[420px] md:w-[480px] lg:w-[540px] max-w-[calc(100vw-2rem)] h-[420px] sm:h-[500px] md:h-[550px] lg:h-[600px] flex flex-col bg-zinc-950/95 border border-zinc-900 rounded-3xl p-4 sm:p-5 gap-3 sm:gap-4 shadow-2xl backdrop-blur-lg animate-in fade-in slide-in-from-bottom-5 duration-200">
            {/* Header with Integrated Close Button */}
            <div className="flex items-center justify-between border-b border-zinc-900/60 pb-2">
              <div className="flex items-center gap-2">
                <h3 className="font-serif font-bold text-sm sm:text-base md:text-lg text-zinc-200 tracking-wide">Truyền Âm Đại Điện</h3>
                <span className="text-[9px] sm:text-[10px] text-zinc-500 font-mono">Bảo Hoàng Thiên</span>
              </div>
              <button
                type="button"
                onClick={() => setChatOpen(false)}
                className="p-1 rounded-lg hover:bg-zinc-900 text-zinc-500 hover:text-zinc-200 transition-all cursor-pointer"
              >
                <X className="w-4 h-4 sm:w-5 sm:h-5" />
              </button>
            </div>

            {/* Channel Tabs */}
            <div className="flex gap-1.5 border-b border-zinc-900 pb-2.5">
              {[
                { id: "world", label: "Thế Giới" },
                { id: "general", label: "Bảo Hoàng" },
                { id: "sect", label: "Bang Hội" },
              ].map((ch) => {
                const isSectActive = ch.id === "sect";
                const isGeneralActive = ch.id === "general";
                const label = isSectActive
                  ? "Bang Hội"
                  : isGeneralActive
                  ? "Bảo Hoàng"
                  : "Thế Giới";

                return (
                  <button
                    key={ch.id}
                    onClick={() => {
                      if (ch.id === "sect" && (!mySectStatus || !mySectStatus.in_sect)) {
                        addToast("Đạo hữu chưa gia nhập tông môn nào, không thể xem tin nhắn kênh tông môn!", "error");
                        return;
                      }
                      if (ch.id === "general") {
                        const hasConnectingHeaven = player?.gu_inventory?.some(g => g.gu_id === "GU_CONNECTING_HEAVEN" && g.status !== "DEAD") ?? false;
                        const hasDivineSense = player?.gu_inventory?.some(g => g.gu_id === "GU_DIVINE_SENSE" && g.status !== "DEAD") ?? false;
                        if (!hasConnectingHeaven || !hasDivineSense) {
                          addToast("Đạo hữu cần sở hữu Thông Thiên Cổ và Thần Niệm Cổ để xem tin nhắn kênh Bảo Hoàng Thiên!", "error");
                          return;
                        }
                      }
                      setActiveChannel(ch.id as any);
                    }}
                    className={`px-2.5 py-1.5 rounded-xl font-serif text-[10px] sm:text-[11px] font-semibold transition-all border cursor-pointer flex items-center gap-1 ${
                      activeChannel === ch.id
                        ? "bg-zinc-900 text-zinc-100 border-zinc-800"
                        : "bg-transparent text-zinc-500 border-transparent hover:text-zinc-300"
                    }`}
                  >
                    <span>{label}</span>
                    {/* Channel specific red dot */}
                    {unreadChannels[ch.id as "world" | "general" | "sect"] && (
                      <span className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />
                    )}
                  </button>
                );
              })}
            </div>

            {/* Message View Area */}
            <div
              ref={chatContainerRef}
              className="bg-zinc-950/80 border border-zinc-900/60 rounded-2xl p-3 flex-grow overflow-y-auto flex flex-col gap-2.5 font-mono text-[11px] sm:text-xs md:text-sm"
            >
              {baoHoangMessages.length === 0 ? (
                <div className="text-zinc-600 text-center py-16">
                  Kênh chat trống. Hãy gửi tin nhắn đầu tiên!
                </div>
              ) : (
                baoHoangMessages.map((msg, index) => {
                  const isMe = msg.sender_username === player?.username;
                  const labelColor =
                    msg.channel === "world"
                      ? "text-red-400"
                      : msg.channel === "general"
                      ? "text-cyan-400"
                      : "text-emerald-400";
                  const chLabel =
                    msg.channel === "world"
                      ? "[Thế Giới]"
                      : msg.channel === "general"
                      ? "[Bảo Hoàng]"
                      : "[Bang Hội]";

                  return (
                    <div key={msg.id || index} className="flex items-start gap-1.5 leading-relaxed break-all">
                      <span className={`${labelColor} font-bold shrink-0`}>{chLabel}</span>
                      <span
                        onClick={() => !isMe && fetchPlayerProfile(msg.sender_username)}
                        className="text-zinc-400 font-bold shrink-0 cursor-pointer hover:text-zinc-100 hover:underline decoration-dotted transition-colors"
                        title={!isMe ? "Bấm để trinh thám cổ sư" : undefined}
                      >
                        {msg.sender_username}
                        <span className="text-zinc-650 text-[9px] font-normal ml-1 normal-case select-none">
                          ({formatRelativeTime(msg.created_at)})
                        </span>
                        :
                      </span>
                      <span className="text-zinc-200">{msg.content}</span>
                    </div>
                  );
                })
              )}
            </div>

            {/* Chat input form */}
            <form onSubmit={handleSendChat} className="flex gap-2">
              <input
                type="text"
                value={chatInput}
                onChange={(e) => setChatInput(e.target.value)}
                placeholder={
                  activeChannel === "world"
                    ? "Kênh Thế Giới (tốn 10 NT)..."
                    : activeChannel === "general"
                    ? "Bảo Hoàng Thiên..."
                    : "Nội bộ Bang Hội..."
                }
                maxLength={100}
                className="flex-grow bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-3 py-2 text-xs md:text-sm text-zinc-100 placeholder-zinc-600 focus:outline-none transition-all font-mono"
              />
              <button
                type="submit"
                disabled={actionLoading || !chatInput.trim()}
                className="bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 hover:border-zinc-700 disabled:bg-zinc-950 disabled:border-zinc-900 disabled:text-zinc-750 text-zinc-300 hover:text-white px-3.5 py-2 md:px-5 md:py-2.5 rounded-xl transition-all flex items-center justify-center shrink-0 cursor-pointer active:scale-95"
              >
                {actionLoading ? (
                  <Loader2 className="w-3.5 h-3.5 md:w-4 md:h-4 animate-spin text-zinc-500" />
                ) : (
                  <Send className="w-3.5 h-3.5 md:w-4 md:h-4" />
                )}
              </button>
            </form>
          </div>
        )}
      </div>
    </div>
  );
};
