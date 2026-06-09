package shared

import "net/http"

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
