// Package v2 implements the GuWorld realtime wire protocol version 2
// (docs/wayfinder/specs/WT-313-wire-protocol-v2.md).
//
// It is deliberately independent of the v1 JSON protocol: the codec only moves
// bytes, and callers map domain state to these records. A TypeScript mirror
// lives in frontend/src/game/network/v2/Codec.ts and both sides share golden
// vectors.
package v2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	// Version is the protocol version carried in the stream magic and the
	// datagram header.
	Version = 2
	// Magic prefixes the world stream handshake.
	Magic = "GWT"
	// StreamWorld is the first byte of the reliable world stream.
	StreamWorld byte = 0x01
	// StreamUI is the first byte of the reliable NDJSON UI stream.
	StreamUI byte = 0x02

	// MaxFrameSize bounds one stream frame.
	MaxFrameSize = 1 << 20
	// MaxDatagramSize is the safe browser payload budget (Safari hard cap).
	MaxDatagramSize = 1024
	// QuantScale converts world units to the 1/16-unit wire grid.
	QuantScale = 16.0
	// MaxQuantCoord is the largest representable world coordinate.
	MaxQuantCoord = 65535.0 / QuantScale
)

// Stream opcodes.
const (
	OpJoin       byte = 0x01
	OpWelcome    byte = 0x02
	OpLeave      byte = 0x03
	OpTransfer   byte = 0x04
	OpReject     byte = 0x05
	OpEvents     byte = 0x06
	OpSnapshot   byte = 0x07
	OpSpawn      byte = 0x08
	OpMoveTarget byte = 0x09
	OpCast       byte = 0x0A
	OpAttack     byte = 0x0B
	OpCapture    byte = 0x0C
	OpStance     byte = 0x0D
	OpResync     byte = 0x0E
	OpKeepalive  byte = 0x0F
)

// Datagram opcodes (lower 6 bits of the header byte).
const (
	OpInput         byte = 0x01
	OpSnapshotDelta byte = 0x02
	OpPing          byte = 0x03
	OpPong          byte = 0x04
)

// Entity delta mask bits.
const (
	MaskPosition       uint8 = 1 << 0
	MaskVitals         uint8 = 1 << 1
	MaskRevision       uint8 = 1 << 2
	MaskStatuses       uint8 = 1 << 3
	MaskCast           uint8 = 1 << 4
	MaskFlags          uint8 = 1 << 5
	MaskTransformation uint8 = 1 << 6
	MaskExtension      uint8 = 1 << 7
)

// Entity extension mask bits.
const (
	ExtAIState       uint8 = 1 << 0
	ExtAggro         uint8 = 1 << 1
	ExtOwner         uint8 = 1 << 2
	ExtCompanionKind uint8 = 1 << 3
	ExtStance        uint8 = 1 << 4
	ExtRadius        uint8 = 1 << 5
	ExtMaxHP         uint8 = 1 << 6
	ExtMaxMana       uint8 = 1 << 7
)

// Vital sub-mask bits.
const (
	VitalHP      uint8 = 1 << 0
	VitalShield  uint8 = 1 << 1
	VitalMana    uint8 = 1 << 2
	VitalThought uint8 = 1 << 3
)

var ErrTruncated = errors.New("v2: truncated payload")
var ErrInvalid = errors.New("v2: invalid encoding")

// Quantize converts a world coordinate to the 1/16-unit wire grid.
func Quantize(value float64) (uint16, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("%w: non-finite coordinate", ErrInvalid)
	}
	scaled := math.Round(value * QuantScale)
	if scaled < 0 || scaled > 65535 {
		return 0, fmt.Errorf("%w: coordinate %v outside [0,%v]", ErrInvalid, value, MaxQuantCoord)
	}
	return uint16(scaled), nil
}

// Dequantize converts a wire coordinate back to world units.
func Dequantize(value uint16) float64 {
	return float64(value) / QuantScale
}

