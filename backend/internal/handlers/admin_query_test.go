package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestAdminPlayerSearchClauseUsesPrimaryKeyForExactUUID(t *testing.T) {
	id := uuid.MustParse("6f85c649-0f7b-41b5-a288-1ff9be1cdacb")
	where, args := adminPlayerSearchClause("all", id.String())
	if where != "players.id = $1" {
		t.Fatalf("expected primary-key predicate, got %q", where)
	}
	if len(args) != 1 || args[0] != id {
		t.Fatalf("expected parsed UUID argument, got %#v", args)
	}
}

func TestParseAdminPageQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, query, sort, direction string
		page, size, offset           int
	}{
		{"defaults", "", "created_at", "desc", 1, 50, 0},
		{"page size and sort", "?page=3&page_size=25&sort=name&direction=asc", "username", "asc", 3, 25, 50},
		{"unsafe sort falls back", "?sort=created_at%20DROP%20TABLE&direction=nope", "created_at", "desc", 1, 50, 0},
		{"legacy offset", "?limit=20&offset=40", "created_at", "desc", 3, 20, 40},
		{"invalid bounds", "?page=-4&page_size=999", "created_at", "desc", 1, 50, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("GET", "/admin"+tt.query, nil)
			got := parseAdminPageQuery(ctx, 50, map[string]string{"created_at": "created_at", "name": "username"}, "created_at")
			if got.Page != tt.page || got.PageSize != tt.size || got.Offset != tt.offset || got.Sort != tt.sort || got.Direction != tt.direction {
				t.Fatalf("unexpected query: %+v", got)
			}
		})
	}
}
