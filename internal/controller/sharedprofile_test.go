package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveProfilesRequiresToken(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/SharedProfileService/ResolveProfiles",
		strings.NewReader(`{"profileIDs":["x"]}`))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestResolveProfiles(t *testing.T) {
	c, token := newTestController(t)

	const profileID = "d97fc63c-9379-4300-9018-4c8d787ac63c"
	if err := c.Store.EnsureUser(context.Background(), profileID, "peacock", "steam", "0123"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	body := `{"profileIDs":["` + profileID + `","00000000-0000-4000-8000-000000000000"]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/SharedProfileService/ResolveProfiles",
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token(profileID))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp []resolvedProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Only the known profile resolves; the unknown id is omitted.
	if len(resp) != 1 {
		t.Fatalf("got %d profiles, want 1: %s", len(resp), rr.Body.String())
	}
	got := resp[0]
	if got.Id != profileID {
		t.Errorf("Id = %q, want %q", got.Id, profileID)
	}
	if got.Gamertag != "peacock" {
		t.Errorf("Gamertag = %q, want %q", got.Gamertag, "peacock")
	}
	if got.LinkedAccounts["steam"] != "0123" {
		t.Errorf("LinkedAccounts[steam] = %q, want %q", got.LinkedAccounts["steam"], "0123")
	}
}
