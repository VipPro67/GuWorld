/**
 * GuWorld realtime wire protocol v2 — TypeScript mirror of
 * backend/internal/realtime/protocol/v2/codec.go.
 *
 * Keep the opcodes, masks and golden vectors in sync with the Go package
 * (docs/wayfinder/specs/WT-313-wire-protocol-v2.md).
 */

export const VERSION = 2;
export const MAGIC = "GWT";
export const STREAM_WORLD = 0x01;
export const STREAM_UI = 0x02;
export const MAX_FRAME_SIZE = 1 << 20;
export const MAX_DATAGRAM_SIZE = 1024;
export const QUANT_SCALE = 16;

export const Op = {
  Join: 0x01,
  Welcome: 0x02,
  Leave: 0x03,
  Transfer: 0x04,
  Reject: 0x05,
  Events: 0x06,
  Snapshot: 0x07,
  Spawn: 0x08,
  MoveTarget: 0x09,
  Cast: 0x0a,
  Attack: 0x0b,
  Capture: 0x0c,
  Stance: 0x0d,
  Resync: 0x0e,
  Keepalive: 0x0f,
} as const;

export const DatagramOp = {
  Input: 0x01,
  SnapshotDelta: 0x02,
  Ping: 0x03,
  Pong: 0x04,
} as const;

export const Mask = {
  Position: 1 << 0,
  Vitals: 1 << 1,
  Revision: 1 << 2,
  Statuses: 1 << 3,
  Cast: 1 << 4,
  Flags: 1 << 5,
  Transformation: 1 << 6,
  Extension: 1 << 7,
} as const;

export const Ext = {
  AIState: 1 << 0,
  Aggro: 1 << 1,
  Owner: 1 << 2,
  CompanionKind: 1 << 3,
  Stance: 1 << 4,
  Radius: 1 << 5,
  MaxHP: 1 << 6,
  MaxMana: 1 << 7,
} as const;

export const Vital = {
  HP: 1 << 0,
  Shield: 1 << 1,
  Mana: 1 << 2,
  Thought: 1 << 3,
} as const;

export class CodecError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "CodecError";
  }
}

const sharedEncoder = new TextEncoder();
const sharedDecoder = new TextDecoder();

export class Writer {
  private buffer: Uint8Array;
  private view: DataView;
  private offset = 0;

  constructor(capacity = 64) {
    this.buffer = new Uint8Array(Math.max(8, capacity));
    this.view = new DataView(this.buffer.buffer);
  }

  public u8(value: number): this {
    this.ensure(1);
    this.view.setUint8(this.offset, value & 0xff);
    this.offset += 1;
    return this;
  }

  public u16(value: number): this {
    this.ensure(2);
    this.view.setUint16(this.offset, value & 0xffff, true);
    this.offset += 2;
    return this;
  }

  public u32(value: number): this {
    this.ensure(4);
    this.view.setUint32(this.offset, value >>> 0, true);
    this.offset += 4;
    return this;
  }

  public u64(value: number | bigint): this {
    this.ensure(8);
    this.view.setBigUint64(this.offset, BigInt(value), true);
    this.offset += 8;
    return this;
  }

  public varint(value: number): this {
    let remaining = Math.max(0, Math.floor(value));
    if (!Number.isSafeInteger(remaining)) throw new CodecError("varint out of safe range");
    while (remaining >= 0x80) {
      this.u8((remaining % 0x80) | 0x80);
      remaining = Math.floor(remaining / 0x80);
    }
    return this.u8(remaining);
  }

  public string(value: string): this {
    const bytes = sharedEncoder.encode(value);
    if (bytes.length > 0xffff) throw new CodecError("string too long");
    this.u16(bytes.length);
    this.raw(bytes);
    return this;
  }

  public raw(bytes: Uint8Array): this {
    this.ensure(bytes.length);
    this.buffer.set(bytes, this.offset);
    this.offset += bytes.length;
    return this;
  }

