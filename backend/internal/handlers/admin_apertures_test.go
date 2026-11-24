package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

func TestAdminListAperturesCoordinates(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if os.Getenv("DB_PASSWORD") == "" {
		t.Setenv("DB_PASSWORD", "gu_pass_123")
	}
	if err := db.Connect(); err != nil {
		t.Skipf("cannot connect to db: %v", err)
	}

	testRegionID := "test_region_aperture_" + uuid.New().String()[:8]
	testRegionName := "Vực Kiểm Thử " + testRegionID

	// Seed test region
	_, err := db.DB.Exec(`
		INSERT INTO regions (id, name, width, height, min_rank, max_rank)
		VALUES ($1, $2, 1000, 1000, 1, 9)
		ON CONFLICT (id) DO NOTHING
	`, testRegionID, testRegionName)
	if err != nil {
		t.Fatalf("seeding test region: %v", err)
	}
	defer func() {
		_, _ = db.DB.Exec(`DELETE FROM regions WHERE id = $1`, testRegionID)
	}()

	// Seed an ownerless immortal aperture
	var ownerlessApertureID string
	err = db.DB.QueryRow(`
		INSERT INTO immortal_apertures (
			player_id, aperture_rank, public_state, grade, time_rate,
			aperture_hp, max_aperture_hp, area_size, immortal_essence,
			state, is_placed, placed_region, placed_x, placed_y
		)
		VALUES (NULL, 6, 'PUBLIC', 'HIGH', 15, 20000, 20000, 5000000, 1000, 'STABLE', TRUE, $1, 612.5, 834.5)
		RETURNING id::text
	`, testRegionID).Scan(&ownerlessApertureID)
	if err != nil {
		t.Fatalf("seeding ownerless aperture: %v", err)
	}
	defer func() {
		_, _ = db.DB.Exec(`DELETE FROM aperture_portals WHERE aperture_id = $1`, ownerlessApertureID)
		_, _ = db.DB.Exec(`DELETE FROM immortal_apertures WHERE id = $1`, ownerlessApertureID)
	}()

	// Seed portal for this aperture
	_, err = db.DB.Exec(`
		INSERT INTO aperture_portals (aperture_id, owner_id, region_id, x, y, name, access_policy, is_active)
		VALUES ($1, NULL, $2, 612.5, 834.5, 'Môn Hộ Tiên Khiếu Vô Chủ Test', 'OPEN', TRUE)
		ON CONFLICT (aperture_id) DO UPDATE SET region_id = $2, x = 612.5, y = 834.5, is_active = TRUE
	`, ownerlessApertureID, testRegionID)
	if err != nil {
		t.Fatalf("seeding portal: %v", err)
	}

	// 1. Test AdminListApertures
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/apertures?search="+testRegionID, nil)
		setTestAdminPrincipal(c, uuid.New())

		AdminListApertures(c)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var resp struct {
			Data []map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Data) == 0 {
			t.Fatalf("expected at least 1 aperture matching %s, got 0", testRegionID)
		}

		found := false
		for _, row := range resp.Data {
			if row["id"] == ownerlessApertureID {
				found = true
				if row["placed_region"] != testRegionID {
					t.Errorf("expected placed_region %s, got %v", testRegionID, row["placed_region"])
				}
				if row["region_name"] != testRegionName {
					t.Errorf("expected region_name %s, got %v", testRegionName, row["region_name"])
				}
				if row["placed_x"] != 612.5 {
					t.Errorf("expected placed_x 612.5, got %v", row["placed_x"])
				}
				if row["placed_y"] != 834.5 {
					t.Errorf("expected placed_y 834.5, got %v", row["placed_y"])
				}
				if row["is_placed"] != true {
					t.Errorf("expected is_placed true, got %v", row["is_placed"])
				}
				break
			}
		}
		if !found {
			t.Errorf("ownerless aperture %s not found in list response", ownerlessApertureID)
		}
	}

	// 2. Test AdminGetApertureDetail
	{
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/apertures/"+ownerlessApertureID, nil)
		c.Params = gin.Params{{Key: "id", Value: ownerlessApertureID}}
		setTestAdminPrincipal(c, uuid.New())

		AdminGetApertureDetail(c)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
		}

		var detailResp struct {
			Aperture     map[string]interface{} `json:"aperture"`
			RegionName   string                 `json:"region_name"`
			PlacedRegion string                 `json:"placed_region"`
			PlacedX      *float64               `json:"placed_x"`
			PlacedY      *float64               `json:"placed_y"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &detailResp); err != nil {
			t.Fatalf("failed to decode detail response: %v", err)
		}

		if detailResp.PlacedRegion != testRegionID {
			t.Errorf("expected PlacedRegion %s, got %s", testRegionID, detailResp.PlacedRegion)
		}
		if detailResp.RegionName != testRegionName {
			t.Errorf("expected RegionName %s, got %s", testRegionName, detailResp.RegionName)
		}
		if detailResp.PlacedX == nil || *detailResp.PlacedX != 612.5 {
			t.Errorf("expected PlacedX 612.5, got %v", detailResp.PlacedX)
		}
		if detailResp.PlacedY == nil || *detailResp.PlacedY != 834.5 {
			t.Errorf("expected PlacedY 834.5, got %v", detailResp.PlacedY)
		}
		if detailResp.Aperture["is_placed"] != true {
			t.Errorf("expected aperture.is_placed true, got %v", detailResp.Aperture["is_placed"])
		}
	}
}
