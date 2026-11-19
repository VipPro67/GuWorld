package handlers

import "testing"

func TestAperturePortalPolicyVisibility(t *testing.T) {
	tests := []struct {
		name      string
		policy    string
		isMine    bool
		sameSect  bool
		sameParty bool
		want      bool
	}{
		{name: "owner sees hidden portal", policy: "HIDDEN", isMine: true, want: true},
		{name: "open portal is visible", policy: "OPEN", want: true},
		{name: "sect member sees sect portal", policy: "SECT", sameSect: true, want: true},
		{name: "party member sees party portal", policy: "PARTY", sameParty: true, want: true},
		{name: "hidden portal requires scouting", policy: "HIDDEN", want: false},
		{name: "sect portal is hidden from outsiders", policy: "SECT", want: false},
		{name: "party portal is hidden from outsiders", policy: "PARTY", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aperturePortalPolicyVisible(tt.policy, tt.isMine, tt.sameSect, tt.sameParty); got != tt.want {
				t.Fatalf("aperturePortalPolicyVisible(%q, %v, %v, %v) = %v, want %v", tt.policy, tt.isMine, tt.sameSect, tt.sameParty, got, tt.want)
			}
		})
	}
}

func TestAperturePortalRequiresScoutingWhenOwnerlessAndPublic(t *testing.T) {
	tests := []struct {
		name        string
		ownerID     string
		publicState string
		want        bool
	}{
		{name: "public ownerless portal", publicState: "PUBLIC", want: true},
		{name: "ruin ownerless portal", publicState: "RUIN", want: true},
		{name: "closing ownerless portal", publicState: "CLOSING"},
		{name: "private ownerless portal", publicState: "PRIVATE"},
		{name: "owned public aperture", ownerID: "owner-id", publicState: "PUBLIC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aperturePortalRequiresScouting(tt.ownerID, tt.publicState); got != tt.want {
				t.Fatalf("aperturePortalRequiresScouting(%q, %q) = %v, want %v", tt.ownerID, tt.publicState, got, tt.want)
			}
		})
	}
}

func TestAperturePortalVisibilityRequiresScoutingForPublicSites(t *testing.T) {
	tests := []struct {
		name             string
		policyVisible    bool
		canDiscover      bool
		requiresScouting bool
		want             bool
	}{
		{name: "public OPEN portal remains hidden until scouted", policyVisible: false, requiresScouting: true, want: false},
		{name: "scouting reveals public portal", canDiscover: true, requiresScouting: true, want: true},
		{name: "owned OPEN portal keeps policy visibility", policyVisible: true, want: true},
		{name: "private hidden portal can be found by scouting", canDiscover: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aperturePortalVisibleToViewer(tt.policyVisible, tt.canDiscover, tt.requiresScouting); got != tt.want {
				t.Fatalf("aperturePortalVisibleToViewer(%v, %v, %v) = %v, want %v", tt.policyVisible, tt.canDiscover, tt.requiresScouting, got, tt.want)
			}
		})
	}
}

func TestScoutedAperturePortalMarkerOnlyRevealsLocation(t *testing.T) {
	detail := AperturePortalMarker{AccessPolicy: "HIDDEN", ConcealmentPower: 12}
	marker := aperturePortalMarker("portal-id", "Southern_Border", 21, 34, "Hidden Garden", detail, "owner-id", "aperture-id", false)
	if marker.Name != "Môn Hộ ẩn" || marker.X != 21 || marker.Y != 34 {
		t.Fatalf("scouted marker = %+v, want exact location with generic name", marker)
	}
	got := marker.Detail.(AperturePortalMarker)
	if got.OwnerID != "" || got.ApertureID != "" || got.AccessPolicy != "" || got.ConcealmentPower != 0 {
		t.Fatalf("scouted marker exposes portal metadata: %+v", got)
	}
}
