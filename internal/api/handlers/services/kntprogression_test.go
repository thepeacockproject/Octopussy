package services_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
)

func TestGetChallengesRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProgressionService/GetChallenges", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestGetChallengesReturnsEmptyArray(t *testing.T) {
	c, token := apitest.NewTestController(t)
	userId := uuid.Nil.String()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProgressionService/GetChallenges", strings.NewReader(`{"query":""}`))
	req.Header.Set("Authorization", "Bearer "+token(userId))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}
