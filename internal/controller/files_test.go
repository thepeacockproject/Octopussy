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

func TestPrivacyPolicyFile(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/privacypolicy/knt/privacypolicy_pc_en.json", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if v := rr.Header().Get("x-ms-meta-version"); v != "20260529" {
		t.Errorf("x-ms-meta-version = %q, want 20260529", v)
	}

	// The body is served verbatim and must remain valid JSON.
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(doc["view"]) != `"menu3.modal.ModalDialogGeneric"` {
		t.Errorf("view = %s, want the modal dialog view", doc["view"])
	}
}