// Encoder appends little-endian values to a growing buffer.
type Encoder struct {
	buf []byte
	err error
}

func NewEncoder(capacity int) *Encoder { return &Encoder{buf: make([]byte, 0, capacity)} }

func (e *Encoder) Bytes() []byte { return e.buf }

func (e *Encoder) Err() error { return e.err }

func (e *Encoder) U8(value byte) *Encoder {
	e.buf = append(e.buf, value)
	return e
}

func (e *Encoder) U16(value uint16) *Encoder {
	e.buf = binary.LittleEndian.AppendUint16(e.buf, value)
	return e
}

func (e *Encoder) U32(value uint32) *Encoder {
	e.buf = binary.LittleEndian.AppendUint32(e.buf, value)
	return e
}

func (e *Encoder) U64(value uint64) *Encoder {
	e.buf = binary.LittleEndian.AppendUint64(e.buf, value)
	return e
}

func (e *Encoder) Varint(value uint64) *Encoder {
	e.buf = binary.AppendUvarint(e.buf, value)
	return e
}

func (e *Encoder) String(value string) *Encoder {
	if len(value) > 0xFFFF {
		e.err = fmt.Errorf("%w: string too long", ErrInvalid)
		return e
	}
	e.U16(uint16(len(value)))
	e.buf = append(e.buf, value...)
	return e
}

// Raw appends bytes without a length prefix.
func (e *Encoder) Raw(value []byte) *Encoder {
	e.buf = append(e.buf, value...)
	return e
}

// Decoder reads little-endian values from a bounded buffer.
type Decoder struct {
	buf []byte
	pos int
	err error
}

func NewDecoder(buf []byte) *Decoder { return &Decoder{buf: buf} }

func (d *Decoder) Err() error { return d.err }

func (d *Decoder) Remaining() int { return len(d.buf) - d.pos }

func (d *Decoder) fail(err error) {
	if d.err == nil {
		d.err = err
	}
}

func (d *Decoder) U8() byte {
	if d.err != nil || d.pos+1 > len(d.buf) {
		d.fail(ErrTruncated)
		return 0
	}
	value := d.buf[d.pos]
	d.pos++
	return value
}

func (d *Decoder) U16() uint16 {
	if d.err != nil || d.pos+2 > len(d.buf) {
		d.fail(ErrTruncated)
		return 0
	}
	value := binary.LittleEndian.Uint16(d.buf[d.pos:])
	d.pos += 2
	return value
}

func (d *Decoder) U32() uint32 {
	if d.err != nil || d.pos+4 > len(d.buf) {
		d.fail(ErrTruncated)
		return 0
	}
	value := binary.LittleEndian.Uint32(d.buf[d.pos:])
	d.pos += 4
	return value
}

func (d *Decoder) U64() uint64 {
	if d.err != nil || d.pos+8 > len(d.buf) {
		d.fail(ErrTruncated)
		return 0
	}
	value := binary.LittleEndian.Uint64(d.buf[d.pos:])
	d.pos += 8
	return value
}

func (d *Decoder) Varint() uint64 {
	if d.err != nil {
		return 0
	}
	value, size := binary.Uvarint(d.buf[d.pos:])
	if size <= 0 {
		d.fail(ErrInvalid)
		return 0
	}
	d.pos += size
	return value
}

func (d *Decoder) String() string {
	length := int(d.U16())
	if d.err != nil {
		return ""
	}
	if d.pos+length > len(d.buf) {
		d.fail(ErrTruncated)
		return ""
	}
	value := string(d.buf[d.pos : d.pos+length])
	d.pos += length
	return value
}

// Bytes reads exactly n raw bytes.
func (d *Decoder) Bytes(n int) []byte {
	if d.err != nil || n < 0 || d.pos+n > len(d.buf) {
		d.fail(ErrTruncated)
		return nil
	}
	value := d.buf[d.pos : d.pos+n]
	d.pos += n
	return value
}

