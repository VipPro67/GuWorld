package handlers

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateAdminRoleChangeInvariants(t *testing.T) {
	admin := adminRole{Key: "admin", Level: 50}
	super := adminRole{Key: "super_admin", Level: 100}
	catalog := []adminRole{admin, super}
	actorID, targetID := uuid.New(), uuid.New()

	tests := []struct {
		name       string
		actorID    uuid.UUID
		targetID   uuid.UUID
		actorRoles []adminRole
		before     []adminRole
		requested  []string
		errText    string
	}{
		{name: "admin grants equal role", actorID: actorID, targetID: targetID, actorRoles: []adminRole{admin}, requested: []string{"admin"}},
		{name: "admin cannot grant super", actorID: actorID, targetID: targetID, actorRoles: []adminRole{admin}, requested: []string{"super_admin"}, errText: "cao hơn"},
		{name: "super can grant super", actorID: actorID, targetID: targetID, actorRoles: []adminRole{super}, requested: []string{"super_admin"}},
		{name: "cannot self elevate", actorID: actorID, targetID: actorID, actorRoles: []adminRole{admin}, before: []adminRole{admin}, requested: []string{"admin", "super_admin"}, errText: "tự nâng quyền"},
		{name: "self demotion is allowed", actorID: actorID, targetID: actorID, actorRoles: []adminRole{super}, before: []adminRole{super}, requested: []string{}},
		{name: "admin cannot demote super", actorID: actorID, targetID: targetID, actorRoles: []adminRole{admin}, before: []adminRole{super}, requested: []string{}, errText: "cao hơn"},
		{name: "unknown role rejected", actorID: actorID, targetID: targetID, actorRoles: []adminRole{super}, requested: []string{"owner"}, errText: "không tồn tại"},
		{name: "duplicate role rejected", actorID: actorID, targetID: targetID, actorRoles: []adminRole{super}, requested: []string{"admin", "admin"}, errText: "bị trùng"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateAdminRoleChange(tt.actorID, tt.targetID, tt.actorRoles, tt.before, tt.requested, catalog)
			if tt.errText == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.errText != "" && (err == nil || !strings.Contains(err.Error(), tt.errText)) {
				t.Fatalf("error = %v, want text %q", err, tt.errText)
			}
		})
	}
}

func TestCannotRemoveLastSuperAdmin(t *testing.T) {
	super := []adminRole{{Key: "super_admin", Level: 100}}
	if !wouldRemoveLastSuperAdmin(super, nil, 1) {
		t.Fatal("last super admin removal must be rejected")
	}
	if wouldRemoveLastSuperAdmin(super, nil, 2) {
		t.Fatal("demotion must be allowed while another super admin remains")
	}
}
