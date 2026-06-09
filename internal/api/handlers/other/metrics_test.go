package other_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
)

// sampleGameStartEvent is a trimmed GameStart telemetry event; the real payload
// also carries large HardwareSpecs/Context blobs we don't need for this test.
const sampleGameStartEvent = `{
	"Name": "GameStart",
	"EventTypeHash": 1633344704,
	"Payload": {"IsDisc": false},
	"Context": {"BuildContext": {"BuildId": "10196713", "BuildType": "game"}},
	"Timestamp": "2026-05-27T13:07:55.926Z"
}`

func TestMetricsIngestAcceptsEvent(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader(sampleGameStartEvent))
	req.Header.Set("Content-Type", "application/json")
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if rr.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rr.Body.String())
	}
}

// The ingest sink is lenient: even a malformed body is acknowledged so the
// fire-and-forget client never sees an error.
func TestMetricsIngestToleratesGarbage(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader("not json at all"))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
}

// The endpoint is unauthenticated: telemetry should not require a bearer token.
func TestMetricsIngestNoAuthRequired(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/ingest", strings.NewReader(sampleGameStartEvent))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized {
		t.Fatal("ingest should not require authentication")
	}
}
