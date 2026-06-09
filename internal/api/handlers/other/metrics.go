package other

import (
	"log/slog"
	"net/http"

	"github.com/thepeacockproject/Octopussy/internal/api/shared"
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
func (c *Handler) handleMetricsIngest(w http.ResponseWriter, r *http.Request) {
	body, ok := shared.ParseBody[metricsEvent](r)

	if ok {
		c.Log.Debug("metrics event ignored",
			slog.String("event", body.Name),
			slog.Int64("hash", body.EventTypeHash),
		)
	}

	w.WriteHeader(http.StatusNoContent)
}
