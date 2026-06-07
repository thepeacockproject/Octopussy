package controller

import (
	"log/slog"
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

// playerProgressionResponse is the body returned by GetPlayerProgression.
type playerProgressionResponse struct {
	PlayerProgression playerProgression `json:"PlayerProgression"`
}

// playerProgression carries a player's total XP and the level it maps to.
type playerProgression struct {
	Xp    int64 `json:"Xp"`
	Level int   `json:"Level"`
}

// handleGetPlayerProgression returns the caller's XP and derived level. Level is computed from the XpThresholds in the game's
// player_levels data.
func (c *Controller) handleGetPlayerProgression(w http.ResponseWriter, r *http.Request) {
	profileID := profileIDFor(r)

	xp, err := c.Store.UserXP(r.Context(), profileID)
	if err != nil {
		c.Log.Error("get player progression failed", slog.String("error", err.Error()))
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	level := c.Content.LevelForXP(xp)

	c.Log.Debug("get player progression",
		slog.String("profileId", profileID),
		slog.Int64("xp", xp),
		slog.Int("level", level),
	)
	c.writeJSON(w, http.StatusOK, playerProgressionResponse{
		PlayerProgression: playerProgression{Xp: xp, Level: level},
	})
}
