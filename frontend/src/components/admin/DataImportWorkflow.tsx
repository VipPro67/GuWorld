"use client";

import { useState } from "react";
import { AlertTriangle, CheckCircle2, Database, Upload } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { AdminConfirmDialog } from "./AdminConfirmDialog";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

// Preflight/commit chạy toàn bộ import đồng bộ trong một request; dataset lớn
// (Gu.csv ~4400 dòng) vượt xa timeout mặc định 30s của authFetch.
const IMPORT_TIMEOUT_MS = 300000;

export const IMPORT_DATASETS = [
  ["item", "Vật phẩm", "Item.csv"], ["gu", "Cổ trùng", "Gu.csv"],
  ["constitution", "Thể chất", "Constitution.csv"], ["race", "Chủng tộc", "Race.csv"],
  ["status", "Trạng thái", "Status.csv"], ["recipe", "Cổ phương", "Recipe.csv"],
  ["shop_item", "Cửa hàng", "ShopItem.csv"], ["arena_shop_item", "Cửa hàng Đấu trường", "ArenaShopItem.csv"],
  ["region", "Khu vực", "Region.csv"], ["loot_table", "Bảng loot", "LootTable.csv"],
  ["monster", "Quái vật", "Monster.csv"], ["monster_spawn", "Spawn quái", "MonsterSpawn.csv"],
  ["map_cell", "Ô bản đồ", "MapCell.csv"], ["sect_mission", "Nhiệm vụ tông môn", "SectMission.csv"],
  ["quest", "Nhiệm vụ", "Quest.csv"], ["dream_realm", "Mộng cảnh", "DreamRealm.csv"],
] as const;

type Issue = { dataset: string; row?: number; field?: string; message: string; severity: string };
type RowDiff = { kind: string; before?: Record<string, unknown>; after?: Record<string, unknown> };
type DatasetResult = { key: string; filename: string; rows: number; inserts: number; updates: number; unchanged: number; diffs: RowDiff[]; validation_errors: Issue[]; reload_required: boolean; restart_required: boolean };
type Preflight = { preflight_id: string; expires_at: string; plan: { can_commit: boolean; totals: { rows: number; inserts: number; updates: number; unchanged: number }; datasets: DatasetResult[]; validation_errors: Issue[] } };

const sameValue = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);
const changedFields = (before: Record<string, unknown> = {}, after: Record<string, unknown> = {}) =>
  Array.from(new Set([...Object.keys(before), ...Object.keys(after)]))
    .filter(key => !sameValue(before[key], after[key]))
    .map(key => ({ key, before: before[key], after: after[key] }));
const fmt = (value: unknown) => value === undefined || value === null ? "—" : typeof value === "string" ? value : JSON.stringify(value);

// Chỉ render phần thay đổi: update = bảng field cũ→mới, insert = thu gọn.
function RowDiffView({ diffs }: { diffs: RowDiff[] }) {
  return <div className="mt-2 space-y-2">
    {diffs.map((diff, index) => {
      const identity = fmt(diff.after?.id ?? diff.before?.id ?? "") || "…";
      if (diff.kind === "insert") {
        const fields = Object.entries(diff.after ?? {}).filter(([, value]) => value !== null && value !== "");
        return <details key={index} className="rounded border border-emerald-900/50 bg-emerald-950/10 p-2">
          <summary className="cursor-pointer text-emerald-400">+ thêm mới {identity}</summary>
          <div className="mt-1 space-y-0.5">{fields.map(([key, value]) => <div key={key} className="flex gap-2"><span className="w-40 shrink-0 text-zinc-500">{key}</span><span className="break-all text-zinc-300">{fmt(value)}</span></div>)}</div>
        </details>;
      }
      const changes = changedFields(diff.before, diff.after);
      return <div key={index} className="rounded border border-amber-900/50 bg-amber-950/10 p-2">
        <div className="text-amber-400">~ sửa {identity} · {changes.length} field</div>
        <table className="mt-1 w-full border-collapse"><tbody>{changes.map(change => <tr key={change.key} className="align-top"><td className="w-32 py-0.5 pr-2 text-zinc-500">{change.key}</td><td className="break-all py-0.5 pr-2 text-red-400">{fmt(change.before)}</td><td className="break-all py-0.5 text-emerald-400">{fmt(change.after)}</td></tr>)}</tbody></table>
      </div>;
    })}
  </div>;
}

