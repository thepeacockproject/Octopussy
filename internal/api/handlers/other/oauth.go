package other

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/api/shared"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
	"github.com/thepeacockproject/Octopussy/internal/steam"
)

// maxFormBody caps how much of an oauth request body we read, guarding against
// an oversized body while staying well above any real login payload.
const maxFormBody = 1 << 20 // 1 MiB

var steamUserIdPattern = regexp.MustCompile(`^\d{1,20}$`)

// oAuthTokenResponse is the body returned by POST /oauth/token.
type oAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// handleOAuthToken issues a signed access token for an external login. It is the
// Go counterpart to Peacock's handleOAuthToken / the IOI kauth oauth/token
// endpoint. The request is form-encoded; the response carries an RS256 JWT
// shaped like the game's real tokens.
//
// TODO: support the refresh_token grant.
func (c *Handler) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	form, err := parseAuthForm(r)
	if err != nil {
		c.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid form body"})
		return
	}

	grantType := form.Get("grant_type")
	platform, platformIdField := platformForGrant(grantType)
	platformId := form.Get(platformIdField)

	if grantType == "" || platformId == "" {
		c.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing grant_type or external user id"})
		return
	}

	if platform == "steam" && !steamUserIdPattern.MatchString(platformId) {
		c.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid steam_userid"})
		return
	}

	gs := form.Get("gs")
	appId := form.Get("steam_appid")

	// Reuse a supplied profile id, else derive a stable one from the external
	// identity so the same account always maps to the same profile.
	userId := form.Get("pId")
	if userId == "" {
		userId = uuid.NewSHA1(uuid.NameSpaceOID, []byte(platform+":"+platformId)).String()
	}

	// Persist the user and its external identity on first login; existing rows
	// are left untouched. The external user id seeds the initial gamertag.
	if err := c.Persistence.EnsureUser(r.Context(), userId, platformId, platform, platformId); err != nil {
		c.Log.Error("failed to persist user", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue token"})
		return
	}

	if platform == "steam" {
		c.refreshSteamEntitlements(r.Context(), userId, platformId, form)
	}

	access, err := c.JWT.Sign(jwt.Claims{
		ClientId:   userId,
		Roles:      "user",
		UniqueName: userId,
		AuthMethod: grantType,
		Platform:   platform,
		Country:    "??",
		Locale:     form.Get("locale"),
		Region:     form.Get("rgn"),
		GameServer: gs,
		BuildId:    form.Get("buildid"),
		PIS:        appId,
		UserId:     platformId,
		Scope:      []string{"ioi:games"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   shared.RequestBaseURL(r),
			Subject:  userId,
			Audience: jwt.Audience(gs),
		},
	})
	if err != nil {
		c.Log.Error("failed to sign token", slog.String("error", err.Error()))
		c.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not issue token"})
		return
	}

	c.Log.Info("issued access token",
		slog.String("profile", userId),
		slog.String("platform", platform),
		slog.String("grant", grantType),
	)

	c.WriteJSON(w, http.StatusOK, oAuthTokenResponse{
		AccessToken:  access,
		ExpiresIn:    int(c.JWT.TTL().Seconds()),
		TokenType:    "bearer",
		RefreshToken: userId,
		Scope:        "ioi:games offline_access",
	})
}

// refreshSteamEntitlements stores the entitlements proven by the login's Steam
// ticket. On failure the previously stored set is kept, and login proceeds.
func (c *Handler) refreshSteamEntitlements(ctx context.Context, userId, steamId string, form url.Values) {
	ents, err := c.Steam.Entitlements(ctx,
		form.Get("steam_clienttoken"), form.Get("steam_identity"), steamId)
	if err == nil && len(ents) > 0 {
		if err := c.Persistence.SetEntitlements(ctx, userId, ents); err != nil {
			c.Log.Error("failed to store entitlements",
				slog.String("profile", userId), slog.String("error", err.Error()))
		}
		return
	}
	if errors.Is(err, steam.ErrDisabled) {
		return
	}

	reason := "no entitlements returned"
	if err != nil {
		reason = err.Error()
	}

	stored, getErr := c.Persistence.GetEntitlements(ctx, userId)
	if getErr == nil && len(stored) > 0 {
		c.Log.Warn("could not refresh steam entitlements; recently acquired DLC may not show",
			slog.String("profile", userId), slog.String("reason", reason))
		return
	}
	c.Log.Error("could not get steam entitlements for user",
		slog.String("profile", userId), slog.String("reason", reason))
}

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

// platformForGrant maps an OAuth grant type to the platform string and the form
// field that carries the external userid for it.
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
