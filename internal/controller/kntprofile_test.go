package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetClientEntitlementsRequiresToken(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/SetClientEntitlements", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestSetClientEntitlementsNoop(t *testing.T) {
	c, token := newTestController(t)
	const profileID = "00000000-0000-4000-8000-000000000001"

	rr := httptest.NewRecorder()
	body := `{"entitlements":["dlc1","dlc2"]}`
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/SetClientEntitlements", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token(profileID))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "" {
		t.Errorf("body = %q, want empty", got)
	}
}

func TestGetEntitlementsRequiresToken(t *testing.T) {
	c, _ := newTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetEntitlements", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestGetEntitlementsReturnsPlatformEntitlements(t *testing.T) {
	c, token := newTestController(t)
	const profileID = "00000000-0000-4000-8000-000000000001"

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetEntitlements", nil)
	req.Header.Set("Authorization", "Bearer "+token(profileID))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp getEntitlementsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.PlatformEntitlements) != len(platformEntitlements) {
		t.Errorf("PlatformEntitlements len = %d, want %d",
			len(resp.PlatformEntitlements), len(platformEntitlements))
	}
	if resp.IoiEntitlements == nil {
		t.Errorf("IoiEntitlements = nil, want non-null empty array")
	}
}