  public bytes(): Uint8Array {
    return this.buffer.slice(0, this.offset);
  }

  private ensure(extra: number): void {
    if (this.offset + extra <= this.buffer.length) return;
    let capacity = this.buffer.length * 2;
    while (capacity < this.offset + extra) capacity *= 2;
    const next = new Uint8Array(capacity);
    next.set(this.buffer.subarray(0, this.offset));
    this.buffer = next;
    this.view = new DataView(next.buffer);
  }
}

export class Reader {
  private readonly bytes: Uint8Array;
  private readonly view: DataView;
  private offset = 0;
  public failed = false;

  constructor(bytes: Uint8Array) {
    this.bytes = bytes;
    this.view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  }

  public u8(): number {
    if (!this.need(1)) return 0;
    const value = this.view.getUint8(this.offset);
    this.offset += 1;
    return value;
  }

  public u16(): number {
    if (!this.need(2)) return 0;
    const value = this.view.getUint16(this.offset, true);
    this.offset += 2;
    return value;
  }

  public u32(): number {
    if (!this.need(4)) return 0;
    const value = this.view.getUint32(this.offset, true);
    this.offset += 4;
    return value;
  }

  public u64(): bigint {
    if (!this.need(8)) return BigInt(0);
    const value = this.view.getBigUint64(this.offset, true);
    this.offset += 8;
    return value;
  }

  public varint(): number {
    let result = 0;
    let shift = 0;
    for (;;) {
      const byte = this.u8();
      if (this.failed) return 0;
      result += (byte & 0x7f) * 2 ** shift;
      if ((byte & 0x80) === 0) return result;
      shift += 7;
      if (shift > 49) {
        this.failed = true;
        return 0;
      }
    }
  }

  public string(): string {
    const length = this.u16();
    if (this.failed) return "";
    if (!this.need(length)) return "";
    const value = sharedDecoder.decode(this.bytes.subarray(this.offset, this.offset + length));
    this.offset += length;
    return value;
  }

  public raw(length: number): Uint8Array {
    if (!this.need(length)) return new Uint8Array(0);
    const value = this.bytes.slice(this.offset, this.offset + length);
    this.offset += length;
    return value;
  }

  public remaining(): number {
    return this.bytes.length - this.offset;
  }

  private need(count: number): boolean {
    if (this.failed) return false;
    if (this.offset + count > this.bytes.length) {
      this.failed = true;
      return false;
    }
    return true;
  }
}

export function quantize(value: number): number {
  if (!Number.isFinite(value)) throw new CodecError("non-finite coordinate");
  const scaled = Math.round(value * QUANT_SCALE);
  if (scaled < 0 || scaled > 0xffff) throw new CodecError(`coordinate ${value} out of range`);
  return scaled;
}

export function dequantize(value: number): number {
  return value / QUANT_SCALE;
}

export function datagramHeader(op: number): Uint8Array {
  if (op > 0x3f) throw new CodecError("opcode exceeds 6 bits");
  return new Uint8Array([(VERSION << 6) | op]);
}

export function decodeDatagram(bytes: Uint8Array): { op: number; payload: Uint8Array } {
  if (bytes.length < 1) throw new CodecError("truncated datagram");
  if (bytes[0] >> 6 !== VERSION) throw new CodecError("datagram version mismatch");
  return { op: bytes[0] & 0x3f, payload: bytes.subarray(1) };
}

export function encodeFrame(op: number, payload: Uint8Array): Uint8Array {
  if (payload.length + 1 > MAX_FRAME_SIZE) throw new CodecError("frame too large");
  const out = new Uint8Array(5 + payload.length);
  new DataView(out.buffer).setUint32(0, payload.length + 1, true);
  out[4] = op;
  out.set(payload, 5);
  return out;
}

