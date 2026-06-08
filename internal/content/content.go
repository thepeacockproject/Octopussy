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

// MenuContent is the aggregate document returned by GetMenuContent. PlayerLevels
// is kept as raw JSON so we reproduce that document verbatim without modeling its
// every nested field; missions are fully parsed into the Mission type.
type MenuContent struct {
	MissionLists        []MissionList   `json:"MissionLists"`
	Missions            []Mission       `json:"Missions"`
	MissionProgressions json.RawMessage `json:"MissionProgressions"`
	PlayerLevels        json.RawMessage `json:"PlayerLevels"`
}

// Mission is the typed description of a single mission/contract (one
// missions/<id>.json file). The store also keeps each mission as raw JSON for
// verbatim serving; this type is for code that needs to read mission fields.
type Mission struct {
	ID       string          `json:"id"`
	Gameplay MissionGameplay `json:"Gameplay"`
	Ui       MissionUI       `json:"Ui"`
	Ugc      MissionUgc      `json:"Ugc"`
	Group    *MissionGroup   `json:"Group"`
}

// MissionGameplay holds a mission's gameplay rules and completion rewards.
type MissionGameplay struct {
	Checkpoint         string   `json:"Checkpoint"`
	Tags               []string `json:"Tags"`
	Type               int      `json:"Type"`
	Rewards            Rewards  `json:"Rewards"`
	RequirementTags    string   `json:"RequirementTags"`
	RequiredCheckpoint string   `json:"RequiredCheckpoint"`
	Entitlements       []string `json:"Entitlements"`
}

// MissionUI holds a mission's presentation strings and imagery.
type MissionUI struct {
	Title            string   `json:"Title"`
	Image            string   `json:"Image"`
	HighResImage     string   `json:"HighResImage"`
	Description      string   `json:"Description"`
	Location         string   `json:"Location"`
	Goal             string   `json:"Goal"`
	Objectives       []string `json:"Objectives"`
	Requirements     []string `json:"Requirements"`
	ShortDescription string   `json:"ShortDescription"`
}

// MissionUgc holds user-generated-content objectives; empty for built-in missions.
type MissionUgc struct {
	Objectives []string `json:"Objectives"`
}

// MissionGroup links a mission into an escalation tree. Parent is null for a
// top-level mission; Children is null for a leaf. Both Mission.Group and Parent
// are pointers so a null/absent group round-trips faithfully.
type MissionGroup struct {
	Parent   *string  `json:"Parent"`
	Children []string `json:"Children"`
}

// Challenge is one entry in the challenge catalog. Definition and KntDefinition
// are static game data (loaded from challenges.json); State and KntState are
// user-specific runtime data that is absent from the static file and filled in
// per request by the handler. A zero-valued State/KntState is therefore the
// expected shape of a freshly loaded Challenge.
type Challenge struct {
	Definition    Definition    `json:"Definition"`
	KntDefinition KntDefinition `json:"KntDefinition"`
}

// TODO: Move this to another file?
type ChallengeWithState struct {
	Challenge
	State    State    `json:"State"`
	KntState KntState `json:"KntState"`
}

