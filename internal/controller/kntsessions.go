package controller

import (
	"encoding/json"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/content"
)

type stopSessionPayload struct {
	GameTime     float64                 `json:"gameTime"`
	Score        int                     `json:"score"`
	SessionID    string                  `json:"sessionId"`
	ScoreDetails stopSessionScoreDetails `json:"scoreDetails"`
}

type stopSessionResponse struct {
	SessionID              string                 `json:"SessionId"`
	Rewards                stopSessionRewards     `json:"Rewards"`
	AnyChallengesCompleted bool                   `json:"AnyChallengesCompleted"`
	TotalRewards           stopSessionRewards     `json:"TotalRewards"`
	CompletionTime         float64                `json:"CompletionTime"`
	SessionProgression     stopSessionProgression `json:"SessionProgression"`
	StopReason             int                    `json:"StopReason"`
	MissionType            interface{}            `json:"MissionType"`
	MissionTier            int                    `json:"MissionTier"`
	MissionProgressions    []interface{}          `json:"MissionProgressions"`
	Score                  int                    `json:"Score"`
	ScoreDetails           interface{}            `json:"ScoreDetails"`
	ScoreDetailsDiff       interface{}            `json:"ScoreDetailsDiff"`
	PreviousScore          int                    `json:"PreviousScore"`
}

type stopSessionScoreDetails struct {
	Time      int `json:"Time"`
	Precision int `json:"Precision"`
	Flair     int `json:"Flair"`
}

type stopSessionRewards struct {
	Xp          int      `json:"Xp"`
	Intel       int      `json:"Intel"`
	Unlockables []string `json:"Unlockables"` // TODO: THIS TYPE IS ALMOST CERTAINLY WRONG!!
}

type stopSessionProgressionItem struct {
	Xp    int `json:"Xp"`
	Level int `json:"Level"`
	Intel int `json:"Intel"`
}

type stopSessionProgression struct {
	Old stopSessionProgressionItem `json:"Old"`
	New stopSessionProgressionItem `json:"New"`
}

type getForPlayPayload struct {
	MissionId string `json:"missionId"`
}

type getForPlayStatic struct {
	Mission content.Mission
}

type getForPlayDynamic struct {
	NewSessionId  string                `json:"NewSessionId"`
	SessionResult interface{}           `json:"SessionResult"`
	Progression   getForPlayProgression `json:"Progression"`
}

type getForPlayProgression struct {
	Group interface{} `json:"Group"`
	Hub   struct {
		Previous intelAndRank `json:"Previous"`
		Current  intelAndRank `json:"Current"`
	} `json:"Hub"`
}

type getForPlayResponse struct {
	Static  getForPlayStatic  `json:"Static"`
	Dynamic getForPlayDynamic `json:"Dynamic"`
}

type getForPlayProgressionWithFuture struct {
	Previous intelAndRank `json:"Previous"`
	Current  intelAndRank `json:"Current"`
}

func (c *Controller) handleGetForPlay(w http.ResponseWriter, r *http.Request) {
	var params getForPlayPayload
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	mission, contractFound := c.Content.Contract(params.MissionId)
	if !contractFound {
		c.writeJSON(w, http.StatusNotFound, map[string]string{"error": "could not find mission" + params.MissionId})
		return
	}

	user, err := c.Store.User(r.Context(), profileIDFor(r))
	if err != nil {
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ctx := r.Context()
	session, err := c.Store.CreateSession(ctx, *user, params.MissionId)

	if err != nil {
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ir := intelAndRank{Intel: user.IntelCurrency, Rank: user.Rank}

	c.writeJSON(w, http.StatusOK, getForPlayResponse{
		Static: getForPlayStatic{
			Mission: mission,
		},
		Dynamic: getForPlayDynamic{
			NewSessionId:  session.SessionID,
			SessionResult: nil, // TODO?
			Progression: getForPlayProgression{
				Group: nil,
				Hub:   getForPlayProgressionWithFuture{Current: ir, Previous: ir},
			},
		},
	})
}

func (c *Controller) handleStopSession(w http.ResponseWriter, r *http.Request) {
	var params stopSessionPayload
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	c.writeJSON(w, http.StatusOK, stopSessionResponse{
		SessionID: params.SessionID,
		// Rewards
		AnyChallengesCompleted: false,
		// TotalRewards?
		CompletionTime: params.GameTime,
		SessionProgression: stopSessionProgression{
			Old: stopSessionProgressionItem{}, // TODO
			New: stopSessionProgressionItem{}, // TODO
		},
		StopReason:          0,
		MissionType:         nil,
		MissionTier:         0,
		MissionProgressions: nil,
		Score:               params.Score,
		ScoreDetails:        nil,
		ScoreDetailsDiff:    nil,
		PreviousScore:       0,
	})
}
