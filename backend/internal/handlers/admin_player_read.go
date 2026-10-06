package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Supported searchBy values: all, id, username, email, item, gu.
// When searchBy is "all", a valid UUID input is auto-detected as an id search.
func adminPlayerSearchClause(searchBy, search string) (string, []interface{}) {
	if search == "" {
		return "", nil
	}

	by := searchBy
	if by == "all" {
		if _, err := uuid.Parse(search); err == nil {
			by = "id"
		}
	}

	like := "%" + search + "%"
	switch by {
	case "id":
		if id, err := uuid.Parse(search); err == nil {
			return `players.id = $1`, []interface{}{id}
		}
		return `players.id::text ILIKE $1`, []interface{}{like}
	case "username":
		return `public.immutable_unaccent(username) ILIKE public.immutable_unaccent($1)`, []interface{}{like}
	case "email":
		return `unaccent(COALESCE(email, '')) ILIKE unaccent($1)`, []interface{}{like}
	case "item":
		itemIDs := models.ItemIDsByName(search)
		if len(itemIDs) > 0 {
			return `EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND (i.item_id ILIKE $1 OR i.item_id = ANY($2)))`, []interface{}{like, itemIDs}
		}
		return `EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND i.item_id ILIKE $1)`, []interface{}{like}
	case "gu":
		guIDs := models.GuIDsByName(search)
		if len(guIDs) > 0 {
			return `EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND (pg.gu_id ILIKE $1 OR pg.gu_id = ANY($2)))`, []interface{}{like, guIDs}
		}
		return `EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND pg.gu_id ILIKE $1)`, []interface{}{like}
	default: // all
		// Prefer username/email matching. For item/gu ownership only use the
		// in-memory registry to resolve exact IDs — an ILIKE fallback here
		// forces a full scan of player_inventory_items / player_gu per search.
		clauses := []string{`public.immutable_unaccent(username) ILIKE public.immutable_unaccent($1)`, `unaccent(COALESCE(email, '')) ILIKE unaccent($1)`}
		args := []interface{}{like}
		argIdx := 2

		itemIDs := models.ItemIDsByName(search)
		if len(itemIDs) > 0 {
			clauses = append(clauses, fmt.Sprintf(`EXISTS (SELECT 1 FROM player_inventory_items i WHERE i.player_id = players.id AND i.item_id = ANY($%d))`, argIdx))
			args = append(args, itemIDs)
			argIdx++
		}

		guIDs := models.GuIDsByName(search)
		if len(guIDs) > 0 {
			clauses = append(clauses, fmt.Sprintf(`EXISTS (SELECT 1 FROM player_gu pg WHERE pg.player_id = players.id AND pg.gu_id = ANY($%d))`, argIdx))
			args = append(args, guIDs)
			argIdx++
		}

		return "(" + strings.Join(clauses, " OR ") + ")", args
	}
}

