"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Compass, Sparkles, DoorOpen, Landmark, ShieldAlert, Map as MapIcon, Package, Eye, Mountain, Waypoints, Store } from "lucide-react";
import { useGameStore, Player } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";
import { usePolling } from "@/lib/usePolling";
import { getWorldResourceGroup, WORLD_RESOURCE_ASSETS } from "@/lib/worldResourceAssets";
import { WorldAtlasModal } from "@/components/WorldAtlasModal";
import { TeleportBuildModal } from "@/components/Teleport/TeleportBuildModal";
import { TeleportDirectoryModal } from "@/components/Teleport/TeleportDirectoryModal";
import { TeleportManagementModal } from "@/components/Teleport/TeleportManagementModal";
import type { TeleportArray } from "@/store/slices/teleportSlice";
import { useUIStore } from "@/store/useUIStore";
import { MonsterSprite } from "@/components/MonsterSprite";

interface ExplorationMapProps {
  player: Player;
}

interface GridCell {
  x: number;
  y: number;
  cell_type: string;
  content: string;
}

interface GridData {
  region_id: string;
  x: number;
  y: number;
  vision_radius: number;
  max_vision_radius?: number;
  cells: GridCell[];
  stats: { movement?: number; investigation?: number };
  players_on_map?: MapPlayer[];
  ground_drops?: { id: string; x: number; y: number; loser_name?: string }[];
  world_width?: number;
  world_height?: number;
}

interface MapPlayer {
  id: string;
  username: string;
  avatar_url: string;
  x: number;
  y: number;
}

const GREAT_REGION_BACKGROUND_PANEL: Record<string, number> = {
  dai_vuc_nam_cuong: 0,
  dai_vuc_trung_chau: 1,
  dai_vuc_bac_nguyen: 2,
  dai_vuc_tay_mac: 3,
  dai_vuc_dong_hai: 4,
  dai_vuc_hac_thien: 5,
  dai_vuc_bach_thien: 6,
};

