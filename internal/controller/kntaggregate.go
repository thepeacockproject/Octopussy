package controller

import "net/http"

// handleGetMenuContent answers KntAggregateService/GetMenuContent with the
// aggregated mission catalog (mission lists, the mission pool, progressions,
// and player levels). The data is static and assembled by the content store.
//
// TODO: MissionProgressions is currently null for everyone; once we track
// per-profile progression it should be populated from the caller's profile.
func (c *Controller) handleGetMenuContent(w http.ResponseWriter, _ *http.Request) {
	c.writeJSON(w, http.StatusOK, c.Content.MenuContent())
}