// EncodeFrame builds one stream frame: u32 length, opcode, payload.
func EncodeFrame(op byte, payload []byte) ([]byte, error) {
	if len(payload)+1 > MaxFrameSize {
		return nil, fmt.Errorf("%w: frame too large", ErrInvalid)
	}
	out := make([]byte, 4+1+len(payload))
	binary.LittleEndian.PutUint32(out, uint32(len(payload)+1))
	out[4] = op
	copy(out[5:], payload)
	return out, nil
}

// DecodeFrame reads one frame from the front of buf and returns the unread tail.
func DecodeFrame(buf []byte) (op byte, payload []byte, rest []byte, err error) {
	if len(buf) < 4 {
		return 0, nil, buf, ErrTruncated
	}
	length := int(binary.LittleEndian.Uint32(buf))
	if length < 1 || length > MaxFrameSize {
		return 0, nil, buf, fmt.Errorf("%w: frame length %d", ErrInvalid, length)
	}
	if len(buf) < 4+length {
		return 0, nil, buf, ErrTruncated
	}
	op = buf[4]
	payload = buf[5 : 4+length]
	rest = buf[4+length:]
	return op, payload, rest, nil
}

// EncodeDatagram builds the 1-byte header + payload.
func EncodeDatagram(op byte, payload []byte) ([]byte, error) {
	if op > 0x3F {
		return nil, fmt.Errorf("%w: opcode %d exceeds 6 bits", ErrInvalid, op)
	}
	if len(payload)+1 > MaxDatagramSize {
		return nil, fmt.Errorf("%w: datagram too large", ErrInvalid)
	}
	out := make([]byte, 1+len(payload))
	out[0] = Version<<6 | op
	copy(out[1:], payload)
	return out, nil
}

// DecodeDatagram validates the version and returns the opcode and payload.
func DecodeDatagram(buf []byte) (op byte, payload []byte, err error) {
	if len(buf) < 1 {
		return 0, nil, ErrTruncated
	}
	if version := buf[0] >> 6; version != Version {
		return 0, nil, fmt.Errorf("%w: datagram version %d", ErrInvalid, version)
	}
	return buf[0] & 0x3F, buf[1:], nil
}

// Input is the unreliable movement vector. Components are normalized to
// [-1,1] and quantized to int8.
type Input struct {
	Seq   uint64
	MoveX int8
	MoveY int8
}

func EncodeInput(input Input) ([]byte, error) {
	payload := NewEncoder(10)
	payload.Varint(input.Seq)
	payload.U8(byte(input.MoveX))
	payload.U8(byte(input.MoveY))
	if payload.Err() != nil {
		return nil, payload.Err()
	}
	return EncodeDatagram(OpInput, payload.Bytes())
}

func DecodeInput(payload []byte) (Input, error) {
	decoder := NewDecoder(payload)
	input := Input{
		Seq:   decoder.Varint(),
		MoveX: int8(decoder.U8()),
		MoveY: int8(decoder.U8()),
	}
	return input, decoder.Err()
}

// SnapshotDelta is one unreliable world snapshot.
type SnapshotDelta struct {
	Tick           uint32
	BaselineID     uint16
	AckSeq         uint64
	HasProjectiles bool
	Entities       []DeltaEntity
	Removed        []uint16
	Projectiles    []Projectile
}

// DeltaEntity carries only the fields flagged in Mask. Ext is used when
// MaskExtension is set; HasExt distinguishes "no extension" from ext==0.
type DeltaEntity struct {
	NetID  uint16
	Mask   uint8
	HasExt bool
	Ext    uint8

	X uint16
	Y uint16

	VitalsSub uint8
	HP        uint64
	Shield    uint64
	Mana      uint64
	Thought   uint64

	Revision uint64
	Statuses []uint8

	SkillSlot   uint8
	CastEndTick uint32
	CastWindup  uint16

	Flags uint8

	FormKind           uint8
	FormNetID          uint16
	ModelScaleX100     uint16
	CollisionRadiusX10 uint16
	CurrentFormHP      uint64
	FormFlags          uint8

	AIState       uint8
	AggroNetID    uint16
	OwnerNetID    uint16
	CompanionKind string
	Stance        uint8
	RadiusX10     uint8
	MaxHP         uint64
	MaxMana       uint64
}

