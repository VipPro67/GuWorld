import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { authFetch } from "@/lib/auth";
import { Activity, ChevronLeft, ChevronRight, Copy, RefreshCw, RotateCcw, Server } from "lucide-react";
import { useAdminUrlState } from "../useAdminUrlState";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
const TRACE_DEFAULTS = { page: "1", page_size: "25", method: "", status: "", path: "", search: "", min_duration_ms: "", sort: "timestamp", direction: "desc" };
const TRACE_KEYS = Object.keys(TRACE_DEFAULTS);

interface Endpoint { method: string; path: string; count: number; error_count: number; avg_duration_ms: number; max_duration_ms: number }
interface Summary {
  total_requests: number; active_requests: number; requests_per_second: number;
  status_2xx: number; status_3xx: number; status_4xx: number; status_5xx: number;
  latency_avg_ms: number; latency_p50_ms: number; latency_p90_ms: number; latency_p99_ms: number;
  top_endpoints: Endpoint[]; slowest_endpoints: Endpoint[];
}
interface SystemStats {
  server: { uptime: string; go_version: string; os: string; arch: string };
  cpu: { process_cpu_percent: number; system_cpu_percent: number; num_cpu: number; num_goroutine: number };
  memory: { alloc_human: string; heap_alloc_human: string; sys_human: string; heap_objects: number; num_gc: number };
  database: { open_connections: number; in_use: number; idle: number; wait_count: number };
  redis: { connected: boolean; ping_latency_ms: number; error?: string };
  realtime: { connected_clients: number; active_channels: number };
}
interface Trace {
  id: string; timestamp: string; method: string; path: string; route?: string; query?: string;
  status: number; duration_ms: number; response_bytes: number; client_ip: string; player_id?: string; username?: string;
  user_agent?: string; error?: string; error_kind?: string; error_chain?: string[]; root_cause?: string;
  stacktrace?: string; worker_run_id?: string; request_body?: unknown; body_truncated?: boolean; content_length?: number;
}

