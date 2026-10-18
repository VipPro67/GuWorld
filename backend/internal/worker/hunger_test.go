package worker

import (
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
)

func TestFormatStarvedGuNotification(t *testing.T) {
	// Case 1: Empty
	title, content := FormatStarvedGuNotification(nil)
	if title != "" || content != "" {
		t.Errorf("expected empty for nil, got title=%q, content=%q", title, content)
	}

	// Case 2: Single Gu
	single := []StarvedGuInfo{
		{GuName: "Thăng Tuyền Cổ"},
	}
	title, content = FormatStarvedGuNotification(single)
	if title != "Cổ trùng chết đói" {
		t.Errorf("expected title 'Cổ trùng chết đói', got %q", title)
	}
	if content != "Thăng Tuyền Cổ đã chết vì đói!" {
		t.Errorf("expected content 'Thăng Tuyền Cổ đã chết vì đói!', got %q", content)
	}

	// Case 3: Multiple Gus (matching user image: Thăng Tuyền Cổ, Sơ Giai Sương Tiễn Cổ, Sơ Giai Hỏa Trảo Cổ)
	multi := []StarvedGuInfo{
		{GuName: "Thăng Tuyền Cổ"},
		{GuName: "Sơ Giai Sương Tiễn Cổ"},
		{GuName: "Sơ Giai Hỏa Trảo Cổ"},
	}
	title, content = FormatStarvedGuNotification(multi)
	if title != "Cổ trùng chết đói" {
		t.Errorf("expected title 'Cổ trùng chết đói', got %q", title)
	}
	expected := "3 cổ trùng đã chết vì đói: Thăng Tuyền Cổ, Sơ Giai Sương Tiễn Cổ, Sơ Giai Hỏa Trảo Cổ."
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}

	// Case 4: Duplicate Gu names
	duplicates := []StarvedGuInfo{
		{GuName: "Hỏa Trảo Cổ"},
		{GuName: "Hỏa Trảo Cổ"},
		{GuName: "Băng Tinh Cổ"},
	}
	title, content = FormatStarvedGuNotification(duplicates)
	if !strings.Contains(content, "Hỏa Trảo Cổ x2") {
		t.Errorf("expected 'Hỏa Trảo Cổ x2' in content, got %q", content)
	}
	if !strings.Contains(content, "3 cổ trùng đã chết vì đói") {
		t.Errorf("expected '3 cổ trùng đã chết vì đói' in content, got %q", content)
	}

	// Case 5: More than 5 distinct Gus
	many := []StarvedGuInfo{
		{GuName: "Gu1"},
		{GuName: "Gu2"},
		{GuName: "Gu3"},
		{GuName: "Gu4"},
		{GuName: "Gu5"},
		{GuName: "Gu6"},
		{GuName: "Gu7"},
	}
	title, content = FormatStarvedGuNotification(many)
	if !strings.Contains(content, "7 cổ trùng đã chết vì đói") {
		t.Errorf("expected '7 cổ trùng đã chết vì đói', got %q", content)
	}
	if !strings.Contains(content, "loại khác") {
		t.Errorf("expected 'loại khác' in content, got %q", content)
	}
}

func TestFormatVitalGuNotification(t *testing.T) {
	// Case 1: Empty
	title, content := FormatVitalGuNotification(nil)
	if title != "" || content != "" {
		t.Errorf("expected empty for nil, got title=%q, content=%q", title, content)
	}

	// Case 2: Single vital Gu
	single := []StarvedGuInfo{
		{GuName: "Xuân Thu Thiền", IsVital: true},
	}
	title, content = FormatVitalGuNotification(single)
	if title != "Bản mệnh cổ chết đói" {
		t.Errorf("expected title 'Bản mệnh cổ chết đói', got %q", title)
	}
	expectedSingle := "Bản mệnh cổ Xuân Thu Thiền đã chết vì đói! Bạn nhận 80 sát thương."
	if content != expectedSingle {
		t.Errorf("expected %q, got %q", expectedSingle, content)
	}

	// Case 3: Multiple vital Gus
	multi := []StarvedGuInfo{
		{GuName: "Xuân Thu Thiền", IsVital: true},
		{GuName: "Nguyệt Quang Cổ", IsVital: true},
	}
	title, content = FormatVitalGuNotification(multi)
	expectedMulti := "2 bản mệnh cổ đã chết vì đói: Xuân Thu Thiền, Nguyệt Quang Cổ! Bạn nhận 160 sát thương phản phệ."
	if content != expectedMulti {
		t.Errorf("expected %q, got %q", expectedMulti, content)
	}
}

