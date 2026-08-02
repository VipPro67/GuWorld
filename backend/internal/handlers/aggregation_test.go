package handlers

import (
	"strings"
	"testing"
)

func TestAggregateGachaRewards(t *testing.T) {
	rewards := []GachaResultItem{
		{Type: "primeval_stones", Amount: 728, Label: "728 Nguyên Thạch"},
		{Type: "item", ID: "kim_chung_trao_co", Amount: 1, Label: "1x Kim Chung Tráo Cổ"},
		{Type: "primeval_stones", Amount: 4927, Label: "4927 Nguyên Thạch"},
		{Type: "primeval_stones", Amount: 1384, Label: "1384 Nguyên Thạch"},
		{Type: "item", ID: "kim_chung_trao_co", Amount: 1, Label: "1x Kim Chung Tráo Cổ"},
	}

	result := aggregateGachaRewards(rewards)
	t.Logf("Result:\n%s", result)

	// We expect "Nguyên Thạch" to appear only once, and amount to be 728 + 4927 + 1384 = 7039.
	if !strings.Contains(result, "7039 Nguyên Thạch") {
		t.Errorf("Expected combined primeval stones of 7039, got: %s", result)
	}

	// We expect "kim_chung_trao_co" to be grouped with count 2.
	if !strings.Contains(result, "2x kim_chung_trao_co") {
		t.Errorf("Expected grouped items '2x kim_chung_trao_co', got: %s", result)
	}
}

func TestAggregateChestSpecialMessages(t *testing.T) {
	messages := []string{
		"Mở Rương mỏ thành công!\n- 💎 Nhận được 230 Nguyên Thạch\n- Nhận được 1x Nguyên Thạch Vụn\n",
		"Mở Rương mỏ thành công!\n- 💎 Nhận được 150 Nguyên Thạch\n- Nhận được 1x Đồng Nát\n",
		"Mở Rương mỏ thành công!\n- Nhận được 1x Nguyên Thạch Vụn\n",
	}

	result := aggregateChestSpecialMessages(messages)
	t.Logf("Result lines: %v", result)

	// We expect header "Mở 3 lần Rương mỏ thành công!"
	if !containsSlice(result, "Mở 3 lần Rương mỏ thành công!") {
		t.Errorf("Expected header 'Mở 3 lần Rương mỏ thành công!'")
	}

	// We expect "- 💎 Nhận được 380 Nguyên Thạch"
	if !containsSlice(result, "- 💎 Nhận được 380 Nguyên Thạch") {
		t.Errorf("Expected stones total to be 380")
	}

	// We expect "- Nhận được 2x Nguyên Thạch Vụn"
	if !containsSlice(result, "- Nhận được 2x Nguyên Thạch Vụn") {
		t.Errorf("Expected '- Nhận được 2x Nguyên Thạch Vụn'")
	}

	// We expect "- Nhận được 1x Đồng Nát"
	if !containsSlice(result, "- Nhận được 1x Đồng Nát") {
		t.Errorf("Expected '- Nhận được 1x Đồng Nát'")
	}
}

func containsSlice(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
