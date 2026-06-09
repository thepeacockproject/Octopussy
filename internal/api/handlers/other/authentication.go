package other

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
	"github.com/thepeacockproject/Octopussy/internal/jwt"
)

// authServerVersion is the backend service version we report (distinct from the
// game's GAME_VER). Observed value from the real service.
const authServerVersion = "1.1.0.24"

// dotNetTimeLayout reproduces the .NET "round-trip"-ish UTC stamp the real
// service emits (7 fractional-second digits, literal Z). Go can't print a bare
// "Z" via the zone verb, so we format without a zone and append it.
const dotNetTimeLayout = "2006-01-02T15:04:05.0000000"

// authConfigInitResponse is the body of the authentication configuration Init
// endpoint. The game calls it right after obtaining its access token to learn
// its profile id, the resource blob location, and server time.
type authConfigInitResponse struct {
	Token         string          `json:"token"`
	BlobConfig    authBlobConfig  `json:"blobconfig"`
	ProfileID     string          `json:"profileid"`
	ServerVersion string          `json:"serverversion"`
	ServerTimeUTC string          `json:"servertimeutc"`
	ProfileInit   authProfileInit `json:"profileinit"`
}

// authBlobConfig points the game at the resource blob store and carries the SAS
// query string it appends when fetching resources.
type authBlobConfig struct {
	BlobURL         string  `json:"bloburl"`
	BlobSig         string  `json:"blobsig"`
	BlobSigDuration float64 `json:"blobsigduration"`
}

// authProfileInit carries account-level bootstrap info for the profile.
type authProfileInit struct {
	IAS     int    `json:"ias"`
	Email   string `json:"email"`
	BaseURL string `json:"baseurl"`
}

// handleAuthConfigInit answers authentication/api/configuration/Init. It runs
// behind the JWT middleware: the profile id and the legacy "token" string are
// derived from the bearer token's claims.
//
// TODO: the blob SAS signature is a stub (we serve resources ourselves with no
// SAS auth), and the account email/ias/baseurl are placeholders until we have a
// real account system.
func (c *Handler) handleAuthConfigInit(w http.ResponseWriter, r *http.Request) {
	claims, ok := jwt.ClaimsFrom(r.Context())
	if !ok {
		c.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	configName := r.URL.Query().Get("configName")
	gamerTag := r.URL.Query().Get("gamerTag")

	// Legacy token format: "{exp}-{iat}-{platform}-{userid}".
	var exp, iat int64
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Unix()
	}
	if claims.IssuedAt != nil {
		iat = claims.IssuedAt.Unix()
	}
	legacyToken := fmt.Sprintf("%d-%d-%s-%s", exp, iat, claims.Platform, claims.UserID)

	base := shared.RequestBaseURL(r)
	expiry := time.Now().UTC().Add(c.JWT.TTL())
	// Stub SAS: structurally like an Azure blob signature, but our resource
	// routes ignore it. Only the expiry is real, to mimic blobsigduration.
	blobSig := fmt.Sprintf("?sv=2025-01-05&se=%s&sr=c&sp=r&sig=%s",
		url.QueryEscape(expiry.Format("2006-01-02T15:04:05Z")), "octopussy-stub-signature")

	c.Log.Debug("auth config init",
		slog.String("config", configName),
		slog.String("gamerTag", gamerTag),
		slog.String("profile", claims.Subject),
	)

	c.WriteJSON(w, http.StatusOK, authConfigInitResponse{
		Token: legacyToken,
		BlobConfig: authBlobConfig{
			BlobURL:         base + "/resources-1-1/",
			BlobSig:         blobSig,
			BlobSigDuration: float64(c.JWT.TTL().Milliseconds()),
		},
		ProfileID:     claims.Subject,
		ServerVersion: authServerVersion,
		ServerTimeUTC: time.Now().UTC().Format(dotNetTimeLayout) + "Z",
		ProfileInit: authProfileInit{
			IAS:     2,
			Email:   "player@octopussy.local",
			BaseURL: "https://thepeacockproject.org",
		},
	})
}
