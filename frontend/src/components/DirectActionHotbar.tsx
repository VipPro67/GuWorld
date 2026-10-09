"use client";

import { useCallback, useEffect, useMemo, useState, type ComponentType, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { ChevronDown, ChevronUp, GripVertical, Keyboard, MoreHorizontal, Pin, PinOff, Plus, Settings2, Eye, EyeOff } from "lucide-react";
import { normalizeHotbarPreferences, reorderHotbarIds, type HotbarPreferences } from "@/lib/action-hotbar";
export { normalizeHotbarPreferences, reorderHotbarIds } from "@/lib/action-hotbar";

export type HotbarAction = {
  id: string;
  label: string;
  icon: ComponentType<{ className?: string }>;
  color: string;
  group: string;
  locked?: boolean;
};

const STORAGE_KEY = "gu_world_direct_action_hotbar_v1";

function formatHotkey(key: string | undefined) {
  if (!key) return "";
  return key.length === 1 ? key.toUpperCase() : key.replace("Arrow", "").replace(" ", "Space");
}

export function DirectActionHotbar({ actions, onAction }: { actions: HotbarAction[]; onAction: (id: string) => void }) {
  const actionIdKey = actions.map((action) => action.id).join("|");
  const actionIds = useMemo(() => actionIdKey ? actionIdKey.split("|") : [], [actionIdKey]);
  const [preferences, setPreferences] = useState<HotbarPreferences>(() => normalizeHotbarPreferences(null, actionIds));
  const [ready, setReady] = useState(false);
  const [moreOpen, setMoreOpen] = useState(false);
  const [captureId, setCaptureId] = useState<string | null>(null);
  const [draggingId, setDraggingId] = useState<string | null>(null);
  const [maxVisible, setMaxVisible] = useState(5);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      setPreferences(normalizeHotbarPreferences(raw ? JSON.parse(raw) : null, actionIds));
    } catch {
      setPreferences(normalizeHotbarPreferences(null, actionIds));
    }
    setReady(true);
  }, [actionIds, actionIdKey]);

  useEffect(() => {
    if (!ready) return;
    try { localStorage.setItem(STORAGE_KEY, JSON.stringify(preferences)); } catch { /* Storage can be unavailable in private contexts. */ }
  }, [preferences, ready]);

  useEffect(() => {
    const updateWidth = () => {
      const mobile = window.innerWidth < 640;
      setIsMobile(mobile);
      setMaxVisible(mobile ? 6 : window.innerWidth < 1024 ? 5 : 7);
    };
    updateWidth();
    window.addEventListener("resize", updateWidth);
    return () => window.removeEventListener("resize", updateWidth);
  }, []);

  const update = useCallback((change: (current: HotbarPreferences) => HotbarPreferences) => {
    setPreferences((current) => change(current));
  }, []);

  const actionById = useMemo(() => new Map(actions.map((action) => [action.id, action])), [actions]);
  const pinned = preferences.pinnedIds.map((id) => actionById.get(id)).filter((item): item is HotbarAction => !!item);
  const visiblePinned = pinned.slice(0, maxVisible);
  const unpinned = actions.filter((action) => !preferences.pinnedIds.includes(action.id));
  const overflowPinned = pinned.slice(maxVisible);

  const runAction = useCallback((action: HotbarAction) => {
    if (action.locked) return;
    setMoreOpen(false);
    onAction(action.id);
  }, [onAction]);

  useEffect(() => {
    if (captureId) return;
    const onKeyDown = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      if (target?.isContentEditable || ["INPUT", "TEXTAREA", "SELECT"].includes(target?.tagName ?? "")) return;
      const pressed = event.key.toLowerCase();
      const match = Object.entries(preferences.hotkeys).find(([, key]) => key.toLowerCase() === pressed);
      if (!match) return;
      const action = actionById.get(match[0]);
      if (action && !action.locked) {
        event.preventDefault();
        runAction(action);
      }
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [actionById, captureId, preferences.hotkeys, runAction]);

  const captureHotkey = (event: ReactKeyboardEvent<HTMLButtonElement>, actionId: string) => {
    event.preventDefault();
    event.stopPropagation();
    if (event.key === "Escape") { setCaptureId(null); return; }
    if (["Control", "Alt", "Shift", "Meta"].includes(event.key)) return;
    const key = event.key.toLowerCase();
    update((current) => {
      const hotkeys = Object.fromEntries(Object.entries(current.hotkeys).filter(([id, value]) => id !== actionId && value.toLowerCase() !== key));
      hotkeys[actionId] = event.key;
      return { ...current, hotkeys };
    });
    setCaptureId(null);
  };

  const pinAction = (action: HotbarAction) => update((current) => ({ ...current, pinnedIds: [...current.pinnedIds, action.id] }));
  const unpinAction = (action: HotbarAction) => update((current) => ({ ...current, pinnedIds: current.pinnedIds.filter((id) => id !== action.id) }));
  const handleDrop = (targetId: string) => {
    if (!draggingId) return;
    update((current) => ({ ...current, pinnedIds: reorderHotbarIds(current.pinnedIds, draggingId, targetId) }));
    setDraggingId(null);
  };

  if (!preferences.visible) {
    return <button type="button" aria-label="Hiện thanh thao tác" title="Hiện thanh thao tác" onClick={() => update((current) => ({ ...current, visible: true }))} className="grid h-10 w-10 place-items-center rounded-xl border border-emerald-300/20 bg-zinc-950/95 text-emerald-200 shadow-xl backdrop-blur transition hover:bg-emerald-950"><Eye className="h-4 w-4" /></button>;
  }

  const renderAction = (action: HotbarAction, isPinned: boolean) => {
    const Icon = action.icon;
    const hotkey = preferences.hotkeys[action.id];
    return <div key={action.id} draggable={isPinned} onDragStart={() => setDraggingId(action.id)} onDragOver={(event) => event.preventDefault()} onDrop={() => handleDrop(action.id)} onDragEnd={() => setDraggingId(null)} className={`group relative flex shrink-0 ${draggingId === action.id ? "opacity-40" : ""}`}>
      <button type="button" title={`${action.label}${hotkey ? ` · Phím ${formatHotkey(hotkey)}` : ""}${action.locked ? " · Chưa mở khóa" : ""}`} aria-label={action.label} disabled={action.locked} onClick={() => runAction(action)} className={`flex flex-col items-center justify-center rounded-xl transition duration-150 ${preferences.compact ? "h-11 w-11" : "h-11 w-11 sm:min-h-12 sm:min-w-[4.25rem] sm:px-2 sm:py-1.5"} ${action.locked ? "cursor-not-allowed opacity-35" : "text-zinc-300 hover:bg-emerald-400/10 hover:text-white active:scale-95"}`}>
        <Icon className={`h-[18px] w-[18px] ${action.color}`} />
        {!preferences.compact && <span className="mt-1 hidden max-w-[5.5rem] truncate text-[9px] font-medium leading-none sm:block">{action.label}</span>}
        {hotkey && <kbd className="absolute -right-0.5 -top-0.5 rounded border border-amber-300/20 bg-zinc-900 px-1 text-[8px] text-amber-200">{formatHotkey(hotkey)}</kbd>}
      </button>
      {isPinned && <GripVertical className="pointer-events-none absolute -left-1 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-zinc-600 opacity-0 transition group-hover:opacity-100" />}
    </div>;
  };

  return <div className="relative">
    {moreOpen && <section className={`max-h-[min(78dvh,40rem)] overflow-hidden rounded-2xl border border-emerald-200/10 bg-zinc-950/95 text-zinc-200 shadow-2xl shadow-black/50 backdrop-blur-xl ${isMobile ? "absolute right-full top-1/2 z-[65] mr-2 w-[min(78vw,34rem)] -translate-y-1/2" : "absolute bottom-full left-1/2 z-[65] mb-2 w-[min(94vw,34rem)] -translate-x-1/2"}`}>
      <header className="flex items-center justify-between border-b border-white/10 px-3 py-2.5">
        <div><p className="text-xs font-semibold tracking-wide text-emerald-100">Thao tác nhanh</p><p className="mt-0.5 text-[10px] text-zinc-500">Chọn để mở trực tiếp · ghim để đưa lên thanh</p></div>
        <div className="flex items-center gap-1">
          <button type="button" title={preferences.compact ? "Hiện tên chức năng" : "Chỉ hiện biểu tượng"} aria-label="Đổi chế độ thu gọn" onClick={() => update((current) => ({ ...current, compact: !current.compact }))} className="grid h-8 w-8 place-items-center rounded-lg text-zinc-400 hover:bg-white/10 hover:text-white"><Settings2 className="h-4 w-4" /></button>
          <button type="button" title="Ẩn thanh thao tác" aria-label="Ẩn thanh thao tác" onClick={() => update((current) => ({ ...current, visible: false }))} className="grid h-8 w-8 place-items-center rounded-lg text-zinc-400 hover:bg-white/10 hover:text-white"><EyeOff className="h-4 w-4" /></button>
          <button type="button" aria-label="Đóng thao tác nhanh" onClick={() => setMoreOpen(false)} className="grid h-8 w-8 place-items-center rounded-lg text-zinc-400 hover:bg-white/10 hover:text-white">×</button>
        </div>
      </header>
      <div className="max-h-[60dvh] space-y-3 overflow-y-auto p-3">
        {overflowPinned.length > 0 && <section><h3 className="mb-1.5 text-[9px] font-semibold uppercase tracking-widest text-amber-200/70">Đã ghim · ngoài màn hình</h3><div className="grid grid-cols-1 gap-1 sm:grid-cols-2">{overflowPinned.map((action) => <div key={action.id} className="flex items-center gap-1 rounded-lg bg-white/[0.03] p-1"><button type="button" disabled={action.locked} onClick={() => runAction(action)} className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-2 text-left text-xs hover:bg-white/[0.06] disabled:opacity-40"><action.icon className={`h-4 w-4 ${action.color}`} /><span className="truncate">{action.label}</span></button><button type="button" title="Bỏ ghim" aria-label={`Bỏ ghim ${action.label}`} onClick={() => unpinAction(action)} className="rounded p-1.5 text-zinc-500 hover:text-amber-200"><PinOff className="h-3.5 w-3.5" /></button></div>)}</div></section>}
        {Array.from(new Set(unpinned.map((action) => action.group))).map((group) => <section key={group}><h3 className="mb-1.5 text-[9px] font-semibold uppercase tracking-widest text-zinc-500">{group}</h3><div className="grid grid-cols-1 gap-1 sm:grid-cols-2">{unpinned.filter((action) => action.group === group).map((action) => <div key={action.id} className="flex items-center gap-1 rounded-lg bg-white/[0.03] p-1">
          <button type="button" title={action.locked ? "Chưa mở khóa" : action.label} disabled={action.locked} onClick={() => runAction(action)} className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-2 text-left text-xs hover:bg-white/[0.06] disabled:cursor-not-allowed disabled:opacity-40"><action.icon className={`h-4 w-4 shrink-0 ${action.color}`} /><span className="truncate">{action.label}</span>{action.locked && <span className="ml-auto text-[9px] text-zinc-500">Khóa</span>}</button>
          <button type="button" title="Ghim lên thanh" aria-label={`Ghim ${action.label}`} onClick={() => pinAction(action)} className="rounded p-1.5 text-zinc-500 hover:text-emerald-200"><Pin className="h-3.5 w-3.5" /></button>
          <button type="button" title={captureId === action.id ? "Bấm phím muốn gán · Esc để hủy" : `Gán phím tắt${preferences.hotkeys[action.id] ? ` (${formatHotkey(preferences.hotkeys[action.id])})` : ""}`} aria-label={`Gán phím tắt ${action.label}`} onClick={() => setCaptureId(action.id)} onKeyDown={(event) => captureHotkey(event, action.id)} className={`rounded p-1.5 ${captureId === action.id ? "text-amber-200" : "text-zinc-600 hover:text-zinc-300"}`}><Keyboard className="h-3.5 w-3.5" /></button>
        </div>)}</div></section>)}
        {unpinned.length === 0 && overflowPinned.length === 0 && <p className="py-6 text-center text-xs text-zinc-500">Tất cả chức năng đã được ghim.</p>}
        {pinned.length > 0 && <section className="border-t border-white/5 pt-2"><h3 className="mb-1.5 text-[9px] font-semibold uppercase tracking-widest text-zinc-500">Sắp xếp thanh · kéo thả hoặc dùng nút mũi tên</h3><div className="space-y-1">{pinned.map((action, index) => <div key={action.id} className="flex items-center gap-2 rounded-lg bg-white/[0.03] px-2 py-1.5 text-xs"><GripVertical className="h-3.5 w-3.5 text-zinc-600" /><span className="min-w-0 flex-1 truncate">{action.label}</span><button type="button" title="Lên" aria-label={`Đưa ${action.label} lên`} disabled={index === 0} onClick={() => update((current) => ({ ...current, pinnedIds: reorderHotbarIds(current.pinnedIds, action.id, current.pinnedIds[index - 1]) }))} className="rounded p-1 text-zinc-500 hover:text-white disabled:opacity-25"><ChevronUp className="h-3.5 w-3.5" /></button><button type="button" title="Xuống" aria-label={`Đưa ${action.label} xuống`} disabled={index === pinned.length - 1} onClick={() => update((current) => ({ ...current, pinnedIds: reorderHotbarIds(current.pinnedIds, action.id, current.pinnedIds[index + 1]) }))} className="rounded p-1 text-zinc-500 hover:text-white disabled:opacity-25"><ChevronDown className="h-3.5 w-3.5" /></button><button type="button" title="Bỏ ghim" aria-label={`Bỏ ghim ${action.label}`} onClick={() => unpinAction(action)} className="rounded p-1 text-zinc-500 hover:text-amber-200"><PinOff className="h-3.5 w-3.5" /></button><button type="button" title={`Gán phím tắt${preferences.hotkeys[action.id] ? ` (${formatHotkey(preferences.hotkeys[action.id])})` : ""}`} aria-label={`Gán phím tắt ${action.label}`} onClick={() => setCaptureId(action.id)} onKeyDown={(event) => captureHotkey(event, action.id)} className={`rounded p-1 ${captureId === action.id ? "text-amber-200" : "text-zinc-600 hover:text-zinc-300"}`}><Keyboard className="h-3.5 w-3.5" /></button></div>)}</div></section>}
      </div>
    </section>}
    <nav aria-label="Direct Action Hotbar" className="flex max-h-[78dvh] max-w-[calc(100vw-1rem)] flex-col items-center gap-0.5 overflow-y-auto rounded-2xl border border-emerald-100/10 bg-zinc-950/95 p-1.5 shadow-2xl shadow-black/50 backdrop-blur-xl sm:max-h-none sm:max-w-[96vw] sm:flex-row sm:overflow-x-auto sm:overflow-y-visible">
      {visiblePinned.map((action) => <div key={action.id} onDragOver={(event) => event.preventDefault()} onDrop={() => handleDrop(action.id)}>{renderAction(action, true)}</div>)}
      <button type="button" onClick={() => setMoreOpen((open) => !open)} aria-label="Thêm thao tác" aria-expanded={moreOpen} title="Thêm thao tác" className="flex h-11 w-11 shrink-0 flex-col items-center justify-center rounded-xl text-emerald-200 transition hover:bg-emerald-400/10 sm:min-w-[3.75rem]"><span className="flex items-center gap-0.5"><Plus className="h-4 w-4" /><MoreHorizontal className="h-3.5 w-3.5" /></span>{!preferences.compact && <span className="mt-1 hidden text-[9px] leading-none sm:block">Thêm</span>}</button>
      <button type="button" title="Ẩn thanh thao tác" aria-label="Ẩn thanh thao tác" onClick={() => update((current) => ({ ...current, visible: false }))} className="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-zinc-600 transition hover:bg-white/5 hover:text-zinc-300 sm:h-8 sm:w-7"><EyeOff className="h-3.5 w-3.5" /></button>
    </nav>
  </div>;
}
