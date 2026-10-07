package models

import "fmt"

// GetRankName returns the Vietnamese name for a cultivation rank.
func GetRankName(rank int) string {
	names := map[int]string{
		0: "Phàm Nhân",
		1: "Nhất Chuyển",
		2: "Nhị Chuyển",
		3: "Tam Chuyển",
		4: "Tứ Chuyển",
		5: "Ngũ Chuyển",
		6: "Lục Chuyển Cổ Tiên",
		7: "Thất Chuyển Cổ Tiên",
		8: "Bát Chuyển Cổ Tiên",
		9: "Cửu Chuyển Cổ Tôn",
	}
	if name, ok := names[rank]; ok {
		return name
	}
	return fmt.Sprintf("%d Chuyển", rank)
}
