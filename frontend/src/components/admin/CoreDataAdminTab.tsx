"use client";

import React from "react";
import { Bug, FlaskConical, Gift, Package, PawPrint, Plus } from "lucide-react";
import { AdminDataGrid, AdminGridColumn } from "./AdminDataGrid";

type CoreDataKind = "gus" | "items" | "recipes" | "monsters" | "loot_tables";

type Props = {
  kind: CoreDataKind;
  rows: ApiResponseData[];
  loading: boolean;
  onCreate: () => void;
  onEdit: (row: ApiResponseData) => void;
  onCopy: (row: ApiResponseData) => void;
  onDelete: (row: ApiResponseData) => void;
};

function arrayLength(value: unknown): number {
  try {
    const parsed = typeof value === "string" ? JSON.parse(value) : value;
    return Array.isArray(parsed) ? parsed.length : 0;
  } catch {
    return 0;
  }
}

const configs: Record<CoreDataKind, { title: string; icon: React.ReactNode; columns: AdminGridColumn<ApiResponseData>[]; rowKey: (row: ApiResponseData) => string }> = {
  gus: {
    title: "Danh Sách Cổ Trùng",
    icon: <Bug className="h-4 w-4 text-amber-400" />,
    rowKey: (row) => `${row.id}:${row.rank}`,
    columns: [
      { key: "id", label: "ID", value: (row) => row.id, className: "font-mono text-zinc-500" },
      { key: "name", label: "Tên", value: (row) => row.name, className: "font-semibold text-zinc-200" },
      { key: "rank", label: "Rank", value: (row) => row.rank },
      { key: "gu_type", label: "Loại", value: (row) => row.gu_type },
      { key: "base_price", label: "Giá", value: (row) => row.base_price, render: (row) => Number(row.base_price ?? 0).toLocaleString("vi-VN"), className: "text-right font-mono text-yellow-500" },
    ],
  },
  items: {
    title: "Danh Sách Vật Phẩm",
    icon: <Package className="h-4 w-4 text-emerald-400" />,
    rowKey: (row) => row.id,
    columns: [
      { key: "id", label: "ID", value: (row) => row.id, className: "font-mono text-zinc-500" },
      { key: "name", label: "Tên", value: (row) => row.name, className: "font-semibold text-zinc-200" },
      { key: "item_type", label: "Loại", value: (row) => row.item_type },
      { key: "rank", label: "Rank", value: (row) => row.rank },
      { key: "base_price", label: "Giá", value: (row) => row.base_price, render: (row) => Number(row.base_price ?? 0).toLocaleString("vi-VN"), className: "text-right font-mono text-yellow-500" },
    ],
  },
  recipes: {
    title: "Danh Sách Cổ Phương",
    icon: <FlaskConical className="h-4 w-4 text-emerald-400" />,
    rowKey: (row) => row.id,
    columns: [
      { key: "id", label: "ID", value: (row) => row.id, className: "font-mono text-zinc-500" },
      { key: "target", label: "Cổ trùng đích", value: (row) => row.target_gu_name || row.target_gu_id, className: "font-semibold text-zinc-200" },
      { key: "success_rate", label: "Tỉ lệ", value: (row) => row.success_rate, render: (row) => `${Math.round(Number(row.success_rate ?? 0) * 100)}%` },
      { key: "primeval_stones_cost", label: "Nguyên thạch", value: (row) => row.primeval_stones_cost, render: (row) => Number(row.primeval_stones_cost ?? 0).toLocaleString("vi-VN"), className: "text-right font-mono text-yellow-500" },
    ],
  },
  monsters: {
    title: "Danh Sách Quái Vật",
    icon: <PawPrint className="h-4 w-4 text-amber-400" />,
    rowKey: (row) => row.id,
    columns: [
      { key: "id", label: "ID", value: (row) => row.id, className: "font-mono text-zinc-500" },
      { key: "name", label: "Tên", value: (row) => row.name, className: "font-semibold text-zinc-200" },
      { key: "type", label: "Loại", value: (row) => row.type },
      { key: "max_hp", label: "HP", value: (row) => row.max_hp, className: "font-mono text-red-400" },
      { key: "base_attack", label: "Tấn công", value: (row) => row.base_attack, className: "font-mono text-orange-400" },
      { key: "base_defense", label: "Phòng thủ", value: (row) => row.base_defense, className: "font-mono text-blue-400" },
      { key: "loot_table_id", label: "Loot", value: (row) => row.loot_table_id, className: "font-mono text-[10px]" },
    ],
  },
  loot_tables: {
    title: "Danh Sách Bảng Loot",
    icon: <Gift className="h-4 w-4 text-amber-400" />,
    rowKey: (row) => row.id,
    columns: [
      { key: "id", label: "ID", value: (row) => row.id, className: "font-mono text-zinc-500" },
      { key: "name", label: "Tên", value: (row) => row.name, className: "font-semibold text-zinc-200" },
      { key: "entries", label: "Số loot", value: (row) => arrayLength(row.entries) },
    ],
  },
};

export function CoreDataAdminTab({ kind, rows, loading, onCreate, onEdit, onCopy, onDelete }: Props) {
  const config = configs[kind];
  return <div className="space-y-4 rounded-xl border border-zinc-800/60 bg-zinc-900/40 p-4">
    <div className="flex items-center justify-between">
      <h3 className="flex items-center gap-2 font-serif text-sm font-bold text-zinc-200">{config.icon}{config.title}</h3>
      <button onClick={onCreate} className="flex items-center gap-1.5 rounded-lg bg-emerald-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-emerald-500"><Plus className="h-3.5 w-3.5" />Thêm</button>
    </div>
    <AdminDataGrid rows={rows} columns={config.columns} rowKey={config.rowKey} loading={loading} onEdit={onEdit} onCopy={onCopy} onDelete={onDelete} />
  </div>;
}
