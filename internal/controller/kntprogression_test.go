package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChallengesRequiresToken(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProgressionService/GetChallenges", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestGetChallengesReturnsEmptyArray(t *testing.T) {
	c, token := newTestController(t)
	const profileID = "00000000-0000-4000-8000-000000000001"

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProgressionService/GetChallenges", nil)
	req.Header.Set("Authorization", "Bearer "+token(profileID))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}
