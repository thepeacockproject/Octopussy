package controller

import (
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
)

// maxFormBody caps how much of an oauth request body we read, guarding against
// an oversized body while staying well above any real login payload.
const maxFormBody = 1 << 20 // 1 MiB

// parseAuthForm parses an x-www-form-urlencoded login body WITHOUT trusting the
// Content-Type header. The game lies to us about it's Content-Type, so we
// can't use net/http.
func parseAuthForm(r *http.Request) (url.Values, error) {
	values := url.Values{}
	for k, v := range r.URL.Query() {
		values[k] = append(values[k], v...)
	}
	if r.Body != nil {
		body, err := io.ReadAll(io.LimitReader(r.Body, maxFormBody))
		if err != nil {
			return nil, err
		}
		parsed, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		for k, v := range parsed {
			values[k] = append(values[k], v...)
		}
	}
	return values, nil
}

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
	form, err := parseAuthForm(r)
	if err != nil {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form body"})
		return
	}

	grantType := form.Get("grant_type")
	platform, useridField := platformForGrant(grantType)
	userID := form.Get(useridField)

	if grantType == "" || userID == "" {
		c.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing grant_type or external user id"})
		return
	}

	gs := form.Get("gs")
	appID := form.Get("steam_appid")

	// Reuse a supplied profile id, else derive a stable one from the external
	// identity so the same account always maps to the same profile.
	profileID := form.Get("pId")
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
		Locale:     form.Get("locale"),
		Region:     form.Get("rgn"),
		GameServer: gs,
		BuildID:    form.Get("buildid"),
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
