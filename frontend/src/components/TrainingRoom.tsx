"use client";

import React, { useEffect, useState } from "react";
import { Swords, X } from "lucide-react";
import { useGameStore } from "../store/useGameStore";

interface TrainingRoomProps {
  onClose: () => void;
}

const defaults = { dummyHP: 100000, dummyShield: 0, dummyAtk: 10, dummyDef: 5, dummySpeed: 1 };

export const TrainingRoom: React.FC<TrainingRoomProps> = ({ onClose }) => {
  const store = useGameStore();
  const startPractice = useGameStore((state) => state.executePracticeCombat);
  const exitPractice = useGameStore((state) => state.exitPracticeCombat);
  const [dummyConfig, setDummyConfig] = useState(defaults);
  const [active, setActive] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    try {
      const saved = localStorage.getItem("guworld_training_dummy");
      if (saved) setDummyConfig({ ...defaults, ...JSON.parse(saved) });
    } catch {
      localStorage.removeItem("guworld_training_dummy");
    }
  }, []);

  const updateConfig = (key: keyof typeof defaults, value: number) => {
    const updated = { ...dummyConfig, [key]: value };
    setDummyConfig(updated);
    localStorage.setItem("guworld_training_dummy", JSON.stringify(updated));
  };

  const enter = async () => {
    setBusy(true);
    const session = await startPractice({}, dummyConfig);
    setActive(Boolean(session));
    setBusy(false);
  };

  const leave = async () => {
    setBusy(true);
    const result = await exitPractice();
    setBusy(false);
    if (result) {
      setActive(false);
      onClose();
    }
  };

  return (
    <div className={active ? "pointer-events-none fixed inset-0 z-50 flex items-start justify-end p-4" : "fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/75 p-4 backdrop-blur-sm"}>
      <section className={`pointer-events-auto w-full ${active ? "max-w-sm" : "max-w-xl"} rounded-2xl border border-[#a88952]/60 bg-[#150f0bf2] p-5 text-[#f4e8cb] shadow-2xl`}>
        <header className="mb-5 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Swords className="h-5 w-5 text-amber-300" />
            <div>
              <h2 className="font-serif text-xl font-semibold">Phòng tập realtime</h2>
              <p className="text-xs text-amber-100/60">Phiên riêng, không tiêu hao tài nguyên nhân vật</p>
            </div>
          </div>
          <button aria-label="Đóng phòng tập" onClick={active ? leave : onClose} className="rounded-lg p-2 hover:bg-white/10">
            <X className="h-4 w-4" />
          </button>
        </header>

        {!active ? (
          <>
            <div className="grid grid-cols-2 gap-3">
              {([
                ["dummyHP", "Sinh lực hình nhân", 1],
                ["dummyShield", "Khiên", 0],
                ["dummyAtk", "Công kích", 0],
                ["dummyDef", "Phòng thủ", 0],
                ["dummySpeed", "Tốc độ", 0],
              ] as const).map(([key, label, min]) => (
                <label key={key} className="text-sm text-amber-100/80">
                  {label}
                  <input
                    type="number"
                    min={min}
                    value={dummyConfig[key]}
                    onChange={(event) => updateConfig(key, Number(event.target.value))}
                    className="mt-1 w-full rounded-lg border border-[#a88952]/40 bg-black/30 px-3 py-2 text-white outline-none focus:border-amber-300"
                  />
                </label>
              ))}
            </div>
            <button disabled={busy} onClick={enter} className="mt-5 w-full rounded-xl bg-amber-500 px-4 py-3 font-semibold text-black transition hover:bg-amber-400 disabled:opacity-50">
              {busy ? "Đang tạo bản đồ riêng…" : "Vào tập luyện"}
            </button>
          </>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-amber-100/70">Bạn đang ở bản đồ tập luyện. Di chuyển và dùng hotbar để chiến đấu với hình nhân.</p>
            <button disabled={busy} onClick={leave} className="w-full rounded-xl border border-amber-300/50 px-4 py-3 font-semibold text-amber-100 hover:bg-amber-300/10 disabled:opacity-50">
              {busy ? "Đang rời bản đồ…" : "Kết thúc tập luyện"}
            </button>
          </div>
        )}
        {store.error && <p role="alert" className="mt-3 text-sm text-red-300">{store.error}</p>}
      </section>
    </div>
  );
};
