"use client";

import React, { useEffect, useState } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Mail, Send, RefreshCw, Lock, AlertCircle, MessageSquare } from "lucide-react";

export const ChatConsole: React.FC = () => {
  const {
    player,
    inbox,
    baoHoangMessages,
    sendPrivateMail,
    fetchInbox,
    sendBaoHoangChat,
    fetchBaoHoangMessages,
    actionLoading,
  } = useGameStore();

  const [activeSubTab, setActiveSubTab] = useState<"mail" | "baohoang">("mail");
  const [receiver, setReceiver] = useState("");
  const [mailContent, setMailContent] = useState("");
  const [chatContent, setChatContent] = useState("");
  const [loading, setLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const handleRefreshInbox = async () => {
    setLoading(true);
    try {
      await fetchInbox();
    } finally {
      setLoading(false);
    }
  };

  const handleRefreshBaoHoang = async () => {
    if (!player || player.rank < 6) return;
    setLoading(true);
    try {
      await fetchBaoHoangMessages();
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (activeSubTab === "mail") {
      handleRefreshInbox();
    } else {
      handleRefreshBaoHoang();
    }
  }, [activeSubTab]);

  const handleSendMail = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!receiver.trim() || !mailContent.trim() || actionLoading) return;
    setErrorMsg(null);

    if (player && player.primeval_stones < 1) {
      setErrorMsg("Không đủ Nguyên Thạch! Cần ít nhất 1 Nguyên Thạch để gửi Cổ trùng truyền tin.");
      return;
    }

    const ok = await sendPrivateMail(receiver.trim(), mailContent.trim());
    if (ok) {
      setMailContent("");
      handleRefreshInbox();
    } else {
      setErrorMsg("Gửi thư thất bại. Hãy chắc chắn đạo hiệu người nhận tồn tại.");
    }
  };

  const handleSendBaoHoang = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!chatContent.trim() || actionLoading) return;
    setErrorMsg(null);

    const ok = await sendBaoHoangChat(chatContent.trim());
    if (ok) {
      setChatContent("");
    } else {
      setErrorMsg("Không thể gửi tin nhắn thần niệm.");
    }
  };

  if (!player) return null;

  return (
    <div className="glass-panel-gold rounded-3xl p-6 w-full max-w-4xl mx-auto space-y-6 text-left relative overflow-hidden group">
      {/* Decorative background grid pattern */}
      <div className="absolute inset-0 bg-[linear-gradient(to_right,rgba(139,92,246,0.01)_1px,transparent_1px),linear-gradient(to_bottom,rgba(139,92,246,0.01)_1px,transparent_1px)] bg-[size:16px_28px] pointer-events-none" />

      {/* Header with Sub-tabs */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center border-b border-zinc-800 pb-4 gap-4 relative z-10">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-violet-950/30 border border-violet-500/20 flex items-center justify-center">
            <MessageSquare className="w-5 h-5 text-violet-400 animate-pulse" />
          </div>
          <div>
            <h3 className="text-lg font-bold text-zinc-100 font-serif">Kênh Truyền Tin & Thần Niệm</h3>
            <p className="text-xs text-zinc-500">Mạng lưới truyền âm Nam Cương và kết nối thần niệm Hoàng Bảo Thiên.</p>
          </div>
        </div>

        {/* Sub-tab selection */}
        <div className="flex bg-zinc-950/80 p-1 border border-zinc-900 rounded-2xl shrink-0">
          <button
            onClick={() => setActiveSubTab("mail")}
            className={`px-4 py-1.5 rounded-xl font-serif text-xs font-bold transition-all cursor-pointer ${
              activeSubTab === "mail" ? "bg-zinc-900 text-amber-400 font-bold" : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            📬 Thư Truyền Tin
          </button>
          <button
            onClick={() => setActiveSubTab("baohoang")}
            className={`px-4 py-1.5 rounded-xl font-serif text-xs font-bold transition-all cursor-pointer ${
              activeSubTab === "baohoang" ? "bg-zinc-900 text-violet-400 font-bold" : "text-zinc-500 hover:text-zinc-300"
            }`}
          >
            ✨ Hoàng Bảo Thiên
          </button>
        </div>
      </div>

      {activeSubTab === "mail" ? (
        // Private Messenger Gu Section
        <div className="grid grid-cols-1 md:grid-cols-12 gap-6 relative z-10 items-start">
          {/* Send Form */}
          <form onSubmit={handleSendMail} className="md:col-span-5 bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 space-y-4">
            <h4 className="font-serif font-bold text-sm text-zinc-200 flex items-center gap-1.5 border-b border-zinc-900 pb-2">
              📜 Thư Tín Cổ Trùng
            </h4>
            
            <div className="text-[10px] text-zinc-500 leading-normal font-mono">
              * Dùng Cổ trùng đưa thư để gửi lời nhắn trực tiếp tới Không Khiếu của cổ sư khác.
              <br />
              <span className="text-amber-500 font-bold">Lưu ý: Tiêu hao 1 Nguyên Thạch làm thức ăn cho Cổ trùng bay đi.</span>
            </div>

            <div className="space-y-3.5">
              <div className="space-y-1">
                <label className="text-[10px] text-zinc-500 uppercase tracking-wider font-mono">Người nhận (Đạo hiệu)</label>
                <input
                  type="text"
                  placeholder="Đạo hiệu cổ sư..."
                  value={receiver}
                  onChange={(e) => {
                    setReceiver(e.target.value);
                    setErrorMsg(null);
                  }}
                  required
                  className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-3 py-2 text-xs text-zinc-150 focus:outline-none transition-colors"
                />
              </div>

              <div className="space-y-1">
                <label className="text-[10px] text-zinc-500 uppercase tracking-wider font-mono">Nội dung thư tín</label>
                <textarea
                  placeholder="Viết thư gửi đi..."
                  value={mailContent}
                  onChange={(e) => {
                    setMailContent(e.target.value);
                    setErrorMsg(null);
                  }}
                  required
                  rows={4}
                  className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-3 py-2 text-xs text-zinc-150 focus:outline-none transition-colors resize-none"
                />
              </div>

              {errorMsg && (
                <div className="text-[10px] font-mono text-rose-400 bg-rose-950/10 border border-rose-950/20 rounded-lg p-2 flex items-start gap-1.5">
                  <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                  <span>{errorMsg}</span>
                </div>
              )}

              <button
                type="submit"
                disabled={actionLoading || !receiver.trim() || !mailContent.trim()}
                className="w-full py-2 bg-amber-500 hover:bg-amber-600 text-zinc-950 font-bold rounded-xl text-xs flex items-center justify-center gap-1.5 transition-all active:scale-[0.98] cursor-pointer font-serif shadow-md shadow-amber-500/5"
              >
                <Send className="w-3.5 h-3.5" />
                Thả Cổ Đưa Thư (1 Đá)
              </button>
            </div>
          </form>

          {/* Inbox List */}
          <div className="md:col-span-7 space-y-4 h-full flex flex-col">
            <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
              <h4 className="font-serif font-bold text-sm text-zinc-200 flex items-center gap-1.5">
                📬 Hòm Thư Đến (Inbox)
              </h4>
              <button
                type="button"
                onClick={handleRefreshInbox}
                disabled={loading}
                className="text-[10px] text-zinc-500 hover:text-zinc-300 cursor-pointer flex items-center gap-1 font-mono bg-transparent border-0"
              >
                <RefreshCw className={`w-3 h-3 ${loading ? "animate-spin" : ""}`} /> Tải lại hòm thư
              </button>
            </div>

            <div className="space-y-2.5 max-h-[360px] overflow-y-auto pr-1 flex-grow">
              {inbox.length === 0 ? (
                <div className="text-center py-16 text-zinc-600 italic text-xs">
                  Không có thư tín mới từ Cổ trùng.
                </div>
              ) : (
                inbox.map((msg) => (
                  <div key={msg.id} className="bg-zinc-950/50 border border-zinc-900 rounded-2xl p-4 space-y-2">
                    <div className="flex justify-between items-center text-[10px] font-mono">
                      <span className="text-emerald-400 font-bold flex items-center gap-1">
                        👤 Từ: {msg.sender_username}
                      </span>
                      <span className="text-zinc-600">
                        {new Date(msg.created_at).toLocaleTimeString()} {new Date(msg.created_at).toLocaleDateString()}
                      </span>
                    </div>
                    <p className="text-xs text-zinc-300 bg-zinc-950/20 border border-zinc-900 p-2.5 rounded-xl whitespace-pre-line leading-relaxed font-mono">
                      {msg.content}
                    </p>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      ) : (
        // Global Bao Hoang Chat Section
        <div className="space-y-4 relative z-10">
          {player.rank < 6 ? (
            // Locked Section
            <div className="border border-zinc-900 bg-zinc-950/55 rounded-3xl p-8 text-center max-w-xl mx-auto space-y-4 py-16 shadow-lg">
              <div className="w-12 h-12 rounded-full bg-rose-955/40 border border-rose-900/30 flex items-center justify-center mx-auto text-rose-500">
                <Lock className="w-6 h-6 animate-pulse" />
              </div>
              <div className="space-y-1">
                <h4 className="font-serif font-bold text-sm text-zinc-300">Hoàng Bảo Thiên - Phong Ấn Thần Niệm</h4>
                <p className="text-xs text-zinc-500 leading-relaxed">
                  Đạo hữu chưa thăng cấp Cổ Tiên (Rank 6 trở lên). Hoàng Bảo Thiên là bảo thiên bảo khố huyền bí, 
                  nơi thần niệm của các Cổ Tiên kết nối giao dịch tài nguyên và truyền tin toàn thế giới. Hãy nỗ lực thăng tiên!
                </p>
              </div>
            </div>
          ) : (
            // Active Global Chat
            <div className="grid grid-cols-1 md:grid-cols-12 gap-6 items-start">
              {/* Message Input Form */}
              <form onSubmit={handleSendBaoHoang} className="md:col-span-4 bg-zinc-950/40 border border-zinc-900 rounded-3xl p-5 space-y-4">
                <h4 className="font-serif font-bold text-sm text-violet-400 flex items-center gap-1.5 border-b border-zinc-900 pb-2">
                  ✨ Truyền Phát Thần Niệm
                </h4>
                
                <div className="text-[10px] text-zinc-500 leading-normal font-mono">
                  * Ký gửi thần niệm của Tiên nhân trực tiếp vào Hoàng Bảo Thiên. Mọi Cổ Tiên trên thế giới đều có thể cảm ứng được.
                  <br />
                  <span className="text-violet-400 font-bold">Chi phí: Miễn phí (Tiêu hao thần niệm vô hình).</span>
                </div>

                <div className="space-y-3.5">
                  <div className="space-y-1">
                    <label className="text-[10px] text-zinc-500 uppercase tracking-wider font-mono">Nội dung truyền đạt</label>
                    <textarea
                      placeholder="Gửi thần niệm của bạn..."
                      value={chatContent}
                      onChange={(e) => {
                        setChatContent(e.target.value);
                        setErrorMsg(null);
                      }}
                      required
                      rows={4}
                      className="w-full bg-zinc-950 border border-zinc-900 focus:border-zinc-800 rounded-xl px-3 py-2 text-xs text-zinc-150 focus:outline-none transition-colors resize-none"
                    />
                  </div>

                  {errorMsg && (
                    <div className="text-[10px] font-mono text-rose-455 bg-rose-955/10 border border-rose-955/20 rounded-lg p-2 flex items-start gap-1.5">
                      <AlertCircle className="w-3.5 h-3.5 shrink-0 mt-0.5" />
                      <span>{errorMsg}</span>
                    </div>
                  )}

                  <button
                    type="submit"
                    disabled={actionLoading || !chatContent.trim()}
                    className="w-full py-2 bg-gradient-to-r from-violet-500 to-indigo-600 hover:from-violet-600 hover:to-indigo-750 text-zinc-100 font-bold rounded-xl text-xs flex items-center justify-center gap-1.5 transition-all active:scale-[0.98] cursor-pointer font-serif shadow-md shadow-violet-500/5 text-shadow-sm"
                  >
                    <Send className="w-3.5 h-3.5 text-violet-200" />
                    Đăng Tải Thần Niệm
                  </button>
                </div>
              </form>

              {/* Messages viewport */}
              <div className="md:col-span-8 space-y-4">
                <div className="flex items-center justify-between border-b border-zinc-900 pb-2">
                  <h4 className="font-serif font-bold text-sm text-violet-400 flex items-center gap-1.5">
                    ✨ Thần Niệm Hoàng Bảo Thiên
                  </h4>
                  <button
                    type="button"
                    onClick={handleRefreshBaoHoang}
                    disabled={loading}
                    className="text-[10px] text-zinc-500 hover:text-zinc-300 cursor-pointer flex items-center gap-1 font-mono bg-transparent border-0"
                  >
                    <RefreshCw className={`w-3 h-3 ${loading ? "animate-spin text-violet-400" : ""}`} /> Tải lại thần niệm
                  </button>
                </div>

                <div className="space-y-2.5 max-h-[360px] overflow-y-auto pr-1">
                  {baoHoangMessages.length === 0 ? (
                    <div className="text-center py-16 text-zinc-650 italic text-xs">
                      Không có tin nhắn thần niệm nào trong tầm cảm ứng.
                    </div>
                  ) : (
                    baoHoangMessages.map((msg) => (
                      <div key={msg.id} className="bg-zinc-950/60 border border-violet-955/10 rounded-2xl p-4 space-y-2 shadow-inner">
                        <div className="flex justify-between items-center text-[10px] font-mono border-b border-zinc-900 pb-1.5">
                          <span className="text-violet-400 font-bold flex items-center gap-1">
                            🔮 Tiên nhân: {msg.sender_username}
                          </span>
                          <span className="text-zinc-600">
                            {new Date(msg.created_at).toLocaleTimeString()} {new Date(msg.created_at).toLocaleDateString()}
                          </span>
                        </div>
                        <p className="text-xs text-zinc-300 bg-zinc-950/10 p-1 rounded leading-relaxed font-mono">
                          {msg.content}
                        </p>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};
