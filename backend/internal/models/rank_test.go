package models

import "testing"

func TestVinhSinhRank(t *testing.T) {
	if got := GetRankName(10); got != "Vĩnh Sinh" {
		t.Errorf("GetRankName(10) = %q; want %q", got, "Vĩnh Sinh")
	}
	for _, rank := range []int{0, 5, 9} {
		if IsVinhSinhRank(rank) {
			t.Errorf("IsVinhSinhRank(%d) = true; want false", rank)
		}
	}
	if !IsVinhSinhRank(10) || !IsVinhSinhRank(11) {
		t.Error("IsVinhSinhRank must be true for rank >= 10")
	}
}
