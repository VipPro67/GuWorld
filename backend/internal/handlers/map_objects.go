package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// MapMarkerView is the unified contract for a static structure marker on the
// world map (sects today, NPCs/mines/merchants later). Dynamic entities keep
// their existing transports: veins and ground drops arrive through
// /explore/cells/rect, monsters through the realtime world snapshot.
//
// Coordinates are local map cells of RegionID, matching players.x/y and the
// cells feed. See docs/guides/GAME_DATA.md → World coordinates.
type MapMarkerView struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	RegionID string `json:"region_id"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Name     string `json:"name"`
	Detail   any    `json:"detail,omitempty"`
}

type SectMarker struct {
	Level       int            `json:"level"`
	ShieldUntil *time.Time     `json:"shield_until"`
	Buildings   map[string]int `json:"buildings"`
	IsMine      bool           `json:"is_mine"`
}

type TeleportMarker struct {
	OwnerID      string `json:"owner_id"`
	Tier         string `json:"tier"`
	AccessPolicy string `json:"access_policy"`
	IsActive     bool   `json:"is_active"`
}

type AperturePortalMarker struct {
	OwnerID          string `json:"owner_id,omitempty"`
	ApertureID       string `json:"aperture_id,omitempty"`
	AccessPolicy     string `json:"access_policy,omitempty"`
	ConcealmentPower int    `json:"concealment_power,omitempty"`
	IsMine           bool   `json:"is_mine"`
}

// sectMarkersInRegion loads sect bases for one map. Defenders are deliberately
// not loaded: markers only need position, level and shield state, and the
// defender list costs one query per sect in GetSectsMap.
func sectMarkersInRegion(regionID string, mySectID *string) []MapMarkerView {
	rows, err := db.DB.Query(`
		SELECT s.id, s.name, s.level, s.map_x, s.map_y, s.shield_until, s.buildings
		FROM sects s
		WHERE s.region = $1`, regionID)
	if err != nil {
		logger.S.Warnf("Failed to load sect markers for region %s: %v", regionID, err)
		return nil
	}
	defer rows.Close()

	out := make([]MapMarkerView, 0)
	for rows.Next() {
		var (
			id, name string
			level    int
			x, y     int
			shield   sql.NullTime
			raw      []byte
		)
		if err := rows.Scan(&id, &name, &level, &x, &y, &shield, &raw); err != nil {
			continue
		}
		detail := SectMarker{Level: level, Buildings: map[string]int{}}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &detail.Buildings)
		}
		if shield.Valid {
			t := shield.Time
			detail.ShieldUntil = &t
		}
		if mySectID != nil && *mySectID == id {
			detail.IsMine = true
		}
		out = append(out, MapMarkerView{
			ID:       id,
			Kind:     "sect",
			RegionID: regionID,
			X:        x,
			Y:        y,
			Name:     name,
			Detail:   detail,
		})
	}
	return out
}

// teleportMarkersInRegion returns the arrays this player may see on the map.
// Visibility mirrors GetMyTeleportArrays (owner OR public OR own sect); private
// arrays of other players must never leak into a marker feed.
func teleportMarkersInRegion(regionID string, authID uuid.UUID, mySectID *string) []MapMarkerView {
	query := `
		SELECT t.id, t.name, t.owner_id, t.x, t.y, t.tier, t.access_policy, t.is_active
		FROM teleport_arrays t
		WHERE t.is_active = TRUE AND t.region_id = $2
		  AND (t.owner_id = $1 OR t.access_policy = 'PUBLIC'`
	args := []interface{}{authID, regionID}
	if mySectID != nil && *mySectID != "" {
		query += " OR (t.sect_id = $3 AND t.access_policy = 'SECT')"
		args = append(args, *mySectID)
	}
	query += ")"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		logger.S.Warnf("Failed to load teleport markers for region %s: %v", regionID, err)
		return nil
	}
	defer rows.Close()

	out := make([]MapMarkerView, 0)
	for rows.Next() {
		var (
			id, name, ownerID string
			x, y              int
			detail            TeleportMarker
		)
		if err := rows.Scan(&id, &name, &ownerID, &x, &y, &detail.Tier, &detail.AccessPolicy, &detail.IsActive); err != nil {
			continue
		}
		detail.OwnerID = ownerID
		out = append(out, MapMarkerView{
			ID:       id,
			Kind:     "teleport",
			RegionID: regionID,
			X:        x,
			Y:        y,
			Name:     name,
			Detail:   detail,
		})
	}
	return out
}

// aperturePortalMarkersInRegion returns active immortal aperture gateways in this region.
// Observers see portals they own, portals exposed by their party/sect policy,
// generally visible policies, or portals their scouting score can discover.
func aperturePortalMarkersInRegion(regionID, viewerRegion string, authID uuid.UUID, mySectID, myPartyID *string, viewer *models.Player) []MapMarkerView {
	rows, err := db.DB.Query(`
		SELECT p.id, p.name, p.owner_id, p.aperture_id, p.x, p.y, p.access_policy, p.concealment_power,
		       host.rank, host.path_masteries, COALESCE(host.race, ''), COALESCE(host.constitution, ''),
		       COALESCE(host.statuses, '{}'::jsonb), host.hp, host.base_hp, host.current_mana, host.max_mana,
		       COALESCE(owner_aperture.dao_marks, '{}'::jsonb), `+db.GuInventoryAggregateEquipped("host.id")+`,
		       (p.owner_id = $1) AS is_mine,
		       COALESCE(p.access_policy = 'SECT' AND EXISTS (
		           SELECT 1 FROM sect_members sm WHERE sm.player_id = p.owner_id AND sm.sect_id = $3
		       ), false) AS same_sect,
		       COALESCE(p.access_policy = 'PARTY' AND EXISTS (
		           SELECT 1 FROM players host WHERE host.id = p.owner_id AND host.party_id = $4 AND host.party_id IS NOT NULL
		       ), false) AS same_party
		FROM aperture_portals p
		JOIN players host ON host.id = p.owner_id
		LEFT JOIN immortal_apertures owner_aperture ON owner_aperture.player_id = host.id
		WHERE p.is_active = TRUE AND p.region_id = $2`, authID, regionID, nullableString(mySectID), nullableString(myPartyID))
	if err != nil {
		logger.S.Warnf("Failed to load aperture portal markers for region %s: %v", regionID, err)
		return nil
	}
	defer rows.Close()

	out := make([]MapMarkerView, 0)
	for rows.Next() {
		var (
			id, name, ownerID, apertureID string
			x, y                          float64
			owner                         models.Player
			detail                        AperturePortalMarker
			isSameSect, isSameParty       bool
		)
		if err := rows.Scan(&id, &name, &ownerID, &apertureID, &x, &y, &detail.AccessPolicy, &detail.ConcealmentPower,
			&owner.Rank, &owner.PathMasteries, &owner.Race, &owner.Constitution, &owner.Statuses, &owner.HP, &owner.BaseHP, &owner.CurrentMana, &owner.MaxMana, &owner.DaoMarks, &owner.GuInventory,
			&detail.IsMine, &isSameSect, &isSameParty); err != nil {
			continue
		}
		detail.IsMine = (ownerID == authID.String())
		owner.CurrentRegion = regionID
		scoutingPower := gamefx.AperturePortalDiscoveryPower(viewer, models.FilterEquippedActive(viewer.GuInventory), viewer.DaoMarks, &owner)
		portalDefense := gamefx.AperturePortalConcealmentPower(&owner, models.FilterEquippedActive(owner.GuInventory), owner.DaoMarks, viewer, regionID, viewerRegion, detail.ConcealmentPower)
		policyVisible := aperturePortalPolicyVisible(detail.AccessPolicy, detail.IsMine, isSameSect, isSameParty)
		if !policyVisible && !gamefx.CanDiscoverAperturePortal(scoutingPower, portalDefense) {
			continue
		}
		marker := aperturePortalMarker(id, regionID, int(x), int(y), name, detail, ownerID, apertureID, policyVisible)

		out = append(out, marker)
	}
	return out
}

func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func aperturePortalPolicyVisible(policy string, isMine, sameSect, sameParty bool) bool {
	return isMine || policy == "OPEN" || policy == "KEY" || policy == "APPROVAL" || policy == "SECT" && sameSect || policy == "PARTY" && sameParty
}

func aperturePortalMarker(id, regionID string, x, y int, name string, detail AperturePortalMarker, ownerID, apertureID string, policyVisible bool) MapMarkerView {
	if !policyVisible && !detail.IsMine {
		name = "Môn Hộ ẩn"
		detail.OwnerID = ""
		detail.ApertureID = ""
		detail.AccessPolicy = ""
		detail.ConcealmentPower = 0
	} else {
		detail.OwnerID = ownerID
		detail.ApertureID = apertureID
	}
	return MapMarkerView{ID: id, Kind: "aperture_portal", RegionID: regionID, X: x, Y: y, Name: name, Detail: detail}
}

// merchantMarkersInRegion returns active placed merchants located on the map
// (e.g. novice valley merchants at safe zones or roaming traders).
func merchantMarkersInRegion(regionID string) []MapMarkerView {
	if db.DB == nil {
		return nil
	}
	rows, err := db.DB.Query(`
		SELECT m.id, p.name, m.x, m.y
		FROM merchants m
		JOIN merchant_profiles p ON p.id = m.profile_id
		WHERE m.region_id = $1 AND m.is_active AND p.enabled
		  AND (m.expires_at IS NULL OR m.expires_at > NOW())
		  AND (m.x > 0 OR m.y > 0)`, regionID)
	if err != nil {
		logger.S.Warnf("Failed to load merchant markers for region %s: %v", regionID, err)
		return nil
	}
	defer rows.Close()

	out := make([]MapMarkerView, 0)
	for rows.Next() {
		var (
			id, name string
			x, y     int
		)
		if err := rows.Scan(&id, &name, &x, &y); err != nil {
			continue
		}
		out = append(out, MapMarkerView{
			ID:       id,
			Kind:     "merchant",
			RegionID: regionID,
			X:        x,
			Y:        y,
			Name:     name,
		})
	}
	return out
}

// GetZoneMapObjects returns static structure markers for one map. The client
// merges this with the cells feed and the realtime snapshot into a single
// marker view-model, so the map and the radar cannot drift apart.
func GetZoneMapObjects(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	viewer, err := playerstate.Load(db.DB, authID, playerstate.Options{Gu: playerstate.GuEquipped})
	if err != nil {
		if playerstate.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi."})
			return
		}
		httperr.Respond(c, httperr.Internal("Không thể tải năng lực trinh sát.", err))
		return
	}
	regionID := strings.TrimSpace(c.Query("region"))
	if regionID == "" {
		regionID = viewer.CurrentRegion
	}
	if regionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu khu vực."})
		return
	}

	_ = ensureRegionMerchants(regionID)

	mySectID := getPlayerSectID(nil, authID)
	var myPartyID *string
	if viewer.PartyID != nil {
		partyID := viewer.PartyID.String()
		myPartyID = &partyID
	}

	out := make([]MapMarkerView, 0)
	out = append(out, sectMarkersInRegion(regionID, mySectID)...)
	out = append(out, teleportMarkersInRegion(regionID, authID, mySectID)...)
	out = append(out, aperturePortalMarkersInRegion(regionID, viewer.CurrentRegion, authID, mySectID, myPartyID, &viewer)...)
	out = append(out, merchantMarkersInRegion(regionID)...)
	c.JSON(http.StatusOK, out)
}
