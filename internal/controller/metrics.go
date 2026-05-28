package controller

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// metricsEvent is the envelope the game POSTs to the metrics ingest endpoint.
// We only decode the lightweight identifying fields; the bulky Payload/Context
// (hardware specs, session info, etc.) is intentionally ignored.
type metricsEvent struct {
	Name          string `json:"Name"`
	EventTypeHash int64  `json:"EventTypeHash"`
}

// handleMetricsIngest accepts telemetry events and discards them. The config
// discovery doc points the game's metrics host here; we acknowledge events so
// the client is happy, but persist nothing.
func (c *Controller) handleMetricsIngest(w http.ResponseWriter, r *http.Request) {
	// Peek at the event name for observability without keeping the payload.
	var evt metricsEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err == nil && evt.Name != "" {
		c.Log.Debug("metrics event ignored",
			slog.String("event", evt.Name),
			slog.Int64("hash", evt.EventTypeHash),
		)
	}

	w.WriteHeader(http.StatusNoContent)
}
