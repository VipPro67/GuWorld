package models

import "fmt"

// GetRankName returns the Vietnamese name for a cultivation rank.
func GetRankName(rank int) string {
	names := map[int]string{
		0:  "Phàm Nhân",
		1:  "Nhất Chuyển",
		2:  "Nhị Chuyển",
		3:  "Tam Chuyển",
		4:  "Tứ Chuyển",
		5:  "Ngũ Chuyển",
		6:  "Lục Chuyển Cổ Tiên",
		7:  "Thất Chuyển Cổ Tiên",
		8:  "Bát Chuyển Cổ Tiên",
		9:  "Cửu Chuyển Cổ Tôn",
		10: "Vĩnh Sinh",
	}
	if name, ok := names[rank]; ok {
		return name
	}
	return fmt.Sprintf("%d Chuyển", rank)
}

// IsVinhSinhRank reports whether the rank is the eternal-life "Vĩnh Sinh"
// realm (Rank 10). Players at this rank face no tribulations and never lose
// lifespan to aging; all other gameplay runs normally.
func IsVinhSinhRank(rank int) bool {
	return rank >= 10
}
