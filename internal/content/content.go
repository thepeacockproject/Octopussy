// Package content loads the static game data (missions, mission lists, player
// levels) from a filesystem and serves it to handlers. The data is split into
// many small files - one JSON file per mission under missions/, plus menu/
// documents - so it stays maintainable and individual contracts can be looked
// up by id.
//
// Expected layout of the provided fs.FS:
//
//	menu/mission_lists.json   the MissionLists array
//	menu/player_levels.json   the PlayerLevels object
//	missions/<id>.json        one mission/contract per file
package content

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// MissionList groups mission ids under a named menu list (e.g. "KNT_Missions").
type MissionList struct {
	Name     string   `json:"Name"`
	Missions []string `json:"Missions"`
}

// MenuContent is the aggregate document returned by GetMenuContent. The mission
// objects and PlayerLevels are kept as raw JSON so we reproduce the game's data
// verbatim without modeling every nested field.
type MenuContent struct {
	MissionLists        []MissionList     `json:"MissionLists"`
	Missions            []json.RawMessage `json:"Missions"`
	MissionProgressions json.RawMessage   `json:"MissionProgressions"`
	PlayerLevels        json.RawMessage   `json:"PlayerLevels"`
}

// Store holds the parsed game data in memory.
type Store struct {
	missionLists []MissionList
	playerLevels json.RawMessage

	// missions maps a mission id to its raw JSON; missionOrder is the stable
	// (sorted-by-id) order used when listing the mission pool.
	missions     map[string]json.RawMessage
	missionOrder []string
}

// idHolder is used to pull just the id out of a mission file.
type idHolder struct {
	ID string `json:"id"`
}

// Load reads the game data from fsys and returns a ready-to-serve Store.
func Load(fsys fs.FS) (*Store, error) {
	s := &Store{missions: make(map[string]json.RawMessage)}

	listsRaw, err := fs.ReadFile(fsys, "menu/mission_lists.json")
	if err != nil {
		return nil, fmt.Errorf("read mission lists: %w", err)
	}
	if err := json.Unmarshal(listsRaw, &s.missionLists); err != nil {
		return nil, fmt.Errorf("parse mission lists: %w", err)
	}

	levels, err := fs.ReadFile(fsys, "menu/player_levels.json")
	if err != nil {
		return nil, fmt.Errorf("read player levels: %w", err)
	}
	s.playerLevels = json.RawMessage(levels)

	entries, err := fs.ReadDir(fsys, "missions")
	if err != nil {
		return nil, fmt.Errorf("read missions dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := fs.ReadFile(fsys, path.Join("missions", e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read mission %s: %w", e.Name(), err)
		}
		var h idHolder
		if err := json.Unmarshal(raw, &h); err != nil {
			return nil, fmt.Errorf("parse mission %s: %w", e.Name(), err)
		}
		if h.ID == "" {
			return nil, fmt.Errorf("mission %s has no id", e.Name())
		}
		s.missions[h.ID] = json.RawMessage(raw)
		s.missionOrder = append(s.missionOrder, h.ID)
	}

	// Deterministic order so the menu content is stable across runs.
	sort.Strings(s.missionOrder)

	return s, nil
}

// MenuContent assembles the aggregate menu document. MissionProgressions is
// currently always null, matching the observed response.
func (s *Store) MenuContent() MenuContent {
	missions := make([]json.RawMessage, 0, len(s.missionOrder))
	for _, id := range s.missionOrder {
		missions = append(missions, s.missions[id])
	}
	return MenuContent{
		MissionLists:        s.missionLists,
		Missions:            missions,
		MissionProgressions: nil, // marshals as null
		PlayerLevels:        s.playerLevels,
	}
}

// Contract returns the raw JSON for a single mission/contract by id. This backs
// future per-contract lookups.
func (s *Store) Contract(id string) (json.RawMessage, bool) {
	raw, ok := s.missions[id]
	return raw, ok
}

// MissionCount reports how many missions were loaded, for startup logging.
func (s *Store) MissionCount() int { return len(s.missionOrder) }
