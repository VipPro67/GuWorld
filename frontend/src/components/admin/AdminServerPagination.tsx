"use client";

import { ChevronLeft, ChevronRight } from "lucide-react";
import { GameButton } from "@/components/ui";

export interface AdminServerPaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onPageChange: (page: number) => void;
  onPageSizeChange?: (pageSize: number) => void;
  pageSizeOptions?: number[];
  loading?: boolean;
}

export function AdminServerPagination({ page, pageSize, total, onPageChange, onPageSizeChange, pageSizeOptions = [10, 25, 50], loading = false }: AdminServerPaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / Math.max(1, pageSize)));
  const safePage = Math.min(Math.max(1, page), totalPages);
  const first = total === 0 ? 0 : (safePage - 1) * pageSize + 1;
  const last = Math.min(total, safePage * pageSize);
  return (
    <nav aria-label="Phân trang" className="flex flex-col gap-2 text-xs text-zinc-500 sm:flex-row sm:items-center sm:justify-between">
      <span>{first}–{last} / {total} dòng</span>
      <div className="flex items-center gap-2">
        {onPageSizeChange && <label className="flex items-center gap-1">Hiển thị <select aria-label="Số dòng mỗi trang" value={pageSize} disabled={loading} onChange={(event) => onPageSizeChange(Number(event.target.value))} className="rounded-lg border border-zinc-700 bg-zinc-900 px-2 py-1.5 text-zinc-200">{pageSizeOptions.map((size) => <option key={size} value={size}>{size}</option>)}</select></label>}
        <span>Trang {safePage}/{totalPages}</span>
        <GameButton aria-label="Trang trước" type="button" variant="secondary" size="xs" disabled={loading || safePage <= 1} onClick={() => onPageChange(safePage - 1)}><ChevronLeft className="h-3.5 w-3.5" /></GameButton>
        <GameButton aria-label="Trang sau" type="button" variant="secondary" size="xs" disabled={loading || safePage >= totalPages} onClick={() => onPageChange(safePage + 1)}><ChevronRight className="h-3.5 w-3.5" /></GameButton>
      </div>
    </nav>
  );
}
