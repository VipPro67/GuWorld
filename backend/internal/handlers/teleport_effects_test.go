package handlers

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestTeleportArrayArrivalEffectAppliesComposedWorldHaste(t *testing.T) {
	profile := composition.TeleportFormationProfile()
	graph := composition.Graph{
		Product: profile.Product, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		CoreGuInstanceID: "core-instance", OutputEffectKey: "core-instance:0:0",
		Nodes: []composition.Node{{
			EffectKey: "core-instance:0:0", Role: composition.NodeOutput, GuInstanceID: "core-instance",
			GuID: "GU_WIND", Rank: 6, Trigger: "ON_USE", Context: "WORLD_TRAVEL_ARRIVAL",
			Executor: "TELEPORT_ARRAY_TRAVEL_ADAPTER",
		}},
	}
	snapshot := teleportArrayEffectProfile{
		ProfileVersion: 1, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		GuRank: 6, CompositionGraph: graph, ComponentGuInstances: []string{"core-instance"},
		Effects: teleportArrayEffectEnvelope{SchemaVersion: effectcatalog.BlockSchemaVersion, Blocks: []gameplayeffects.Block{{
			ID: "ARRIVAL_01", Trigger: gameplayeffects.TriggerOnUse, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatus, StatusID: statusfx.StatusHaste, Turns: 9, Value: 12, Chance: 1}},
		}}},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	player := models.Player{}
	player.ID = uuid.New()
	player.Rank, player.SubRank = 1, 1
	message, err := applyTeleportArrayArrivalEffect(&player, raw)
	if err != nil {
		t.Fatal(err)
	}
	if message != "Tăng Tốc 9 phút" || !player.Statuses.Has(statusfx.StatusHaste) {
		t.Fatalf("arrival effect message=%q statuses=%+v", message, player.Statuses)
	}
	if got := CalculatePlayerStats(&player)["speed"]; got != 11.2 {
		t.Fatalf("arrival HASTE speed=%v, want 11.2", got)
	}
}

func TestTeleportArrayArrivalEffectRejectsUnsupportedCommand(t *testing.T) {
	profile := composition.TeleportFormationProfile()
	snapshot := teleportArrayEffectProfile{
		ProfileVersion: 1, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		GuRank: 6,
		CompositionGraph: composition.Graph{
			Product: profile.Product, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
			CoreGuInstanceID: "core-instance", OutputEffectKey: "core-instance:0:0",
			Nodes: []composition.Node{{EffectKey: "core-instance:0:0", Role: composition.NodeOutput, GuInstanceID: "core-instance", GuID: "GU_WIND", Rank: 6,
				Trigger: "ON_USE", Context: "WORLD_TRAVEL_ARRIVAL", Executor: "TELEPORT_ARRAY_TRAVEL_ADAPTER"}},
		},
		ComponentGuInstances: []string{"core-instance"},
		Effects: teleportArrayEffectEnvelope{SchemaVersion: effectcatalog.BlockSchemaVersion, Blocks: []gameplayeffects.Block{{
			ID: "ARRIVAL_01", Trigger: gameplayeffects.TriggerOnUse, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 1}},
		}}},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := applyTeleportArrayArrivalEffect(&models.Player{}, raw); err == nil {
		t.Fatal("unsupported travel operation was accepted")
	}
}

func TestTeleportArrayRejectsConditionUnavailableAtArrival(t *testing.T) {
	profile := composition.TeleportFormationProfile()
	snapshot := teleportArrayEffectProfile{
		ProfileVersion: 1, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		GuRank: 6,
		CompositionGraph: composition.Graph{
			Product: profile.Product, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
			CoreGuInstanceID: "core-instance", OutputEffectKey: "core-instance:0:0",
			Nodes: []composition.Node{{EffectKey: "core-instance:0:0", Role: composition.NodeOutput, GuInstanceID: "core-instance", GuID: "GU_WIND", Rank: 6,
				Trigger: "ON_USE", Context: "WORLD_TRAVEL_ARRIVAL", Executor: "TELEPORT_ARRAY_TRAVEL_ADAPTER"}},
		},
		ComponentGuInstances: []string{"core-instance"},
		Effects: teleportArrayEffectEnvelope{SchemaVersion: effectcatalog.BlockSchemaVersion, Blocks: []gameplayeffects.Block{{
			ID: "ARRIVAL_01", Trigger: gameplayeffects.TriggerOnUse, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
			Conditions: []gameplayeffects.Condition{{Field: "MASTERY", Operator: "GTE", Value: 2}},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHeal, SubType: "FLAT", Value: 10}},
		}}},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateTeleportArrayEffectProfile(raw); err == nil {
		t.Fatal("arrival snapshot accepted condition field unavailable to the teleport adapter")
	}
}

func TestTeleportArrayRejectsLegacyCompositionEdgeWithoutRuleRevision(t *testing.T) {
	profile := composition.TeleportFormationProfile()
	snapshot := teleportArrayEffectProfile{
		ProfileVersion: 1, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		CompositionGraph: composition.Graph{
			Product: profile.Product, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
			CoreGuInstanceID: "core-instance", OutputEffectKey: "output",
			Nodes: []composition.Node{{EffectKey: "output", Role: composition.NodeOutput, GuInstanceID: "core-instance", GuID: "GU_WIND", Rank: 6, Trigger: "ON_USE", Context: "WORLD_TRAVEL_ARRIVAL", Executor: "TELEPORT_ARRAY_TRAVEL_ADAPTER"}},
			Edges: []composition.Edge{{From: "support", To: "output", Kind: composition.ContributionBridge}},
		},
		ComponentGuInstances: []string{"core-instance"},
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := validateTeleportArrayEffectProfile(raw); err == nil {
		t.Fatal("unversioned legacy composition edge was accepted")
	}
}
