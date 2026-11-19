package services_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/services"
)

func TestResolveProfilesRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/SharedProfileService/ResolveProfiles",
		strings.NewReader(`{"profileIDs":["x"]}`))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestResolveProfiles(t *testing.T) {
	c, token := apitest.NewTestController(t)

	const userId = "d97fc63c-9379-4300-9018-4c8d787ac63c"
	if err := c.Persistence.EnsureUser(context.Background(), userId, "peacock", "steam", "0123"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	body := `{"profileIDs":["` + userId + `","` + uuid.Nil.String() + `"]}`
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/SharedProfileService/ResolveProfiles",
		strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token(userId))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %s)", rr.Code, http.StatusOK, rr.Body.String())
	}

	var resp []services.ResolvedProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// Only the known profile resolves; the unknown id is omitted.
	if len(resp) != 1 {
		t.Fatalf("got %d profiles, want 1: %s", len(resp), rr.Body.String())
	}
	got := resp[0]
	if got.Id != userId {
		t.Errorf("Id = %q, want %q", got.Id, userId)
	}
	if got.Gamertag != "peacock" {
		t.Errorf("Gamertag = %q, want %q", got.Gamertag, "peacock")
	}
	if got.LinkedAccounts["steam"] != "0123" {
		t.Errorf("LinkedAccounts[steam] = %q, want %q", got.LinkedAccounts["steam"], "0123")
	}
}