func TestLazyEvaluationTicksCalculation(t *testing.T) {
	now := time.Now()
	interval := 3 * time.Hour

	// Case 1: Less than 3 hours -> 0 ticks
	lastTick1 := now.Add(-2 * time.Hour)
	ticks1 := int(now.Sub(lastTick1) / interval)
	if ticks1 != 0 {
		t.Errorf("expected 0 ticks for 2 hours, got %d", ticks1)
	}

	// Case 2: Exactly 3 hours -> 1 tick
	lastTick2 := now.Add(-3 * time.Hour)
	ticks2 := int(now.Sub(lastTick2) / interval)
	if ticks2 != 1 {
		t.Errorf("expected 1 tick for 3 hours, got %d", ticks2)
	}

	// Case 3: 7 hours -> 2 ticks, remainder 1 hour preserved
	lastTick3 := now.Add(-7 * time.Hour)
	ticks3 := int(now.Sub(lastTick3) / interval)
	if ticks3 != 2 {
		t.Errorf("expected 2 ticks for 7 hours, got %d", ticks3)
	}
	newTick3 := lastTick3.Add(time.Duration(ticks3) * interval)
	if diff := now.Sub(newTick3); diff < 59*time.Minute || diff > 61*time.Minute {
		t.Errorf("expected remainder ~1 hour, got %v", diff)
	}
}

func TestVaultGuLazyDecay(t *testing.T) {
	items := []vaultGuItem{
		{ID: "GU_TEST_1", Rank: 1, Quantity: 1, Hunger: 100},
		{ID: "GU_TEST_2", Rank: 1, Quantity: 1, Hunger: 10},
	}
	rate := 6.0
	ticks := 2

	// Decay with 2 ticks: loss = 2 * 6 = 12
	for i := range items {
		items[i].Hunger -= float64(ticks) * rate
		if items[i].Hunger < 0 {
			items[i].Hunger = 0
		}
	}

	if items[0].Hunger != 88 {
		t.Errorf("expected GU_TEST_1 hunger 88, got %v", items[0].Hunger)
	}
	if items[1].Hunger != 0 {
		t.Errorf("expected GU_TEST_2 hunger 0 (starved), got %v", items[1].Hunger)
	}

	// Filtering starved
	kept := items[:0]
	for _, it := range items {
		if it.Hunger > 0 {
			kept = append(kept, it)
		}
	}
	if len(kept) != 1 || kept[0].ID != "GU_TEST_1" {
		t.Errorf("expected 1 surviving item (GU_TEST_1), got %d items", len(kept))
	}
}

func TestIsHungerTransientError(t *testing.T) {
	transient := []string{"57014", "40P01", "55P03"}
	for _, code := range transient {
		if !IsHungerTransientError(&pq.Error{Code: pq.ErrorCode(code)}) {
			t.Errorf("IsHungerTransientError(%s) = false, want true", code)
		}
	}
	permanent := []string{"23505", "25P02", "42P01"}
	for _, code := range permanent {
		if IsHungerTransientError(&pq.Error{Code: pq.ErrorCode(code)}) {
			t.Errorf("IsHungerTransientError(%s) = true, want false", code)
		}
	}
	if IsHungerTransientError(nil) {
		t.Error("IsHungerTransientError(nil) = true, want false")
	}
}
