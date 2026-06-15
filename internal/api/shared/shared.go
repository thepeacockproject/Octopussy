package shared

import (
	"encoding/json"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/jwt"
)

func ParseBody[T any](r *http.Request) (T, bool) {
	var v T

	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, false
	}

	return v, true
}

// RequestBaseURL reconstructs the scheme://host this server was reached at,
// honoring reverse-proxy headers. It is used to point config responses back at
// Octopussy itself.
func RequestBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = fwd
	}
	return scheme + "://" + host
}

// GetUserId resolves the caller's profile id from the validated token
// claims, preferring the subject and falling back to the userid claim.
func GetUserId(r *http.Request) (userId string, ok bool) {
	claims, ok := jwt.ClaimsFrom(r.Context())
	if !ok {
		return "", false
	}

	if claims.Subject != "" {
		return claims.Subject, true
	}

	if claims.UserId != "" {
		return claims.UserId, true
	}

	return "", false
}
