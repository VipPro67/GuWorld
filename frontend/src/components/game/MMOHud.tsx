"use client";

import { useEffect, useRef, useState } from "react";
import { BookOpen, Compass, Crown, Flame, Gem, History, MapPin, Menu, Minus, Package, PawPrint, Plus, ScrollText, Settings2, ShoppingBag, Sparkles, Store, Swords, Trophy, Users, X, Eye, EyeOff, ChevronDown, ChevronUp, Shield, Crosshair, Pickaxe, Gift } from "lucide-react";
import type { WorldSkillInfo } from "@/game/network/Protocol";
import type { RuntimeEntity } from "@/game/core/WorldRuntime";
import { getMonsterArt } from "@/game/assets/MonsterArt";
import { formatManaCost } from "@/lib/gu-mana";
import { useGameStore, guSpecFor, type Player } from "@/store/useGameStore";
import { useWorldStore } from "@/store/useWorldStore";
import { translatePath } from "@/lib/translate";
import { NotificationBell } from "@/components/NotificationBell";
import { VirtualJoystick } from "./VirtualJoystick";
import { TeleportDirectoryModal } from "@/components/Teleport/TeleportDirectoryModal";
import { TeleportBuildModal } from "@/components/Teleport/TeleportBuildModal";
import type { MMOPanel } from "./MMOBusinessPanels";
import type { MapPortal } from "@/game/world/MapLoader";
import { useQuickActionIds, QUICK_ACTIONS_MAP } from "@/lib/mmo-quick-actions";
import { resolvePlayerViewMode, resolveTargetViewMode, hotbarKeyLabel, updateUiPreferences, useHotbarKeybinds } from "@/lib/ui-preferences";
import { QuickActionCustomizer } from "./QuickActionCustomizer";
import { toWorldMarkerResources } from "@/game/world/markers/resource.ts";
import { collectMarkers, findNearestMarker } from "@/game/world/markers/collect";
import type { WorldMarker } from "@/game/world/markers/types";
import { getGuRankColor } from "@/lib/gu-frames";
import { getGuIconUrl } from "@/lib/gu-icons";

export interface HudTarget {
  id: string;
  kind: "player" | "monster";
  hp: number;
  maxHp: number;
  shield: number;
  name: string;
  templateId: string;
  aiState: string;
  pvpEnabled?: boolean;
}

export interface MMOHudProps {
  connected: boolean;
  worldActive?: boolean;
  tickRate: number;
  mana: number;
  maxMana: number;
  hp: number;
  maxHp: number;
  shield: number;
  skills: WorldSkillInfo[];
  guSlots: Array<{ slot: number; guId: string | null; name: string | null; rank: number | null; supported: boolean }>;
  cooldownRemaining: number[];
  castName: string;
  castProgress: number | null;
  target: HudTarget | null;
  statusText: string;
  position: { x: number; y: number };
  nearby: RuntimeEntity[];
  zoneId: string;
  player: Player | null;
  activePanel: MMOPanel;
  menuOpen: boolean;
  onToggleMenu: () => void;
  onCloseMenu: () => void;
  onOpenPanel: (panel: Exclude<MMOPanel, null>) => void;
  onOpenView: (view: string) => void;
  onCast: (slot: number) => void;
  onAttack: () => void;
  rightClickAction: "attack" | "skill";
  rightClickSlot: number;
  onClearTarget: () => void;
  onSelectTarget?: (id: string) => void;
  onMoveJoystick?: (vector: { x: number; y: number }) => void;
  onZoomIn?: () => number;
  onZoomOut?: () => number;
  mapPortals?: MapPortal[];
  aimingSlot?: number | null;
  onCancelAim?: () => void;
  onHoverSkill?: (slot: number | null) => void;
}

const MODE_LABEL: Record<WorldSkillInfo["mode"], string> = {
  TARGET: "Mục tiêu",
  DIRECTION: "Định hướng",
  GROUND: "Diện rộng",
  SELF: "Bản thân",
};

const RADAR_MARKER_CLASS: Record<WorldMarker["kind"], string> = {
  vein: "h-2.5 w-2.5 rounded-full bg-emerald-400 border border-emerald-950 shadow-[0_0_6px_#10b981]",
  drop: "h-2 w-2 rounded-full bg-amber-400 border border-amber-950 shadow-[0_0_5px_#f59e0b]",
  teleport: "h-2.5 w-2.5 rounded-full bg-cyan-400 border border-blue-950 shadow-[0_0_6px_#38bdf8]",
  portal: "h-3 w-3 rounded-full bg-blue-400 border border-white shadow-[0_0_8px_#38bdf8] animate-pulse",
  sect: "h-3.5 w-3.5 rounded border text-[8px]",
  resource: "h-3 w-3 rotate-45 rounded-sm bg-emerald-400 border border-emerald-950 shadow-[0_0_7px_#10b981]",
};

function radarMarkerSpec(marker: WorldMarker): { className: string; glyph: string; title: string } {
  if (marker.kind === "sect") {
    const isMine = marker.sect?.isMine ?? false;
    return {
      className: `${RADAR_MARKER_CLASS.sect} ${isMine ? "bg-blue-600 border-blue-200 shadow-[0_0_8px_#3b82f6]" : "bg-amber-600 border-amber-200 shadow-[0_0_8px_#f59e0b]"}`,
      glyph: "🏯",
      title: `Tông Môn: ${marker.name} (Cấp ${marker.sect?.level ?? 1})`,
    };
  }
  if (marker.kind === "vein") {
    return {
      className: RADAR_MARKER_CLASS.vein,
      glyph: "",
      title: `${marker.name} (${marker.vein?.rarity || "Thường"})`,
    };
  }
  if (marker.kind === "drop") {
    return { className: RADAR_MARKER_CLASS.drop, glyph: "", title: `Chiến Lợi Phẩm (${marker.name})` };
  }
  if (marker.kind === "portal") {
    return { className: RADAR_MARKER_CLASS.portal, glyph: "", title: `Cổng: Đến ${marker.name}` };
  }
  if (marker.kind === "resource") {
    const famous = marker.resource?.point_class === "FAMOUS";
    return {
      className: `${RADAR_MARKER_CLASS.resource} ${famous ? "bg-amber-400 shadow-[0_0_7px_#f59e0b]" : ""}`,
      glyph: "",
      title: `${famous ? "Núi Danh" : "Mỏ"}: ${marker.name} · ${marker.resource?.stock_total ?? 0}/${marker.resource?.stock_cap ?? 0}`,
    };
  }
  return { className: RADAR_MARKER_CLASS.teleport, glyph: "", title: `Truyền Tống: ${marker.name}` };
}

const MODE_BADGE: Record<
  WorldSkillInfo["mode"],
  { label: string; shortLabel: string; icon: string; color: string; bg: string; border: string }
> = {
  TARGET: {
    label: "Đơn thể",
    shortLabel: "Đơn",
    icon: "🎯",
    color: "text-sky-300",
    bg: "bg-sky-950/90",
    border: "border-sky-500/70",
  },
  DIRECTION: {
    label: "Định hướng",
    shortLabel: "Hướng",
    icon: "🏹",
    color: "text-cyan-300",
    bg: "bg-cyan-950/90",
    border: "border-cyan-500/70",
  },
  GROUND: {
    label: "Vùng đất",
    shortLabel: "Vùng",
    icon: "💥",
    color: "text-amber-300",
    bg: "bg-amber-950/90",
    border: "border-amber-500/70",
  },
  SELF: {
    label: "Bản thân",
    shortLabel: "Tự",
    icon: "🛡️",
    color: "text-emerald-300",
    bg: "bg-emerald-950/90",
    border: "border-emerald-500/70",
  },
};

const frame = "border border-[#a88952]/70 bg-[#211810f5] text-[#f4e8cb] shadow-[0_8px_28px_rgba(0,0,0,.55),inset_0_1px_rgba(255,238,197,.08)] backdrop-blur-sm";

const menuGroups: Array<{ title: string; items: Array<{ label: string; icon: typeof Package; view?: string; panel?: Exclude<MMOPanel, null>; special?: "divination" | "quick-actions" | "teleport-directory" | "teleport-build"; minRank?: number }> }> = [
  { title: "Thế giới", items: [
    { label: "Danh sách map", icon: Compass, panel: "maps" },
    { label: "Truyền Tống Trận", icon: Compass, special: "teleport-directory" },
    { label: "Bố trí Truyền Tống Trận", icon: Plus, special: "teleport-build" },
    { label: "Tiên khiếu", icon: Sparkles, view: "aperture", minRank: 6 },
    { label: "Mỏ tài nguyên", icon: Gem, panel: "resources" },
    { label: "Bí cảnh", icon: Gem, view: "dungeon" },
    { label: "Mộng cảnh", icon: Compass, view: "dream-realm" },
    { label: "Tông môn", icon: Crown, view: "sect" },
    { label: "Nô thú", icon: PawPrint, view: "pets" },
  ] },
  { title: "Nhân vật", items: [
    { label: "Nhân vật", icon: Crown, view: "character" },
    { label: "Túi đồ", icon: Package, panel: "inventory" },
    { label: "Tu luyện", icon: Sparkles, panel: "cultivation" },
    { label: "Luyện Cổ", icon: Flame, view: "refine" },
    { label: "Nhiệm vụ", icon: ScrollText, view: "achievements" },
    { label: "Tổ đội", icon: Users, panel: "party" },
  ] },
  { title: "Giao thương", items: [
    { label: "Cửa hàng", icon: Store, view: "shop" },
    { label: "Chợ Tiên", icon: ShoppingBag, view: "auction" },
    { label: "Cát Bảo Các", icon: Gem, view: "premium-store" },
  ] },
  { title: "Hoạt động", items: [
    { label: "Võ đài", icon: Swords, view: "arena" },
    { label: "Sinh tử đài", icon: Swords, view: "death-arena" },
    { label: "Đại hội võ thuật", icon: Trophy, view: "martial-arena" },
    { label: "Phòng tập", icon: Swords, view: "practice-room" },
    { label: "Xếp hạng", icon: Trophy, view: "leaderboard" },
    { label: "Giải trí", icon: Sparkles, view: "gacha" },
  ] },
  { title: "Khác", items: [
    { label: "Thiên Cơ Các", icon: Compass, special: "divination" },
    { label: "Cài đặt & Tiện ích", icon: Settings2, panel: "settings" },
    { label: "Lịch sử chiến đấu", icon: History, view: "combat-history" },
    { label: "Ký sự tu hành", icon: BookOpen, view: "audit-logs" },
    { label: "Góp ý", icon: ScrollText, view: "feedback" },
  ] },
];

function pct(value: number, max: number) {
  return max > 0 ? Math.max(0, Math.min(100, value / max * 100)) : 0;
}

