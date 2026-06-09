package content

import (
	"encoding/json"
	"testing"
	"testing/fstest"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"menu/mission_lists.json": {Data: []byte(
			`[{"Name":"KNT_Missions","Missions":["bbbb","aaaa"]}]`)},
		"menu/player_levels.json":  {Data: []byte(`{"XpThresholds":[3950,7900]}`)},
		"missions/aaaa.json":       {Data: []byte(`{"id":"aaaa","Ui":{"Title":"A"},"Group":null}`)},
		"missions/bbbb.json":       {Data: []byte(`{"id":"bbbb","Ui":{"Title":"B"},"Group":null}`)},
		"privacypolicy_pc_en.json": {Data: []byte(`{"view":"menu3.modal.ModalDialogGeneric"}`)},
		"challenges.json": {Data: []byte(`[{` +
			`"Definition":{"Id":"chal-1","Name":"N","Description":"D"},` +
			`"KntDefinition":{"Id":"chal-1","Tags":["t"],"Image":"i","HighResImage":"h",` +
			`"Rewards":{"Xp":10,"Intel":5,"Unlockables":["u1"]},` +
			`"IsSpoiler":true,"SpoilerRequiredCheckpoint":"cp"}}]`)},
		"unlockables.json": {Data: []byte(`[{` +
			`"Title":"T","Description":"D","Image":"img","Comment":"","HighResImage":"",` +
			`"FeaturedImage":"","IsFeatured":false,"IsNew":false,"RequiredCheckpoint":"",` +
			`"IntelShop":true,` +
			`"UnlockableState":{"Id":"unl-1","State":1},` +
			`"RequiredLevel":2,"LevelOk":true,"PriceCurrency":"Intel","PriceValue":4000,` +
			`"PriceOk":true,"GrantedItems":[],"CanBuy":false,"Unlocks":["x"],` +
			`"Type":"gadget","Category":"gadget"}]`)},
	}
}

func TestLoadAndMenuContent(t *testing.T) {
	s, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.MissionCount() != 2 {
		t.Fatalf("MissionCount = %d, want 2", s.MissionCount())
	}

	mc := s.MenuContent()
	if len(mc.MissionLists) != 1 || mc.MissionLists[0].Name != "KNT_Missions" {
		t.Fatalf("unexpected mission lists: %+v", mc.MissionLists)
	}
	if len(mc.Missions) != 2 {
		t.Fatalf("got %d missions, want 2", len(mc.Missions))
	}

	// Missions are emitted in stable sorted-by-id order.
	if mc.Missions[0].ID != "aaaa" {
		t.Errorf("first mission id = %q, want aaaa (sorted)", mc.Missions[0].ID)
	}

	// MissionProgressions must serialize as null, matching the real response.
	out, err := json.Marshal(mc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var generic map[string]json.RawMessage
	if err := json.Unmarshal(out, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(generic["MissionProgressions"]) != "null" {
		t.Errorf("MissionProgressions = %s, want null", generic["MissionProgressions"])
	}
}

func TestContractLookup(t *testing.T) {
	s, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	mission, ok := s.Contract("bbbb")
	if !ok {
		t.Fatal("expected to find contract bbbb")
	}
	if mission.ID != "bbbb" {
		t.Errorf("id = %q, want bbbb", mission.ID)
	}
	if mission.Ui.Title != "B" {
		t.Errorf("title = %q, want B", mission.Ui.Title)
	}

	if _, ok := s.Contract("does-not-exist"); ok {
		t.Error("expected missing contract to return ok=false")
	}
}

func TestLoadChallenges(t *testing.T) {
	s, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.ChallengeCount() != 1 {
		t.Fatalf("ChallengeCount = %d, want 1", s.ChallengeCount())
	}

	c := s.Challenges()[0]
	if c.Definition.Id != "chal-1" || c.Definition.Name != "N" {
		t.Errorf("unexpected Definition: %+v", c.Definition)
	}
	if !c.KntDefinition.IsSpoiler || c.KntDefinition.SpoilerRequiredCheckpoint != "cp" {
		t.Errorf("unexpected KntDefinition: %+v", c.KntDefinition)
	}
	if c.KntDefinition.Rewards.Xp != 10 || c.KntDefinition.Rewards.Intel != 5 {
		t.Errorf("unexpected Rewards: %+v", c.KntDefinition.Rewards)
	}
}

func TestLoadUnlockables(t *testing.T) {
	s, err := Load(testFS())
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.UnlockableCount() != 1 {
		t.Fatalf("UnlockableCount = %d, want 1", s.UnlockableCount())
	}

	u := s.Unlockables()[0]
	if u.Title != "T" || u.Type != "gadget" || u.Category != "gadget" {
		t.Errorf("unexpected static fields: %+v", u)
	}
	if u.PriceValue != 4000 || u.RequiredLevel != 2 || !u.IntelShop {
		t.Errorf("unexpected numeric/bool fields: %+v", u)
	}
	if len(u.Unlocks) != 1 || u.Unlocks[0] != "x" {
		t.Errorf("unexpected Unlocks: %+v", u.Unlocks)
	}

	// UnlockableWithState pairs the static catalog entry with per-user state.
	ws := UnlockableWithState{
		Unlockable: u,
		UnlockableState: UnlockableStateWithState{
			UnlockableState: u.UnlockableState,
			State:           1,
		},
	}
	out, err := json.Marshal(ws)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(got["UnlockableState"]) != `{"Id":"unl-1","State":1}` {
		t.Errorf("UnlockableState = %s", got["UnlockableState"])
	}
}

func TestLoadMissingFile(t *testing.T) {
	fsys := fstest.MapFS{
		"menu/player_levels.json": {Data: []byte(`{}`)},
	}
	if _, err := Load(fsys); err == nil {
		t.Fatal("expected error when mission_lists.json is missing")
	}
}
