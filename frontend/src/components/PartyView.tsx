"use client";

import { useState, useEffect } from "react";
import { useGameStore } from "@/store/useGameStore";
import { Users, Crown } from "lucide-react";

export function PartyView() {
  const player = useGameStore((state) => state.player);
  const party = useGameStore((state) => state.party);
  const partyInvitations = useGameStore((state) => state.partyInvitations);
  const createParty = useGameStore((state) => state.createParty);
  const fetchPartyStatus = useGameStore((state) => state.fetchPartyStatus);
  const fetchPartyInvitations = useGameStore((state) => state.fetchPartyInvitations);
  const invitePlayer = useGameStore((state) => state.invitePlayer);
  const respondPartyInvitation = useGameStore((state) => state.respondPartyInvitation);
  const leaveParty = useGameStore((state) => state.leaveParty);
  const fetchPlayerProfile = useGameStore((state) => state.fetchPlayerProfile);

  const [inviteUsername, setInviteUsername] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [partyLoading, setPartyLoading] = useState(false);

  useEffect(() => {
    fetchPartyStatus();
    fetchPartyInvitations();
  }, [fetchPartyStatus, fetchPartyInvitations]);

  if (!player) return null;

  const handleCreateParty = async () => {
    setPartyLoading(true);
    try {
      await createParty();
    } finally {
      setPartyLoading(false);
    }
  };

  const handleInvitePlayer = async () => {
    if (!inviteUsername.trim()) return;
    setPartyLoading(true);
    setInviteError(null);
    try {
      const ok = await invitePlayer(inviteUsername.trim());
      if (ok) {
        setInviteUsername("");
      } else {
        setInviteError("Mời thất bại. Hãy đảm bảo đạo hiệu đúng và cự ly / tổ đội chưa có.");
      }
    } finally {
      setPartyLoading(false);
    }
  };

  const handleLeaveParty = async () => {
    setPartyLoading(true);
    try {
      await leaveParty();
    } finally {
      setPartyLoading(false);
    }
  };

  return (
    <div className="w-full max-w-2xl mx-auto space-y-4 p-4 font-serif">
      <div className="bg-zinc-950/80 border border-zinc-900 rounded-3xl p-6 space-y-6 backdrop-blur-md shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
          <h3 className="font-serif font-bold text-base text-amber-400 flex items-center gap-2">
            <Users className="w-5 h-5" /> TỔ ĐỘI CHIẾN HỮU
          </h3>
          {party && (
            <span className="text-xs font-mono text-zinc-400 bg-zinc-900 border border-zinc-800 px-2.5 py-1 rounded-full uppercase tracking-widest">
              {party.members.length}/4 Cổ sư
            </span>
          )}
        </div>

        {!party ? (
          <div className="text-center py-6 space-y-4">
            <p className="text-xs text-zinc-400 leading-relaxed max-w-md mx-auto">
              Đạo hữu chưa gia nhập tổ đội. Lập tổ đội để đi chung bản đồ và cùng chia sẻ Nguyên Thạch & Vật phẩm rơi ra khi chiến thắng.
            </p>

            <button
              type="button"
              onClick={handleCreateParty}
              disabled={partyLoading}
              className="w-full sm:w-auto px-8 py-2.5 bg-emerald-950/40 border border-emerald-500/40 text-emerald-400 hover:bg-emerald-900/50 font-bold rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer shadow-lg disabled:opacity-50"
            >
              {partyLoading ? "Đang lập tổ..." : "Lập Tổ Đội Mới"}
            </button>

            {partyInvitations && partyInvitations.length > 0 && (
              <div className="pt-6 border-t border-zinc-900/80 text-left space-y-3">
                <span className="text-xs text-zinc-500 uppercase tracking-wider font-mono block font-bold">
                  Lời mời vào tổ đội ({partyInvitations.length})
                </span>
                <div className="space-y-2 max-h-[200px] overflow-y-auto pr-1">
                  {partyInvitations.map((inv) => (
                    <div
                      key={inv.id}
                      className="flex items-center justify-between bg-zinc-900/70 border border-zinc-800/80 rounded-xl p-3 text-xs"
                    >
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-zinc-800 text-zinc-300 flex items-center justify-center font-bold text-xs overflow-hidden shrink-0 border border-zinc-700">
                          {inv.inviter_avatar ? (
                            <img src={inv.inviter_avatar} alt="" className="w-full h-full object-cover" />
                          ) : (
                            inv.inviter_name.slice(0, 2).toUpperCase()
                          )}
                        </div>
                        <span className="font-semibold text-zinc-200">
                          {inv.inviter_name}
                        </span>
                      </div>
                      <div className="flex gap-2 shrink-0">
                        <button
                          type="button"
                          onClick={() => respondPartyInvitation(inv.id, "accept")}
                          disabled={partyLoading}
                          className="px-3 py-1.5 bg-emerald-950/40 border border-emerald-500/40 text-emerald-400 font-bold rounded-lg text-xs hover:bg-emerald-900/50 transition-all cursor-pointer disabled:opacity-50"
                        >
                          Đồng ý
                        </button>
                        <button
                          type="button"
                          onClick={() => respondPartyInvitation(inv.id, "reject")}
                          disabled={partyLoading}
                          className="px-3 py-1.5 bg-rose-950/20 border border-rose-900/40 text-rose-400 font-bold rounded-lg text-xs hover:bg-rose-950/40 transition-all cursor-pointer disabled:opacity-50"
                        >
                          Từ chối
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="space-y-5">
            {/* Member list */}
            <div className="space-y-2.5">
              <span className="text-xs text-zinc-500 uppercase tracking-wider font-mono block font-bold">
                Thành viên tổ đội ({party.members.length}/4)
              </span>
              {party.members.map((member) => {
                const isLeader = member.id === party.leader_id;
                const isMe = member.id === player.id;
                return (
                  <div
                    key={member.id}
                    className="flex items-center justify-between bg-zinc-900/70 border border-zinc-800/80 rounded-xl p-3 text-xs"
                  >
                    <div
                      onClick={() => !isMe && fetchPlayerProfile(member.username)}
                      className={`flex items-center gap-3 ${!isMe ? "cursor-pointer hover:text-white" : ""}`}
                      title={!isMe ? "Bấm để trinh thám cổ sư" : undefined}
                    >
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold text-xs overflow-hidden ${
                        isMe ? "bg-emerald-500 text-zinc-950" : "bg-zinc-800 text-zinc-300 border border-zinc-700"
                      }`}>
                        {member.avatar_url ? (
                          <img src={member.avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          member.username.slice(0, 2).toUpperCase()
                        )}
                      </div>
                      <span className={`font-semibold text-sm ${isMe ? "text-emerald-400" : "text-zinc-200 hover:underline decoration-dotted"}`}>
                        {member.username} {isMe && "(Bạn)"}
                      </span>
                    </div>
                    {isLeader && (
                      <span className="text-[10px] text-amber-400 bg-amber-950/40 border border-amber-800/40 px-2 py-1 rounded-lg font-mono uppercase tracking-wider flex items-center gap-1.5">
                        <Crown className="w-3 h-3 text-amber-400" /> Trưởng nhóm
                      </span>
                    )}
                  </div>
                );
              })}
            </div>

            {/* Invite Form for Leader */}
            {party.leader_id === player.id && (
              <div className="pt-4 border-t border-zinc-800/80 space-y-2.5">
                <span className="text-xs text-zinc-500 uppercase tracking-wider font-mono block font-bold">
                  Chiêu mộ thành viên
                </span>
                <div className="flex gap-2">
                  <input
                    type="text"
                    placeholder="Đạo hiệu đối phương..."
                    value={inviteUsername}
                    onChange={(e) => {
                      setInviteUsername(e.target.value);
                      setInviteError(null);
                    }}
                    className="bg-zinc-950 border border-zinc-800 rounded-xl px-3.5 py-2 text-xs text-zinc-100 placeholder-zinc-600 flex-grow focus:outline-none focus:border-amber-500/50 transition-colors"
                  />
                  <button
                    type="button"
                    onClick={handleInvitePlayer}
                    disabled={partyLoading || !inviteUsername.trim()}
                    className="bg-zinc-900 border border-zinc-800 hover:bg-zinc-800 text-amber-400 font-bold px-4 py-2 rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer font-mono shrink-0 disabled:opacity-50"
                  >
                    Mời
                  </button>
                </div>
                {inviteError && (
                  <p className="text-xs text-rose-400 font-mono">{inviteError}</p>
                )}
              </div>
            )}

            {/* Leave Button */}
            <div className="pt-2 border-t border-zinc-800/80">
              <button
                type="button"
                onClick={handleLeaveParty}
                disabled={partyLoading}
                className="w-full py-2.5 bg-rose-950/20 border border-rose-900/40 text-rose-400 hover:bg-rose-950/40 font-bold rounded-xl text-xs transition-all active:scale-[0.98] cursor-pointer disabled:opacity-50"
              >
                Rời Khỏi Tổ Đội
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
