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
	"github.com/thepeacockproject/Octopussy/internal/steam"
)

func TestSetClientEntitlementsRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/SetClientEntitlements", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestSetClientEntitlementsNoop(t *testing.T) {
	c, token := apitest.NewTestController(t)
	userId := uuid.Nil.String()

	rr := httptest.NewRecorder()
	body := `{"entitlements":["dlc1","dlc2"]}`
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/SetClientEntitlements", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token(userId))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if got := strings.TrimSpace(rr.Body.String()); got != "" {
		t.Errorf("body = %q, want empty", got)
	}
}

func TestGetPlayerProgressionRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetPlayerProgression", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func TestGetPlayerProgression(t *testing.T) {
	c, token := apitest.NewTestController(t)
	userId := uuid.Nil.String()

	// Seed the user, then give them enough XP to be level 2 under the test
	// thresholds [3950, 7900].
	ctx := context.Background()
	if err := c.Persistence.EnsureUser(ctx, userId, "gamer", "steam", "steam-123"); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	user, err := c.Persistence.GetUser(ctx, userId)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}

	// TODO: Create a function for this on content
	user.XPEarned = 5500
	user.Rank = c.Content.LevelForXP(user.XPEarned)

	if err := c.Persistence.UpdateUser(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetPlayerProgression", nil)
	req.Header.Set("Authorization", "Bearer "+token(userId))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}

	var resp services.PlayerProgressionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.PlayerProgression.Xp != 5500 {
		t.Errorf("Xp = %d, want 5500", resp.PlayerProgression.Xp)
	}
	if resp.PlayerProgression.Level != 2 {
		t.Errorf("Level = %d, want 2", resp.PlayerProgression.Level)
	}
}

func TestGetEntitlementsRequiresToken(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetEntitlements", nil)
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rr.Code)
	}
}

func getEntitlements(t *testing.T, h http.Handler, token string) services.GetEntitlementsResponse {
	t.Helper()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetEntitlements", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	var resp services.GetEntitlementsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

func steamLogin(t *testing.T, h http.Handler, steamId string) string {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/oauth/token",
		strings.NewReader("grant_type=external_steam&gs=steam-prod_1&steam_userid="+steamId)))

	if rr.Code != http.StatusOK {
		t.Fatalf("login status = %d, want 200 (body %s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return resp.AccessToken
}

func TestGetEntitlementsReturnsStored(t *testing.T) {
	c, _ := apitest.NewTestController(t)
	token := steamLogin(t, c.Handler(), "123")

	err := c.Persistence.SetEntitlements(context.Background(), "steam", "123", []string{steam.AppId, "3950810"})
	if err != nil {
		t.Fatalf("set entitlements: %v", err)
	}

	resp := getEntitlements(t, c.Handler(), token)
	if got := strings.Join(resp.PlatformEntitlements, ","); got != steam.AppId+",3950810" {
		t.Errorf("PlatformEntitlements = %s, want %s,3950810", got, steam.AppId)
	}
}

// TestGetEntitlementsNoneVerified checks that outside tests, a user with no
// verified entitlements gets none.
func TestGetEntitlementsNoneVerified(t *testing.T) {
	c, _ := apitest.NewTestController(t)
	token := steamLogin(t, c.Handler(), "123")
	sv, err := steam.NewValidator(steam.Config{Method: steam.MethodSteam, APIKey: "unused"})
	if err != nil {
		t.Fatalf("validator: %v", err)
	}
	c.Steam = sv

	resp := getEntitlements(t, c.Handler(), token)
	if resp.PlatformEntitlements == nil || len(resp.PlatformEntitlements) != 0 {
		t.Errorf("PlatformEntitlements = %v, want non-null empty array", resp.PlatformEntitlements)
	}
}

func TestGetEntitlementsReturnsPlatformEntitlements(t *testing.T) {
	c, _ := apitest.NewTestController(t)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/services/KntProfileService/GetEntitlements", nil)
	req.Header.Set("Authorization", "Bearer "+steamLogin(t, c.Handler(), "123"))
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}

	var resp services.GetEntitlementsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.PlatformEntitlements) != len(steam.Catalog) {
		t.Errorf("PlatformEntitlements len = %d, want %d (NONE catalog)",
			len(resp.PlatformEntitlements), len(steam.Catalog))
	}
	if resp.IoiEntitlements == nil {
		t.Errorf("IoiEntitlements = nil, want non-null empty array")
	}
}
