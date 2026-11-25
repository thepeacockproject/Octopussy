package services

import (
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
)

// playerProgression carries a player's total XP and the level it maps to.
type playerProgression struct {
	Xp    int64 `json:"Xp"`
	Level int   `json:"Level"`
}

type getEntitlementsResponse struct {
	PlatformEntitlements []string `json:"PlatformEntitlements"`
	IoiEntitlements      []string `json:"IoiEntitlements"`
}

// playerProgressionResponse is the body returned by GetPlayerProgression.
type playerProgressionResponse struct {
	PlayerProgression playerProgression `json:"PlayerProgression"`
}

type signupIoiAccountRequest struct {
	Email                      string `json:"email"`
	LanguageCode               string `json:"languageCode"`
	MarketingConsentPrechecked bool   `json:"marketingConsentPrechecked"`
}

type signupIoiAccountResponse struct {
	Success      bool   `json:"Success"`
	ErrorCode    int    `json:"ErrorCode"`
	IsConfirmed  bool   `json:"IsConfirmed"`
	LinkedEmail  string `json:"LinkedEmail"`
	IoiAccountId string `json:"IoiAccountId"`
	BaseUrl      string `json:"BaseUrl"`
}

func (c *Handler) handleSetClientEntitlements(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// handleGetEntitlements reports the platform entitlements verified for the
// caller at login. With none on record it falls back to the Steam validator's
// policy.
func (c *Handler) handleGetEntitlements(w http.ResponseWriter, r *http.Request) {
	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ents, err := c.Persistence.GetEntitlements(r.Context(), userId)
	if err != nil {
		c.Log.Error("get entitlements failed", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if len(ents) == 0 {
		ents = c.Steam.Fallback()
	}

	c.WriteJSON(w, http.StatusOK, getEntitlementsResponse{
		PlatformEntitlements: ents,
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

	user, err := c.Persistence.GetUser(ctx, userId)
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

func (c *Handler) handleSignupIoiAccount(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[signupIoiAccountRequest](r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	userId, ok := shared.GetUserId(r)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c.WriteJSON(w, http.StatusOK, signupIoiAccountResponse{
		Success:      true,
		ErrorCode:    0,
		IsConfirmed:  true,
		LinkedEmail:  body.Email,
		IoiAccountId: userId,
		BaseUrl:      "https://thepeacockproject.org",
	})
}
