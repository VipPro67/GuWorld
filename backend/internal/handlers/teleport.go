package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// TeleportRequest represents the teleport request payload.
type TeleportRequest struct {
	RegionID string `json:"region_id" binding:"required"`
}

// TeleportResponse represents the teleport response.
type TeleportResponse struct {
	Player   *models.Player `json:"player"`
	X        int            `json:"x"`
	Y        int            `json:"y"`
	Region   string         `json:"region"`
	Cost     int64          `json:"cost"`
	Currency string         `json:"currency"` // "primeval_stones" or "immortal_stones"
	Message  string         `json:"message"`
}

// TeleportToRegion handles POST /api/explore/teleport
func TeleportToRegion(c *gin.Context) {
	// Auth: get player ID from context (middleware should have set it)
	playerIDRaw, exists := c.Get("player_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	playerID, ok := playerIDRaw.(uuid.UUID)
	if !ok {
		c.JSON(401, gin.H{"error": "Invalid player ID"})
		return
	}

	var req TeleportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if req.RegionID == "" {
		c.JSON(400, gin.H{"error": "Region ID is required"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback()

	// Load player for update with row lock
	player, err := evaluateManaTx(tx, playerID)
	if err == sql.ErrNoRows {
		c.JSON(404, gin.H{"error": "Player not found"})
		return
	} else if err != nil {
		logger.S.Errorf("Failed to load player %s: %v", playerID, err)
		c.JSON(500, gin.H{"error": "Internal server error"})
		return
	}

	// Validate region exists and active
	destZone := GetZoneByID(req.RegionID)
	if destZone == nil {
		c.JSON(400, gin.H{"error": "Region not found or inactive"})
		return
	}
	if !destZone.IsActive {
		c.JSON(400, gin.H{"error": "Region is inactive"})
		return
	}

	// Cannot teleport while in combat
	if player.ActiveCombat != nil {
		c.JSON(400, gin.H{"error": "Cannot teleport while in combat"})
		return
	}

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(400, gin.H{"error": errLeader.Error()})
		return
	}

	// HP check: must have >20% HP to teleport (same as explore)
	if hpTooLowForTeleport(&player) {
		c.JSON(400, gin.H{"error": "HP too low to teleport"})
		return
	}

	// Get destination safe point
	safeX, safeY := GetSafePoint(req.RegionID)

	// Calculate distance (Chebyshev)
	dx := math.Abs(float64(player.X - safeX))
	dy := math.Abs(float64(player.Y - safeY))
	distance := math.Max(dx, dy)

	// Base cost: 0.01 per unit distance
	baseCost := distance * 0.01

	// Rank multiplier: 1.0 + (rank-1)*0.15 + sub_rank*0.03
	rankMult := 1.0 + float64(player.Rank-1)*0.015 + float64(player.SubRank)*0.003

	// Inventory multiplier: based on storage usage
	stats := CalculatePlayerStats(&player)
	maxSlots := int(stats["storage"])
	if player.Rank >= 6 {
		maxSlots = 999 // Immortals have effectively unlimited storage
	}
	currentItems := len(player.GuInventory)
	// Also count unique items? We'll approximate with GuInventory length as per HasExplorationInventorySpace
	inventoryRatio := 0.0
	if maxSlots > 0 {
		inventoryRatio = float64(currentItems) / float64(maxSlots)
	}
	inventoryMult := 1.0 + inventoryRatio*0.5

	// Total cost
	totalCost := int64(math.Round(baseCost * rankMult * inventoryMult))

	// Determine currency
	var currency string
	var costAmount int64
	var enoughPrimeval bool
	var enoughImmortal bool

	if player.Rank < 6 {
		// Only primeval stones available
		currency = "primeval_stones"
		costAmount = totalCost
		enoughPrimeval = player.PrimevalStones >= float64(costAmount)
		if !enoughPrimeval {
			c.JSON(400, gin.H{"error": fmt.Sprintf("Not enough Primeval Stones. Need %d, have %d", costAmount, player.PrimevalStones)})
			return
		}
	} else {
		// Rank 6+: can use either; prefer immortal for long distances? We'll decide:
		// If distance > 5000, prefer immortal; else primeval.
		// But we must check availability.
		if distance > 5000 {
			// Try immortal first
			currency = "immortal_stones"
			costAmount = totalCost
			enoughImmortal = player.ImmortalStones >= float64(costAmount)
			if enoughImmortal {
				// use immortal
			} else {
				// fall back to primeval if enough
				currency = "primeval_stones"
				costAmount = totalCost
				enoughPrimeval = player.PrimevalStones >= float64(costAmount)
				if !enoughPrimeval {
					c.JSON(400, gin.H{"error": fmt.Sprintf("Not enough stones. Need %d Immortal or Primeval, have Immortal: %d, Primeval: %d", costAmount, player.ImmortalStones, player.PrimevalStones)})
					return
				}
			}
		} else {
			// Prefer primeval for short distances
			currency = "primeval_stones"
			costAmount = totalCost
			enoughPrimeval = player.PrimevalStones >= float64(costAmount)
			if enoughPrimeval {
				// use primeval
			} else {
				// fall back to immortal
				currency = "immortal_stones"
				costAmount = totalCost
				enoughImmortal = player.ImmortalStones >= float64(costAmount)
				if !enoughImmortal {
					c.JSON(400, gin.H{"error": fmt.Sprintf("Not enough stones. Need %d Primeval or Immortal, have Primeval: %d, Immortal: %d", costAmount, player.PrimevalStones, player.ImmortalStones)})
					return
				}
			}
		}
	}

	// Update player position and region atomically with relative currency deduction
	var updateErr error
	if currency == "primeval_stones" {
		player.PrimevalStones -= float64(costAmount)
		_, updateErr = tx.ExecContext(c.Request.Context(), `
			UPDATE players
			SET x = $1, y = $2, current_region = $3,
			    primeval_stones = GREATEST(0, primeval_stones - $4)
			WHERE id = $5
		`, safeX, safeY, req.RegionID, costAmount, player.ID)
	} else {
		player.ImmortalStones -= float64(costAmount)
		_, updateErr = tx.ExecContext(c.Request.Context(), `
			UPDATE players
			SET x = $1, y = $2, current_region = $3,
			    immortal_stones = GREATEST(0, immortal_stones - $4)
			WHERE id = $5
		`, safeX, safeY, req.RegionID, costAmount, player.ID)
	}
	if updateErr != nil {
		logger.S.Errorf("Failed to update player position after teleport: %v", updateErr)
		c.JSON(500, gin.H{"error": "Failed to save position"})
		return
	}

	player.X = safeX
	player.Y = safeY
	player.CurrentRegion = req.RegionID
	SyncPartyMembersLocation(tx, &player)

	if err := tx.Commit(); err != nil {
		c.JSON(500, gin.H{"error": "Failed to commit transaction"})
		return
	}

	player.X = safeX
	player.Y = safeY
	player.CurrentRegion = req.RegionID
	if err != nil {
		logger.S.Errorf("Failed to update player position after teleport: %v", err)
		c.JSON(500, gin.H{"error": "Failed to save position"})
		return
	}

	// Broadcast position update via SSE (world index)
	worldPosIndex.Set(WorldPosEntry{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		Region:    player.CurrentRegion,
		TS:        time.Now().UnixMilli(),
	})

	// Prepare response
	resp := TeleportResponse{
		Player:   &player,
		X:        player.X,
		Y:        player.Y,
		Region:   player.CurrentRegion,
		Cost:     costAmount,
		Currency: currency,
		Message:  fmt.Sprintf("Teleported to %s (safe zone)", destZone.Name),
	}

	c.JSON(200, resp)
}

// Helper: get zone by ID (reuse from zone_registry)
func GetZoneByID(id string) *models.Region {
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()
	return zoneReg.byID[id]
}

// Helper: hpTooLowForTeleport (check if HP > 20% max HP)
func hpTooLowForTeleport(p *models.Player) bool {
	if p.HP <= 0 {
		return true
	}
	maxHP := p.BaseHP + CalculatePlayerStats(p)["hp"]
	if maxHP <= 0 {
		return true
	}
	return p.HP/maxHP <= 0.2
}