func AdminListPlayers(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	paging := parseAdminPageQuery(c, 50, map[string]string{
		"created_at": "created_at", "username": "username", "email": "email", "rank": "rank",
		"primeval_stones": "primeval_stones", "immortal_stones": "immortal_stones", "current_region": "current_region",
	}, "created_at")
	search := c.Query("search")
	searchBy := c.DefaultQuery("search_by", "all")

	where, args := adminPlayerSearchClause(searchBy, search)

	var total int
	countQuery := `SELECT COUNT(*) FROM players`
	if where != "" {
		countQuery += ` WHERE ` + where
	}
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		logger.S.Errorf("admin players: count failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đếm số lượng người chơi"})
		return
	}

	query := `SELECT id, username, COALESCE(email, ''), rank, sub_rank, current_mana, max_mana, hp, base_hp, primeval_stones, immortal_stones, COALESCE(is_admin, false), created_at, COALESCE(current_region, ''), COALESCE(aptitude_type, ''), wall_progress, human_qi, lifespan FROM players`
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` ORDER BY ` + paging.Sort + ` ` + paging.Direction + `, id ` + paging.Direction + ` LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)

	dataArgs := append(append([]interface{}{}, args...), paging.PageSize, paging.Offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, dataArgs...)
	if err != nil {
		logger.S.Errorf("admin players: list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách người chơi"})
		return
	}
	defer rows.Close()

	players := []gin.H{}
	for rows.Next() {
		var id uuid.UUID
		var username, email, currentRegion, aptitudeType string
		var rank, subRank int
		var primevalStones float64
		var currentMana, maxMana, hp, baseHP, immortalStones, wallProgress, humanQi, lifespan float64
		var isAdmin bool
		var createdAt time.Time

		if err := rows.Scan(&id, &username, &email, &rank, &subRank, &currentMana, &maxMana, &hp, &baseHP, &primevalStones, &immortalStones, &isAdmin, &createdAt, &currentRegion, &aptitudeType, &wallProgress, &humanQi, &lifespan); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu người chơi"})
			fmt.Println("Error scanning player row:", err)
			return
		}
		players = append(players, gin.H{
			"id":              id,
			"username":        username,
			"email":           email,
			"rank":            rank,
			"sub_rank":        subRank,
			"current_mana":    currentMana,
			"max_mana":        maxMana,
			"hp":              hp,
			"base_hp":         baseHP,
			"primeval_stones": primevalStones,
			"immortal_stones": immortalStones,
			"is_admin":        isAdmin,
			"created_at":      createdAt,
			"current_region":  currentRegion,
			"aptitude_type":   aptitudeType,
			"wall_progress":   wallProgress,
			"human_qi":        humanQi,
			"lifespan":        lifespan,
		})
	}
	if err := rows.Err(); err != nil {
		logger.S.Errorf("admin players: row iteration failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc dữ liệu người chơi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      players,
		"total":     total,
		"page":      paging.Page,
		"page_size": paging.PageSize,
	})
}

func AdminGetPlayerDetail(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	player, err := loadPlayerForStatus(db.DB, playerID, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	var guCount int
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_gu WHERE player_id=$1`, playerID).Scan(&guCount)
	var itemCount int
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM player_inventory_items WHERE player_id=$1`, playerID).Scan(&itemCount)
	var immortalJade float64
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(immortal_jade, 0) FROM wallets WHERE user_id=$1`, playerID).Scan(&immortalJade)

	var selectedTitle, selectedFrame string
	var thoughtRegenRate float64
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(selected_title,''), COALESCE(selected_frame,''), COALESCE(thought_regen_rate,0) FROM players WHERE id=$1`, playerID).Scan(&selectedTitle, &selectedFrame, &thoughtRegenRate)

	passiveStats := GetPassiveStats(&player)

	equippedGus := []gin.H{}
	for _, gu := range player.GuInventory {
		if gu.Equipped {
			name := gu.GuID
			var path []string
			var desc string
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				name = spec.Name
				path = spec.Path
				desc = spec.Description
			} else if spec, ok := models.GetGuAny(gu.GuID); ok {
				name = spec.Name
				path = spec.Path
				desc = spec.Description
			}
			equippedGus = append(equippedGus, gin.H{
				"instance_id":  gu.InstanceID,
				"gu_id":        gu.GuID,
				"name":         name,
				"rank":         gu.Rank,
				"hunger":       gu.Hunger,
				"status":       gu.Status,
				"is_vital":     gu.IsVital,
				"growth_hours": gu.GrowthHours,
				"path":         path,
				"description":  desc,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                    player.ID,
		"username":              player.Username,
		"email":                 player.Email,
		"rank":                  player.Rank,
		"sub_rank":              player.SubRank,
		"aptitude_type":         player.AptitudeType,
		"aptitude_percent":      player.AptitudePercent,
		"aptitude_grade":        player.AptitudeGrade,
		"current_mana":          player.CurrentMana,
		"max_mana":              player.MaxMana,
		"primeval_stones":       player.PrimevalStones,
		"immortal_stones":       player.ImmortalStones,
		"immortal_jade":         immortalJade,
		"hp":                    player.HP,
		"base_hp":               player.BaseHP,
		"wall_progress":         player.WallProgress,
		"human_qi":              player.HumanQi,
		"current_region":        player.CurrentRegion,
		"x":                     player.X,
		"y":                     player.Y,
		"is_admin":              player.IsAdmin,
		"created_at":            player.CreatedAt,
		"lifespan":              player.Lifespan,
		"soul_hp":               player.SoulHP,
		"max_soul":              player.MaxSoul,
		"soul_tier":             player.SoulTier,
		"explore_count":         player.ExploreCount,
		"aperture_stress":       player.ApertureStress,
		"avatar_url":            player.AvatarURL,
		"bio":                   player.Bio,
		"race":                  player.Race,
		"gender":                player.Gender,
		"constitution":          player.Constitution,
		"statuses":              player.Statuses,
		"growth_data":           player.GrowthData,
		"path_masteries":        player.PathMasteries,
		"passive_stats":         passiveStats,
		"equipped_gus":          equippedGus,
		"current_thought":       player.CurrentThought,
		"max_thought":           player.MaxThought,
		"thought_regen_rate":    thoughtRegenRate,
		"time_speed_multiplier": player.TimeSpeedMultiplier,
		"needs_soul_refine":     player.NeedsSoulRefine,
		"selected_title":        selectedTitle,
		"selected_frame":        selectedFrame,
		"fake_rank":             player.FakeRank,
		"fake_sub_rank":         player.FakeSubRank,
		"fake_vital_gu":         player.FakeVitalGu,
		"is_disguised":          player.IsDisguised,
		"needs_onboarding":      player.NeedsOnboarding,
		"gu_count":              guCount,
		"item_count":            itemCount,
	})
}

// AdminListPlayerItems returns a paginated, lazily-loaded list of a player's inventory items.
func AdminListPlayerItems(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM players WHERE id=$1)`, playerID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	where := `player_id = $1`
	args := []interface{}{playerID}
	if search != "" {
		like := "%" + search + "%"
		itemIDs := models.ItemIDsByName(search)
		if len(itemIDs) > 0 {
			where += ` AND (item_id ILIKE $2 OR item_id = ANY($3))`
			args = append(args, like, itemIDs)
		} else {
			where += ` AND item_id ILIKE $2`
			args = append(args, like)
		}
	}

	query := `SELECT item_id, category, quantity, created_at, updated_at FROM player_inventory_items WHERE ` + where + ` ORDER BY item_id ASC`
	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải vật phẩm"})
		return
	}
	defer rows.Close()

	data := []gin.H{}
	for rows.Next() {
		var itemID, category string
		var quantity int
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&itemID, &category, &quantity, &createdAt, &updatedAt); err != nil {
			continue
		}
		spec, ok := models.ItemRegistry[itemID]
		name, itemType := itemID, ""
		rank := 0
		if ok {
			name = spec.Name
			itemType = spec.Type
			rank = spec.Rank
		}
		data = append(data, gin.H{
			"item_id":   itemID,
			"name":      name,
			"item_type": itemType,
			"rank":      rank,
			"category":  category,
			"quantity":  quantity,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	// Include recipe progress rows (Cổ phương), including locked progress, so all fields remain editable.
	if recipeRows, err := db.DB.QueryContext(c.Request.Context(), `SELECT recipe_id, completion_rate, is_unlocked, deduce_count, last_deduce_at, favorite FROM player_recipe_progress WHERE player_id = $1 ORDER BY recipe_id`, playerID); err == nil {
		defer recipeRows.Close()
		for recipeRows.Next() {
			var recipeID string
			var completionRate float64
			var isUnlocked bool
			var deduceCount int
			var lastDeduceAt interface{}
			var favorite bool
			if err := recipeRows.Scan(&recipeID, &completionRate, &isUnlocked, &deduceCount, &lastDeduceAt, &favorite); err != nil {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(recipeID), strings.ToLower(search)) {
				continue
			}
			name := recipeID
			var targetGuID string
			if tg, _, _, err := ParseRecipeID(recipeID); err == nil {
				targetGuID = tg
			} else {
				_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(target_gu_id,'') FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1`, recipeID).Scan(&targetGuID)
			}
			if spec, ok := models.GetGuAny(targetGuID); ok && spec.Name != "" {
				name = spec.Name
			}
			data = append(data, gin.H{
				"item_id":         recipeID,
				"name":            name,
				"item_type":       "recipe",
				"rank":            0,
				"category":        "recipe",
				"quantity":        1,
				"completion_rate": completionRate,
				"is_unlocked":     isUnlocked,
				"deduce_count":    deduceCount,
				"last_deduce_at":  lastDeduceAt,
				"favorite":        favorite,
			})
		}
	}
	total := len(data)
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	data = data[start:end]

	c.JSON(http.StatusOK, gin.H{"data": data, "total": total})
}

// AdminListPlayerGus returns a paginated, lazily-loaded list of a player's Gu instances.
func AdminListPlayerGus(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	var exists bool
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM players WHERE id=$1)`, playerID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}

	start := offset
	end := offset + limit

	// Expose persistence fields used by the admin editor, including read-only ownership metadata.
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, gu_id, gu_rank, rank, level, hunger, state, location, is_locked, equipped, is_vital, integration_time, equipped_at, growth_hours, created_at, updated_at, last_fed_at FROM player_gu WHERE player_id=$1 AND location <> 'ACTOR' ORDER BY created_at ASC, id ASC`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc chi tiết cổ trùng"})
		return
	}
	defer rows.Close()
	all := make([]gin.H, 0)
	for rows.Next() {
		var id, guID, state, location string
		var guRank, rank, level int
		var hunger, integrationTime, growthHours float64
		var locked, equipped, vital bool
		var equippedAt, createdAt, updatedAt, lastFedAt interface{}
		if err := rows.Scan(&id, &guID, &guRank, &rank, &level, &hunger, &state, &location, &locked, &equipped, &vital, &integrationTime, &equippedAt, &growthHours, &createdAt, &updatedAt, &lastFedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc chi tiết cổ trùng"})
			return
		}
		name := guID
		if spec, ok := models.GetGu(guID, rank); ok { name = spec.Name } else if spec, ok := models.GetGuAny(guID); ok { name = spec.Name }
		_ = guRank // returned separately because both rank columns are persisted
		all = append(all, gin.H{"instance_id": id, "gu_id": guID, "name": name, "gu_rank": guRank, "rank": rank, "level": level, "hunger": hunger, "status": state, "location": location, "is_locked": locked, "equipped": equipped, "is_vital": vital, "integration_time": integrationTime, "equipped_at": equippedAt, "growth_hours": growthHours, "created_at": createdAt, "updated_at": updatedAt, "last_fed_at": lastFedAt})
	}
	if err := rows.Err(); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc chi tiết cổ trùng"}); return }
	if search != "" {
		lower := strings.ToLower(search)
		ids := models.GuIDsByName(search)
		filtered := make([]gin.H, 0, len(all))
		for _, row := range all { guID := row["gu_id"].(string); instanceID := row["instance_id"].(string); matched := strings.Contains(strings.ToLower(guID), lower) || strings.Contains(strings.ToLower(instanceID), lower); for _, id := range ids { if guID == id { matched = true; break } }; if matched { filtered = append(filtered, row) } }
		all = filtered
	}
	if start > len(all) { start = len(all) }; if end > len(all) { end = len(all) }
	c.JSON(http.StatusOK, gin.H{"data": all[start:end], "total": len(all)})
}

func AdminSearchGus(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	guSearch := c.Query("gu_id")
	instSearch := c.Query("instance_id")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := (page - 1) * limit

	if guSearch == "" && instSearch == "" {
		c.JSON(http.StatusOK, gin.H{"data": []gin.H{}, "total": 0})
		return
	}

	guWhere := ""
	invWhere := ""
	shopWhere := ""
	args := []interface{}{}
	argIdx := 1
	if guSearch != "" {
		like := "%" + guSearch + "%"
		guIDs := models.GuIDsByName(guSearch)
		if len(guIDs) > 0 {
			guWhere = fmt.Sprintf("(g->>'gu_id' ILIKE $%d OR g->>'gu_id' = ANY($%d))", argIdx, argIdx+1)
			invWhere = fmt.Sprintf("(pg.gu_id ILIKE $%d OR pg.gu_id = ANY($%d))", argIdx, argIdx+1)
			shopWhere = fmt.Sprintf("(s.id ILIKE $%d OR s.id = ANY($%d))", argIdx, argIdx+1)
			args = append(args, like, guIDs)
			argIdx += 2
		} else {
			guWhere = fmt.Sprintf("g->>'gu_id' ILIKE $%d", argIdx)
			invWhere = fmt.Sprintf("pg.gu_id ILIKE $%d", argIdx)
			shopWhere = fmt.Sprintf("s.id ILIKE $%d", argIdx)
			args = append(args, like)
			argIdx++
		}
	}
	if instSearch != "" {
		instClause := fmt.Sprintf("g->>'instance_id' ILIKE $%d", argIdx)
		invInstClause := fmt.Sprintf("pg.id::text ILIKE $%d", argIdx)
		args = append(args, "%"+instSearch+"%")
		argIdx++
		if guWhere == "" {
			guWhere = instClause
		} else {
			guWhere = "(" + guWhere + " OR " + instClause + ")"
		}
		if invWhere == "" {
			invWhere = invInstClause
		} else {
			invWhere = "(" + invWhere + " OR " + invInstClause + ")"
		}
	}

	inventoryBranch := `SELECT 'inventory' AS location,
		p.id AS owner_id, p.username AS owner_username,
		NULL::text AS listing_id,
		pg.id::text AS instance_id, pg.gu_id AS gu_id,
		pg.gu_rank AS rank,
		COALESCE(pg.hunger, 0) AS hunger,
		COALESCE(pg.state, '') AS status,
		COALESCE(pg.equipped, false) AS equipped,
		COALESCE(pg.is_vital, false) AS is_vital,
		'' AS listing_status, NULL::text AS shop_name
	FROM players p
	JOIN player_gu pg ON pg.player_id = p.id
	WHERE ` + invWhere

	auctionBranch := `SELECT 'auction' AS location,
		a.seller_id AS owner_id, pl.username AS owner_username,
		a.id::text AS listing_id,
		g->>'instance_id' AS instance_id, g->>'gu_id' AS gu_id,
		(g->>'rank')::int AS rank,
		COALESCE((g->>'hunger')::int, 0) AS hunger,
		COALESCE(g->>'status', '') AS status,
		COALESCE((g->>'equipped')::boolean, false) AS equipped,
		COALESCE((g->>'is_vital')::boolean, false) AS is_vital,
		COALESCE(a.status, '') AS listing_status, NULL::text AS shop_name
	FROM auction a
	CROSS JOIN LATERAL jsonb_array_elements(
		CASE WHEN jsonb_typeof(a.item_data) = 'array' THEN a.item_data ELSE '[]'::jsonb END
	) g
	JOIN players pl ON pl.id = a.seller_id
	WHERE a.item_type = 'Gu' AND ` + guWhere

	guDictSub := `(SELECT id, MAX(rank) AS rank, MIN(name) AS name FROM gu_dictionary GROUP BY id)`

	shopBranch := `SELECT 'shop' AS location,
		NULL::uuid AS owner_id, NULL::text AS owner_username,
		s.id AS listing_id,
		NULL::text AS instance_id, s.id AS gu_id,
		d.rank AS rank,
		NULL::int AS hunger,
		'SHOP' AS status,
		false AS equipped,
		false AS is_vital,
		'SHOP' AS listing_status, d.name AS shop_name
	FROM caravan_shop_items s
	JOIN ` + guDictSub + ` d ON d.id = s.id
	WHERE ` + shopWhere

	arenaShopBranch := `SELECT 'shop' AS location,
		NULL::uuid AS owner_id, NULL::text AS owner_username,
		s.id AS listing_id,
		NULL::text AS instance_id, s.id AS gu_id,
		d.rank AS rank,
		NULL::int AS hunger,
		'SHOP' AS status,
		false AS equipped,
		false AS is_vital,
		'SHOP' AS listing_status, d.name AS shop_name
	FROM arena_shop_items s
	JOIN ` + guDictSub + ` d ON d.id = s.id
	WHERE ` + shopWhere

	branches := []string{inventoryBranch, auctionBranch}
	if shopWhere != "" {
		branches = append(branches, shopBranch, arenaShopBranch)
	}
	union := strings.Join(branches, " UNION ALL ")

	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM (`+union+`) AS t`, args...).Scan(&total); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tìm cổ trùng"})
		return
	}

	query := `SELECT location, owner_id::text, owner_username, listing_id, instance_id, gu_id, rank, hunger, status, equipped, is_vital, listing_status, shop_name FROM (` + union + `) AS t ORDER BY t.location, t.owner_username, t.gu_id LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	dataArgs := append(append([]interface{}{}, args...), limit, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), query, dataArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tìm cổ trùng"})
		return
	}
	defer rows.Close()

	data := []gin.H{}
	for rows.Next() {
		var location, ownerID, ownerUsername, listingID, instanceID, guID, status, listingStatus, shopName string
		var rank, hunger int
		var equipped, isVital bool
		if err := rows.Scan(&location, &ownerID, &ownerUsername, &listingID, &instanceID, &guID, &rank, &hunger, &status, &equipped, &isVital, &listingStatus, &shopName); err != nil {
			continue
		}
		name := guID
		if spec, ok := models.GetGu(guID, rank); ok {
			name = spec.Name
		} else if spec, ok := models.GetGuAny(guID); ok {
			name = spec.Name
		}
		data = append(data, gin.H{
			"location":       location,
			"owner_id":       ownerID,
			"owner_username": ownerUsername,
			"listing_id":     listingID,
			"instance_id":    instanceID,
			"gu_id":          guID,
			"name":           name,
			"rank":           rank,
			"hunger":         hunger,
			"status":         status,
			"equipped":       equipped,
			"is_vital":       isVital,
			"listing_status": listingStatus,
			"shop_name":      shopName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": data, "total": total})
}