export function decodeFrame(bytes: Uint8Array): { op: number; payload: Uint8Array; rest: Uint8Array } {
  if (bytes.length < 4) throw new CodecError("truncated frame header");
  const length = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength).getUint32(0, true);
  if (length < 1 || length > MAX_FRAME_SIZE) throw new CodecError("invalid frame length");
  if (bytes.length < 4 + length) throw new CodecError("truncated frame");
  return {
    op: bytes[4],
    payload: bytes.subarray(5, 4 + length),
    rest: bytes.subarray(4 + length),
  };
}

export interface InputMessage {
  seq: number;
  moveX: number;
  moveY: number;
}

export function encodeInput(input: InputMessage): Uint8Array {
  const payload = new Writer(8);
  payload.varint(input.seq).u8(input.moveX).u8(input.moveY);
  const out = new Uint8Array(1 + payload.bytes().length);
  out.set(datagramHeader(DatagramOp.Input));
  out.set(payload.bytes(), 1);
  return out;
}

export function decodeInput(payload: Uint8Array): InputMessage {
  const reader = new Reader(payload);
  const message: InputMessage = {
    seq: reader.varint(),
    moveX: (reader.u8() << 24) >> 24,
    moveY: (reader.u8() << 24) >> 24,
  };
  if (reader.failed) throw new CodecError("truncated input");
  return message;
}

export interface DeltaEntity {
  netId: number;
  mask: number;
  hasExt: boolean;
  ext: number;
  x: number;
  y: number;
  vitalsSub: number;
  hp: number;
  shield: number;
  mana: number;
  thought: number;
  revision: number;
  statuses: number[];
  skillSlot: number;
  castEndTick: number;
  castWindup: number;
  flags: number;
  formKind: number;
  formNetId: number;
  modelScaleX100: number;
  collisionRadiusX10: number;
  currentFormHp: number;
  formFlags: number;
  aiState: number;
  aggroNetId: number;
  ownerNetId: number;
  companionKind: string;
  stance: number;
  radiusX10: number;
  maxHp: number;
  maxMana: number;
}

export interface Projectile {
  id: number;
  sourceNetId: number;
  skillId: string;
  vfx: string;
  x: number;
  y: number;
  radiusX10: number;
}

export interface SnapshotDelta {
  tick: number;
  baselineId: number;
  ackSeq: number;
  hasProjectiles: boolean;
  entities: DeltaEntity[];
  removed: number[];
  projectiles: Projectile[];
}

export function emptyDeltaEntity(netId = 0): DeltaEntity {
  return {
    netId,
    mask: 0,
    hasExt: false,
    ext: 0,
    x: 0,
    y: 0,
    vitalsSub: 0,
    hp: 0,
    shield: 0,
    mana: 0,
    thought: 0,
    revision: 0,
    statuses: [],
    skillSlot: 0,
    castEndTick: 0,
    castWindup: 0,
    flags: 0,
    formKind: 0,
    formNetId: 0,
    modelScaleX100: 0,
    collisionRadiusX10: 0,
    currentFormHp: 0,
    formFlags: 0,
    aiState: 0,
    aggroNetId: 0,
    ownerNetId: 0,
    companionKind: "",
    stance: 0,
    radiusX10: 0,
    maxHp: 0,
    maxMana: 0,
  };
}

