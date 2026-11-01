package gamefx

import "testing"

// RefineSessionTimeScale: mỗi cấp Chân Ý Luyện Đạo giảm 5% tổng thời gian
// phiên, trần 20%, không âm sàn dưới 0.5.
func TestRefineSessionTimeScale(t *testing.T) {
	cases := []struct {
		realm int
		want  float64
	}{
		{0, 1.0},  // Ordinary — giữ nguyên
		{1, 0.95}, // Master
		{2, 0.90}, // Grandmaster
		{3, 0.85}, // Great Grandmaster
		{4, 0.80}, // Supreme Grandmaster — trần −20%
		{7, 0.80}, // vượt trần vẫn kẹp tại cap
	}
	for _, c := range cases {
		if got := RefineSessionTimeScale(c.realm); got != c.want {
			t.Errorf("RefineSessionTimeScale(%d) = %v, want %v", c.realm, got, c.want)
		}
	}
}
