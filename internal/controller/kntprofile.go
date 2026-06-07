package controller

import (
	"net/http"
)

func (c *Controller) handleSetClientEntitlements(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// TODO
var platformEntitlements = []string{
	"3768760", "3950810", "3950820", "3950830", "3950850", "3950870",
	"3950880", "3950900", "3950910", "3950930", "3986520", "4626160",
}

// getEntitlementsResponse mirrors the game's expected shape. IoiEntitlements is
// left empty for now; only the platform SKUs are reported.
type getEntitlementsResponse struct {
	PlatformEntitlements []string `json:"PlatformEntitlements"`
	IoiEntitlements      []string `json:"IoiEntitlements"`
}

// handleGetEntitlements reports the entitlements owned by the caller. Octopussy
// grants the full platform catalog to everyone and reports no IOI entitlements.
func (c *Controller) handleGetEntitlements(w http.ResponseWriter, _ *http.Request) {
	c.writeJSON(w, http.StatusOK, getEntitlementsResponse{
		PlatformEntitlements: platformEntitlements,
		IoiEntitlements:      []string{},
	})
}
