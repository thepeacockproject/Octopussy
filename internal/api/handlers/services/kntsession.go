package services

import (
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
	"github.com/thepeacockproject/Octopussy/internal/content"
)

type intelAndRank struct {
	Intel int `json:"Intel"`
	Rank  int `json:"Rank"`
}

type stopSessionRequest struct {
	GameTime     float64                 `json:"gameTime"`
	Score        int                     `json:"score"`
	SessionId    string                  `json:"sessionId"`
	ScoreDetails stopSessionScoreDetails `json:"scoreDetails"`
}

type stopSessionResponse struct {
	SessionId              string                   `json:"SessionId"`
	Rewards                stopSessionRewards       `json:"Rewards"`
	AnyChallengesCompleted bool                     `json:"AnyChallengesCompleted"`
	MissionId              string                   `json:"MissionId"`
	TotalRewards           stopSessionRewards       `json:"TotalRewards"`
	CompletionTime         float64                  `json:"CompletionTime"`
	SessionProgression     stopSessionProgression   `json:"SessionProgression"`
	StopReason             int                      `json:"StopReason"`
	MissionType            *interface{}             `json:"MissionType"`
	MissionTier            int                      `json:"MissionTier"`
	MissionProgressions    []interface{}            `json:"MissionProgressions"`
	Score                  int                      `json:"Score"`
	ScoreDetails           *stopSessionScoreDetails `json:"ScoreDetails"`
	ScoreDetailsDiff       *stopSessionScoreDetails `json:"ScoreDetailsDiff"`
	PreviousScore          int                      `json:"PreviousScore"`
}

type stopSessionScoreDetails struct {
	Time      int `json:"Time"`
	Precision int `json:"Precision"`
	Flair     int `json:"Flair"`
}

type stopSessionRewards struct {
	Xp          int64    `json:"Xp"`
	Intel       int      `json:"Intel"`
	Unlockables []string `json:"Unlockables"` // TODO: THIS TYPE IS ALMOST CERTAINLY WRONG!!
}

type stopSessionProgressionItem struct {
	Xp    int64 `json:"Xp"`
	Level int   `json:"Level"`
	Intel int   `json:"Intel"`
}

type stopSessionProgression struct {
	Old stopSessionProgressionItem `json:"Old"`
	New stopSessionProgressionItem `json:"New"`
}

type getForPlayRequest struct {
	MissionId string `json:"missionId"`
}

type getForPlayStatic struct {
	Mission content.Mission
}

type getForPlayDynamic struct {
	NewSessionId  string                 `json:"NewSessionId"`
	SessionResult interface{}            `json:"SessionResult"`
	Progression   *getForPlayProgression `json:"Progression"`
}

type getForPlayProgression struct {
	Group interface{}    `json:"Group"`
	Hub   *getForPlayHub `json:"Hub"`
}

type getForPlayHub struct {
	Previous intelAndRank `json:"Previous"`
	Current  intelAndRank `json:"Current"`
}

type getForPlayResponse struct {
	Static  getForPlayStatic  `json:"Static"`
	Dynamic getForPlayDynamic `json:"Dynamic"`
}

type completeActiveChallengeRequest struct {
	ProfileId   string `json:"profileId"`
	ChallengeId string `json:"challengeId"`
}

type completeActiveChallengeResponse struct {
	CompletedChallengeIds  []string                `json:"CompletedChallengeIds"`
	UpdatedChallengeStates []updatedChallengeState `json:"UpdatedChallengeStates"`
}

type updatedChallengeState struct {
	Id              string `json:"Id"`
	CompletionState string `json:"CompletionState"`
	ProgressCurrent string `json:"ProgressCurrent"`
	ProgressMax     string `json:"ProgressMax"`
}

func (c *Handler) handleGetForPlay(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[getForPlayRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mission, contractFound := c.Content.Contract(body.MissionId)
	if !contractFound {
		c.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "could not find mission" + body.MissionId})
		return
	}

	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	user, err := c.Persistence.GetUser(ctx, userId)
	if err != nil {
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	session, err := c.Persistence.CreateSession(ctx, *user, body.MissionId)

	if err != nil {
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ir := intelAndRank{Intel: user.IntelCurrency, Rank: user.Rank}

	c.WriteJSON(w, http.StatusOK, getForPlayResponse{
		Static: getForPlayStatic{
			Mission: mission,
		},
		Dynamic: getForPlayDynamic{
			NewSessionId:  session.SessionId,
			SessionResult: nil, // TODO?
			Progression: &getForPlayProgression{
				Group: nil,
				Hub: &getForPlayHub{
					Current:  ir,
					Previous: ir,
				},
			},
		},
	})
}

// TODO: Cleanup
func (c *Handler) handleStopSession(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[stopSessionRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: If sessionId is 00000000-0000-0000-0000-000000000000, don't grants unlocks?

	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	session, err := c.Persistence.GetCurrentSessionByUser(ctx, userId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	mission, found := c.Content.Contract(session.ContractId)
	if !found {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user, err := c.Persistence.GetUser(ctx, userId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	rewards := stopSessionRewards{
		Xp:          mission.Gameplay.Rewards.Xp,
		Intel:       mission.Gameplay.Rewards.Intel,
		Unlockables: mission.Gameplay.Rewards.Unlockables,
	}

	for _, unlockable := range rewards.Unlockables {
		c.Persistence.CreateUnlockable(ctx, userId, unlockable)
	}

	oldProgression := stopSessionProgressionItem{
		int64(session.XpWas),
		session.RankWas,
		session.IntelCurrencyWas,
	}

	user.XPEarned += rewards.Xp
	user.IntelCurrency += rewards.Intel
	user.Rank = c.Content.LevelForXP(user.XPEarned)

	c.Persistence.UpdateUser(ctx, user)

	newProgression := stopSessionProgressionItem{
		user.XPEarned,
		user.Rank,
		user.IntelCurrency,
	}

	// TODO: Get previous score from leaderboards
	c.WriteJSON(w, http.StatusOK, stopSessionResponse{
		SessionId:              session.SessionId,
		Rewards:                rewards,
		AnyChallengesCompleted: false,
		MissionId:              session.ContractId,
		TotalRewards:           rewards,
		CompletionTime:         body.GameTime,
		SessionProgression: stopSessionProgression{
			Old: oldProgression,
			New: newProgression,
		},
		StopReason:          0,
		MissionType:         nil,
		MissionTier:         0,
		MissionProgressions: nil,
		Score:               body.Score,
		ScoreDetails:        &body.ScoreDetails,
		ScoreDetailsDiff:    nil,
		PreviousScore:       0,
	})
}

func (c *Handler) handleCompleteActiveChallenge(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[completeActiveChallengeRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	_, err := c.Persistence.CreateChallenge(ctx, userId, body.ChallengeId)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	challenge, ok := c.Content.Challenge(body.ChallengeId)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, unlockable := range challenge.KntDefinition.Rewards.Unlockables {
		c.Persistence.CreateUnlockable(ctx, userId, unlockable)
	}

	user, err := c.Persistence.GetUser(ctx, userId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user.XPEarned += challenge.KntDefinition.Rewards.Xp
	user.IntelCurrency += challenge.KntDefinition.Rewards.Intel
	user.Rank = c.Content.LevelForXP(user.XPEarned)

	c.Persistence.UpdateUser(ctx, user)

	// TODO: Server-side has information about related challenges not in JSON, eg. completionist, infiltration and global challenges.
	c.WriteJSON(w, http.StatusOK, completeActiveChallengeResponse{
		CompletedChallengeIds:  []string{},
		UpdatedChallengeStates: []updatedChallengeState{},
	})
}
