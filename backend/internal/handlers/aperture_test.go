package handlers

import "testing"

func TestApplyBulkReward(t *testing.T) {
	cases := []struct {
		base     int64
		combined float64
		bulk     float64
		want     int64
	}{
		{base: 100, combined: 1.0, bulk: 1.0, want: 100},
		{base: 100, combined: 1.0, bulk: 0.8, want: 80},
		{base: 10, combined: 1.0, bulk: 0.8, want: 8},
		{base: 1, combined: 1.0, bulk: 0.8, want: 1},
		{base: 3, combined: 0.5, bulk: 0.8, want: 1},
		{base: 0, combined: 1.0, bulk: 0.8, want: 0},
	}
	for _, c := range cases {
		got := applyBulkReward(c.base, c.combined, c.bulk)
		if got != c.want {
			t.Fatalf("applyBulkReward(%d, %v, %v) = %d, want %d", c.base, c.combined, c.bulk, got, c.want)
		}
	}
}

func TestComputeBulkRepairCost(t *testing.T) {
	cases := []struct {
		hp, maxHP int
		destroyed bool
		fee       float64
		wantBase  int
		wantTotal int
	}{
		{hp: 500, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 10, wantTotal: 13},
		{hp: 0, maxHP: 1000, destroyed: true, fee: 1.3, wantBase: 0, wantTotal: 0},
		{hp: 1000, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 0, wantTotal: 0},
		{hp: 900, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 2, wantTotal: 3},
		{hp: 999, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 1, wantTotal: 2},
	}
	for _, c := range cases {
		base, total := computeBulkRepairCost(c.hp, c.maxHP, c.destroyed, c.fee)
		if base != c.wantBase || total != c.wantTotal {
			t.Fatalf("computeBulkRepairCost(%d, %d, %v, %v) = (%d, %d), want (%d, %d)", c.hp, c.maxHP, c.destroyed, c.fee, base, total, c.wantBase, c.wantTotal)
		}
	}
}

func TestRaceHasDeadApertureSuppressesTribulations(t *testing.T) {
	if !raceHasDeadAperture("cuong_thi") {
		t.Fatal("cuong_thi must have a dead aperture and be exempt from tribulations")
	}
	if raceHasDeadAperture("human") {
		t.Fatal("human must continue to face tribulations")
	}
}