const badge = (status: number) => status >= 500 ? "text-red-300 bg-red-950/60" : status >= 400 ? "text-orange-300 bg-orange-950/60" : status >= 300 ? "text-sky-300 bg-sky-950/60" : "text-emerald-300 bg-emerald-950/60";
const methodColor = (method: string) => method === "GET" ? "text-sky-400" : method === "POST" ? "text-emerald-400" : method === "DELETE" ? "text-red-400" : method === "WORKER" ? "text-purple-400" : "text-amber-400";
const formatBytes = (bytes: number) => bytes >= 1024 * 1024
  ? `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  : bytes >= 1024 ? `${(bytes / 1024).toFixed(1)} KB` : `${bytes} B`;

export function MonitoringAdminTab({ addToast }: { addToast: (msg: string, type: "success" | "error" | "info") => void }) {
  const [system, setSystem] = useState<SystemStats | null>(null);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [traces, setTraces] = useState<Trace[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(false);
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [query, updateQuery] = useAdminUrlState(TRACE_DEFAULTS, TRACE_KEYS);
  const page = Math.max(1, Number(query.page) || 1);
  const filters = useMemo(() => ({ method: query.method, status: query.status, path: query.path, search: query.search, min_duration_ms: query.min_duration_ms }), [query.method, query.min_duration_ms, query.path, query.search, query.status]);
  const setPage = (value: number | ((page: number) => number)) => updateQuery({ page: String(typeof value === "function" ? value(page) : value) });
  const setFilters = (next: typeof filters) => updateQuery({ ...next, page: "1" });
  const requestSequence = useRef(0);
  const limit = Math.max(1, Number(query.page_size) || 25);

  const load = useCallback(async (quiet = false) => {
    const sequence = ++requestSequence.current;
    if (!quiet) setLoading(true);
    const params = new URLSearchParams({ page: String(page), page_size: String(limit), sort: query.sort, direction: query.direction });
    Object.entries(filters).forEach(([key, value]) => { if (value) params.set(key, value); });
    try {
      const [systemRes, summaryRes, tracesRes] = await Promise.all([
        authFetch(`${BACKEND_URL}/api/admin/monitor/system`),
        authFetch(`${BACKEND_URL}/api/admin/monitor/requests/summary`),
        authFetch(`${BACKEND_URL}/api/admin/monitor/requests/traces?${params}`),
      ]);
      if (!systemRes.ok || !summaryRes.ok || !tracesRes.ok) throw new Error("Không thể tải dữ liệu giám sát");
      const [systemData, summaryData, traceData] = await Promise.all([systemRes.json(), summaryRes.json(), tracesRes.json()]);
      if (sequence !== requestSequence.current) return;
      setSystem(systemData); setSummary(summaryData); setTraces(traceData.traces || []); setTotal(traceData.total || 0); setUpdatedAt(new Date());
    } catch (error) {
      if (!quiet) addToast(error instanceof Error ? error.message : "Lỗi tải monitoring", "error");
    } finally { if (!quiet && sequence === requestSequence.current) setLoading(false); }
  }, [addToast, filters, page, limit, query.direction, query.sort]);

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0);
    return () => window.clearTimeout(timer);
  }, [load]);
  useEffect(() => {
    if (!autoRefresh) return;
    const timer = window.setInterval(() => void load(true), 10000);
    return () => window.clearInterval(timer);
  }, [autoRefresh, load]);

  const reset = async () => {
    if (!window.confirm("Xóa toàn bộ request metrics và trace đang lưu trong bộ nhớ?")) return;
    const response = await authFetch(`${BACKEND_URL}/api/admin/monitor/requests/reset`, { method: "POST" });
    if (!response.ok) return addToast("Không thể reset metrics", "error");
    addToast("Đã reset request metrics", "success"); setPage(1); await load();
  };

  const card = (label: string, value: React.ReactNode, hint?: string) => <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-3"><div className="text-[10px] uppercase tracking-wider text-zinc-500">{label}</div><div className="mt-1 text-xl font-mono font-semibold text-zinc-100">{value}</div>{hint && <div className="mt-1 text-[10px] text-zinc-500">{hint}</div>}</div>;
  const endpointTable = (title: string, rows: Endpoint[]) => <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-4 overflow-hidden"><h3 className="mb-3 text-xs font-bold text-zinc-300">{title}</h3><div className="overflow-x-auto"><table className="w-full text-left text-[11px]"><thead className="text-zinc-600"><tr><th className="pb-2">Endpoint</th><th>Lượt</th><th>Lỗi</th><th>TB</th><th>Max</th></tr></thead><tbody>{rows?.map((row) => <tr key={`${row.method}-${row.path}`} className="border-t border-zinc-800/60"><td className="py-2 pr-3"><span className={`font-bold ${methodColor(row.method)}`}>{row.method}</span> <span className="font-mono text-zinc-400">{row.path}</span></td><td>{row.count}</td><td className={row.error_count ? "text-red-400" : "text-zinc-500"}>{row.error_count}</td><td>{row.avg_duration_ms}ms</td><td>{row.max_duration_ms}ms</td></tr>)}</tbody></table></div></div>;

  return <div className="space-y-4">
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-zinc-800 bg-zinc-900/40 p-4">
    <div><h2 className="flex items-center gap-2 font-serif font-bold text-zinc-100"><Activity className="h-4 w-4 text-emerald-400" />Giám sát hệ thống</h2><p className="mt-1 text-[10px] text-zinc-500">Dữ liệu runtime và 500 HTTP request gần nhất · cập nhật {updatedAt ? updatedAt.toLocaleTimeString("vi-VN") : "—"}</p></div>
      <div className="flex items-center gap-2"><label className="flex items-center gap-2 text-xs text-zinc-400"><input type="checkbox" checked={autoRefresh} onChange={(e) => setAutoRefresh(e.target.checked)} />Tự tải 10s</label><button onClick={() => void load()} disabled={loading} className="rounded-lg bg-zinc-800 p-2 text-zinc-300 hover:bg-zinc-700"><RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} /></button><button onClick={() => void reset()} className="flex items-center gap-1 rounded-lg border border-red-900/60 px-2.5 py-2 text-xs text-red-400 hover:bg-red-950/40"><RotateCcw className="h-3.5 w-3.5" />Reset</button></div>
    </div>

    {summary && <><div className="grid grid-cols-2 gap-3 md:grid-cols-4 lg:grid-cols-6">{card("Tổng request", summary.total_requests)}{card("Đang xử lý", summary.active_requests)}{card("Request / giây", summary.requests_per_second)}{card("P50", `${summary.latency_p50_ms}ms`)}{card("P90", `${summary.latency_p90_ms}ms`)}{card("P99", `${summary.latency_p99_ms}ms`, `TB ${summary.latency_avg_ms}ms`)}</div><div className="flex flex-wrap gap-2 text-xs"><span className="rounded bg-emerald-950/60 px-2 py-1 text-emerald-300">2xx {summary.status_2xx}</span><span className="rounded bg-sky-950/60 px-2 py-1 text-sky-300">3xx {summary.status_3xx}</span><span className="rounded bg-orange-950/60 px-2 py-1 text-orange-300">4xx {summary.status_4xx}</span><span className="rounded bg-red-950/60 px-2 py-1 text-red-300">5xx {summary.status_5xx}</span></div></>}

    {system && <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">{card("CPU process", `${system.cpu.process_cpu_percent}%`, `${system.cpu.num_cpu} cores · system ${system.cpu.system_cpu_percent}%`)}{card("Bộ nhớ heap", system.memory.heap_alloc_human, `${system.memory.heap_objects.toLocaleString()} objects · ${system.memory.num_gc} GC`)}{card("Database", system.database.open_connections, `${system.database.in_use} dùng · ${system.database.idle} rảnh`)}{card("Redis", system.redis.connected ? "Online" : "Offline", system.redis.connected ? `${system.redis.ping_latency_ms.toFixed(2)}ms` : system.redis.error)}{card("Realtime", system.realtime.connected_clients, `${system.realtime.active_channels} channels`)}{card("Uptime", system.server.uptime, `${system.server.go_version} · ${system.cpu.num_goroutine} goroutines`)}</div>}

    {summary && <div className="grid gap-4 lg:grid-cols-2">{endpointTable("Endpoint nhiều request nhất", summary.top_endpoints)}{endpointTable("Endpoint chậm nhất", summary.slowest_endpoints)}</div>}

    <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-4"><div className="mb-3 flex items-center gap-2"><Server className="h-4 w-4 text-amber-400" /><h3 className="text-sm font-bold text-zinc-200">HTTP & realtime traces</h3></div>
      <div className="mb-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-5"><select value={filters.method} onChange={(e) => { setFilters({ ...filters, method: e.target.value }); }} className="rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-1.5 text-xs"><option value="">Mọi method</option>{["GET","POST","PUT","DELETE","PATCH","WORKER"].map(x => <option key={x}>{x}</option>)}</select><select value={filters.status} onChange={(e) => { setFilters({ ...filters, status: e.target.value }); }} className="rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-1.5 text-xs"><option value="">Mọi status</option><option value="errors">Chỉ lỗi</option><option value="200">200</option><option value="400">400</option><option value="401">401</option><option value="403">403</option><option value="404">404</option><option value="500">500</option></select><input value={filters.path} onChange={(e) => { setFilters({ ...filters, path: e.target.value }); }} placeholder="Path..." className="rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-1.5 text-xs" /><input value={filters.search} onChange={(e) => { setFilters({ ...filters, search: e.target.value }); }} placeholder="Trace ID / User / Player / IP / Root cause..." className="rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-1.5 text-xs" /><input type="number" value={filters.min_duration_ms} onChange={(e) => { setFilters({ ...filters, min_duration_ms: e.target.value }); }} placeholder="Latency tối thiểu (ms)" className="rounded-lg border border-zinc-700 bg-zinc-800 px-2 py-1.5 text-xs" /></div>
      <div className="space-y-2">
        {traces.map((trace) => <div key={`${trace.id}-${trace.timestamp}`} className="rounded-lg border border-zinc-800 bg-zinc-950/50">
          <button onClick={() => setExpanded(expanded === trace.id ? null : trace.id)} className="grid w-full gap-2 p-3 text-left text-xs md:grid-cols-[minmax(0,1fr)_auto_auto] md:items-center">
            <span className="min-w-0"><span className={`mr-2 font-bold ${methodColor(trace.method)}`}>{trace.method}</span><span className="break-all font-mono text-zinc-300">{trace.path}</span>{trace.username && <span className="ml-2 text-zinc-500">· {trace.username}</span>}<span className="mt-1 block truncate text-zinc-500">{trace.root_cause || trace.error || ""}</span></span>
            <span className={`w-fit rounded px-2 py-1 font-mono ${badge(trace.status)}`}>{trace.status}</span>
            <span className="text-zinc-500">{trace.duration_ms.toFixed(1)}ms · {new Date(trace.timestamp).toLocaleString("vi-VN")}</span>
          </button>
          {expanded === trace.id && <div className="space-y-2 border-t border-zinc-800 p-3 text-xs">
            <div className="flex flex-wrap gap-x-4 gap-y-1 text-zinc-400"><span>Trace: <code className="text-zinc-200">{trace.id}</code></span>{trace.error_kind && <span>Loại: <code className="text-zinc-200">{trace.error_kind}</code></span>}{trace.worker_run_id && <span>Worker run: <code className="text-zinc-200">{trace.worker_run_id}</code></span>}{trace.player_id && <span>Player: <code className="text-zinc-200">{trace.player_id}</code></span>}</div>
            {trace.root_cause && <p className="break-words text-red-300"><strong>Root cause:</strong> {trace.root_cause}</p>}
            {trace.error_chain?.length ? <div><div className="mb-1 text-zinc-500">Error chain</div><ol className="list-inside list-decimal space-y-1 break-words font-mono text-zinc-300">{trace.error_chain.map((cause, index) => <li key={`${index}-${cause}`}>{cause}</li>)}</ol></div> : null}
            {trace.error && !trace.error_chain?.length && <p className="break-words text-zinc-300">{trace.error}</p>}
            {trace.stacktrace && <details><summary className="cursor-pointer text-zinc-500">Stack trace</summary><pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded bg-black/30 p-2 font-mono text-[10px] text-zinc-400">{trace.stacktrace}</pre></details>}
            {trace.request_body !== undefined && <details><summary className="cursor-pointer text-zinc-500">Request body{trace.body_truncated ? " (đã cắt ngắn)" : ""}</summary><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words rounded bg-black/30 p-2 font-mono text-[10px] text-zinc-400">{JSON.stringify(trace.request_body, null, 2)}</pre></details>}
          </div>}
        </div>)}
        {!traces.length && <p className="py-6 text-center text-xs text-zinc-600">Không có trace phù hợp.</p>}
      </div>
      <div className="mt-4 flex items-center justify-between border-t border-zinc-800 pt-3"><span className="text-[11px] text-zinc-500">{total} traces</span><div className="flex gap-2"><button disabled={page === 1} onClick={() => setPage(p => p - 1)} className="rounded bg-zinc-800 p-1.5 disabled:opacity-30"><ChevronLeft className="h-4 w-4" /></button><span className="px-2 py-1 text-xs text-zinc-500">{page}</span><button disabled={page * limit >= total} onClick={() => setPage(p => p + 1)} className="rounded bg-zinc-800 p-1.5 disabled:opacity-30"><ChevronRight className="h-4 w-4" /></button></div></div>
    </div>
  </div>;
}