export function writeDeltaEntity(writer: Writer, entity: DeltaEntity): void {
  writer.u16(entity.netId);
  const mask = entity.hasExt ? entity.mask | Mask.Extension : entity.mask;
  writer.u8(mask);
  if (mask & Mask.Position) writer.u16(entity.x).u16(entity.y);
  if (mask & Mask.Vitals) {
    writer.u8(entity.vitalsSub);
    if (entity.vitalsSub & Vital.HP) writer.varint(entity.hp);
    if (entity.vitalsSub & Vital.Shield) writer.varint(entity.shield);
    if (entity.vitalsSub & Vital.Mana) writer.varint(entity.mana);
    if (entity.vitalsSub & Vital.Thought) writer.varint(entity.thought);
  }
  if (mask & Mask.Revision) writer.varint(entity.revision);
  if (mask & Mask.Statuses) {
    writer.u8(entity.statuses.length);
    for (const status of entity.statuses) writer.u8(status);
  }
  if (mask & Mask.Cast) writer.u8(entity.skillSlot).u32(entity.castEndTick).u16(entity.castWindup);
  if (mask & Mask.Flags) writer.u8(entity.flags);
  if (mask & Mask.Transformation) {
    writer
      .u8(entity.formKind)
      .u16(entity.formNetId)
      .u16(entity.modelScaleX100)
      .u16(entity.collisionRadiusX10)
      .varint(entity.currentFormHp)
      .u8(entity.formFlags);
  }
  if (mask & Mask.Extension) {
    writer.u8(entity.ext);
    if (entity.ext & Ext.AIState) writer.u8(entity.aiState);
    if (entity.ext & Ext.Aggro) writer.u16(entity.aggroNetId);
    if (entity.ext & Ext.Owner) writer.u16(entity.ownerNetId);
    if (entity.ext & Ext.CompanionKind) writer.string(entity.companionKind);
    if (entity.ext & Ext.Stance) writer.u8(entity.stance);
    if (entity.ext & Ext.Radius) writer.u8(entity.radiusX10);
    if (entity.ext & Ext.MaxHP) writer.varint(entity.maxHp);
    if (entity.ext & Ext.MaxMana) writer.varint(entity.maxMana);
  }
}

export function readDeltaEntity(reader: Reader): DeltaEntity {
  const entity = emptyDeltaEntity(reader.u16());
  const mask = reader.u8();
  entity.mask = mask & ~Mask.Extension;
  entity.hasExt = (mask & Mask.Extension) !== 0;
  if (entity.mask & Mask.Position) {
    entity.x = reader.u16();
    entity.y = reader.u16();
  }
  if (entity.mask & Mask.Vitals) {
    entity.vitalsSub = reader.u8();
    if (entity.vitalsSub & Vital.HP) entity.hp = reader.varint();
    if (entity.vitalsSub & Vital.Shield) entity.shield = reader.varint();
    if (entity.vitalsSub & Vital.Mana) entity.mana = reader.varint();
    if (entity.vitalsSub & Vital.Thought) entity.thought = reader.varint();
  }
  if (entity.mask & Mask.Revision) entity.revision = reader.varint();
  if (entity.mask & Mask.Statuses) {
    const count = reader.u8();
    entity.statuses = [];
    for (let i = 0; i < count; i += 1) entity.statuses.push(reader.u8());
  }
  if (entity.mask & Mask.Cast) {
    entity.skillSlot = reader.u8();
    entity.castEndTick = reader.u32();
    entity.castWindup = reader.u16();
  }
  if (entity.mask & Mask.Flags) entity.flags = reader.u8();
  if (entity.mask & Mask.Transformation) {
    entity.formKind = reader.u8();
    entity.formNetId = reader.u16();
    entity.modelScaleX100 = reader.u16();
    entity.collisionRadiusX10 = reader.u16();
    entity.currentFormHp = reader.varint();
    entity.formFlags = reader.u8();
  }
  if (entity.hasExt) {
    entity.ext = reader.u8();
    if (entity.ext & Ext.AIState) entity.aiState = reader.u8();
    if (entity.ext & Ext.Aggro) entity.aggroNetId = reader.u16();
    if (entity.ext & Ext.Owner) entity.ownerNetId = reader.u16();
    if (entity.ext & Ext.CompanionKind) entity.companionKind = reader.string();
    if (entity.ext & Ext.Stance) entity.stance = reader.u8();
    if (entity.ext & Ext.Radius) entity.radiusX10 = reader.u8();
    if (entity.ext & Ext.MaxHP) entity.maxHp = reader.varint();
    if (entity.ext & Ext.MaxMana) entity.maxMana = reader.varint();
  }
  return entity;
}

