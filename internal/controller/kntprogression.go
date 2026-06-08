package controller

import (
	"encoding/json"
	"net/http"
)

type GetChallengesRequest struct {
	Query string `json:"query"`
}

// KntProgressionService backs challenges and related progression data.
func (c *Controller) handleGetChallenges(w http.ResponseWriter, r *http.Request) {
	var req GetChallengesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
}