export function DataImportWorkflow({ canImportDataset, canImportAll, addToast }: { canImportDataset: boolean; canImportAll: boolean; addToast: (message: string, type: "success" | "error" | "info") => void }) {
  const [scope, setScope] = useState<"dataset" | "all">("dataset");
  const [dataset, setDataset] = useState<string>(IMPORT_DATASETS[0][0]);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preflight | null>(null);
  const [loading, setLoading] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [reason, setReason] = useState("");

  const clearPreview = () => { setPreview(null); setReason(""); };
  const preflight = async () => {
    setLoading(true); clearPreview();
    try {
      const body = new FormData();
      if (scope === "dataset" && file) body.append("file", file);
      const path = scope === "all" ? "/api/admin/data/import" : `/api/admin/data/import/${dataset}`;
      const response = await authFetch(`${BACKEND_URL}${path}`, { method: "POST", body, timeoutMs: IMPORT_TIMEOUT_MS });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Preflight thất bại");
      setPreview(data);
      if (!data.plan.can_commit) addToast("Preflight có lỗi nghiêm trọng; database chưa bị thay đổi.", "error");
    } catch (error) { addToast(error instanceof Error ? error.message : String(error), "error"); }
    finally { setLoading(false); }
  };
  const commit = async () => {
    if (!preview || !reason.trim()) return;
    setLoading(true);
    try {
      const path = scope === "all" ? "/api/admin/data/import/commit" : `/api/admin/data/import/${dataset}/commit`;
	  const target = scope === "all" ? "all" : dataset;
	  const response = await authFetch(`${BACKEND_URL}${path}`, { method: "POST", headers: { "Content-Type": "application/json", "Idempotency-Key": preview.preflight_id }, body: JSON.stringify({ preflight_id: preview.preflight_id, reason: reason.trim(), confirmation_target: target }), timeoutMs: IMPORT_TIMEOUT_MS });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Commit import thất bại");
      addToast(data.reload_ok ? "Import đã commit và registry/cache đã tải lại." : "Import đã commit; cần restart do reload chưa hoàn tất.", data.reload_ok ? "success" : "info");
      setConfirming(false); clearPreview(); setFile(null);
    } catch (error) { addToast(error instanceof Error ? error.message : String(error), "error"); }
    finally { setLoading(false); }
  };
  const allowed = scope === "all" ? canImportAll : canImportDataset;

  return <div className="space-y-5">
    <div className="rounded-xl border border-zinc-800/60 bg-zinc-900/40 p-6">
      <h3 className="mb-2 flex items-center gap-2 text-sm font-serif font-bold text-zinc-200"><Database className="h-4 w-4 text-emerald-400" />Workflow import có kiểm soát</h3>
      <p className="mb-5 text-xs text-zinc-500">Chọn nguồn → parse/validate → dry-run → xem diff → xác nhận → commit transaction → reload cache → audit.</p>
      <div className="grid gap-4 md:grid-cols-3">
        <label className="text-xs text-zinc-400">Phạm vi
          <select value={scope} onChange={event => { setScope(event.target.value as "dataset" | "all"); setFile(null); clearPreview(); }} className="mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-200">
            <option value="dataset">Một dataset</option>{canImportAll && <option value="all">Toàn bộ dataset (rủi ro cao)</option>}
          </select>
        </label>
        {scope === "dataset" && <label className="text-xs text-zinc-400">Dataset
          <select value={dataset} onChange={event => { setDataset(event.target.value); clearPreview(); }} className="mt-1 w-full rounded-lg border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-200">{IMPORT_DATASETS.map(([key, label, csv]) => <option key={key} value={key}>{label} — {csv}</option>)}</select>
        </label>}
        {scope === "dataset" && <label className="text-xs text-zinc-400">Nguồn CSV (không chọn = file server)
          <input type="file" accept=".csv,text/csv" onChange={event => { setFile(event.target.files?.[0] ?? null); clearPreview(); }} className="mt-1 block w-full text-xs text-zinc-400 file:mr-2 file:rounded file:border-0 file:bg-zinc-700 file:px-2 file:py-2 file:text-zinc-100" />
        </label>}
      </div>
      <button disabled={!allowed || loading} onClick={preflight} className="mt-5 flex items-center gap-2 rounded-lg bg-emerald-600 px-4 py-2 text-xs font-bold text-white disabled:opacity-40"><Upload className="h-4 w-4" />{loading ? "Đang phân tích…" : "Parse, validate & dry-run"}</button>
      {!allowed && <p className="mt-2 text-xs text-red-400">Bạn không có permission phù hợp cho phạm vi import này.</p>}
    </div>

    {preview && <div className="rounded-xl border border-zinc-700 bg-zinc-900/60 p-5">
      <div className="mb-4 flex items-center justify-between"><h4 className="text-sm font-bold text-zinc-100">Kết quả preflight</h4><span className="text-[10px] text-zinc-500">Hết hạn: {new Date(preview.expires_at).toLocaleString()}</span></div>
      <div className="grid grid-cols-2 gap-2 md:grid-cols-4">{([['Rows', preview.plan.totals.rows], ['Inserts', preview.plan.totals.inserts], ['Updates', preview.plan.totals.updates], ['Unchanged', preview.plan.totals.unchanged]] as const).map(([label, value]) => <div key={label} className="rounded-lg bg-zinc-950 p-3"><div className="text-[10px] uppercase text-zinc-500">{label}</div><div className="text-xl font-bold text-zinc-100">{value}</div></div>)}</div>
      {(preview.plan.validation_errors ?? []).length > 0 && <div className="mt-4 rounded-lg border border-red-900/60 bg-red-950/30 p-3 text-xs text-red-300"><div className="mb-2 font-bold">Validation errors — không ghi database</div>{(preview.plan.validation_errors ?? []).map((issue, index) => <div key={index}>{issue.dataset}{issue.row ? ` dòng ${issue.row}` : ""}{issue.field ? ` [${issue.field}]` : ""}: {issue.message}</div>)}</div>}
      <div className="mt-4 max-h-80 space-y-3 overflow-auto">{(preview.plan.datasets ?? []).map(item => { const diffs = item.diffs ?? []; return <div key={item.key} className="rounded-lg border border-zinc-800 p-3 text-xs"><div className="flex flex-wrap gap-3 font-semibold text-zinc-200"><span>{item.filename}</span><span className="text-emerald-400">+{item.inserts}</span><span className="text-amber-400">~{item.updates}</span><span className="text-zinc-500">={item.unchanged}</span>{item.reload_required && <span className="text-sky-400">reload registry/cache</span>}{item.restart_required && <span className="text-red-400">cần restart</span>}</div>{diffs.length > 0 && <details className="mt-2"><summary className="cursor-pointer text-zinc-400">Xem diff ({diffs.length}{diffs.length === 200 ? "+" : ""})</summary><RowDiffView diffs={diffs} /></details>}</div>; })}</div>
      <div className="mt-4 rounded-lg border border-amber-800/50 bg-amber-950/20 p-3 text-xs text-amber-300"><AlertTriangle className="mr-2 inline h-4 w-4" />Sau commit, server sẽ reload registry và invalidate cache. Dataset đánh dấu “cần restart” hoặc reload lỗi phải được restart trước khi dữ liệu có hiệu lực đầy đủ.</div>
      <button disabled={!preview.plan.can_commit} onClick={() => setConfirming(true)} className="mt-4 flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-xs font-bold text-white disabled:opacity-40"><CheckCircle2 className="h-4 w-4" />Xác nhận commit transaction</button>
    </div>}

	<AdminConfirmDialog open={confirming} title={`Commit dữ liệu “${scope === "all" ? "all" : dataset}” đã dry-run?`} description="Thao tác sẽ ghi database trong một transaction và được audit." riskLevel={scope === "all" ? "critical" : "high"} loading={loading} confirmDisabled={!reason.trim()} confirmLabel="Commit import" onCancel={() => setConfirming(false)} onConfirm={commit}>
      <div className="space-y-3"><p>Diff phía sau đã được kiểm tra. Registry/cache sẽ được reload sau khi transaction commit; nếu reload lỗi, hệ thống sẽ cảnh báo restart.</p><label className="block text-xs text-zinc-400">Lý do bắt buộc<input value={reason} onChange={event => setReason(event.target.value)} className="mt-1 w-full rounded border border-zinc-700 bg-zinc-950 px-3 py-2 text-zinc-100" /></label></div>
    </AdminConfirmDialog>
  </div>;
}
