package services_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/thepeacockproject/Octopussy/internal/api/apitest"
	"github.com/thepeacockproject/Octopussy/internal/api/handlers/services"
)

func TestCustomProfileReadAllRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/services/CustomProfileDataService/ReadAll", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestCustomProfileWritePersists(t *testing.T) {
	c, token := apitest.NewTestController(t)
	const profileID = "00000000-0000-4000-8000-000000000001"
	bearer := "Bearer " + token(profileID)

	// Write a new value.
	rr := httptest.NewRecorder()
	body := `{"profileId":"` + profileID + `","key":"attractToMainMenuFlow","data":"true"}`
	req := httptest.NewRequest(http.MethodPost,
		"/services/CustomProfileDataService/Write", strings.NewReader(body))
	req.Header.Set("Authorization", bearer)
	c.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("write status = %d, want 200", rr.Code)
	}

	// Read it back for that profile (no body on ReadAll).
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/services/CustomProfileDataService/ReadAll", nil)
	req.Header.Set("Authorization", bearer)
	c.Handler().ServeHTTP(rr, req)

	var resp services.CustomProfileReadAllResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := resp.Data["attractToMainMenuFlow"]; got != "true" {
		t.Errorf("after write, attractToMainMenuFlow = %q, want %q", got, "true")
	}
}
