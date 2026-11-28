"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  AlertTriangle,
  Clock,
  Compass,
  Droplets,
  Edit3,
  Flame,
  Gem,
  Layers,
  Mountain,
  Plus,
  RefreshCw,
  Search,
  Sparkles,
  Trash2,
  Wrench,
  X,
  Zap,
} from "lucide-react";
import { authFetch } from "@/lib/auth";
import { prepareAdminAction } from "./actions";
import { formatNumber } from "@/lib/utils";

const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || "http://localhost:8080";

type PrimevalVein = {
  id: string;
  region_id: string;
  region_name?: string;
  x: number;
  y: number;
  vein_type: "mortal" | "immortal";
  rarity: string;
  accumulated_since: string;
  produced_minutes: number;
  expires_at: string;
  created_at: string;
  accumulated_stones: number;
  accumulated_qi: number;
  accumulated_chests: number;
  cap_minutes: number;
  remaining_minutes: number;
  is_depleted: boolean;
};

type ResourcePointTemplate = {
  PointType: string;
  Name: string;
  Path: string;
  Category: string;
  ProduceKey: string;
  PointClass: string;
};

type WorldResourcePoint = {
  id: string;
  scope: string;
  region_id: string;
  region_name?: string;
  x: number;
  y: number;
  point_type: string;
  name: string;
  path: string;
  grade: "LOW" | "MID" | "TOP";
  point_class: "NORMAL" | "FAMOUS";
  level: number;
  vitality: number;
  stock: Record<string, number>;
  stock_cap: number;
  last_produced_at: string;
  state: "ACTIVE" | "EXHAUSTED" | string;
  created_at: string;
  updated_at: string;
  template?: ResourcePointTemplate;
};

type RegionOption = {
  id: string;
  name: string;
};

const RARITY_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  "Thường": { bg: "bg-zinc-800/80", text: "text-zinc-300", border: "border-zinc-700" },
  "Hiếm": { bg: "bg-emerald-950/80", text: "text-emerald-300", border: "border-emerald-800/60" },
  "Sử Thi": { bg: "bg-purple-950/80", text: "text-purple-300", border: "border-purple-800/60" },
  "Truyền Thuyết": { bg: "bg-amber-950/80", text: "text-amber-300", border: "border-amber-800/60" },
  "Siêu Cấp": { bg: "bg-rose-950/80", text: "text-rose-300", border: "border-rose-800/60" },
};

const GRADE_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  LOW: { bg: "bg-zinc-800/80", text: "text-zinc-400", border: "border-zinc-700" },
  MID: { bg: "bg-blue-950/80", text: "text-blue-300", border: "border-blue-800/60" },
  TOP: { bg: "bg-amber-950/80", text: "text-amber-300", border: "border-amber-800/60" },
};

