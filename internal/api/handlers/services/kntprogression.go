package services

import (
	"net/http"
	"strings"

	"github.com/samber/lo"
	"github.com/thepeacockproject/Octopussy/internal/api/shared"
	"github.com/thepeacockproject/Octopussy/internal/content"
	shared2 "github.com/thepeacockproject/Octopussy/internal/shared"
	"github.com/thepeacockproject/Octopussy/internal/store"
)

type getChallengesRequest struct {
	Query string `json:"query"`
}

func (c *Handler) handleGetChallenges(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[getChallengesRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	profileId := profileIDFor(r)

	challenges, err := c.Store.GetChallenges(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.Query == "" {
		c.WriteJSON(w, http.StatusOK, []content.ChallengeWithState{})
	} else if body.Query == "all" {
		result := lo.Map(
			c.Content.Challenges(),
			func(c content.Challenge, _ int) content.ChallengeWithState {
				return mapToChallengeWithState(c, challenges)
			},
		)

		c.WriteJSON(w, http.StatusOK, result)
	} else {
		query := strings.Split(body.Query, "|")

		result := lo.FilterMap(
			c.Content.Challenges(),
			func(c content.Challenge, _ int) (content.ChallengeWithState, bool) {
				if lo.Some(c.KntDefinition.Tags, query) {
					return mapToChallengeWithState(c, challenges), true
				}

				return content.ChallengeWithState{}, false
			},
		)

		c.WriteJSON(w, http.StatusOK, result)
	}
}

func (c *Handler) handleGetChallengeMosaic(w http.ResponseWriter, r *http.Request) {
	// TODO: Rename to UserId instead of Profile, or at least be consistent.
	session, err := c.Store.GetCurrentSessionByUser(r.Context(), profileIDFor(r))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: Rename to Mission/MissionId
	mission, found := c.Content.Contract(session.ContractID)
	if !found {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	profileId := profileIDFor(r)

	challenges, err := c.Store.GetChallenges(ctx, profileId)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	result := lo.FilterMap(
		c.Content.Challenges(),
		func(c content.Challenge, _ int) (content.ChallengeWithState, bool) {
			if lo.Some(c.KntDefinition.Tags, mission.Gameplay.Tags) {
				return mapToChallengeWithState(c, challenges), true
			}

			return content.ChallengeWithState{}, false
		},
	)

	c.WriteJSON(w, http.StatusOK, result)
}

func mapToChallengeWithState(c content.Challenge, challenges []store.UserChallenge) content.ChallengeWithState {
	_, ok := lo.Find(challenges, func(item store.UserChallenge) bool {
		return item.ChallengeID == c.Definition.Id
	})

	return content.ChallengeWithState{
		Challenge: c,
		State: content.StateWithState{
			State:           c.State,
			CompletionState: shared2.ShortHandIf(ok, content.CompletionState_Completed, content.CompletionState_NotCompleted),
			ProgressCurrent: shared2.ShortHandIf(ok, 1, 0),
		},
		KntState: content.KntState{
			NewlyCompleted: false, // TODO: Track if not shown in mosaic before
		},
	}
}
