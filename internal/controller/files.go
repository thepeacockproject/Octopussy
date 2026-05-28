package controller

import "net/http"

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