// Definition is the static, user-agnostic description of a challenge.
type Definition struct {
	Id          string `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

// State is the per-user progress on a challenge. It is not part of the static
// data and is populated for the requesting player.
type State struct {
	Id              string `json:"Id"`
	CompletionState int    `json:"CompletionState"`
	ProgressCurrent int    `json:"ProgressCurrent"`
	ProgressMax     int    `json:"ProgressMax"`
}

// KntDefinition is the KNT-specific static metadata for a challenge: tags,
// imagery, and the rewards granted on completion.
type KntDefinition struct {
	Id                        string   `json:"Id"`
	Tags                      []string `json:"Tags"`
	Image                     string   `json:"Image"`
	HighResImage              string   `json:"HighResImage"`
	Rewards                   Rewards  `json:"Rewards"`
	IsSpoiler                 bool     `json:"IsSpoiler"`
	SpoilerRequiredCheckpoint string   `json:"SpoilerRequiredCheckpoint"`
}

// Rewards are the XP, Intel currency, and unlockables granted on completion,
// shared by both challenges and missions.
type Rewards struct {
	Xp          int64    `json:"Xp"`
	Intel       int      `json:"Intel"`
	Unlockables []string `json:"Unlockables"`
}

// KntState is the per-user KNT challenge state. Like State, it is not part of
// the static data and is populated for the requesting player.
type KntState struct {
	NewlyCompleted bool `json:"NewlyCompleted"`
}

// Unlockable is one entry in the unlockable catalog (gadgets, gear, etc.). It
// holds only the static, user-agnostic fields; the per-user UnlockableState is
// absent here and carried by UnlockablesWithState.
type Unlockable struct {
	Title              string   `json:"Title"`
	Description        string   `json:"Description"`
	Image              string   `json:"Image"`
	Comment            string   `json:"Comment"`
	HighResImage       string   `json:"HighResImage"`
	FeaturedImage      string   `json:"FeaturedImage"`
	IsFeatured         bool     `json:"IsFeatured"`
	IsNew              bool     `json:"IsNew"`
	RequiredCheckpoint string   `json:"RequiredCheckpoint"`
	IntelShop          bool     `json:"IntelShop"`
	RequiredLevel      int      `json:"RequiredLevel"`
	LevelOk            bool     `json:"LevelOk"`
	PriceCurrency      string   `json:"PriceCurrency"`
	PriceValue         int      `json:"PriceValue"`
	PriceOk            bool     `json:"PriceOk"`
	GrantedItems       []string `json:"GrantedItems"`
	CanBuy             bool     `json:"CanBuy"`
	Unlocks            []string `json:"Unlocks"`
	Type               string   `json:"Type"`
	Category           string   `json:"Category"`
}

// UnlockableState is the per-user state of an unlockable: its id and a numeric
// ownership/unlock state. Like the challenge State, it is user-specific and is
// populated for the requesting player rather than coming from the static file.
// TODO: Move this to another file?
type UnlockableState struct {
	Id    string `json:"Id"`
	State int    `json:"State"`
}

// UnlockablesWithState is an Unlockable paired with its per-user state, the
// shape returned to the client.
type UnlockablesWithState struct {
	Unlockable
	UnlockableState UnlockableState `json:"UnlockableState"`
}

// Store holds the parsed game data in memory.
type Store struct {
	missionLists []MissionList
	playerLevels json.RawMessage

	// xpThresholds is the parsed XpThresholds array from player_levels.json: the
	// cumulative XP needed to reach each successive level (ascending). It backs
	// level calculation for player progression.
	xpThresholds []int64

	// privacyPolicy is the verbatim KNT privacy-policy dialog document, served
	// under /files/privacypolicy/. The game requires it before going online.
	privacyPolicy json.RawMessage

	// missions maps a mission id to its parsed Mission; missionOrder is the
	// stable (sorted-by-id) order used when listing the mission pool.
	missions     map[string]Mission
	missionOrder []string

	// challenges is the static challenge catalog in file order. Each entry's
	// State and KntState are zero-valued (those are user-specific and absent
	// from challenges.json); handlers fill them per requesting player.
	challenges []Challenge

	unlockables []Unlockable
}

// Load reads the game data from fsys and returns a ready-to-serve Store.
func Load(fsys fs.FS) (*Store, error) {
	s := &Store{missions: make(map[string]Mission)}

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
	s.playerLevels = levels

	var pl struct {
		XpThresholds []int64 `json:"XpThresholds"`
	}

	if err := json.Unmarshal(levels, &pl); err != nil {
		return nil, fmt.Errorf("parse player levels: %w", err)
	}

	s.xpThresholds = pl.XpThresholds

	policy, err := fs.ReadFile(fsys, "privacypolicy_pc_en.json")
	if err != nil {
		return nil, fmt.Errorf("read privacy policy: %w", err)
	}
	s.privacyPolicy = policy

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

		if m.ID == "" {
			return fmt.Errorf("mission %s has no id", e.Name())
		}

		s.missions[m.ID] = m
		s.missionOrder = append(s.missionOrder, m.ID)
	}

	// Deterministic order so the menu content is stable across runs.
	sort.Strings(s.missionOrder)

	return nil
}

// loadChallenges reads the static challenge catalog from challenges.json. Only
// the Definition and KntDefinition of each entry are present in the file; the
// per-user State and KntState are left zero-valued for handlers to fill in.
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
		MissionProgressions: nil, // marshals as null
		PlayerLevels:        s.playerLevels,
	}
}

// LevelForXP returns the player level for a given total XP. Level 1 is the base;
// each XpThreshold the XP has reached grants one more level. Thresholds are
// ascending, so the scan stops at the first one not yet reached.
func (s *Store) LevelForXP(xp int64) int {
	level := 1
	for _, threshold := range s.xpThresholds {
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

// Contract returns the parsed mission/contract for a single id. This backs
// per-contract lookups.
func (s *Store) Contract(id string) (Mission, bool) {
	mission, ok := s.missions[id]
	return mission, ok
}

// MissionCount reports how many missions were loaded, for startup logging.
func (s *Store) MissionCount() int { return len(s.missionOrder) }

// Challenges returns the static challenge catalog in file order. The State and
// KntState on each returned Challenge are zero-valued; callers copy the entries
// and fill those fields in for the requesting player.
func (s *Store) Challenges() []Challenge { return s.challenges }

// ChallengeCount reports how many challenges were loaded, for startup logging.
func (s *Store) ChallengeCount() int { return len(s.challenges) }

// Unlockable returns the static unlockable catalog in file order. Callers pair
// each entry with a per-user UnlockableState (see UnlockablesWithState).
func (s *Store) Unlockables() []Unlockable { return s.unlockables }

// UnlockablesCount reports how many unlockables were loaded, for startup logging.
func (s *Store) UnlockablesCount() int { return len(s.unlockables) }
