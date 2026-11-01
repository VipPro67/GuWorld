"use client";

import { useState } from "react";
import {
  X,
  Plus,
  RotateCcw,
  Check,
  ChevronLeft,
  ChevronRight,
  Sparkles,
  Settings2,
} from "lucide-react";
import {
  ALL_QUICK_ACTIONS,
  QUICK_ACTIONS_MAP,
  MAX_QUICK_ACTIONS,
  MIN_QUICK_ACTIONS,
  useQuickActionIds,
} from "@/lib/mmo-quick-actions";

interface QuickActionCustomizerProps {
  inline?: boolean;
  onClose?: () => void;
}

const CATEGORIES = ["Tất cả", "Thế giới", "Nhân vật", "Giao thương", "Hoạt động", "Khác"] as const;

export function QuickActionCustomizer({ inline = false, onClose }: QuickActionCustomizerProps) {
  const { ids, setIds, reset } = useQuickActionIds();
  const [selectedCategory, setSelectedCategory] = useState<typeof CATEGORIES[number]>("Tất cả");

  const handleToggle = (actionId: string) => {
    if (ids.includes(actionId)) {
      if (ids.length <= MIN_QUICK_ACTIONS) return;
      setIds(ids.filter((id) => id !== actionId));
    } else {
      if (ids.length >= MAX_QUICK_ACTIONS) return;
      setIds([...ids, actionId]);
    }
  };

  const handleMove = (index: number, direction: -1 | 1) => {
    const nextIndex = index + direction;
    if (nextIndex < 0 || nextIndex >= ids.length) return;
    const next = [...ids];
    const temp = next[index];
    next[index] = next[nextIndex];
    next[nextIndex] = temp;
    setIds(next);
  };

  const filteredActions = ALL_QUICK_ACTIONS.filter(
    (item) => selectedCategory === "Tất cả" || item.category === selectedCategory,
  );

  const content = (
    <div className="space-y-4 text-xs font-serif text-[#f4e8cb]">
      {/* Active Quick Bar Preview */}
      <div className="rounded-xl border border-[#8f6d40]/40 bg-[#1f1610]/80 p-3 shadow-inner">
        <div className="mb-2 flex items-center justify-between text-[11px]">
          <span className="font-semibold text-[#f6d99f] flex items-center gap-1.5">
            <Sparkles className="h-3.5 w-3.5 text-amber-400" />
            Thanh Tiện ích hiện tại ({ids.length}/{MAX_QUICK_ACTIONS})
          </span>
          <button
            type="button"
            onClick={reset}
            className="flex items-center gap-1 text-[10px] text-zinc-400 hover:text-amber-300 transition"
            title="Khôi phục 4 phím tắt mặc định"
          >
            <RotateCcw className="h-3 w-3" />
            Mặc định
          </button>
        </div>

        <div className="flex flex-wrap items-center gap-1.5">
          {ids.map((id, index) => {
            const def = QUICK_ACTIONS_MAP[id];
            if (!def) return null;
            const Icon = def.icon;
            return (
              <div
                key={id}
                className="group relative flex items-center gap-1.5 rounded-lg border border-[#a88952]/60 bg-[#2f2015] px-2 py-1.5 shadow-md"
              >
                <Icon className="h-3.5 w-3.5 text-amber-300 shrink-0" />
                <span className="text-[11px] font-semibold text-[#f9ecd2]">{def.label}</span>

                {/* Move Left / Right buttons */}
                <div className="flex items-center opacity-70 group-hover:opacity-100">
                  {index > 0 && (
                    <button
                      type="button"
                      onClick={() => handleMove(index, -1)}
                      className="p-0.5 hover:text-amber-300 text-zinc-400"
                      title="Di chuyển sang trái"
                    >
                      <ChevronLeft className="h-3 w-3" />
                    </button>
                  )}
                  {index < ids.length - 1 && (
                    <button
                      type="button"
                      onClick={() => handleMove(index, 1)}
                      className="p-0.5 hover:text-amber-300 text-zinc-400"
                      title="Di chuyển sang phải"
                    >
                      <ChevronRight className="h-3 w-3" />
                    </button>
                  )}
                  <button
                    type="button"
                    disabled={ids.length <= MIN_QUICK_ACTIONS}
                    onClick={() => handleToggle(id)}
                    className="ml-0.5 p-0.5 hover:text-rose-400 text-zinc-400 disabled:opacity-30"
                    title={ids.length <= MIN_QUICK_ACTIONS ? "Tối thiểu 1 phím tắt" : "Xóa phím tắt"}
                  >
                    <X className="h-3 w-3" />
                  </button>
                </div>
              </div>
            );
          })}

          {/* Always pinned master menu */}
          <div className="flex items-center gap-1.5 rounded-lg border border-[#6b5333]/50 bg-[#1a120d] px-2 py-1.5 opacity-60">
            <span className="text-[10px] text-zinc-400">+ Danh mục (cố định)</span>
          </div>
        </div>

        <p className="mt-2 text-[10px] text-[#a69273]">
          Ghi chú: Nút &quot;Danh mục&quot; luôn xuất hiện ở cuối để bạn mở toàn bộ chức năng.
        </p>
      </div>

      {/* Categories Filter Tabs */}
      <div>
        <div className="mb-2 flex items-center justify-between">
          <span className="text-[11px] font-semibold text-[#d4b578]">Chọn chức năng thêm vào</span>
          <span className="text-[10px] text-zinc-400">Tối đa 5 nút tiện ích</span>
        </div>
        <div className="flex flex-wrap gap-1 border-b border-[#8f6d40]/30 pb-2">
          {CATEGORIES.map((cat) => (
            <button
              key={cat}
              type="button"
              onClick={() => setSelectedCategory(cat)}
              className={`rounded-md px-2 py-1 text-[10.5px] transition ${
                selectedCategory === cat
                  ? "bg-[#6b4728] text-amber-200 font-bold border border-[#b89456]"
                  : "bg-[#251a12]/60 text-zinc-400 hover:text-[#f4e8cb] hover:bg-[#342419]"
              }`}
            >
              {cat}
            </button>
          ))}
        </div>
      </div>

      {/* Available Actions Grid */}
      <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-3 max-h-[260px] overflow-y-auto pr-1">
        {filteredActions.map((item) => {
          const isSelected = ids.includes(item.id);
          const Icon = item.icon;
          const isFull = ids.length >= MAX_QUICK_ACTIONS && !isSelected;

          return (
            <button
              key={item.id}
              type="button"
              disabled={isFull}
              onClick={() => handleToggle(item.id)}
              className={`flex items-center justify-between rounded-lg border p-2 text-left transition ${
                isSelected
                  ? "border-amber-500/70 bg-gradient-to-r from-[#442c19] to-[#2e1d10] text-[#f7e6c4]"
                  : isFull
                    ? "border-[#4a3a2a]/40 bg-[#16100b]/40 text-zinc-500 cursor-not-allowed"
                    : "border-[#6b5235]/50 bg-[#251a13]/70 text-[#d8c5a2] hover:bg-[#3d2a1b] hover:border-[#b89456]/60"
              }`}
            >
              <div className="flex items-center gap-2 min-w-0">
                <Icon className={`h-4 w-4 shrink-0 ${isSelected ? "text-amber-300" : "text-[#bda87f]"}`} />
                <div className="min-w-0">
                  <div className="truncate text-[11px] font-semibold">{item.label}</div>
                  <div className="text-[9px] text-[#8e7a5c] truncate">{item.category}</div>
                </div>
              </div>
              <div className="shrink-0 ml-1">
                {isSelected ? (
                  <Check className="h-3.5 w-3.5 text-emerald-400" />
                ) : (
                  <Plus className={`h-3.5 w-3.5 ${isFull ? "text-zinc-600" : "text-[#a88952]"}`} />
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );

  if (inline) {
    return (
      <section className="rounded-2xl border border-zinc-900 bg-zinc-900/40 p-3 sm:p-4">
        <div className="flex items-center gap-2 mb-3">
          <Settings2 className="w-4 h-4 text-amber-400" />
          <h3 className="font-serif text-sm font-bold text-zinc-100">
            Tùy chỉnh Nút Tiện Ích Thao Tác Nhanh
          </h3>
        </div>
        <p className="mb-3 text-[11px] text-zinc-400">
          Chọn các tính năng hay dùng nhất (Bản đồ, Túi đồ, Luyện Cổ, Nhiệm vụ...) để hiển thị trên thanh Tiện ích ngoài màn hình.
        </p>
        {content}
      </section>
    );
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center select-none md:p-4">
      <div className="fixed inset-0 bg-black/65 backdrop-blur-sm" onClick={onClose} />
      <div className="relative z-10 h-full w-full overflow-y-auto border border-[#a88952]/70 bg-[#16100bf5] p-4 text-[#f4e8cb] shadow-2xl md:h-auto md:max-h-[90dvh] md:w-[min(94vw,440px)] md:rounded-2xl">
        <div className="mb-3 flex items-center justify-between border-b border-[#b89a66]/30 pb-2">
          <div className="flex items-center gap-2">
            <Settings2 className="h-4 w-4 text-[#d8b979]" />
            <span className="font-serif text-sm font-semibold text-[#f6d99f]">
              Tùy chỉnh Nút Tiện Ích
            </span>
          </div>
          {onClose && (
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg p-1 text-[#d8b979] hover:bg-white/10 hover:text-white"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        {content}
        <div className="mt-3 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg border border-[#b89456]/60 bg-gradient-to-r from-[#5a3b22] to-[#3f2714] px-4 py-1.5 text-xs font-semibold text-[#faecd1] hover:brightness-110 active:scale-95"
          >
            Hoàn tất
          </button>
        </div>
      </div>
    </div>
  );
}
