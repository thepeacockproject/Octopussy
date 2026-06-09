package base

import "net/http"

// handleHealth is a simple liveness probe.
func (c *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	c.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