// Projectile rides in the delta when HasProjectiles is set.
type Projectile struct {
	ID          uint16
	SourceNetID uint16
	SkillID     string
	VFX         string
	X           uint16
	Y           uint16
	RadiusX10   uint8
}

func EncodeDeltaEntity(encoder *Encoder, entity DeltaEntity) {
	encoder.U16(entity.NetID)
	mask := entity.Mask
	if entity.HasExt {
		mask |= MaskExtension
	}
	encoder.U8(mask)
	if mask&MaskPosition != 0 {
		encoder.U16(entity.X).U16(entity.Y)
	}
	if mask&MaskVitals != 0 {
		encoder.U8(entity.VitalsSub)
		if entity.VitalsSub&VitalHP != 0 {
			encoder.Varint(entity.HP)
		}
		if entity.VitalsSub&VitalShield != 0 {
			encoder.Varint(entity.Shield)
		}
		if entity.VitalsSub&VitalMana != 0 {
			encoder.Varint(entity.Mana)
		}
		if entity.VitalsSub&VitalThought != 0 {
			encoder.Varint(entity.Thought)
		}
	}
	if mask&MaskRevision != 0 {
		encoder.Varint(entity.Revision)
	}
	if mask&MaskStatuses != 0 {
		encoder.U8(uint8(len(entity.Statuses)))
		for _, status := range entity.Statuses {
			encoder.U8(status)
		}
	}
	if mask&MaskCast != 0 {
		encoder.U8(entity.SkillSlot).U32(entity.CastEndTick).U16(entity.CastWindup)
	}
	if mask&MaskFlags != 0 {
		encoder.U8(entity.Flags)
	}
	if mask&MaskTransformation != 0 {
		encoder.U8(entity.FormKind).U16(entity.FormNetID).U16(entity.ModelScaleX100)
		encoder.U16(entity.CollisionRadiusX10).Varint(entity.CurrentFormHP).U8(entity.FormFlags)
	}
	if mask&MaskExtension != 0 {
		encoder.U8(entity.Ext)
		if entity.Ext&ExtAIState != 0 {
			encoder.U8(entity.AIState)
		}
		if entity.Ext&ExtAggro != 0 {
			encoder.U16(entity.AggroNetID)
		}
		if entity.Ext&ExtOwner != 0 {
			encoder.U16(entity.OwnerNetID)
		}
		if entity.Ext&ExtCompanionKind != 0 {
			encoder.String(entity.CompanionKind)
		}
		if entity.Ext&ExtStance != 0 {
			encoder.U8(entity.Stance)
		}
		if entity.Ext&ExtRadius != 0 {
			encoder.U8(entity.RadiusX10)
		}
		if entity.Ext&ExtMaxHP != 0 {
			encoder.Varint(entity.MaxHP)
		}
		if entity.Ext&ExtMaxMana != 0 {
			encoder.Varint(entity.MaxMana)
		}
	}
}

