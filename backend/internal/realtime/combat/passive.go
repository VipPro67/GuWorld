package combat

// Status stack policies shared by the compiler and zone runtime.
const (
	StackReplace = "REPLACE"
	StackRefresh = "REFRESH"
	StackStack   = "STACK"
	StackMax     = "MAX"
	StackIgnore  = "IGNORE"
)

func ValidStackPolicy(policy string) bool {
	switch policy {
	case StackReplace, StackRefresh, StackStack, StackMax, StackIgnore:
		return true
	default:
		return false
	}
}

// HiddenPassive is a server-only conditional proc. Discovery is tracked
// separately from proc state and never exposed to other clients.
type HiddenPassive struct {
	ID            string
	HPPercentLTE  float64
	RequiredHits  int
	ICDTicks      uint64
	Shield        float64
	Heal          float64
	NextReadyTick uint64
	HitCount      int
	Discovered    bool
	TriggeredOnce bool
}

// PreviewHiddenPassive is a temporary preview rule until the Gu loadout
// adapter can attach authored hidden passives per owned Gu. It matches the
// classic contract: HP low + N incoming hits, internal cooldown, server-only.
func PreviewHiddenPassive() HiddenPassive {
	return HiddenPassive{
		ID: "PREVIEW_LAST_STAND", HPPercentLTE: 0.4, RequiredHits: 3,
		ICDTicks: 20 * 10, Shield: 25,
	}
}
