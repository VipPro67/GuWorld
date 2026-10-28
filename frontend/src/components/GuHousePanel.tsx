"use client";

import { useEffect, useState, useMemo } from "react";
import { Castle, Shield, Wrench, Sparkles, Plus, Check } from "lucide-react";
import { useGameStore } from "@/store/useGameStore";
import type { GuHouseCombatProfile } from "@/store/slices/killerMovesSlice";

export function GuHousePanel() {
  const guHouses = useGameStore((s) => s.guHouses);
  const guHousesLoading = useGameStore((s) => s.guHousesLoading);
  const fetchGuHouses = useGameStore((s) => s.fetchGuHouses);
  const previewGuHouse = useGameStore((s) => s.previewGuHouse);
  const createGuHouse = useGameStore((s) => s.createGuHouse);
  const toggleEquip = useGameStore((s) => s.toggleEquipGuHouse);
  const repairHouse = useGameStore((s) => s.repairGuHouse);
  const gus = useGameStore((s) => s.killerMoveGus);
  const guDictionary = useGameStore((s) => s.guDictionary);

  const [name, setName] = useState("");
  const [selectedCoreIDs, setSelectedCoreIDs] = useState<string[]>([]);
  const [auxGuCount, setAuxGuCount] = useState<number>(30);
  const [preview, setPreview] = useState<GuHouseCombatProfile | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    void fetchGuHouses();
  }, [fetchGuHouses]);

  // Only Tiên Cổ (Rank >= 6) can be core hạch tâm
  const immortalGus = useMemo(() => {
    return gus.filter((g) => g.rank >= 6 && g.state !== "DEAD");
  }, [gus]);

  // Update preview
  useEffect(() => {
    if (selectedCoreIDs.length < 2) {
      setPreview(null);
      return;
    }
    let active = true;
    const timer = setTimeout(() => {
      void previewGuHouse({
        name: name || "Tiên Cổ Ốc",
        core_gu_instance_ids: selectedCoreIDs,
        aux_gu_count: auxGuCount,
      }).then((res) => {
        if (active) setPreview(res);
      });
    }, 200);
    return () => {
      active = false;
      clearTimeout(timer);
    };
  }, [selectedCoreIDs, auxGuCount, name, previewGuHouse]);

  const toggleCoreGu = (id: string) => {
    if (selectedCoreIDs.includes(id)) {
      setSelectedCoreIDs((prev) => prev.filter((item) => item !== id));
    } else {
      if (selectedCoreIDs.length >= 5) return;
      setSelectedCoreIDs((prev) => [...prev, id]);
    }
  };

  const handleCreate = async () => {
    if (!name.trim() || selectedCoreIDs.length < 2) return;
    setCreating(true);
    const success = await createGuHouse({
      name: name.trim(),
      core_gu_instance_ids: selectedCoreIDs,
      aux_gu_count: auxGuCount,
    });
    setCreating(false);
    if (success) {
      setName("");
      setSelectedCoreIDs([]);
      setAuxGuCount(30);
    }
  };

  return (
    <div className="space-y-6">
      {/* Intro Header */}
      <div className="rounded-2xl border border-amber-900/40 bg-zinc-950/80 p-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl border border-amber-500/30 bg-amber-950/40 text-amber-400">
            <Castle className="h-5 w-5" />
          </div>
          <div>
            <h3 className="text-base font-bold text-amber-200">Tiên Cổ Ốc (Immortal Gu House)</h3>
            <p className="text-xs text-zinc-400">
              Sát chiêu Tiên đạo thực thể hóa thành chiến hạm / pháo đài. Hấp thụ toàn bộ sát thương và miễn nhiễm phản phệ cho Cổ Tiên.
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Crafting Section */}
        <section className="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-950/60 p-5">
          <div className="flex items-center justify-between border-b border-zinc-800/80 pb-3">
            <h4 className="flex items-center gap-2 text-sm font-bold text-zinc-100">
              <Sparkles className="h-4 w-4 text-amber-400" />
              Đúc Kết Tiên Cổ Ốc
            </h4>
            <span className="font-mono text-xs text-zinc-500">
              {selectedCoreIDs.length}/5 Tiên Cổ
            </span>
          </div>

          <div>
            <label className="text-xs text-zinc-400">Tên Cổ Ốc</label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="VD: Vấn Đạo Lâu, Vạn Niên Đấu Phi Xa..."
              className="mt-1 w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3.5 py-2 text-xs text-zinc-100 outline-none focus:border-amber-500"
            />
          </div>

          <div>
            <label className="text-xs text-zinc-400">
              Chọn Tiên Cổ Hạch Tâm (Cần 2 đến 5 Tiên Cổ Rank 6+)
            </label>
            {immortalGus.length === 0 ? (
              <div className="mt-2 rounded-xl border border-dashed border-zinc-800 p-4 text-center text-xs text-zinc-600">
                Bạn chưa sở hữu Tiên Cổ (Rank 6 trở lên) để làm trụ cột Cổ Ốc.
              </div>
            ) : (
              <div className="mt-2 max-h-44 space-y-1.5 overflow-y-auto">
                {immortalGus.map((gu) => {
                  const spec = guDictionary[gu.gu_id];
                  const selected = selectedCoreIDs.includes(gu.instance_id);
                  return (
                    <button
                      key={gu.instance_id}
                      type="button"
                      onClick={() => toggleCoreGu(gu.instance_id)}
                      className={`flex w-full items-center justify-between rounded-xl border px-3 py-2 text-xs transition-colors ${
                        selected
                          ? "border-amber-600 bg-amber-950/40 text-amber-200"
                          : "border-zinc-800 bg-zinc-900/60 text-zinc-300 hover:border-zinc-700"
                      }`}
                    >
                      <span className="font-semibold">{spec?.name || gu.gu_id} · Rank {gu.rank}</span>
                      {selected ? <Check className="h-4 w-4 text-amber-400" /> : <Plus className="h-4 w-4 text-zinc-600" />}
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <div>
            <div className="flex items-center justify-between">
              <label className="text-xs text-zinc-400">Số lượng Phàm Cổ làm vỏ ốc</label>
              <span className="font-mono text-xs text-amber-300">{auxGuCount} Cổ</span>
            </div>
            <input
              type="range"
              min={10}
              max={150}
              step={5}
              value={auxGuCount}
              onChange={(e) => setAuxGuCount(Number(e.target.value))}
              className="mt-2 w-full accent-amber-500"
            />
            <p className="mt-1 text-[10px] text-zinc-500">
              Phàm Cổ cấu thành lớp giáp ngoài, trực tiếp gia tăng Máu (HP) và Phòng Thủ cho Cổ Ốc.
            </p>
          </div>

          {preview && (
            <div className="rounded-xl border border-amber-900/40 bg-amber-950/20 p-3.5 space-y-2 text-xs">
              <div className="font-bold text-amber-300">Dự Tính Chỉ Số Cổ Ốc:</div>
              <div className="grid grid-cols-3 gap-2 text-[11px] font-mono">
                <div className="rounded-lg bg-zinc-900/80 p-2">
                  <div className="text-zinc-500 text-[10px]">Độ Bền / Máu</div>
                  <div className="text-emerald-400 font-bold mt-0.5">{preview.max_house_hp} HP</div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-2">
                  <div className="text-zinc-500 text-[10px]">Giáp Phòng Ngự</div>
                  <div className="text-cyan-400 font-bold mt-0.5">{preview.house_defense} Giáp</div>
                </div>
                <div className="rounded-lg bg-zinc-900/80 p-2">
                  <div className="text-zinc-500 text-[10px]">Kháng Sát Thương</div>
                  <div className="text-amber-400 font-bold mt-0.5">{Math.round(preview.damage_reduction * 100)}%</div>
                </div>
              </div>
              <div className="text-[11px] text-zinc-400">
                Tích hợp {preview.actions?.length || 0} Trận Kích (Sát chiêu đặc hữu của Tiên Cổ Ốc).
              </div>
            </div>
          )}

          <button
            onClick={() => void handleCreate()}
            disabled={creating || selectedCoreIDs.length < 2 || !name.trim()}
            className="w-full rounded-xl bg-amber-600 py-2.5 text-xs font-bold text-white shadow-lg shadow-amber-950/40 hover:bg-amber-500 disabled:opacity-40"
          >
            {creating ? "Đang đúc kết…" : "Đúc Kết Tiên Cổ Ốc"}
          </button>
        </section>

        {/* Owned List */}
        <section className="space-y-4">
          <div className="flex items-center gap-2 border-b border-zinc-800/80 pb-3 text-sm font-bold text-zinc-100">
            <Castle className="h-4 w-4 text-cyan-400" />
            Tiên Cổ Ốc Sở Hữu
          </div>

          {guHousesLoading ? (
            <div className="py-8 text-center text-xs text-zinc-500">Đang tải Tiên Cổ Ốc…</div>
          ) : guHouses.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-zinc-800 p-8 text-center text-xs text-zinc-500">
              Chưa có Tiên Cổ Ốc nào. Hãy đúc kết một tòa pháo đài để mang vào chiến trận!
            </div>
          ) : (
            <div className="space-y-3">
              {guHouses.map((house) => (
                <article
                  key={house.id}
                  className="rounded-2xl border border-zinc-800 bg-zinc-950/70 p-4 space-y-3"
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h5 className="text-sm font-bold text-amber-200">{house.name}</h5>
                      <p className="text-[11px] text-zinc-400">
                        Rank {house.rank} · {house.core_gu_instances.length} Tiên Cổ · {house.aux_gu_count} Phàm Cổ
                      </p>
                    </div>
                    {house.is_equipped && (
                      <span className="rounded-full border border-emerald-900 bg-emerald-950/50 px-2.5 py-0.5 text-[10px] font-semibold text-emerald-300">
                        Đang xuất chiến
                      </span>
                    )}
                  </div>

                  {/* Durability bar */}
                  <div>
                    <div className="flex justify-between text-[10px] font-mono text-zinc-400">
                      <span>Độ Bền Pháo Đài</span>
                      <span className={house.current_durability < 40 ? "text-red-400" : "text-emerald-400"}>
                        {Math.round(house.current_durability)}%
                      </span>
                    </div>
                    <div className="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-zinc-800">
                      <div
                        className={`h-full transition-all ${
                          house.current_durability < 40 ? "bg-red-500" : "bg-emerald-500"
                        }`}
                        style={{ width: `${house.current_durability}%` }}
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-3 gap-2 text-[10px] font-mono text-zinc-400">
                    <div className="rounded-lg bg-zinc-900 p-1.5 text-center">
                      Máu: <span className="text-emerald-300">{house.house_profile.max_house_hp}</span>
                    </div>
                    <div className="rounded-lg bg-zinc-900 p-1.5 text-center">
                      Giáp: <span className="text-cyan-300">{house.house_profile.house_defense}</span>
                    </div>
                    <div className="rounded-lg bg-zinc-900 p-1.5 text-center">
                      Giảm ST: <span className="text-amber-300">{Math.round(house.house_profile.damage_reduction * 100)}%</span>
                    </div>
                  </div>

                  <div className="flex gap-2 pt-1">
                    <button
                      onClick={() => void toggleEquip(house.id)}
                      className={`flex-1 rounded-xl border py-2 text-xs font-bold transition-colors ${
                        house.is_equipped
                          ? "border-zinc-700 bg-zinc-900 text-zinc-300"
                          : "border-emerald-900 bg-emerald-950/40 text-emerald-300 hover:bg-emerald-950/70"
                      }`}
                    >
                      {house.is_equipped ? "Thu Hồi" : "Xuất Chiến"}
                    </button>

                    {house.current_durability < 100 && (
                      <button
                        onClick={() => void repairHouse(house.id)}
                        className="flex items-center gap-1.5 rounded-xl border border-sky-900/60 bg-sky-950/30 px-3 py-2 text-xs font-semibold text-sky-200 hover:bg-sky-950/60"
                      >
                        <Wrench className="h-3.5 w-3.5" />
                        Tu Sửa
                      </button>
                    )}
                  </div>
                </article>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