func DecodeDeltaEntity(decoder *Decoder) DeltaEntity {
	entity := DeltaEntity{}
	entity.NetID = decoder.U16()
	mask := decoder.U8()
	entity.Mask = mask &^ MaskExtension
	entity.HasExt = mask&MaskExtension != 0
	if entity.Mask&MaskPosition != 0 {
		entity.X = decoder.U16()
		entity.Y = decoder.U16()
	}
	if entity.Mask&MaskVitals != 0 {
		entity.VitalsSub = decoder.U8()
		if entity.VitalsSub&VitalHP != 0 {
			entity.HP = decoder.Varint()
		}
		if entity.VitalsSub&VitalShield != 0 {
			entity.Shield = decoder.Varint()
		}
		if entity.VitalsSub&VitalMana != 0 {
			entity.Mana = decoder.Varint()
		}
		if entity.VitalsSub&VitalThought != 0 {
			entity.Thought = decoder.Varint()
		}
	}
	if entity.Mask&MaskRevision != 0 {
		entity.Revision = decoder.Varint()
	}
	if entity.Mask&MaskStatuses != 0 {
		count := int(decoder.U8())
		entity.Statuses = make([]uint8, 0, count)
		for i := 0; i < count; i++ {
			entity.Statuses = append(entity.Statuses, decoder.U8())
		}
	}
	if entity.Mask&MaskCast != 0 {
		entity.SkillSlot = decoder.U8()
		entity.CastEndTick = decoder.U32()
		entity.CastWindup = decoder.U16()
	}
	if entity.Mask&MaskFlags != 0 {
		entity.Flags = decoder.U8()
	}
	if entity.Mask&MaskTransformation != 0 {
		entity.FormKind = decoder.U8()
		entity.FormNetID = decoder.U16()
		entity.ModelScaleX100 = decoder.U16()
		entity.CollisionRadiusX10 = decoder.U16()
		entity.CurrentFormHP = decoder.Varint()
		entity.FormFlags = decoder.U8()
	}
	if entity.HasExt {
		entity.Ext = decoder.U8()
		if entity.Ext&ExtAIState != 0 {
			entity.AIState = decoder.U8()
		}
		if entity.Ext&ExtAggro != 0 {
			entity.AggroNetID = decoder.U16()
		}
		if entity.Ext&ExtOwner != 0 {
			entity.OwnerNetID = decoder.U16()
		}
		if entity.Ext&ExtCompanionKind != 0 {
			entity.CompanionKind = decoder.String()
		}
		if entity.Ext&ExtStance != 0 {
			entity.Stance = decoder.U8()
		}
		if entity.Ext&ExtRadius != 0 {
			entity.RadiusX10 = decoder.U8()
		}
		if entity.Ext&ExtMaxHP != 0 {
			entity.MaxHP = decoder.Varint()
		}
		if entity.Ext&ExtMaxMana != 0 {
			entity.MaxMana = decoder.Varint()
		}
	}
	return entity
}

// EncodeSnapshotDelta builds the datagram payload for one delta snapshot.
func EncodeSnapshotDelta(snapshot SnapshotDelta) ([]byte, error) {
	encoder := NewEncoder(64 + len(snapshot.Entities)*8)
	encoder.U32(snapshot.Tick)
	encoder.U16(snapshot.BaselineID)
	encoder.Varint(snapshot.AckSeq)
	flags := byte(0)
	if len(snapshot.Projectiles) > 0 {
		flags |= 1
	}
	encoder.U8(flags)
	if len(snapshot.Entities) > 255 {
		return nil, fmt.Errorf("%w: too many entities", ErrInvalid)
	}
	encoder.U8(uint8(len(snapshot.Entities)))
	for _, entity := range snapshot.Entities {
		EncodeDeltaEntity(encoder, entity)
	}
	if len(snapshot.Removed) > 255 {
		return nil, fmt.Errorf("%w: too many removals", ErrInvalid)
	}
	encoder.U8(uint8(len(snapshot.Removed)))
	for _, netID := range snapshot.Removed {
		encoder.U16(netID)
	}
	if flags&1 != 0 {
		if len(snapshot.Projectiles) > 255 {
			return nil, fmt.Errorf("%w: too many projectiles", ErrInvalid)
		}
		encoder.U8(uint8(len(snapshot.Projectiles)))
		for _, projectile := range snapshot.Projectiles {
			encoder.U16(projectile.ID).U16(projectile.SourceNetID)
			encoder.String(projectile.SkillID).String(projectile.VFX)
			encoder.U16(projectile.X).U16(projectile.Y).U8(projectile.RadiusX10)
		}
	}
	return EncodeDatagram(OpSnapshotDelta, encoder.Bytes())
}

