package handlers

import (
	"reflect"
	"testing"
)

func row(id string, valid bool) realtimeGuRow {
	return realtimeGuRow{entry: RealtimeEquippedGu{InstanceID: id, GuID: "GU_" + id, Rank: 1}, valid: valid}
}

func guMap(rows ...realtimeGuRow) map[string]realtimeGuRow {
	m := make(map[string]realtimeGuRow, len(rows))
	for _, r := range rows {
		m[r.entry.InstanceID] = r
	}
	return m
}

func ids(entries []RealtimeEquippedGu) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.InstanceID)
	}
	return out
}

// Valid Gu fill empty slots in equipped order; explicitly pinned slots keep
// their static position; Gu the player no longer owns release their slot.
func TestOrderRealtimeHotbar(t *testing.T) {
	t.Run("no assignment falls back to equipped order", func(t *testing.T) {
		got := orderRealtimeHotbar(nil,
			[]string{"c", "a", "b"},
			guMap(row("c", true), row("a", true), row("b", true)))
		if want := []string{"c", "a", "b"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", got, want)
		}
	})

	t.Run("pins reserve their slot even when invalid", func(t *testing.T) {
		// "b" is pinned at slot 2 but currently sleeping: the slot stays empty
		// (reserved) and must not be stolen by "a" or "c".
		got := orderRealtimeHotbar(
			[]string{"a", "b"},
			[]string{"a", "b", "c"},
			guMap(row("a", true), row("b", false), row("c", true)))
		// Slot 2 keeps an ID-only reserved placeholder for b (no profile, no
		// form skills leak); slot 3 is filled by remaining valid "c".
		if want := []string{"a", "b", "c"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
		if got[1].InstanceID != "b" || got[1].GuID != "" {
			t.Fatalf("reserved pin must be an ID-only placeholder: %+v", got[1])
		}
		if got[2].GuID != "GU_c" {
			t.Fatalf("slot 3 must be filled with c: %+v", got[2])
		}
	})

	t.Run("interior holes never renumber later pinned slots", func(t *testing.T) {
		// Setting "["a", "", "b"]": b stays at Ô3 forever, even with no other
		// equipped Gu to fill the hole yet.
		got := orderRealtimeHotbar(
			[]string{"a", "", "b"},
			[]string{"a", "b"},
			guMap(row("a", true), row("b", true)))
		if want := []string{"a", "", "b"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
		if got[1] != (RealtimeEquippedGu{}) {
			t.Fatalf("interior hole must be a zero placeholder: %+v", got[1])
		}

		// A newly equipped, unpinned c fills the hole; b keeps Ô3.
		got = orderRealtimeHotbar(
			[]string{"a", "", "b"},
			[]string{"a", "b", "c"},
			guMap(row("a", true), row("b", true), row("c", true)))
		if want := []string{"a", "c", "b"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
	})

	t.Run("pins on gone Gu are released and slot is refilled", func(t *testing.T) {
		got := orderRealtimeHotbar(
			[]string{"gone", "b"},
			[]string{"b", "a"},
			guMap(row("a", true), row("b", true)))
		// "gone" leaves the player: slot 1 becomes fillable again and takes the
		// first remaining valid in equipped order; "b" keeps slot 2.
		if want := []string{"a", "b"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
	})

	t.Run("unpinned valid fill the first empty slots", func(t *testing.T) {
		got := orderRealtimeHotbar(
			[]string{"c", ""},
			[]string{"c", "a", "b", "d"},
			guMap(row("c", true), row("a", true), row("b", true), row("d", true)))
		// Explicit layout: Ô1=c, Ô2 empty → filled by equipped order a,b;
		// Ô3=d would fill too but the explicit layout only pinned one empty
		// extra … slot 2 and 3 belong to the layout: slot 2 fills from
		// remaining, slot 3 fills from remaining.
		if want := []string{"c", "a", "b", "d"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
	})

	t.Run("trailing unfilled slots are trimmed", func(t *testing.T) {
		got := orderRealtimeHotbar(
			[]string{"a", "", ""},
			[]string{"a"},
			guMap(row("a", true)))
		if len(got) != 1 || got[0].InstanceID != "a" {
			t.Fatalf("%+v, want only [a]", got)
		}
	})

	t.Run("duplicate pins collapse to the first slot", func(t *testing.T) {
		got := orderRealtimeHotbar(
			[]string{"a", "a"},
			[]string{"a", "b"},
			guMap(row("a", true), row("b", true)))
		if want := []string{"a", "b"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
	})

	t.Run("more equipped than slots keeps explicit pins", func(t *testing.T) {
		got := orderRealtimeHotbar(
			[]string{"f", "e", "d", "c", "b", "a"},
			[]string{"a", "b", "c", "d", "e", "f"},
			guMap(row("a", true), row("b", true), row("c", true), row("d", true), row("e", true), row("f", true)))
		if want := []string{"f", "e", "d", "c", "b", "a"}; !reflect.DeepEqual(ids(got), want) {
			t.Fatalf("%v, want %v", ids(got), want)
		}
	})
}