export function MMOHud(props: MMOHudProps) {
  const [questsOpen, setQuestsOpen] = useState(true);
  const [minimapCollapsed, setMinimapCollapsed] = useState(false);
  const [mobileNavCollapsed, setMobileNavCollapsed] = useState(false);
  const [tribulationPanelExpanded, setTribulationPanelExpanded] = useState(false);
  const [playerViewMode, setPlayerViewMode] = useState<"expanded" | "compact" | "minimized">(() => resolvePlayerViewMode());

  const changePlayerViewMode = (mode: "expanded" | "compact" | "minimized") => {
    setPlayerViewMode(mode);
    updateUiPreferences({ playerViewMode: mode });
  };

  const [targetViewMode, setTargetViewMode] = useState<"expanded" | "compact" | "minimized">(() => resolveTargetViewMode());

  const changeTargetViewMode = (mode: "expanded" | "compact" | "minimized") => {
    setTargetViewMode(mode);
    updateUiPreferences({ targetViewMode: mode });
  };

  const hotbarKeybinds = useHotbarKeybinds();
  // Only render slots that actually hold a Gu or a realtime skill; empty
  // placeholders are gone so the HUD reflects the current loadout.
  const hotbarSlots = Array.from({ length: 12 }, (_, index) => index + 1).filter((slot) =>
    props.skills.some((entry) => entry.slot === slot) ||
    props.guSlots.some((entry) => entry.slot === slot && entry.guId),
  );

  const monsterDictionary = useGameStore((state) => state.monsterDictionary);
  const guDictionary = useGameStore((state) => state.guDictionary);
  const passiveStats = useGameStore((state) => state.passiveStats);
  const fetchPassiveStats = useGameStore((state) => state.fetchPassiveStats);

  useEffect(() => {
    if (props.connected && props.player && !passiveStats) {
      void fetchPassiveStats();
    }
  }, [props.connected, props.player?.id, passiveStats, fetchPassiveStats]);

  // Compute player's Investigation stat (base 1.0 + equipped Gu passives)
  let equippedInvestigation = 0;
  if (props.player?.gu_inventory) {
    for (const gu of props.player.gu_inventory) {
      if (gu.equipped && gu.status === "ACTIVE") {
        const spec = guDictionary[gu.gu_id];
        if (spec?.effects?.blocks) {
          for (const block of spec.effects.blocks) {
            if (block.operations) {
              for (const op of block.operations) {
                if (op.op === "STAT_MOD" && (op.stat === "INVESTIGATION" || op.stat === "investigation")) {
                  equippedInvestigation += Number(op.value) || 0;
                }
              }
            }
          }
        }
      }
    }
  }

  const passiveInvestigation = passiveStats?.stats?.investigation;
  const investigationStat = typeof passiveInvestigation === "number" && passiveInvestigation > 0
    ? passiveInvestigation
    : 1.0 + equippedInvestigation;

  // Radar range scales dynamically with player's Investigation stat
  const baseRadarRange = Math.min(64, Math.max(16, Math.round(18 + investigationStat * 4)));
  const [radarOffset, setRadarOffset] = useState(0);
  const [cameraZoom, setCameraZoom] = useState(1.0);
  const radarRange = Math.min(80, Math.max(12, baseRadarRange + radarOffset));

  const handleZoomIn = () => {
    if (props.onZoomIn) {
      const next = props.onZoomIn();
      setCameraZoom(next);
    }
    setRadarOffset((v) => Math.max(-14, v - 5));
  };

  const handleZoomOut = () => {
    if (props.onZoomOut) {
      const next = props.onZoomOut();
      setCameraZoom(next);
    }
    setRadarOffset((v) => Math.min(30, v + 5));
  };

  const targetSpec = props.target?.templateId
    ? monsterDictionary[props.target.templateId]
    : props.target?.name
      ? Object.values(monsterDictionary).find((m) => m.name === props.target?.name)
      : null;

  let targetPathLabel = "";
  if (targetSpec?.path) {
    targetPathLabel = translatePath(targetSpec.path);
  } else if (targetSpec?.path_masteries && Object.keys(targetSpec.path_masteries).length > 0) {
    const topPath = Object.entries(targetSpec.path_masteries).sort((a, b) => b[1] - a[1])[0][0];
    targetPathLabel = translatePath(topPath);
  }

  const targetGuList: string[] = [];
  if (targetSpec?.gu_inventory && targetSpec.gu_inventory.length > 0) {
    for (const item of targetSpec.gu_inventory) {
      const name = item.name || guDictionary[item.gu_id]?.name || item.gu_id;
      if (name && !targetGuList.includes(name)) targetGuList.push(name);
    }
  }
  if (targetSpec?.gu_pool && targetSpec.gu_pool.length > 0) {
    for (const pool of targetSpec.gu_pool) {
      const name = guDictionary[pool.gu_id]?.name || pool.gu_id;
      if (name && !targetGuList.includes(name)) targetGuList.push(name);
    }
  }

  // Scouting power vs target stealth/difficulty
  const playerRank = props.player?.rank ?? 1;
  const baseRankPower = Math.pow(2, Math.max(0, playerRank - 1)) * 10;
  const scoutingPower = baseRankPower + (investigationStat * 6.0);

  const monsterRank = targetSpec?.rank ?? (targetSpec?.type === "Boss" ? 5 : targetSpec?.type === "Elite" ? 3 : 1);
  const monsterTypeBonus = targetSpec?.type === "Boss" ? 30 : targetSpec?.type === "Elite" ? 15 : 0;
  const monsterStealthResistance = (Math.pow(2, Math.max(0, monsterRank - 1)) * 10) + monsterTypeBonus;

  let investigationStatus = {
    badge: `Thần Thức Áp Đảo (${investigationStat.toFixed(1)})`,
    level: "high" as "high" | "medium" | "low",
  };
  if (scoutingPower < monsterStealthResistance * 0.75) {
    investigationStatus = {
      badge: `Thần Thức Bị Áp Chế (${investigationStat.toFixed(1)})`,
      level: "low",
    };
  } else if (scoutingPower < monsterStealthResistance * 1.15) {
    investigationStatus = {
      badge: `Thần Thức Tương Đương (${investigationStat.toFixed(1)})`,
      level: "medium",
    };
  }

  const dailyMissions = useGameStore((state) => state.dailyMissions);
  const fetchDailyMissions = useGameStore((state) => state.fetchDailyMissions);
  const openDivinationModal = useGameStore((state) => state.openDivinationModal);

  // World Entities (Ground Drops, Veins, Teleport Arrays)
  const groundDrops = useGameStore((state) => state.groundDrops);
  const fetchGroundDrops = useGameStore((state) => state.fetchGroundDrops);
  const lootGroundDrop = useGameStore((state) => state.lootGroundDrop);

  const nearbyCells = useWorldStore((state) => state.nearbyCells);
  const fetchNearbyCells = useWorldStore((state) => state.fetchNearbyCells);
  const harvestVein = useWorldStore((state) => state.harvestVein);
  const zoneMarkers = useWorldStore((state) => state.zoneMarkers);
  const zoneMarkersRegion = useWorldStore((state) => state.zoneMarkersRegion);
  const fetchZoneMarkers = useWorldStore((state) => state.fetchZoneMarkers);
  const fetchWorldResourcePoints = useWorldStore((state) => state.fetchWorldResourcePoints);
  const resourcePoints = useWorldStore((state) => state.resourcePoints);
  const resourcePointsRegion = useWorldStore((state) => state.resourcePointsRegion);

  const [showTeleportDirectory, setShowTeleportDirectory] = useState(false);
  const [showTeleportBuild, setShowTeleportBuild] = useState(false);
  const [showQuickActionModal, setShowQuickActionModal] = useState(false);
  const { ids: quickActionIds } = useQuickActionIds();

  const activeTribulation = useGameStore((state) => state.activeTribulation);
  const fetchActiveTribulation = useGameStore((state) => state.fetchActiveTribulation);
  const realtimeResolveTribulationVictory = useGameStore((state) => state.realtimeResolveTribulationVictory);
  const realtimeExitTribulation = useGameStore((state) => state.realtimeExitTribulation);
  const [tribulationActionLoading, setTribulationActionLoading] = useState(false);
  const tribulationRemaining = activeTribulation
    ? (typeof activeTribulation.remaining_enemies === "number"
      ? activeTribulation.remaining_enemies
      : activeTribulation.enemies.length)
    : 0;
  const tribulationTotal = activeTribulation?.total_enemies ?? tribulationRemaining;

  // Welcome zone IDs include the runtime instance ("map#instance"). Region
  // lookups (teleport arrays, sect markers, aperture panels) need the map/region
  // part only, otherwise nothing matches on the live world.
  const zoneMapId = (props.zoneId || "").split("#")[0];
  const zoneRegionId = props.player?.current_region || zoneMapId;

  useEffect(() => {
    if (zoneMapId === "aperture_tribulation") {
      void fetchActiveTribulation(true);
      // Keep the settlement counter aligned with the authoritative event while
      // the player is inside the encounter. The in-map radar count is only a
      // local view and must never be used as "quái còn lại".
      const timer = window.setInterval(() => {
        void fetchActiveTribulation(true);
      }, 5000);
      return () => window.clearInterval(timer);
    }
    setTribulationPanelExpanded(false);
  }, [zoneMapId, fetchActiveTribulation]);

  // Poll world state without restarting the timers every time the player moves.
  // Drops are time-sensitive; cells change much less often and are fetched on
  // a slower cadence to avoid repeating the large rectangle query.
  const positionRef = useRef(props.position);
  useEffect(() => {
    positionRef.current = props.position;
  }, [props.position]);
  useEffect(() => {
    if (!props.connected || props.worldActive === false || !props.player) return;
    const fetchDrops = () => {
      const { x, y } = positionRef.current;
      const px = Math.round(x);
      const py = Math.round(y);
      void fetchGroundDrops({ x: px, y: py, radius: 28 });
    };
    const fetchCells = () => {
      const { x, y } = positionRef.current;
      void fetchNearbyCells(Math.round(x), Math.round(y), 28);
    };
    fetchDrops();
    fetchCells();
    void fetchZoneMarkers(zoneRegionId);
    if (zoneRegionId && zoneRegionId !== "aperture_tribulation") void fetchWorldResourcePoints(zoneRegionId);
    const dropsTimer = window.setInterval(fetchDrops, 8000);
    const cellsTimer = window.setInterval(fetchCells, 30000);
    const markersTimer = window.setInterval(() => void fetchZoneMarkers(zoneRegionId), 30000);
    const resourceTimer = window.setInterval(() => {
      if (zoneRegionId && zoneRegionId !== "aperture_tribulation") void fetchWorldResourcePoints(zoneRegionId);
    }, 30000);
    return () => {
      window.clearInterval(dropsTimer);
      window.clearInterval(cellsTimer);
      window.clearInterval(markersTimer);
      window.clearInterval(resourceTimer);
    };
  }, [props.connected, props.worldActive, props.player?.id, zoneRegionId, fetchGroundDrops, fetchNearbyCells, fetchZoneMarkers, fetchWorldResourcePoints]);

  const handleInvitePlayer = async (username: string) => {
    let party = useGameStore.getState().party;
    if (!party) {
      const created = await useGameStore.getState().createParty();
      if (!created) return;
      party = useGameStore.getState().party;
    }
    if (party && party.leader_id !== props.player?.id) {
      useGameStore.getState().addToast("Chỉ trưởng nhóm mới có quyền mời thành viên!", "error");
      return;
    }
    await useGameStore.getState().invitePlayer(username);
  };

  const handleWhisperPlayer = (username: string) => {
    const gs = useGameStore.getState();
    gs.setActiveChannel("private");
    void gs.fetchPrivateThread(username);
    gs.setChatOpen(true);
  };

  const mySectStatus = useGameStore((state) => state.mySectStatus);
  const attackSect = useGameStore((state) => state.attackSect);

  // Single marker view-model: the radar and the proximity prompts below read the
  // same array the Phaser world layer draws, so they cannot drift apart.
  const markers = collectMarkers({
    regionId: zoneRegionId,
    cells: nearbyCells,
    zoneMarkers: zoneMarkersRegion === zoneRegionId ? zoneMarkers : [],
    groundDrops,
    mapPortals: props.mapPortals,
    mySectId: mySectStatus?.sect?.id,
    resourcePoints: resourcePointsRegion === zoneRegionId ? toWorldMarkerResources(resourcePoints) : [],
  });

  const closestSect = findNearestMarker(markers, props.position, "sect");
  const closestDrop = findNearestMarker(markers, props.position, "drop");
  const closestVein = findNearestMarker(markers, props.position, "vein");
  const closestTp = findNearestMarker(markers, props.position, "teleport");
  const distanceTo = (marker: { x: number; y: number } | null) =>
    marker ? Math.hypot(marker.x - props.position.x, marker.y - props.position.y) : 0;

  const targetArt = props.target?.kind === "monster" ? getMonsterArt(props.target.templateId) : null;
  const immortal = (props.player?.rank ?? 1) >= 6;
  const manaName = immortal ? "Tiên nguyên" : "Chân nguyên";

  useEffect(() => {
    if (props.connected && props.player) void fetchDailyMissions();
  }, [props.connected, props.player?.id, fetchDailyMissions]);

  if (!props.connected) return null;
  const mission = dailyMissions.find((item) => !item.claimed) as { title?: string; description?: string; progress?: number; target?: number; claimable?: boolean } | undefined;
  const nearby = props.nearby.filter((entity) => !entity.isSelf && entity.hp > 0 && Math.abs(entity.x - props.position.x) <= radarRange && Math.abs(entity.y - props.position.y) <= radarRange);
  const actionItems = [
    ...quickActionIds
      .map((id) => {
        const def = QUICK_ACTIONS_MAP[id];
        if (!def) return null;
        return {
          label: def.label,
          icon: def.icon,
          action: () => {
            if (def.panel) props.onOpenPanel(def.panel);
            else if (def.view) props.onOpenView(def.view);
            else if (def.special === "divination") openDivinationModal();
          },
          key: def.id,
        };
      })
      .filter((item): item is { label: string; icon: typeof Package; action: () => void; key: string } => item !== null),
    { label: "Danh mục", icon: Menu, action: props.onToggleMenu, key: "menu" },
  ];

  const soulHp = props.player?.soul_hp ?? 100;
  const maxSoul = props.player?.max_soul ?? 100;
  const soulTier = props.player?.soul_tier || "Phàm Hồn";
  const thoughts = props.player?.current_thought ?? 100;
  const maxThoughts = props.player?.max_thought ?? 100;
  const playerStatuses = props.player?.statuses ?? [];

  const questBoxTop =
    playerViewMode === "expanded"
      ? "top-[230px]"
      : playerViewMode === "compact"
        ? "top-[160px]"
        : "top-[60px]";

  return (
    <div className="pointer-events-none absolute inset-0 z-20 select-none font-serif">
      {/* Player Frame: Minimized Capsule Mode (Tối ưu cho mobile khi giao tranh) */}
      {playerViewMode === "minimized" && (
        <div className="pointer-events-auto absolute left-2 top-2 flex items-center gap-1.5 rounded-full border border-[#a88952]/70 bg-[#16100bf2] px-2 py-1 shadow-lg backdrop-blur-md text-xs text-[#f5dfab] z-20">
          <button
            type="button"
            onClick={() => props.onOpenView("character")}
            className="h-6 w-6 rounded-full overflow-hidden border border-[#d4b478] shrink-0"
            title="Mở bảng nhân vật"
          >
            {props.player?.avatar_url ? (
              <img src={props.player.avatar_url} alt="" className="h-full w-full object-cover" />
            ) : (
              (props.player?.username?.slice(0, 1).toUpperCase() ?? "C")
            )}
          </button>
          <div className="flex flex-col gap-0.5 w-16">
            <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden border border-red-950">
              <div className="h-full bg-red-500 transition-[width]" style={{ width: `${pct(props.hp, props.maxHp)}%` }} />
            </div>
            <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden border border-cyan-950">
              <div className="h-full bg-cyan-400 transition-[width]" style={{ width: `${pct(props.mana, props.maxMana)}%` }} />
            </div>
          </div>
          <NotificationBell size="sm" />
          <button
            type="button"
            onClick={() => changePlayerViewMode("compact")}
            title="Mở rộng bảng trạng thái"
            className="p-0.5 text-[#d8b979] hover:text-white"
          >
            <ChevronDown className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* Player Frame: Compact Mode (Gọn nhẹ cho Mobile) */}
      {playerViewMode === "compact" && (
        <div className={`pointer-events-auto absolute left-2 top-2 w-[min(calc(100vw-140px),230px)] rounded-xl p-2 z-20 ${frame}`}>
          <div className="flex items-center justify-between gap-1">
            <button
              type="button"
              onClick={() => props.onOpenView("character")}
              className="flex items-center gap-1.5 min-w-0 text-left flex-1"
              title="Mở bảng nhân vật"
            >
              <div className="h-7 w-7 shrink-0 place-items-center overflow-hidden rounded-full border border-[#d4b478] bg-[#283522] text-xs font-bold text-amber-100 flex items-center justify-center">
                {props.player?.avatar_url ? (
                  <img src={props.player.avatar_url} alt="" className="h-full w-full object-cover" />
                ) : (
                  (props.player?.username?.slice(0, 1).toUpperCase() ?? "C")
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-xs font-bold text-[#ffeac0]">{props.player?.username ?? "Cổ sư"}</div>
                <div className="text-[9px] text-[#c4a979]">Chuyển {props.player?.rank ?? 1} · {props.player?.sub_rank ?? 1}</div>
              </div>
            </button>
            <div className="flex items-center gap-0.5 shrink-0">
              <NotificationBell size="sm" />
              <button
                type="button"
                onClick={() => changePlayerViewMode("expanded")}
                title="Mở rộng chi tiết (Tứ Trụ & Buff)"
                className="p-1 text-[#d8b979] hover:text-white"
              >
                <ChevronDown className="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                onClick={() => changePlayerViewMode("minimized")}
                title="Thu nhỏ Capsule"
                className="p-1 text-[#d8b979] hover:text-white"
              >
                <Minus className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>

          {/* 2 Thanh kép HP & CN */}
          <div className="mt-1.5 space-y-1 font-sans text-[9px]">
            <div className="relative h-3 overflow-hidden rounded-full border border-[#9b6654] bg-[#2b1110]">
              <div className="h-full bg-gradient-to-r from-[#8f2c29] to-[#e45e4d] transition-[width]" style={{ width: `${pct(props.hp, props.maxHp)}%` }} />
              {props.shield > 0 && (
                <div className="absolute top-0 left-0 h-full bg-sky-400/40 border-r border-white transition-[width]" style={{ width: `${Math.min(100, (props.shield / props.maxHp) * 100)}%` }} />
              )}
              <span className="absolute inset-0 grid place-items-center font-semibold text-white text-[8px] drop-shadow">
                HP {Math.round(props.hp)}/{Math.round(props.maxHp)}
              </span>
            </div>
            <div className="relative h-3 overflow-hidden rounded-full border border-[#6b896f] bg-[#10231d]">
              <div className="h-full bg-gradient-to-r from-[#317764] to-[#74d4ab] transition-[width]" style={{ width: `${pct(props.mana, props.maxMana)}%` }} />
              <span className="absolute inset-0 grid place-items-center font-semibold text-white text-[8px] drop-shadow">
                CN {formatManaCost(props.mana)}{immortal ? "" : `/${formatManaCost(props.maxMana)}`}
              </span>
            </div>
          </div>

          {/* 2 Vạch Micro HỒN & NIỆM */}
          <div className="mt-1 grid grid-cols-2 gap-1.5 font-sans text-[8px] text-[#cdbb9b]">
            <div className="flex items-center gap-1" title={`Hồn phách: ${Math.round(soulHp)}/${Math.round(maxSoul)} (${soulTier})`}>
              <span className="text-purple-300 font-bold">H:</span>
              <div className="flex-1 h-1.5 bg-black/60 rounded-full overflow-hidden border border-purple-900/60">
                <div className="h-full bg-purple-500 transition-[width]" style={{ width: `${pct(soulHp, maxSoul)}%` }} />
              </div>
            </div>
            <div className="flex items-center gap-1" title={`Niệm lực: ${Math.round(thoughts)}/${Math.round(maxThoughts)}`}>
              <span className="text-amber-300 font-bold">N:</span>
              <div className="flex-1 h-1.5 bg-black/60 rounded-full overflow-hidden border border-amber-900/60">
                <div className="h-full bg-amber-400 transition-[width]" style={{ width: `${pct(thoughts, maxThoughts)}%` }} />
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Player Frame: Expanded Mode (Đầy đủ Tứ Trụ & Buff Tray) */}
      {playerViewMode === "expanded" && (
        <div className={`pointer-events-auto absolute left-2 top-2 w-[min(calc(100vw-140px),265px)] rounded-xl p-2.5 z-20 ${frame}`}>
          <div className="flex items-center justify-between gap-1">
            <button
              type="button"
              onClick={() => props.onOpenView("character")}
              className="flex items-center gap-2 text-left min-w-0 flex-1"
              title="Mở bảng nhân vật"
            >
              <div className="grid h-9 w-9 shrink-0 place-items-center overflow-hidden rounded-full border-2 border-[#d4b478] bg-[#283522] text-sm font-bold text-amber-100 shadow-[0_0_0_2px_#49331f]">
                {props.player?.avatar_url ? (
                  <img src={props.player.avatar_url} alt="" className="h-full w-full object-cover" />
                ) : (
                  (props.player?.username?.slice(0, 1).toUpperCase() ?? "C")
                )}
              </div>
              <div className="min-w-0 flex-1">
                <div className="truncate text-[13px] font-bold tracking-wide text-[#ffeac0]">
                  {props.player?.username ?? "Cổ sư"}
                </div>
                <div className="text-[10px] text-[#c4a979]">
                  Chuyển {props.player?.rank ?? 1} · Cảnh giới {props.player?.sub_rank ?? 1}
                </div>
              </div>
            </button>
            <div className="flex items-center gap-0.5 shrink-0">
              <NotificationBell size="sm" />
              <button
                type="button"
                onClick={() => changePlayerViewMode("compact")}
                title="Thu gọn"
                className="p-1 text-[#d8b979] hover:text-white"
              >
                <ChevronUp className="h-3.5 w-3.5" />
              </button>
              <button
                type="button"
                onClick={() => changePlayerViewMode("minimized")}
                title="Thu nhỏ Capsule"
                className="p-1 text-[#d8b979] hover:text-white"
              >
                <Minus className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>

          {/* 4 Thanh Tứ Trụ Năng Lượng */}
          <div className="mt-2 space-y-1 font-sans text-[9px]">
            {/* 1. HP (Khí Huyết) */}
            <div className="relative h-3.5 overflow-hidden rounded-full border border-[#9b6654] bg-[#2b1110]">
              <div
                className="h-full bg-gradient-to-r from-[#8f2c29] to-[#e45e4d] transition-[width]"
                style={{ width: `${pct(props.hp, props.maxHp)}%` }}
              />
              {props.shield > 0 && (
                <div
                  className="absolute top-0 left-0 h-full bg-cyan-300/40 border-r-2 border-white transition-[width]"
                  style={{ width: `${Math.min(100, (props.shield / props.maxHp) * 100)}%` }}
                />
              )}
              <span className="absolute inset-0 grid place-items-center font-semibold text-white drop-shadow-md">
                HP {Math.round(props.hp)}/{Math.round(props.maxHp)}
              </span>
            </div>
            {props.shield > 0 && (
              <div className="flex items-center justify-center gap-1 rounded border border-sky-300/50 bg-sky-950/80 px-1.5 py-0.5 text-[9px] font-bold text-sky-200">
                <Shield className="h-2.5 w-2.5" />
                Khiên {Math.round(props.shield)}
              </div>
            )}

            {/* 2. Chân Nguyên (Mana) */}
            <div className="relative h-3.5 overflow-hidden rounded-full border border-[#6b896f] bg-[#10231d]">
              {!immortal && (
                <div
                  className="h-full bg-gradient-to-r from-[#317764] to-[#74d4ab] transition-[width]"
                  style={{ width: `${pct(props.mana, props.maxMana)}%` }}
                />
              )}
              <span className="absolute inset-0 grid place-items-center font-semibold text-white drop-shadow-md">
                {manaName} {formatManaCost(props.mana)}{immortal ? "" : `/${formatManaCost(props.maxMana)}`}
              </span>
            </div>

            {/* 3. Hồn Phách (Linh Hồn) */}
            <div className="relative h-3 overflow-hidden rounded-full border border-purple-900/60 bg-[#1e102a]">
              <div
                className="h-full bg-gradient-to-r from-purple-800 to-fuchsia-500 transition-[width]"
                style={{ width: `${pct(soulHp, maxSoul)}%` }}
              />
              <span className="absolute inset-0 flex items-center justify-between px-2 text-[8px] font-semibold text-purple-100 drop-shadow">
                <span>HỒN</span>
                <span>{Math.round(soulHp)}/{Math.round(maxSoul)} [{soulTier}]</span>
              </span>
            </div>

            {/* 4. Niệm Lực (Tâm Niệm) */}
            <div className="relative h-3 overflow-hidden rounded-full border border-amber-900/60 bg-[#25180c]">
              <div
                className="h-full bg-gradient-to-r from-amber-700 to-yellow-500 transition-[width]"
                style={{ width: `${pct(thoughts, maxThoughts)}%` }}
              />
              <span className="absolute inset-0 flex items-center justify-between px-2 text-[8px] font-semibold text-amber-100 drop-shadow">
                <span>NIỆM</span>
                <span>{Math.round(thoughts)}/{Math.round(maxThoughts)}</span>
              </span>
            </div>
          </div>

          {/* Buff & Debuff Tray */}
          {playerStatuses.length > 0 && (
            <div className="mt-1.5 flex flex-wrap gap-1 border-t border-[#b89a66]/20 pt-1.5">
              {playerStatuses.map((st) => (
                <div
                  key={st.id}
                  className="relative flex items-center gap-0.5 rounded border border-[#a88952]/40 bg-black/60 px-1 py-0.5 text-[8px] text-[#f4e8cb]"
                  title={st.source ? `${st.id} (${st.source})` : st.id}
                >
                  <Sparkles className="h-2.5 w-2.5 text-amber-300" />
                  <span className="truncate max-w-[60px]">{st.id}</span>
                </div>
              ))}
            </div>
          )}

          <div className="mt-1.5 flex items-center justify-between border-t border-[#b89a66]/25 pt-1.5 text-[10px] text-[#dac69e]">
            <span>Nguyên thạch</span>
            <span className="font-sans font-semibold text-[#f5d381]">
              {props.player?.primeval_stones?.toLocaleString("vi-VN") ?? 0}
            </span>
          </div>
        </div>
      )}

      {zoneMapId === "aperture_tribulation" ? (
        <div className={`pointer-events-none absolute left-2 ${questBoxTop} w-[min(92vw,280px)] z-20 text-[#f4e8cb]`}>
          <button
            type="button"
            onClick={() => setTribulationPanelExpanded((expanded) => !expanded)}
            aria-expanded={tribulationPanelExpanded}
            className="pointer-events-auto flex items-center gap-2 rounded-full border border-amber-500/60 bg-[#16100bf2] px-3 py-2 text-xs font-semibold text-amber-200 shadow-lg md:hidden"
          >
            <Sparkles className="h-4 w-4" /> Kiếp nạn · {tribulationRemaining} quái
            {tribulationPanelExpanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
          </button>
          <div className={`${tribulationPanelExpanded ? "block" : "hidden"} pointer-events-auto mt-2 max-h-[min(48vh,320px)] overflow-y-auto rounded-2xl border border-amber-500/50 bg-[#16100bf2] p-3 shadow-[0_8px_32px_rgba(0,0,0,.6)] backdrop-blur-md space-y-2.5 md:mt-0 md:block md:max-h-none md:overflow-visible`}>
          <div className="flex items-center justify-between border-b border-amber-500/30 pb-2">
            <span className="flex items-center gap-1.5 font-serif font-bold text-xs text-amber-300">
              <Sparkles className="h-4 w-4 text-amber-400 animate-pulse" />
              Kiếp Nạn Tiên Khiếu (40×40)
            </span>
            <span className="text-[10px] px-2 py-0.5 rounded-full bg-rose-950/80 border border-rose-500/50 text-rose-300 font-mono">
              {tribulationRemaining}/{tribulationTotal} Quái / Boss Kiếp
            </span>
          </div>

          <p className="text-[10px] text-[#cdbb9b] leading-relaxed">
            Dùng <strong className="text-amber-200">WASD / Chuột</strong> di chuyển và phím <strong className="text-amber-200">Q / W / E / R</strong> tung chiêu Cổ đánh bại Boss Kiếp Nạn!
          </p>

          <div className="flex flex-col gap-2 pt-1">
            <button
              type="button"
              disabled={tribulationActionLoading}
              onClick={async () => {
                if (!activeTribulation) {
                  useGameStore.getState().addToast("Không tìm thấy đợt tai kiếp khả dụng", "error");
                  return;
                }
                setTribulationActionLoading(true);
                try {
                  await realtimeResolveTribulationVictory(activeTribulation.event.id, {});
                } finally {
                  setTribulationActionLoading(false);
                }
              }}
              className="w-full py-2 px-3 rounded-xl bg-gradient-to-r from-emerald-700 via-teal-700 to-emerald-600 hover:from-emerald-600 hover:to-teal-600 text-emerald-100 font-bold text-xs shadow-md border border-emerald-500/50 flex items-center justify-center gap-2 active:scale-95 transition cursor-pointer"
            >
              <Trophy className="h-3.5 w-3.5 text-amber-300" />
              {tribulationActionLoading ? "Đang kết toán..." : "Hoàn Tất & Kết Toán Vượt Kiếp"}
            </button>

            <button
              type="button"
              disabled={tribulationActionLoading}
              onClick={async () => {
                setTribulationActionLoading(true);
                try {
                  await realtimeExitTribulation();
                } finally {
                  setTribulationActionLoading(false);
                }
              }}
              className="w-full py-1.5 px-3 rounded-lg bg-slate-900/80 hover:bg-slate-800 text-slate-300 text-[11px] border border-slate-700/80 flex items-center justify-center gap-1.5 transition cursor-pointer"
            >
              <X className="h-3.5 w-3.5 text-rose-400" />
              Thoát Tiên Khiếu (Trở Về Vị Trí Cũ)
            </button>
          </div>
          </div>
        </div>
      ) : (
        <div className={`mmo-panel-theme mmo-paper-theme mmo-paper-surface pointer-events-auto absolute left-2 ${questBoxTop} hidden w-[min(88vw,250px)] overflow-hidden rounded-md md:block z-10 transition-all`}>
          <button type="button" onClick={() => setQuestsOpen((open) => !open)} className="mmo-paper-heading flex w-full items-center justify-between px-3 py-2 text-left text-xs font-bold">
            <span className="flex items-center gap-2"><span className="mmo-paper-seal">令</span><BookOpen className="h-3.5 w-3.5" /> Cáo thị nhiệm vụ</span><span>{questsOpen ? "−" : "+"}</span>
          </button>
          {questsOpen && <button type="button" onClick={() => props.onOpenView("achievements")} className="mmo-bamboo-action mx-2 mb-2 w-[calc(100%-1rem)] px-3 py-2 text-left text-[11px]">
            {mission ? <><div className="font-semibold text-[#3f2a19]">{mission.title ?? "Nhiệm vụ"}</div><div className="mt-1 line-clamp-2 text-[#70583c]">{mission.description ?? "Xem tiến độ nhiệm vụ"}</div><div className="mt-1 font-sans text-[10px] font-semibold text-[#875a2d]">{mission.claimable ? "Có thể nhận thưởng" : `${mission.progress ?? 0}/${mission.target ?? 0}`} · Mở sổ nhiệm vụ</div></> : "Mở sổ nhiệm vụ và thành tựu"}
          </button>}
        </div>
      )}

      {/* Target Frame: Minimized (Ẩn luôn / Thu nhỏ) */}
      {props.target && targetViewMode === "minimized" && (
        <div className="pointer-events-auto absolute left-2 top-[max(120px,18dvh)] md:left-1/2 md:top-[320px] md:-translate-x-1/2 xl:top-2 flex items-center gap-2 rounded-full border border-[#a88952]/60 bg-[#16100bf2] px-3 py-1 shadow-lg backdrop-blur-md text-xs text-[#f5dfab]">
          {props.target.kind === "player" ? (
            <Users className="h-3.5 w-3.5 text-sky-400" />
          ) : (
            <Crosshair className="h-3.5 w-3.5 text-red-400 animate-pulse" />
          )}
          <span className="font-semibold max-w-[130px] truncate">{props.target.name || (props.target.kind === "player" ? "Người chơi" : "Mục tiêu")}</span>
          <span className="font-sans text-[10px] text-red-300">{Math.round(props.target.hp)}/{Math.round(props.target.maxHp)}</span>
          <div className="flex items-center gap-1 border-l border-[#a88952]/40 pl-1.5 ml-0.5">
            <button
              type="button"
              onClick={() => changeTargetViewMode("compact")}
              title="Hiện bảng mục tiêu"
              className="p-1 text-[#d8b979] hover:text-white transition"
            >
              <Eye className="h-3.5 w-3.5" />
            </button>
            <button
              type="button"
              onClick={props.onClearTarget}
              title="Bỏ chọn mục tiêu"
              className="p-1 text-zinc-400 hover:text-red-400 transition"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}

      {/* Target Frame: Compact (Rút gọn) */}
      {props.target && targetViewMode === "compact" && (
        <div className={`pointer-events-auto absolute left-2 top-[max(120px,18dvh)] w-[min(72vw,290px)] rounded-xl p-2.5 md:left-1/2 md:top-[320px] md:w-[min(92vw,290px)] md:-translate-x-1/2 xl:top-2 ${frame}`}>
          {props.target.kind === "player" ? (
            <div>
              <div className="flex items-center gap-2">
                <div className="h-10 w-10 shrink-0 rounded-full border-2 border-sky-400/80 bg-slate-900 flex items-center justify-center text-sm font-bold text-sky-200 shadow-md">
                  {props.target.name ? props.target.name.slice(0, 1).toUpperCase() : "P"}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-1 text-xs">
                    <span className="truncate font-semibold text-sky-200">{props.target.name || "Người chơi"}</span>
                    <div className="flex items-center gap-0.5">
                      <button type="button" onClick={() => changeTargetViewMode("expanded")} title="Mở rộng chi tiết" className="rounded p-1 text-[#d8b979] hover:bg-white/10 hover:text-white transition"><ChevronDown className="h-3.5 w-3.5" /></button>
                      <button type="button" onClick={() => changeTargetViewMode("minimized")} title="Ẩn bảng mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-zinc-200 transition"><EyeOff className="h-3.5 w-3.5" /></button>
                      <button type="button" onClick={props.onClearTarget} aria-label="Bỏ mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-red-400 transition"><X className="h-3.5 w-3.5" /></button>
                    </div>
                  </div>
                  <div className="mt-0.5 flex items-center gap-1.5 text-[10px]">
                    <span className="rounded bg-sky-950/80 text-sky-300 border border-sky-500/40 px-1.5 py-0.2 text-[9px] font-semibold">
                      Cổ Sư Đồng Đạo
                    </span>
                    {props.target.pvpEnabled && <span className="rounded bg-rose-950/80 text-rose-300 border border-rose-500/40 px-1.5 py-0.2 text-[9px] font-semibold">⚔ Đang tham chiến</span>}
                  </div>
                </div>
              </div>
              <div className="mt-2 h-2 overflow-hidden rounded bg-black/70"><div className="h-full bg-gradient-to-r from-red-600 to-rose-400" style={{ width: `${pct(props.target.hp, props.target.maxHp)}%` }} /></div>
              <div className="mt-1 flex items-center justify-between font-sans text-[10px] text-[#cdbb9b]">
                <span>{Math.round(props.target.hp)}/{Math.round(props.target.maxHp)} HP{props.target.shield > 0 ? ` · Khiên ${Math.round(props.target.shield)}` : ""}</span>
                <button type="button" onClick={() => changeTargetViewMode("expanded")} className="text-[10px] text-[#d8b979] hover:underline flex items-center gap-0.5 font-serif font-medium">Hành động ▾</button>
              </div>
              <div className="mt-2 grid grid-cols-4 gap-1 pt-1.5 border-t border-[#a88952]/30">
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) void useGameStore.getState().fetchPlayerProfile(props.target.name);
                  }}
                  className="flex flex-col items-center justify-center gap-0.5 py-1 px-1 rounded bg-[#2e2319] hover:bg-[#433324] border border-[#a88952]/40 text-[#f5dfab] text-[10px] active:scale-95 transition cursor-pointer"
                  title="Xem hồ sơ người chơi"
                >
                  <Eye className="h-3.5 w-3.5 text-amber-300" />
                  <span>Hồ Sơ</span>
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) void handleInvitePlayer(props.target.name);
                  }}
                  className="flex flex-col items-center justify-center gap-0.5 py-1 px-1 rounded bg-[#1e2a3a] hover:bg-[#2a3c54] border border-sky-500/40 text-sky-200 text-[10px] active:scale-95 transition cursor-pointer"
                  title="Mời vào tổ đội"
                >
                  <Users className="h-3.5 w-3.5 text-sky-400" />
                  <span>Mời Đội</span>
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) handleWhisperPlayer(props.target.name);
                  }}
                  className="flex flex-col items-center justify-center gap-0.5 py-1 px-1 rounded bg-[#203628] hover:bg-[#2e4c3a] border border-emerald-500/40 text-emerald-200 text-[10px] active:scale-95 transition cursor-pointer"
                  title="Gửi tin nhắn riêng (Mật ngữ)"
                >
                  <ScrollText className="h-3.5 w-3.5 text-emerald-400" />
                  <span>Mật Ngữ</span>
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) void useGameStore.getState().initiateTrade(props.target.name);
                  }}
                  className="flex flex-col items-center justify-center gap-0.5 py-1 px-1 rounded bg-[#342416] hover:bg-[#4a3420] border border-amber-500/40 text-amber-200 text-[10px] active:scale-95 transition cursor-pointer"
                  title="Gửi lời mời giao dịch"
                >
                  <Store className="h-3.5 w-3.5 text-amber-400" />
                  <span>Giao Dịch</span>
                </button>
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-center gap-2">
                {targetArt && <span aria-label={`Ảnh ${props.target.name}`} role="img" className="h-11 w-11 shrink-0 rounded-md border border-[#a88952]/60 bg-[#302318] bg-no-repeat" style={{ backgroundImage: `url(${targetArt.src})`, backgroundSize: `${targetArt.columns * 100}% ${targetArt.rows * 100}%`, backgroundPosition: `${targetArt.column / Math.max(1, targetArt.columns - 1) * 100}% ${targetArt.row / Math.max(1, targetArt.rows - 1) * 100}%` }} />}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-1 text-xs">
                    <span className="truncate font-semibold text-[#f5dfab]">{props.target.name || "Quái"}</span>
                    <div className="flex items-center gap-0.5">
                      <button type="button" onClick={() => changeTargetViewMode("expanded")} title="Mở rộng thám trắc chi tiết" className="rounded p-1 text-[#d8b979] hover:bg-white/10 hover:text-white transition"><ChevronDown className="h-3.5 w-3.5" /></button>
                      <button type="button" onClick={() => changeTargetViewMode("minimized")} title="Ẩn bảng mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-zinc-200 transition"><EyeOff className="h-3.5 w-3.5" /></button>
                      <button type="button" onClick={props.onClearTarget} aria-label="Bỏ mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-red-400 transition"><X className="h-3.5 w-3.5" /></button>
                    </div>
                  </div>
                  <div className="mt-0.5 flex items-center gap-1.5 text-[10px]">
                    {targetSpec && (
                      <span className={`rounded px-1.5 py-0.2 text-[9px] font-semibold ${
                        targetSpec.type === "Boss" ? "bg-rose-950/80 text-rose-300 border border-rose-500/40" :
                        targetSpec.type === "Elite" ? "bg-purple-950/80 text-purple-300 border border-purple-500/40" :
                        "bg-amber-950/60 text-amber-300 border border-amber-500/30"
                      }`}>
                        {targetSpec.type === "Boss" ? "Thủ Lĩnh" : targetSpec.type === "Elite" ? "Tinh Anh" : "Thường"}
                      </span>
                    )}
                    {props.target.kind === "monster" && ["aggro", "chase", "attack"].includes(props.target.aiState) && <span className="font-sans text-[10px] text-[#f18c7c]">Đang truy đuổi</span>}
                  </div>
                </div>
              </div>
              <div className="mt-2 h-2 overflow-hidden rounded bg-black/70"><div className="h-full bg-[#d05348]" style={{ width: `${pct(props.target.hp, props.target.maxHp)}%` }} /></div>
              <div className="mt-1 flex items-center justify-between font-sans text-[10px] text-[#cdbb9b]">
                <span>{Math.round(props.target.hp)}/{Math.round(props.target.maxHp)} HP{props.target.shield > 0 ? ` · Khiên ${Math.round(props.target.shield)}` : ""}</span>
                <button type="button" onClick={() => changeTargetViewMode("expanded")} className="text-[10px] text-[#d8b979] hover:underline flex items-center gap-0.5 font-serif font-medium">Thám trắc ▾</button>
              </div>
            </div>
          )}
        </div>
      )}

      {/* Target Frame: Expanded (Chi tiết / Thần thức thám trắc) */}
      {props.target && targetViewMode === "expanded" && (
        <div className={`pointer-events-auto absolute left-2 top-[max(120px,18dvh)] w-[min(88vw,330px)] max-h-[62dvh] overflow-y-auto rounded-xl p-3 md:left-1/2 md:top-[300px] md:w-[min(94vw,330px)] md:-translate-x-1/2 md:max-h-none xl:top-2 ${frame}`}>
          {props.target.kind === "player" ? (
            <div>
              <div className="flex items-start gap-2.5">
                <div className="h-12 w-12 shrink-0 rounded-full border-2 border-sky-400/90 bg-slate-900 flex items-center justify-center text-lg font-bold text-sky-200 shadow-inner">
                  {props.target.name ? props.target.name.slice(0, 1).toUpperCase() : "P"}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-1 text-xs">
                    <span className="truncate font-serif font-bold text-base text-sky-200">{props.target.name || "Người chơi"}</span>
                    <div className="flex items-center gap-0.5">
                      <button type="button" onClick={() => changeTargetViewMode("compact")} title="Thu gọn bảng mục tiêu" className="rounded p-1 text-[#d8b979] hover:bg-white/10 hover:text-white transition"><ChevronUp className="h-4 w-4" /></button>
                      <button type="button" onClick={() => changeTargetViewMode("minimized")} title="Ẩn bảng mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-zinc-200 transition"><EyeOff className="h-3.5 w-3.5" /></button>
                      <button type="button" onClick={props.onClearTarget} aria-label="Bỏ mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-red-400 transition"><X className="h-4 w-4" /></button>
                    </div>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-1 text-[10px]">
                    <span className="rounded bg-sky-950/90 text-sky-300 border border-sky-500/50 px-1.5 py-0.5 font-semibold text-[9px]">
                      Cổ Sư Đồng Đạo
                    </span>
                    {props.target.pvpEnabled
                      ? <span className="rounded bg-rose-950/80 text-rose-300 border border-rose-500/40 px-1.5 py-0.5 text-[9px]">⚔ Đang tham chiến</span>
                      : <span className="rounded bg-emerald-950/70 text-emerald-300 border border-emerald-500/40 px-1.5 py-0.5 text-[9px]">Đang tắt PvP</span>}
                  </div>
                </div>
              </div>

              <div className="mt-2.5">
                <div className="relative h-2.5 overflow-hidden rounded-full border border-red-900/60 bg-black/80">
                  <div className="h-full bg-gradient-to-r from-red-700 via-rose-500 to-red-500 transition-[width]" style={{ width: `${pct(props.target.hp, props.target.maxHp)}%` }} />
                </div>
                <div className="mt-1 flex items-center justify-between font-sans text-[10px] text-[#cdbb9b]">
                  <span>Sinh mệnh: <strong className="text-red-200">{Math.round(props.target.hp)}</strong>/{Math.round(props.target.maxHp)} ({Math.round(pct(props.target.hp, props.target.maxHp))}%)</span>
                  {props.target.shield > 0 && <span className="text-sky-300">Khiên {Math.round(props.target.shield)}</span>}
                </div>
              </div>

              <div className="mt-3 grid grid-cols-2 gap-1.5">
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) void useGameStore.getState().fetchPlayerProfile(props.target.name);
                  }}
                  className="flex items-center justify-center gap-1.5 py-2 px-2 rounded-lg bg-gradient-to-r from-purple-900/60 to-purple-800/60 hover:from-purple-800 hover:to-purple-700 border border-purple-500/40 text-purple-200 text-xs font-serif font-semibold active:scale-95 transition shadow-sm cursor-pointer"
                >
                  <Eye className="h-4 w-4 text-purple-300" />
                  <span>Xem Hồ Sơ</span>
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) void handleInvitePlayer(props.target.name);
                  }}
                  className="flex items-center justify-center gap-1.5 py-2 px-2 rounded-lg bg-gradient-to-r from-blue-900/60 to-cyan-800/60 hover:from-blue-800 hover:to-cyan-700 border border-cyan-500/40 text-cyan-200 text-xs font-serif font-semibold active:scale-95 transition shadow-sm cursor-pointer"
                >
                  <Users className="h-4 w-4 text-cyan-300" />
                  <span>Mời Tổ Đội</span>
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) handleWhisperPlayer(props.target.name);
                  }}
                  className="flex items-center justify-center gap-1.5 py-2 px-2 rounded-lg bg-gradient-to-r from-emerald-900/60 to-teal-800/60 hover:from-emerald-800 hover:to-teal-700 border border-emerald-500/40 text-emerald-200 text-xs font-serif font-semibold active:scale-95 transition shadow-sm cursor-pointer"
                >
                  <ScrollText className="h-4 w-4 text-emerald-300" />
                  <span>Gửi Mật Ngữ</span>
                </button>
                <button
                  type="button"
                  onClick={() => {
                    if (props.target?.name) void useGameStore.getState().initiateTrade(props.target.name);
                  }}
                  className="flex items-center justify-center gap-1.5 py-2 px-2 rounded-lg bg-gradient-to-r from-amber-900/60 to-yellow-800/60 hover:from-amber-800 hover:to-yellow-700 border border-amber-500/40 text-amber-200 text-xs font-serif font-semibold active:scale-95 transition shadow-sm cursor-pointer"
                >
                  <Store className="h-4 w-4 text-amber-300" />
                  <span>Giao Dịch</span>
                </button>
              </div>

              <div className="mt-2.5 rounded-lg border border-[#a88952]/25 bg-[#201811]/90 p-2 text-[10px] text-[#cdbb9b]">
                <p className="italic text-center text-[#d4b478]">
                  &quot;Cùng bước trên đại đạo trường sinh, phúc họa khôn lường, kết duyên tao ngộ.&quot;
                </p>
              </div>
            </div>
          ) : (
            <div>
              <div className="flex items-start gap-2.5">
                {targetArt && <span aria-label={`Ảnh ${props.target.name}`} role="img" className="h-12 w-12 shrink-0 rounded-md border border-[#a88952]/60 bg-[#302318] bg-no-repeat shadow-inner" style={{ backgroundImage: `url(${targetArt.src})`, backgroundSize: `${targetArt.columns * 100}% ${targetArt.rows * 100}%`, backgroundPosition: `${targetArt.column / Math.max(1, targetArt.columns - 1) * 100}% ${targetArt.row / Math.max(1, targetArt.rows - 1) * 100}%` }} />}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-1 text-xs">
                    <span className="truncate font-serif font-bold text-sm text-[#f5dfab]">{props.target.name || "Quái"}</span>
                    <div className="flex items-center gap-0.5">
                      <button type="button" onClick={() => changeTargetViewMode("compact")} title="Thu gọn bảng mục tiêu" className="rounded p-1 text-[#d8b979] hover:bg-white/10 hover:text-white transition"><ChevronUp className="h-4 w-4" /></button>
                      <button type="button" onClick={() => changeTargetViewMode("minimized")} title="Ẩn bảng mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-zinc-200 transition"><EyeOff className="h-3.5 w-3.5" /></button>
                      <button type="button" onClick={props.onClearTarget} aria-label="Bỏ mục tiêu" className="rounded p-1 text-zinc-400 hover:bg-white/10 hover:text-red-400 transition"><X className="h-4 w-4" /></button>
                    </div>
                  </div>
                  <div className="mt-1 flex flex-wrap items-center gap-1 text-[10px]">
                    {targetSpec && (
                      <span className={`rounded px-1.5 py-0.5 font-semibold text-[9px] ${
                        targetSpec.type === "Boss" ? "bg-rose-950/90 text-rose-300 border border-rose-500/50" :
                        targetSpec.type === "Elite" ? "bg-purple-950/90 text-purple-300 border border-purple-500/50" :
                        "bg-amber-950/70 text-amber-300 border border-amber-500/40"
                      }`}>
                        {targetSpec.type === "Boss" ? "Thủ Lĩnh" : targetSpec.type === "Elite" ? "Tinh Anh" : "Thường"}
                      </span>
                    )}
                    {targetPathLabel && (
                      <span className="rounded bg-[#203628] px-1.5 py-0.5 text-[9px] text-emerald-200 border border-emerald-600/30">
                        {targetPathLabel}
                      </span>
                    )}
                    {props.target.kind === "monster" && ["aggro", "chase", "attack"].includes(props.target.aiState) && (
                      <span className="font-sans text-[10px] text-[#f18c7c] animate-pulse">Đang truy đuổi</span>
                    )}
                  </div>
                </div>
              </div>

              <div className="mt-2.5">
                <div className="relative h-2.5 overflow-hidden rounded-full border border-red-900/60 bg-black/80">
                  <div className="h-full bg-gradient-to-r from-red-700 via-rose-500 to-red-500 transition-[width]" style={{ width: `${pct(props.target.hp, props.target.maxHp)}%` }} />
                </div>
                <div className="mt-1 flex items-center justify-between font-sans text-[10px] text-[#cdbb9b]">
                  <span>Sinh mệnh: <strong className="text-red-200">{Math.round(props.target.hp)}</strong>/{Math.round(props.target.maxHp)} ({Math.round(pct(props.target.hp, props.target.maxHp))}%)</span>
                  {props.target.shield > 0 && <span className="text-sky-300">Khiên {Math.round(props.target.shield)}</span>}
                </div>
              </div>

              <div className="mt-2 flex items-center justify-between rounded-md border border-[#a88952]/30 bg-[#251b14]/70 px-2 py-1 text-[10px]">
                <span className="flex items-center gap-1.5 text-[#f3ddac]">
                  <Eye className="h-3 w-3 text-amber-300" />
                  <span>Thần thức:</span>
                  <strong className={investigationStatus.level === "high" ? "text-emerald-300" : investigationStatus.level === "medium" ? "text-amber-300" : "text-rose-400"}>
                    {investigationStatus.badge}
                  </strong>
                </span>
                <span className="text-[9px] text-[#bda87f]">Cổ Giới Chi Điển</span>
              </div>

              <div className="mt-2 grid grid-cols-3 gap-1 text-center font-sans">
                <div className="rounded-lg border border-[#a88952]/30 bg-[#2c2016]/80 p-1.5">
                  <div className="flex items-center justify-center gap-0.5 text-[10px] text-zinc-400">
                    <Swords className="h-3 w-3 text-amber-400" />
                    <span>Tấn công</span>
                  </div>
                  <div className="mt-0.5 font-bold text-xs text-amber-100">
                    {investigationStatus.level === "low" ? (
                      <span className="text-zinc-500 font-normal">???</span>
                    ) : (
                      targetSpec ? targetSpec.base_attack : "—"
                    )}
                  </div>
                </div>
                <div className="rounded-lg border border-[#a88952]/30 bg-[#2c2016]/80 p-1.5">
                  <div className="flex items-center justify-center gap-0.5 text-[10px] text-zinc-400">
                    <Shield className="h-3 w-3 text-sky-400" />
                    <span>Giáp / Thủ</span>
                  </div>
                  <div className="mt-0.5 font-bold text-xs text-sky-100">
                    {investigationStatus.level === "low" ? (
                      <span className="text-zinc-500 font-normal">???</span>
                    ) : targetSpec ? (
                      targetSpec.base_defense > 1 ? targetSpec.base_defense : `${Math.round(targetSpec.base_defense * 100)}%`
                    ) : "—"}
                  </div>
                </div>
                <div className="rounded-lg border border-[#a88952]/30 bg-[#2c2016]/80 p-1.5">
                  <div className="flex items-center justify-center gap-0.5 text-[10px] text-zinc-400">
                    <PawPrint className="h-3 w-3 text-emerald-400" />
                    <span>Thuần hóa</span>
                  </div>
                  <div className="mt-0.5 font-bold text-xs text-emerald-200">
                    {investigationStatus.level === "low" ? (
                      <span className="text-zinc-500 font-normal">???</span>
                    ) : targetSpec?.is_capturable ? (
                      `${Math.round((targetSpec.capture_chance ?? 0.2) * 100)}%`
                    ) : (
                      "Không"
                    )}
                  </div>
                </div>
              </div>

              <div className="mt-2 rounded-lg border border-[#a88952]/25 bg-[#201811]/90 p-2 text-[10px]">
                <div className="flex items-center justify-between text-zinc-400 mb-1">
                  <span className="flex items-center gap-1 font-semibold text-[#e8d7b3]">
                    <Sparkles className="h-3 w-3 text-amber-400" /> Cổ Trùng & Tuyệt Kỹ:
                  </span>
                  <span className="text-[9px] text-zinc-500">
                    {investigationStatus.level === "low"
                      ? "Bị che giấu"
                      : targetGuList.length > 0 ? `${targetGuList.length} Cổ` : "Hoang dã"}
                  </span>
                </div>
                {investigationStatus.level === "low" ? (
                  <p className="text-[10px] text-rose-300/80 italic">
                    Thần thức bị áp chế, không thể thấu thị Cổ trùng (Cần nâng chỉ số Trinh Sát để dò xét).
                  </p>
                ) : investigationStatus.level === "medium" && targetGuList.length > 2 ? (
                  <div className="flex flex-wrap gap-1">
                    {targetGuList.slice(0, 2).map((gu, idx) => (
                      <span key={idx} className="rounded bg-[#3d2c1e] border border-[#a88952]/40 px-1.5 py-0.5 text-[9px] text-[#f4e2bf]">
                        {gu}
                      </span>
                    ))}
                    <span className="rounded bg-black/50 border border-[#a88952]/20 px-1.5 py-0.5 text-[9px] text-amber-300/80 italic">
                      +{targetGuList.length - 2} Cổ ẩn tàng (???)
                    </span>
                  </div>
                ) : targetGuList.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {targetGuList.map((gu, idx) => (
                      <span key={idx} className="rounded bg-[#3d2c1e] border border-[#a88952]/40 px-1.5 py-0.5 text-[9px] text-[#f4e2bf]">
                        {gu}
                      </span>
                    ))}
                  </div>
                ) : (
                  <p className="text-[10px] text-[#9a8d75] italic">
                    {targetSpec?.description || "Dã thú tự nhiên, dựa vào bản năng hung bạo và linh khí thiên địa."}
                  </p>
                )}
              </div>

              {targetGuList.length > 0 && targetSpec?.description && investigationStatus.level !== "low" && (
                <p className="mt-1.5 px-1 text-[9px] text-[#9a8d75] italic line-clamp-2">
                  {targetSpec.description}
                </p>
              )}
            </div>
          )}
        </div>
      )}

      <div className={`pointer-events-auto absolute right-2 top-2 w-[124px] rounded-2xl p-1.5 sm:w-[172px] ${frame} transition-all duration-200 z-20`}>
        <div className="flex items-center justify-between gap-1 px-1 text-[11px] text-[#f3ddac]">
          <span className="flex min-w-0 items-center gap-1 truncate" title={props.zoneId}>
            <MapPin className="h-3 w-3 shrink-0 text-amber-400" />
            <span className="truncate">
              {zoneMapId === "aperture_tribulation"
                ? "⚡ Tiên Khiếu (40×40)"
                : zoneMapId.replaceAll("_", " ") || "Khu vực"}
            </span>
          </span>
          <div className="flex items-center gap-0.5 shrink-0">
            <span className="flex items-center gap-0.5 text-[9px] font-sans font-semibold text-emerald-400" title="Chỉ số Trinh Sát / Thần Thức hiện tại">
              <Eye className="h-2.5 w-2.5" /> {investigationStat.toFixed(1)}
            </span>
            <button
              type="button"
              onClick={() => setMinimapCollapsed((prev) => !prev)}
              title={minimapCollapsed ? "Mở radar bản đồ" : "Thu gọn radar bản đồ"}
              className="p-0.5 text-[#d8b979] hover:text-white"
            >
              {minimapCollapsed ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronUp className="h-3.5 w-3.5" />}
            </button>
          </div>
        </div>
        {!minimapCollapsed ? (
          <>
            <div className="relative mx-auto mt-1.5 aspect-square w-full overflow-hidden rounded-full border-[5px] border-[#80613d] bg-[#213827] shadow-[inset_0_0_35px_#08160c,0_2px_8px_#000]" aria-label={`Radar lân cận, ${nearby.length} đối tượng`}>
              <div className="absolute inset-0 opacity-40 transition-all duration-300" style={{ backgroundImage: "linear-gradient(#8baf7b33 1px, transparent 1px), linear-gradient(90deg, #8baf7b33 1px, transparent 1px)", backgroundSize: `${Math.max(10, Math.min(28, Math.round(360 / radarRange)))}px ${Math.max(10, Math.min(28, Math.round(360 / radarRange)))}px` }} />
              <div className="absolute left-1/2 top-0 h-full w-px bg-[#e7d3a2]/20" /><div className="absolute left-0 top-1/2 h-px w-full bg-[#e7d3a2]/20" />
              {nearby.map((entity) => {
                const entSpec = entity.templateId ? monsterDictionary[entity.templateId] : null;
                const isBoss = entSpec?.type === "Boss" || entity.name?.includes("Boss") || entity.name?.includes("Thú Vương");
                const isElite = entSpec?.type === "Elite" || entity.name?.includes("Tinh Anh");

                // If investigation is low, can't distinguish Boss/Elite on radar (appears as simple red dot)
                const canDistinguish = investigationStat >= 2.0 || scoutingPower >= ((entSpec?.rank ?? 1) * 20);

                let dotColor = "bg-[#e07553] shadow-[0_0_4px_#e07553]";
                let dotBorder = "border border-black/70";
                let dotSize = "h-2.5 w-2.5";

                if (entity.kind === "player") {
                  dotColor = "bg-[#79c6e5]";
                } else if (canDistinguish && isBoss) {
                  dotColor = "bg-purple-500 shadow-[0_0_8px_#c084fc]";
                  dotBorder = "border-2 border-amber-300";
                  dotSize = "h-3.5 w-3.5";
                } else if (canDistinguish && isElite) {
                  dotColor = "bg-amber-400 shadow-[0_0_6px_#fbbf24]";
                  dotBorder = "border border-amber-100";
                  dotSize = "h-3 w-3";
                }

                return (
                  <button
                    key={entity.id}
                    type="button"
                    onClick={() => props.onSelectTarget?.(entity.id)}
                    className={`absolute -translate-x-1/2 -translate-y-1/2 rounded-full transition-transform hover:scale-150 cursor-pointer ${dotSize} ${dotBorder} ${dotColor}`}
                    style={{ left: `${50 + (entity.x - props.position.x) / radarRange * 46}%`, top: `${50 + (entity.y - props.position.y) / radarRange * 46}%` }}
                    title={`${entity.name || (entity.kind === "monster" ? "Quái" : "Người chơi")}: ${Math.hypot(entity.x - props.position.x, entity.y - props.position.y).toFixed(1)} ô`}
                  />
                );
              })}

              {/* Structures on Radar — one marker list shared with the world layer */}
              {markers.map((marker) => {
                const dx = marker.x - props.position.x;
                const dy = marker.y - props.position.y;
                if (Math.abs(dx) > radarRange || Math.abs(dy) > radarRange) return null;
                const spec = radarMarkerSpec(marker);
                return (
                  <div
                    key={`${marker.kind}:${marker.id}`}
                    className={`absolute flex -translate-x-1/2 -translate-y-1/2 items-center justify-center ${spec.className}`}
                    style={{ left: `${50 + (dx / radarRange) * 46}%`, top: `${50 + (dy / radarRange) * 46}%` }}
                    title={`${spec.title}: ${Math.hypot(dx, dy).toFixed(1)} ô`}
                  >
                    {spec.glyph}
                  </div>
                );
              })}
              <span className="absolute left-1/2 top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 border-white bg-[#f3cd76] shadow-[0_0_8px_#ffda77]" />
              <span className="absolute left-1/2 top-1 -translate-x-1/2 font-sans text-[9px] text-[#ead7a8]">B</span>
            </div>
            <div className="mt-1 flex items-center justify-between font-sans text-[10px] text-[#cabf9e]">
              <div className="flex items-center gap-1 font-sans">
                <span className="font-semibold text-amber-200">{cameraZoom.toFixed(1)}x</span>
                <span className="text-zinc-500">·</span>
                <span>{nearby.length} gần</span>
              </div>
              <span className="flex items-center gap-0.5">
                <button
                  type="button"
                  aria-label="Phóng to"
                  onClick={handleZoomIn}
                  disabled={cameraZoom >= 1.6}
                  className="h-6 w-6 flex items-center justify-center rounded bg-black/50 border border-[#a88952]/50 hover:bg-[#a88952]/20 active:scale-95 text-[#f5dfab] disabled:opacity-30 disabled:pointer-events-none transition"
                  title="Phóng to bản đồ & camera"
                >
                  <Plus className="h-3 w-3" />
                </button>
                <button
                  type="button"
                  aria-label="Thu nhỏ"
                  onClick={handleZoomOut}
                  disabled={cameraZoom <= 0.6}
                  className="h-6 w-6 flex items-center justify-center rounded bg-black/50 border border-[#a88952]/50 hover:bg-[#a88952]/20 active:scale-95 text-[#f5dfab] disabled:opacity-30 disabled:pointer-events-none transition"
                  title="Thu nhỏ bản đồ & camera"
                >
                  <Minus className="h-3 w-3" />
                </button>
              </span>
            </div>
          </>
        ) : (
          <div className="mt-0.5 text-center font-sans text-[9px] text-[#cabf9e]">
            {props.position.x.toFixed(0)}, {props.position.y.toFixed(0)} · {nearby.length} gần
          </div>
        )}
      </div>

      {/* Mobile Virtual Joystick for 360-degree smooth movement */}
      {props.onMoveJoystick && (
        <div className="pointer-events-auto absolute bottom-4 left-4 z-20 md:hidden">
          <VirtualJoystick onMove={props.onMoveJoystick} />
        </div>
      )}

      {props.statusText && <div className="absolute bottom-[125px] left-1/2 -translate-x-1/2 rounded border border-violet-300/30 bg-black/75 px-3 py-1 font-sans text-[11px] text-violet-100 sm:bottom-[140px] md:bottom-36 shadow-lg z-20">{props.statusText}</div>}
      {props.castProgress !== null && <div className="absolute bottom-[90px] left-1/2 w-[min(90vw,320px)] -translate-x-1/2 sm:bottom-[105px] md:bottom-28 z-20"><div className="mb-1 text-center text-xs text-amber-100">{props.castName}</div><div className="h-2 overflow-hidden rounded bg-black/70 shadow-inner"><div className="h-full bg-amber-400" style={{ width: `${pct(props.castProgress, 1)}%` }} /></div></div>}

      {/* Proximity Interaction Prompts (Nguyên Tuyền, Tiên Tuyền, Chiến Lợi Phẩm, Truyền Tống Trận, Tông Môn) */}
      {(closestDrop || closestVein || closestTp || closestSect) && (
        <div className="pointer-events-auto absolute bottom-[145px] left-1/2 -translate-x-1/2 sm:bottom-[160px] md:bottom-44 flex flex-col items-center gap-1.5 z-20 max-w-[min(94vw,360px)]">
          {closestSect && (() => {
            const isMySect = closestSect.sect?.isMine ?? false;
            const isShielded = Boolean(closestSect.sect?.shieldUntil && closestSect.sect.shieldUntil > Date.now());

            return (
              <div className={`flex items-center justify-between gap-2.5 rounded-2xl border ${isMySect ? "border-blue-500/70 bg-[#0d1f35e6]" : "border-amber-500/70 bg-[#251608e6]"} px-3.5 py-1.5 shadow-2xl backdrop-blur-md animate-in fade-in slide-in-from-bottom-2 w-full`}>
                <div className="flex items-center gap-2 min-w-0">
                  <div className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ${isMySect ? "bg-blue-500/20 border-blue-400/40 text-blue-300" : "bg-amber-500/20 border-amber-400/40 text-amber-300"} border`}>
                    <Crown className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-1.5 text-xs font-serif font-bold text-amber-200 truncate">
                      <span>{closestSect.name}</span>
                      <span className="rounded bg-black/60 border border-amber-500/30 px-1 text-[9px] text-amber-300">
                        Cấp {closestSect.sect?.level ?? 1}
                      </span>
                    </div>
                    <div className="text-[10px] text-zinc-300 truncate">
                      {isMySect ? "Trụ sở môn phái" : isShielded ? "🛡️ Đang trong thời gian bảo hộ" : `Cự ly: ${distanceTo(closestSect).toFixed(1)} ô`}
                    </div>
                  </div>
                </div>
                {isMySect ? (
                  <button
                    type="button"
                    onClick={() => props.onOpenView("sect")}
                    className="shrink-0 rounded-xl bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 border border-blue-400/50 px-3 py-1 text-xs font-serif font-bold text-white shadow-md active:scale-95 transition cursor-pointer"
                  >
                    Trụ Sở
                  </button>
                ) : isShielded ? (
                  <span className="shrink-0 rounded-xl bg-zinc-800/80 border border-zinc-700 px-2.5 py-1 text-[11px] font-serif text-zinc-400">
                    Bảo Hộ
                  </span>
                ) : (
                  <button
                    type="button"
                    onClick={async () => {
                      if (closestSect) {
                        await attackSect(closestSect.id);
                        void fetchZoneMarkers(zoneRegionId);
                      }
                    }}
                    className="shrink-0 rounded-xl bg-gradient-to-r from-rose-600 to-red-600 hover:from-rose-500 hover:to-red-500 border border-rose-400/50 px-3 py-1 text-xs font-serif font-bold text-white shadow-md active:scale-95 transition cursor-pointer"
                  >
                    Khiêu Chiến
                  </button>
                )}
              </div>
            );
          })()}
          {closestDrop && (
            <div className="flex items-center justify-between gap-2.5 rounded-2xl border border-amber-500/70 bg-[#1e150ee6] px-3.5 py-1.5 shadow-2xl backdrop-blur-md animate-in fade-in slide-in-from-bottom-2 w-full">
              <div className="flex items-center gap-2 min-w-0">
                <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-amber-500/20 border border-amber-400/40 text-amber-300">
                  <Sparkles className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <div className="text-xs font-serif font-bold text-amber-200 truncate">
                    🎁 Chiến Lợi Phẩm
                  </div>
                  <div className="text-[10px] text-[#cfbe9e] truncate">
                    {closestDrop.drop?.killer_name ? `Hạ bởi: ${closestDrop.drop.killer_name}` : "Rơi thạch, tài nguyên"} ({distanceTo(closestDrop).toFixed(1)} ô)
                  </div>
                </div>
              </div>
              <button
                type="button"
                onClick={async () => {
                  if (closestDrop) await lootGroundDrop(closestDrop.interaction?.action.targetId ?? closestDrop.id);
                }}
                className="shrink-0 rounded-xl bg-gradient-to-r from-amber-600 to-yellow-600 hover:from-amber-500 hover:to-yellow-500 border border-amber-400/50 px-3 py-1 text-xs font-serif font-bold text-white shadow-md active:scale-95 transition cursor-pointer"
              >
                Nhặt
              </button>
            </div>
          )}

          {closestVein && (
            <div className="flex items-center justify-between gap-2.5 rounded-2xl border border-emerald-500/70 bg-[#0e1d15e6] px-3.5 py-1.5 shadow-2xl backdrop-blur-md animate-in fade-in slide-in-from-bottom-2 w-full">
              <div className="flex items-center gap-2 min-w-0">
                <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-emerald-500/20 border border-emerald-400/40 text-emerald-300">
                  <Gem className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5 text-xs font-serif font-bold text-emerald-200 truncate">
                    <span>{closestVein.name}</span>
                    <span className="rounded bg-emerald-950/80 border border-emerald-500/40 px-1 text-[9px] text-emerald-300">
                      {closestVein.vein?.rarity || "Thường"}
                    </span>
                  </div>
                  <div className="text-[10px] text-[#a7d4ba] truncate">
                    Tích: <strong className="text-amber-300">{closestVein.vein?.accumulated_stones?.toLocaleString("vi-VN") ?? 0}</strong> NT ({distanceTo(closestVein).toFixed(1)} ô)
                  </div>
                </div>
              </div>
              <button
                type="button"
                onClick={async () => {
                  const veinId = closestVein?.interaction?.action.targetId;
                  if (veinId) await harvestVein(veinId);
                }}
                className="shrink-0 rounded-xl bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 border border-emerald-400/50 px-3 py-1 text-xs font-serif font-bold text-white shadow-md active:scale-95 transition cursor-pointer"
              >
                Thu hoạch
              </button>
            </div>
          )}

          {closestTp && (
            <div className="flex items-center justify-between gap-2.5 rounded-2xl border border-cyan-500/70 bg-[#0e1724e6] px-3.5 py-1.5 shadow-2xl backdrop-blur-md animate-in fade-in slide-in-from-bottom-2 w-full">
              <div className="flex items-center gap-2 min-w-0">
                <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-cyan-500/20 border border-cyan-400/40 text-cyan-300">
                  <Compass className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <div className="text-xs font-serif font-bold text-cyan-200 truncate">
                    🌀 {closestTp.name}
                  </div>
                  <div className="text-[10px] text-cyan-300/80 truncate">
                    Cự ly: {distanceTo(closestTp).toFixed(1)} ô
                  </div>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setShowTeleportDirectory(true)}
                className="shrink-0 rounded-xl bg-gradient-to-r from-cyan-600 to-blue-600 hover:from-cyan-500 hover:to-blue-500 border border-cyan-400/50 px-3 py-1 text-xs font-serif font-bold text-white shadow-md active:scale-95 transition cursor-pointer"
              >
                Danh bạ
              </button>
            </div>
          )}
        </div>
      )}

      {/* Mobile Combat Hub: 2-Row Cluster + Big Attack Button (Tối ưu ngón cái tay phải, không bị che/tràn màn hình) */}
      <div className="pointer-events-auto absolute bottom-2 right-2 flex items-end gap-1.5 md:hidden z-20 select-none">
        {hotbarSlots.length > 0 && <div className={`grid grid-cols-4 gap-1 rounded-xl p-1 shadow-xl ${frame}`}>
          {hotbarSlots.map((slot) => {
            const skill = props.skills.find((entry) => entry.slot === slot);
            const guSlot = props.guSlots.find((entry) => entry.slot === slot);
            const remaining = props.cooldownRemaining[slot] ?? 0;
            const isAimingThis = props.aimingSlot === slot;
            const formActive = props.nearby.find((entity) => entity.isSelf)?.formActive === true;
            const isFormActivation = skill?.form_activation === true;
            const isFormAbility = skill?.form_ability === true;
            const isToggled =
              skill?.is_toggle &&
              playerStatuses.some(
                (st) =>
                  (typeof st === "string" ? st : st.id) === skill.id ||
                  (typeof st === "string" ? st : st.id) === guSlot?.guId,
              );
            const spec = guSlot?.guId ? guSpecFor(guSlot.guId, guSlot.rank ?? undefined) : undefined;
            const isPassive = !skill && !!guSlot?.guId && (
              spec?.gu_type === "Passive" ||
              spec?.gu_type === "Buff" ||
              !spec?.effects?.blocks?.some((b) => b.trigger === "ON_ACTION")
            );
            return (
              <button
                key={slot}
                type="button"
                disabled={!skill}
                onClick={() => skill && props.onCast(slot)}
                title={
                  skill
                    ? `${guSlot?.name ?? skill.name} · ${MODE_LABEL[skill.mode]} · Tầm: ${skill.range}m · (${formatManaCost(skill.mana_cost)} nguyên)`
                    : isPassive
                      ? `${guSlot?.name} · Bị động`
                      : guSlot?.name
                        ? `${guSlot.name} · Chưa có chiêu realtime`
                        : "Trống"
                }
                className={`relative flex h-10 w-10 flex-col justify-between overflow-hidden rounded-lg border p-0.5 text-center transition active:scale-90 ${
                  isAimingThis
                    ? "border-amber-400 bg-gradient-to-b from-[#4a3512] to-[#241a08] text-amber-100 ring-2 ring-amber-400 shadow-[0_0_12px_rgba(251,191,36,0.8)] scale-105 z-20 animate-pulse"
                    : isToggled || (isFormActivation && formActive)
                      ? "border-emerald-400 bg-gradient-to-b from-[#1b4332] to-[#081c15] text-[#d8f3dc] ring-2 ring-emerald-500/70"
                      : skill
                        ? "border-[#9d865b] bg-gradient-to-b from-[#254636] to-[#162d22] text-[#f9e9c8] active:border-[#dfc382]"
                        : "border-[#544b3b]/60 bg-gradient-to-b from-[#242019]/90 to-[#151310]/90 text-[#867d6c] opacity-70"
                }`}
              >
                <span className="absolute left-0 top-0 rounded-br bg-black/85 px-1 py-0.2 font-mono text-[8px] font-bold text-[#f4d695] border-r border-b border-[#8f6d40]/50 leading-none z-10">
                  {hotbarKeyLabel(slot, hotbarKeybinds)}
                </span>
                {isToggled || (isFormActivation && formActive) ? (
                  <span className="absolute right-0 top-0 rounded-bl bg-emerald-700 px-1 py-0.2 font-sans text-[7px] font-bold text-white z-10 animate-pulse">
                    ON
                  </span>
                ) : skill ? (
                  <span
                    className={`absolute right-0 top-0 rounded-bl px-1 py-0.2 font-sans text-[7.5px] font-bold leading-none z-10 border-l border-b ${MODE_BADGE[skill.mode].bg} ${MODE_BADGE[skill.mode].border} ${MODE_BADGE[skill.mode].color}`}
                  >
                    {MODE_BADGE[skill.mode].icon}
                  </span>
                ) : isPassive ? (
                  <span className="absolute right-0 top-0 rounded-bl bg-purple-950/90 border-l border-b border-purple-500/70 px-1 py-0.2 font-sans text-[7px] text-purple-300 leading-none z-10">
                    ⚡
                  </span>
                ) : null}
                {guSlot?.guId && getGuRankColor(guSlot.rank) && (
                  <div
                    className={`pointer-events-none absolute inset-0 rounded-lg border ${getGuRankColor(guSlot.rank).border}/40 z-0`}
                  />
                )}
                <div className="flex-1 flex flex-col items-center justify-center w-full mt-0.5 px-0.5 z-10">
                  {guSlot?.guId && getGuIconUrl(guSlot.guId, guSlot.name) && (
                    <img
                      src={getGuIconUrl(guSlot.guId, guSlot.name)!}
                      alt=""
                      className="h-4 w-4 object-contain drop-shadow"
                    />
                  )}
                  <span className="line-clamp-1 text-[7.5px] font-semibold leading-tight tracking-tight">
                    {isFormActivation ? "Biến thân" : isFormAbility ? `Dạng: ${skill?.name}` : guSlot?.name ?? (skill?.name || "Trống")}
                  </span>
                </div>
                <span className="font-mono text-[7px] text-cyan-200/90 truncate tracking-tighter">
                  {skill ? `${formatManaCost(skill.mana_cost)}` : isPassive ? "Bị động" : guSlot?.guId ? "—" : ""}
                </span>
                {remaining > 0.05 && (
                  <span className="absolute inset-0 z-20 grid place-items-center bg-black/80 font-sans font-bold text-xs text-amber-200 backdrop-blur-[1px]">
                    {remaining.toFixed(1)}s
                  </span>
                )}
              </button>
            );
          })}
        </div>}

        {/* Big Attack Button */}
        <button
          type="button"
          onClick={props.onAttack}
          title={props.rightClickAction === "skill" ? `Cổ ${props.rightClickSlot}` : "Đánh thường"}
          className="relative flex h-14 w-14 shrink-0 flex-col items-center justify-center rounded-full border-2 border-[#e4be74] bg-gradient-to-b from-[#694123] to-[#301a0b] text-[#fff0cd] shadow-[0_0_15px_rgba(228,190,116,0.35)] active:scale-90 transition-transform overflow-hidden"
        >
          <Swords className="h-6 w-6 text-amber-200" />
          <span className="text-[10px] font-bold text-amber-100">
            {props.rightClickAction === "skill" ? `Cổ ${props.rightClickSlot}` : "Đánh"}
          </span>
        </button>
      </div>

      {/* Aiming Status Pill (Hiện khi đang ngắm chiêu trên cả PC và Mobile) */}
      {props.aimingSlot && (
        <div className="pointer-events-auto absolute bottom-20 sm:bottom-24 md:bottom-28 left-1/2 -translate-x-1/2 z-30 flex items-center gap-2 rounded-full border border-amber-400/90 bg-[#140e0ae6] px-3.5 py-1.5 shadow-[0_0_18px_rgba(251,191,36,0.6)] backdrop-blur-md text-xs text-amber-200">
          <span className="h-2 w-2 rounded-full bg-amber-400 animate-ping" />
          <span className="font-semibold text-amber-100">
            {(() => {
              const s = props.skills.find((entry) => entry.slot === props.aimingSlot);
              const g = props.guSlots.find((entry) => entry.slot === props.aimingSlot);
              return `${g?.name ?? s?.name ?? `Ô ${props.aimingSlot}`}: Nhấp chọn ${s?.mode === "TARGET" ? "mục tiêu" : s?.mode === "DIRECTION" ? "hướng" : "vị trí"}`;
            })()}
          </span>
          <button
            type="button"
            onClick={props.onCancelAim}
            className="ml-1 flex items-center gap-1 rounded-full bg-red-600/90 hover:bg-red-500 px-2.5 py-0.5 text-[10px] font-bold text-white transition shadow cursor-pointer active:scale-95"
            title="Hủy ngắm chiêu (Esc)"
          >
            <X className="h-3 w-3" />
            <span>Hủy (Esc)</span>
          </button>
        </div>
      )}

      {/* Desktop Hotbar (Thanh ngang tiêu chuẩn cho PC / Tablet) */}
      <div className={`pointer-events-auto absolute bottom-2 left-1/2 hidden md:flex max-w-[min(98vw,480px)] -translate-x-1/2 items-center gap-1 sm:gap-1.5 md:gap-2 overflow-x-auto rounded-xl p-1 sm:p-1.5 md:p-2 sm:max-w-none sm:w-auto shadow-2xl ${frame}`}>
        <button
          type="button"
          onClick={props.onAttack}
          title={props.rightClickAction === "skill" ? `Chuột phải · Cổ ${props.rightClickSlot}` : "Chuột phải / Phím R · Đánh thường"}
          className="relative flex h-12 min-w-12 flex-col items-center justify-center rounded-lg border border-[#c9a35b] bg-gradient-to-b from-[#694123] to-[#422915] text-[#fff0cd] hover:brightness-110 active:scale-95 transition shadow sm:h-16 sm:w-20 md:h-20 md:w-24 shrink-0 overflow-hidden"
        >
          <span className="absolute left-0 top-0 rounded-br bg-black/85 px-1 py-0.5 font-mono text-[8px] sm:text-[9.5px] font-bold text-[#f4d695] border-r border-b border-[#8f6d40]/50 leading-none z-10">
            R-Click
          </span>
          <Swords className="h-4 w-4 sm:h-5 sm:w-5 md:h-6 md:w-6 text-amber-200 mt-1 sm:mt-0" />
          <span className="text-[9px] sm:text-xs md:text-sm font-semibold mt-0.5">
            {props.rightClickAction === "skill" ? `Cổ ${props.rightClickSlot}` : "Đánh"}
          </span>
        </button>
        {hotbarSlots.map((slot) => {
          const skill = props.skills.find((entry) => entry.slot === slot);
          const guSlot = props.guSlots.find((entry) => entry.slot === slot);
          const remaining = props.cooldownRemaining[slot] ?? 0;
          const isAimingThis = props.aimingSlot === slot;
          const formActive = props.nearby.find((entity) => entity.isSelf)?.formActive === true;
          const isFormActivation = skill?.form_activation === true;
          const isFormAbility = skill?.form_ability === true;
          const isToggled =
            skill?.is_toggle &&
            playerStatuses.some(
              (st) =>
                (typeof st === "string" ? st : st.id) === skill.id ||
                (typeof st === "string" ? st : st.id) === guSlot?.guId,
            );
          const spec = guSlot?.guId ? guSpecFor(guSlot.guId, guSlot.rank ?? undefined) : undefined;
          const isPassive = !skill && !!guSlot?.guId && (
            spec?.gu_type === "Passive" ||
            spec?.gu_type === "Buff" ||
            !spec?.effects?.blocks?.some((b) => b.trigger === "ON_ACTION")
          );
          return (
            <button
              key={slot}
              type="button"
              disabled={!skill}
              onClick={() => skill && props.onCast(slot)}
              onMouseEnter={() => props.onHoverSkill?.(slot)}
              onMouseLeave={() => props.onHoverSkill?.(null)}
              title={
                skill
                  ? `${guSlot?.name ?? skill.name} · Phím [${hotbarKeyLabel(slot, hotbarKeybinds)}]
Cơ chế: ${MODE_BADGE[skill.mode].icon} ${MODE_BADGE[skill.mode].label}
Tầm xa: ${skill.range.toFixed(1)} ô${skill.radius_units ? ` · Bán kính: ${skill.radius_units} ô` : ""}${skill.shape ? ` · Dạng: ${skill.shape}` : ""}
Tiêu hao: ${formatManaCost(skill.mana_cost)} ${manaName.toLowerCase()} · Hồi: ${(skill.cooldown_ticks / 20).toFixed(1)}s${isToggled ? " · (ĐANG DUY TRÌ)" : ""}`
                  : isPassive
                    ? `${guSlot?.name} · Cổ bị động (tự động kích hoạt hiệu ứng)`
                    : guSlot?.name
                      ? `${guSlot.name} · Chưa hỗ trợ realtime`
                      : `Ô ${slot} chưa gán Cổ`
              }
              className={`relative flex h-12 min-w-[46px] flex-col justify-between overflow-hidden rounded-lg border px-1 pt-3 pb-0.5 text-center transition shadow sm:h-16 sm:w-20 sm:pt-4 sm:pb-1 md:h-20 md:w-24 shrink-0 ${
                isAimingThis
                  ? "border-amber-400 bg-gradient-to-b from-[#4a3512] to-[#241a08] text-amber-100 ring-2 ring-amber-400 shadow-[0_0_14px_rgba(251,191,36,0.8)] scale-105 z-20 animate-pulse"
                  : isToggled || (isFormActivation && formActive)
                    ? "border-emerald-400 bg-gradient-to-b from-[#1b4332] to-[#081c15] text-[#d8f3dc] ring-2 ring-emerald-500/70 shadow-[0_0_12px_rgba(52,211,153,0.4)]"
                    : skill
                      ? "border-[#9d865b] bg-gradient-to-b from-[#254636] to-[#162d22] text-[#f9e9c8] hover:border-[#dfc382] hover:brightness-110 active:scale-95"
                      : "border-[#544b3b] bg-gradient-to-b from-[#242019] to-[#151310] text-[#867d6c] opacity-80"
              }`}
            >
              <span className="absolute left-0 top-0 rounded-br bg-black/85 px-1.5 py-0.5 font-mono text-[9px] sm:text-[11px] md:text-xs font-bold text-[#f4d695] border-r border-b border-[#8f6d40]/50 leading-none z-10">
                {hotbarKeyLabel(slot, hotbarKeybinds)}
              </span>
              {isToggled || (isFormActivation && formActive) ? (
                <span className="absolute right-0 top-0 rounded-bl bg-emerald-700 px-1 py-0.5 font-sans text-[7px] sm:text-[9px] font-bold text-white border-l border-b border-emerald-400 z-10 animate-pulse">
                  ON
                </span>
              ) : skill ? (
                <span
                  className={`absolute right-0 top-0 rounded-bl px-1 py-0.5 font-sans text-[7px] sm:text-[8.5px] font-bold border-l border-b leading-tight z-10 flex items-center gap-0.5 ${MODE_BADGE[skill.mode].bg} ${MODE_BADGE[skill.mode].border} ${MODE_BADGE[skill.mode].color}`}
                >
                  <span>{MODE_BADGE[skill.mode].icon}</span>
                  <span className="hidden sm:inline">{MODE_BADGE[skill.mode].label}</span>
                  <span className="sm:hidden">{MODE_BADGE[skill.mode].shortLabel}</span>
                </span>
              ) : isPassive ? (
                <span className="absolute right-0 top-0 rounded-bl px-1 py-0.5 font-sans text-[7px] sm:text-[8.5px] font-bold border-l border-b leading-tight z-10 bg-purple-950/85 border-purple-500/70 text-purple-300">
                  ⚡ Bị động
                </span>
              ) : null}
              {guSlot?.guId && getGuRankColor(guSlot.rank) && (
                <div
                  className={`pointer-events-none absolute inset-0 rounded-xl border ${getGuRankColor(guSlot.rank).border}/40 z-0`}
                />
              )}
              <div className="flex-1 flex flex-col items-center justify-center w-full px-0.5 z-10">
                {guSlot?.guId && getGuIconUrl(guSlot.guId, guSlot.name) && (
                  <img
                    src={getGuIconUrl(guSlot.guId, guSlot.name)!}
                    alt=""
                    className="h-5 w-5 sm:h-7 sm:w-7 object-contain drop-shadow mb-0.5"
                  />
                )}
                <span className="line-clamp-1 text-[8px] sm:text-[10px] md:text-xs font-semibold leading-tight tracking-tight">
                  {isFormActivation ? "Biến thân" : isFormAbility ? `Dạng: ${skill?.name}` : guSlot?.name ?? skill?.name ?? "Chưa gán"}
                </span>
              </div>
              <span className="font-mono text-[7px] sm:text-[9.5px] md:text-[10px] text-cyan-200/90 truncate tracking-tighter">
                {skill ? `${formatManaCost(skill.mana_cost)} nguyên` : isPassive ? "Bị động" : guSlot?.guId ? "Chưa hỗ trợ" : "Cổ trùng"}
              </span>
              {remaining > 0.05 && (
                <span className="absolute inset-0 z-20 grid place-items-center bg-black/75 font-sans font-bold text-xs sm:text-base md:text-lg text-amber-200 backdrop-blur-[1px]">
                  {remaining.toFixed(1)}s
                </span>
              )}
            </button>
          );
        })}
      </div>

      {props.menuOpen && (
        <>
          <div
            className="pointer-events-auto fixed inset-0 z-40 bg-black/60 backdrop-blur-[2px] transition-opacity"
            onClick={props.onCloseMenu}
            aria-label="Đóng danh mục"
          />
          <div
            className={`pointer-events-auto fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50 w-[min(94vw,360px)] max-h-[min(82dvh,600px)] overflow-y-auto rounded-2xl p-3.5 shadow-2xl md:fixed md:left-auto md:top-auto md:translate-x-0 md:translate-y-0 md:bottom-[70px] md:right-3 md:w-[350px] md:max-h-[min(80dvh,540px)] ${frame}`}
            role="dialog"
            aria-label="Danh mục chức năng"
          >
            <div className="mb-2.5 flex items-center justify-between border-b border-[#b89a66]/30 pb-2 text-sm font-semibold text-[#f6d99f]">
              <div className="flex items-center gap-2">
                <Menu className="h-4 w-4 text-[#d8b979]" />
                <span>Danh mục Cổ Giới</span>
              </div>
              <div className="flex items-center gap-1.5">
                <button
                  type="button"
                  onClick={() => {
                    props.onCloseMenu();
                    props.onOpenPanel("settings");
                  }}
                  title="Mở Cài đặt & Tiện ích"
                  className="flex items-center gap-1 rounded-md border border-[#8f6d40]/40 bg-[#3a291d]/80 px-2 py-0.5 text-[10.5px] font-sans text-[#ead6ad] hover:bg-[#624329] hover:text-white transition"
                >
                  <Settings2 className="h-3 w-3 text-amber-300" />
                  <span>Cài đặt</span>
                </button>
                <button
                  type="button"
                  onClick={props.onCloseMenu}
                  aria-label="Đóng danh mục"
                  className="rounded-md p-1 hover:bg-white/10 text-[#d8b979] hover:text-white transition"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            </div>
            {menuGroups.map((group) => (
              <section key={group.title} className="mb-3 last:mb-0">
                <h3 className="mb-1 text-[10px] uppercase tracking-[.15em] text-[#bda87f]">{group.title}</h3>
                <div className="grid grid-cols-2 gap-1">
                  {group.items.map((item) => {
                    const disabled = !!item.minRank && (props.player?.rank ?? 0) < item.minRank;
                    return (
                      <button
                        key={item.label}
                        type="button"
                        disabled={disabled}
                        onClick={() => {
                          if (item.panel) props.onOpenPanel(item.panel);
                          else if (item.view) props.onOpenView(item.view);
                          else if (item.special === "divination") {
                            props.onCloseMenu();
                            openDivinationModal();
                          } else if (item.special === "quick-actions") {
                            props.onCloseMenu();
                            setShowQuickActionModal(true);
                          } else if (item.special === "teleport-directory") {
                            props.onCloseMenu();
                            setShowTeleportDirectory(true);
                          } else if (item.special === "teleport-build") {
                            props.onCloseMenu();
                            setShowTeleportBuild(true);
                          }
                        }}
                        className="flex min-h-9 items-center gap-2 rounded-md border border-[#8f6d40]/45 bg-[#3a291d]/70 px-2 text-left text-[11px] text-[#ead8b6] hover:bg-[#624329] disabled:cursor-not-allowed disabled:opacity-40"
                        title={disabled ? `Cần đạt ${item.minRank} chuyển` : item.label}
                      >
                        <item.icon className="h-3.5 w-3.5 shrink-0" />
                        <span className="truncate">{item.label}</span>
                      </button>
                    );
                  })}
                </div>
              </section>
            ))}
          </div>
        </>
      )}

      <nav
        aria-label="Chức năng MMO"
        className={`pointer-events-auto absolute right-2 ${minimapCollapsed ? "top-[48px]" : "top-[168px] sm:top-[215px]"} w-[54px] md:w-auto flex flex-col md:flex-row gap-1 rounded-xl p-1 transition-all duration-200 md:top-auto md:bottom-2 md:right-2 z-20 ${frame}`}
      >
        <div className="flex md:hidden items-center justify-between px-0.5 pb-0.5 border-b border-[#a88952]/20">
          <button
            type="button"
            onClick={() => props.onOpenPanel("settings")}
            className="flex items-center gap-0.5 text-[7.5px] uppercase font-bold text-[#bda87f] hover:text-amber-300 transition"
            title="Mở Cài đặt & Tiện ích"
          >
            <Settings2 className="h-2.5 w-2.5 text-amber-400" />
            <span>{mobileNavCollapsed ? "Menu" : "Tiện ích"}</span>
          </button>
          <button
            type="button"
            onClick={() => setMobileNavCollapsed((prev) => !prev)}
            title={mobileNavCollapsed ? "Hiện nút chức năng" : "Ẩn nút chức năng"}
            className="p-0.5 text-[#d8b979] hover:text-white"
          >
            {mobileNavCollapsed ? <ChevronDown className="h-3 w-3" /> : <ChevronUp className="h-3 w-3" />}
          </button>
        </div>

        {(!mobileNavCollapsed || (typeof window !== "undefined" && window.innerWidth >= 768)) && (
          actionItems.map((item) => (
            <button
              key={item.key}
              type="button"
              onClick={item.action}
              title={item.label}
              aria-label={item.label}
              aria-expanded={item.key === "menu" ? props.menuOpen : undefined}
              className={`flex w-full md:w-10 h-9 md:h-10 flex-col items-center justify-center gap-0.5 rounded-lg border text-[7.5px] md:text-[8px] transition active:scale-95 ${
                props.activePanel === item.key || (item.key === "menu" && props.menuOpen)
                  ? "border-[#f4d894] bg-[#805e32] text-white shadow-sm"
                  : "border-[#8f6d40] bg-[#3a291d] text-[#e7d5ae] hover:bg-[#624329]"
              }`}
            >
              <item.icon className="h-3.5 w-3.5 shrink-0" />
              <span className="w-full text-center truncate px-0.5 font-sans leading-none">{item.label}</span>
            </button>
          ))
        )}
      </nav>

      {showQuickActionModal && (
        <QuickActionCustomizer onClose={() => setShowQuickActionModal(false)} />
      )}

      {showTeleportDirectory && props.player && (
        <TeleportDirectoryModal
          player={props.player}
          onClose={() => setShowTeleportDirectory(false)}
        />
      )}
      {showTeleportBuild && props.player && (
        <TeleportBuildModal
          player={props.player}
          onClose={() => setShowTeleportBuild(false)}
        />
      )}
    </div>
  );
}
