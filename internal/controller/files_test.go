package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOnlineConfigFile(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/onlineconfig.json", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp onlineConfigFile
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	o := resp.Online
	if o.WebSocketsEnabled {
		t.Error("WebSocketsEnabled = true, want false")
	}
}
