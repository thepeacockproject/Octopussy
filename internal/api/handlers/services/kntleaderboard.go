package services

import (
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
	shared2 "github.com/thepeacockproject/Octopussy/internal/shared"
)

type getLeaderboardsRequest struct {
	MissionId string `json:"missionId"`
	ProfileId string `json:"profileId"`
	Page      int    `json:"page"`
	PageSize  int    `json:"pageSize"`
}

type getLeaderboardsResponse struct {
	Entries          *[]leaderboardEntry `json:"Entries"`
	CurrentPlayer    leaderboardEntry    `json:"CurrentPlayer"`
	HasMore          bool                `json:"HasMore"`
	PercentileSpread []float64           `json:"PercentileSpread"`
}

type leaderboardEntry struct {
	Rank            int          `json:"Rank"`
	Score           int          `json:"Score"`
	ProfileId       string       `json:"ProfileId"`
	PercentileIndex int          `json:"PercentileIndex"`
	ScoreDetails    scoreDetails `json:"ScoreDetails"`
}

func (c *Handler) handleGetLeaderboards(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[getLeaderboardsRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: Store (multiple?) scores for a personal best leaderboard OR centrale service
	c.WriteJSON(w, http.StatusOK, getLeaderboardsResponse{
		Entries: shared2.ShortHandIf(body.PageSize == 0, nil, &[]leaderboardEntry{}),
		CurrentPlayer: leaderboardEntry{
			Rank:            1,
			Score:           9001,
			ProfileId:       userId,
			PercentileIndex: 1,
			ScoreDetails: scoreDetails{
				Time:      8000,
				Precision: 1000,
				Flair:     1,
			},
		},
		HasMore: false,
		PercentileSpread: []float64{
			0.1,
		},
	})
}

func (c *Handler) handleGetFriendLeaderboards(w http.ResponseWriter, r *http.Request) {
	// TODO: Actually implement this
	c.handleGetLeaderboards(w, r)
}
