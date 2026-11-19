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

	"github.com/samber/lo"
)

// Store holds the parsed game data in memory.
type Store struct {
	// privacyPolicy is the verbatim KNT privacy-policy dialog document, served
	// under /files/privacypolicy/. The game requires it before going online.
	privacyPolicy json.RawMessage

	missionLists []MissionList
	playerLevels PlayerLevels

	// missions maps a mission id to its parsed Mission; missionOrder is the
	// stable (sorted-by-id) order used when listing the mission pool.
	missions     map[string]Mission
	missionOrder []string

	// challenges is the static challenge catalog in file order. Each entry's
	// ChallengeState and ChallengeKntState are zero-valued (those are user-specific and absent
	// from challenges.json); handlers fill them per requesting player.
	challenges []Challenge

	unlockables []Unlockable
}

// Load reads the game data from fsys and returns a ready-to-serve Store.
func Load(fsys fs.FS) (*Store, error) {
	s := &Store{missions: make(map[string]Mission)}

	if err := s.loadPrivacyPolicy(fsys); err != nil {
		return nil, err
	}

	if err := s.loadMissionLists(fsys); err != nil {
		return nil, err
	}

	if err := s.loadPlayerLevels(fsys); err != nil {
		return nil, err
	}

	if err := s.loadMissions(fsys); err != nil {
		return nil, err
	}

	if err := s.loadChallenges(fsys); err != nil {
		return nil, err
	}

	if err := s.loadUnlockables(fsys); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) loadPrivacyPolicy(fsys fs.FS) error {
	raw, err := fs.ReadFile(fsys, "privacypolicy_pc_en.json")
	if err != nil {
		return fmt.Errorf("read privacy policy: %w", err)
	}

	s.privacyPolicy = raw

	return nil
}

func (s *Store) loadMissionLists(fsys fs.FS) error {
	raw, err := fs.ReadFile(fsys, "menu/mission_lists.json")
	if err != nil {
		return fmt.Errorf("read mission lists: %w", err)
	}

	if err := json.Unmarshal(raw, &s.missionLists); err != nil {
		return fmt.Errorf("parse mission lists: %w", err)
	}

	return nil
}

func (s *Store) loadPlayerLevels(fsys fs.FS) error {
	raw, err := fs.ReadFile(fsys, "menu/player_levels.json")
	if err != nil {
		return fmt.Errorf("read player levels: %w", err)
	}

	if err := json.Unmarshal(raw, &s.playerLevels); err != nil {
		return fmt.Errorf("parse player levels: %w", err)
	}

	return nil
}

// loadMissions reads every missions/<id>.json file into the store, keyed by the
// id declared inside each file, and records a deterministic (sorted-by-id) order
// so the mission pool is stable across runs.
func (s *Store) loadMissions(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, "missions")
	if err != nil {
		return fmt.Errorf("read missions dir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		raw, err := fs.ReadFile(fsys, path.Join("missions", e.Name()))
		if err != nil {
			return fmt.Errorf("read mission %s: %w", e.Name(), err)
		}

		var m Mission
		if err := json.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("parse mission %s: %w", e.Name(), err)
		}

		if m.Id == "" {
			return fmt.Errorf("mission %s has no id", e.Name())
		}

		s.missions[m.Id] = m
		s.missionOrder = append(s.missionOrder, m.Id)
	}

	// Deterministic order so the menu content is stable across runs.
	sort.Strings(s.missionOrder)

	return nil
}

// loadChallenges reads the static challenge catalog from challenges.json. Only
// the ChallengeDefinition and ChallengeKntDefinition of each entry are present in the file; the
// per-user ChallengeState and ChallengeKntState are left zero-valued for handlers to fill in.
func (s *Store) loadChallenges(fsys fs.FS) error {
	raw, err := fs.ReadFile(fsys, "challenges.json")
	if err != nil {
		return fmt.Errorf("read challenges: %w", err)
	}

	if err := json.Unmarshal(raw, &s.challenges); err != nil {
		return fmt.Errorf("parse challenges: %w", err)
	}

	return nil
}

// loadUnlockables reads the static unlockable catalog from unlockables.json.
// Only the user-agnostic fields are kept; any per-user UnlockableState in the
// file is ignored, to be filled in per requesting player (see UnlockablesWithState).
func (s *Store) loadUnlockables(fsys fs.FS) error {
	raw, err := fs.ReadFile(fsys, "unlockables.json")
	if err != nil {
		return fmt.Errorf("read unlockables: %w", err)
	}

	if err := json.Unmarshal(raw, &s.unlockables); err != nil {
		return fmt.Errorf("parse unlockables: %w", err)
	}

	return nil
}

// MenuContent assembles the aggregate menu document. MissionProgressions is
// currently always null, matching the observed response.
func (s *Store) MenuContent() MenuContent {
	missions := make([]Mission, 0, len(s.missionOrder))

	for _, id := range s.missionOrder {
		missions = append(missions, s.missions[id])
	}

	return MenuContent{
		MissionLists:        s.missionLists,
		Missions:            missions,
		MissionProgressions: []MissionProgression{},
		PlayerLevels:        s.playerLevels,
	}
}

// LevelForXP returns the player level for a given total XP. Level 1 is the base;
// each XpThreshold the XP has reached grants one more level. Thresholds are
// ascending, so the scan stops at the first one not yet reached.
func (s *Store) LevelForXP(xp int64) int {
	level := 1
	for _, threshold := range s.playerLevels.XpThresholds {
		if xp < threshold {
			break
		}
		level++
	}
	return level
}

// PrivacyPolicy returns the raw KNT privacy-policy document, served verbatim
// under /files/privacypolicy/. It is always present once Load succeeds.
func (s *Store) PrivacyPolicy() json.RawMessage { return s.privacyPolicy }

// Mission returns the parsed mission for a single id.
func (s *Store) Mission(id string) (Mission, bool) {
	mission, ok := s.missions[id]
	return mission, ok
}

// MissionCount reports how many missions were loaded, for startup logging.
func (s *Store) MissionCount() int { return len(s.missionOrder) }

// TODO: Create lookup map?
func (s *Store) Challenge(challengeId string) (Challenge, bool) {
	return lo.Find(s.challenges, func(item Challenge) bool {
		return item.Definition.Id == challengeId
	})
}

// Challenges returns the static challenge catalog in file order. The ChallengeState and
// ChallengeKntState on each returned Challenge are zero-valued; callers copy the entries
// and fill those fields in for the requesting player.
func (s *Store) Challenges() []Challenge { return s.challenges }

// ChallengeCount reports how many challenges were loaded, for startup logging.
func (s *Store) ChallengeCount() int { return len(s.challenges) }

// TODO: Create lookup map?
func (s *Store) Unlockable(unlockableId string) (Unlockable, bool) {
	return lo.Find(s.unlockables, func(item Unlockable) bool {
		return item.UnlockableState.Id == unlockableId
	})
}

// Unlockables returns the static unlockable catalog in file order. Callers pair
// each entry with a per-user UnlockableState (see UnlockablesWithState).
func (s *Store) Unlockables() []Unlockable { return s.unlockables }

// UnlockableCount reports how many unlockables were loaded, for startup logging.
func (s *Store) UnlockableCount() int { return len(s.unlockables) }
