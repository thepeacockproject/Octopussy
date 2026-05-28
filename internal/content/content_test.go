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
		"menu/player_levels.json": {Data: []byte(`{"XpThresholds":[3950,7900]}`)},
		"missions/aaaa.json":      {Data: []byte(`{"id":"aaaa","Ui":{"Title":"A"},"Group":null}`)},
		"missions/bbbb.json":      {Data: []byte(`{"id":"bbbb","Ui":{"Title":"B"},"Group":null}`)},
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
	var first idHolder
	if err := json.Unmarshal(mc.Missions[0], &first); err != nil {
		t.Fatalf("decode mission: %v", err)
	}
	if first.ID != "aaaa" {
		t.Errorf("first mission id = %q, want aaaa (sorted)", first.ID)
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

	raw, ok := s.Contract("bbbb")
	if !ok {
		t.Fatal("expected to find contract bbbb")
	}
	var h idHolder
	if err := json.Unmarshal(raw, &h); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if h.ID != "bbbb" {
		t.Errorf("id = %q, want bbbb", h.ID)
	}

	if _, ok := s.Contract("does-not-exist"); ok {
		t.Error("expected missing contract to return ok=false")
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
