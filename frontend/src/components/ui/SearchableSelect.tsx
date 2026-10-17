"use client";

import React, { useState, useMemo, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { Search, X, Check, ChevronDown } from "lucide-react";

export interface SearchableSelectOption {
  value: string;
  label: string;
  subLabel?: string;
  badge?: string;
  badgeColor?: string;
  disabled?: boolean;
}

export interface SearchableSelectProps {
  options: SearchableSelectOption[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  disabled?: boolean;
  className?: string;
  title?: string;
  label?: string;
  required?: boolean;
  emptyMessage?: string;
  clearable?: boolean;
  id?: string;
}

// Utility to normalize Vietnamese string for diacritic-insensitive search
function normalizeSearchText(str: string): string {
  return str
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/đ/g, "d");
}

export const SearchableSelect: React.FC<SearchableSelectProps> = ({
  options,
  value,
  onChange,
  placeholder = "-- Chọn --",
  searchPlaceholder = "Tìm kiếm theo tên hoặc mã...",
  disabled = false,
  className = "",
  title,
  label,
  required = false,
  emptyMessage = "Không tìm thấy kết quả phù hợp",
  clearable = false,
  id,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  // Selected option
  const selectedOption = useMemo(() => {
    return options.find((opt) => opt.value === value);
  }, [options, value]);

  // Filtered options based on search term
  const filteredOptions = useMemo(() => {
    if (!search.trim()) return options;
    const query = normalizeSearchText(search);
    return options.filter((opt) => {
      const labelNorm = normalizeSearchText(opt.label);
      const valNorm = normalizeSearchText(opt.value);
      const subNorm = opt.subLabel ? normalizeSearchText(opt.subLabel) : "";
      const badgeNorm = opt.badge ? normalizeSearchText(opt.badge) : "";
      return (
        labelNorm.includes(query) ||
        valNorm.includes(query) ||
        subNorm.includes(query) ||
        badgeNorm.includes(query)
      );
    });
  }, [options, search]);

  // Auto-focus input when opened
  useEffect(() => {
    if (isOpen) {
      setSearch("");
      const timer = setTimeout(() => {
        inputRef.current?.focus();
      }, 50);
      return () => clearTimeout(timer);
    }
  }, [isOpen]);

  // Handle ESC key
  useEffect(() => {
    if (!isOpen) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        setIsOpen(false);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [isOpen]);

  const handleSelect = (val: string) => {
    onChange(val);
    setIsOpen(false);
  };

  const modalTitle = title || label || "Chọn mục";

  return (
    <div className={`relative ${className}`} id={id}>
      {label && (
        <label className="block text-xs text-zinc-400 mb-1 font-medium font-serif">
          {label}
          {required && <span className="text-red-500 ml-1">*</span>}
        </label>
      )}

      {/* Trigger Button */}
      <button
        type="button"
        disabled={disabled}
        onClick={() => setIsOpen(true)}
        className={`w-full flex items-center justify-between gap-2 bg-zinc-950 border border-zinc-800 hover:border-zinc-700 disabled:opacity-40 disabled:cursor-not-allowed rounded-xl px-3 py-2.5 text-xs text-left transition-colors font-mono cursor-pointer ${
          selectedOption ? "text-zinc-200" : "text-zinc-500"
        }`}
      >
        <span className="truncate flex-1">
          {selectedOption ? selectedOption.label : placeholder}
        </span>
        <div className="flex items-center gap-1 shrink-0 text-zinc-500">
          {clearable && value && (
            <span
              role="button"
              tabIndex={0}
              onClick={(e) => {
                e.stopPropagation();
                onChange("");
              }}
              className="p-0.5 hover:text-zinc-300 rounded"
              title="Xóa lựa chọn"
            >
              <X className="w-3.5 h-3.5" />
            </span>
          )}
          <ChevronDown className={`w-4 h-4 transition-transform ${isOpen ? "rotate-180" : ""}`} />
        </div>
      </button>

      {/* Search & Pick Modal via Portal */}
      {isOpen &&
        typeof document !== "undefined" &&
        createPortal(
          <div
            className="fixed inset-0 z-[9999] flex items-start sm:items-center justify-center p-3 sm:p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-150 overflow-y-auto"
            onClick={() => setIsOpen(false)}
          >
            <div
              className="relative w-full max-w-lg bg-zinc-950 border border-zinc-800 rounded-2xl sm:rounded-3xl shadow-2xl flex flex-col max-h-[85dvh] overflow-hidden my-auto animate-in zoom-in-95 duration-150"
              onClick={(e) => e.stopPropagation()}
            >
              {/* Modal Header */}
              <div className="flex items-center justify-between px-4 sm:px-5 py-3.5 border-b border-zinc-800 bg-zinc-900/50 shrink-0">
                <div className="font-serif font-bold text-sm text-zinc-200">{modalTitle}</div>
                <button
                  type="button"
                  onClick={() => setIsOpen(false)}
                  className="p-1 rounded-lg text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors cursor-pointer"
                  title="Đóng"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>

              {/* Search Bar */}
              <div className="p-3 border-b border-zinc-800/80 bg-zinc-950 shrink-0">
                <div className="relative">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
                  <input
                    ref={inputRef}
                    type="text"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder={searchPlaceholder}
                    className="w-full pl-9 pr-8 py-2 bg-zinc-900 border border-zinc-800 rounded-xl text-xs text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-amber-500/60 font-mono transition-colors"
                  />
                  {search && (
                    <button
                      type="button"
                      onClick={() => setSearch("")}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-zinc-500 hover:text-zinc-300"
                    >
                      <X className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
                <div className="flex justify-between items-center mt-2 px-1 text-[11px] text-zinc-500 font-mono">
                  <span>{filteredOptions.length} kết quả</span>
                  {selectedOption && (
                    <span className="truncate max-w-[200px] text-amber-400/80">
                      Hiện tại: {selectedOption.label}
                    </span>
                  )}
                </div>
              </div>

              {/* Options List */}
              <div className="flex-1 overflow-y-auto p-2 space-y-1 custom-scrollbar min-h-0">
                {/* Default empty / clear option if placeholder allowed */}
                {clearable && (
                  <button
                    type="button"
                    onClick={() => handleSelect("")}
                    className={`w-full flex items-center justify-between px-3 py-2.5 rounded-xl text-xs text-left transition-colors font-mono cursor-pointer ${
                      !value
                        ? "bg-amber-500/10 border border-amber-500/30 text-amber-300 font-bold"
                        : "hover:bg-zinc-900 text-zinc-400"
                    }`}
                  >
                    <span>{placeholder}</span>
                    {!value && <Check className="w-4 h-4 text-amber-400 shrink-0" />}
                  </button>
                )}

                {filteredOptions.length === 0 ? (
                  <div className="py-10 text-center text-xs text-zinc-500 font-mono">
                    {emptyMessage}
                  </div>
                ) : (
                  filteredOptions.map((opt) => {
                    const isSelected = opt.value === value;
                    return (
                      <button
                        key={opt.value}
                        type="button"
                        disabled={opt.disabled}
                        onClick={() => handleSelect(opt.value)}
                        className={`w-full flex items-center justify-between gap-2 px-3 py-2.5 rounded-xl text-xs text-left transition-all font-mono cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed ${
                          isSelected
                            ? "bg-amber-500/15 border border-amber-500/40 text-amber-200 font-bold shadow-sm"
                            : "hover:bg-zinc-900 text-zinc-200 border border-transparent"
                        }`}
                      >
                        <div className="flex flex-col min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="truncate">{opt.label}</span>
                            {opt.badge && (
                              <span
                                className={`text-[9px] px-1.5 py-0.5 rounded border shrink-0 font-semibold ${
                                  opt.badgeColor || "bg-zinc-800 text-zinc-300 border-zinc-700"
                                }`}
                              >
                                {opt.badge}
                              </span>
                            )}
                          </div>
                          {opt.subLabel && (
                            <span className="text-[10px] text-zinc-500 truncate mt-0.5">
                              {opt.subLabel}
                            </span>
                          )}
                        </div>
                        {isSelected && (
                          <Check className="w-4 h-4 text-amber-400 shrink-0 ml-2" />
                        )}
                      </button>
                    );
                  })
                )}
              </div>
            </div>
          </div>,
          document.body
        )}
    </div>
  );
};
