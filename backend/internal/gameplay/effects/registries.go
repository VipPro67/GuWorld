package effects

import (
	"sort"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

// Statuses is a compatibility projection of the authoritative effect catalog.
func Statuses() []string {
	result := make([]string, 0, len(effectcatalog.StatusCatalog))
	for id := range effectcatalog.StatusCatalog {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}
