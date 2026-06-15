package services

import (
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
)

// SharedProfileService exposes public-facing profile data (gamertag and linked
// platform accounts) for one or more users:
//
//	POST /services/SharedProfileService/ResolveProfiles
//	  body:     {"profileIDs": ["<id>", ...]}
//	  response: [{"Id": "<id>", "LinkedAccounts": {"steam": "..."}, "Gamertag": "..."}]
//
// It runs behind the JWT middleware. Unknown ids are simply omitted from the
// response.

// resolveProfilesRequest is the body posted to ResolveProfiles.
type resolveProfilesRequest struct {
	ProfileIds []string `json:"profileIDs"`
}

// resolveProfileResponse is one entry in the ResolveProfiles response. Field
// order and casing match the game's expected shape.
type resolveProfileResponse struct {
	Id             string            `json:"Id"`
	LinkedAccounts map[string]string `json:"LinkedAccounts"`
	Gamertag       string            `json:"Gamertag"`
}

func (c *Handler) handleGetFriendsCount(w http.ResponseWriter, r *http.Request) {
	c.WriteJSON(w, http.StatusOK, 0)
}

// handleResolveProfiles returns the gamertag and linked platform accounts for
// each requested profile id, preserving request order and skipping ids that
// don't resolve to a known user.
func (c *Handler) handleResolveProfiles(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[resolveProfilesRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profiles, err := c.Persistence.GetProfilesByIds(r.Context(), body.ProfileIds)
	if err != nil {
		c.Log.Error("resolve profiles failed", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	out := make([]resolveProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, resolveProfileResponse{
			Id:             p.Id,
			LinkedAccounts: p.LinkedAccounts,
			Gamertag:       p.Gamertag,
		})
	}

	c.Log.Debug("resolve profiles",
		slog.Int("requested", len(body.ProfileIds)),
		slog.Int("resolved", len(out)),
	)
	c.WriteJSON(w, http.StatusOK, out)
}
