"use client";

import React, { useCallback, useEffect, useState } from "react";
import { authFetch } from "@/lib/auth";
import { Edit3, Plus, RefreshCw, Save, Trash2, X } from "lucide-react";
import type { PreparedAdminAction } from "./actions";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";
type NPCRow = Record<string, unknown>;

export const ApertureNPCAdmin: React.FC<{
  apertureId: string;
  canEdit: boolean;
  canDelete: boolean;
  addToast: (msg: string, type: "success" | "error" | "info") => void;
  exec: (url: string, method: string, body?: ApiResponseData) => Promise<boolean>;
  prepareDelete: (label: string, target: string) => PreparedAdminAction | null;
}> = ({ apertureId, canEdit, canDelete, addToast, exec, prepareDelete }) => {
  const [rows, setRows] = useState<NPCRow[]>([]);
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<{ row: NPCRow | null; json: string } | null>(null);
  const base = `${BACKEND_URL}/api/admin/apertures/${apertureId}/variant-npcs`;
  const load = useCallback(async () => {
    setBusy(true);
    try { const res=await authFetch(base);if(!res.ok)throw new Error("Không thể tải danh sách cổ sư dị nhân");setRows(await res.json()); }
    catch(error){addToast(error instanceof Error?error.message:"Lỗi tải cổ sư dị nhân","error");}
    finally{setBusy(false);}
  },[base,addToast]);
  useEffect(()=>{void load();},[load]);
  const createDefaults={template_source:"VARIANT_RACE",template_id:"",species:"",race:"",name:"",rank:1,sub_rank:1,experience:0,talent:50,loyalty:70,hp:120,max_hp:120,base_attack:14,base_defense:9,speed:9,count:1,hunger:0,max_hunger:100,upkeep_rate:0,lifecycle_state:"ACTIVE",execution_state:"STORED",behavior_profile:"WORKER",ai_profile:"",capabilities:{MOVE:true,FOLLOW:true,GUARD:true,COMBAT:true,PRODUCE:true,REFINE:true,TRAIN:true,EQUIP_GU:true},traits:{},progression:{},formation_slot:0,path:"REFINEMENT",role:"NONE"};
  const edit=(row:NPCRow|null)=>setEditing({row,json:JSON.stringify(row||createDefaults,null,2)});
  const save=async()=>{if(!editing)return;let payload:NPCRow;try{payload=JSON.parse(editing.json) as NPCRow;}catch{addToast("JSON không hợp lệ","error");return;}const id=String(editing.row?.id||"");if(await exec(`${base}${id?`/${id}`:""}`,id?"PUT":"POST",payload as ApiResponseData)){setEditing(null);void load();}};
  const dismiss=async(row:NPCRow)=>{const id=String(row.id);const prepared=prepareDelete("giải tán cổ sư dị nhân",`${row.name} · ${row.race} · ${id}`);if(prepared&&await exec(`${base}/${id}`,"DELETE",prepared))void load();};
  return <section className="rounded-xl border border-zinc-800 p-3 space-y-2">
    <div className="flex items-center justify-between"><h4 className="text-xs font-bold text-zinc-200">Cổ sư dị nhân không phải người chơi ({rows.length})</h4><div className="flex gap-2"><button onClick={()=>void load()} disabled={busy} className="text-[10px] text-zinc-500"><RefreshCw className={`inline w-3 h-3 mr-1 ${busy?"animate-spin":""}`}/>Làm mới</button>{canEdit&&<button onClick={()=>edit(null)} className="px-2 py-1 rounded bg-amber-600/20 text-amber-400 text-[10px]"><Plus className="inline w-3 h-3 mr-1"/>Thêm cổ sư</button>}</div></div>
    {rows.length===0?<p className="text-[10px] text-zinc-500">Chưa có cổ sư NPC trong tiên khiếu này.</p>:rows.map(row=>{const equipment=Array.isArray(row.equipped_gu)?row.equipped_gu as NPCRow[]:[];return <div key={String(row.id)} className="flex items-start justify-between gap-2 border-t border-zinc-800/60 pt-2"><div className="min-w-0"><div className="text-xs text-zinc-200 truncate">{String(row.name)} · {String(row.race)} · Rank {String(row.rank)}.{String(row.sub_rank)}</div><div className="text-[10px] text-zinc-500 truncate">{String(row.lifecycle_state)} / {String(row.execution_state)} · {String(row.role)} · {String(row.path)} · HP {String(row.hp)}/{String(row.max_hp)} · trung thành {String(row.loyalty)}</div><div className="text-[10px] text-zinc-600 truncate">Cổ trùng trang bị: {equipment.length?equipment.map(g=>`${String(g.gu_id)} R${String(g.rank)} · ô ${String(g.slot)}`).join(" | "):"không có"}</div></div><div className="flex gap-1">{canEdit&&<button title="Sửa toàn bộ field" onClick={()=>edit(row)} className="p-1 text-zinc-500 hover:text-amber-400"><Edit3 className="w-3.5 h-3.5"/></button>}{canDelete&&String(row.lifecycle_state)==="ACTIVE"&&<button title="Giải tán (giữ lịch sử)" onClick={()=>void dismiss(row)} className="p-1 text-zinc-500 hover:text-red-400"><Trash2 className="w-3.5 h-3.5"/></button>}</div></div>;})}
    {editing&&<div className="fixed inset-0 z-[70] bg-black/70 flex items-center justify-center p-4" onClick={()=>setEditing(null)}><div onClick={e=>e.stopPropagation()} className="w-full max-w-3xl bg-zinc-950 border border-zinc-800 rounded-xl p-4 space-y-3"><div className="flex justify-between items-center"><h3 className="text-sm font-bold text-zinc-200">{editing.row?"Sửa cổ sư dị nhân":"Tạo cổ sư dị nhân NPC"}</h3><button onClick={()=>setEditing(null)}><X className="w-4 h-4 text-zinc-500"/></button></div><p className="text-[10px] text-zinc-500">Bảng dữ liệu đầy đủ của non_player_actors. Không sửa id, actor_kind, owner_player_id, aperture_id, legacy fields hoặc version. Khi sửa, trường equipped_gu quản lý danh sách Cổ trùng theo player_gu_id và slot; hệ thống kiểm tra quyền sở hữu, trạng thái và cấp bậc. Cổ sư mới tạo trước, sau đó mới giao Cổ.</p><textarea value={editing.json} onChange={e=>setEditing({...editing,json:e.target.value})} rows={24} spellCheck={false} className="w-full bg-zinc-900 border border-zinc-800 rounded-lg p-3 text-[11px] font-mono text-zinc-200"/><div className="flex justify-end"><button onClick={()=>void save()} className="px-3 py-2 rounded bg-amber-600 text-xs text-white"><Save className="inline w-3.5 h-3.5 mr-1"/>Lưu</button></div></div></div>}
  </section>;
};
