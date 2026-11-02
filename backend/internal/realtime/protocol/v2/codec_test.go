package v2

import (
	"encoding/hex"
	"testing"
)

func TestInputRoundTrip(t *testing.T) {
	encoded, err := EncodeInput(Input{Seq: 42, MoveX: 127, MoveY: -64})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if len(encoded) != 4 {
		t.Fatalf("input size = %d, want 4", len(encoded))
	}
	op, payload, err := DecodeDatagram(encoded)
	if err != nil || op != OpInput {
		t.Fatalf("header = %d err=%v", op, err)
	}
	decoded, err := DecodeInput(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != (Input{Seq: 42, MoveX: 127, MoveY: -64}) {
		t.Fatalf("roundtrip = %+v", decoded)
	}
}

func TestMovingEntityDeltaIsSevenBytes(t *testing.T) {
	entity := DeltaEntity{
		NetID: 300,
		Mask:  MaskPosition,
		X:     1600, // 100 units
		Y:     800,  // 50 units
	}
	snapshot := SnapshotDelta{
		Tick:       1234,
		BaselineID: 7,
		AckSeq:     99,
		Entities:   []DeltaEntity{entity},
	}
	encoded, err := EncodeSnapshotDelta(snapshot)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// header 1 + tick 4 + baseline 2 + ack 1 + flags 1 + count 1 + removed 1
	// + entity (net_id 2 + mask 1 + x 2 + y 2) = 18 bytes.
	if len(encoded) != 18 {
		t.Fatalf("delta size = %d, want 18", len(encoded))
	}
	op, payload, err := DecodeDatagram(encoded)
	if err != nil || op != OpSnapshotDelta {
		t.Fatalf("header = %d err=%v", op, err)
	}
	decoded, err := DecodeSnapshotDelta(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded.Entities) != 1 || decoded.Entities[0].NetID != 300 ||
		Dequantize(decoded.Entities[0].X) != 100 || Dequantize(decoded.Entities[0].Y) != 50 {
		t.Fatalf("decoded = %+v", decoded)
	}
}

func TestDeltaRoundTripAllFields(t *testing.T) {
	entity := DeltaEntity{
		NetID:              1,
		Mask:               MaskVitals | MaskRevision | MaskStatuses | MaskCast | MaskFlags | MaskTransformation,
		HasExt:             true,
		Ext:                ExtAIState | ExtAggro | ExtOwner | ExtCompanionKind | ExtStance | ExtRadius | ExtMaxHP | ExtMaxMana,
		VitalsSub:          VitalHP | VitalShield | VitalMana | VitalThought,
		HP:                 900,
		Shield:             50,
		Mana:               300,
		Thought:            12,
		Revision:           44,
		Statuses:           []uint8{1, 7, 9},
		SkillSlot:          3,
		CastEndTick:        9999,
		CastWindup:         4,
		Flags:              0b1010,
		FormKind:           2,
		FormNetID:          88,
		ModelScaleX100:     150,
		CollisionRadiusX10: 9,
		CurrentFormHP:      500,
		FormFlags:          1,
		AIState:            4,
		AggroNetID:         77,
		OwnerNetID:         55,
		CompanionKind:      "BEAST",
		Stance:             2,
		RadiusX10:          8,
		MaxHP:              1200,
		MaxMana:            600,
	}
	snapshot := SnapshotDelta{
		Tick:       77,
		BaselineID: 2,
		AckSeq:     1500,
		Entities:   []DeltaEntity{entity},
		Removed:    []uint16{9, 10},
		Projectiles: []Projectile{{
			ID: 5, SourceNetID: 1, SkillID: "blood_moon_gu", VFX: "slash",
			X: 320, Y: 640, RadiusX10: 3,
		}},
	}
	encoded, err := EncodeSnapshotDelta(snapshot)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	_, payload, err := DecodeDatagram(encoded)
	if err != nil {
		t.Fatalf("header: %v", err)
	}
	decoded, err := DecodeSnapshotDelta(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	got := decoded.Entities[0]
	if got.HP != entity.HP || got.Shield != entity.Shield || got.Mana != entity.Mana ||
		got.Thought != entity.Thought || got.Revision != entity.Revision ||
		len(got.Statuses) != 3 || got.CastEndTick != entity.CastEndTick ||
		got.FormNetID != entity.FormNetID || got.CurrentFormHP != entity.CurrentFormHP ||
		got.CompanionKind != entity.CompanionKind || got.MaxHP != entity.MaxHP || got.MaxMana != entity.MaxMana {
		t.Fatalf("entity roundtrip mismatch:\n got %+v\nwant %+v", got, entity)
	}
	if len(decoded.Removed) != 2 || decoded.Removed[1] != 10 {
		t.Fatalf("removed = %v", decoded.Removed)
	}
	if len(decoded.Projectiles) != 1 || decoded.Projectiles[0].SkillID != "blood_moon_gu" ||
		decoded.Projectiles[0].X != 320 {
		t.Fatalf("projectiles = %+v", decoded.Projectiles)
	}
	if !decoded.HasProjectiles {
		t.Fatal("HasProjectiles flag lost")
	}
}

func TestQuantizeBounds(t *testing.T) {
	if _, err := Quantize(-1); err == nil {
		t.Fatal("negative coordinate must fail")
	}
	if _, err := Quantize(MaxQuantCoord + 1); err == nil {
		t.Fatal("out-of-range coordinate must fail")
	}
	value, err := Quantize(12.5)
	if err != nil || value != 200 {
		t.Fatalf("quantize = %d err=%v", value, err)
	}
	if Dequantize(value) != 12.5 {
		t.Fatalf("dequantize = %v", Dequantize(value))
	}
}

func TestFrameRoundTripAndTruncation(t *testing.T) {
	first, err := EncodeFrame(OpJoin, []byte{1, 2, 3})
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	second, err := EncodeFrame(OpLeave, nil)
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	buf := append(append([]byte{}, first...), second...)

	op, payload, rest, err := DecodeFrame(buf)
	if err != nil || op != OpJoin || len(payload) != 3 {
		t.Fatalf("first frame op=%d payload=%v err=%v", op, payload, err)
	}
	op, payload, rest, err = DecodeFrame(rest)
	if err != nil || op != OpLeave || len(payload) != 0 || len(rest) != 0 {
		t.Fatalf("second frame op=%d payload=%v rest=%d err=%v", op, payload, len(rest), err)
	}
	if _, _, _, err := DecodeFrame(first[:len(first)-1]); err == nil {
		t.Fatal("truncated frame must fail")
	}
}

func TestDatagramRejectsOtherVersion(t *testing.T) {
	buf := []byte{(3 << 6) | OpInput, 1, 2, 3}
	if _, _, err := DecodeDatagram(buf); err == nil {
		t.Fatal("expected version mismatch")
	}
	if _, err := EncodeDatagram(0x40, nil); err == nil {
		t.Fatal("opcode above 6 bits must fail")
	}
}

func TestEventsRoundTrip(t *testing.T) {
	payload, err := EncodeEvents(100, []Event{
		{EventID: 101, Tick: 5, TypeID: 3, Source: 1, Target: 2, Payload: []byte(`{"damage":12}`)},
		{EventID: 102, Tick: 6, TypeID: 0, Source: 2, Target: 0},
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	from, events, err := DecodeEvents(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if from != 100 || len(events) != 2 || string(events[0].Payload) != `{"damage":12}` ||
		events[1].EventID != 102 {
		t.Fatalf("roundtrip from=%d events=%+v", from, events)
	}
}

func TestGoldenSnapshotDelta(t *testing.T) {
	snapshot := SnapshotDelta{
		Tick:       1234,
		BaselineID: 7,
		AckSeq:     99,
		Entities: []DeltaEntity{
			{NetID: 300, Mask: MaskPosition, X: 1600, Y: 800},
			{
				NetID: 12, Mask: MaskPosition | MaskVitals | MaskRevision,
				X: 32, Y: 64, VitalsSub: VitalHP | VitalMana, HP: 900, Mana: 120,
				Revision: 44,
			},
		},
		Removed: []uint16{5},
	}
	encoded, err := EncodeSnapshotDelta(snapshot)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	const want = "82d204000007006300022c0101400620030c000720004000058407782c010500"
	if got := hex.EncodeToString(encoded); got != want {
		t.Fatalf("golden mismatch:\n got %s\nwant %s", got, want)
	}
}
