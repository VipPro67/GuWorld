import React, { useEffect, useState, useRef } from "react";
import { useGameStore } from "@/store/useGameStore";
import {
  MessageSquare,
  Send,
  Loader2,
  X,
  ArrowLeftRight,
  ArrowUpDown,
  CornerUpLeft,
} from "lucide-react";

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
  if (diffHour < 24) return `${diffHour}h trước`;
  const diffDay = Math.floor(diffHour / 24);
  return `${diffDay} ngày trước`;
};

export const GlobalChatOverlay: React.FC = () => {
  const {
    player,
    baoHoangMessages,
    achievements,
    sendBaoHoangChat,
    actionLoading,
    fetchPlayerProfile,
    addToast,
    activeChannel,
    setActiveChannel,
    chatOpen,
    setChatOpen,
    unreadChannels,
    subscribeAllBaoHoangMessages,
    fetchBaoHoangMessages,
  } = useGameStore();

  const [chatInput, setChatInput] = useState("");
  const [showTimeIds, setShowTimeIds] = useState<Record<string, boolean>>({});

  const toggleTimeVisibility = (id: string) => {
    setShowTimeIds((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));
  };
  
  const [chatWidth, setChatWidth] = useState<"sm" | "md" | "lg">(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("guworld_chat_width");
      if (saved === "sm" || saved === "md" || saved === "lg") return saved;
    }
    return "md";
  });

  const [chatHeight, setChatHeight] = useState<"sm" | "md" | "lg">(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("guworld_chat_height");
      if (saved === "sm" || saved === "md" || saved === "lg") return saved;
    }
    return "md";
  });

  const chatContainerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const handleReply = (username: string) => {
    setChatInput((prev) => {
      const mention = `@${username} `;
      if (prev.includes(mention)) return prev;
      return mention + prev;
    });
    if (inputRef.current) {
      inputRef.current.focus();
    }
  };

  // Persist width and height adjustments locally
  useEffect(() => {
    if (typeof window !== "undefined") {
      localStorage.setItem("guworld_chat_width", chatWidth);
    }
  }, [chatWidth]);

  useEffect(() => {
    if (typeof window !== "undefined") {
      localStorage.setItem("guworld_chat_height", chatHeight);
    }
  }, [chatHeight]);

  // Auto scroll to bottom
  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [baoHoangMessages, chatOpen]);

  // Handle SSE subscriptions on mount
  useEffect(() => {
    subscribeAllBaoHoangMessages();
  }, [subscribeAllBaoHoangMessages]);

  const handleSendChat = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!chatInput.trim() || actionLoading) return;
    const success = await sendBaoHoangChat(chatInput.trim(), activeChannel);
    if (success) {
      setChatInput("");
    }
  };

  const toggleWidth = () => {
    setChatWidth((prev) => (prev === "sm" ? "md" : prev === "md" ? "lg" : "sm"));
  };

  const toggleHeight = () => {
    setChatHeight((prev) => (prev === "sm" ? "md" : prev === "md" ? "lg" : "sm"));
  };

  const widthClasses = {
    sm: "w-[320px] sm:w-[380px] md:w-[440px]",
    md: "w-[380px] sm:w-[500px] md:w-[560px]",
    lg: "w-[440px] sm:w-[620px] md:w-[720px] lg:w-[820px]",
  }[chatWidth];

  const heightClasses = {
    sm: "h-[360px] sm:h-[440px] md:h-[520px]",
    md: "h-[460px] sm:h-[560px] md:h-[640px]",
    lg: "h-[560px] sm:h-[680px] md:h-[760px] lg:h-[860px]",
  }[chatHeight];

  return (
    <div className="fixed bottom-6 right-6 z-50">
      {/* Toggle Button */}
      {!chatOpen && (
        <button
          onClick={() => {
            setChatOpen(true);
            subscribeAllBaoHoangMessages();
            fetchBaoHoangMessages(activeChannel);
          }}
          className="w-12 h-12 rounded-full flex items-center justify-center shadow-2xl transition-all duration-300 border relative bg-gradient-to-br from-emerald-600 to-teal-700 border-emerald-500/30 text-white hover:scale-105 hover:shadow-[0_0_20px_rgba(16,185,129,0.4)] active:scale-95 cursor-pointer"
        >
          <MessageSquare className="w-5 h-5" />

          {/* Red dot notification */}
          {(unreadChannels.world || unreadChannels.general || unreadChannels.sect) && (
            <span className="absolute top-0.5 right-0.5 flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-red-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-red-500"></span>
            </span>
          )}
        </button>
      )}

      {/* Chat Overlay Panel */}
      {chatOpen && (
        <div className={`absolute bottom-0 right-0 ${widthClasses} ${heightClasses} max-w-[calc(100vw-2rem)] max-h-[calc(100vh-2rem)] flex flex-col bg-white/95 dark:bg-zinc-950/95 border border-zinc-200 dark:border-zinc-900 rounded-3xl p-2 sm:p-4 gap-3 sm:gap-4 shadow-2xl backdrop-blur-lg animate-in fade-in slide-in-from-bottom-5 duration-200`}>
          {/* Channel Tabs */}
          <div className="flex items-center justify-between">
            <div className="flex gap-1.5 border-b border-zinc-150 dark:border-zinc-900 pb-2">
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
                      if (ch.id === "general") {
                        const hasConnectingHeaven = player?.gu_inventory?.some(g => g.gu_id === "GU_CONNECTING_HEAVEN" && g.status !== "DEAD") ?? false;
                        const hasDivineSense = player?.gu_inventory?.some(g => g.gu_id === "GU_DIVINE_SENSE" && g.status !== "DEAD") ?? false;
                        if (!hasConnectingHeaven || !hasDivineSense) {
                          addToast("Đạo hữu cần sở hữu Thông Thiên Cổ và Thần Niệm Cổ để xem tin nhắn kênh Bảo Hoàng Thiên!", "error");
                          return;
                        }
                      }
                      setActiveChannel(ch.id as any);
                      subscribeAllBaoHoangMessages();
                      fetchBaoHoangMessages(ch.id);
                    }}
                    className={`px-2.5 py-1.5 rounded-xl font-serif text-[10px] sm:text-[11px] font-semibold transition-all border cursor-pointer flex items-center gap-1 ${
                      activeChannel === ch.id
                        ? "bg-zinc-150 dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 border-zinc-300/60 dark:border-zinc-800"
                        : "bg-transparent text-zinc-400 dark:text-zinc-500 border-transparent hover:text-zinc-700 dark:hover:text-zinc-300"
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
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={toggleWidth}
                title="Thay đổi chiều rộng"
                className="p-1 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-900 text-zinc-400 dark:text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-200 transition-all cursor-pointer flex items-center justify-center"
              >
                <ArrowLeftRight className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={toggleHeight}
                title="Thay đổi chiều cao"
                className="p-1 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-900 text-zinc-400 dark:text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-200 transition-all cursor-pointer flex items-center justify-center"
              >
                <ArrowUpDown className="w-3.5 h-3.5" />
              </button>
              <button
                type="button"
                onClick={() => setChatOpen(false)}
                className="p-1 rounded-lg hover:bg-zinc-100 dark:hover:bg-zinc-900 text-zinc-400 dark:text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-200 transition-all cursor-pointer"
              >
                <X className="w-4 h-4 sm:w-5 sm:h-5" />
              </button>
            </div>
          </div>

          {/* Message View Area */}
          <div
            ref={chatContainerRef}
            className="bg-zinc-50/50 dark:bg-zinc-950/80 border border-zinc-150 dark:border-zinc-900/60 rounded-2xl p-2 flex-grow overflow-y-auto flex flex-col gap-1 font-mono text-[10px] sm:text-[11px] md:text-xs"
          >
            {baoHoangMessages.length === 0 ? (
              <div className="text-zinc-400 dark:text-zinc-650 text-center py-16">
                Kênh chat trống. Hãy gửi tin nhắn đầu tiên!
              </div>
            ) : (
              baoHoangMessages.map((msg, index) => {
                const isMe = msg.sender_username === player?.username;
                const isSystem = msg.sender_id === "11111111-1111-1111-1111-111111111111";
                const labelColor =
                  msg.channel === "world"
                    ? "text-red-500 dark:text-red-400"
                    : msg.channel === "general"
                    ? "text-cyan-600 dark:text-cyan-400"
                    : "text-emerald-600 dark:text-emerald-400";
                // Card container styles
                const cardClass = isSystem
                  ? "w-full bg-amber-500/10 dark:bg-amber-950/15 border border-amber-300/30 dark:border-amber-500/10 rounded-lg p-1.5 sm:p-2"
                  : "w-full bg-zinc-100/70 dark:bg-zinc-900/30 border border-zinc-200/50 dark:border-zinc-800/40 rounded-lg p-1.5 sm:p-2 group";

                const nameColorClass = isSystem
                  ? "text-amber-700 dark:text-amber-400 font-bold"
                  : isMe
                  ? "text-sky-600 dark:text-sky-400 font-bold"
                  : "text-zinc-500 dark:text-zinc-400 font-bold cursor-pointer hover:text-zinc-800 dark:hover:text-zinc-100 hover:underline decoration-dotted transition-colors";

                const contentColorClass = isSystem ? "text-amber-900 dark:text-amber-100/90" : "text-zinc-800 dark:text-zinc-200";

                return (
                  <div key={msg.id || index} className={`${cardClass} leading-relaxed break-all flex items-start gap-1.5`}>
                    <div className="w-5 h-5 rounded-full shrink-0 overflow-hidden bg-zinc-300 dark:bg-zinc-800 flex items-center justify-center text-[7px] font-bold text-zinc-500 mt-0.5">
                      {msg.sender_avatar ? (
                        <img src={msg.sender_avatar} alt="" className="w-full h-full object-cover" />
                      ) : (
                        msg.sender_username.slice(0, 2).toUpperCase()
                      )}
                    </div>
                    <div 
                      className="min-w-0 flex-1 leading-relaxed break-all cursor-pointer"
                      onClick={() => toggleTimeVisibility(msg.id || index.toString())}
                    >
                      <span
                        onClick={(e) => {
                          e.stopPropagation();
                          if (!isSystem && !isMe) fetchPlayerProfile(msg.sender_username);
                        }}
                        className={`${nameColorClass} mr-1.5`}
                        title={!isSystem && !isMe ? "Bấm để trinh thám cổ sư" : undefined}
                      >
                        {msg.sender_title && (
                          <span className="mr-1" title={(achievements.find((a: any) => a.id === msg.sender_title) as any)?.name || msg.sender_title}>
                            {(achievements.find((a: any) => a.id === msg.sender_title) as any)?.icon || "🏆"}
                          </span>
                        )}
                        {msg.sender_username}
                        {showTimeIds[msg.id || index.toString()] && (
                          <span className="text-zinc-400 dark:text-zinc-650 text-[8.5px] font-normal ml-1 normal-case select-none">
                            ({formatRelativeTime(msg.created_at)})
                          </span>
                        )}
                        :
                      </span>
                      <span className={`${contentColorClass} ml-0.5`}>{msg.content}</span>
                    </div>
                    {!isSystem && (
                      <button
                        type="button"
                        onClick={() => handleReply(msg.sender_username)}
                        className="opacity-0 group-hover:opacity-100 sm:opacity-50 hover:sm:opacity-100 p-1 rounded hover:bg-zinc-200/50 dark:hover:bg-zinc-800/50 text-zinc-400 dark:text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300 transition-all cursor-pointer self-start shrink-0 ml-1"
                        title="Phản hồi (Nhắc tên)"
                      >
                        <CornerUpLeft className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                );
              })
            )}
          </div>

          {/* Chat input form */}
          <form onSubmit={handleSendChat} className="flex gap-2">
            <input
              ref={inputRef}
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
              maxLength={500}
              className="flex-grow bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-900 focus:border-zinc-300 dark:focus:border-zinc-800 rounded-xl px-3 py-2 text-xs md:text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 dark:placeholder-zinc-600 focus:outline-none transition-all font-mono"
            />
            <button
              type="submit"
              disabled={actionLoading || !chatInput.trim()}
              className="bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 hover:bg-zinc-200 dark:hover:bg-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 disabled:bg-zinc-50 dark:disabled:bg-zinc-950 disabled:border-zinc-100 dark:disabled:border-zinc-900 disabled:text-zinc-400 dark:disabled:text-zinc-750 text-zinc-700 dark:text-zinc-300 hover:text-zinc-900 dark:hover:text-white px-3.5 py-2 md:px-5 md:py-2.5 rounded-xl transition-all flex items-center justify-center shrink-0 cursor-pointer active:scale-95"
            >
              {actionLoading ? (
                <Loader2 className="w-3.5 h-3.5 md:w-4 md:h-4 animate-spin text-zinc-400 dark:text-zinc-500" />
              ) : (
                <Send className="w-3.5 h-3.5 md:w-4 md:h-4" />
              )}
            </button>
          </form>
        </div>
      )}
    </div>
  );
};
