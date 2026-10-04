"use client";

import React, { useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  ChevronsLeft,
  ChevronsRight,
  ArrowRight,
} from "lucide-react";
import { getPaginationRange, type PageSize } from "@/lib/pagination";

interface Props {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  pageSize: PageSize;
  startIndex: number;
  endIndex: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: PageSize) => void;
  pageSizeOptions?: PageSize[];
  itemLabel?: string;
  compact?: boolean;
  scrollTargetId?: string;
}

export default function WikiPagination({
  currentPage,
  totalPages,
  totalItems,
  pageSize,
  startIndex,
  endIndex,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [24, 36, 60, 120, "all"],
  itemLabel = "mục",
  compact = false,
  scrollTargetId,
}: Props) {
  const [jumpInput, setJumpInput] = useState("");

  if (totalItems === 0) return null;

  const scrollToTarget = () => {
    if (typeof window === "undefined") return;
    if (scrollTargetId) {
      const el = document.getElementById(scrollTargetId);
      if (el) {
        el.scrollIntoView({ behavior: "smooth", block: "start" });
        return;
      }
    }
    // Fallback scroll slightly up if near bottom
    window.scrollTo({ top: 180, behavior: "smooth" });
  };

  const handlePageSelect = (page: number) => {
    if (page === currentPage || page < 1 || page > totalPages) return;
    onPageChange(page);
    scrollToTarget();
  };

  const handleJumpSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const parsed = parseInt(jumpInput, 10);
    if (!isNaN(parsed)) {
      const clamped = Math.min(Math.max(1, parsed), totalPages);
      handlePageSelect(clamped);
      setJumpInput("");
    }
  };

  const pageRange = getPaginationRange(currentPage, totalPages, 1);

  // Compact Mode (for top header bar)
  if (compact) {
    return (
      <div className="flex items-center gap-2 font-mono text-xs">
        {/* Page size selector */}
        <div className="flex items-center gap-1 text-[11px] text-[var(--text-muted)]">
          <select
            value={pageSize}
            onChange={(e) => {
              const val = e.target.value === "all" ? "all" : Number(e.target.value);
              onPageSizeChange(val);
            }}
            className="px-2 py-1 rounded-lg bg-zinc-950/70 border border-zinc-800 text-[var(--text-secondary)] text-[11px] font-mono hover:border-zinc-700 focus:border-indigo-500 focus:outline-none transition-colors cursor-pointer"
            title="Số mục mỗi trang"
          >
            {pageSizeOptions.map((opt) => (
              <option key={opt} value={opt} className="bg-zinc-900 text-zinc-200">
                {opt === "all" ? "Tất cả" : `${opt} / trang`}
              </option>
            ))}
          </select>
        </div>

        {pageSize !== "all" && totalPages > 1 && (
          <div className="flex items-center gap-1 ml-1">
            <span className="text-[11px] text-[var(--text-muted)] mr-1">
              {currentPage}/{totalPages}
            </span>
            <button
              onClick={() => handlePageSelect(currentPage - 1)}
              disabled={currentPage <= 1}
              className="p-1 rounded-md bg-zinc-950/70 border border-zinc-800 hover:border-indigo-500/40 text-zinc-300 hover:text-white disabled:opacity-30 disabled:pointer-events-none transition-colors cursor-pointer"
              title="Trang trước"
              aria-label="Trang trước"
            >
              <ChevronLeft className="w-3.5 h-3.5" />
            </button>
            <button
              onClick={() => handlePageSelect(currentPage + 1)}
              disabled={currentPage >= totalPages}
              className="p-1 rounded-md bg-zinc-950/70 border border-zinc-800 hover:border-indigo-500/40 text-zinc-300 hover:text-white disabled:opacity-30 disabled:pointer-events-none transition-colors cursor-pointer"
              title="Trang tiếp"
              aria-label="Trang tiếp"
            >
              <ChevronRight className="w-3.5 h-3.5" />
            </button>
          </div>
        )}
      </div>
    );
  }

  // Full Mode (for bottom footer)
  return (
    <div className="mt-8 pt-6 border-t border-zinc-900/80 flex flex-col gap-4 font-mono">
      {/* Top summary row: Range info + Page size pills */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs text-[var(--text-muted)]">
        <div>
          <span>Hiển thị </span>
          <span className="font-bold text-[var(--text-primary)]">
            {startIndex + 1}
          </span>
          <span> – </span>
          <span className="font-bold text-[var(--text-primary)]">
            {endIndex}
          </span>
          <span> trên tổng </span>
          <span className="font-bold text-indigo-400">
            {totalItems.toLocaleString()}
          </span>
          <span> {itemLabel}</span>
          {pageSize !== "all" && totalPages > 1 && (
            <span className="ml-2 text-zinc-500 font-normal">
              (Trang {currentPage} / {totalPages})
            </span>
          )}
        </div>

        {/* Page size quick selector */}
        <div className="flex items-center gap-1.5 self-start sm:self-auto">
          <span className="text-[11px] text-zinc-500">Mỗi trang:</span>
          <div className="flex items-center gap-1 bg-zinc-950/60 p-0.5 rounded-lg border border-zinc-850">
            {pageSizeOptions.map((opt) => {
              const isActive = pageSize === opt;
              return (
                <button
                  key={opt}
                  onClick={() => onPageSizeChange(opt)}
                  className={`px-2 py-0.5 rounded-md text-[10px] font-semibold transition-all cursor-pointer ${
                    isActive
                      ? "bg-indigo-600/90 text-white shadow-xs"
                      : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800/60"
                  }`}
                >
                  {opt === "all" ? "Tất cả" : opt}
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {/* Main pagination controls (only if totalPages > 1 and not "all") */}
      {pageSize !== "all" && totalPages > 1 && (
        <div className="flex flex-wrap items-center justify-between gap-3 pt-2">
          {/* Page buttons */}
          <div className="flex flex-wrap items-center gap-1">
            {/* First Page */}
            <button
              onClick={() => handlePageSelect(1)}
              disabled={currentPage <= 1}
              className="p-1.5 rounded-lg bg-zinc-950/70 border border-zinc-800 hover:border-indigo-500/40 text-zinc-300 hover:text-white disabled:opacity-25 disabled:pointer-events-none transition-colors cursor-pointer"
              title="Về trang đầu"
              aria-label="Về trang đầu"
            >
              <ChevronsLeft className="w-4 h-4" />
            </button>

            {/* Prev Page */}
            <button
              onClick={() => handlePageSelect(currentPage - 1)}
              disabled={currentPage <= 1}
              className="p-1.5 rounded-lg bg-zinc-950/70 border border-zinc-800 hover:border-indigo-500/40 text-zinc-300 hover:text-white disabled:opacity-25 disabled:pointer-events-none transition-colors cursor-pointer mr-1"
              title="Trang trước"
              aria-label="Trang trước"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>

            {/* Numeric Pages */}
            {pageRange.map((page, idx) => {
              if (page === "...") {
                return (
                  <span
                    key={`dot-${idx}`}
                    className="px-2 py-1 text-xs text-zinc-600 select-none"
                  >
                    …
                  </span>
                );
              }

              const isCurrent = page === currentPage;
              return (
                <button
                  key={`page-${page}`}
                  onClick={() => handlePageSelect(page)}
                  className={`min-w-[32px] h-8 px-2 rounded-lg text-xs font-mono font-semibold transition-all cursor-pointer ${
                    isCurrent
                      ? "bg-indigo-600 text-white border border-indigo-400/50 shadow-sm shadow-indigo-600/30 scale-105"
                      : "bg-zinc-950/70 border border-zinc-800/80 text-zinc-300 hover:text-white hover:border-indigo-500/30 hover:bg-zinc-900"
                  }`}
                >
                  {page}
                </button>
              );
            })}

            {/* Next Page */}
            <button
              onClick={() => handlePageSelect(currentPage + 1)}
              disabled={currentPage >= totalPages}
              className="p-1.5 rounded-lg bg-zinc-950/70 border border-zinc-800 hover:border-indigo-500/40 text-zinc-300 hover:text-white disabled:opacity-25 disabled:pointer-events-none transition-colors cursor-pointer ml-1"
              title="Trang tiếp"
              aria-label="Trang tiếp"
            >
              <ChevronRight className="w-4 h-4" />
            </button>

            {/* Last Page */}
            <button
              onClick={() => handlePageSelect(totalPages)}
              disabled={currentPage >= totalPages}
              className="p-1.5 rounded-lg bg-zinc-950/70 border border-zinc-800 hover:border-indigo-500/40 text-zinc-300 hover:text-white disabled:opacity-25 disabled:pointer-events-none transition-colors cursor-pointer"
              title="Đến trang cuối"
              aria-label="Đến trang cuối"
            >
              <ChevronsRight className="w-4 h-4" />
            </button>
          </div>

          {/* Jump to page form */}
          <form
            onSubmit={handleJumpSubmit}
            className="flex items-center gap-1.5 text-xs text-[var(--text-muted)]"
          >
            <span>Đến trang:</span>
            <input
              type="number"
              min={1}
              max={totalPages}
              value={jumpInput}
              onChange={(e) => setJumpInput(e.target.value)}
              placeholder={String(currentPage)}
              className="w-16 px-2 py-1 rounded-lg bg-zinc-950/80 border border-zinc-800 text-center text-xs text-[var(--text-primary)] font-mono focus:border-indigo-500 focus:outline-none transition-colors"
            />
            <button
              type="submit"
              className="px-2.5 py-1 rounded-lg bg-zinc-900 hover:bg-indigo-600/80 border border-zinc-800 hover:border-indigo-500 text-zinc-300 hover:text-white text-xs font-mono transition-all flex items-center gap-1 cursor-pointer"
            >
              <span>Đi</span>
              <ArrowRight className="w-3 h-3" />
            </button>
          </form>
        </div>
      )}
    </div>
  );
}
