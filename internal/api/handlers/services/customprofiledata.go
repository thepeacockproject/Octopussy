package services

import (
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
)

type customProfileReadAllResponse struct {
	Data map[string]string `json:"Data"`
}

type customProfileWriteRequest struct {
	ProfileId string `json:"profileId"`
	Key       string `json:"key"`
	Data      string `json:"data"`
}

// handleCustomProfileReadAll returns every stored key/value pair for a profile.
// The request has no body; the profile id comes from the bearer token. A
// brand-new profile is seeded with defaultCustomProfileData first, so the game
// always gets a sane initial state.
func (c *Handler) handleCustomProfileReadAll(w http.ResponseWriter, r *http.Request) {
	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// TODO: Do this right after ensure so we can also migrate existing accounts if needed
	// Seed defaults for a never-before-seen profile so the client isn't handed
	// an empty bag on first launch.
	count, err := c.Store.CountCustomProfileData(ctx, userId)
	if err != nil {
		c.Log.Error("custom profile count failed", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if count == 0 {
		if err := c.Store.SeedCustomProfileDefaults(ctx, userId, map[string]string{}); err != nil {
			c.Log.Error("custom profile seed failed", slog.String("error", err.Error()))
			c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
	}

	data, err := c.Store.GetCustomProfileData(ctx, userId)
	if err != nil {
		c.Log.Error("custom profile read failed", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	c.Log.Debug("custom profile read all",
		slog.String("userId", userId),
		slog.Int("keys", len(data)),
	)

	c.WriteJSON(w, http.StatusOK, customProfileReadAllResponse{
		Data: data,
	})
}

// handleCustomProfileWrite upserts a single key/value pair and answers 200 OK
// with an empty body, matching the game's expectation.
func (c *Handler) handleCustomProfileWrite(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[customProfileWriteRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if body.Key == "" {
		c.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing key"})
		return
	}

	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	if err := c.Store.SetCustomProfileData(ctx, userId, body.Key, body.Data); err != nil {
		c.Log.Error("custom profile write failed", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	c.Log.Debug("custom profile write",
		slog.String("userId", userId),
		slog.String("key", body.Key),
	)

	w.WriteHeader(http.StatusOK)
}