export function encodeSnapshotDelta(snapshot: SnapshotDelta): Uint8Array {
  if (snapshot.entities.length > 255) throw new CodecError("too many entities");
  if (snapshot.removed.length > 255) throw new CodecError("too many removals");
  const payload = new Writer(64 + snapshot.entities.length * 8);
  payload.u32(snapshot.tick).u16(snapshot.baselineId).varint(snapshot.ackSeq);
  const hasProjectiles = snapshot.projectiles.length > 0;
  payload.u8(hasProjectiles ? 1 : 0);
  payload.u8(snapshot.entities.length);
  for (const entity of snapshot.entities) writeDeltaEntity(payload, entity);
  payload.u8(snapshot.removed.length);
  for (const netId of snapshot.removed) payload.u16(netId);
  if (hasProjectiles) {
    if (snapshot.projectiles.length > 255) throw new CodecError("too many projectiles");
    payload.u8(snapshot.projectiles.length);
    for (const projectile of snapshot.projectiles) {
      payload
        .u16(projectile.id)
        .u16(projectile.sourceNetId)
        .string(projectile.skillId)
        .string(projectile.vfx)
        .u16(projectile.x)
        .u16(projectile.y)
        .u8(projectile.radiusX10);
    }
  }
  const body = payload.bytes();
  if (body.length + 1 > MAX_DATAGRAM_SIZE) throw new CodecError("delta exceeds datagram budget; use the stream");
  const out = new Uint8Array(1 + body.length);
  out.set(datagramHeader(DatagramOp.SnapshotDelta));
  out.set(body, 1);
  return out;
}

export function decodeSnapshotDelta(payload: Uint8Array): SnapshotDelta {
  const reader = new Reader(payload);
  const snapshot: SnapshotDelta = {
    tick: reader.u32(),
    baselineId: reader.u16(),
    ackSeq: reader.varint(),
    hasProjectiles: false,
    entities: [],
    removed: [],
    projectiles: [],
  };
  const flags = reader.u8();
  snapshot.hasProjectiles = (flags & 1) !== 0;
  const entityCount = reader.u8();
  for (let i = 0; i < entityCount; i += 1) snapshot.entities.push(readDeltaEntity(reader));
  const removedCount = reader.u8();
  for (let i = 0; i < removedCount; i += 1) snapshot.removed.push(reader.u16());
  if (snapshot.hasProjectiles) {
    const projectileCount = reader.u8();
    for (let i = 0; i < projectileCount; i += 1) {
      snapshot.projectiles.push({
        id: reader.u16(),
        sourceNetId: reader.u16(),
        skillId: reader.string(),
        vfx: reader.string(),
        x: reader.u16(),
        y: reader.u16(),
        radiusX10: reader.u8(),
      });
    }
  }
  if (reader.failed) throw new CodecError("truncated snapshot delta");
  return snapshot;
}

export interface Spawn {
  netId: number;
  entityId: string;
  name: string;
  templateId: string;
  kind: number;
  partyId: string;
  flags: number;
  radiusX10: number;
  maxHp: number;
  maxMana: number;
  maxThought: number;
  ownerNetId: number;
  companionKind: string;
  stance: number;
}

export function emptySpawn(netId = 0): Spawn {
  return {
    netId,
    entityId: "",
    name: "",
    templateId: "",
    kind: 0,
    partyId: "",
    flags: 0,
    radiusX10: 0,
    maxHp: 0,
    maxMana: 0,
    maxThought: 0,
    ownerNetId: 0,
    companionKind: "",
    stance: 0,
  };
}

export function writeSpawn(writer: Writer, spawn: Spawn): void {
  writer.u16(spawn.netId);
  writer.string(spawn.entityId).string(spawn.name).string(spawn.templateId);
  writer.u8(spawn.kind).string(spawn.partyId).u8(spawn.flags).u8(spawn.radiusX10);
  writer.varint(spawn.maxHp).varint(spawn.maxMana).varint(spawn.maxThought);
  writer.u16(spawn.ownerNetId).string(spawn.companionKind).u8(spawn.stance);
}

