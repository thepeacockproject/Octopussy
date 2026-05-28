package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/jwt"
)

// CustomProfileDataService persists arbitrary client-side profile state as an
// opaque string key/value bag (see internal/store). Two endpoints back it:
//
//	POST /services/CustomProfileDataService/ReadAll  -> {"Data": {key: value, ...}}
//	POST /services/CustomProfileDataService/Write     -> 200 OK
//
// Both run behind the JWT middleware, so the caller's profile id comes from the
// bearer token. Values are stored verbatim; the server never parses them.

// defaultProfileID is the fallback when a request carries no usable profile id
// in its token (shouldn't happen behind the JWT middleware, but keeps the
// handlers robust in dev).
const defaultProfileID = "00000000-0000-0000-0000-000000000000"

// profileIDFor resolves the caller's profile id from the validated token
// claims, preferring the subject and falling back to the userid claim.
func profileIDFor(r *http.Request) string {
	claims, ok := jwt.ClaimsFrom(r.Context())
	if !ok {
		return defaultProfileID
	}
	if claims.Subject != "" {
		return claims.Subject
	}
	if claims.UserID != "" {
		return claims.UserID
	}
	return defaultProfileID
}

// customProfileWriteRequest is the body posted to Write: a single key/value
// pair to upsert for a profile.
type customProfileWriteRequest struct {
	ProfileID string `json:"profileId"`
	Key       string `json:"key"`
	Data      string `json:"data"`
}

// customProfileReadAllResponse mirrors the game's expected shape: a single
// "Data" object mapping every stored key to its string value.
type customProfileReadAllResponse struct {
	Data map[string]string `json:"Data"`
}

// handleCustomProfileReadAll returns every stored key/value pair for a profile.
// The request has no body; the profile id comes from the bearer token. A
// brand-new profile is seeded with defaultCustomProfileData first, so the game
// always gets a sane initial state.
func (c *Controller) handleCustomProfileReadAll(w http.ResponseWriter, r *http.Request) {
	profileID := profileIDFor(r)
	ctx := r.Context()

	// Seed defaults for a never-before-seen profile so the client isn't handed
	// an empty bag on first launch.
	count, err := c.Store.CountCustomProfileData(ctx, profileID)
	if err != nil {
		c.Log.Error("custom profile count failed", slog.String("error", err.Error()))
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if count == 0 {
		if err := c.Store.SeedCustomProfileDefaults(ctx, profileID, map[string]string{}); err != nil {
			c.Log.Error("custom profile seed failed", slog.String("error", err.Error()))
			c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	}

	data, err := c.Store.ReadCustomProfileData(ctx, profileID)
	if err != nil {
		c.Log.Error("custom profile read failed", slog.String("error", err.Error()))
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	c.Log.Debug("custom profile read all",
		slog.String("profileId", profileID),
		slog.Int("keys", len(data)),
	)
	c.writeJSON(w, http.StatusOK, customProfileReadAllResponse{Data: data})
}

// handleCustomProfileWrite upserts a single key/value pair and answers 200 OK
// with an empty body, matching the game's expectation.
func (c *Controller) handleCustomProfileWrite(w http.ResponseWriter, r *http.Request) {
	var req customProfileWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Key == "" {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing key"})
		return
	}

	// The token is authoritative for the profile id; the body also carries one,
	// but we only trust it as a fallback when the token has none (dev).
	profileID := profileIDFor(r)
	if profileID == defaultProfileID && req.ProfileID != "" {
		profileID = req.ProfileID
	}

	if err := c.Store.WriteCustomProfileData(r.Context(), profileID, req.Key, req.Data); err != nil {
		c.Log.Error("custom profile write failed", slog.String("error", err.Error()))
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	c.Log.Debug("custom profile write",
		slog.String("profileId", profileID),
		slog.String("key", req.Key),
	)
	w.WriteHeader(http.StatusOK)
}