// DecodeSnapshotDelta parses a datagram payload produced by EncodeSnapshotDelta.
func DecodeSnapshotDelta(payload []byte) (SnapshotDelta, error) {
	decoder := NewDecoder(payload)
	snapshot := SnapshotDelta{
		Tick:       decoder.U32(),
		BaselineID: decoder.U16(),
		AckSeq:     decoder.Varint(),
	}
	flags := decoder.U8()
	snapshot.HasProjectiles = flags&1 != 0
	entityCount := int(decoder.U8())
	snapshot.Entities = make([]DeltaEntity, 0, entityCount)
	for i := 0; i < entityCount && decoder.Err() == nil; i++ {
		snapshot.Entities = append(snapshot.Entities, DecodeDeltaEntity(decoder))
	}
	removedCount := int(decoder.U8())
	snapshot.Removed = make([]uint16, 0, removedCount)
	for i := 0; i < removedCount && decoder.Err() == nil; i++ {
		snapshot.Removed = append(snapshot.Removed, decoder.U16())
	}
	if snapshot.HasProjectiles {
		projectileCount := int(decoder.U8())
		snapshot.Projectiles = make([]Projectile, 0, projectileCount)
		for i := 0; i < projectileCount && decoder.Err() == nil; i++ {
			projectile := Projectile{
				ID:          decoder.U16(),
				SourceNetID: decoder.U16(),
				SkillID:     decoder.String(),
				VFX:         decoder.String(),
				X:           decoder.U16(),
				Y:           decoder.U16(),
				RadiusX10:   decoder.U8(),
			}
			snapshot.Projectiles = append(snapshot.Projectiles, projectile)
		}
	}
	return snapshot, decoder.Err()
}

// SnapshotFull is a reliable baseline/full snapshot carried on the world
// stream (opcode Snapshot). Deltas stay on datagrams.
type SnapshotFull struct {
	Tick        uint32
	BaselineID  uint16
	BaseTick    uint32
	AckSeq      uint64
	IsBaseline  bool
	Spawns      []Spawn
	Removed     []uint16
	Projectiles []Projectile
}

// EncodeSnapshotFull builds the stream payload for a baseline/full snapshot.
func EncodeSnapshotFull(snapshot SnapshotFull) ([]byte, error) {
	if len(snapshot.Spawns) > 0xFFFF || len(snapshot.Removed) > 0xFFFF || len(snapshot.Projectiles) > 255 {
		return nil, fmt.Errorf("%w: snapshot full exceeds count limits", ErrInvalid)
	}
	encoder := NewEncoder(64 + len(snapshot.Spawns)*96)
	encoder.U32(snapshot.Tick).U16(snapshot.BaselineID).U32(snapshot.BaseTick).Varint(snapshot.AckSeq)
	if snapshot.IsBaseline {
		encoder.U8(1)
	} else {
		encoder.U8(0)
	}
	encoder.U16(uint16(len(snapshot.Spawns)))
	for _, spawn := range snapshot.Spawns {
		EncodeSpawn(encoder, spawn)
	}
	encoder.U16(uint16(len(snapshot.Removed)))
	for _, netID := range snapshot.Removed {
		encoder.U16(netID)
	}
	encoder.U8(uint8(len(snapshot.Projectiles)))
	for _, projectile := range snapshot.Projectiles {
		encoder.U16(projectile.ID).U16(projectile.SourceNetID)
		encoder.String(projectile.SkillID).String(projectile.VFX)
		encoder.U16(projectile.X).U16(projectile.Y).U8(projectile.RadiusX10)
	}
	return encoder.Bytes(), encoder.Err()
}

