package handlers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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
