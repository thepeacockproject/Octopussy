package content

// MenuContent is the aggregate document returned by GetMenuContent. PlayerLevels
// is kept as raw JSON so we reproduce that document verbatim without modeling its
// every nested field; missions are fully parsed into the Mission type.
type MenuContent struct {
	MissionLists        []MissionList        `json:"MissionLists"`
	Missions            []Mission            `json:"Missions"`
	MissionProgressions []MissionProgression `json:"MissionProgressions"`
	PlayerLevels        PlayerLevels         `json:"PlayerLevels"`
}

// MissionList groups mission ids under a named menu list (e.g. "KNT_Missions").
type MissionList struct {
	Name     string   `json:"Name"`
	Missions []string `json:"Missions"`
}

// Mission is the typed description of a single mission/contract (one
// missions/<id>.json file). The store also keeps each mission as raw JSON for
// verbatim serving; this type is for code that needs to read mission fields.
type Mission struct {
	Id       string          `json:"id"`
	Gameplay MissionGameplay `json:"Gameplay"`
	UI       MissionUI       `json:"Ui"`
	UGC      MissionUGC      `json:"Ugc"`
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

// MissionUGC holds user-generated-content objectives; empty for built-in missions.
type MissionUGC struct {
	Objectives []string `json:"Objectives"`
}

// MissionGroup links a mission into an escalation tree. Parent is null for a
// top-level mission; Children is null for a leaf. Both Mission.Group and Parent
// are pointers so a null/absent group round-trips faithfully.
type MissionGroup struct {
	Parent   *string  `json:"Parent"`
	Children []string `json:"Children"`
}

type MissionProgression struct {
	Id        string                   `json:"id"`
	ProfileId string                   `json:"ProfileId"`
	MissionId string                   `json:"MissionId"`
	Common    MissionProgressionCommon `json:"Common"`
	Group     *MissionProgressionGroup `json:"Group"`
}

type MissionProgressionCommon struct {
	Completed      bool   `json:"Completed"`
	CompletionTime string `json:"CompletionTime"`
}

type MissionProgressionGroup struct {
	NextMissionId string `json:"NextMissionId"`
}

type PlayerLevels struct {
	XpThresholds []int64 `json:"XpThresholds"`
}

// Challenge is one entry in the challenge catalog. ChallengeDefinition and ChallengeKntDefinition
// are static game data (loaded from challenges.json); ChallengeState and ChallengeKntState are
// user-specific runtime data that is absent from the static file and filled in
// per request by the handler. A zero-valued ChallengeState/KntState is therefore the
// expected shape of a freshly loaded Challenge.
type Challenge struct {
	Definition    ChallengeDefinition    `json:"Definition"`
	State         ChallengeState         `json:"State"`
	KntDefinition ChallengeKntDefinition `json:"KntDefinition"`
}

// ChallengeDefinition is the static, user-agnostic description of a challenge.
type ChallengeDefinition struct {
	Id          string `json:"Id"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
}

// ChallengeState is the per-user progress on a challenge. It is not part of the static
// data and is populated for the requesting player.
type ChallengeState struct {
	Id          string `json:"Id"`
	ProgressMax int    `json:"ProgressMax"`
}

// ChallengeKntDefinition is the KNT-specific static metadata for a challenge: tags,
// imagery, and the rewards granted on completion.
type ChallengeKntDefinition struct {
	Id                        string   `json:"Id"`
	Tags                      []string `json:"Tags"`
	Image                     string   `json:"Image"`
	HighResImage              string   `json:"HighResImage"`
	Rewards                   Rewards  `json:"Rewards"`
	IsSpoiler                 bool     `json:"IsSpoiler"`
	SpoilerRequiredCheckpoint string   `json:"SpoilerRequiredCheckpoint"`
}

type ChallengeWithState struct {
	Challenge
	State    ChallengeStateWithState `json:"State"`
	KntState ChallengeKntState       `json:"KntState"`
}

type ChallengeStateWithState struct {
	ChallengeState
	CompletionState CompletionState `json:"CompletionState"`
	ProgressCurrent int             `json:"ProgressCurrent"`
}

type ChallengeKntState struct {
	NewlyCompleted bool `json:"NewlyCompleted"`
}

// Unlockable is one entry in the unlockable catalog (gadgets, gear, etc.). It
// holds only the static, user-agnostic fields; the per-user UnlockableState is
// absent here and carried by UnlockablesWithState.
type Unlockable struct {
	Title              string          `json:"Title"`
	Description        string          `json:"Description"`
	Image              string          `json:"Image"`
	Comment            string          `json:"Comment"`
	HighResImage       string          `json:"HighResImage"`
	FeaturedImage      string          `json:"FeaturedImage"`
	IsFeatured         bool            `json:"IsFeatured"`
	IsNew              bool            `json:"IsNew"`
	RequiredCheckpoint string          `json:"RequiredCheckpoint"`
	IntelShop          bool            `json:"IntelShop"`
	UnlockableState    UnlockableState `json:"UnlockableState"`
	RequiredLevel      int             `json:"RequiredLevel"`
	PriceCurrency      string          `json:"PriceCurrency"`
	PriceValue         int             `json:"PriceValue"`
	GrantedItems       []string        `json:"GrantedItems"`
	Unlocks            []string        `json:"Unlocks"`
	Type               string          `json:"Type"`
	Category           string          `json:"Category"`
}

type UnlockableState struct {
	Id string `json:"Id"`
}

type UnlockableWithState struct {
	Unlockable
	LevelOk         bool                     `json:"LevelOk"`
	PriceOk         bool                     `json:"PriceOk"`
	CanBuy          bool                     `json:"CanBuy"`
	UnlockableState UnlockableStateWithState `json:"UnlockableState"`
}

type UnlockableStateWithState struct {
	UnlockableState
	State UnlockedState `json:"State"`
}

// Rewards are the XP, Intel currency, and unlockables granted on completion,
// shared by both challenges and missions.
type Rewards struct {
	Xp          int64    `json:"Xp"`
	Intel       int      `json:"Intel"`
	Unlockables []string `json:"Unlockables"`
}