// DecodeSnapshotFull parses a stream baseline/full snapshot payload.
func DecodeSnapshotFull(payload []byte) (SnapshotFull, error) {
	decoder := NewDecoder(payload)
	snapshot := SnapshotFull{
		Tick:       decoder.U32(),
		BaselineID: decoder.U16(),
		BaseTick:   decoder.U32(),
		AckSeq:     decoder.Varint(),
		IsBaseline: decoder.U8() == 1,
	}
	spawnCount := int(decoder.U16())
	snapshot.Spawns = make([]Spawn, 0, spawnCount)
	for i := 0; i < spawnCount && decoder.Err() == nil; i++ {
		snapshot.Spawns = append(snapshot.Spawns, DecodeSpawn(decoder))
	}
	removedCount := int(decoder.U16())
	snapshot.Removed = make([]uint16, 0, removedCount)
	for i := 0; i < removedCount && decoder.Err() == nil; i++ {
		snapshot.Removed = append(snapshot.Removed, decoder.U16())
	}
	projectileCount := int(decoder.U8())
	snapshot.Projectiles = make([]Projectile, 0, projectileCount)
	for i := 0; i < projectileCount && decoder.Err() == nil; i++ {
		snapshot.Projectiles = append(snapshot.Projectiles, Projectile{
			ID:          decoder.U16(),
			SourceNetID: decoder.U16(),
			SkillID:     decoder.String(),
			VFX:         decoder.String(),
			X:           decoder.U16(),
			Y:           decoder.U16(),
			RadiusX10:   decoder.U8(),
		})
	}
	return snapshot, decoder.Err()
}

// Welcome is the binary handshake record. Skills stay JSON (one-time join).
type Welcome struct {
	SessionID         string
	ZoneID            string
	MapVersion        string
	ContentVersion    string
	TickRate          uint16
	MovementSpeedX100 uint16
	ServerTick        uint32
	EntityID          string
	EntityRevision    uint64
	BaselineID        uint16
	MapHash           string
	SkillsJSON        []byte
	LoadoutRevision   string
	Snapshot          []byte
}

func EncodeWelcome(welcome Welcome) ([]byte, error) {
	if len(welcome.SkillsJSON) > 0xFFFF {
		return nil, fmt.Errorf("%w: skills blob too large", ErrInvalid)
	}
	if len(welcome.Snapshot) > MaxFrameSize {
		return nil, fmt.Errorf("%w: welcome snapshot too large", ErrInvalid)
	}
	encoder := NewEncoder(128 + len(welcome.SkillsJSON) + len(welcome.Snapshot))
	encoder.String(welcome.SessionID).String(welcome.ZoneID)
	encoder.String(welcome.MapVersion).String(welcome.ContentVersion)
	encoder.U16(welcome.TickRate).U16(welcome.MovementSpeedX100).U32(welcome.ServerTick)
	encoder.String(welcome.EntityID).Varint(welcome.EntityRevision).U16(welcome.BaselineID)
	encoder.String(welcome.MapHash)
	encoder.U16(uint16(len(welcome.SkillsJSON)))
	encoder.Raw(welcome.SkillsJSON)
	encoder.String(welcome.LoadoutRevision)
	encoder.U32(uint32(len(welcome.Snapshot)))
	encoder.Raw(welcome.Snapshot)
	return encoder.Bytes(), encoder.Err()
}

func DecodeWelcome(payload []byte) (Welcome, error) {
	decoder := NewDecoder(payload)
	welcome := Welcome{
		SessionID:         decoder.String(),
		ZoneID:            decoder.String(),
		MapVersion:        decoder.String(),
		ContentVersion:    decoder.String(),
		TickRate:          decoder.U16(),
		MovementSpeedX100: decoder.U16(),
		ServerTick:        decoder.U32(),
		EntityID:          decoder.String(),
		EntityRevision:    decoder.Varint(),
		BaselineID:        decoder.U16(),
		MapHash:           decoder.String(),
	}
	skillsLength := int(decoder.U16())
	welcome.SkillsJSON = append([]byte(nil), decoder.Bytes(skillsLength)...)
	welcome.LoadoutRevision = decoder.String()
	snapshotLength := int(decoder.U32())
	welcome.Snapshot = append([]byte(nil), decoder.Bytes(snapshotLength)...)
	return welcome, decoder.Err()
}

