package controller

import (
	"net/http"
)

// KntProgressionService backs challenges and related progression data.
func (c *Controller) handleGetChallenges(w http.ResponseWriter, _ *http.Request) {
	// TODO
	// body.query "" -> empty array
	// body.query "all" -> literally all challenges
	// there are other values
	c.writeJSON(w, http.StatusOK, []any{})
}
