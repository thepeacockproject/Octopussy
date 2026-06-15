package services

import (
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
)

// TODO
var platformEntitlements = []string{
	"3768760", "3950810", "3950820", "3950830", "3950850", "3950870",
	"3950880", "3950900", "3950910", "3950930", "3986520", "4626160",
}

// playerProgression carries a player's total XP and the level it maps to.
type playerProgression struct {
	Xp    int64 `json:"Xp"`
	Level int   `json:"Level"`
}

// getEntitlementsResponse mirrors the game's expected shape. IoiEntitlements is
// left empty for now; only the platform SKUs are reported.
type getEntitlementsResponse struct {
	PlatformEntitlements []string `json:"PlatformEntitlements"`
	IoiEntitlements      []string `json:"IoiEntitlements"`
}

// playerProgressionResponse is the body returned by GetPlayerProgression.
type playerProgressionResponse struct {
	PlayerProgression playerProgression `json:"PlayerProgression"`
}

func (c *Handler) handleSetClientEntitlements(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleGetEntitlements reports the entitlements owned by the caller. Octopussy
// grants the full platform catalog to everyone and reports no IOI entitlements.
func (c *Handler) handleGetEntitlements(w http.ResponseWriter, _ *http.Request) {
	c.WriteJSON(w, http.StatusOK, getEntitlementsResponse{
		PlatformEntitlements: platformEntitlements,
		IoiEntitlements:      []string{},
	})
}

func (c *Handler) handleGetPlayerProgression(w http.ResponseWriter, r *http.Request) {
	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	user, err := c.Store.GetUser(ctx, userId)
	if err != nil {
		c.Log.Error("get player progression failed", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	// TODO: Validate if Rank still matches the XPEarned, probably best to do that in /oauth/token.

	c.WriteJSON(w, http.StatusOK, playerProgressionResponse{
		PlayerProgression: playerProgression{Xp: user.XPEarned, Level: user.Rank},
	})
}
