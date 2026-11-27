package handlers

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestLoadQuestsFromCSV_StoryRoadmap(t *testing.T) {
	if err := loadQuestsFromCSV(); err != nil {
		t.Fatalf("loadQuestsFromCSV failed: %v", err)
	}
	if len(questRegistry) != 34 {
		t.Fatalf("expected 34 story quest steps, got %d", len(questRegistry))
	}
	if questRegistry[0].ID != "quest_1_awaken" || questRegistry[0].Chapter != 1 || questRegistry[0].StepOrder != 1 {
		t.Fatalf("unexpected first quest: %+v", questRegistry[0])
	}
	last := questRegistry[len(questRegistry)-1]
	if last.ID != "story_7_4_pinnacle" || last.Chapter != 7 || last.StepOrder != 34 {
		t.Fatalf("unexpected last quest: %+v", last)
	}
}

func TestEvaluateQuestProgress_HybridModes(t *testing.T) {
	var player models.Player
	player.Rank = 1
	player.GuInventory = models.GuInventory{{GuID: "GU_MOONLIGHT", Rank: 1, IsVital: false, Equipped: true}}

	vitalQuest := QuestDefinition{
		ID:        "quest_2_vital_gu",
		Condition: QuestCondition{Mode: "state", Metric: "has_vital_gu", Count: 1},
	}
	if ok, _, _ := EvaluateQuestProgress(vitalQuest, &player, nil); ok {
		t.Fatalf("expected has_vital_gu to be false when only equipped=true and is_vital=false")
	}

	player.GuInventory[0].IsVital = true
	if ok, _, _ := EvaluateQuestProgress(vitalQuest, &player, nil); !ok {
		t.Fatalf("expected has_vital_gu to be true when is_vital=true")
	}

	feedQuest := QuestDefinition{
		ID:        "story_1_3_feed_gu",
		Condition: QuestCondition{Mode: "event", Metric: "feed_gu", Count: 1},
	}
	if ok, cur, tgt := EvaluateQuestProgress(feedQuest, &player, nil); ok || cur != 0 || tgt != 1 {
		t.Fatalf("expected event quest incomplete (0/1), got ok=%v cur=%d tgt=%d", ok, cur, tgt)
	}

	raw, _ := json.Marshal(storyProgressPayload{Count: 1})
	if ok, cur, _ := EvaluateQuestProgress(feedQuest, &player, raw); !ok || cur != 1 {
		t.Fatalf("expected event quest complete (1/1), got ok=%v cur=%d", ok, cur)
	}

	refineQuest := QuestDefinition{
		ID:        "quest_4_refine_gu",
		Condition: QuestCondition{Mode: "hybrid", Metric: "refine_or_rank2_gu", Count: 1},
	}
	player.GuInventory = append(player.GuInventory, models.GuInstance{GuID: "GU_TEST", Rank: 2})
	if ok, _, _ := EvaluateQuestProgress(refineQuest, &player, nil); !ok {
		t.Fatalf("expected hybrid quest complete via rank 2 Gu state")
	}
}
