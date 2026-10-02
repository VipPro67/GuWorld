"use client";

import { useState, useEffect, Fragment } from "react";
import { useGameStore, type PlayerPet, type ApertureVariantNPC } from "@/store/useGameStore";
import type { PartyFormationEntry } from "@/store/slices/partySlice";
import { Users, Crown, Swords } from "lucide-react";

function formationEntryLabel(
  e: PartyFormationEntry,
  party: { members: { id: string; username: string }[] },
  pets: PlayerPet[],
  npcs: ApertureVariantNPC[],
): string {
  if (e.kind === "player") {
    const m = party.members.find((x) => x.id === e.id);
    return `🧑 ${m?.username ?? "Cổ sư đã rời đội"}`;
  }
  if (e.kind === "pet") {
    const p = pets.find((x) => x.id === e.id);
    return p ? `🐾 ${p.count > 1 ? `Bầy ${p.nickname} (${p.count})` : p.nickname}` : "🐾 Thú (không còn)";
  }
  const n = npcs.find((x) => x.id === e.id);
  return n ? `👹 ${n.name} (R${n.rank})` : "👹 Dị nhân (không còn)";
}

// allySlots index 1..8 -> [x(row), y(col)]; col 1 = front (left), col 3 = back.
const POS_CELLS: [number, number][] = [[0, 0], [2, 1], [1, 1], [3, 1], [2, 2], [1, 2], [3, 2], [1, 3], [3, 3]];

