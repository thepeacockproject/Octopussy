package controller

import (
	"log/slog"
	"net/http"
	"strconv"
)

// privacyPolicyVersion is sent as the x-ms-meta-version response header on the
// privacy-policy document. It mirrors the blob metadata the client originally
// received from IOI's storage and is bumped when the policy text changes.
const privacyPolicyVersion = "20260529"

// Static documents served under /files/. These were originally hosted on IOI's
// blob storage; we serve our own copies so the client has somewhere to fetch
// them from (the config discovery doc points here).

// onlineConfigFile is the body of /files/onlineconfig.json. It carries client
// tuning knobs (timeouts, token renewal cadence, websocket settings).
type onlineConfigFile struct {
	Online onlineConfigFileOnline `json:"Online"`
}

type onlineConfigFileOnline struct {
	BlobSignatureRenewalMinutes         float64 `json:"BlobSignatureRenewalMinutes"`
	BlobSignatureRenewalCooldownSeconds float64 `json:"BlobSignatureRenewalCooldownSeconds"`
	GPSTokenRenewalMinutes              float64 `json:"GPSTokenRenewalMinutes"`
	GPSTokenRenewalCooldownSeconds      float64 `json:"GPSTokenRenewalCooldownSeconds"`
	DefaultHttpTimeout                  float64 `json:"DefaultHttpTimeout"`
	WebSocketsRoute                     string  `json:"WebSocketsRoute"`
	DefaultWebSocketTimeout             float64 `json:"DefaultWebSocketTimeout"`
	WebSocketsEnabled                   bool    `json:"WebSocketsEnabled"`
}

// handleOnlineConfigFile serves /files/onlineconfig.json. The values are static
// for now; websockets stay disabled until we implement that path.
func (c *Controller) handleOnlineConfigFile(w http.ResponseWriter, _ *http.Request) {
	c.writeJSON(w, http.StatusOK, onlineConfigFile{
		Online: onlineConfigFileOnline{
			BlobSignatureRenewalMinutes:         5.0,
			BlobSignatureRenewalCooldownSeconds: 30.0,
			GPSTokenRenewalMinutes:              5.0,
			GPSTokenRenewalCooldownSeconds:      30.0,
			DefaultHttpTimeout:                  30.0,
			WebSocketsRoute:                     "authentication/ws",
			DefaultWebSocketTimeout:             30.0,
			WebSocketsEnabled:                   false,
		},
	})
}

// handlePrivacyPolicy serves the static KNT privacy-policy dialog document. The
// body is stored verbatim and embedded in the binary; the x-ms-meta-version
// header reproduces what the original blob storage returned, which the client
// checks to decide whether the cached policy is current.
func (c *Controller) handlePrivacyPolicy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("x-ms-meta-version", privacyPolicyVersion)
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", strconv.Itoa(len(c.Content.PrivacyPolicy())))
	} else if r.Method == http.MethodGet {
		if _, err := w.Write(c.Content.PrivacyPolicy()); err != nil {
			c.Log.Error("failed to write privacy policy", slog.String("error", err.Error()))
		}
	}
}
