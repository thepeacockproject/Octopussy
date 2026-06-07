package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
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
	ProfileIDs []string `json:"profileIDs"`
}

// resolvedProfileResponse is one entry in the ResolveProfiles response. Field
// order and casing match the game's expected shape.
type resolvedProfileResponse struct {
	Id             string            `json:"Id"`
	LinkedAccounts map[string]string `json:"LinkedAccounts"`
	Gamertag       string            `json:"Gamertag"`
}

// handleResolveProfiles returns the gamertag and linked platform accounts for
// each requested profile id, preserving request order and skipping ids that
// don't resolve to a known user.
func (c *Controller) handleResolveProfiles(w http.ResponseWriter, r *http.Request) {
	var req resolveProfilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	profiles, err := c.Store.ResolveProfiles(r.Context(), req.ProfileIDs)
	if err != nil {
		c.Log.Error("resolve profiles failed", slog.String("error", err.Error()))
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	out := make([]resolvedProfileResponse, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, resolvedProfileResponse{
			Id:             p.ID,
			LinkedAccounts: p.LinkedAccounts,
			Gamertag:       p.Gamertag,
		})
	}

	c.Log.Debug("resolve profiles",
		slog.Int("requested", len(req.ProfileIDs)),
		slog.Int("resolved", len(out)),
	)
	c.writeJSON(w, http.StatusOK, out)
}
