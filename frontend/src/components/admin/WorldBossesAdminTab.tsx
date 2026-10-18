"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { HeartPulse, RefreshCw, Search, Skull, Trash2, X } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const REFRESH_MS = 10_000;

type WorldBossRow = {
  region_id: string;
  instance_id: string;
  spawn_id: string;
  monster_template_id: string;
  name: string;
  rank: number;
  sub_rank: number;
  x: number;
  y: number;
  spawn_x: number;
  spawn_y: number;
  hp: number;
  max_hp: number;
  state: string;
  target_id?: string;
  respawn_seconds: number;
  respawn_at?: string;
  revision: number;
  last_update_at: string;
};

const STATE_LABEL: Record<string, string> = {
  IDLE: "Đứng yên",
  CHASE: "Truy đuổi",
  ATTACK: "Giao chiến",
  RETURN: "Hồi vị",
  DEFEATED: "Đã hạ",
};

export const WorldBossesAdminTab: React.FC<{
  addToast: (message: string, type: "success" | "error" | "info") => void;
  canEdit: boolean;
  canDelete: boolean;
}> = ({ addToast, canEdit, canDelete }) => {
  const [rows, setRows] = useState<WorldBossRow[]>([]);
  const [bossRegions, setBossRegions] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [regionFilter, setRegionFilter] = useState("");
  const [searchText, setSearchText] = useState("");
  const [edit, setEdit] = useState<WorldBossRow | null>(null);
  const [form, setForm] = useState({ x: 0, y: 0, hp: 0 });
  const [busyId, setBusyId] = useState("");

  const load = useCallback(async (silent = false) => {
    setLoading(true);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/world-bosses`);
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể tải boss đang sống");
      setRows(Array.isArray(result.data) ? result.data : []);
      setBossRegions(Array.isArray(result.boss_regions) ? result.boss_regions : []);
    } catch (error) {
      // Polling failures must not spam a toast every refresh interval.
      if (!silent) addToast(error instanceof Error ? error.message : "Lỗi tải boss", "error");
    } finally {
      setLoading(false);
    }
  }, [addToast]);

  useEffect(() => {
    void load();
    const timer = setInterval(() => void load(true), REFRESH_MS);
    return () => clearInterval(timer);
  }, [load]);

  const regions = useMemo(
    () => [...new Set([...bossRegions, ...rows.map((row) => row.region_id)])].sort(),
    [bossRegions, rows],
  );

  const visible = useMemo(() => {
    const needle = searchText.trim().toLowerCase();
    return rows.filter((row) => {
      if (regionFilter && row.region_id !== regionFilter) return false;
      if (!needle) return true;
      return [row.name, row.monster_template_id, row.region_id, row.instance_id, row.spawn_id]
        .some((value) => String(value).toLowerCase().includes(needle));
    });
  }, [rows, regionFilter, searchText]);

  const update = async (boss: WorldBossRow, payload: Record<string, unknown>, label: string) => {
    const action = prepareAdminAction("updateWorldBoss", label, `${boss.name} · ${boss.region_id} (${boss.instance_id})`);
    if (!action) return;
    setBusyId(boss.instance_id);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/world-bosses/${encodeURIComponent(boss.region_id)}/${encodeURIComponent(boss.instance_id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey, ...(action.reason ? { "X-Admin-Reason": action.reason } : {}) },
        body: JSON.stringify(payload),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || `Không thể ${label}`);
      addToast(result.message || `Đã ${label}`, "success");
      setEdit(null);
      await load();
    } catch (error) {
      addToast(error instanceof Error ? error.message : `Lỗi ${label}`, "error");
    } finally {
      setBusyId("");
    }
  };

  const beginEdit = (boss: WorldBossRow) => {
    // Keep the live float values so an unchanged save cannot nudge the boss.
    setForm({ x: boss.x, y: boss.y, hp: boss.hp });
    setEdit(boss);
  };

  const saveEdit = async () => {
    if (!edit) return;
    await update(edit, { x: form.x, y: form.y, hp: form.hp }, "cập nhật boss");
  };

  const spawnRegion = async () => {
    if (!regionFilter) return;
    const action = prepareAdminAction("spawnWorldBoss", "spawn/hồi sinh boss từ ô boss", regionFilter);
    if (!action) return;
    setLoading(true);
    try {
      const response = await authFetch(`${BACKEND_URL}/api/admin/world-bosses/spawn`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ region_id: regionFilter }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể spawn boss");
      addToast(result.message || "Đã spawn boss", "success");
      if (Array.isArray(result.errors) && result.errors.length > 0) {
        addToast(result.errors.slice(0, 3).join("; "), "info");
      }
      await load();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi spawn boss", "error");
    } finally {
      setLoading(false);
    }
  };

  const purgeRegion = async () => {
    if (!regionFilter) return;
    const endpoint = `${BACKEND_URL}/api/admin/world-bosses?region_id=${encodeURIComponent(regionFilter)}`;
    const action = prepareAdminAction("purgeWorldBosses", "xoá toàn bộ boss đang sống", regionFilter);
    if (!action) return;
    setLoading(true);
    try {
      const response = await authFetch(endpoint, {
        method: "DELETE",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ ...action, confirmation_target: new URL(endpoint).pathname }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.error || "Không thể xoá boss khu vực");
      addToast(result.message || "Đã xoá boss khu vực", "success");
      await load();
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Lỗi xoá boss khu vực", "error");
    } finally {
      setLoading(false);
    }
  };

  const time = (value?: string) => (value ? new Date(value).toLocaleString("vi-VN") : "—");
  const stateBadge = (row: WorldBossRow) => {
    const dead = row.state === "DEFEATED";
    return <span className={`inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-semibold ${dead ? "bg-zinc-800 text-zinc-400" : row.state === "ATTACK" || row.state === "CHASE" ? "bg-red-500/10 text-red-300" : "bg-emerald-500/10 text-emerald-300"}`}>
      {STATE_LABEL[row.state] ?? row.state}
    </span>;
  };

  return <section className="rounded-xl border border-zinc-800 bg-zinc-900/30 p-4 space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h2 className="flex items-center gap-2 text-sm font-bold text-zinc-100"><Skull className="h-4 w-4 text-red-400" />Boss thế giới đang sống</h2>
        <p className="mt-1 text-xs text-zinc-500">Dữ liệu runtime trong Redis: vị trí, HP, mục tiêu, hồi sinh. Tự làm mới mỗi {REFRESH_MS / 1000}s.</p>
      </div>
      <div className="flex items-center gap-2">
        {canEdit && <button type="button" onClick={() => void spawnRegion()} disabled={!regionFilter || loading} title="Tạo mới hoặc hồi sinh boss từ ô boss (map_cells) của khu vực đang lọc" className="rounded-lg border border-emerald-900/60 px-3 py-2 text-xs text-emerald-300 hover:bg-emerald-950/40 disabled:opacity-40">Spawn từ ô boss</button>}
        {canDelete && <button type="button" onClick={() => void purgeRegion()} disabled={!regionFilter || loading} title="Xoá mọi boss đang sống của khu vực đang lọc (snapshot sau sẽ tạo lại)" className="rounded-lg border border-red-900/60 px-3 py-2 text-xs text-red-300 hover:bg-red-950/40 disabled:opacity-40"><Trash2 className="mr-1.5 inline h-3.5 w-3.5" />Xoá boss khu vực</button>}
        <button type="button" onClick={() => void load()} disabled={loading} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-800 disabled:opacity-50"><RefreshCw className={`mr-1.5 inline h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />Làm mới</button>
      </div>
    </div>

    <div className="flex flex-wrap gap-2">
      <label className="flex min-w-[240px] flex-1 items-center gap-2 rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2"><Search className="h-3.5 w-3.5 text-zinc-500" /><input value={searchText} onChange={(event) => setSearchText(event.target.value)} placeholder="Tìm tên boss, khu vực, instance, spawn" className="w-full bg-transparent text-xs text-zinc-200 outline-none" /></label>
      <select value={regionFilter} onChange={(event) => setRegionFilter(event.target.value)} className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-2 text-xs text-zinc-300"><option value="">Tất cả khu vực ({regions.length})</option>{regions.map((region) => <option key={region} value={region}>{region}</option>)}</select>
    </div>

    <div className="overflow-x-auto rounded-lg border border-zinc-800">
      <table className="w-full min-w-[980px] text-left text-xs">
        <thead className="bg-zinc-950 text-[10px] uppercase text-zinc-500">
          <tr><th className="px-3 py-2.5">Boss</th><th className="px-3 py-2.5">Vị trí hiện tại</th><th className="px-3 py-2.5">HP</th><th className="px-3 py-2.5">Trạng thái</th><th className="px-3 py-2.5">Hồi sinh</th><th className="px-3 py-2.5 text-right">Thao tác</th></tr>
        </thead>
        <tbody className="divide-y divide-zinc-800/70">
          {visible.length === 0 ? <tr><td colSpan={6} className="py-10 text-center text-zinc-500">{loading ? "Đang tải…" : "Không có boss nào đang sống khớp bộ lọc"}</td></tr> : visible.map((boss) => {
            const dead = boss.state === "DEFEATED";
            const pct = boss.max_hp > 0 ? Math.max(0, Math.min(100, (boss.hp / boss.max_hp) * 100)) : 0;
            const busy = busyId === boss.instance_id;
            return <tr key={`${boss.region_id}:${boss.instance_id}`} className="hover:bg-zinc-900/70">
              <td className="px-3 py-3"><span className="font-semibold text-zinc-200">{boss.name}</span><span className="mt-0.5 block text-[10px] text-zinc-500">{boss.monster_template_id} · Chuyển {boss.rank}.{boss.sub_rank}</span><span className="block font-mono text-[9px] text-zinc-600">{boss.instance_id}</span></td>
              <td className="px-3 py-3 text-zinc-300"><span className="font-semibold">{boss.region_id}</span><span className="mt-0.5 block text-[10px] text-zinc-500">({Math.round(boss.x)}, {Math.round(boss.y)}) · spawn ({boss.spawn_x}, {boss.spawn_y})</span></td>
              <td className="px-3 py-3"><span className={dead ? "text-zinc-500" : "text-zinc-200"}>{Math.round(boss.hp)} / {Math.round(boss.max_hp)}</span><span className="mt-1 block h-1.5 w-28 overflow-hidden rounded bg-zinc-800"><span className={`block h-full ${pct < 30 ? "bg-red-500" : "bg-emerald-500"}`} style={{ width: `${pct}%` }} /></span></td>
              <td className="px-3 py-3">{stateBadge(boss)}{boss.target_id ? <span className="mt-1 block font-mono text-[9px] text-zinc-500">target {boss.target_id.slice(0, 8)}</span> : null}</td>
              <td className="px-3 py-3 text-zinc-400"><span>{boss.respawn_seconds}s</span><span className="mt-0.5 block text-[10px] text-zinc-500">{time(boss.respawn_at)}</span><span className="block text-[10px] text-zinc-600">rev {boss.revision} · {time(boss.last_update_at)}</span></td>
              <td className="px-3 py-3 text-right whitespace-nowrap">
                {canEdit && <>
                  <button type="button" disabled={busy} onClick={() => beginEdit(boss)} title="Sửa vị trí / HP" className="mr-1 rounded border border-zinc-700 px-2 py-1 text-[10px] text-zinc-300 hover:bg-zinc-800 disabled:opacity-40">Sửa</button>
                  <button type="button" disabled={busy || dead} onClick={() => void update(boss, { hp: 0 }, "hạ gục boss")} title="Hạ gục (bắt đầu đếm hồi sinh)" className="mr-1 rounded border border-red-900/60 px-2 py-1 text-[10px] text-red-300 hover:bg-red-950/40 disabled:opacity-40">Hạ gục</button>
                  <button type="button" disabled={busy || !dead} onClick={() => void update(boss, { state: "IDLE" }, "hồi sinh boss")} title="Hồi sinh ngay tại điểm spawn, đầy HP" className="rounded border border-emerald-900/60 px-2 py-1 text-[10px] text-emerald-300 hover:bg-emerald-950/40 disabled:opacity-40">Hồi sinh</button>
                </>}
                {!canEdit && <span className="text-[10px] text-zinc-600">Chỉ xem</span>}
              </td>
            </tr>;
          })}
        </tbody>
      </table>
    </div>

    <p className="text-[10px] text-zinc-600">Tổng {visible.length} / {rows.length} boss đang sống. Boss chưa materialize không nằm trong bảng — chọn khu vực rồi bấm "Spawn từ ô boss" để tạo từ ô boss (map_cells).</p>

    {edit && <div className="fixed inset-0 z-50 flex items-center justify-center bg-zinc-950/80 p-4 backdrop-blur-sm"><div className="w-full max-w-lg space-y-4 rounded-2xl border border-zinc-800 bg-zinc-950 p-5">
      <div className="flex items-start justify-between"><div><h3 className="text-sm font-bold text-zinc-100">Sửa boss · {edit.name}</h3><p className="mt-1 font-mono text-[10px] text-zinc-600">{edit.region_id} / {edit.instance_id}</p></div><button type="button" onClick={() => setEdit(null)} className="rounded p-1.5 text-zinc-400 hover:bg-zinc-800"><X className="h-4 w-4" /></button></div>
      <div className="grid gap-3 sm:grid-cols-3">
        <Field label="X" type="number" value={form.x} onChange={(value) => setForm({ ...form, x: Number(value) })} />
        <Field label="Y" type="number" value={form.y} onChange={(value) => setForm({ ...form, y: Number(value) })} />
        <Field label={`HP (0-${Math.round(edit.max_hp)})`} type="number" value={form.hp} onChange={(value) => setForm({ ...form, hp: Number(value) })} />
      </div>
      <p className="text-[10px] text-zinc-500">HP = 0 sẽ chuyển boss sang trạng thái đã hạ và bắt đầu hồi sinh sau {edit.respawn_seconds}s. Toạ độ phải nằm trong khu vực.</p>
      <div className="flex justify-end gap-2"><button type="button" onClick={() => setEdit(null)} className="rounded-lg border border-zinc-700 px-3 py-2 text-xs text-zinc-300 hover:bg-zinc-800">Huỷ</button><button type="button" disabled={busyId === edit.instance_id} onClick={() => void saveEdit()} className="rounded-lg bg-amber-500 px-3 py-2 text-xs font-semibold text-zinc-950 hover:bg-amber-400 disabled:opacity-50"><HeartPulse className="mr-1.5 inline h-3.5 w-3.5" />Lưu</button></div>
    </div></div>}
  </section>;
};

const Field: React.FC<{ label: string; value: unknown; type?: string; onChange: (value: string) => void }> = ({ label, value, type = "text", onChange }) => <label className="block space-y-1"><span className="text-[10px] font-semibold uppercase text-zinc-500">{label}</span><input type={type} value={String(value ?? "")} onChange={(event) => onChange(event.target.value)} className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-200 outline-none focus:border-amber-500/40" /></label>;