// Spawn is the full metadata record sent once when an entity enters AOI.
type Spawn struct {
	NetID         uint16
	EntityID      string
	Name          string
	TemplateID    string
	Kind          uint8
	PartyID       string
	Flags         uint8
	RadiusX10     uint8
	MaxHP         uint64
	MaxMana       uint64
	MaxThought    uint64
	OwnerNetID    uint16
	CompanionKind string
	Stance        uint8
}

func EncodeSpawn(encoder *Encoder, spawn Spawn) {
	encoder.U16(spawn.NetID)
	encoder.String(spawn.EntityID)
	encoder.String(spawn.Name)
	encoder.String(spawn.TemplateID)
	encoder.U8(spawn.Kind)
	encoder.String(spawn.PartyID)
	encoder.U8(spawn.Flags)
	encoder.U8(spawn.RadiusX10)
	encoder.Varint(spawn.MaxHP)
	encoder.Varint(spawn.MaxMana)
	encoder.Varint(spawn.MaxThought)
	encoder.U16(spawn.OwnerNetID)
	encoder.String(spawn.CompanionKind)
	encoder.U8(spawn.Stance)
}

func DecodeSpawn(decoder *Decoder) Spawn {
	return Spawn{
		NetID:         decoder.U16(),
		EntityID:      decoder.String(),
		Name:          decoder.String(),
		TemplateID:    decoder.String(),
		Kind:          decoder.U8(),
		PartyID:       decoder.String(),
		Flags:         decoder.U8(),
		RadiusX10:     decoder.U8(),
		MaxHP:         decoder.Varint(),
		MaxMana:       decoder.Varint(),
		MaxThought:    decoder.Varint(),
		OwnerNetID:    decoder.U16(),
		CompanionKind: decoder.String(),
		Stance:        decoder.U8(),
	}
}

// Event is one gamefx event. Payload stays JSON in v2.
type Event struct {
	EventID uint64
	Tick    uint64
	TypeID  uint8
	Source  uint16
	Target  uint16
	Payload []byte
}

func EncodeEvents(fromEventID uint64, events []Event) ([]byte, error) {
	encoder := NewEncoder(32 + len(events)*32)
	encoder.Varint(fromEventID)
	if len(events) > 255 {
		return nil, fmt.Errorf("%w: too many events", ErrInvalid)
	}
	encoder.U8(uint8(len(events)))
	for _, event := range events {
		encoder.Varint(event.EventID).Varint(event.Tick)
		encoder.U8(event.TypeID).U16(event.Source).U16(event.Target)
		if len(event.Payload) > 0xFFFF {
			return nil, fmt.Errorf("%w: event payload too large", ErrInvalid)
		}
		encoder.U16(uint16(len(event.Payload)))
		encoder.Raw(event.Payload)
	}
	return encoder.Bytes(), encoder.Err()
}

func DecodeEvents(payload []byte) (fromEventID uint64, events []Event, err error) {
	decoder := NewDecoder(payload)
	fromEventID = decoder.Varint()
	count := int(decoder.U8())
	events = make([]Event, 0, count)
	for i := 0; i < count && decoder.Err() == nil; i++ {
		event := Event{
			EventID: decoder.Varint(),
			Tick:    decoder.Varint(),
			TypeID:  decoder.U8(),
			Source:  decoder.U16(),
			Target:  decoder.U16(),
		}
		length := int(decoder.U16())
		event.Payload = append([]byte(nil), decoder.Bytes(length)...)
		events = append(events, event)
	}
	return fromEventID, events, decoder.Err()
}
