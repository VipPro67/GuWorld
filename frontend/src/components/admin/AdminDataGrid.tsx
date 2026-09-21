"use client";

import React, { ReactNode, useEffect, useMemo, useState } from "react";
import { ChevronLeft, ChevronRight, Copy, Edit3, Search, Trash2 } from "lucide-react";

export type AdminGridColumn<T> = {
  key: string;
  label: string;
  value: (row: T) => unknown;
  render?: (row: T) => ReactNode;
  className?: string;
  sortable?: boolean;
};

type Props<T> = {
  rows: T[];
  columns: AdminGridColumn<T>[];
  rowKey: (row: T) => string;
  loading?: boolean;
  searchPlaceholder?: string;
  emptyLabel?: string;
  pageSizeOptions?: number[];
  searchValue?: (row: T) => unknown;
  onEdit?: (row: T) => void;
  onCopy?: (row: T) => void;
  onDelete?: (row: T) => void;
};

function searchable(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "object") return JSON.stringify(value).toLocaleLowerCase("vi");
  return String(value).toLocaleLowerCase("vi");
}

function compareValues(left: unknown, right: unknown): number {
  if (typeof left === "number" && typeof right === "number") return left - right;
  if (typeof left === "boolean" && typeof right === "boolean") return Number(left) - Number(right);
  return String(left ?? "").localeCompare(String(right ?? ""), "vi", { numeric: true, sensitivity: "base" });
}

