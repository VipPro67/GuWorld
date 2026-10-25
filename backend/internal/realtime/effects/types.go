// Package effects audits Gu effect blocks against the realtime zone runtime.
// It is deliberately read-only and produces a deterministic report for PR-17:
// every authored block receives an explicit migration status, and unsupported
// mechanics are never silently ignored.
package effects

import "encoding/json"

const SchemaVersion = 1

type Status string

const (
	StatusSupported     Status = "SUPPORTED"
	StatusNeedsReauthor Status = "NEEDS_REAUTHOR"
	StatusUnsupported   Status = "UNSUPPORTED"
	StatusNotActive     Status = "NOT_ACTIVE"
)

func (s Status) Rank() int {
	switch s {
	case StatusUnsupported:
		return 3
	case StatusNeedsReauthor:
		return 2
	case StatusSupported:
		return 1
	default:
		return 0
	}
}

// Worst returns the more severe status. NOT_ACTIVE is the least severe so a
// world-only block does not downgrade an entry that also has a combat block.
func Worst(left, right Status) Status {
	if right.Rank() > left.Rank() {
		return right
	}
	return left
}

func (s Status) Valid() bool {
	switch s {
	case StatusSupported, StatusNeedsReauthor, StatusUnsupported, StatusNotActive:
		return true
	default:
		return false
	}
}

type EntryRef struct {
	ID   string `json:"id"`
	Rank int    `json:"rank"`
}

// GuEntry is the subset of Gu.csv needed by the audit.
type GuEntry struct {
	ID          string          `json:"id"`
	Name        string          `json:"name,omitempty"`
	Rank        int             `json:"rank"`
	Type        string          `json:"gu_type,omitempty"`
	PrimaryRole string          `json:"primary_role,omitempty"`
	ManaCost    float64         `json:"base_mana_cost,omitempty"`
	Effects     json.RawMessage `json:"effects"`
}

type OperationAudit struct {
	Index    int      `json:"index"`
	Op       string   `json:"op"`
	SubType  string   `json:"sub_type,omitempty"`
	Status   Status   `json:"status"`
	Reasons  []string `json:"reasons,omitempty"`
	Proposal string   `json:"proposal,omitempty"`
}

type BlockAudit struct {
	ID         string           `json:"id"`
	Trigger    string           `json:"trigger"`
	Target     string           `json:"target"`
	Status     Status           `json:"status"`
	Reasons    []string         `json:"reasons,omitempty"`
	Proposal   string           `json:"proposal,omitempty"`
	Operations []OperationAudit `json:"operations"`
}

type EntryAudit struct {
	ID          string       `json:"id"`
	Rank        int          `json:"rank"`
	Type        string       `json:"gu_type,omitempty"`
	PrimaryRole string       `json:"primary_role,omitempty"`
	Status      Status       `json:"status"`
	Reasons     []string     `json:"reasons,omitempty"`
	MVP         bool         `json:"mvp,omitempty"`
	Blocks      []BlockAudit `json:"blocks"`
}

type Count struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type Summary struct {
	Entries           int     `json:"entries"`
	Supported         int     `json:"supported"`
	NeedsReauthor     int     `json:"needs_reauth"`
	Unsupported       int     `json:"unsupported"`
	NotActive         int     `json:"not_active"`
	Blocks            int     `json:"blocks"`
	BlocksSupported   int     `json:"blocks_supported"`
	BlocksNeedsReauth int     `json:"blocks_needs_reauth"`
	BlocksUnsupported int     `json:"blocks_unsupported"`
	BlocksNotActive   int     `json:"blocks_not_active"`
	Operations        int     `json:"operations"`
	Unclassified      int     `json:"unclassified"`
	Triggers          []Count `json:"triggers"`
	OperationKinds    []Count `json:"operation_kinds"`
}

type Report struct {
	SchemaVersion int          `json:"schema_version"`
	Summary       Summary      `json:"summary"`
	MVP           []EntryRef   `json:"mvp"`
	MissingMVP    []EntryRef   `json:"missing_mvp,omitempty"`
	Warnings      []string     `json:"warnings,omitempty"`
	Entries       []EntryAudit `json:"entries"`
}

// DefaultMVP is the first playable slice. It covers damage, area damage,
// defense, control, healing and passive stats using low-rank Gu that already
// exist in Gu.csv. Hidden passives are not authored in the current data; PR-19
// must add one explicitly rather than pretending a block exists.
func DefaultMVP() []EntryRef {
	return []EntryRef{
		{ID: "GU_WATER_ARROW", Rank: 1},
		{ID: "GU_SPIRAL_HAMMER_2", Rank: 2},
		{ID: "GU_WATER_SHIELD", Rank: 2},
		{ID: "GU_WHITE_FROST", Rank: 3},
		{ID: "GU_WATER_CLEANSING", Rank: 4},
		{ID: "GU_STRENGTH", Rank: 1},
	}
}