export function readSpawn(reader: Reader): Spawn {
  return {
    netId: reader.u16(),
    entityId: reader.string(),
    name: reader.string(),
    templateId: reader.string(),
    kind: reader.u8(),
    partyId: reader.string(),
    flags: reader.u8(),
    radiusX10: reader.u8(),
    maxHp: reader.varint(),
    maxMana: reader.varint(),
    maxThought: reader.varint(),
    ownerNetId: reader.u16(),
    companionKind: reader.string(),
    stance: reader.u8(),
  };
}

export interface SnapshotFull {
  tick: number;
  baselineId: number;
  baseTick: number;
  ackSeq: number;
  isBaseline: boolean;
  spawns: Spawn[];
  states: DeltaEntity[];
  removed: number[];
  projectiles: Projectile[];
}

export function encodeSnapshotFull(snapshot: SnapshotFull): Uint8Array {
  if (snapshot.spawns.length > 0xffff || snapshot.removed.length > 0xffff) {
    throw new CodecError("snapshot full exceeds count limits");
  }
  const payload = new Writer(64 + snapshot.spawns.length * 96);
  payload.u32(snapshot.tick).u16(snapshot.baselineId).u32(snapshot.baseTick).varint(snapshot.ackSeq);
  payload.u8(snapshot.isBaseline ? 1 : 0);
  payload.u16(snapshot.spawns.length);
  for (const spawn of snapshot.spawns) writeSpawn(payload, spawn);
  if (snapshot.states.length > 0xffff) throw new CodecError("too many baseline states");
  payload.u16(snapshot.states.length);
  for (const state of snapshot.states) writeDeltaEntity(payload, state);
  payload.u16(snapshot.removed.length);
  for (const netId of snapshot.removed) payload.u16(netId);
  if (snapshot.projectiles.length > 255) throw new CodecError("too many projectiles");
  payload.u8(snapshot.projectiles.length);
  for (const projectile of snapshot.projectiles) {
    payload
      .u16(projectile.id)
      .u16(projectile.sourceNetId)
      .string(projectile.skillId)
      .string(projectile.vfx)
      .u16(projectile.x)
      .u16(projectile.y)
      .u8(projectile.radiusX10);
  }
  return payload.bytes();
}

export function decodeSnapshotFull(payload: Uint8Array): SnapshotFull {
  const reader = new Reader(payload);
  const snapshot: SnapshotFull = {
    tick: reader.u32(),
    baselineId: reader.u16(),
    baseTick: reader.u32(),
    ackSeq: reader.varint(),
    isBaseline: reader.u8() === 1,
    spawns: [],
    states: [],
    removed: [],
    projectiles: [],
  };
  const spawnCount = reader.u16();
  for (let i = 0; i < spawnCount; i += 1) snapshot.spawns.push(readSpawn(reader));
  const stateCount = reader.u16();
  for (let i = 0; i < stateCount; i += 1) snapshot.states.push(readDeltaEntity(reader));
  const removedCount = reader.u16();
  for (let i = 0; i < removedCount; i += 1) snapshot.removed.push(reader.u16());
  const projectileCount = reader.u8();
  for (let i = 0; i < projectileCount; i += 1) {
    snapshot.projectiles.push({
      id: reader.u16(),
      sourceNetId: reader.u16(),
      skillId: reader.string(),
      vfx: reader.string(),
      x: reader.u16(),
      y: reader.u16(),
      radiusX10: reader.u8(),
    });
  }
  if (reader.failed) throw new CodecError("truncated snapshot full");
  return snapshot;
}

export interface Welcome {
  sessionId: string;
  zoneId: string;
  mapVersion: string;
  contentVersion: string;
  tickRate: number;
  movementSpeedX100: number;
  serverTick: number;
  entityId: string;
  entityRevision: number;
  baselineId: number;
  mapHash: string;
  skillsJson: Uint8Array;
  loadoutRevision: string;
  snapshot: Uint8Array;
}

