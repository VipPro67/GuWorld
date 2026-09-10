package worker

import (
	"strings"
	"testing"
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
