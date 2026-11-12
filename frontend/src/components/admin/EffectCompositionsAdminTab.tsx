"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Archive, Check, ChevronDown, LoaderCircle, Plus, RefreshCw, Save, Trash2 } from "lucide-react";
import { authFetch } from "@/lib/auth";
import { BACKEND_URL } from "@/lib/config";

type Edge = { from: string; to: string; kind: string; rule_id: string };
type Source = { key: string; gu_id: string; rank: number; block_id: string; operation_index: number; role: "OUTPUT" | "SUPPORT"; source_revision?: string };
type SourceOption = Source & { trigger: string; context: string; executor: string; operation: { op: string; sub_type?: string; value?: number; code?: string }; target: { selector: string }; capabilities?: string[] };
type RuleSet = { revision: string; rules: Array<{ id: string; kind: string; core_capability: string; support_capability: string }> };
type Profile = { id: string; label: string; rule_set: RuleSet };
type Draft = { schema_version: number; name: string; product_profile: string; output_effect_key: string; rule_set_revision: string; sources: Source[]; edges: Edge[] };
type ContextResult = { id: string; label: string; status: string; operations: Array<{ block_id?: string; operation_index: number; operation: string; status: string; reason?: string }> };
type Preview = { draft: Draft; effects: { schema_version: number; blocks: Array<Record<string, unknown>> }; compatibility: ContextResult[] };
type SavedDraft = { id: string; name: string; product_profile: string; definition: Draft; updated_at: string };

const compatibilityTone = (status: string) => status === "supported"
  ? "text-emerald-300"
  : status === "unsupported"
    ? "text-red-300"
    : status === "conditional"
      ? "text-amber-300"
      : "text-orange-300";

const endpoint = `${BACKEND_URL}/api/admin/effect-compositions`;