export function encodeWelcome(welcome: Welcome): Uint8Array {
  if (welcome.skillsJson.length > 0xffff) throw new CodecError("skills blob too large");
  const payload = new Writer(128 + welcome.skillsJson.length + welcome.snapshot.length);
  payload
    .string(welcome.sessionId)
    .string(welcome.zoneId)
    .string(welcome.mapVersion)
    .string(welcome.contentVersion)
    .u16(welcome.tickRate)
    .u16(welcome.movementSpeedX100)
    .u32(welcome.serverTick)
    .string(welcome.entityId)
    .varint(welcome.entityRevision)
    .u16(welcome.baselineId)
    .string(welcome.mapHash)
    .u16(welcome.skillsJson.length);
  payload.raw(welcome.skillsJson);
  payload.string(welcome.loadoutRevision);
  payload.u32(welcome.snapshot.length);
  payload.raw(welcome.snapshot);
  return payload.bytes();
}

export function decodeWelcome(payload: Uint8Array): Welcome {
  const reader = new Reader(payload);
  const welcome: Welcome = {
    sessionId: reader.string(),
    zoneId: reader.string(),
    mapVersion: reader.string(),
    contentVersion: reader.string(),
    tickRate: reader.u16(),
    movementSpeedX100: reader.u16(),
    serverTick: reader.u32(),
    entityId: reader.string(),
    entityRevision: reader.varint(),
    baselineId: reader.u16(),
    mapHash: reader.string(),
    skillsJson: new Uint8Array(0),
    loadoutRevision: "",
    snapshot: new Uint8Array(0),
  };
  const skillsLength = reader.u16();
  welcome.skillsJson = reader.raw(skillsLength);
  welcome.loadoutRevision = reader.string();
  const snapshotLength = reader.u32();
  welcome.snapshot = reader.raw(snapshotLength);
  if (reader.failed) throw new CodecError("truncated welcome");
  return welcome;
}

export function encodeMoveTarget(seq: number, x: number, y: number): Uint8Array {
  const payload = new Writer(8);
  payload.varint(seq).u16(quantize(x)).u16(quantize(y));
  return encodeFrame(Op.MoveTarget, payload.bytes());
}

export function encodeCast(input: {
  seq: number;
  slot: number;
  mode: number;
  targetNetId: number;
  aimX?: number;
  aimY?: number;
  x?: number;
  y?: number;
}): Uint8Array {
  const payload = new Writer(16);
  payload.varint(input.seq).u8(input.slot).u8(input.mode).u16(input.targetNetId);
  payload.u16((input.aimX ?? 0) << 4).u16((input.aimY ?? 0) << 4);
  payload.u16(quantize(input.x ?? 0)).u16(quantize(input.y ?? 0));
  return encodeFrame(Op.Cast, payload.bytes());
}

export function encodeAttack(input: { seq: number; targetNetId: number; aimX?: number; aimY?: number }): Uint8Array {
  const payload = new Writer(10);
  payload.varint(input.seq).u16(input.targetNetId);
  payload.u16((input.aimX ?? 0) << 4).u16((input.aimY ?? 0) << 4);
  return encodeFrame(Op.Attack, payload.bytes());
}

export function encodeCapture(input: { seq: number; targetNetId: number; guId?: string }): Uint8Array {
  const payload = new Writer(16);
  payload.varint(input.seq).u16(input.targetNetId).string(input.guId ?? "");
  return encodeFrame(Op.Capture, payload.bytes());
}

export function encodeStance(input: { seq: number; stance: number; focusNetId: number }): Uint8Array {
  const payload = new Writer(8);
  payload.varint(input.seq).u8(input.stance).u16(input.focusNetId);
  return encodeFrame(Op.Stance, payload.bytes());
}

