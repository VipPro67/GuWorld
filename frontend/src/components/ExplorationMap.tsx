"use client";

import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Compass, Sparkles, DoorOpen, Droplets, Footprints, Landmark, ShieldAlert, Map as MapIcon, Package, Eye, Mountain, Swords, Waypoints, Store, UserRound } from "lucide-react";
import { useGameStore, Player, GroundDrop } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";
import { usePolling } from "@/lib/usePolling";
import { getWorldResourceGroup, WORLD_RESOURCE_ASSETS } from "@/lib/worldResourceAssets";
import { GameModal } from "@/components/ui/GameModal";
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
  ground_drops?: GroundDrop[];
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
  const fetchPlayerProfile = useGameStore((state) => state.fetchPlayerProfile);
  const profileLoading = useGameStore((state) => state.profileLoading);
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
  const [selectedGroundDrop, setSelectedGroundDrop] = useState<GroundDrop | null>(null);
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
    const dropOnCell = gridData?.ground_drops?.find((drop) => drop.x === x && drop.y === y);
    if (playerOnCell) {
      setSelectedPvpTarget(playerOnCell);
      setSelectedGroundDrop(dropOnCell || null);
      return;
    }
    if (dropOnCell) {
      setSelectedPvpTarget(null);
      setSelectedGroundDrop(dropOnCell);
      return;
    }
    setSelectedPvpTarget(null);
    setSelectedGroundDrop(null);
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
        return <Droplets className="w-6 h-6 text-cyan-300 drop-shadow-[0_0_8px_rgba(34,211,238,0.65)]" />;
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
  const markerIconSizeClass = diameter <= 5 ? "h-4 w-4" : diameter <= 7 ? "h-3.5 w-3.5" : "h-3 w-3";
  const markerBadgePaddingClass = diameter <= 5 ? "p-1" : "p-0.5";
  const cellLabelSizeClass = diameter <= 5 ? "text-[9px]" : diameter <= 7 ? "text-[8px]" : diameter <= 9 ? "text-[7px]" : "text-[6px]";

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
      <GameModal
        open={Boolean(selectedPvpTarget || selectedGroundDrop)}
        onClose={() => {
          setSelectedPvpTarget(null);
          setSelectedGroundDrop(null);
        }}
        title={selectedPvpTarget && selectedGroundDrop ? "Tương tác tại vị trí" : selectedPvpTarget ? "Cổ sư trên bản đồ" : "Chiến lợi phẩm"}
        size="lg"
        className="border-zinc-700 bg-zinc-950"
      >
        <div className="space-y-5">
          {selectedPvpTarget && (
            <section className="rounded-2xl border border-purple-500/30 bg-purple-950/35 p-4">
              <div className="flex items-center gap-3">
                <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-full border border-purple-300/60 bg-purple-900 text-sm font-bold text-purple-100">
                  {selectedPvpTarget.avatar_url
                    ? <img src={selectedPvpTarget.avatar_url} alt="" className="h-full w-full object-cover" />
                    : selectedPvpTarget.username.slice(0, 2).toUpperCase()}
                </div>
                <div className="min-w-0">
                  <h3 className="truncate font-bold text-purple-100">{selectedPvpTarget.username}</h3>
                  <p className="text-xs text-zinc-400">Tọa độ ({selectedPvpTarget.x}, {selectedPvpTarget.y}) · Cách {Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y))} ô</p>
                </div>
              </div>
              <div className="mt-4 grid gap-2 sm:grid-cols-2">
                <button
                  type="button"
                  onClick={() => {
                    const username = selectedPvpTarget.username;
                    setSelectedPvpTarget(null);
                    setSelectedGroundDrop(null);
                    void fetchPlayerProfile(username);
                  }}
                  disabled={profileLoading}
                  className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-zinc-600 bg-zinc-800/80 px-4 py-2 text-sm font-semibold text-zinc-100 transition-colors hover:bg-zinc-700 disabled:opacity-50"
                >
                  <UserRound className="h-4 w-4" /> Xem hồ sơ
                </button>
                <button
                  type="button"
                  onClick={() => void handleChallengePlayer()}
                  disabled={actionLoading || Boolean(player.active_combat) || isMemberNotLeader}
                  className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-rose-400/50 bg-rose-700/80 px-4 py-2 text-sm font-bold text-white transition-colors hover:bg-rose-600 disabled:opacity-50"
                >
                  {Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y)) <= 2
                    ? <><Swords className="h-4 w-4" /> Khai chiến</>
                    : <><Footprints className="h-4 w-4" /> Tiến gần mục tiêu</>}
                </button>
                <button
                  type="button"
                  onClick={() => void handleMapTrade()}
                  disabled={actionLoading || Boolean(player.active_combat) || isMemberNotLeader || Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y)) > 10}
                  className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-emerald-500/50 bg-emerald-800/50 px-4 py-2 text-sm font-semibold text-emerald-100 transition-colors hover:bg-emerald-700/70 disabled:opacity-50 sm:col-span-2"
                >
                  Giao dịch tại chỗ
                </button>
              </div>
              <p className="mt-3 text-xs leading-relaxed text-zinc-400">
                {Math.max(Math.abs(selectedPvpTarget.x - player.x), Math.abs(selectedPvpTarget.y - player.y)) <= 2
                  ? "Có thể khai chiến trong tầm hiện tại."
                  : "Tiến gần sẽ di chuyển tối đa 5 ô mỗi lần; khi vào tầm, nút sẽ chuyển thành Khai chiến."}
              </p>
            </section>
          )}
          {selectedGroundDrop && (
            <section className="rounded-2xl border border-amber-500/30 bg-amber-950/30 p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="font-bold text-amber-200">Chiến lợi phẩm của {selectedGroundDrop.loser_name}</h3>
                  <p className="mt-1 text-xs text-zinc-400">
                    Rơi tại ({selectedGroundDrop.x}, {selectedGroundDrop.y})
                    {selectedGroundDrop.killer_name ? ` · Hạ bởi ${selectedGroundDrop.killer_name}` : ""}
                  </p>
                </div>
                <Package className="h-5 w-5 shrink-0 text-amber-300" />
              </div>
              <div className="mt-3 flex flex-wrap gap-2 text-xs text-zinc-200">
                {(selectedGroundDrop.loot_data?.primeval_stones || 0) > 0 && <span className="rounded-lg border border-zinc-700 bg-zinc-900 px-2.5 py-1.5">Nguyên thạch ×{selectedGroundDrop.loot_data.primeval_stones}</span>}
                {(selectedGroundDrop.loot_data?.immortal_stones || 0) > 0 && <span className="rounded-lg border border-zinc-700 bg-zinc-900 px-2.5 py-1.5">Tiên nguyên thạch ×{selectedGroundDrop.loot_data.immortal_stones}</span>}
                {(selectedGroundDrop.loot_data?.items || []).map((item, index) => <span key={`${item.item_id}-${index}`} className="rounded-lg border border-zinc-700 bg-zinc-900 px-2.5 py-1.5">{itemDictionary[item.item_id]?.name || item.item_id} ×{item.quantity}</span>)}
                {(selectedGroundDrop.loot_data?.gus || []).map((gu: ApiResponseData, index: number) => <span key={`${gu.instance_id || gu.gu_id}-${index}`} className="rounded-lg border border-zinc-700 bg-zinc-900 px-2.5 py-1.5">{gu.name || gu.gu_name || gu.gu_id || "Cổ"}</span>)}
                {(selectedGroundDrop.loot_data?.recipes || []).map((recipe, index) => <span key={`${recipe}-${index}`} className="rounded-lg border border-zinc-700 bg-zinc-900 px-2.5 py-1.5">Công thức: {recipe}</span>)}
              </div>
              <p className="mt-3 text-xs leading-relaxed text-amber-200/80">
                {selectedGroundDrop.is_protected && selectedGroundDrop.protected_until && new Date(selectedGroundDrop.protected_until).getTime() > Date.now()
                  ? `Đang được bảo hộ đến ${new Date(selectedGroundDrop.protected_until).toLocaleString("vi-VN")}`
                  : "Không còn bảo hộ"}
                {" · "}Hết hạn {new Date(selectedGroundDrop.expires_at).toLocaleString("vi-VN")}
              </p>
              <div className="mt-4 grid gap-2 sm:grid-cols-2">
                {(player.x !== selectedGroundDrop.x || player.y !== selectedGroundDrop.y) && (
                  <button
                    type="button"
                    onClick={async () => {
                      const result = await movePlayer(selectedGroundDrop.x, selectedGroundDrop.y);
                      if (result) await loadGrid();
                    }}
                    disabled={actionLoading || Boolean(player.active_combat) || isMemberNotLeader}
                    className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-sky-400/50 bg-sky-800/60 px-4 py-2 text-sm font-semibold text-sky-100 transition-colors hover:bg-sky-700/70 disabled:opacity-50"
                  >
                    <Footprints className="h-4 w-4" /> Tiến gần chiến lợi phẩm
                  </button>
                )}
                <button
                  type="button"
                  onClick={async () => {
                    const result = await lootGroundDrop(selectedGroundDrop.id);
                    if (result?.success) {
                      setSelectedGroundDrop(null);
                      await loadGrid();
                    }
                  }}
                  disabled={actionLoading || player.x !== selectedGroundDrop.x || player.y !== selectedGroundDrop.y}
                  className="inline-flex min-h-11 items-center justify-center gap-2 rounded-xl border border-amber-400/60 bg-amber-700/80 px-4 py-2 text-sm font-bold text-amber-50 transition-colors hover:bg-amber-600 disabled:opacity-50"
                >
                  <Package className="h-4 w-4" /> Nhặt chiến lợi phẩm
                </button>
              </div>
              {player.x !== selectedGroundDrop.x || player.y !== selectedGroundDrop.y
                ? <p className="mt-3 text-xs text-zinc-400">Nhân vật phải đứng đúng ô mới nhặt được. Mỗi lần tiến gần, tối đa 5 ô.</p>
                : null}
            </section>
          )}
        </div>
      </GameModal>
      <section aria-label="Chú giải bản đồ" className={`mx-auto mb-3 flex w-full flex-wrap items-center gap-x-3 gap-y-2 rounded-xl border border-zinc-700/80 bg-zinc-950/90 px-3 py-2 text-[10px] shadow-lg ${maxContainerWidth}`}>
        <span className="font-bold uppercase tracking-wider text-zinc-400">Ký hiệu</span>
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-sky-200"><Waypoints className="h-4 w-4 text-sky-300" />Điểm dịch chuyển</span>
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-cyan-100"><Droplets className="h-4 w-4 text-cyan-300" />Nguyên Tuyền</span>
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-amber-100"><Package className="h-4 w-4 text-amber-300" />Chiến lợi phẩm</span>
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-emerald-100"><DoorOpen className="h-4 w-4 text-emerald-300" />Cổng Bí Cảnh</span>
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-violet-100"><DoorOpen className="h-4 w-4 text-violet-300" />Cổng Tiên Khiếu</span>
        <span className="inline-flex items-center gap-1.5 whitespace-nowrap text-lime-100"><Mountain className="h-4 w-4 text-lime-300" />Điểm tài nguyên</span>
      </section>
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
                ? "bg-emerald-950/80 border-emerald-300/75 ring-1 ring-inset ring-emerald-300/25"
                : cellData.cell_type === "INHERITANCE_ALTAR"
                  ? "bg-purple-950/80 border-purple-300/75 ring-1 ring-inset ring-purple-300/25"
                  : cellData.cell_type === "Boss" || cellData.cell_type === "Monster"
                    ? "bg-rose-950/75 border-rose-300/70 ring-1 ring-inset ring-rose-300/20"
                    : cellData.cell_type === "Resource" || cellData.cell_type === "PrimevalVein"
                      ? "bg-cyan-950/80 border-cyan-300/75 ring-1 ring-inset ring-cyan-300/25"
                      : primaryMarker?.kind === "sect"
                        ? "bg-amber-950/75 border-amber-300/70 ring-1 ring-inset ring-amber-300/20"
                        : primaryMarker?.kind === "teleport"
                          ? "bg-sky-950/80 border-sky-300/75 ring-1 ring-inset ring-sky-300/25"
                          : primaryMarker?.kind === "aperture_portal"
                            ? "bg-violet-950/80 border-violet-300/75 ring-1 ring-inset ring-violet-300/25"
                            : primaryMarker?.kind === "resource_point"
                              ? "bg-lime-950/75 border-lime-300/70 ring-1 ring-inset ring-lime-300/20"
                              : dropOnCell
                                ? "bg-amber-950/75 border-amber-300/70 ring-1 ring-inset ring-amber-300/20"
                                : "";
              const portalMarker = markersOnCell.find((marker) => marker.kind === "aperture_portal");
              const teleportMarker = markersOnCell.find((marker) => marker.kind === "teleport");
              const merchantMarker = markersOnCell.find((marker) => marker.kind === "merchant");
              const resourcePoint = markersOnCell.find((marker) => marker.kind === "resource_point");
              const markerTitle = markersOnCell.map((marker) => marker.name).join(", ");
              const cellLabel = cellData.cell_type === "PrimevalVein"
                ? "Nguyên Tuyền"
                : cellData.cell_type === "ENTRY_PORTAL" || cellData.cell_type === "EXIT_PORTAL" || cellData.cell_type === "PORTAL_EXIT"
                  ? "Cổng Bí Cảnh"
                  : primaryMarker?.kind === "teleport"
                    ? "Điểm dịch chuyển"
                    : primaryMarker?.kind === "aperture_portal"
                      ? "Cổng Tiên Khiếu"
                      : primaryMarker?.kind === "resource_point"
                        ? "Điểm tài nguyên"
                        : dropOnCell
                          ? "Chiến lợi phẩm"
                          : cellData.cell_type !== "Empty"
                            ? getEntityName(cellData.cell_type, cellData.content)
                            : primaryMarker?.name;
              const isCellInteractive = !isOutOfBounds && (
                isMovable || isPlayerCell || otherPlayersOnCell.length > 0 || dropsOnCell.length > 0 || markersOnCell.length > 0 || cellData.cell_type !== "Empty"
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
                      <span className={`absolute right-1 top-1 flex max-w-[75%] flex-wrap justify-end gap-1 rounded-md border border-zinc-500/80 bg-zinc-950/95 shadow-lg ${markerBadgePaddingClass}`} title={markerTitle}>
                        {markersOnCell.map((marker) => (
                          <span key={`${marker.kind}:${marker.id}`} aria-hidden="true">
                            {marker.kind === "sect" ? <ShieldAlert className={`${markerIconSizeClass} text-amber-300`} />
                              : marker.kind === "teleport" ? <Waypoints className={`${markerIconSizeClass} text-sky-300`} />
                                : marker.kind === "aperture_portal" ? <DoorOpen className={`${markerIconSizeClass} text-violet-300`} />
                                  : marker.kind === "merchant" ? <Store className={`${markerIconSizeClass} text-amber-300`} />
                                    : <Mountain className={`${markerIconSizeClass} text-lime-300`} />}
                          </span>
                        ))}
                      </span>
                    )}
                    {dropsOnCell.length > 0 && (
                      <span className="absolute left-1 top-1 flex items-center gap-1 rounded-md border border-amber-300/80 bg-zinc-950/95 px-1.5 py-1 shadow-lg" title={`${dropsOnCell.length} túi chiến lợi phẩm`}>
                        <Package className="h-4 w-4 text-amber-300" />
                        {dropsOnCell.length > 1 && <span className="text-[8px] font-bold leading-none text-amber-100">{dropsOnCell.length}</span>}
                      </span>
                    )}
                  </div>

                  {cellLabel && !isOutOfBounds && !isPlayerCell && (
                    <span className={`absolute bottom-0.5 left-1/2 z-20 max-w-[calc(100%-0.5rem)] -translate-x-1/2 truncate rounded border border-white/15 bg-zinc-950/95 px-1 py-0.5 text-center font-mono font-semibold leading-tight text-zinc-100 shadow ${cellLabelSizeClass}`}>
                      {cellLabel}
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