export const MinesAdminTab: React.FC<{
  addToast: (message: string, type: "success" | "error" | "info") => void;
  canEdit: boolean;
  canDelete: boolean;
}> = ({ addToast, canEdit, canDelete }) => {
  const [subTab, setSubTab] = useState<"veins" | "points">("veins");
  const [regions, setRegions] = useState<RegionOption[]>([]);
  const [templates, setTemplates] = useState<ResourcePointTemplate[]>([]);

  // 1. Primeval Veins state
  const [veins, setVeins] = useState<PrimevalVein[]>([]);
  const [veinStats, setVeinStats] = useState({ mortalCount: 0, immortalCount: 0, mortalCap: 31, immortalCap: 5 });
  const [veinsLoading, setVeinsLoading] = useState(false);
  const [veinRegionFilter, setVeinRegionFilter] = useState("");
  const [veinTypeFilter, setVeinTypeFilter] = useState("");
  const [veinRarityFilter, setVeinRarityFilter] = useState("");
  const [veinSearch, setVeinSearch] = useState("");

  // Primeval Vein Modal state
  const [editingVein, setEditingVein] = useState<PrimevalVein | null>(null);
  const [isCreatingVein, setIsCreatingVein] = useState(false);
  const [veinForm, setVeinForm] = useState({
    region_id: "",
    x: 0,
    y: 0,
    vein_type: "mortal" as "mortal" | "immortal",
    rarity: "Thường",
    lifetime_days: 30,
    produced_minutes: 0,
    reset_accumulation: false,
  });

  // 2. World Resource Points state
  const [points, setPoints] = useState<WorldResourcePoint[]>([]);
  const [pointStats, setPointStats] = useState({ activeCount: 0, exhaustedCount: 0, famousCount: 0 });
  const [pointsLoading, setPointsLoading] = useState(false);
  const [pointRegionFilter, setPointRegionFilter] = useState("");
  const [pointGradeFilter, setPointGradeFilter] = useState("");
  const [pointClassFilter, setPointClassFilter] = useState("");
  const [pointStateFilter, setPointStateFilter] = useState("");
  const [pointSearch, setPointSearch] = useState("");

  // World Point Modal state
  const [editingPoint, setEditingPoint] = useState<WorldResourcePoint | null>(null);
  const [isCreatingPoint, setIsCreatingPoint] = useState(false);
  const [pointForm, setPointForm] = useState({
    region_id: "",
    x: 0,
    y: 0,
    point_type: "HERB_VALLEY",
    name: "",
    path: "",
    grade: "MID" as "LOW" | "MID" | "TOP",
    point_class: "NORMAL" as "NORMAL" | "FAMOUS",
    level: 1,
    vitality: 100,
    state: "ACTIVE",
  });

  const [saving, setSaving] = useState(false);

  // Load Regions and Templates once
  useEffect(() => {
    const fetchMetadata = async () => {
      try {
        const [regRes, tplRes] = await Promise.all([
          authFetch(`${BACKEND_URL}/api/admin/regions?page=1&limit=200`),
          authFetch(`${BACKEND_URL}/api/admin/world-resource-points/templates`),
        ]);
        if (regRes.ok) {
          const regData = await regRes.json();
          const items = Array.isArray(regData.data) ? regData.data : [];
          setRegions(items.map((r: { id: string; name: string }) => ({ id: r.id, name: r.name || r.id })));
        }
        if (tplRes.ok) {
          const tplData = await tplRes.json();
          setTemplates(Array.isArray(tplData.data) ? tplData.data : []);
        }
      } catch (err) {
        console.warn("Failed to load regions/templates metadata:", err);
      }
    };
    void fetchMetadata();
  }, []);

  // Fetch Primeval Veins
  const fetchVeins = useCallback(async () => {
    setVeinsLoading(true);
    try {
      const params = new URLSearchParams();
      if (veinRegionFilter) params.append("region_id", veinRegionFilter);
      if (veinTypeFilter) params.append("vein_type", veinTypeFilter);
      if (veinRarityFilter) params.append("rarity", veinRarityFilter);
      if (veinSearch) params.append("search", veinSearch);

      const res = await authFetch(`${BACKEND_URL}/api/admin/primeval-veins?${params.toString()}`);
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể tải danh sách Nguyên Tuyền");

      setVeins(Array.isArray(data.data) ? data.data : []);
      setVeinStats({
        mortalCount: Number(data.mortal_count) || 0,
        immortalCount: Number(data.immortal_count) || 0,
        mortalCap: Number(data.mortal_cap) || 31,
        immortalCap: Number(data.immortal_cap) || 5,
      });
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Lỗi tải Nguyên Tuyền", "error");
    } finally {
      setVeinsLoading(false);
    }
  }, [veinRegionFilter, veinTypeFilter, veinRarityFilter, veinSearch, addToast]);

  // Fetch World Resource Points
  const fetchPoints = useCallback(async () => {
    setPointsLoading(true);
    try {
      const params = new URLSearchParams();
      if (pointRegionFilter) params.append("region_id", pointRegionFilter);
      if (pointGradeFilter) params.append("grade", pointGradeFilter);
      if (pointClassFilter) params.append("point_class", pointClassFilter);
      if (pointStateFilter) params.append("state", pointStateFilter);
      if (pointSearch) params.append("search", pointSearch);

      const res = await authFetch(`${BACKEND_URL}/api/admin/world-resource-points?${params.toString()}`);
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Không thể tải điểm tài nguyên thế giới");

      setPoints(Array.isArray(data.data) ? data.data : []);
      setPointStats({
        activeCount: Number(data.active_count) || 0,
        exhaustedCount: Number(data.exhausted_count) || 0,
        famousCount: Number(data.famous_count) || 0,
      });
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Lỗi tải điểm tài nguyên", "error");
    } finally {
      setPointsLoading(false);
    }
  }, [pointRegionFilter, pointGradeFilter, pointClassFilter, pointStateFilter, pointSearch, addToast]);

  useEffect(() => {
    if (subTab === "veins") {
      void fetchVeins();
    } else {
      void fetchPoints();
    }
  }, [subTab, fetchVeins, fetchPoints]);

  // Handlers for Primeval Veins
  const openCreateVeinModal = () => {
    setVeinForm({
      region_id: regions[0]?.id || "",
      x: 0,
      y: 0,
      vein_type: "mortal",
      rarity: "Thường",
      lifetime_days: 30,
      produced_minutes: 0,
      reset_accumulation: false,
    });
    setIsCreatingVein(true);
  };

  const openEditVeinModal = (vein: PrimevalVein) => {
    setEditingVein(vein);
    setVeinForm({
      region_id: vein.region_id,
      x: vein.x,
      y: vein.y,
      vein_type: vein.vein_type,
      rarity: vein.rarity,
      lifetime_days: 30,
      produced_minutes: vein.produced_minutes,
      reset_accumulation: false,
    });
  };

  const handleSaveVein = async () => {
    if (!veinForm.region_id) {
      addToast("Vui lòng chọn khu vực", "error");
      return;
    }
    setSaving(true);
    try {
      if (isCreatingVein) {
        const action = prepareAdminAction("createPrimevalVein", "tạo Nguyên Tuyền", `${veinForm.region_id}:(${veinForm.x},${veinForm.y})`);
        if (!action) return;
        const res = await authFetch(`${BACKEND_URL}/api/admin/primeval-veins`, {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
          body: JSON.stringify(veinForm),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Không thể tạo Nguyên Tuyền");
        addToast(data.message || "Tạo Nguyên Tuyền thành công", "success");
        setIsCreatingVein(false);
      } else if (editingVein) {
        const action = prepareAdminAction("updatePrimevalVein", "cập nhật Nguyên Tuyền", editingVein.id);
        if (!action) return;
        const res = await authFetch(`${BACKEND_URL}/api/admin/primeval-veins/${editingVein.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
          body: JSON.stringify({
            vein_type: veinForm.vein_type,
            rarity: veinForm.rarity,
            x: veinForm.x,
            y: veinForm.y,
            produced_minutes: veinForm.produced_minutes,
            reset_accumulation: veinForm.reset_accumulation,
          }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Không thể cập nhật Nguyên Tuyền");
        addToast(data.message || "Cập nhật Nguyên Tuyền thành công", "success");
        setEditingVein(null);
      }
      void fetchVeins();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Thao tác thất bại", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteVein = async (vein: PrimevalVein) => {
    const action = prepareAdminAction("deletePrimevalVein", "xóa Nguyên Tuyền", vein.id);
    if (!action) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/primeval-veins/${vein.id}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ reason: action.reason }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Lỗi xóa Nguyên Tuyền");
      addToast(data.message || "Xóa Nguyên Tuyền thành công", "success");
      void fetchVeins();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Lỗi xóa", "error");
    }
  };

  const handleSpawnRandomVein = async (veinType: "mortal" | "immortal") => {
    const action = prepareAdminAction("spawnRandomVein", `sinh ngẫu nhiên ${veinType === "immortal" ? "Tiên Tuyền" : "Phàm Tuyền"}`, veinType);
    if (!action) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/primeval-veins/spawn-random`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ vein_type: veinType }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Lỗi sinh ngẫu nhiên");
      addToast(data.message || "Sinh thành công", "success");
      void fetchVeins();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Thất bại", "error");
    }
  };

  // Handlers for World Resource Points
  const openCreatePointModal = () => {
    const defaultTpl = templates[0];
    setPointForm({
      region_id: regions[0]?.id || "",
      x: 0,
      y: 0,
      point_type: defaultTpl?.PointType || "HERB_VALLEY",
      name: defaultTpl?.Name || "",
      path: defaultTpl?.Path || "",
      grade: "MID",
      point_class: (defaultTpl?.PointClass as "NORMAL" | "FAMOUS") || "NORMAL",
      level: 1,
      vitality: 100,
      state: "ACTIVE",
    });
    setIsCreatingPoint(true);
  };

  const openEditPointModal = (pt: WorldResourcePoint) => {
    setEditingPoint(pt);
    setPointForm({
      region_id: pt.region_id,
      x: pt.x,
      y: pt.y,
      point_type: pt.point_type,
      name: pt.name,
      path: pt.path,
      grade: pt.grade,
      point_class: pt.point_class,
      level: pt.level,
      vitality: pt.vitality,
      state: pt.state,
    });
  };

  const handleTemplateChange = (pointType: string) => {
    const found = templates.find((t) => t.PointType === pointType);
    setPointForm((prev) => ({
      ...prev,
      point_type: pointType,
      name: found ? found.Name : prev.name,
      path: found ? found.Path : prev.path,
      point_class: (found?.PointClass as "NORMAL" | "FAMOUS") || prev.point_class,
    }));
  };

  const handleSavePoint = async () => {
    if (!pointForm.region_id || !pointForm.point_type) {
      addToast("Vui lòng chọn khu vực và loại mẫu mỏ", "error");
      return;
    }
    setSaving(true);
    try {
      if (isCreatingPoint) {
        const action = prepareAdminAction("createWorldResourcePoint", "tạo mỏ tài nguyên", `${pointForm.region_id}:(${pointForm.x},${pointForm.y})`);
        if (!action) return;
        const res = await authFetch(`${BACKEND_URL}/api/admin/world-resource-points`, {
          method: "POST",
          headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
          body: JSON.stringify(pointForm),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Không thể tạo điểm tài nguyên");
        addToast(data.message || "Tạo mỏ tài nguyên thành công", "success");
        setIsCreatingPoint(false);
      } else if (editingPoint) {
        const action = prepareAdminAction("updateWorldResourcePoint", "cập nhật mỏ tài nguyên", editingPoint.id);
        if (!action) return;
        const res = await authFetch(`${BACKEND_URL}/api/admin/world-resource-points/${editingPoint.id}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
          body: JSON.stringify({
            name: pointForm.name,
            path: pointForm.path,
            grade: pointForm.grade,
            point_class: pointForm.point_class,
            level: pointForm.level,
            vitality: pointForm.vitality,
            state: pointForm.state,
            x: pointForm.x,
            y: pointForm.y,
          }),
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || "Không thể cập nhật điểm tài nguyên");
        addToast(data.message || "Cập nhật mỏ thành công", "success");
        setEditingPoint(null);
      }
      void fetchPoints();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Thao tác thất bại", "error");
    } finally {
      setSaving(false);
    }
  };

  const handleDeletePoint = async (pt: WorldResourcePoint) => {
    const action = prepareAdminAction("deleteWorldResourcePoint", "xóa mỏ tài nguyên", pt.id);
    if (!action) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/world-resource-points/${pt.id}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ reason: action.reason }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Lỗi xóa mỏ tài nguyên");
      addToast(data.message || "Xóa mỏ thành công", "success");
      void fetchPoints();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Lỗi xóa", "error");
    }
  };

  const handleSweepWorldPoints = async () => {
    const action = prepareAdminAction("sweepWorldResourcePoints", "quét bổ sung mỏ thế giới", "all");
    if (!action) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/world-resource-points/sweep`, {
        method: "POST",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({}),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Lỗi quét bổ sung mỏ");
      addToast(data.message || "Quét bổ sung thành công", "success");
      void fetchPoints();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Lỗi quét bổ sung", "error");
    }
  };

  const handleRestoreVitality = async (pt: WorldResourcePoint) => {
    const action = prepareAdminAction("updateWorldResourcePoint", "hồi phục sinh khí mỏ", pt.id);
    if (!action) return;
    try {
      const res = await authFetch(`${BACKEND_URL}/api/admin/world-resource-points/${pt.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json", "Idempotency-Key": action.idempotencyKey },
        body: JSON.stringify({ vitality: 100, state: "ACTIVE" }),
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Lỗi hồi phục sinh khí");
      addToast("Đã hồi phục 100% sinh khí cho mỏ!", "success");
      void fetchPoints();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Thao tác thất bại", "error");
    }
  };

  return (
    <div className="space-y-6">
      {/* Top Tab Switcher */}
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-zinc-800 pb-3">
        <div className="flex items-center gap-2">
          <button
            onClick={() => setSubTab("veins")}
            className={`flex items-center gap-2 rounded-xl px-4 py-2.5 text-xs font-semibold font-mono transition-all cursor-pointer ${
              subTab === "veins"
                ? "bg-cyan-950/80 text-cyan-300 border border-cyan-800/80 shadow-lg shadow-cyan-950/40"
                : "bg-zinc-900/60 text-zinc-400 border border-zinc-800/80 hover:text-zinc-200 hover:bg-zinc-800/60"
            }`}
          >
            <Droplets className="h-4 w-4 text-cyan-400" />
            1. Quản Lý Nguyên Tuyền ({veinStats.mortalCount + veinStats.immortalCount})
          </button>
          <button
            onClick={() => setSubTab("points")}
            className={`flex items-center gap-2 rounded-xl px-4 py-2.5 text-xs font-semibold font-mono transition-all cursor-pointer ${
              subTab === "points"
                ? "bg-lime-950/80 text-lime-300 border border-lime-800/80 shadow-lg shadow-lime-950/40"
                : "bg-zinc-900/60 text-zinc-400 border border-zinc-800/80 hover:text-zinc-200 hover:bg-zinc-800/60"
            }`}
          >
            <Mountain className="h-4 w-4 text-lime-400" />
            2. Mỏ Tài Nguyên Thế Giới ({points.length})
          </button>
        </div>

        {subTab === "veins" ? (
          <div className="flex flex-wrap items-center gap-2">
            {canEdit && (
              <>
                <button
                  onClick={() => void handleSpawnRandomVein("mortal")}
                  className="flex items-center gap-1.5 rounded-lg border border-cyan-800/60 bg-cyan-950/50 px-3 py-1.5 text-xs font-mono text-cyan-300 hover:bg-cyan-900/60 transition-colors cursor-pointer"
                  title="Tự động sinh ngẫu nhiên 1 Phàm Tuyền theo quy tắc phân bổ Đại Vực"
                >
                  <Sparkles className="h-3.5 w-3.5" /> + Phàm Tuyền
                </button>
                <button
                  onClick={() => void handleSpawnRandomVein("immortal")}
                  className="flex items-center gap-1.5 rounded-lg border border-amber-800/60 bg-amber-950/50 px-3 py-1.5 text-xs font-mono text-amber-300 hover:bg-amber-900/60 transition-colors cursor-pointer"
                  title="Tự động sinh ngẫu nhiên 1 Tiên Tuyền theo quy tắc phân bổ Đại Vực (Rank >= 5)"
                >
                  <Sparkles className="h-3.5 w-3.5" /> + Tiên Tuyền
                </button>
                <button
                  onClick={openCreateVeinModal}
                  className="flex items-center gap-1.5 rounded-lg bg-cyan-600 px-3 py-1.5 text-xs font-bold font-mono text-zinc-950 hover:bg-cyan-500 transition-colors cursor-pointer"
                >
                  <Plus className="h-3.5 w-3.5" /> Thêm Nguyên Tuyền
                </button>
              </>
            )}
            <button
              onClick={() => void fetchVeins()}
              disabled={veinsLoading}
              className="flex items-center gap-1.5 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${veinsLoading ? "animate-spin" : ""}`} />
            </button>
          </div>
        ) : (
          <div className="flex flex-wrap items-center gap-2">
            {canEdit && (
              <>
                <button
                  onClick={() => void handleSweepWorldPoints()}
                  className="flex items-center gap-1.5 rounded-lg border border-lime-800/60 bg-lime-950/50 px-3 py-1.5 text-xs font-mono text-lime-300 hover:bg-lime-900/60 transition-colors cursor-pointer"
                  title="Kích hoạt worker bù đắp mỏ thiếu hụt trên bản đồ"
                >
                  <Zap className="h-3.5 w-3.5" /> Quét Bù Mỏ (Sweep)
                </button>
                <button
                  onClick={openCreatePointModal}
                  className="flex items-center gap-1.5 rounded-lg bg-lime-600 px-3 py-1.5 text-xs font-bold font-mono text-zinc-950 hover:bg-lime-500 transition-colors cursor-pointer"
                >
                  <Plus className="h-3.5 w-3.5" /> Thêm Mỏ Mới
                </button>
              </>
            )}
            <button
              onClick={() => void fetchPoints()}
              disabled={pointsLoading}
              className="flex items-center gap-1.5 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-1.5 text-xs text-zinc-300 hover:text-white transition-colors cursor-pointer"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${pointsLoading ? "animate-spin" : ""}`} />
            </button>
          </div>
        )}
      </div>

      {/* ========================================================================= */}
      {/* 1. PRIMEVAL VEINS VIEW */}
      {/* ========================================================================= */}
      {subTab === "veins" && (
        <div className="space-y-4">
          {/* Summary Badges */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-3.5">
              <span className="text-[11px] font-mono text-zinc-500 uppercase">Tổng Nguyên Tuyền</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-zinc-100">{veins.length}</span>
              </div>
            </div>
            <div className="rounded-xl border border-cyan-900/40 bg-cyan-950/20 p-3.5">
              <span className="text-[11px] font-mono text-cyan-400 uppercase">Phàm Tuyền</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-cyan-300">{veinStats.mortalCount}</span>
                <span className="text-xs font-mono text-zinc-500">/ {veinStats.mortalCap} trần</span>
              </div>
            </div>
            <div className="rounded-xl border border-amber-900/40 bg-amber-950/20 p-3.5">
              <span className="text-[11px] font-mono text-amber-400 uppercase">Tiên Tuyền</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-amber-300">{veinStats.immortalCount}</span>
                <span className="text-xs font-mono text-zinc-500">/ {veinStats.immortalCap} trần</span>
              </div>
            </div>
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-3.5">
              <span className="text-[11px] font-mono text-zinc-500 uppercase">Trạng Thái Đã Cạn</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-rose-400">
                  {veins.filter((v) => v.is_depleted).length}
                </span>
                <span className="text-xs font-mono text-zinc-500">hết sản lượng</span>
              </div>
            </div>
          </div>

          {/* Filter Bar */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-3 flex flex-wrap items-center gap-3">
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-zinc-500" />
              <input
                type="text"
                placeholder="Tìm theo khu vực, tên vùng hoặc ID..."
                value={veinSearch}
                onChange={(e) => setVeinSearch(e.target.value)}
                className="w-full rounded-lg border border-zinc-800 bg-zinc-950 py-1.5 pl-9 pr-3 text-xs text-zinc-200 placeholder:text-zinc-600 focus:outline-none focus:border-cyan-500"
              />
            </div>
            <select
              value={veinRegionFilter}
              onChange={(e) => setVeinRegionFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả khu vực ({regions.length})</option>
              {regions.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name} ({r.id})
                </option>
              ))}
            </select>
            <select
              value={veinTypeFilter}
              onChange={(e) => setVeinTypeFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả loại (Phàm/Tiên)</option>
              <option value="mortal">Phàm Tuyền</option>
              <option value="immortal">Tiên Tuyền</option>
            </select>
            <select
              value={veinRarityFilter}
              onChange={(e) => setVeinRarityFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả phẩm chất</option>
              {Object.keys(RARITY_COLORS).map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          </div>

          {/* Data Table */}
          {veinsLoading ? (
            <div className="flex items-center justify-center py-20 text-xs font-mono text-zinc-500">
              <RefreshCw className="h-5 w-5 animate-spin mr-2 text-cyan-400" /> Đang tải Nguyên Tuyền...
            </div>
          ) : veins.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-800 p-12 text-center text-xs font-mono text-zinc-500">
              Không tìm thấy Nguyên Tuyền nào phù hợp bộ lọc.
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950">
              <table className="w-full text-left text-xs">
                <thead className="bg-zinc-900/80 text-[11px] font-mono uppercase text-zinc-400 border-b border-zinc-800">
                  <tr>
                    <th className="px-4 py-3">Tọa Độ & Vùng</th>
                    <th className="px-3 py-3">Loại</th>
                    <th className="px-3 py-3">Phẩm Chất</th>
                    <th className="px-3 py-3">Tích Lũy Hiện Tại</th>
                    <th className="px-3 py-3">Sản Lượng Đã Tạo</th>
                    <th className="px-3 py-3">Hạn Tồn Tại</th>
                    <th className="px-4 py-3 text-right">Thao Tác</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/60 font-mono">
                  {veins.map((v) => {
                    const rarityStyle = RARITY_COLORS[v.rarity] || RARITY_COLORS["Thường"];
                    const percentProduced = Math.min(100, Math.round((v.produced_minutes / Math.max(1, v.cap_minutes)) * 100));

                    return (
                      <tr key={v.id} className="hover:bg-zinc-900/40 transition-colors">
                        <td className="px-4 py-3">
                          <div className="font-semibold text-zinc-200">
                            ({v.x}, {v.y})
                          </div>
                          <div className="text-[11px] text-zinc-400 truncate max-w-[200px]" title={v.region_name || v.region_id}>
                            {v.region_name || v.region_id}
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          {v.vein_type === "immortal" ? (
                            <span className="inline-flex items-center gap-1 rounded-md border border-amber-800/60 bg-amber-950/60 px-2 py-0.5 text-[11px] font-bold text-amber-300">
                              <Sparkles className="h-3 w-3" /> Tiên Tuyền
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 rounded-md border border-cyan-800/60 bg-cyan-950/60 px-2 py-0.5 text-[11px] font-bold text-cyan-300">
                              <Droplets className="h-3 w-3" /> Phàm Tuyền
                            </span>
                          )}
                        </td>
                        <td className="px-3 py-3">
                          <span className={`inline-block rounded-md border px-2 py-0.5 text-[11px] font-bold ${rarityStyle.bg} ${rarityStyle.text} ${rarityStyle.border}`}>
                            {v.rarity}
                          </span>
                        </td>
                        <td className="px-3 py-3">
                          <div className="text-zinc-200 font-semibold">
                            +{formatNumber(v.accumulated_stones, 0)} {v.vein_type === "immortal" ? "Tiên Thạch" : "Nguyên Thạch"}
                          </div>
                          <div className="text-[11px] text-zinc-400 flex items-center gap-2">
                            <span>+{formatNumber(v.accumulated_qi, 0)} Nhân Khí</span>
                            {v.accumulated_chests > 0 && <span className="text-amber-400">+{v.accumulated_chests} Rương</span>}
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          <div className="flex items-center gap-2">
                            <div className="w-16 bg-zinc-800 h-1.5 rounded-full overflow-hidden">
                              <div
                                className={`h-full ${v.is_depleted ? "bg-rose-500" : "bg-cyan-500"}`}
                                style={{ width: `${percentProduced}%` }}
                              />
                            </div>
                            <span className={`text-[11px] ${v.is_depleted ? "text-rose-400 font-bold" : "text-zinc-400"}`}>
                              {percentProduced}%
                            </span>
                          </div>
                          <span className="text-[10px] text-zinc-500">
                            {v.produced_minutes} / {v.cap_minutes} phút
                          </span>
                        </td>
                        <td className="px-3 py-3">
                          <div className="text-zinc-300 flex items-center gap-1 text-[11px]">
                            <Clock className="h-3 w-3 text-zinc-500" />
                            {new Date(v.expires_at).toLocaleDateString("vi-VN")}
                          </div>
                          <span className="text-[10px] text-zinc-500">
                            {new Date(v.expires_at) > new Date() ? "Còn hạn" : "Đã hết hạn"}
                          </span>
                        </td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {canEdit && (
                              <button
                                onClick={() => openEditVeinModal(v)}
                                className="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 hover:text-white hover:bg-zinc-800 cursor-pointer"
                                title="Sửa thông số"
                              >
                                <Edit3 className="h-3.5 w-3.5" />
                              </button>
                            )}
                            {canDelete && (
                              <button
                                onClick={() => void handleDeleteVein(v)}
                                className="p-1.5 rounded-lg border border-rose-900/60 bg-rose-950/40 text-rose-300 hover:bg-rose-900/60 cursor-pointer"
                                title="Xóa Nguyên Tuyền"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ========================================================================= */}
      {/* 2. WORLD RESOURCE POINTS VIEW */}
      {/* ========================================================================= */}
      {subTab === "points" && (
        <div className="space-y-4">
          {/* Summary Badges */}
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="rounded-xl border border-zinc-800 bg-zinc-900/40 p-3.5">
              <span className="text-[11px] font-mono text-zinc-500 uppercase">Tổng Mỏ Thế Giới</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-zinc-100">{points.length}</span>
              </div>
            </div>
            <div className="rounded-xl border border-lime-900/40 bg-lime-950/20 p-3.5">
              <span className="text-[11px] font-mono text-lime-400 uppercase">Đang Hoạt Động</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-lime-300">{pointStats.activeCount}</span>
                <span className="text-xs font-mono text-zinc-500">mỏ sản xuất</span>
              </div>
            </div>
            <div className="rounded-xl border border-rose-900/40 bg-rose-950/20 p-3.5">
              <span className="text-[11px] font-mono text-rose-400 uppercase">Cạn Kiệt (Exhausted)</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-rose-300">{pointStats.exhaustedCount}</span>
                <span className="text-xs font-mono text-zinc-500">cần bồi dưỡng</span>
              </div>
            </div>
            <div className="rounded-xl border border-amber-900/40 bg-amber-950/20 p-3.5">
              <span className="text-[11px] font-mono text-amber-400 uppercase">Danh Sơn (Famous)</span>
              <div className="mt-1 flex items-baseline gap-2">
                <span className="text-xl font-bold font-mono text-amber-300">{pointStats.famousCount}</span>
                <span className="text-xs font-mono text-zinc-500">Boss trấn giữ</span>
              </div>
            </div>
          </div>

          {/* Filter Bar */}
          <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-3 flex flex-wrap items-center gap-3">
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-zinc-500" />
              <input
                type="text"
                placeholder="Tìm theo tên mỏ, loại mẫu hoặc khu vực..."
                value={pointSearch}
                onChange={(e) => setPointSearch(e.target.value)}
                className="w-full rounded-lg border border-zinc-800 bg-zinc-950 py-1.5 pl-9 pr-3 text-xs text-zinc-200 placeholder:text-zinc-600 focus:outline-none focus:border-lime-500"
              />
            </div>
            <select
              value={pointRegionFilter}
              onChange={(e) => setPointRegionFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả khu vực ({regions.length})</option>
              {regions.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name} ({r.id})
                </option>
              ))}
            </select>
            <select
              value={pointClassFilter}
              onChange={(e) => setPointClassFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả phân loại</option>
              <option value="NORMAL">Mỏ Thường</option>
              <option value="FAMOUS">Danh Sơn (Boss)</option>
            </select>
            <select
              value={pointGradeFilter}
              onChange={(e) => setPointGradeFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả cấp bậc</option>
              <option value="LOW">Cấp Thấp (LOW)</option>
              <option value="MID">Cấp Trung (MID)</option>
              <option value="TOP">Cấp Cao (TOP)</option>
            </select>
            <select
              value={pointStateFilter}
              onChange={(e) => setPointStateFilter(e.target.value)}
              className="rounded-lg border border-zinc-800 bg-zinc-950 px-3 py-1.5 text-xs text-zinc-300 focus:outline-none cursor-pointer"
            >
              <option value="">Tất cả trạng thái</option>
              <option value="ACTIVE">Hoạt động (ACTIVE)</option>
              <option value="EXHAUSTED">Cạn kiệt (EXHAUSTED)</option>
            </select>
          </div>

          {/* Points Table */}
          {pointsLoading ? (
            <div className="flex items-center justify-center py-20 text-xs font-mono text-zinc-500">
              <RefreshCw className="h-5 w-5 animate-spin mr-2 text-lime-400" /> Đang tải điểm tài nguyên thế giới...
            </div>
          ) : points.length === 0 ? (
            <div className="rounded-xl border border-dashed border-zinc-800 p-12 text-center text-xs font-mono text-zinc-500">
              Không có điểm tài nguyên thế giới nào phù hợp bộ lọc.
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-zinc-800 bg-zinc-950">
              <table className="w-full text-left text-xs">
                <thead className="bg-zinc-900/80 text-[11px] font-mono uppercase text-zinc-400 border-b border-zinc-800">
                  <tr>
                    <th className="px-4 py-3">Tên & Mẫu Mỏ</th>
                    <th className="px-3 py-3">Tọa Độ & Vùng</th>
                    <th className="px-3 py-3">Cấp Bậc & Loại</th>
                    <th className="px-3 py-3">Sinh Khí (Vitality)</th>
                    <th className="px-3 py-3">Kho Tích Lũy</th>
                    <th className="px-3 py-3">Trạng Thái</th>
                    <th className="px-4 py-3 text-right">Thao Tác</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-800/60 font-mono">
                  {points.map((pt) => {
                    const gradeStyle = GRADE_COLORS[pt.grade] || GRADE_COLORS.LOW;
                    const stockKeys = Object.keys(pt.stock || {});

                    return (
                      <tr key={pt.id} className="hover:bg-zinc-900/40 transition-colors">
                        <td className="px-4 py-3">
                          <div className="font-semibold text-zinc-100 flex items-center gap-1.5">
                            {pt.point_class === "FAMOUS" ? (
                              <Mountain className="h-3.5 w-3.5 text-amber-400 shrink-0" />
                            ) : (
                              <Gem className="h-3.5 w-3.5 text-lime-400 shrink-0" />
                            )}
                            {pt.name}
                          </div>
                          <div className="text-[11px] text-zinc-400 flex items-center gap-1.5 mt-0.5">
                            <span>{pt.point_type}</span>
                            {pt.path && <span className="text-zinc-500">· {pt.path}</span>}
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          <div className="font-semibold text-zinc-200">
                            ({pt.x}, {pt.y})
                          </div>
                          <div className="text-[11px] text-zinc-400 truncate max-w-[180px]" title={pt.region_name || pt.region_id}>
                            {pt.region_name || pt.region_id}
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          <div className="flex flex-col gap-1 items-start">
                            <span className={`rounded-md border px-1.5 py-0.5 text-[10px] font-bold ${gradeStyle.bg} ${gradeStyle.text} ${gradeStyle.border}`}>
                              {pt.grade}
                            </span>
                            {pt.point_class === "FAMOUS" && (
                              <span className="rounded-md border border-amber-800/60 bg-amber-950/60 px-1.5 py-0.5 text-[10px] font-bold text-amber-300">
                                Danh Sơn
                              </span>
                            )}
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          <div className="flex items-center gap-2">
                            <div className="w-14 bg-zinc-800 h-1.5 rounded-full overflow-hidden">
                              <div
                                className={`h-full ${pt.vitality > 50 ? "bg-lime-500" : pt.vitality > 20 ? "bg-amber-500" : "bg-rose-500"}`}
                                style={{ width: `${Math.min(100, pt.vitality)}%` }}
                              />
                            </div>
                            <span className="text-[11px] text-zinc-300 font-semibold">{pt.vitality}/100</span>
                          </div>
                        </td>
                        <td className="px-3 py-3">
                          {stockKeys.length === 0 ? (
                            <span className="text-[11px] text-zinc-600">Trống</span>
                          ) : (
                            <div className="space-y-0.5 max-w-[180px]">
                              {stockKeys.slice(0, 2).map((k) => (
                                <div key={k} className="text-[11px] text-zinc-300 truncate">
                                  {pt.stock[k]}x {k}
                                </div>
                              ))}
                              {stockKeys.length > 2 && <span className="text-[10px] text-zinc-500">+{stockKeys.length - 2} loại khác</span>}
                            </div>
                          )}
                        </td>
                        <td className="px-3 py-3">
                          {pt.state === "ACTIVE" ? (
                            <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-lime-400">
                              <span className="h-1.5 w-1.5 rounded-full bg-lime-400 animate-pulse" /> Hoạt động
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-400">
                              <AlertTriangle className="h-3 w-3" /> Cạn kiệt
                            </span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-right">
                          <div className="flex items-center justify-end gap-1.5">
                            {canEdit && (
                              <>
                                <button
                                  onClick={() => void handleRestoreVitality(pt)}
                                  className="p-1.5 rounded-lg border border-lime-900/60 bg-lime-950/40 text-lime-300 hover:bg-lime-900/60 cursor-pointer"
                                  title="Hồi phục 100% sinh khí (Restore Vitality)"
                                >
                                  <Zap className="h-3.5 w-3.5" />
                                </button>
                                <button
                                  onClick={() => openEditPointModal(pt)}
                                  className="p-1.5 rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-300 hover:text-white hover:bg-zinc-800 cursor-pointer"
                                  title="Sửa thông số"
                                >
                                  <Edit3 className="h-3.5 w-3.5" />
                                </button>
                              </>
                            )}
                            {canDelete && (
                              <button
                                onClick={() => void handleDeletePoint(pt)}
                                className="p-1.5 rounded-lg border border-rose-900/60 bg-rose-950/40 text-rose-300 hover:bg-rose-900/60 cursor-pointer"
                                title="Xóa mỏ tài nguyên"
                              >
                                <Trash2 className="h-3.5 w-3.5" />
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {/* ========================================================================= */}
      {/* MODALS */}
      {/* ========================================================================= */}

      {/* 1. Modal Thêm / Sửa Nguyên Tuyền */}
      {(isCreatingVein || editingVein) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm">
          <div className="w-full max-w-lg rounded-2xl border border-zinc-800 bg-zinc-950 p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
              <h3 className="text-base font-serif font-bold text-zinc-100 flex items-center gap-2">
                <Droplets className="h-5 w-5 text-cyan-400" />
                {isCreatingVein ? "Thêm Nguyên Tuyền Mới" : "Sửa Nguyên Tuyền"}
              </h3>
              <button
                onClick={() => {
                  setIsCreatingVein(false);
                  setEditingVein(null);
                }}
                className="text-zinc-500 hover:text-zinc-300"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3 text-xs font-mono">
              <div className="space-y-1">
                <label className="text-zinc-400">Khu vực (Region ID)</label>
                {isCreatingVein ? (
                  <select
                    value={veinForm.region_id}
                    onChange={(e) => setVeinForm((prev) => ({ ...prev, region_id: e.target.value }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                  >
                    {regions.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.name} ({r.id})
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    disabled
                    value={veinForm.region_id}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 p-2 text-zinc-500 cursor-not-allowed"
                  />
                )}
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-zinc-400">Tọa độ X</label>
                  <input
                    type="number"
                    value={veinForm.x}
                    onChange={(e) => setVeinForm((prev) => ({ ...prev, x: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400">Tọa độ Y</label>
                  <input
                    type="number"
                    value={veinForm.y}
                    onChange={(e) => setVeinForm((prev) => ({ ...prev, y: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-zinc-400">Loại Nguyên Tuyền</label>
                  <select
                    value={veinForm.vein_type}
                    onChange={(e) => setVeinForm((prev) => ({ ...prev, vein_type: e.target.value as "mortal" | "immortal" }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                  >
                    <option value="mortal">Phàm Tuyền (Nguyên Thạch)</option>
                    <option value="immortal">Tiên Tuyền (Tiên Nguyên Thạch)</option>
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400">Phẩm chất</label>
                  <select
                    value={veinForm.rarity}
                    onChange={(e) => setVeinForm((prev) => ({ ...prev, rarity: e.target.value }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                  >
                    {Object.keys(RARITY_COLORS).map((r) => (
                      <option key={r} value={r}>
                        {r}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {isCreatingVein ? (
                <div className="space-y-1">
                  <label className="text-zinc-400">Thời hạn tồn tại (Ngày)</label>
                  <input
                    type="number"
                    min="1"
                    max="365"
                    value={veinForm.lifetime_days}
                    onChange={(e) => setVeinForm((prev) => ({ ...prev, lifetime_days: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                  />
                </div>
              ) : (
                <div className="space-y-3 pt-2">
                  <div className="space-y-1">
                    <label className="text-zinc-400">Sản lượng đã tích tụ (Phút)</label>
                    <input
                      type="number"
                      min="0"
                      value={veinForm.produced_minutes}
                      onChange={(e) => setVeinForm((prev) => ({ ...prev, produced_minutes: Number(e.target.value) }))}
                      className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-cyan-500"
                    />
                  </div>
                  <label className="flex items-center gap-2 cursor-pointer text-zinc-300">
                    <input
                      type="checkbox"
                      checked={veinForm.reset_accumulation}
                      onChange={(e) => setVeinForm((prev) => ({ ...prev, reset_accumulation: e.target.checked }))}
                      className="rounded border-zinc-800 bg-zinc-900 text-cyan-500 focus:ring-0"
                    />
                    <span>Đặt lại thời điểm tích tụ (Reset unharvested accumulation)</span>
                  </label>
                </div>
              )}
            </div>

            <div className="flex items-center justify-end gap-2 pt-3 border-t border-zinc-800">
              <button
                onClick={() => {
                  setIsCreatingVein(false);
                  setEditingVein(null);
                }}
                className="rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-2 text-xs font-mono text-zinc-400 hover:text-white"
              >
                Hủy
              </button>
              <button
                onClick={() => void handleSaveVein()}
                disabled={saving}
                className="rounded-lg bg-cyan-600 px-4 py-2 text-xs font-mono font-bold text-zinc-950 hover:bg-cyan-500 disabled:opacity-50"
              >
                {saving ? "Đang lưu..." : isCreatingVein ? "Tạo Nguyên Tuyền" : "Lưu Thay Đổi"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 2. Modal Thêm / Sửa Mỏ Tài Nguyên Thế Giới */}
      {(isCreatingPoint || editingPoint) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4 backdrop-blur-sm">
          <div className="w-full max-w-lg rounded-2xl border border-zinc-800 bg-zinc-950 p-6 shadow-2xl space-y-4">
            <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
              <h3 className="text-base font-serif font-bold text-zinc-100 flex items-center gap-2">
                <Mountain className="h-5 w-5 text-lime-400" />
                {isCreatingPoint ? "Thêm Mỏ Tài Nguyên Thế Giới" : "Sửa Mỏ Tài Nguyên"}
              </h3>
              <button
                onClick={() => {
                  setIsCreatingPoint(false);
                  setEditingPoint(null);
                }}
                className="text-zinc-500 hover:text-zinc-300"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="space-y-3 text-xs font-mono">
              <div className="space-y-1">
                <label className="text-zinc-400">Loại Mẫu Mỏ (Template)</label>
                {isCreatingPoint ? (
                  <select
                    value={pointForm.point_type}
                    onChange={(e) => handleTemplateChange(e.target.value)}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  >
                    {templates.map((t) => (
                      <option key={t.PointType} value={t.PointType}>
                        {t.Name} ({t.PointType}) — {t.Path} [{t.PointClass}]
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    disabled
                    value={`${pointForm.name} (${pointForm.point_type})`}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 p-2 text-zinc-500 cursor-not-allowed"
                  />
                )}
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-zinc-400">Tên hiển thị</label>
                  <input
                    type="text"
                    value={pointForm.name}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, name: e.target.value }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400">Đạo (Path)</label>
                  <input
                    type="text"
                    value={pointForm.path}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, path: e.target.value }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  />
                </div>
              </div>

              <div className="space-y-1">
                <label className="text-zinc-400">Khu vực (Region ID)</label>
                {isCreatingPoint ? (
                  <select
                    value={pointForm.region_id}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, region_id: e.target.value }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  >
                    {regions.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.name} ({r.id})
                      </option>
                    ))}
                  </select>
                ) : (
                  <input
                    type="text"
                    disabled
                    value={pointForm.region_id}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900/50 p-2 text-zinc-500 cursor-not-allowed"
                  />
                )}
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-zinc-400">Tọa độ X</label>
                  <input
                    type="number"
                    value={pointForm.x}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, x: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400">Tọa độ Y</label>
                  <input
                    type="number"
                    value={pointForm.y}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, y: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-zinc-400">Cấp bậc (Grade)</label>
                  <select
                    value={pointForm.grade}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, grade: e.target.value as "LOW" | "MID" | "TOP" }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  >
                    <option value="LOW">Cấp Thấp (LOW)</option>
                    <option value="MID">Cấp Trung (MID)</option>
                    <option value="TOP">Cấp Cao (TOP)</option>
                  </select>
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400">Phân loại (Class)</label>
                  <select
                    value={pointForm.point_class}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, point_class: e.target.value as "NORMAL" | "FAMOUS" }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  >
                    <option value="NORMAL">Mỏ Thường (NORMAL)</option>
                    <option value="FAMOUS">Danh Sơn / Boss (FAMOUS)</option>
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <label className="text-zinc-400">Sinh khí (Vitality: 0-100)</label>
                  <input
                    type="number"
                    min="0"
                    max="100"
                    value={pointForm.vitality}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, vitality: Number(e.target.value) }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-zinc-400">Trạng thái (State)</label>
                  <select
                    value={pointForm.state}
                    onChange={(e) => setPointForm((prev) => ({ ...prev, state: e.target.value }))}
                    className="w-full rounded-lg border border-zinc-800 bg-zinc-900 p-2 text-zinc-200 focus:outline-none focus:border-lime-500"
                  >
                    <option value="ACTIVE">Hoạt động (ACTIVE)</option>
                    <option value="EXHAUSTED">Cạn kiệt (EXHAUSTED)</option>
                  </select>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end gap-2 pt-3 border-t border-zinc-800">
              <button
                onClick={() => {
                  setIsCreatingPoint(false);
                  setEditingPoint(null);
                }}
                className="rounded-lg border border-zinc-800 bg-zinc-900 px-4 py-2 text-xs font-mono text-zinc-400 hover:text-white"
              >
                Hủy
              </button>
              <button
                onClick={() => void handleSavePoint()}
                disabled={saving}
                className="rounded-lg bg-lime-600 px-4 py-2 text-xs font-mono font-bold text-zinc-950 hover:bg-lime-500 disabled:opacity-50"
              >
                {saving ? "Đang lưu..." : isCreatingPoint ? "Tạo Mỏ Tài Nguyên" : "Lưu Thay Đổi"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
