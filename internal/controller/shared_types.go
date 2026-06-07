package controller

type intelAndRank struct {
	Intel int `json:"Intel"`
	Rank  int `json:"Rank"`
}

// playerProgression carries a player's total XP and the level it maps to.
type playerProgression struct {
	Xp    int64 `json:"Xp"`
	Level int   `json:"Level"`
}
