package controller

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
)

// oAuthTokenResponse is the body returned by POST /oauth/token.
type oAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// platformForGrant maps an OAuth grant type to the platform string and the form
// field that carries the external user ID for it.
func platformForGrant(grantType string) (platform, useridField string) {
	switch grantType {
	case "external_steam":
		return "steam", "steam_userid"
	case "external_epic":
		return "epic", "epic_userid"
	case "external_apple":
		return "apple", "apple_userid"
	default:
		// Unknown grant: fall back to a generic "userid" field.
		return "unknown", "userid"
	}
}

// handleOAuthToken issues a signed access token for an external login. It is the
// Go counterpart to Peacock's handleOAuthToken / the IOI kauth oauth/token
// endpoint. The request is form-encoded; the response carries an RS256 JWT
// shaped like the game's real tokens.
//
// TODO: validate the steam_clienttoken / external session ticket against the platform, persist the external-id -> profile-id mapping, and support the refresh_token grant. For now we trust the supplied external id and derive a stable profile id from it.
func (c *Controller) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form body"})
		return
	}

	grantType := r.PostForm.Get("grant_type")
	platform, useridField := platformForGrant(grantType)
	userID := r.PostForm.Get(useridField)

	if grantType == "" || userID == "" {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing grant_type or external user id"})
		return
	}

	gs := r.PostForm.Get("gs")
	appID := r.PostForm.Get("steam_appid")

	// Reuse a supplied profile id, else derive a stable one from the external
	// identity so the same account always maps to the same profile.
	profileID := r.PostForm.Get("pId")
	if profileID == "" {
		profileID = uuid.NewSHA1(uuid.NameSpaceOID, []byte(platform+":"+userID)).String()
	}

	access, err := c.JWT.Sign(jwt.Claims{
		ClientID:   profileID,
		Roles:      "user",
		UniqueName: profileID,
		AuthMethod: grantType,
		Platform:   platform,
		Country:    "??",
		Locale:     r.PostForm.Get("locale"),
		Region:     r.PostForm.Get("rgn"),
		GameServer: gs,
		BuildID:    r.PostForm.Get("buildid"),
		PIS:        appID,
		UserID:     userID,
		Scope:      []string{"ioi:games"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   requestBaseURL(r),
			Subject:  profileID,
			Audience: jwt.Audience(gs),
		},
	})
	if err != nil {
		c.Log.Error("failed to sign token", slog.String("error", err.Error()))
		c.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue token"})
		return
	}

	c.Log.Info("issued access token",
		slog.String("profile", profileID),
		slog.String("platform", platform),
		slog.String("grant", grantType),
	)

	c.writeJSON(w, http.StatusOK, oAuthTokenResponse{
		AccessToken:  access,
		ExpiresIn:    int(c.JWT.TTL().Seconds()),
		TokenType:    "bearer",
		RefreshToken: profileID,
		Scope:        "ioi:games offline_access",
	})
}
