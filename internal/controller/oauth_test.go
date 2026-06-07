package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The exact body the game's Steam auth client sends.
const steamLoginBody = "grant_type=external_steam&steam_userid=0123&pId=&steam_appid=3768760&fsp=false&buildid=10236351&locale=US&rgn=en&gs=steam-prod_1"

// TestOAuthTokenWithoutContentType reproduces the original bug: the client does
// not send a Content-Type, so net/http's ParseForm would leave PostForm empty.
// The handler must still parse the body and issue a token.
func TestOAuthTokenWithoutContentType(t *testing.T) {
	c, _ := newTestController(t)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(steamLoginBody))
	// Deliberately NO Content-Type header.
	rr := httptest.NewRecorder()
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var resp oAuthTokenResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("access_token is empty")
	}
	if resp.TokenType != "bearer" {
		t.Errorf("token_type = %q, want bearer", resp.TokenType)
	}
}

// TestOAuthTokenWithContentType confirms the normal path still works when the
// proper form Content-Type is present.
func TestOAuthTokenWithContentType(t *testing.T) {
	c, _ := newTestController(t)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(steamLoginBody))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}
}

// TestOAuthTokenMissingFields verifies the error path still rejects a body with
// no grant_type or external user id.
func TestOAuthTokenMissingFields(t *testing.T) {
	c, _ := newTestController(t)

	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader("foo=bar"))
	rr := httptest.NewRecorder()
	c.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