function FormationGrid(props: {
  cells: (PartyFormationEntry | null)[];
  leaderName: string;
  party: { members: { id: string; username: string }[] };
  pets: PlayerPet[];
  npcs: ApertureVariantNPC[];
  selectedIdx: number | null;
  onCellClick?: (idx: number) => void;
  onClear?: (idx: number) => void;
  draggable?: boolean;
  onGridDrop?: (idx: number, data: { cell: number | null; entry: PartyFormationEntry | null }) => void;
}) {
  const { cells, leaderName, party, pets, npcs, selectedIdx, onCellClick, onClear, draggable, onGridDrop } = props;
  const [dragOver, setDragOver] = useState<number | null>(null);
  const entryAt = (x: number, y: number): { idx: number; e: PartyFormationEntry } | null => {
    for (let pos = 1; pos <= 8; pos++) {
      const [cx, cy] = POS_CELLS[pos];
      if (cx === x && cy === y) {
        const e = cells[pos - 1];
        return e ? { idx: pos - 1, e } : null;
      }
    }
    return null;
  };
  const posIdxAt = (x: number, y: number): number => {
    for (let pos = 1; pos <= 8; pos++) {
      if (POS_CELLS[pos][0] === x && POS_CELLS[pos][1] === y) return pos - 1;
    }
    return -1;
  };
  return (
    <div className="max-w-[380px] mx-auto w-full">
      <div className="grid grid-cols-[auto_repeat(3,1fr)] gap-1.5 items-stretch text-center">
        <div />
        {["Tiền", "Trung", "Hậu"].map((h) => (
          <div key={h} className="text-[10px] text-zinc-600 font-mono uppercase">{h}</div>
        ))}
        {[1, 2, 3].map((x) => (
          <Fragment key={`row${x}`}>
            <div className="text-[10px] text-zinc-600 font-mono self-center">H{x}</div>
            {[1, 2, 3].map((y) => {
              const isLeader = x === 2 && y === 3;
              const found = isLeader ? null : entryAt(x, y);
              const idx = isLeader ? -1 : found ? found.idx : posIdxAt(x, y);
              const selected = idx >= 0 && idx === selectedIdx;
              return (
                <div
                  key={`${x}-${y}`}
                  draggable={!!draggable && !!found && !!onGridDrop}
                  onDragStart={(ev) => {
                    if (!found) return;
                    ev.dataTransfer.setData("application/json", JSON.stringify({ cell: idx }));
                    ev.dataTransfer.effectAllowed = "move";
                  }}
                  onDragOver={(ev) => {
                    if (!draggable || !onGridDrop) return;
                    ev.preventDefault();
                    ev.dataTransfer.dropEffect = "move";
                    setDragOver(idx);
                  }}
                  onDragLeave={() => setDragOver((d) => (d === idx ? null : d))}
                  onDrop={(ev) => {
                    if (!onGridDrop) return;
                    ev.preventDefault();
                    setDragOver(null);
                    try {
                      const data = JSON.parse(ev.dataTransfer.getData("application/json"));
                      onGridDrop(idx, {
                        cell: typeof data.cell === "number" ? data.cell : null,
                        entry: data.entry && data.entry.id ? data.entry : null,
                      });
                    } catch { /* ignore bad drop */ }
                  }}
                  onClick={() => { if (!isLeader && onCellClick) onCellClick(idx); }}
                  className={`relative aspect-square rounded-lg border px-1 py-1 text-[11px] leading-tight flex flex-col items-center justify-center gap-0.5 transition-all ${
                    isLeader
                      ? "bg-emerald-950/40 border-emerald-600/50 text-emerald-300 font-bold"
                      : found
                        ? selected
                          ? "bg-amber-950/50 border-amber-500 text-amber-200 cursor-pointer"
                          : "bg-zinc-800/80 border-zinc-700 text-zinc-200 cursor-pointer hover:border-amber-700"
                        : selectedIdx !== null && onCellClick
                          ? "bg-zinc-900/60 border-dashed border-zinc-700 text-zinc-600 cursor-pointer hover:border-emerald-700"
                          : "bg-zinc-900/40 border-zinc-800/60 text-zinc-700"
                  } ${dragOver === idx ? "ring-2 ring-emerald-500 border-emerald-500" : ""}`}
                  title={isLeader ? "Trưởng nhóm (cố định)" : found ? formationEntryLabel(found.e, party, pets, npcs) : "Ô trống"}
                >
                  {isLeader ? (
                    <><span className="text-sm">👑</span><span className="max-w-full truncate">{leaderName}</span></>
                  ) : found ? (
                    <>
                      <span className="max-w-full truncate">{formationEntryLabel(found.e, party, pets, npcs)}</span>
                      {selected && onClear && (
                        <button type="button" onClick={(ev) => { ev.stopPropagation(); onClear(idx); }} className="text-[10px] px-1.5 rounded bg-rose-950/60 text-rose-300">✕ bỏ</button>
                      )}
                    </>
                  ) : (
                    <span className="text-zinc-700">·</span>
                  )}
                </div>
              );
            })}
          </Fragment>
        ))}
      </div>
      <p className="text-[10px] text-zinc-600 mt-1">Trái = tiền tuyến (hứng đòn trước), phải = hậu phương. 👑 cố định giữa-hậu.</p>
    </div>
  );
}

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
  const savePartyFormation = useGameStore((state) => state.savePartyFormation);
  const playerPets = useGameStore((state) => state.playerPets);
  const fetchPlayerPets = useGameStore((state) => state.fetchPlayerPets);
  const variantNPCs = useGameStore((state) => state.variantNPCs);
  const fetchNPCs = useGameStore((state) => state.fetchNPCs);

  const [inviteUsername, setInviteUsername] = useState("");
  const [inviteError, setInviteError] = useState<string | null>(null);
  const [partyLoading, setPartyLoading] = useState(false);
  const [editingFormation, setEditingFormation] = useState(false);
  // 8 cells indexed by grid pos-1 (POS_CELLS); null = empty.
  const [formationCells, setFormationCells] = useState<(PartyFormationEntry | null)[]>(Array(8).fill(null));
  const [selectedCell, setSelectedCell] = useState<number | null>(null);

  const expandFormation = (list: PartyFormationEntry[]): (PartyFormationEntry | null)[] => {    const cells: (PartyFormationEntry | null)[] = Array(8).fill(null);
    const rest: PartyFormationEntry[] = [];
    for (const e of list) {
      if (e.pos && e.pos >= 1 && e.pos <= 8 && !cells[e.pos - 1]) cells[e.pos - 1] = { kind: e.kind, id: e.id, pos: e.pos };
      else rest.push(e);
    }
    for (const e of rest) {
      const free = cells.findIndex((c) => !c);
      if (free < 0) break;
      cells[free] = { kind: e.kind, id: e.id, pos: free + 1 };
    }
    return cells;
  };

  const placeCandidate = (entry: PartyFormationEntry) => {
    setFormationCells((cells) => {
      const next = [...cells];
      const target = selectedCell !== null && !next[selectedCell] ? selectedCell : next.findIndex((c) => !c);
      if (target < 0) return cells;
      next[target] = { ...entry, pos: target + 1 };
      return next;
    });
    setSelectedCell(null);
  };

  const handleGridDrop = (idx: number, data: { cell: number | null; entry: PartyFormationEntry | null }) => {
    if (idx < 0) return;
    setFormationCells((cells) => {
      const next = [...cells];
      if (data.cell !== null) {
        if (data.cell === idx) return cells;
        const tmp = next[data.cell];
        next[data.cell] = next[idx];
        next[idx] = tmp;
        return next;
      }
      if (data.entry) {
        const old = next.findIndex((c) => c && c.kind === data.entry!.kind && c.id === data.entry!.id);
        if (old >= 0) next[old] = null;
        next[idx] = { ...data.entry, pos: idx + 1 };
        return next;
      }
      return cells;
    });
    setSelectedCell(null);
  };

  useEffect(() => {
    fetchPartyStatus();
    fetchPartyInvitations();
  }, [fetchPartyStatus, fetchPartyInvitations]);

  const isLeader = !!party && !!player && party.leader_id === player.id;
  useEffect(() => {
    if (isLeader) {
      void fetchPlayerPets();
      void fetchNPCs();
    }
  }, [isLeader, fetchPlayerPets, fetchNPCs]);

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

            {/* Battle formation (leader arranges, members read-only) */}
            <div className="pt-3 border-t border-zinc-800/80 space-y-2.5">
              <div className="flex items-center justify-between">
                <span className="text-xs text-zinc-500 uppercase tracking-wider font-mono block font-bold">
                  <Swords className="w-3.5 h-3.5 inline mr-1 text-amber-400" />
                  Đội hình xuất trận
                  {(party.formation?.length ?? 0) === 0 && (
                    <span className="normal-case font-sans font-normal"> (tự động: cả đội + thú/NPC theo slot)</span>
                  )}
                </span>
                {isLeader && (
                  <button
                    type="button"
                    onClick={() => {
                      if (editingFormation) {
                        setEditingFormation(false);
                        setSelectedCell(null);
                      } else {
                        setFormationCells(expandFormation(party.formation ?? []));
                        setSelectedCell(null);
                        setEditingFormation(true);
                      }
                    }}
                    className="text-xs text-amber-400 hover:text-amber-300"
                  >
                    {editingFormation ? "Hủy" : "Xếp đội hình"}
                  </button>
                )}
              </div>

              {!editingFormation ? (
                <div className="space-y-1.5">
                  <FormationGrid
                    cells={expandFormation(party.formation ?? [])}
                    leaderName={player.username}
                    party={party}
                    pets={playerPets}
                    npcs={variantNPCs}
                    selectedIdx={null}
                  />
                  {(party.formation?.length ?? 0) === 0 && (
                    <div className="text-[11px] text-zinc-600 px-1">
                      Chưa xếp tay — vào trận tự kéo cả đội cùng region + thú/NPC theo slot đội hình riêng.
                    </div>
                  )}
                </div>
              ) : (
                <div className="space-y-2.5">
                  <FormationGrid
                    cells={formationCells}
                    leaderName={player.username}
                    party={party}
                    pets={playerPets}
                    npcs={variantNPCs}
                    selectedIdx={selectedCell}
                    draggable
                    onGridDrop={handleGridDrop}
                    onCellClick={(idx) => {
                      if (selectedCell === null) {
                        if (formationCells[idx]) setSelectedCell(idx);
                      } else if (selectedCell === idx) {
                        setSelectedCell(null);
                      } else {
                        setFormationCells((cells) => {
                          const next = [...cells];
                          const tmp = next[selectedCell];
                          next[selectedCell] = next[idx];
                          next[idx] = tmp;
                          return next;
                        });
                        setSelectedCell(idx);
                      }
                    }}
                    onClear={(idx) => {
                      setFormationCells((cells) => cells.map((c, j) => (j === idx ? null : c)));
                      setSelectedCell(null);
                    }}
                  />
                  <div className="text-[11px] text-zinc-500">Kéo-thả để xếp (desktop), hoặc chạm ô để chọn rồi chạm ô khác để đổi chỗ, chạm ✕ để bỏ. Rồi chạm thú/dị nhân/member dưới để đặt vào ô đã chọn.</div>
                  {formationCells.some((c) => !c) && (
                    <div className="space-y-1.5">
                      <span className="text-[11px] text-zinc-500 uppercase tracking-wider font-mono">Thêm vào {selectedCell !== null ? `ô ${selectedCell + 1}` : "ô trống đầu tiên"}:</span>
                      <div className="flex flex-wrap gap-1.5 max-h-[160px] overflow-y-auto pr-1">
                        {party.members.filter((m) => m.id !== player.id && !formationCells.some((d) => d?.kind === "player" && d.id === m.id)).map((m) => (
                          <button key={m.id} type="button" draggable onDragStart={(ev) => { ev.dataTransfer.setData("application/json", JSON.stringify({ entry: { kind: "player", id: m.id } })); ev.dataTransfer.effectAllowed = "copy"; }} onClick={() => placeCandidate({ kind: "player", id: m.id })} className="text-[11px] px-2 py-1 rounded-lg bg-zinc-800 border border-zinc-700 text-zinc-300 hover:border-emerald-600 cursor-grab active:cursor-grabbing">
                            🧑 {m.username}
                          </button>
                        ))}
                        {playerPets.filter((p) => !formationCells.some((d) => d?.kind === "pet" && d.id === p.id)).map((p) => (
                          <button key={p.id} type="button" draggable onDragStart={(ev) => { ev.dataTransfer.setData("application/json", JSON.stringify({ entry: { kind: "pet", id: p.id } })); ev.dataTransfer.effectAllowed = "copy"; }} onClick={() => placeCandidate({ kind: "pet", id: p.id })} className="text-[11px] px-2 py-1 rounded-lg bg-zinc-800 border border-zinc-700 text-zinc-300 hover:border-emerald-600 cursor-grab active:cursor-grabbing">
                            🐾 {p.nickname} x{p.count}
                          </button>
                        ))}
                        {variantNPCs.filter((n) => n.role === "NONE" && !formationCells.some((d) => d?.kind === "npc" && d.id === n.id)).map((n) => (
                          <button key={n.id} type="button" draggable onDragStart={(ev) => { ev.dataTransfer.setData("application/json", JSON.stringify({ entry: { kind: "npc", id: n.id } })); ev.dataTransfer.effectAllowed = "copy"; }} onClick={() => placeCandidate({ kind: "npc", id: n.id })} className="text-[11px] px-2 py-1 rounded-lg bg-zinc-800 border border-zinc-700 text-zinc-300 hover:border-emerald-600 cursor-grab active:cursor-grabbing">
                            👹 {n.name} (R{n.rank})
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                  <button
                    type="button"
                    disabled={partyLoading}
                    onClick={async () => {
                      setPartyLoading(true);
                      try {
                        const order = formationCells.flatMap((c, i) => (c ? [{ kind: c.kind, id: c.id, pos: i + 1 }] : []));
                        if (await savePartyFormation(order)) {
                          setEditingFormation(false);
                          setSelectedCell(null);
                        }
                      } finally {
                        setPartyLoading(false);
                      }
                    }}
                    className="w-full py-2 bg-amber-950/40 border border-amber-600/40 text-amber-300 hover:bg-amber-900/40 font-bold rounded-xl text-xs transition-all disabled:opacity-50"
                  >
                    Lưu Đội Hình ({formationCells.filter(Boolean).length}/8 sau trưởng nhóm)
                  </button>
                  <p className="text-[11px] text-zinc-600 leading-relaxed">
                    Chỉ ai trong lưới mới xuất trận đúng ô đó (kẻ không đủ điều kiện: khác region/hết HP, thú đói/ngủ/thiếu soul, dị nhân đang làm việc — tự bỏ qua). Xóa hết + lưu để về chế độ tự động.
                  </p>
                </div>
              )}
            </div>

            {/* Invite Guidance for Leader */}
            {party.leader_id === player.id && (
              <div className="pt-3 border-t border-zinc-800/80">
                <p className="text-xs text-amber-400/90 font-serif bg-amber-950/20 border border-amber-800/30 rounded-xl p-3 flex items-center gap-2">
                  <span>💡</span>
                  <span>Nhấp chọn Cổ sư khác trực tiếp trên bản đồ Ngũ Vực để gửi lời mời vào tổ đội.</span>
                </p>
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
