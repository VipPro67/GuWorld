"use client";

import { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";
import { useGameStore } from "@/store/useGameStore";

interface RespawnOption {
  kind: string;
  label: string;
  region: string;
  x: number;
  y: number;
}

export function RespawnChoiceModal() {
  const player = useGameStore((state) => state.player);
  const token = useGameStore((state) => state.token);
  const [options, setOptions] = useState<RespawnOption[]>([]);
  const [pending, setPending] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!player || !token) return;
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/respawn-choice`);
      if (!response.ok) return;
      const data = await response.json();
      setPending(Boolean(data.pending));
      setOptions(Array.isArray(data.options) ? data.options : []);
    } catch {
      // Retry on the next poll; the choice is persisted server-side.
    }
  }, [player, token]);

  useEffect(() => {
    if (!player || !token) {
      setPending(false);
      return;
    }
    void load();
    // Respawn choices are only created on death. Keep a modest foreground
    // check so the modal appears promptly, and stop spending requests in a
    // background tab.
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") void load();
    }, 15000);
    return () => window.clearInterval(timer);
  }, [player?.id, token, load]);

  useEffect(() => {
    if (!player || !token) return;
    const onVisible = () => {
      if (document.visibilityState === "visible") void load();
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => document.removeEventListener("visibilitychange", onVisible);
  }, [player?.id, token, load]);

  const choose = async (kind: string) => {
    setSaving(true);
    setError("");
    try {
      const response = await authFetch(`${BACKEND_URL}/api/player/respawn-choice`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ kind }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không thể hồi sinh tại điểm đã chọn");
      setPending(false);
      await useGameStore.getState().refreshPlayerSummary();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Không thể hồi sinh tại điểm đã chọn");
      await load();
    } finally {
      setSaving(false);
    }
  };

  if (!pending) return null;
  return (
    <div className="fixed inset-0 z-[10001] flex items-center justify-center bg-black/85 p-4 backdrop-blur-sm">
      <div className="w-full max-w-md space-y-4 rounded-2xl border border-amber-700/50 bg-zinc-950 p-6 shadow-2xl">
        <div className="text-center">
          <div className="mb-2 text-4xl">🕯️</div>
          <h2 className="text-xl font-serif font-bold text-amber-300">Chọn nơi hồi sinh</h2>
          <p className="mt-2 text-sm text-zinc-400">Đạo hữu đã được đưa về điểm an toàn. Chọn nơi hồi sinh trong 10 phút.</p>
        </div>
        <div className="space-y-2">
          {options.map((option) => (
            <button
              key={option.kind}
              disabled={saving}
              onClick={() => void choose(option.kind)}
              className="w-full rounded-xl border border-zinc-700 bg-zinc-900 px-4 py-3 text-left transition hover:border-amber-500 hover:bg-amber-950/30 disabled:opacity-50"
            >
              <span className="block font-semibold text-zinc-100">{option.label}</span>
              <span className="mt-1 block text-xs text-zinc-500">{option.region} · ({option.x}, {option.y})</span>
            </button>
          ))}
        </div>
        {error && <p className="text-center text-sm text-red-400">{error}</p>}
        <p className="text-center text-xs text-zinc-600">Máu khi hồi sinh: 1 HP</p>
      </div>
    </div>
  );
}
