package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

const adminMaxPageSize = 200

type adminPageQuery struct {
	Page      int
	PageSize  int
	Offset    int
	Sort      string
	Direction string
}

func parseAdminPageQuery(c *gin.Context, defaultSize int, allowedSort map[string]string, defaultSort string) adminPageQuery {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	sizeValue := c.Query("page_size")
	if sizeValue == "" {
		sizeValue = c.Query("limit")
	}
	pageSize, err := strconv.Atoi(sizeValue)
	if err != nil || pageSize < 1 || pageSize > adminMaxPageSize {
		pageSize = defaultSize
	}
	if offsetValue := c.Query("offset"); c.Query("page") == "" && offsetValue != "" {
		if offset, parseErr := strconv.Atoi(offsetValue); parseErr == nil && offset >= 0 {
			page = offset/pageSize + 1
		}
	}
	sortKey := c.Query("sort")
	sortColumn, ok := allowedSort[sortKey]
	if !ok {
		sortKey = defaultSort
		sortColumn = allowedSort[defaultSort]
	}
	direction := strings.ToLower(c.DefaultQuery("direction", "desc"))
	if direction != "asc" {
		direction = "desc"
	}
	return adminPageQuery{Page: page, PageSize: pageSize, Offset: (page - 1) * pageSize, Sort: sortColumn, Direction: direction}
}

// fastCountTable returns the total row count for an unfiltered table,
// checking Redis first, then PostgreSQL pg_class.reltuples estimation, and falling back
// to exact COUNT(*) if necessary. Caches the result in Redis for cacheTTL.
func fastCountTable(c *gin.Context, tableName string, cacheKey string, cacheTTL time.Duration) (int, error) {
	if cacheKey != "" {
		var cachedCount int
		if err := cache.GetJSON(cacheKey, &cachedCount); err == nil && cachedCount > 0 {
			return cachedCount, nil
		}
	}

	// Try fast catalog estimation from pg_class (0.1ms, no sequential table scan)
	var estimatedCount int64
	errEst := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(reltuples::bigint, 0) FROM pg_class WHERE relname = $1`, tableName).Scan(&estimatedCount)
	if errEst == nil && estimatedCount > 0 {
		count := int(estimatedCount)
		if cacheKey != "" {
			_ = cache.SetJSON(cacheKey, count, cacheTTL)
		}
		return count, nil
	}

	// Fallback to exact count if statistics are not yet populated
	var total int
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COUNT(*) FROM `+tableName).Scan(&total); err != nil {
		return 0, err
	}
	if cacheKey != "" {
		_ = cache.SetJSON(cacheKey, total, cacheTTL)
	}
	return total, nil
}
