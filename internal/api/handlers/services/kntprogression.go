package services

import (
	"encoding/json"
	"net/http"
)

type GetChallengesRequest struct {
	Query string `json:"query"`
}

// KntProgressionService backs challenges and related progression data.
func (c *Handler) handleGetChallenges(w http.ResponseWriter, r *http.Request) {
	var req GetChallengesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	c.WriteJSON(w, http.StatusOK, []any{})
}