export const ExplorationMap: React.FC<ExplorationMapProps> = ({ player }) => {
  const regions = useGameStore((state) => state.regions);
  const regionsLoaded = useGameStore((state) => state.regionsLoaded);
  const fetchRegions = useGameStore((state) => state.fetchRegions);
  const fetchGrid = useGameStore((state) => state.fetchGrid);
  const movePlayer = useGameStore((state) => state.movePlayer);
  const enterAperturePortal = useGameStore((state) => state.enterPortal);
  const harvestVein = useWorldStore((state) => state.harvestVein);
  const mapMarkers = useWorldStore((state) => state.zoneMarkers);
  const mapMarkersRegion = useWorldStore((state) => state.zoneMarkersRegion);
  const fetchZoneMarkers = useWorldStore((state) => state.fetchZoneMarkers);
  const resourcePoints = useWorldStore((state) => state.resourcePoints);
  const resourcePointsRegion = useWorldStore((state) => state.resourcePointsRegion);
  const fetchWorldResourcePoints = useWorldStore((state) => state.fetchWorldResourcePoints);
  const worldMonsters = useWorldStore((state) => state.worldMonsters);
  const fetchWorldMonsters = useWorldStore((state) => state.fetchWorldMonsters);
  const engageAscensionWorldMonster = useWorldStore((state) => state.engageAscensionWorldMonster);
  const collectWorldResourcePoint = useWorldStore((state) => state.collectWorldResourcePoint);
  const lootGroundDrop = useGameStore((state) => state.lootGroundDrop);
  const initiateTrade = useGameStore((state) => state.initiateTrade);
  const addToast = useGameStore((state) => state.addToast);
  const actionLoading = useGameStore((state) => state.actionLoading);
  const party = useGameStore((state) => state.party);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const itemDictionary = useGameStore((state) => state.itemDictionary);
  const monsterDictionary = useGameStore((state) => state.monsterDictionary);
  const myTeleportArrays = useGameStore((state) => state.myTeleportArrays);
  const fetchMyTeleportArrays = useGameStore((state) => state.fetchMyTeleportArrays);

  const [gridSnapshot, setGridSnapshot] = useState<GridData | null>(null);
  const gridRequestRevision = useRef(0);
  const [preferredRadius, setPreferredRadius] = useState<number>(() => {
    if (typeof window !== "undefined") {
      const saved = localStorage.getItem("gu_explore_vision_radius");
      if (saved) {
        const val = parseInt(saved, 10);
        if (val >= 2) return val;
      }
    }
    return 2; // Mặc định 5x5
  });
  const [lastLog, setLastLog] = useState<string | null>(null);
  const [selectedPvpTarget, setSelectedPvpTarget] = useState<MapPlayer | null>(null);
  const [autoFarmRunning, setAutoFarmRunning] = useState(false);
  const autoFarmRunningRef = useRef(false);
  const [showAtlas, setShowAtlas] = useState(false);
  const [showBuildTeleport, setShowBuildTeleport] = useState(false);
  const [showDirectoryModal, setShowDirectoryModal] = useState(false);
  const [showManageModal, setShowManageModal] = useState(false);
  const [selectedManageArray, setSelectedManageArray] = useState<TeleportArray | null>(null);
  const lastMoveTimeRef = useRef(0);
  const setView = useUIStore((state) => state.setView);
  const gridData = gridSnapshot?.region_id === player.current_region
    && gridSnapshot.x === player.x
    && gridSnapshot.y === player.y
    ? gridSnapshot
    : null;

  useEffect(() => () => {
    autoFarmRunningRef.current = false;
  }, []);

  const standingOnTeleport = useMemo(() => {
    return mapMarkers.find(
      (m) => m.kind === "teleport" && m.x === player.x && m.y === player.y
    );
  }, [mapMarkers, player.x, player.y]);

  const handleOpenManage = useCallback((arrayId: string) => {
    const myArr = myTeleportArrays.find((a) => a.id === arrayId);
    if (myArr) {
      setSelectedManageArray(myArr);
      setShowManageModal(true);
    } else {
      void fetchMyTeleportArrays().then(() => {
        const found = useGameStore.getState().myTeleportArrays.find((a) => a.id === arrayId);
        if (found) {
          setSelectedManageArray(found);
          setShowManageModal(true);
        }
      });
    }
  }, [myTeleportArrays, fetchMyTeleportArrays]);

  useEffect(() => {
    if (!regionsLoaded) fetchRegions();
  }, [regionsLoaded, fetchRegions]);

  const loadGrid = useCallback(async () => {
    const requestRevision = ++gridRequestRevision.current;
    const requestedPlayer = useGameStore.getState().player;
    if (!requestedPlayer) return null;
    const data = await fetchGrid(preferredRadius);
    const currentPlayer = useGameStore.getState().player;
    if (!data || !Array.isArray(data.cells) || !currentPlayer
      || currentPlayer.id !== requestedPlayer.id
      || currentPlayer.current_region !== requestedPlayer.current_region
      || currentPlayer.x !== requestedPlayer.x
      || currentPlayer.y !== requestedPlayer.y
      || data.x !== requestedPlayer.x
      || data.y !== requestedPlayer.y) return null;

    const snapshot = { ...data, region_id: requestedPlayer.current_region } as GridData;
    if (requestRevision === gridRequestRevision.current) setGridSnapshot(snapshot);
    return snapshot;
  }, [fetchGrid, preferredRadius]);

  const handleRadiusChange = (newRadius: number) => {
    setPreferredRadius(newRadius);
    if (typeof window !== "undefined") {
      localStorage.setItem("gu_explore_vision_radius", String(newRadius));
    }
  };

  useEffect(() => {
    void loadGrid();
  }, [player.current_region, player.x, player.y, loadGrid]);
  usePolling(() => loadGrid().then(() => undefined), 10000);
  useEffect(() => { void fetchWorldMonsters(); }, [player.current_region, player.x, player.y, fetchWorldMonsters]);
  usePolling(() => fetchWorldMonsters().then(() => undefined), 5000);

  useEffect(() => {
    if (mapMarkersRegion !== player.current_region) void fetchZoneMarkers(player.current_region);
    if (resourcePointsRegion !== player.current_region) void fetchWorldResourcePoints(player.current_region);
  }, [player.current_region, mapMarkersRegion, resourcePointsRegion, fetchZoneMarkers, fetchWorldResourcePoints]);

  const isMemberNotLeader = Boolean(player.party_id) && (party === null || party.leader_id !== player.id);
  const currentRegion = regions.find((r) => r.id === player.current_region);
  const regionName = currentRegion?.name || "Thế Giới Cổ Sư";
  const greatRegion = useMemo(() => {
    let region = currentRegion;
    const visited = new Set<string>();
    while (region && !visited.has(region.id)) {
      visited.add(region.id);
      if (region.zone_type === "NGU_VUC") return region;
      region = region.parent_id ? regions.find((item) => item.id === region?.parent_id) : undefined;
    }
    return undefined;
  }, [currentRegion, regions]);
  const backgroundPanel = greatRegion ? GREAT_REGION_BACKGROUND_PANEL[greatRegion.id] : undefined;
  const mapBackgroundStyle: React.CSSProperties = {
    backgroundColor: "#111812",
    backgroundImage: `linear-gradient(180deg, rgba(7, 12, 12, .48), rgba(7, 10, 12, .68)), url('/assets/world/${backgroundPanel === undefined ? "ground-moss.webp" : "great-regions-atlas.png"}')`,
    backgroundSize: backgroundPanel === undefined ? "auto, cover" : "auto, 700% 100%",
    backgroundPosition: backgroundPanel === undefined ? "center, center" : `center, ${(backgroundPanel / 6) * 100}% center`,
    backgroundRepeat: "no-repeat",
  };

  const handleCellClick = async (x: number, y: number, isMovable: boolean, cell?: GridCell, marker?: { kind: string; id: string; isMine?: boolean; name?: string }, resourcePointId?: string) => {
    if (autoFarmRunningRef.current) return;
    const playerOnCell = gridData?.players_on_map?.find((target) => target.x === x && target.y === y && target.id !== player.id);
    if (playerOnCell) {
      setSelectedPvpTarget(playerOnCell);
      return;
    }
    if (marker?.kind === "merchant") {
      const isNovice = marker.id.startsWith("merchant_tan_thu_") || Boolean(marker.name?.includes("Tân Thủ"));
      const maxDist = isNovice ? 15 : 2;
      if (Math.hypot(x - player.x, y - player.y) <= maxDist) {
        void useGameStore.getState().fetchMerchants();
        useUIStore.getState().setView("shop");
        return;
      }
    }
    if (marker?.kind === "aperture_portal" && !marker.isMine && Math.hypot(x - player.x, y - player.y) <= 3) {
      await enterAperturePortal(marker.id);
      await loadGrid();
      return;
    }
    if (marker?.kind === "teleport") {
      if (player.x === x && player.y === y) {
        if (marker.isMine) {
          handleOpenManage(marker.id);
          return;
        }
        setShowDirectoryModal(true);
        return;
      }
      if (!isMovable || actionLoading || player.active_combat) return;
      if (isMemberNotLeader) {
        addToast("Chỉ có Trưởng nhóm mới có quyền di chuyển trên bản đồ!", "error");
        return;
      }
      const now = Date.now();
      if (now - lastMoveTimeRef.current < 600) return;
      lastMoveTimeRef.current = now;

      setLastLog(null);
      const res = await movePlayer(x, y);
      if (res) {
        setLastLog(res.log);
        await loadGrid();
        if (res.player?.x === x && res.player?.y === y) {
          setShowDirectoryModal(true);
        }
      }
      return;
    }
    if (resourcePointId && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 6) {
      if (await collectWorldResourcePoint(resourcePointId)) await fetchWorldResourcePoints(player.current_region);
      return;
    }
    if (cell?.cell_type === "PrimevalVein"
      && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 3) {
      await harvestVein(cell.content);
      await loadGrid();
      return;
    }

    // Check ground drops on cell
    const dropOnCell = gridData?.ground_drops?.find((d) => d.x === x && d.y === y);
    if (dropOnCell && Math.max(Math.abs(x - player.x), Math.abs(y - player.y)) <= 1) {
      const res = await lootGroundDrop(dropOnCell.id);
      if (res?.success) {
        await loadGrid();
        return;
      }
    }

    if (!isMovable || actionLoading || player.active_combat) return;
    if (isMemberNotLeader) {
      addToast("Chỉ có Trưởng nhóm mới có quyền di chuyển trên bản đồ!", "error");
      return;
    }

    const now = Date.now();
    if (now - lastMoveTimeRef.current < 600) return;
    lastMoveTimeRef.current = now;

    setLastLog(null);
    const res = await movePlayer(x, y);
    if (res) {
      setLastLog(res.log);
      await loadGrid();
    }
  };

  const stopAutoFarm = () => {
    autoFarmRunningRef.current = false;
    setAutoFarmRunning(false);
  };

  const startAutoFarm = async () => {
    if (autoFarmRunningRef.current) return;
    if (isMemberNotLeader || player.active_combat) {
      addToast("Chỉ Trưởng nhóm hoặc người chơi solo có thể bắt đầu auto farm khi không chiến đấu.", "error");
      return;
    }

    const store = useGameStore.getState();
    if (!store.combatSettings) await store.fetchCombatSettings();
    if (!useGameStore.getState().combatSettings?.skip_battle) {
      addToast("Hãy bật ‘Tự động xử lý chiến đấu khi farm’ trong Lobby → Cài Đặt trước.", "error");
      return;
    }

    const origin = { x: player.x, y: player.y };
    const region = player.current_region;
    const bounds = regions.find((item) => item.id === region);
    if (bounds && ["HUB", "SAFE_ZONE"].includes(bounds.zone_type || "")) {
      addToast("Auto farm chỉ hoạt động ngoài vùng an toàn.", "error");
      return;
    }
    let latestGrid = await loadGrid();
    if (!latestGrid || !Array.isArray(latestGrid.cells)) {
      addToast("Không thể tải bản đồ để bắt đầu auto farm.", "error");
      return;
    }
    const patrolOffsets = [
      [-2, -2], [-1, -2], [0, -2], [1, -2], [2, -2],
      [2, -1], [2, 0], [2, 1], [2, 2], [1, 2],
      [0, 2], [-1, 2], [-2, 2], [-2, 1], [-2, 0], [-2, -1],
    ];
    const worldWidth = latestGrid.world_width || 100000;
    const worldHeight = latestGrid.world_height || 100000;
    const patrolPoints = patrolOffsets
      .map(([dx, dy]) => ({ x: origin.x + dx, y: origin.y + dy }))
      .filter((point) => point.x >= 0 && point.y >= 0 && point.x < worldWidth && point.y < worldHeight)
      .filter((point) => !bounds || (
        point.x >= (bounds.min_x ?? 0) && point.x <= (bounds.max_x ?? worldWidth - 1) &&
        point.y >= (bounds.min_y ?? 0) && point.y <= (bounds.max_y ?? worldHeight - 1)
      ));
    if (patrolPoints.length === 0) {
      addToast("Không có ô tuần tra hợp lệ quanh vị trí hiện tại.", "error");
      return;
    }
    let patrolIndex = 0;
    autoFarmRunningRef.current = true;
    setAutoFarmRunning(true);
    setGridSnapshot(latestGrid);
    addToast("Auto farm đã bắt đầu: sẽ tìm quái trong tầm nhìn, nếu chưa có thì tuần tra quanh vị trí hiện tại.", "info");

    while (autoFarmRunningRef.current) {
      const before = useGameStore.getState().player as Player | null;
      if (!before || before.current_region !== region || before.active_combat) break;
      const visibleMonsters = latestGrid.cells
        .filter((cell) => (cell.cell_type === "Monster" || cell.cell_type === "Boss")
          && Math.max(Math.abs(cell.x - origin.x), Math.abs(cell.y - origin.y)) <= 2
          && !(cell.x === before.x && cell.y === before.y))
        .sort((left, right) => Math.max(Math.abs(left.x - before.x), Math.abs(left.y - before.y)) - Math.max(Math.abs(right.x - before.x), Math.abs(right.y - before.y)));
      let destination = visibleMonsters[0] ? { x: visibleMonsters[0].x, y: visibleMonsters[0].y } : patrolPoints[patrolIndex];
      if (!visibleMonsters[0]) patrolIndex = (patrolIndex + 1) % patrolPoints.length;
      if (destination.x === before.x && destination.y === before.y) {
        patrolIndex = (patrolIndex + 1) % patrolPoints.length;
        destination = patrolPoints[patrolIndex];
      }

      const result = await useGameStore.getState().movePlayer(destination.x, destination.y);
      if (!autoFarmRunningRef.current) break;
      const after = useGameStore.getState().player as Player | null;
      if (!result || !after || after.active_combat) break;
      if (after.current_region !== region) {
        addToast("Auto farm đã dừng vì nhân vật chuyển sang khu vực khác.", "info");
        break;
      }
      if (after.x === before.x && after.y === before.y) {
        addToast("Auto farm đã dừng vì không thể tiếp tục di chuyển trên tuyến này.", "error");
        break;
      }
      const refreshedGrid = await loadGrid();
      if (refreshedGrid) {
        latestGrid = refreshedGrid;
        setGridSnapshot(latestGrid);
      } else {
        addToast("Auto farm đã dừng vì không tải lại được bản đồ.", "error");
        break;
      }
      await new Promise<void>((resolve) => window.setTimeout(resolve, 800));
    }

    autoFarmRunningRef.current = false;
    setAutoFarmRunning(false);
  };

  const handleChallengePlayer = async () => {
    if (!selectedPvpTarget || actionLoading || player.active_combat || isMemberNotLeader) return;
    const now = Date.now();
    if (now - lastMoveTimeRef.current < 600) return;
    lastMoveTimeRef.current = now;
    const target = selectedPvpTarget;
    setLastLog(null);
    const result = await movePlayer(target.x, target.y, target.id);
    if (result) {
      setLastLog(result.log);
      await loadGrid();
      const movedPlayer = useGameStore.getState().player;
      if (result.combat_triggered || !movedPlayer || Math.max(Math.abs(target.x - movedPlayer.x), Math.abs(target.y - movedPlayer.y)) <= 2) {
        setSelectedPvpTarget(null);
      }
    }
  };

  const handleMapTrade = async () => {
    if (!selectedPvpTarget || actionLoading || player.active_combat || isMemberNotLeader) return;
    const distance = Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y));
    if (distance > 10) return;
    if (await initiateTrade(selectedPvpTarget.username, "map")) setSelectedPvpTarget(null);
  };

  const getEntityName = (type: string, content: string) => {
    if (type === "EXIT_PORTAL" || type === "PORTAL_EXIT") return "Phúc Địa Thoát Ly Trận";
    if (type === "ENTRY_PORTAL") return `Phúc Địa Giới Môn Trận: ${content.replace(/^spatial_realm_/, "").replace(/_/g, " ")}`;
    if (type === "INHERITANCE_ALTAR") return `Đài Truyền Thừa: ${content.replace(/^altar_/, "").replace(/_/g, " ")}`;
    if (type === "Monster" || type === "Boss") return monsterDictionary[content]?.name || content.split(" ")[0];
    if (type === "Resource") return itemDictionary[content]?.name || content.replace(/_/g, " ").replace(/\b\w/g, (c: string) => c.toUpperCase());
    if (type === "WildGu") return guDictionary[content]?.name || content;
    if (type === "PrimevalVein") return "Nguyên Tuyền";
    return content;
  };

  const getCellIcon = (type: string, content: string) => {
    switch (type) {
      case "Resource": {
        const item = itemDictionary[content];
        const group = getWorldResourceGroup(content, item?.item_type || item?.type);
        return <img src={WORLD_RESOURCE_ASSETS[group]} alt="" className="h-6 w-6 object-contain" />;
      }
      case "Monster":
      case "Boss":
        return (
          <MonsterSprite
            templateId={content}
            alt={getEntityName(type, content)}
            className={`${type === "Boss" ? "w-[84%] h-[84%]" : "w-[72%] h-[72%]"} object-contain drop-shadow-[0_0_8px_rgba(251,146,60,0.45)]`}
          />
        );
      case "WildGu":
        return <Sparkles className="w-5 h-5 text-amber-400 drop-shadow-[0_0_8px_rgba(251,191,36,0.5)] animate-spin" />;
      case "PrimevalVein":
        return <Landmark className="w-5 h-5 text-cyan-400 drop-shadow-[0_0_8px_rgba(34,211,238,0.5)]" />;
      case "ENTRY_PORTAL":
      case "EXIT_PORTAL":
      case "PORTAL_EXIT":
        return <DoorOpen className="w-5 h-5 text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.5)]" />;
      case "INHERITANCE_ALTAR":
        return <Landmark className="w-5 h-5 text-purple-400 drop-shadow-[0_0_8px_rgba(192,132,252,0.5)]" />;
      default:
        return null;
    }
  };

  const maxAllowedRadius = Math.max(2, gridData?.max_vision_radius || 2);
  const radius = Math.min(maxAllowedRadius, Math.max(2, preferredRadius));
  const diameter = radius * 2 + 1;
  const maxContainerWidth = radius <= 2 ? "max-w-[560px]" : radius === 3 ? "max-w-[660px]" : "max-w-[760px]";

  const cellMap = useMemo(() => {
    const map = new Map<string, GridCell>();
    if (gridData?.cells) {
      for (const cell of gridData.cells) {
        map.set(`${cell.x},${cell.y}`, cell);
      }
    }
    return map;
  }, [gridData?.cells]);

  const markersByCell = useMemo(() => {
    const map = new Map<string, { kind: string; id: string; name: string; isMine?: boolean }[]>();
    if (mapMarkersRegion === player.current_region) {
      for (const marker of mapMarkers) {
        const key = `${marker.x},${marker.y}`;
        map.set(key, [...(map.get(key) || []), {
          kind: marker.kind,
          id: marker.id,
          name: marker.name,
          isMine: marker.detail?.is_mine === true || marker.detail?.owner_id === player.id || marker.detail?.OwnerID === player.id,
        }]);
      }
    }
    if (resourcePointsRegion === player.current_region) {
      for (const point of resourcePoints) {
        if (point.region_id && point.region_id !== player.current_region) continue;
        if (point.state === "GONE") continue;
        const key = `${Math.round(point.x)},${Math.round(point.y)}`;
        map.set(key, [...(map.get(key) || []), { kind: "resource_point", id: point.id, name: point.name }]);
      }
    }
    return map;
  }, [mapMarkers, mapMarkersRegion, player.current_region, player.id, resourcePoints, resourcePointsRegion]);

  const gridArray = useMemo(() => {
    const arr = [];
    for (let dy = -radius; dy <= radius; dy++) {
      for (let dx = -radius; dx <= radius; dx++) {
        const x = player.x + dx;
        const y = player.y + dy;
        const isOutOfBounds =
          x < 0 || y < 0 || (gridData?.world_width && x >= gridData.world_width) || (gridData?.world_height && y >= gridData.world_height);
        arr.push({ x, y, isOutOfBounds });
      }
    }
    return arr;
  }, [player.x, player.y, radius, gridData?.world_width, gridData?.world_height]);

  return (
    <div className="w-full flex flex-col gap-4 max-w-5xl mx-auto p-2 sm:p-4">
      {/* Header Bar */}
      <div className="flex flex-wrap items-center justify-between gap-3 bg-zinc-950/80 border border-zinc-800 rounded-2xl p-4 shadow-xl backdrop-blur-md">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
            <Compass className="w-6 h-6 animate-spin-slow" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-xs uppercase font-mono tracking-widest text-emerald-500 font-semibold">Khu vực hiện tại</span>
              <span className="text-xs text-zinc-500">·</span>
              <span className="text-xs font-mono text-zinc-400">({player.x}, {player.y})</span>
            </div>
            <h2 className="text-lg font-serif font-bold text-zinc-100">{regionName}</h2>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2.5 sm:gap-3">
          {/* Vision Range Selector */}
          <div className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-xl bg-zinc-900/90 border border-zinc-800 text-xs shadow-md">
            <Eye className="w-4 h-4 text-emerald-400 shrink-0" />
            <span className="text-zinc-400 font-serif hidden sm:inline">Tầm nhìn:</span>
            <select
              aria-label="Chọn kích thước lưới bản đồ"
              value={radius}
              onChange={(e) => handleRadiusChange(Number(e.target.value))}
              className="bg-zinc-950 border border-zinc-700/60 rounded-lg px-2 py-1 text-emerald-400 font-mono text-xs font-semibold focus:outline-none cursor-pointer"
            >
              <option value={2}>5×5 (Ô to chuẩn)</option>
              {maxAllowedRadius >= 3 && <option value={3}>7×7 (Mở rộng)</option>}
              {maxAllowedRadius >= 4 && <option value={4}>9×9 (Xa)</option>}
              {maxAllowedRadius >= 5 && <option value={5}>11×11 (Toàn cảnh)</option>}
            </select>
          </div>

          <button
            onClick={() => setShowAtlas(true)}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400 hover:bg-amber-500/20 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] font-serif text-xs font-semibold shadow-lg shadow-amber-950/20 cursor-pointer"
          >
            <MapIcon className="w-3.5 h-3.5" />
            <span>Đại Vực</span>
          </button>
          <button
            onClick={() => setShowBuildTeleport(true)}
            className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-sky-500/10 border border-sky-500/30 text-sky-400 hover:bg-sky-500/20 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] font-serif text-xs font-semibold shadow-lg shadow-sky-950/20 cursor-pointer"
            title="Bố trí Truyền Tống Trận (Trận Đạo)"
          >
            <Waypoints className="w-3.5 h-3.5 text-sky-300" />
            <span>Bố Trận</span>
          </button>
          {standingOnTeleport && (
            <button
              onClick={() => setShowDirectoryModal(true)}
              className="flex items-center gap-2 px-3.5 py-1.5 rounded-xl bg-emerald-500/15 border border-emerald-500/40 text-emerald-300 hover:bg-emerald-500/25 transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] font-serif text-xs font-semibold shadow-lg shadow-emerald-950/20 cursor-pointer animate-pulse"
              title="Đang đứng trên Truyền Tống Trận - Nhấp để dịch chuyển"
            >
              <Sparkles className="w-3.5 h-3.5 text-emerald-300" />
              <span>Dịch Chuyển</span>
            </button>
          )}
        </div>
      </div>

      <section className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-emerald-800/60 bg-emerald-950/20 p-3 shadow-lg">
        <div className="min-w-0">
          <p className="text-xs font-bold text-emerald-200">Auto farm quanh vị trí hiện tại</p>
          <p className="mt-1 text-[10px] text-zinc-400">
            {autoFarmRunning
              ? "Đang tìm quái trong tầm; nếu chưa có sẽ tuần tra trong bán kính 2 ô. Lệnh hiện tại hoàn tất trước khi dừng."
              : "Tự nhắm quái thường đang thấy; khi không có mục tiêu sẽ tuần tra vòng quanh bán kính 2 ô."}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {autoFarmRunning ? (
            <button type="button" onClick={stopAutoFarm} className="min-h-10 rounded-lg border border-rose-500/40 bg-rose-950/40 px-3 text-xs font-semibold text-rose-200 hover:bg-rose-900/50">Dừng auto farm</button>
          ) : (
            <button type="button" onClick={() => void startAutoFarm()} disabled={isMemberNotLeader || actionLoading || Boolean(player.active_combat)} className="min-h-10 rounded-lg border border-emerald-500/50 bg-emerald-700/70 px-3 text-xs font-bold text-white hover:bg-emerald-600 disabled:opacity-50">Bắt đầu auto farm</button>
          )}
        </div>
      </section>

      {worldMonsters.filter((actor) => actor.owner_player_id && actor.monster_template_id === "monster_earth_calamity_boss").map((actor) => (
        <section key={actor.instance_id} className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-rose-700/50 bg-rose-950/20 p-3 shadow-lg">
          <div className="flex items-center gap-2">
            <MonsterSprite templateId={actor.monster_template_id} alt="Địa Tai Bản Yếu" className="h-10 w-10 object-contain" />
            <div>
              <div className="text-xs font-bold text-rose-200">Địa Tai Bản Yếu · Chuyển 5.4</div>
              <div className="text-[10px] text-zinc-400">Thực thể ngoài grid · ({Math.round(actor.x)}, {Math.round(actor.y)}) · HP {Math.round(actor.hp)}/{Math.round(actor.max_hp)}</div>
              {actor.owner_player_id !== player.id && <div className="text-[10px] text-amber-300">Người độ kiếp cần nghênh chiến; thành viên tổ đội cùng khu vực sẽ tham chiến cùng.</div>}
            </div>
          </div>
          {actor.owner_player_id === player.id && (
            <button disabled={actionLoading || Boolean(player.active_combat)} onClick={() => void engageAscensionWorldMonster(actor)} className="rounded-lg bg-rose-700 px-4 py-2 text-xs font-bold text-white hover:bg-rose-600 disabled:opacity-50">Nghênh chiến</button>
          )}
        </section>
      ))}

      {/* Exploration Grid Canvas */}
      {selectedPvpTarget && (
        <section className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-purple-700/50 bg-purple-950/30 p-4">
          <div>
            <p className="font-bold text-purple-200">Cổ sư: {selectedPvpTarget.username}</p>
            <p className="text-xs text-zinc-400">({selectedPvpTarget.x}, {selectedPvpTarget.y}) · PvP cần trong tầm đánh; giao dịch trên map cần cách tối đa 10 ô.</p>
          </div>
          <div className="flex gap-2">
            <button onClick={() => void handleChallengePlayer()} disabled={actionLoading || Boolean(player.active_combat) || isMemberNotLeader} className="rounded-lg border border-purple-500/60 bg-purple-700/30 px-4 py-2 text-sm font-bold text-purple-100 hover:bg-purple-700/50 disabled:opacity-50">{Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y)) <= 2 ? "Khai chiến" : "Tiến gần mục tiêu"}</button>
            <button onClick={() => void handleMapTrade()} disabled={actionLoading || Boolean(player.active_combat) || isMemberNotLeader || Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y)) > 10} className="rounded-lg border border-emerald-500/50 bg-emerald-700/30 px-4 py-2 text-sm font-bold text-emerald-100 hover:bg-emerald-700/50 disabled:opacity-50">Giao dịch tại chỗ</button>
            <button onClick={() => setSelectedPvpTarget(null)} className="rounded-lg border border-zinc-700 px-3 py-2 text-sm text-zinc-300 hover:bg-zinc-800">Đóng</button>
          </div>
        </section>
      )}
      <div className="relative border border-zinc-900 rounded-3xl p-2 sm:p-3 shadow-2xl overflow-hidden flex flex-col items-center justify-center" style={mapBackgroundStyle}>
        {gridData ? (
          <div
            className={`grid gap-px w-full ${maxContainerWidth} aspect-square transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] duration-300`}
            style={{
              gridTemplateColumns: `repeat(${diameter}, minmax(0, 1fr))`,
              gridTemplateRows: `repeat(${diameter}, minmax(0, 1fr))`,
            }}
          >
            {gridArray.map(({ x, y, isOutOfBounds }) => {
              const cellData = cellMap.get(`${x},${y}`) || { x, y, cell_type: "Empty", content: "" };
              const isPlayerCell = x === player.x && y === player.y;
              const isMovable = !isOutOfBounds && !isPlayerCell && !isMemberNotLeader;
              const otherPlayersOnCell = isOutOfBounds
                ? []
                : gridData.players_on_map?.filter((p) => p.x === x && p.y === y) || [];
              const dropsOnCell = isOutOfBounds ? [] : gridData.ground_drops?.filter((d) => d.x === x && d.y === y) || [];
              const dropOnCell = dropsOnCell[0] || null;
              const markersOnCell = isOutOfBounds ? [] : markersByCell.get(`${x},${y}`) || [];
              const primaryMarker = markersOnCell.find((marker) => marker.kind === "aperture_portal")
                || markersOnCell.find((marker) => marker.kind === "teleport")
                || markersOnCell.find((marker) => marker.kind === "sect")
                || markersOnCell.find((marker) => marker.kind === "merchant")
                || markersOnCell[0];
              const cellTint = cellData.cell_type === "ENTRY_PORTAL" || cellData.cell_type === "EXIT_PORTAL" || cellData.cell_type === "PORTAL_EXIT"
                ? "bg-emerald-950/60 border-emerald-500/50"
                : cellData.cell_type === "INHERITANCE_ALTAR"
                  ? "bg-purple-950/60 border-purple-500/50"
                  : cellData.cell_type === "Boss" || cellData.cell_type === "Monster"
                    ? "bg-rose-950/45 border-rose-500/40"
                    : cellData.cell_type === "Resource" || cellData.cell_type === "PrimevalVein"
                      ? "bg-cyan-950/45 border-cyan-500/40"
                      : primaryMarker?.kind === "sect"
                        ? "bg-amber-950/45 border-amber-500/40"
                        : primaryMarker?.kind === "teleport"
                          ? "bg-sky-950/45 border-sky-500/40"
                          : primaryMarker?.kind === "aperture_portal"
                            ? "bg-violet-950/45 border-violet-500/40"
                            : dropOnCell
                              ? "bg-amber-950/35 border-amber-500/35"
                              : "";
              const portalMarker = markersOnCell.find((marker) => marker.kind === "aperture_portal");
              const teleportMarker = markersOnCell.find((marker) => marker.kind === "teleport");
              const merchantMarker = markersOnCell.find((marker) => marker.kind === "merchant");
              const resourcePoint = markersOnCell.find((marker) => marker.kind === "resource_point");
              const markerTitle = markersOnCell.map((marker) => marker.name).join(", ");
              const isCellInteractive = !isOutOfBounds && (
                isMovable || isPlayerCell || otherPlayersOnCell.length > 0 || markersOnCell.length > 0 || cellData.cell_type !== "Empty"
              );
              const cellAccessibleName = isOutOfBounds
                ? "Hư vô (Ngoại biên bản đồ)"
                : `Tọa độ: (${x}, ${y})${isPlayerCell ? " [Vị trí của bạn]" : ""}${
                    cellData.cell_type !== "Empty" ? ` | ${getEntityName(cellData.cell_type, cellData.content)}` : ""
                  }${markerTitle ? ` | ${markerTitle}` : ""}${otherPlayersOnCell.length > 0 ? ` | Cổ sư: ${otherPlayersOnCell.map((p) => p.username).join(", ")}` : ""}${dropOnCell ? " | Chiến lợi phẩm" : ""}`;

              return (
                <div
                  key={`${x},${y}`}
                  role={isCellInteractive ? "button" : undefined}
                  tabIndex={isCellInteractive ? 0 : undefined}
                  aria-label={isCellInteractive ? cellAccessibleName : undefined}
                  onClick={isCellInteractive ? () => handleCellClick(x, y, isMovable, cellData, portalMarker || teleportMarker || merchantMarker || primaryMarker, resourcePoint?.id) : undefined}
                  onKeyDown={isCellInteractive ? (event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      void handleCellClick(x, y, isMovable, cellData, portalMarker || teleportMarker || merchantMarker || primaryMarker, resourcePoint?.id);
                    }
                  } : undefined}
                  className={`aspect-square rounded-md border flex flex-col justify-center items-center relative transition-[color,background-color,border-color,box-shadow,opacity,transform,filter] duration-200 ${
                    isPlayerCell
                      ? "bg-emerald-500/25 border-emerald-400 shadow-[0_0_16px_rgba(16,185,129,0.35)] z-20"
                    : isMovable
                      ? `${cellTint || "bg-emerald-950/30 border-emerald-500/40"} hover:bg-emerald-900/50 hover:border-emerald-300 cursor-pointer z-10 shadow-sm`
                      : isOutOfBounds
                      ? "bg-zinc-950 border-zinc-900/30 opacity-20 cursor-not-allowed"
                      : "bg-zinc-950/60 border-zinc-800/60 opacity-40 cursor-default"
                  }`}
                  title={cellAccessibleName}
                >
                  <div className="w-full h-full flex items-center justify-center gap-1 relative z-10 pb-3">
                    {isPlayerCell && (
                      <div className="w-7 h-7 sm:w-8 sm:h-8 rounded-full bg-emerald-500 flex items-center justify-center text-[10px] text-zinc-950 font-bold border-2 border-emerald-300 shadow-[0_0_12px_rgba(52,211,153,0.8)] overflow-hidden">
                        {player.avatar_url ? <img src={player.avatar_url} alt="" className="w-full h-full object-cover" /> : "Ta"}
                      </div>
                    )}
                    {otherPlayersOnCell.length > 0 && !isPlayerCell && (
                      <div
                        className="w-6 h-6 sm:w-7 sm:h-7 rounded-full bg-purple-600/80 flex items-center justify-center text-[9px] text-zinc-100 font-bold border border-purple-400 shadow-[0_0_10px_rgba(168,85,247,0.5)] overflow-hidden"
                        title={`Cổ sư: ${otherPlayersOnCell[0].username}`}
                      >
                        {otherPlayersOnCell[0].avatar_url ? (
                          <img src={otherPlayersOnCell[0].avatar_url} alt="" className="w-full h-full object-cover" />
                        ) : (
                          otherPlayersOnCell[0].username.slice(0, 2).toUpperCase()
                        )}
                      </div>
                    )}
                    {!isOutOfBounds && !isPlayerCell && (getCellIcon(cellData.cell_type, cellData.content) || (
                      primaryMarker?.kind === "merchant" ? (
                        <Store className="w-6 h-6 text-amber-400 drop-shadow-[0_0_8px_rgba(251,191,36,0.6)] animate-pulse" />
                      ) : null
                    ))}
                    {markersOnCell.length > 0 && (
                      <span className="absolute right-0.5 top-0.5 flex max-w-[70%] flex-wrap justify-end gap-0.5 rounded-md border border-zinc-600/70 bg-zinc-950/90 p-0.5 shadow" title={markerTitle}>
                        {markersOnCell.map((marker) => (
                          <span key={`${marker.kind}:${marker.id}`} aria-hidden="true">
                            {marker.kind === "sect" ? <ShieldAlert className="h-3 w-3 text-amber-300" />
                              : marker.kind === "teleport" ? <Waypoints className="h-3 w-3 text-sky-300" />
                                : marker.kind === "aperture_portal" ? <DoorOpen className="h-3 w-3 text-violet-300" />
                                  : marker.kind === "merchant" ? <Store className="h-3 w-3 text-amber-400" />
                                    : <Mountain className="h-3 w-3 text-lime-300" />}
                          </span>
                        ))}
                      </span>
                    )}
                    {dropsOnCell.length > 0 && (
                      <span className="absolute bottom-0.5 left-0.5 flex items-center gap-0.5 rounded-full border border-amber-300/60 bg-zinc-950/95 px-1 py-0.5 shadow" title={`${dropsOnCell.length} túi chiến lợi phẩm`}>
                        <Package className="h-3.5 w-3.5 text-amber-300" />
                        {dropsOnCell.length > 1 && <span className="text-[8px] font-bold leading-none text-amber-100">{dropsOnCell.length}</span>}
                      </span>
                    )}
                  </div>

                  {(cellData.cell_type !== "Empty" || primaryMarker) && !isOutOfBounds && !isPlayerCell && (
                    <span className="text-[6px] leading-tight font-mono text-zinc-400 absolute bottom-0.5 px-0.5 truncate max-w-full text-center">
                      {cellData.cell_type !== "Empty" ? getEntityName(cellData.cell_type, cellData.content) : primaryMarker.name}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        ) : (
          <div className="text-sm text-zinc-500 py-16 flex flex-col items-center gap-3">
            <Compass className="w-8 h-8 animate-spin text-zinc-600" />
            <span>Đang cảm ứng thiên địa, giải tỏa sương mù...</span>
          </div>
        )}
      </div>

      {/* Combat Warning Banner */}
      {player.active_combat && (
        <div className="bg-rose-950/40 border border-rose-900/60 rounded-2xl p-4 flex items-center gap-3 shadow-lg animate-pulse">
          <ShieldAlert className="w-6 h-6 text-rose-500 shrink-0" />
          <div className="text-sm font-bold text-rose-300 font-serif">
            Đang chạm trán dã thú/đối thủ! Chuyển tiếp vào trận đấu để ứng chiến.
          </div>
        </div>
      )}

      {/* Exploration Log */}
      {lastLog && (
        <div className="bg-zinc-950/80 border border-zinc-800 rounded-2xl p-4 flex items-start gap-3 shadow-md">
          <div className="w-8 h-8 rounded-xl bg-amber-500/10 border border-amber-500/20 flex items-center justify-center shrink-0 mt-0.5 text-amber-400">
            <Sparkles className="w-4 h-4" />
          </div>
          <div className="space-y-1">
            <span className="text-[10px] text-zinc-500 uppercase tracking-widest block font-mono font-semibold">Ghi chép du hành</span>
            <p className="text-xs text-zinc-300 font-mono leading-relaxed whitespace-pre-line">{lastLog}</p>
          </div>
        </div>
      )}

      {/* World Atlas Modal */}
      {showAtlas && (
        <WorldAtlasModal
          regions={regions}
          currentRegionId={player.current_region}
          x={player.x}
          y={player.y}
          onClose={() => setShowAtlas(false)}
        />
      )}

      {/* Teleport Formation Modals */}
      {showBuildTeleport && (
        <TeleportBuildModal
          player={player}
          onClose={() => setShowBuildTeleport(false)}
          onSuccess={async () => {
            setShowBuildTeleport(false);
            await fetchZoneMarkers(player.current_region);
            await loadGrid();
            await fetchMyTeleportArrays();
          }}
        />
      )}

      {showDirectoryModal && (
        <TeleportDirectoryModal
          player={player}
          onClose={() => setShowDirectoryModal(false)}
          onOpenManage={handleOpenManage}
        />
      )}

      {showManageModal && selectedManageArray && (
        <TeleportManagementModal
          player={player}
          array={selectedManageArray}
          onClose={() => {
            setShowManageModal(false);
            setSelectedManageArray(null);
          }}
          onUpdated={async () => {
            await fetchZoneMarkers(player.current_region);
            await loadGrid();
            await fetchMyTeleportArrays();
          }}
        />
      )}
    </div>
  );
};