export const EVENT_TYPE_NAMES: Record<number, string> = {
  1: "CAST_ACCEPTED", 2: "CAST_REJECTED", 3: "BASIC_ATTACK", 4: "CAST_INTERRUPTED",
  5: "DASH_ENDED", 6: "PROJECTILE_SPAWNED", 7: "HIT_CONFIRMED", 8: "DAMAGE_APPLIED",
  9: "HEAL_APPLIED", 10: "SHIELD_APPLIED", 11: "STATUS_APPLIED", 12: "STATUS_EXPIRED",
  13: "HIDDEN_PASSIVE_TRIGGERED", 14: "MOVE_TARGET_REJECTED", 15: "ENTITY_DIED",
  16: "CAPTURE_STARTED", 17: "CAPTURE_INTERRUPTED", 18: "CAPTURE_SUCCESS", 19: "CAPTURE_FAILED",
  20: "SOUL_BACKFIRE", 21: "COMPANION_INJURED", 22: "COMPANION_PERMADEATH",
  23: "TRANSFORMATION_STARTED", 24: "TRANSFORMATION_ENDED", 25: "TRANSFORMATION_BROKEN", 26: "TELEPORT",
};

export interface DecodedEvent {
  eventId: number;
  tick: number;
  typeId: number;
  source: number;
  target: number;
  payload: Uint8Array;
}

export function decodeEvents(payload: Uint8Array): { fromEventId: number; events: DecodedEvent[] } {
  const reader = new Reader(payload);
  const fromEventId = reader.varint();
  const count = reader.u8();
  const events: DecodedEvent[] = [];
  for (let i = 0; i < count; i += 1) {
    const eventId = reader.varint();
    const tick = reader.varint();
    const typeId = reader.u8();
    const source = reader.u16();
    const target = reader.u16();
    const length = reader.u16();
    events.push({ eventId, tick, typeId, source, target, payload: reader.raw(length) });
  }
  if (reader.failed) throw new CodecError("truncated events");
  return { fromEventId, events };
}

export function encodeJoin(request: { zoneId: string; mapVersion: string; contentVersion: string; requestId: number }): Uint8Array {
  const writer = new Writer(64);
  writer.string(request.zoneId).string(request.mapVersion).string(request.contentVersion).u32(request.requestId);
  return writer.bytes();
}

export interface TransferWire {
  fromZoneId: string;
  toZoneId: string;
  transferId: string;
  targetX: number;
  targetY: number;
  legacyX: number;
  legacyY: number;
  hp: number;
}

export function encodeTransfer(transfer: TransferWire): Uint8Array {
  const writer = new Writer(64);
  writer
    .string(transfer.fromZoneId)
    .string(transfer.toZoneId)
    .string(transfer.transferId)
    .u16(transfer.targetX)
    .u16(transfer.targetY)
    .u32(transfer.legacyX >>> 0)
    .u32(transfer.legacyY >>> 0)
    .varint(transfer.hp);
  return writer.bytes();
}

export function decodeTransfer(payload: Uint8Array): TransferWire {
  const reader = new Reader(payload);
  const transfer: TransferWire = {
    fromZoneId: reader.string(),
    toZoneId: reader.string(),
    transferId: reader.string(),
    targetX: reader.u16(),
    targetY: reader.u16(),
    legacyX: reader.u32() | 0,
    legacyY: reader.u32() | 0,
    hp: reader.varint(),
  };
  if (reader.failed) throw new CodecError("truncated transfer");
  return transfer;
}

export function encodePingDatagram(clientMs: number): Uint8Array {
  const writer = new Writer(8);
  writer.u64(Math.max(0, Math.floor(clientMs)));
  const body = writer.bytes();
  const out = new Uint8Array(1 + body.length);
  out.set(datagramHeader(DatagramOp.Ping));
  out.set(body, 1);
  return out;
}

export function decodePongDatagram(payload: Uint8Array): { clientMs: number; serverTick: number } {
  const reader = new Reader(payload);
  const clientMs = Number(reader.u64());
  const serverTick = reader.u32();
  if (reader.failed) throw new CodecError("truncated pong");
  return { clientMs, serverTick };
}