/** Shared client-side grid for the small, static admin dictionaries. */
export function AdminDataGrid<T>({
  rows,
  columns,
  rowKey,
  loading = false,
  searchPlaceholder = "Tìm trong tất cả các cột...",
  emptyLabel = "Chưa có dữ liệu",
  pageSizeOptions = [10, 25, 50],
  searchValue = (row) => row,
  onEdit,
  onCopy,
  onDelete,
}: Props<T>) {
  const [query, setQuery] = useState("");
  const [sortKey, setSortKey] = useState(columns.find((column) => column.sortable !== false)?.key ?? "");
  const [sortDirection, setSortDirection] = useState<"asc" | "desc">("asc");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(pageSizeOptions[0] ?? 10);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  const filteredRows = useMemo(() => {
    const normalizedQuery = query.trim().toLocaleLowerCase("vi");
    const filtered = normalizedQuery
      ? rows.filter((row) => searchable(searchValue(row)).includes(normalizedQuery))
      : [...rows];
    const sortColumn = columns.find((column) => column.key === sortKey);
    if (sortColumn) {
      filtered.sort((left, right) => compareValues(sortColumn.value(left), sortColumn.value(right)) * (sortDirection === "asc" ? 1 : -1));
    }
    return filtered;
  }, [columns, query, rows, searchValue, sortDirection, sortKey]);

  const totalPages = Math.max(1, Math.ceil(filteredRows.length / pageSize));
  const visibleRows = filteredRows.slice((Math.min(page, totalPages) - 1) * pageSize, Math.min(page, totalPages) * pageSize);

  useEffect(() => setPage(1), [query, pageSize]);
  useEffect(() => setPage((current) => Math.min(current, totalPages)), [totalPages]);
  useEffect(() => setSelected(new Set()), [rows]);

  const visibleKeys = visibleRows.map(rowKey);
  const allVisibleSelected = visibleKeys.length > 0 && visibleKeys.every((key) => selected.has(key));
  const toggleVisible = () => setSelected((current) => {
    const next = new Set(current);
    if (allVisibleSelected) visibleKeys.forEach((key) => next.delete(key));
    else visibleKeys.forEach((key) => next.add(key));
    return next;
  });

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative flex-1 sm:max-w-md">
          <Search className="absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-500" />
          <input aria-label="Tìm kiếm dữ liệu" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={searchPlaceholder} className="w-full rounded-lg border border-zinc-700 bg-zinc-800 py-2 pl-9 pr-3 text-xs text-zinc-200 placeholder:text-zinc-500" />
        </div>
        <div className="text-[11px] text-zinc-500">Đã chọn <span className="font-bold text-amber-400">{selected.size}</span> · {filteredRows.length}/{rows.length} dòng</div>
      </div>
      <div className="max-h-[560px] overflow-auto rounded-xl border border-zinc-800/80">
        <table className="w-full min-w-max border-collapse text-left text-xs">
          <thead className="sticky top-0 z-10 bg-zinc-900 text-[10px] uppercase tracking-wider text-zinc-400">
            <tr>
              <th className="px-3 py-2.5"><input aria-label="Chọn tất cả dòng trên trang" type="checkbox" checked={allVisibleSelected} onChange={toggleVisible} /></th>
              {columns.map((column) => <th key={column.key} onClick={() => {
                if (column.sortable === false) return;
                if (sortKey === column.key) setSortDirection((direction) => direction === "asc" ? "desc" : "asc");
                else { setSortKey(column.key); setSortDirection("asc"); }
              }} className={`px-3 py-2.5 font-semibold ${column.sortable === false ? "" : "cursor-pointer select-none hover:text-zinc-200"} ${sortKey === column.key ? "text-amber-400" : ""}`}>
                {column.label}{sortKey === column.key ? (sortDirection === "asc" ? " ▲" : " ▼") : ""}
              </th>)}
              {(onEdit || onCopy || onDelete) && <th className="px-3 py-2.5 text-right">Thao tác</th>}
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-800/50 text-zinc-300">
            {visibleRows.map((row) => {
              const key = rowKey(row);
              return <tr key={key} className={`hover:bg-zinc-900/60 ${selected.has(key) ? "bg-amber-950/20" : ""}`}>
                <td className="px-3 py-2.5"><input aria-label={`Chọn ${key}`} type="checkbox" checked={selected.has(key)} onChange={() => setSelected((current) => { const next = new Set(current); if (next.has(key)) next.delete(key); else next.add(key); return next; })} /></td>
                {columns.map((column) => <td key={column.key} className={`px-3 py-2.5 ${column.className ?? ""}`}>{column.render ? column.render(row) : String(column.value(row) ?? "—")}</td>)}
                {(onEdit || onCopy || onDelete) && <td className="px-3 py-2.5"><div className="flex justify-end gap-1">
                  {onEdit && <button aria-label={`Sửa ${key}`} title="Sửa" onClick={() => onEdit(row)} className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-amber-400"><Edit3 className="h-3.5 w-3.5" /></button>}
                  {onCopy && <button aria-label={`Sao chép ${key}`} title="Sao chép để tạo mới" onClick={() => onCopy(row)} className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-sky-400"><Copy className="h-3.5 w-3.5" /></button>}
                  {onDelete && <button aria-label={`Xóa ${key}`} title="Xóa" onClick={() => onDelete(row)} className="rounded p-1 text-zinc-400 hover:bg-zinc-800 hover:text-rose-400"><Trash2 className="h-3.5 w-3.5" /></button>}
                </div></td>}
              </tr>;
            })}
            {!visibleRows.length && <tr><td colSpan={columns.length + 1 + Number(Boolean(onEdit || onCopy || onDelete))} className="py-8 text-center italic text-zinc-500">{loading ? "Đang tải dữ liệu..." : emptyLabel}</td></tr>}
          </tbody>
        </table>
      </div>
      <div className="flex flex-col gap-2 text-[11px] text-zinc-500 sm:flex-row sm:items-center sm:justify-between">
        <label>Hiển thị <select value={pageSize} onChange={(event) => setPageSize(Number(event.target.value))} className="mx-1 rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-zinc-200">{pageSizeOptions.map((size) => <option key={size}>{size}</option>)}</select> dòng/trang</label>
        <div className="flex items-center gap-2"><span>Trang {Math.min(page, totalPages)}/{totalPages}</span><button aria-label="Trang trước" disabled={page <= 1} onClick={() => setPage((current) => current - 1)} className="rounded border border-zinc-700 bg-zinc-800 p-1 text-zinc-300 disabled:opacity-40"><ChevronLeft className="h-3.5 w-3.5" /></button><button aria-label="Trang sau" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)} className="rounded border border-zinc-700 bg-zinc-800 p-1 text-zinc-300 disabled:opacity-40"><ChevronRight className="h-3.5 w-3.5" /></button></div>
      </div>
    </div>
  );
}
