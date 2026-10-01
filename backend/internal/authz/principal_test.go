package authz

import (
	"testing"

	"github.com/google/uuid"
)

func TestAdminPrincipalHasPermission(t *testing.T) {
	principal := AdminPrincipal{
		PlayerID:    uuid.New(),
		Permissions: map[string]struct{}{AdminAccessPermission: {}},
	}
	if !principal.HasPermission(AdminAccessPermission) {
		t.Fatal("expected principal to have admin access")
	}
	if principal.HasPermission("admin.unknown") {
		t.Fatal("principal unexpectedly has an unassigned permission")
	}
}

func TestApplyLegacyAdminFallback(t *testing.T) {
	playerID := uuid.New()
	tests := []struct {
		name        string
		principal   AdminPrincipal
		foundPlayer bool
		isAdmin     bool
		wantApplied bool
	}{
		{name: "legacy admin", principal: AdminPrincipal{PlayerID: playerID, Permissions: map[string]struct{}{}}, foundPlayer: true, isAdmin: true, wantApplied: true},
		{name: "explicit role wins", principal: AdminPrincipal{PlayerID: playerID, Roles: []string{"auditor"}, Permissions: map[string]struct{}{}}, foundPlayer: true, isAdmin: true},
		{name: "non admin", principal: AdminPrincipal{PlayerID: playerID, Permissions: map[string]struct{}{}}, foundPlayer: true},
		{name: "missing player", principal: AdminPrincipal{PlayerID: playerID, Permissions: map[string]struct{}{}}, isAdmin: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applied := applyLegacyAdminFallback(&tt.principal, tt.foundPlayer, tt.isAdmin)
			if applied != tt.wantApplied {
				t.Fatalf("fallback applied = %v, want %v", applied, tt.wantApplied)
			}
			if applied && (!tt.principal.HasPermission(AdminAccessPermission) || !tt.principal.HasPermission("players.read") || !tt.principal.LegacyAdminFallback || len(tt.principal.Roles) != 1 || tt.principal.Roles[0] != SuperAdminRole) {
				t.Fatalf("fallback principal was not populated correctly: %+v", tt.principal)
			}
		})
	}
}
