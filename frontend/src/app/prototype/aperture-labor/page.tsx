'use client';

// THROWAWAY UI PROTOTYPE for Wayfinder #480:
// "Three radically different variants of the Immortal Aperture Variant Labor & Immortals UI, switchable via ?variant=, on /prototype/aperture-labor."

import React, { Suspense, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import {
  AlertTriangle,
  Crown,
  Flame,
  Hammer,
  MapPin,
  Shield,
  Sparkles,
  Sprout,
  Swords,
  Users,
  Wheat,
} from 'lucide-react';
import { PrototypeSwitcher } from '@/components/prototype/PrototypeSwitcher';
import {
  INITIAL_EVENTS,
  INITIAL_NPCS,
  INITIAL_POINTS,
  INITIAL_RACES,
  LaborIntensity,
  NPCRole,
  RaceKey,
  RaceState,
  ResourcePointMock,
  VariantEventMock,
  VariantNPCMock,
} from './mockData';

interface SharedStateProps {
  races: RaceState[];
  points: ResourcePointMock[];
  events: VariantEventMock[];
  npcs: VariantNPCMock[];
  essence: number;
  stones: number;
  onSetLabor: (race: RaceKey, intensity: LaborIntensity) => void;
  onToggleRelief: (race: RaceKey) => void;
  onFeedFood: (race: RaceKey) => void;
  onBuildOrUpgrade: (race: RaceKey) => void;
  onResolveEvent: (id: string, mode: 'PACIFY' | 'SUPPRESS') => void;
  onAscendNPC: (id: string) => void;
  onChangeNPCRole: (id: string, role: NPCRole) => void;
  onToggleFormation: (id: string) => void;
  onAssignPointRace: (pointId: string, race: RaceKey) => void;
}

const LABOR_OPTIONS: { key: LaborIntensity; label: string; mult: string }[] = [
  { key: 'LOW', label: 'Nhẹ', mult: '0.7x' },
  { key: 'NORMAL', label: 'Thường', mult: '1.0x' },
  { key: 'HIGH', label: 'Cao', mult: '1.4x' },
  { key: 'EXPLOIT', label: 'Bóc lột', mult: '1.8x' },
];

const ROLE_OPTIONS: { key: NPCRole; label: string }[] = [
  { key: 'NONE', label: 'Tự do / Xuất chiến' },
  { key: 'REFINER', label: 'Luyện Cổ' },
  { key: 'MINER', label: 'Khai Khoáng' },
  { key: 'FARMER', label: 'Canh Tác' },
  { key: 'STEWARD', label: 'Quản Đốc' },
];

function computePointYieldPreview(pt: ResourcePointMock, races: RaceState[]) {
  if (!pt.assignedRace || pt.assignedPop <= 0) {
    return { mult: 1.0, status: 'Tự nhiên (chưa gán dân)', isAffinity: false };
  }
  const r = races.find((x) => x.race === pt.assignedRace);
  const laborMult =
    r?.laborIntensity === 'LOW'
      ? 0.7
      : r?.laborIntensity === 'HIGH'
      ? 1.4
      : r?.laborIntensity === 'EXPLOIT'
      ? 1.8
      : 1.0;
  const isAffinity = pt.assignedRace === pt.workerRace;
  const popMult = isAffinity ? 1 + 0.5 * Math.min(1, pt.assignedPop / 500) : 0.8;
  return {
    mult: Number((laborMult * popMult).toFixed(2)),
    status: isAffinity ? 'Đúng sở trường (+50% tối đa)' : 'Sai sở trường (0.8x)',
    isAffinity,
  };
}

// ============================================================================
// VARIANT A: Sổ Bộ Tiên Khiếu (Imperial Ledger & 9-Race Master-Detail)
// ============================================================================
function VariantA(props: SharedStateProps) {
  const [selectedRace, setSelectedRace] = useState<RaceKey>('HAIRY_MAN');
  const [detailTab, setDetailTab] = useState<'overview' | 'points' | 'events' | 'npcs'>('overview');

  const activeRace = props.races.find((r) => r.race === selectedRace) ?? props.races[0];
  const raceEvents = props.events.filter((e) => !e.resolved && e.race === selectedRace);
  const raceNPCs = props.npcs.filter((n) => n.race === selectedRace);

  return (
    <div className="mx-auto max-w-7xl p-4 pb-20">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-amber-500/30 bg-stone-900/90 px-4 py-3">
        <div>
          <h1 className="text-lg font-bold text-amber-200">Tiên Khiếu Dị Nhân — Sổ Bộ Quản Trị</h1>
          <p className="text-xs text-stone-400">
            Master-Detail theo 9 chủng tộc · Tiên Nguyên: {props.essence.toLocaleString()} · Tiên Nguyên Thạch: {props.stones.toLocaleString()}
          </p>
        </div>
        <div className="flex items-center gap-2 text-xs">
          <span className="rounded-full bg-red-950/70 px-3 py-1 text-red-300 border border-red-500/30">
            Biến cố chưa xử lý: {props.events.filter((e) => !e.resolved).length}
          </span>
          <span className="rounded-full bg-purple-950/70 px-3 py-1 text-purple-300 border border-purple-500/30">
            Cổ Tiên Dị Nhân: {props.npcs.filter((n) => n.isImmortal).length}
          </span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-12">
        {/* Left Rail: 9-Race Ledger */}
        <div className="lg:col-span-4 rounded-xl border border-stone-800 bg-stone-900/80 divide-y divide-stone-800">
          {props.races.map((r) => {
            const hasEvent = props.events.some((e) => !e.resolved && e.race === r.race);
            const daoOk = r.daoMarksCurrent >= r.daoMarksRequired;
            const isSel = r.race === selectedRace;
            return (
              <button
                key={r.race}
                type="button"
                onClick={() => setSelectedRace(r.race)}
                className={`w-full text-left px-3.5 py-2.5 transition flex items-center justify-between ${
                  isSel ? 'bg-amber-500/15 border-l-2 border-amber-400' : 'hover:bg-stone-800/60'
                }`}
              >
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-semibold text-sm text-stone-100">{r.name}</span>
                    {r.settlementType ? (
                      <span className="rounded bg-stone-800 px-1.5 py-0.5 text-[10px] text-amber-300">
                        {r.settlementType}
                      </span>
                    ) : (
                      <span className="rounded bg-stone-800/50 px-1.5 py-0.5 text-[10px] text-stone-500">
                        Chưa mở
                      </span>
                    )}
                    {hasEvent && <AlertTriangle className="h-3.5 w-3.5 text-red-400" />}
                    {r.starving && <span className="text-[10px] text-orange-400 font-semibold">ĐÓI</span>}
                  </div>
                  <div className="text-[11px] text-stone-400 mt-0.5">
                    {r.population.toLocaleString()} / {r.capacity.toLocaleString()} dân · {daoOk ? 'Đạo Ngân Đủ' : 'Thiếu Đạo Ngân'}
                  </div>
                </div>
                <div className="text-right text-[11px]">
                  <div className="text-stone-300">{r.guMasterCount} Sư / {r.guImmortalCount} Tiên</div>
                  <div className={r.rebellionRate >= 40 ? 'text-red-400' : 'text-emerald-400'}>
                    Loạn {r.rebellionRate}%
                  </div>
                </div>
              </button>
            );
          })}
        </div>

        {/* Right Workspace */}
        <div className="lg:col-span-8 rounded-xl border border-stone-800 bg-stone-900/80 p-4 space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-stone-800 pb-3">
            <div>
              <h2 className="text-base font-bold text-amber-200">{activeRace.name} — {activeRace.specialty}</h2>
              <p className="text-xs text-stone-400">
                Đạo Ngân {activeRace.daoPath}: {activeRace.daoMarksCurrent.toLocaleString()} / {activeRace.daoMarksRequired.toLocaleString()} · Sinh sản: {activeRace.reproRateLabel} · Tỉ lệ Cổ Tiên: {activeRace.immortalRateLabel}
              </p>
            </div>
            <div className="flex gap-1 bg-stone-950 p-1 rounded-lg border border-stone-800 text-xs">
              {(['overview', 'points', 'events', 'npcs'] as const).map((t) => (
                <button
                  key={t}
                  type="button"
                  onClick={() => setDetailTab(t)}
                  className={`px-2.5 py-1 rounded ${
                    detailTab === t ? 'bg-amber-500/20 text-amber-200 font-semibold' : 'text-stone-400 hover:text-stone-200'
                  }`}
                >
                  {t === 'overview' ? 'Thành Trì & Lương' : t === 'points' ? 'Điểm Tài Nguyên' : t === 'events' ? `Biến Cố (${raceEvents.length})` : `Cổ Sư/Tiên (${raceNPCs.length})`}
                </button>
              ))}
            </div>
          </div>

          {detailTab === 'overview' && (
            <div className="space-y-4 text-xs">
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                <div className="rounded-lg bg-stone-950/80 p-3 border border-stone-800">
                  <div className="text-stone-400">Dân số / Sức chứa</div>
                  <div className="text-sm font-bold text-stone-100 mt-1">
                    {activeRace.population.toLocaleString()} / {activeRace.capacity.toLocaleString()}
                  </div>
                </div>
                <div className="rounded-lg bg-stone-950/80 p-3 border border-stone-800">
                  <div className="text-stone-400">Lương thực kho</div>
                  <div className={`text-sm font-bold mt-1 ${activeRace.starving ? 'text-red-400' : 'text-emerald-300'}`}>
                    {activeRace.foodStored.toLocaleString()} / {activeRace.foodStorageCap.toLocaleString()}
                  </div>
                </div>
                <div className="rounded-lg bg-stone-950/80 p-3 border border-stone-800">
                  <div className="text-stone-400">Hạnh phúc / Áp bức</div>
                  <div className="text-sm font-bold text-stone-100 mt-1">
                    {activeRace.happiness}% / {activeRace.oppression}%
                  </div>
                </div>
                <div className="rounded-lg bg-stone-950/80 p-3 border border-stone-800">
                  <div className="text-stone-400">Vị trí Thành Trì</div>
                  <div className="text-sm font-bold text-amber-300 mt-1">
                    {activeRace.settlementPos ? `(${activeRace.settlementPos.x}, ${activeRace.settlementPos.y})` : 'Chưa đặt'}
                  </div>
                </div>
              </div>

              <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-stone-800 bg-stone-950/60 p-3">
                <div>
                  <div className="font-semibold text-stone-200">Cường độ lao động</div>
                  <div className="text-stone-400">Tác động sản lượng, hao sinh khí điểm tài nguyên & lương thực</div>
                </div>
                <div className="flex gap-1.5">
                  {LABOR_OPTIONS.map((opt) => (
                    <button
                      key={opt.key}
                      type="button"
                      onClick={() => props.onSetLabor(activeRace.race, opt.key)}
                      className={`px-2.5 py-1 rounded border ${
                        activeRace.laborIntensity === opt.key
                          ? 'border-amber-400 bg-amber-500/20 text-amber-200 font-semibold'
                          : 'border-stone-700 text-stone-400 hover:bg-stone-800'
                      }`}
                    >
                      {opt.label} ({opt.mult})
                    </button>
                  ))}
                </div>
              </div>

              <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-stone-800 bg-stone-950/60 p-3">
                <div className="flex items-center gap-3">
                  <button
                    type="button"
                    onClick={() => props.onBuildOrUpgrade(activeRace.race)}
                    className="rounded-lg bg-amber-600/80 hover:bg-amber-500 px-3 py-1.5 font-semibold text-stone-950"
                  >
                    {activeRace.settlementType ? `Nâng cấp Thành Trì (${activeRace.settlementType})` : 'Xây Thành Trì VILLAGE (0 dân)'}
                  </button>
                  <button
                    type="button"
                    onClick={() => props.onFeedFood(activeRace.race)}
                    className="rounded-lg border border-emerald-500/40 bg-emerald-950/50 hover:bg-emerald-900/50 px-3 py-1.5 text-emerald-200"
                  >
                    Nạp Lương Thực (+2,000)
                  </button>
                </div>
                <label className="flex items-center gap-2 cursor-pointer text-stone-300">
                  <input
                    type="checkbox"
                    checked={activeRace.autoEssenceRelief}
                    onChange={() => props.onToggleRelief(activeRace.race)}
                    className="accent-amber-400"
                  />
                  Cứu trợ bằng Tiên Nguyên khi hết lương thực
                </label>
              </div>
            </div>
          )}

          {detailTab === 'points' && (
            <div className="space-y-2 text-xs">
              {props.points.map((pt) => {
                const prev = computePointYieldPreview(pt, props.races);
                return (
                  <div key={pt.id} className="flex flex-wrap items-center justify-between gap-2 rounded-lg border border-stone-800 bg-stone-950/70 p-3">
                    <div>
                      <div className="font-semibold text-stone-100">{pt.name} ({pt.path})</div>
                      <div className="text-stone-400">
                        Sở trường: <span className="text-amber-300">{pt.workerRace}</span> · Đang gán: {pt.assignedRace ?? 'Trống'} ({pt.assignedPop} dân) ·{' '}
                        <span className={prev.isAffinity ? 'text-emerald-400' : 'text-orange-300'}>{prev.status}</span> · Hệ số: <b>{prev.mult}x</b>
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => props.onAssignPointRace(pt.id, activeRace.race)}
                      className="rounded border border-amber-500/40 px-2.5 py-1 text-amber-200 hover:bg-amber-500/20"
                    >
                      Chuyển {activeRace.name} vào điểm này
                    </button>
                  </div>
                );
              })}
            </div>
          )}

          {detailTab === 'events' && (
            <div className="space-y-2 text-xs">
              {raceEvents.length === 0 ? (
                <div className="text-stone-400 py-6 text-center">Không có biến cố nổi loạn nào cho {activeRace.name}.</div>
              ) : (
                raceEvents.map((ev) => (
                  <div key={ev.id} className="rounded-lg border border-red-500/40 bg-red-950/20 p-3 flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <div className="font-bold text-red-300">[{ev.eventType}] · {ev.createdAt}</div>
                      <div className="text-stone-200 mt-0.5">{ev.description}</div>
                    </div>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => props.onResolveEvent(ev.id, 'PACIFY')}
                        className="rounded bg-emerald-600/80 hover:bg-emerald-500 px-2.5 py-1 font-semibold text-stone-950"
                      >
                        An dân (Tốn lương/thạch)
                      </button>
                      <button
                        type="button"
                        onClick={() => props.onResolveEvent(ev.id, 'SUPPRESS')}
                        className="rounded bg-red-600/80 hover:bg-red-500 px-2.5 py-1 font-semibold text-white"
                      >
                        Trấn áp (Tăng áp bức)
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          )}

          {detailTab === 'npcs' && (
            <div className="space-y-2 text-xs">
              {raceNPCs.length === 0 ? (
                <div className="text-stone-400 py-6 text-center">Chưa có Cổ Sư / Cổ Tiên của {activeRace.name}.</div>
              ) : (
                raceNPCs.map((npc) => (
                  <div key={npc.id} className="rounded-lg border border-stone-800 bg-stone-950/70 p-3 flex flex-wrap items-center justify-between gap-2">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-stone-100">{npc.name}</span>
                        <span className={`rounded px-1.5 py-0.5 text-[10px] font-semibold ${npc.isImmortal ? 'bg-purple-500/25 text-purple-200' : 'bg-stone-800 text-stone-300'}`}>
                          {npc.isImmortal ? `Cổ Tiên R${npc.rank}` : `Cổ Sư R${npc.rank}.${npc.subRank}`}
                        </span>
                        {npc.formationSlot > 0 && (
                          <span className="rounded bg-amber-500/20 px-1.5 py-0.5 text-[10px] text-amber-200">Đội hình #{npc.formationSlot}</span>
                        )}
                      </div>
                      <div className="text-stone-400 mt-0.5">
                        {npc.path} · Trung thành: {npc.loyalty}% · Đạo Ngân: {npc.daoMarks} · Cổ: {npc.guEquipped.join(', ')}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <select
                        value={npc.role}
                        onChange={(e) => props.onChangeNPCRole(npc.id, e.target.value as NPCRole)}
                        className="rounded border border-stone-700 bg-stone-900 px-2 py-1 text-stone-200"
                      >
                        {ROLE_OPTIONS.map((ro) => (
                          <option key={ro.key} value={ro.key}>{ro.label}</option>
                        ))}
                      </select>
                      {!npc.isImmortal && npc.rank === 5 && (
                        <button
                          type="button"
                          onClick={() => props.onAscendNPC(npc.id)}
                          className="rounded bg-purple-600/80 hover:bg-purple-500 px-2.5 py-1 font-semibold text-white"
                        >
                          Hộ pháp Thăng Tiên
                        </button>
                      )}
                      {npc.isImmortal && (
                        <button
                          type="button"
                          onClick={() => props.onToggleFormation(npc.id)}
                          className="rounded border border-amber-500/40 px-2.5 py-1 text-amber-200 hover:bg-amber-500/20"
                        >
                          {npc.formationSlot > 0 ? 'Rút đội hình' : 'Xuất chiến'}
                        </button>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// VARIANT B: Bản Đồ Phân Khu & Chỉ Huy Trực Quan (Spatial Map-First)
// ============================================================================
function VariantB(props: SharedStateProps) {
  const [selectedCell, setSelectedCell] = useState<{ x: number; y: number }>({ x: 1, y: 1 });
  const activeEvents = props.events.filter((e) => !e.resolved);

  const cellSettlement = props.races.find(
    (r) => r.settlementPos?.x === selectedCell.x && r.settlementPos?.y === selectedCell.y
  );
  const cellPoint = props.points.find(
    (p) => p.pos.x === selectedCell.x && p.pos.y === selectedCell.y
  );

  return (
    <div className="mx-auto max-w-7xl p-4 pb-20 space-y-4">
      {/* Top Crisis Strip */}
      {activeEvents.length > 0 && (
        <div className="rounded-xl border border-red-500/40 bg-red-950/30 p-3 flex flex-wrap items-center justify-between gap-2 text-xs">
          <div className="flex items-center gap-2 text-red-200 font-semibold">
            <AlertTriangle className="h-4 w-4 text-red-400" />
            <span>Cảnh báo khẩn: {activeEvents[0].race} — {activeEvents[0].description}</span>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => props.onResolveEvent(activeEvents[0].id, 'PACIFY')}
              className="rounded bg-emerald-600 px-2.5 py-1 font-semibold text-stone-950"
            >
              An dân ngay
            </button>
            <button
              type="button"
              onClick={() => props.onResolveEvent(activeEvents[0].id, 'SUPPRESS')}
              className="rounded bg-red-600 px-2.5 py-1 font-semibold text-white"
            >
              Trấn áp
            </button>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-12">
        {/* Left: 5x5 Spatial Aperture Grid */}
        <div className="lg:col-span-7 rounded-xl border border-stone-800 bg-stone-900/90 p-4">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-bold text-amber-200 flex items-center gap-2">
              <MapPin className="h-4 w-4" /> Bản Đồ Ô Đất Tiên Khiếu (5×5)
            </h2>
            <span className="text-[11px] text-stone-400">Chọn ô để xem Thành Trì hoặc Điểm Tài Nguyên</span>
          </div>
          <div className="grid grid-cols-5 gap-2">
            {Array.from({ length: 25 }).map((_, idx) => {
              const x = idx % 5;
              const y = Math.floor(idx / 5);
              const s = props.races.find((r) => r.settlementPos?.x === x && r.settlementPos?.y === y);
              const p = props.points.find((pt) => pt.pos.x === x && pt.pos.y === y);
              const isSel = selectedCell.x === x && selectedCell.y === y;
              return (
                <button
                  key={`${x}-${y}`}
                  type="button"
                  onClick={() => setSelectedCell({ x, y })}
                  className={`h-20 rounded-lg border p-1.5 text-left transition flex flex-col justify-between ${
                    isSel
                      ? 'border-amber-400 bg-amber-500/20'
                      : s
                      ? 'border-purple-500/40 bg-purple-950/30 hover:bg-purple-900/30'
                      : p
                      ? 'border-emerald-500/40 bg-emerald-950/25 hover:bg-emerald-900/25'
                      : 'border-stone-800 bg-stone-950/60 hover:bg-stone-800/50'
                  }`}
                >
                  <div className="text-[10px] text-stone-500 font-mono">({x},{y})</div>
                  {s ? (
                    <div>
                      <div className="text-[11px] font-bold text-purple-200 truncate">{s.name}</div>
                      <div className="text-[10px] text-purple-300/80">{s.settlementType}</div>
                    </div>
                  ) : p ? (
                    <div>
                      <div className="text-[11px] font-bold text-emerald-200 truncate">{p.name}</div>
                      <div className="text-[10px] text-stone-400">{p.assignedRace ?? 'Trống'}</div>
                    </div>
                  ) : (
                    <div className="text-[10px] text-stone-600">Đất trống</div>
                  )}
                </button>
              );
            })}
          </div>
        </div>

        {/* Right: Contextual Cell Inspector + Immortal Command Dock */}
        <div className="lg:col-span-5 space-y-4">
          <div className="rounded-xl border border-stone-800 bg-stone-900/90 p-4 text-xs space-y-3">
            <h3 className="font-bold text-amber-200">Thanh Tra Ô ({selectedCell.x}, {selectedCell.y})</h3>
            {cellSettlement ? (
              <div className="space-y-2">
                <div className="text-sm font-bold text-stone-100">{cellSettlement.name} · {cellSettlement.settlementType}</div>
                <div className="text-stone-400">
                  Dân: {cellSettlement.population.toLocaleString()} / {cellSettlement.capacity.toLocaleString()} · Lương: {cellSettlement.foodStored.toLocaleString()}
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {LABOR_OPTIONS.map((o) => (
                    <button
                      key={o.key}
                      type="button"
                      onClick={() => props.onSetLabor(cellSettlement.race, o.key)}
                      className={`px-2 py-1 rounded border ${
                        cellSettlement.laborIntensity === o.key ? 'border-amber-400 bg-amber-500/20 text-amber-200' : 'border-stone-700 text-stone-400'
                      }`}
                    >
                      {o.label}
                    </button>
                  ))}
                </div>
                <div className="flex gap-2 pt-1">
                  <button
                    type="button"
                    onClick={() => props.onBuildOrUpgrade(cellSettlement.race)}
                    className="rounded bg-amber-600/80 px-2.5 py-1 font-semibold text-stone-950"
                  >
                    Nâng cấp
                  </button>
                  <button
                    type="button"
                    onClick={() => props.onToggleRelief(cellSettlement.race)}
                    className="rounded border border-stone-700 px-2.5 py-1 text-stone-200"
                  >
                    Cứu trợ Tiên Nguyên: {cellSettlement.autoEssenceRelief ? 'BẬT' : 'TẮT'}
                  </button>
                </div>
              </div>
            ) : cellPoint ? (
              <div className="space-y-2">
                <div className="text-sm font-bold text-emerald-200">{cellPoint.name}</div>
                <div className="text-stone-400">
                  Sở trường: {cellPoint.workerRace} · Đang gán: {cellPoint.assignedRace ?? 'Không'} · Sinh khí: {cellPoint.vitality}%
                </div>
                <div className="flex flex-wrap gap-1">
                  {props.races.filter((r) => r.population > 0).map((r) => (
                    <button
                      key={r.race}
                      type="button"
                      onClick={() => props.onAssignPointRace(cellPoint.id, r.race)}
                      className={`px-2 py-1 rounded border text-[11px] ${
                        cellPoint.assignedRace === r.race ? 'border-emerald-400 bg-emerald-500/20 text-emerald-200' : 'border-stone-700 text-stone-400'
                      }`}
                    >
                      {r.name}
                    </button>
                  ))}
                </div>
              </div>
            ) : (
              <div className="text-stone-400">Ô đất trống — có thể đặt Thành Trì mới cho tộc chưa có chỗ ở (Mặc Nhân, Nấm Nhân).</div>
            )}
          </div>

          <div className="rounded-xl border border-purple-500/30 bg-stone-900/90 p-4 text-xs space-y-2">
            <h3 className="font-bold text-purple-200 flex items-center gap-2">
              <Crown className="h-4 w-4" /> Cổ Tiên & Cổ Sư Hộ Pháp
            </h3>
            {props.npcs.map((n) => (
              <div key={n.id} className="flex items-center justify-between rounded bg-stone-950/70 px-2.5 py-2 border border-stone-800">
                <div>
                  <span className="font-semibold text-stone-100">{n.name}</span>{' '}
                  <span className="text-purple-300">(R{n.rank})</span>
                  <div className="text-[11px] text-stone-400">{n.role} · {n.formationSlot > 0 ? `Xuất chiến #${n.formationSlot}` : 'Thủ khiếu'}</div>
                </div>
                {n.isImmortal ? (
                  <button
                    type="button"
                    onClick={() => props.onToggleFormation(n.id)}
                    className="rounded border border-purple-400/40 px-2 py-1 text-purple-200"
                  >
                    {n.formationSlot > 0 ? 'Rút' : 'Xuất chiến'}
                  </button>
                ) : n.rank === 5 ? (
                  <button
                    type="button"
                    onClick={() => props.onAscendNPC(n.id)}
                    className="rounded bg-purple-600 px-2 py-1 text-white font-semibold"
                  >
                    Thăng Tiên
                  </button>
                ) : null}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// ============================================================================
// VARIANT C: Cung Đình Dị Nhân (4-Pillar Functional Governance Tabs)
// ============================================================================
function VariantC(props: SharedStateProps) {
  const [pillar, setPillar] = useState<'society' | 'affinity' | 'security' | 'immortals'>('society');

  return (
    <div className="mx-auto max-w-7xl p-4 pb-20 space-y-4">
      {/* 4-Pillar Navigation */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
        {[
          { key: 'society', label: '1. Xã Tắc & Lương Thực', icon: Wheat },
          { key: 'affinity', label: '2. Lao Động & Sở Trường', icon: Hammer },
          { key: 'security', label: `3. Trị An (${props.events.filter((e) => !e.resolved).length})`, icon: Shield },
          { key: 'immortals', label: '4. Cổ Sư & Cổ Tiên Điện', icon: Swords },
        ].map((p) => {
          const Icon = p.icon;
          const active = pillar === p.key;
          return (
            <button
              key={p.key}
              type="button"
              onClick={() => setPillar(p.key as typeof pillar)}
              className={`rounded-xl border p-3 text-left transition flex items-center gap-2.5 ${
                active ? 'border-amber-400 bg-amber-500/15 text-amber-100' : 'border-stone-800 bg-stone-900/70 text-stone-400 hover:bg-stone-800/70'
              }`}
            >
              <Icon className="h-4 w-4 shrink-0" />
              <span className="text-xs font-bold">{p.label}</span>
            </button>
          );
        })}
      </div>

      {pillar === 'society' && (
        <div className="overflow-x-auto rounded-xl border border-stone-800 bg-stone-900/80">
          <table className="w-full text-left text-xs">
            <thead className="border-b border-stone-800 bg-stone-950/70 text-stone-400">
              <tr>
                <th className="p-3">Chủng Tộc</th>
                <th className="p-3">Thành Trì</th>
                <th className="p-3">Dân Số</th>
                <th className="p-3">Đạo Ngân</th>
                <th className="p-3">Lương Thực</th>
                <th className="p-3">Cứu Trợ</th>
                <th className="p-3">Thao Tác</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-800">
              {props.races.map((r) => (
                <tr key={r.race} className="hover:bg-stone-800/40">
                  <td className="p-3 font-semibold text-stone-100">{r.name}</td>
                  <td className="p-3 text-amber-300">{r.settlementType ?? 'Chưa có'}</td>
                  <td className="p-3">{r.population.toLocaleString()} / {r.capacity.toLocaleString()}</td>
                  <td className="p-3">
                    <span className={r.daoMarksCurrent >= r.daoMarksRequired ? 'text-emerald-400' : 'text-orange-400'}>
                      {r.daoMarksCurrent}/{r.daoMarksRequired}
                    </span>
                  </td>
                  <td className="p-3">{r.starving ? <span className="text-red-400 font-bold">ĐÓI (0)</span> : r.foodStored.toLocaleString()}</td>
                  <td className="p-3">
                    <button
                      type="button"
                      onClick={() => props.onToggleRelief(r.race)}
                      className={`rounded px-2 py-0.5 ${r.autoEssenceRelief ? 'bg-amber-500/20 text-amber-200' : 'bg-stone-800 text-stone-400'}`}
                    >
                      {r.autoEssenceRelief ? 'Bật' : 'Tắt'}
                    </button>
                  </td>
                  <td className="p-3 flex gap-1.5">
                    <button type="button" onClick={() => props.onBuildOrUpgrade(r.race)} className="rounded bg-stone-800 px-2 py-1 hover:bg-stone-700">Xây/Nâng</button>
                    <button type="button" onClick={() => props.onFeedFood(r.race)} className="rounded bg-emerald-900/50 px-2 py-1 text-emerald-200">+Lương</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {pillar === 'affinity' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
          {props.points.map((pt) => {
            const prev = computePointYieldPreview(pt, props.races);
            return (
              <div key={pt.id} className="rounded-xl border border-stone-800 bg-stone-900/80 p-4 space-y-2">
                <div className="flex justify-between">
                  <span className="font-bold text-stone-100">{pt.name}</span>
                  <span className="text-amber-300">Sở trường: {pt.workerRace}</span>
                </div>
                <div className="text-stone-400">Hệ số hiện tại: <b className="text-emerald-300">{prev.mult}x</b> ({prev.status})</div>
                <div className="flex flex-wrap gap-1 pt-1">
                  {props.races.filter((r) => r.population > 0).map((r) => (
                    <button
                      key={r.race}
                      type="button"
                      onClick={() => props.onAssignPointRace(pt.id, r.race)}
                      className={`px-2 py-1 rounded border ${
                        pt.assignedRace === r.race ? 'border-amber-400 bg-amber-500/20 text-amber-200' : 'border-stone-700 text-stone-400'
                      }`}
                    >
                      {r.name}
                    </button>
                  ))}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {pillar === 'security' && (
        <div className="space-y-3 text-xs">
          {props.events.filter((e) => !e.resolved).map((ev) => (
            <div key={ev.id} className="rounded-xl border border-red-500/40 bg-red-950/20 p-4 flex flex-wrap items-center justify-between gap-3">
              <div>
                <div className="font-bold text-red-300">{ev.eventType} · {ev.race}</div>
                <div className="text-stone-200 mt-1">{ev.description}</div>
              </div>
              <div className="flex gap-2">
                <button type="button" onClick={() => props.onResolveEvent(ev.id, 'PACIFY')} className="rounded bg-emerald-600 px-3 py-1.5 font-semibold text-stone-950">An dân</button>
                <button type="button" onClick={() => props.onResolveEvent(ev.id, 'SUPPRESS')} className="rounded bg-red-600 px-3 py-1.5 font-semibold text-white">Trấn áp</button>
              </div>
            </div>
          ))}
        </div>
      )}

      {pillar === 'immortals' && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-xs">
          {props.npcs.map((n) => (
            <div key={n.id} className="rounded-xl border border-purple-500/30 bg-stone-900/80 p-4 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-bold text-sm text-stone-100">{n.name}</span>
                <span className="rounded bg-purple-500/20 px-2 py-0.5 text-purple-200">
                  {n.isImmortal ? `Cổ Tiên R${n.rank}` : `Cổ Sư R${n.rank}.${n.subRank}`}
                </span>
              </div>
              <div className="text-stone-400">{n.path} · Đạo Ngân: {n.daoMarks} · Cổ: {n.guEquipped.join(', ')}</div>
              <div className="flex flex-wrap gap-2 pt-1">
                <select
                  value={n.role}
                  onChange={(e) => props.onChangeNPCRole(n.id, e.target.value as NPCRole)}
                  className="rounded border border-stone-700 bg-stone-950 px-2 py-1 text-stone-200"
                >
                  {ROLE_OPTIONS.map((ro) => (
                    <option key={ro.key} value={ro.key}>{ro.label}</option>
                  ))}
                </select>
                {!n.isImmortal && n.rank === 5 && (
                  <button type="button" onClick={() => props.onAscendNPC(n.id)} className="rounded bg-purple-600 px-3 py-1 font-semibold text-white">
                    Hộ pháp Thăng Tiên
                  </button>
                )}
                {n.isImmortal && (
                  <button type="button" onClick={() => props.onToggleFormation(n.id)} className="rounded border border-amber-500/40 px-3 py-1 text-amber-200">
                    {n.formationSlot > 0 ? `Đang xuất chiến (#${n.formationSlot})` : 'Đưa vào đội hình'}
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// ROUTE SHELL
// ============================================================================
function ApertureLaborPrototypeInner() {
  const searchParams = useSearchParams();
  const variant = searchParams.get('variant') ?? 'A';

  const [races, setRaces] = useState<RaceState[]>(INITIAL_RACES);
  const [points, setPoints] = useState<ResourcePointMock[]>(INITIAL_POINTS);
  const [events, setEvents] = useState<VariantEventMock[]>(INITIAL_EVENTS);
  const [npcs, setNpcs] = useState<VariantNPCMock[]>(INITIAL_NPCS);
  const [essence, setEssence] = useState(420);
  const [stones, setStones] = useState(8500);

  const sharedProps: SharedStateProps = {
    races,
    points,
    events,
    npcs,
    essence,
    stones,
    onSetLabor: (race, intensity) =>
      setRaces((prev) => prev.map((r) => (r.race === race ? { ...r, laborIntensity: intensity } : r))),
    onToggleRelief: (race) =>
      setRaces((prev) =>
        prev.map((r) => (r.race === race ? { ...r, autoEssenceRelief: !r.autoEssenceRelief } : r))
      ),
    onFeedFood: (race) =>
      setRaces((prev) =>
        prev.map((r) =>
          r.race === race
            ? { ...r, foodStored: Math.min(r.foodStorageCap, r.foodStored + 2000), starving: false }
            : r
        )
      ),
    onBuildOrUpgrade: (race) => {
      setStones((s) => Math.max(0, s - 500));
      setRaces((prev) =>
        prev.map((r) => {
          if (r.race !== race) return r;
          if (!r.settlementType) {
            return {
              ...r,
              settlementType: 'VILLAGE',
              capacity: 10000,
              foodStorageCap: 10000,
              settlementPos: { x: 0, y: 0 },
            };
          }
          const next =
            r.settlementType === 'VILLAGE'
              ? { t: 'TOWN' as const, c: 50000 }
              : r.settlementType === 'TOWN'
              ? { t: 'CITY' as const, c: 200000 }
              : { t: 'FORTRESS' as const, c: 1000000 };
          return { ...r, settlementType: next.t, capacity: next.c };
        })
      );
    },
    onResolveEvent: (id, mode) => {
      const target = events.find((e) => e.id === id);
      setEvents((prev) => prev.map((e) => (e.id === id ? { ...e, resolved: true } : e)));
      if (target) {
        setRaces((prev) =>
          prev.map((r) =>
            r.race === target.race
              ? {
                  ...r,
                  rebellionRate: Math.max(5, r.rebellionRate - 30),
                  oppression: mode === 'PACIFY' ? Math.max(0, r.oppression - 20) : Math.min(100, r.oppression + 15),
                  happiness: mode === 'PACIFY' ? Math.min(100, r.happiness + 15) : Math.max(0, r.happiness - 5),
                }
              : r
          )
        );
      }
    },
    onAscendNPC: (id) => {
      setEssence((e) => Math.max(0, e - 50));
      setNpcs((prev) =>
        prev.map((n) =>
          n.id === id ? { ...n, rank: 6, subRank: 1, isImmortal: true, daoMarks: 1500 } : n
        )
      );
    },
    onChangeNPCRole: (id, role) =>
      setNpcs((prev) => prev.map((n) => (n.id === id ? { ...n, role } : n))),
    onToggleFormation: (id) =>
      setNpcs((prev) =>
        prev.map((n) => (n.id === id ? { ...n, formationSlot: n.formationSlot > 0 ? 0 : 2 } : n))
      ),
    onAssignPointRace: (pointId, race) =>
      setPoints((prev) => prev.map((p) => (p.id === pointId ? { ...p, assignedRace: race } : p))),
  };

  return (
    <div className="min-h-screen bg-stone-950 text-stone-100">
      {variant === 'A' && <VariantA {...sharedProps} />}
      {variant === 'B' && <VariantB {...sharedProps} />}
      {variant === 'C' && <VariantC {...sharedProps} />}
      <PrototypeSwitcher
        variants={[
          { key: 'A', label: 'Sổ Bộ 9 Tộc (Master-Detail)' },
          { key: 'B', label: 'Bản Đồ Ô Đất & Chỉ Huy' },
          { key: 'C', label: '4 Trụ Cột Quản Trị' },
        ]}
        current={variant}
      />
    </div>
  );
}

export default function ApertureLaborPrototypePage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-stone-950 p-6 text-stone-400">Đang tải prototype...</div>}>
      <ApertureLaborPrototypeInner />
    </Suspense>
  );
}