export function EffectCompositionsAdminTab({ canWrite, addToast }: { canWrite: boolean; addToast: (message: string, type: "success" | "error" | "info") => void }) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [profileID, setProfileID] = useState("");
  const [options, setOptions] = useState<SourceOption[]>([]);
  const [drafts, setDrafts] = useState<SavedDraft[]>([]);
  const [draftID, setDraftID] = useState("");
  const [name, setName] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [selectedOption, setSelectedOption] = useState("");
  const [selectedRole, setSelectedRole] = useState<"OUTPUT" | "SUPPORT">("OUTPUT");
  const [preview, setPreview] = useState<Preview | null>(null);
  const [loading, setLoading] = useState(false);
  const [catalogLoading, setCatalogLoading] = useState(true);

  const selectedProfile = profiles.find(profile => profile.id === profileID);
  const selectedKeys = useMemo(() => new Set(sources.map(source => source.key)), [sources]);
  const availableOptions = options.filter(option => !selectedKeys.has(option.key));

  const loadDrafts = useCallback(async () => {
    const response = await authFetch(endpoint);
    if (!response.ok) throw new Error("Không tải được composition drafts");
    const data = await response.json() as { drafts: SavedDraft[] };
    setDrafts(data.drafts || []);
  }, []);

  const loadCatalog = useCallback(async (id: string) => {
    if (!id) return;
    setCatalogLoading(true);
    try {
      const response = await authFetch(`${endpoint}/catalog?product_profile=${encodeURIComponent(id)}`);
      if (!response.ok) throw new Error("Không tải được effect source catalog");
      const data = await response.json() as { profiles: Profile[]; sources: SourceOption[] };
      setProfiles(data.profiles || []);
      setOptions(data.sources || []);
    } catch (error) {
      addToast(error instanceof Error ? error.message : "Không tải được catalog", "error");
    } finally { setCatalogLoading(false); }
  }, [addToast]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const response = await authFetch(`${endpoint}/catalog`);
        if (!response.ok) throw new Error("Không tải được product profile");
        const data = await response.json() as { profiles: Profile[] };
        if (cancelled) return;
        const loaded = data.profiles || [];
        setProfiles(loaded);
        if (loaded.length) setProfileID(loaded[0].id);
        await loadDrafts();
      } catch (error) { if (!cancelled) addToast(error instanceof Error ? error.message : "Không tải được composition catalog", "error"); }
    })();
    return () => { cancelled = true; };
  }, [addToast, loadDrafts]);

  useEffect(() => { void loadCatalog(profileID); }, [profileID, loadCatalog]);

  const resetDraft = () => { setDraftID(""); setName(""); setSources([]); setPreview(null); };
  const addSource = () => {
    const option = options.find(item => item.key === selectedOption);
    if (!option) return;
    const role = sources.length === 0 ? "OUTPUT" : selectedRole;
    const next = [...sources.filter(source => role !== "OUTPUT" || source.role !== "OUTPUT"), { ...option, role }];
    setSources(next);
    setPreview(null);
    setSelectedOption("");
  };
  const chooseDraft = (draft: SavedDraft) => {
    setDraftID(draft.id);
    setName(draft.name);
    setProfileID(draft.product_profile);
    setSources(draft.definition.sources || []);
    setPreview(null);
  };

  const makeDraft = (): Draft => {
    const output = sources.find(source => source.role === "OUTPUT");
    const supportNodes = sources.filter(source => source.role === "SUPPORT");
    const rule = selectedProfile?.rule_set.rules[0];
    return {
      schema_version: 1, name, product_profile: profileID, output_effect_key: output?.key || "",
      rule_set_revision: selectedProfile?.rule_set.revision || "", sources,
      edges: output && rule ? supportNodes.map(source => ({ from: source.key, to: output.key, kind: rule.kind, rule_id: rule.id })) : [],
    };
  };

  const validate = async (candidate = makeDraft()) => {
    setLoading(true); setPreview(null);
    try {
      const response = await authFetch(`${endpoint}/validate`, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(candidate) });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Composition chưa hợp lệ");
      setPreview(data.preview as Preview);
      return true;
    } catch (error) { addToast(error instanceof Error ? error.message : "Validation thất bại", "error"); return false; }
    finally { setLoading(false); }
  };

  const save = async () => {
    if (!canWrite) return;
    const candidate = makeDraft();
    if (!await validate(candidate)) return;
    setLoading(true);
    try {
      const response = await authFetch(draftID ? `${endpoint}/${draftID}` : endpoint, {
        method: draftID ? "PUT" : "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ draft: candidate }),
      });
      const data = await response.json();
      if (!response.ok) throw new Error(data.error || "Không lưu được composition draft");
      setDraftID(data.id);
      addToast("Composition draft đã lưu sau khi revalidate source và rule hiện tại.", "success");
      await loadDrafts();
      if (data.preview) setPreview(data.preview as Preview);
    } catch (error) { addToast(error instanceof Error ? error.message : "Lưu draft thất bại", "error"); }
    finally { setLoading(false); }
  };

  const archive = async () => {
    if (!draftID || !canWrite || !window.confirm("Archive composition draft này?")) return;
    setLoading(true);
    try {
      const response = await authFetch(`${endpoint}/${draftID}/archive`, { method: "POST" });
      if (!response.ok) throw new Error("Không archive được draft");
      addToast("Composition draft đã archive.", "success"); resetDraft(); await loadDrafts();
    } catch (error) { addToast(error instanceof Error ? error.message : "Archive thất bại", "error"); }
    finally { setLoading(false); }
  };

  return <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_340px]">
    <section className="space-y-4 rounded-xl border border-zinc-800 bg-zinc-950/70 p-4">
      <header><h2 className="text-sm font-bold text-zinc-100">Composition drafts dùng chung</h2><p className="mt-1 text-xs text-zinc-500">Chọn effect source theo Gu definition/rank, xem provenance và rule, rồi lưu draft. Gameplay activation vẫn do từng product lifecycle quản lý.</p></header>
      <div className="grid gap-3 md:grid-cols-2">
        <label className="text-xs text-zinc-400">Product profile<select value={profileID} onChange={event => { setProfileID(event.target.value); setSources([]); setPreview(null); }} className="mt-1 block w-full rounded border border-zinc-700 bg-zinc-900 p-2 text-zinc-100">{profiles.map(profile => <option key={profile.id} value={profile.id}>{profile.label} · {profile.id}</option>)}</select></label>
        <label className="text-xs text-zinc-400">Tên draft<input value={name} onChange={event => { setName(event.target.value); setPreview(null); }} maxLength={120} className="mt-1 block w-full rounded border border-zinc-700 bg-zinc-900 p-2 text-zinc-100" /></label>
      </div>
      {selectedProfile && <div className="rounded border border-zinc-800 bg-black/30 p-2 text-[10px] text-zinc-400">Rule revision <code className="text-amber-300">{selectedProfile.rule_set.revision}</code>{selectedProfile.rule_set.rules.map(rule => <div key={rule.id}>{rule.id} · {rule.kind} · requires {rule.core_capability} + {rule.support_capability}</div>)}</div>}
      <div className="flex flex-wrap gap-2">
        <select value={selectedOption} onChange={event => setSelectedOption(event.target.value)} disabled={catalogLoading || !canWrite} className="min-w-64 flex-1 rounded border border-zinc-700 bg-zinc-900 p-2 text-xs text-zinc-100"><option value="">{catalogLoading ? "Đang tải effect source…" : "Chọn effect source"}</option>{availableOptions.map(option => <option key={option.key} value={option.key}>{option.gu_id} R{option.rank} · {option.block_id}[{option.operation_index}] {option.operation.op}/{option.operation.sub_type || ""} · {option.trigger}</option>)}</select>
        <select value={selectedRole} onChange={event => setSelectedRole(event.target.value as "OUTPUT" | "SUPPORT")} className="rounded border border-zinc-700 bg-zinc-900 p-2 text-xs text-zinc-100"><option value="OUTPUT">Output</option><option value="SUPPORT">Support</option></select>
        <button type="button" onClick={addSource} disabled={!selectedOption || !canWrite} className="rounded bg-zinc-800 px-3 py-2 text-xs text-zinc-100 disabled:opacity-40"><Plus className="mr-1 inline h-3 w-3" />Thêm</button>
      </div>
      <div className="space-y-2">{sources.map((source, index) => {
        const option = options.find(item => item.key === source.key);
        return <div key={`${source.key}-${index}`} className="flex flex-wrap items-center justify-between gap-2 rounded border border-zinc-800 p-2 text-xs"><div><span className={source.role === "OUTPUT" ? "text-amber-300" : "text-sky-300"}>{source.role}</span><span className="ml-2 text-zinc-200">{source.gu_id} R{source.rank} · {source.block_id}[{source.operation_index}]</span><div className="mt-1 text-[10px] text-zinc-500">{option?.context || "source không còn trong catalog"} · revision {source.source_revision?.slice(0, 12) || "chưa pin"}</div></div><button type="button" disabled={!canWrite} onClick={() => { setSources(current => current.filter((_, i) => i !== index)); setPreview(null); }} className="text-red-400 disabled:opacity-40"><Trash2 className="h-3.5 w-3.5" /></button></div>;
      })}</div>
      <div className="flex flex-wrap gap-2"><button type="button" disabled={loading || !canWrite} onClick={() => void validate()} className="rounded border border-zinc-700 bg-zinc-800 px-3 py-2 text-xs text-zinc-100 disabled:opacity-40">{loading ? <LoaderCircle className="mr-1 inline h-3 w-3 animate-spin" /> : <Check className="mr-1 inline h-3 w-3" />}Validate & preview</button><button type="button" disabled={loading || !canWrite} onClick={() => void save()} className="rounded bg-amber-700 px-3 py-2 text-xs font-semibold text-white disabled:opacity-40"><Save className="mr-1 inline h-3 w-3" />{draftID ? "Validate & save" : "Validate & save draft"}</button>{draftID && <button type="button" disabled={loading || !canWrite} onClick={() => void archive()} className="rounded border border-red-900/70 px-3 py-2 text-xs text-red-300 disabled:opacity-40"><Archive className="mr-1 inline h-3 w-3" />Archive</button>}<button type="button" onClick={resetDraft} className="rounded border border-zinc-800 px-3 py-2 text-xs text-zinc-400">Draft mới</button></div>
      {!canWrite && <p className="text-xs text-amber-300">Bạn có quyền xem composition draft; cần quyền sửa từ điển Cổ để tạo hoặc cập nhật.</p>}
      {preview && <details open className="rounded border border-zinc-800 bg-black/30 p-3"><summary className="cursor-pointer text-xs font-semibold text-zinc-200">Preview · {preview.effects.blocks.length} Block v2 output</summary><div className="mt-3 grid gap-2 md:grid-cols-2">{preview.compatibility.map(context => <div key={context.id} className="rounded border border-zinc-800 p-2"><div className="flex justify-between text-[11px]"><span>{context.label}</span><span className={compatibilityTone(context.status)}>{context.status}</span></div>{context.operations.map((operation, i) => <div key={i} className="mt-1 text-[10px] text-zinc-500">{operation.block_id}[{operation.operation_index}] {operation.operation} · <span className={compatibilityTone(operation.status)}>{operation.status}</span>{operation.reason ? ` · ${operation.reason}` : ""}</div>)}</div>)}</div><pre className="mt-3 max-h-72 overflow-auto rounded bg-black p-3 text-[10px] text-zinc-300">{JSON.stringify(preview.effects, null, 2)}</pre></details>}
    </section>
    <aside className="rounded-xl border border-zinc-800 bg-zinc-950/70 p-4"><div className="mb-3 flex items-center justify-between"><h3 className="text-xs font-bold text-zinc-200">Draft đã lưu</h3><button type="button" onClick={() => void loadDrafts()} title="Làm mới"><RefreshCw className="h-3.5 w-3.5 text-zinc-500" /></button></div><div className="space-y-2">{drafts.filter(draft => !profileID || draft.product_profile === profileID).map(draft => <button key={draft.id} type="button" onClick={() => chooseDraft(draft)} className={`w-full rounded border p-2 text-left ${draft.id === draftID ? "border-amber-700 bg-amber-950/20" : "border-zinc-800 hover:border-zinc-700"}`}><span className="block text-xs text-zinc-200">{draft.name}</span><span className="mt-1 block text-[10px] text-zinc-500">{draft.product_profile} · {draft.definition.sources.length} source</span></button>)}{drafts.length === 0 && <p className="text-xs text-zinc-600">Chưa có draft nào.</p>}</div><p className="mt-4 border-t border-zinc-800 pt-3 text-[10px] text-zinc-500"><ChevronDown className="mr-1 inline h-3 w-3" />Draft tham chiếu source revision mới nhất. Khi gameplay activate, backend phải revalidate và tạo snapshot bất biến.</p></aside>
    </div>;
}
